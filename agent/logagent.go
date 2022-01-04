package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hpcloud/tail"
	"github.com/segmentio/kafka-go"
	"github.com/y7ut/logagent/conf"
	"github.com/y7ut/logagent/etcd"
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
	LogChannel = make(chan *Log, 50)
	Start      = make(chan *Collector)
	Close      = make(chan *Collector)
	dataPath   = "./runtime/"
)

func (app *App) setAgent(collector Collector, agent *LogAgent) {
	app.mu.Lock()
	app.Agents[collector] = agent
	app.mu.Unlock()
}

func (app *App) deleteAgent(collector Collector) {
	app.mu.Lock()
	delete(app.Agents, collector)
	app.mu.Unlock()
}

func (app *App) getAgent(collector Collector) (*LogAgent, bool) {
	var result *LogAgent
	var ok bool
	app.mu.Lock()
	result, ok = app.Agents[collector]
	app.mu.Unlock()
	return result, ok
}

func (app *App) allAgent() (Agents map[Collector]*LogAgent) {
	app.mu.Lock()
	Agents = app.Agents
	app.mu.Unlock()
	return Agents
}

func (app *App) listAgent() map[Collector]bool {
	result := make(map[Collector]bool)
	app.mu.Lock()
	for k := range app.Agents {
		// log.Println(k)
		result[k] = true
	}
	app.mu.Unlock()
	return result
}

var wg sync.WaitGroup
var exitwg sync.WaitGroup

func InitAgent(c Collector) (*LogAgent, error) {
	var fileName string

	ctx, cancel := context.WithCancel(context.Background())

	switch c.Style {
	case "File":
		fileName = c.Path
		go func() {
			<-ctx.Done()
			Close <- &c
		}()
	case "Date":
		if !(strings.Contains(c.Path, "2006-01-02") || strings.Contains(c.Path, "20060102")) {
			cancel()
			return nil, fmt.Errorf("logagent file name(%s) format error", c.Path)
		}
		now := time.Now()
		fileName = now.Format(c.Path)
		yyyy, mm, dd := now.Date()
		go func() {
			select {
			case <-time.After(time.Date(yyyy, mm, dd+1, 0, 0, 0, 0, now.Location()).Sub(now)):
				// 一个引爆倒计时
				exitwg.Add(1)
				Close <- &c
				exitwg.Wait()
				time.Sleep(500 * time.Millisecond)
				wg.Add(1)
				Start <- &c
				wg.Wait()
				return
			case <-ctx.Done():
				// 不触发倒计时
				Close <- &c
				return
			}
		}()
	default:
		cancel()
		return nil, fmt.Errorf("logagent Type(%s) format error", c.Style)
	}

	offset, err := getLogFileOffset(fileName)

	if err != nil {
		log.Printf("can not load offset num from %s", fileName)
		offset = 0
	} else {
		log.Printf("success load offset num from %s", fileName)
	}

	config := tail.Config{
		ReOpen:    true, // true则文件被删掉阻塞等待新建该文件，false则文件被删掉时程序结束
		Follow:    true, // true则一直阻塞并监听指定文件，false则一次读完就结束程序
		Location:  &tail.SeekInfo{Offset: offset, Whence: 1},
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

func Init() *App {

	_, err := os.Stat(dataPath)

	if err != nil && os.IsNotExist(err) {
		log.Println("runtime dir not found.")
		err := os.Mkdir(dataPath, os.ModePerm)
		if err != nil {
			panic("create Runtime dir Error")
		} else {
			log.Println("create runtime dir success.")
		}
	}

	etcd.Init()
	app := &App{Agents: make(map[Collector]*LogAgent)}
	return app
}

func (app *App) Run() {

	// 收集所有消息，每个消息中包含了应该对应Topic的Kafka-Producer
	go masterChannel(LogChannel)

	// 结束和关闭
	go func() {
		for {
			select {
			// 有新的小伙伴加入了监控
			case collector := <-Start:
				currentLogAgent, err := InitAgent(*collector)
				if err != nil {
					log.Println(err)
					wg.Done()
					continue
				}

				app.setAgent(*collector, currentLogAgent)

				go generator(currentLogAgent)
				go bigProducer(currentLogAgent)
				log.Println("watching ", currentLogAgent.LogFile)
				wg.Done()

			case collector := <-Close:
				//用上下文cancel触发关闭行为

				// 很多情况都可以触发这个行为，必须更换日期了，或者退出程序
				shutdownLogAgent, ok := app.getAgent(*collector)

				if !ok {
					log.Println("close a unsafe Agent")
					continue
				}

				// 记录offset
				offset, _ := shutdownLogAgent.Tail.Tell()
				log.Printf("closeing  %s  offset at %d:", shutdownLogAgent.LogFile, offset)
				putLogFileOffset(shutdownLogAgent.Tail.Filename, offset)

				if err := shutdownLogAgent.Tail.Stop(); err != nil {
					exitwg.Done()
					log.Println("failed to close tail:", err)
					continue
				}

				app.deleteAgent(*collector)
				log.Println("Close collector success :", collector)
				exitwg.Done()
			}
		}
	}()
	// 热启动
	go func() {
		for eidtCollectors := range watchEtcdConfig() {
			listAgent := app.listAgent()
			for _, c := range eidtCollectors {
				_, ok := app.getAgent(c)
				if ok {
					// 如果有的话就hash一下
					listAgent[c] = false
				} else {
					//没有发送到新增chan
					wg.Add(1)
					log.Println("Eidt Add Agent :", c)
					Start <- &c
					wg.Wait()
					time.Sleep(500 * time.Millisecond)
				}
			}
			for c, alive := range listAgent {
				if alive {
					//没有保存的删除了
					log.Println("Eidt Close Agent :", c)
					exitwg.Add(1)
					currentAgent, ok := app.getAgent(c)
					if ok {
						currentAgent.Cancel()
					}
					exitwg.Wait()
					time.Sleep(500 * time.Millisecond)
				}
			}
		}
	}()

	var count int
	// 初始化时实际是想监控文件的通道发送一个新的文件日志代理
	initCollectors, err := getEtcdCollectorConfig()
	if err != nil {
		log.Printf("Init Ectd Config error: %s", err)
		return
	}
	wg.Add(len(initCollectors))
	for _, collector := range initCollectors {
		_, ok := app.getAgent(collector)
		if ok {
			log.Println("already exist:" + collector.Path)
			continue
		}
		Start <- &collector
		count++
		log.Println("init Add Agent :", collector)
		time.Sleep(500 * time.Millisecond)
	}
	wg.Wait()
	log.Printf("total start %d logagent\n", count)

	for s := range sign() {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
			log.Println("Safe Exit:", s)
			app.safeExit()
		}
	}

}

func sign() <-chan os.Signal {
	c := make(chan os.Signal, 2)

	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2}
	// 监听信号
	if !signal.Ignored(syscall.SIGHUP) {
		signals = append(signals, syscall.SIGHUP)
	}

	signal.Notify(c, signals...)

	return c
}


