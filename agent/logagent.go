package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hpcloud/tail"
	"github.com/segmentio/kafka-go"
	"github.com/y7ut/logagent/sender"
)

type App struct {
	Agents map[Collector]*LogAgent
	mu     sync.Mutex
}

type LogAgent struct {
	Tail    *tail.Tail         // 这个代理的tail
	Topic   string             // 需要的kafka主题
	LogFile string             // 这个代理实际监控的日志文件
	Offset  int64              // 偏移值
	Receive chan *Log          // 读取这个代理产生的Log的通道，代理通过tail产生数据就会发送到这个通道中
	context context.Context    //用于 控制这个 logagent 所用到的协程
	Cancel  context.CancelFunc // 控制直接中断这个logAgent
	Sender  *kafka.Writer
}

// 日志消息
type Log struct {
	Content   string
	Source    *LogAgent
	CreatedAt time.Time
}

var (
	LogChannel = make(chan *Log)
	Start      = make(chan *Collector)
	Close      = make(chan *Collector)
)

var wg sync.WaitGroup

func InitAgent(c Collector) (*LogAgent, error) {
	var fileName string

	ctx, cancel := context.WithCancel(context.Background())

	switch c.Style {
	case "File":
		fileName = c.Path
		go func() {
			select {
			case <-ctx.Done():
				Close <- &c
			}
		}()
	case "Date":
		// 现在是直接获取当前的
		formatNum := strings.LastIndex(c.Path, "*")
		if formatNum == -1 {
			cancel()
			return nil, fmt.Errorf("logagent file name(%s) format error", c.Path)
		}
		fileName = time.Now().Format(c.Path[:formatNum] + "200601021504.log")
		tick := time.Tick(10 * time.Second)
		// TODO: 之后会添加按日期归档的
		go func() {
			select {
			case <-tick:
				Close <- &c
				wg.Add(1)
				Start <- &c
				wg.Wait()
			case <-ctx.Done():
				Close <- &c
			}
		}()
	}

	config := tail.Config{
		ReOpen:    true, // true则文件被删掉阻塞等待新建该文件，false则文件被删掉时程序结束
		Follow:    true, // true则一直阻塞并监听指定文件，false则一次读完就结束程序
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2},
		MustExist: false, // true则没有找到文件就报错并结束，false则没有找到文件就阻塞保持住
		Poll:      true,  // 使用Linux的Poll函数，poll的作用是把当前的文件指针挂到等待队列
	}
	tailer, err := tail.TailFile(fileName, config)

	if err != nil {
		cancel()
		return nil, err
	}

	logAgent := &LogAgent{Tail: tailer, Topic: c.Topic, LogFile: fileName, Offset: 0, Receive: make(chan *Log), context: ctx, Cancel: cancel, Sender: sender.InitTopicWriter(c.Topic)}

	return logAgent, nil
}

func (app *App) setAgent(collector *Collector, agent *LogAgent) {
	app.mu.Lock()
	app.Agents[*collector] = agent
	app.mu.Unlock()
}

func (app *App) deleteAgent(collector *Collector) {
	app.mu.Lock()
	delete(app.Agents, *collector)
	app.mu.Unlock()
}

func (app *App) getAgent(collector *Collector) (*LogAgent, bool) {
	var result *LogAgent
	var ok bool
	app.mu.Lock()
	result, ok = app.Agents[*collector]
	app.mu.Unlock()
	return result, ok
}

func Init() *App {
	app := &App{Agents: map[Collector]*LogAgent{}}
	return app
}

func (app *App) Run() {
	// 收集所有消息，每个消息中包含了应该对应Topic的Kafka-Producer
	go masterChannel(LogChannel)

	go func() {
		var count int
		// 初始化时实际是想监控文件的通道发送一个新的文件日志代理
		for _, collector := range getEtcdCollectorConfig() {
			fmt.Println("First Add Agent :", collector)
			wg.Add(1)
			Start <- &collector
			wg.Wait()
			// time.Sleep(time.Second)
			count++
		}
		log.Printf("success start %d logagent", count)
	}()

	for {
		select {
		// 有新的小伙伴加入了监控
		case collector := <-Start:
			// fmt.Println(collector)
			// 有人要走 走的时候需要留下一些东西，记录offset等
			// 很多情况都可以触发这个行为，必须更换日期了，或者退出程序

			currentLogAgent, err := InitAgent(*collector)
			// if currentLogAgent.LogFile == "./log/access.log" {
			// 	go func ()  {
			// 		log.Println("will close ")
			// 		time.Sleep(15*time.Second)
			// 		currentLogAgent.Cancel()
			// 	}()
			// }
			if err != nil {
				log.Println(err)

			} else {
				app.setAgent(collector, currentLogAgent)
				go generator(currentLogAgent.Tail.Lines, currentLogAgent)
				go producer(currentLogAgent.Receive)
			}
			wg.Done()

		case collector := <-Close:

			// 有人要走 走的时候需要留下一些东西，记录offset等
			// 很多情况都可以触发这个行为，必须更换日期了，或者退出程序
			// fmt.Println(app.Agents)

			shutdownLogAgent, ok := app.getAgent(collector)
			if !ok {
				log.Println("close a unsafe Agent")
			}
			log.Println("closeing " + shutdownLogAgent.LogFile)
			if err := shutdownLogAgent.Tail.Stop(); err != nil {
				log.Println("failed to close tail:", err)
			}

			close(shutdownLogAgent.Receive)

			if err := shutdownLogAgent.Sender.Close(); err != nil {
				log.Println("failed to close writer:", err)
			}

			app.deleteAgent(collector)
		}
	}

}

// 消息生成器
func generator(Lines <-chan *tail.Line, sourceAgent *LogAgent) {
	for line := range Lines {
		// 将所有的消息发送到一个统一的频道用于处理消息和限流
		LogChannel <- &Log{Content: line.Text, Source: sourceAgent, CreatedAt: line.Time}
	}
}

// 生成者
func producer(messages <-chan *Log) {
	for logmsg := range messages {
		log.Printf("%s Sender to  %s - %s\n", logmsg.Content, logmsg.Source.Topic, logmsg.Source.LogFile)
		err := logmsg.Source.Sender.WriteMessages(logmsg.Source.context, kafka.Message{
			Key:   []byte(logmsg.Source.LogFile),
			Value: []byte(logmsg.Content),
		})
		if err != nil {
			log.Println("failed to write messages:", err)
		}

	}
}

func masterChannel(messages <-chan *Log) {
	for logmsg := range messages {
		log.Printf("%s Log: %s - %s\n", logmsg.Source.LogFile, logmsg.Content, logmsg.CreatedAt.Format("2006-01-02 15:04:05"))
		taskReceive := logmsg.Source.Receive
		taskReceive <- logmsg
	}
}
