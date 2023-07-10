package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hpcloud/tail"
)

type App struct {
	Agents map[string]*LogAgent
	mu     sync.Mutex
}

type LogAgent struct {
	Tail      *tail.Tail // 这个代理的tail
	Collector Collector  // 所服务的收集任务
	Offset    int64      // 偏移值
	Cancel    context.CancelFunc
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

func (app *App) setAgent(path string, agent *LogAgent) {
	app.mu.Lock()
	app.Agents[path] = agent
	app.mu.Unlock()
}

func (app *App) deleteAgent(path string) {
	app.mu.Lock()
	delete(app.Agents, path)
	app.mu.Unlock()
}

func (app *App) getAgent(path string) (*LogAgent, bool) {
	var result *LogAgent
	var ok bool
	app.mu.Lock()
	result, ok = app.Agents[path]
	app.mu.Unlock()
	return result, ok
}

func (app *App) allAgent() (Agents map[string]*LogAgent) {
	app.mu.Lock()
	Agents = app.Agents
	app.mu.Unlock()
	return Agents
}

func (app *App) ListAgent() map[*LogAgent]bool {
	result := make(map[*LogAgent]bool)
	app.mu.Lock()
	for _, v := range app.Agents {
		result[v] = true
	}
	app.mu.Unlock()
	return result
}

func InitAgent(c Collector, ctx context.Context) (*LogAgent, error) {
	var fileName string
	ctx, cancel := context.WithCancel(ctx)

	switch c.Style {
	case "File":
		fileName = c.Path
	case "Date":
		if !(strings.Contains(c.Path, "2006-01-02") || strings.Contains(c.Path, "20060102")) {
			cancel()
			return nil, fmt.Errorf("logagent file name(%s) format error", c.Path)
		}
		now := time.Now()
		fileName = now.Format(c.Path)
		yyyy, mm, dd := now.Date()
		LifeCycle := time.Date(yyyy, mm, dd+1, 0, 0, 0, 0, now.Location()).Sub(now)
		go func() {
			select {
			case <-time.After(LifeCycle):
				// 正常的日期格式任务，有一个一天的倒计时
				// 退出当天的
				Close <- &c
				time.Sleep(time.Second)
				// 为这个任务刷新时间
				Start <- &c

				return
			case <-ctx.Done():
				// 不触发倒计时:
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

	return &LogAgent{Tail: tailer, Collector: c, Offset: 0, Cancel: cancel}, nil
}

// 监听创建事件
func (app *App) RegisterFirst() int {
	var count int
	configFromEtcd, err := getEtcdCollectorConfig()
	if err != nil {
		log.Panic(err)
	}

	for _, collector := range configFromEtcd {
		_, ok := app.getAgent(collector.Path)
		if ok {
			log.Println("already exist:" + collector.Path)
			continue
		}
		Start <- &collector
		count++
		log.Println("init Add Agent :", collector)
		time.Sleep(500 * time.Millisecond)
	}
	return count
}

// 监听创建事件
func (app *App) AgentRegister(ctx context.Context) {
	for {
		select {
		// 激活协程使用CTX控制，可以上文关闭后，关闭激活协程，以及子协程
		case <-ctx.Done():
			return
		// 有新的小伙伴加入了监控
		case collector := <-Start:
			// 创建Agent
			currentLogAgent, err := InitAgent(*collector, ctx)

			if err != nil {
				log.Println(err)
				continue
			}
			// 激活注册创建的Agent
			app.setAgent(collector.Path, currentLogAgent)

			go generator(currentLogAgent)

			log.Println("watching ", currentLogAgent.Collector.Path)

		}
	}
}

// 监听离开事件， 这个事件脱离全局上下文的范围，独立存在
func (app *App) AgentLeave() {
	for collector := range Close {

		app.mu.Lock()
		for k := range app.Agents {
			fmt.Println(k)
		}
		app.mu.Unlock()

		// 很多情况都可以触发这个行为，必须更换日期了，或者退出程序
		shutdownLogAgent, ok := app.getAgent(collector.Path)
		if !ok {
			log.Println("close a unsafe Agent:", collector.Path)
			continue
		}

		// 退出之前，记录 offset
		offset, _ := shutdownLogAgent.Tail.Tell()
		log.Printf("closeing  %s  , with offset at %d:", shutdownLogAgent.Collector.Path, offset)
		putLogFileOffset(shutdownLogAgent.Tail.Filename, offset)

		if err := shutdownLogAgent.Tail.Stop(); err != nil {
			log.Println("failed to close tail:", err)
			continue
		}

		// 为时间型任务的取消掉自爆协程
		shutdownLogAgent.Cancel()

		app.deleteAgent(shutdownLogAgent.Collector.Path)
		log.Println("Close collector success :", collector)
	}
}