func (app *App) safeExit() {
	AllAgents := app.allAgent()

	for c, logagent := range AllAgents {
		//没有保存的删除了
		exitwg.Add(1)
		log.Println("Safe Close Agent of :", c)
		logagent.Cancel()
		exitwg.Wait()
		time.Sleep(500 * time.Millisecond)
	}

	etcd.CloseEvent()
	os.Exit(0)
}

// 消息生成器
func generator(sourceAgent *LogAgent) {
	for line := range sourceAgent.Tail.Lines {
		// 将所有的消息发送到一个统一的频道用于处理消息和限流
		LogChannel <- &Log{Content: line.Text, Source: sourceAgent, CreatedAt: line.Time}
	}

}

// 生產者
func bigProducer(agent *LogAgent) {
	tick := time.NewTicker(1 * time.Second)
	var (
		start      bool
		topic      string
		SenderMu   sync.Mutex
		bigMessage []kafka.Message
		sender     *kafka.Writer
		ctx        context.Context
	)
	size := conf.APPConfig.Kafka.QueueSize
	for {
		select {
		case logmsg := <-agent.Receive:
			if logmsg == nil {
				continue
			}
			bigMessage = append(bigMessage, kafka.Message{
				Key:   []byte(logmsg.Source.LogFile),
				Value: []byte(logmsg.Content),
			})
			ctx = logmsg.Source.context
			sender = logmsg.Source.Sender
			topic = logmsg.Source.Topic
			start = true
		case <-agent.context.Done():
			tick.Stop()
			log.Println("closeing Kafka Sender of ", agent.LogFile)
			if err := agent.Sender.Close(); err != nil {
				log.Println("failed to close writer:", err)
			}
			return
		case <-tick.C:
			var currentMessages []kafka.Message
			SenderMu.Lock()
			currentLen := len(bigMessage)
			// 如果队列装得下
			if currentLen <= size {
				currentMessages = bigMessage
				bigMessage = []kafka.Message{}
			} else {
				// 否则就切割一下队列
				currentMessages = bigMessage[:size]
				bigMessage = bigMessage[size:]
			}
			SenderMu.Unlock()

			if currentLen != 0 && start {
				log.Printf("Sender to  %s total %d\n", topic, len(currentMessages))
				err := sender.WriteMessages(ctx, currentMessages...)

				if err != nil {
					log.Println("failed to write messages:", err)
				}
			}
		}
	}
}

func masterChannel(messages <-chan *Log) {
	for logmsg := range messages {
		// log.Printf("%s Log: %s - %s\n", logmsg.Source.LogFile, logmsg.Content, logmsg.CreatedAt.Format("2006-01-02 15:04:05"))
		taskReceive := logmsg.Source.Receive
		select {
		case <-logmsg.Source.context.Done():
			close(taskReceive)
			log.Println("closeing Kafka Sender of ", logmsg.Source.LogFile)
		default:

			taskReceive <- logmsg
		}

	}
}
