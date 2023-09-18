package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/y7ut/logagent/etcd"
)

var (
	StartChan = make(chan Collector)
	CloseChan = make(chan Collector)
)

type App struct {
	runtimePath string
	Agents      map[string]*LogAgent
	mu          sync.Mutex
}

func NewApp(runtimePath string) *App {
	return &App{runtimePath: runtimePath, Agents: make(map[string]*LogAgent)}
}

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
	allAgents := app.allAgent()

	for _, agent := range allAgents {
		result[agent] = true
	}

	return result
}

// 监听创建事件
func (app *App) RegisterFirst() (int, error) {
	var count int
	configFromEtcd, err := getEtcdCollectorConfig()
	if err != nil {
		return 0, err
	}

	for _, collector := range configFromEtcd {
		_, ok := app.getAgent(collector.Path)
		if ok {
			log.Println("already exist:" + collector.Path)
			continue
		}
		StartChan <- collector
		count++
		log.Println("init Add Agent :", collector)
		time.Sleep(300 * time.Millisecond)
	}

	return count, nil
}

// 监听创建事件
func (app *App) ListenCollectorStart(ctx context.Context) {
	for {
		select {
		// 激活协程使用CTX控制，可以上文关闭后，关闭激活协程，以及子协程
		case <-ctx.Done():
			return
		// 有新的小伙伴加入了监控
		case collector := <-StartChan:

			// 创建Agent
			currentLogAgent, err := NewAgent(collector)

			if err != nil {
				log.Println(err)
				continue
			}
			// 激活注册创建的Agent
			app.setAgent(collector.Path, currentLogAgent)

			// 启动Agent
			currentLogAgent.Start(ctx)

			log.Println("watching ", currentLogAgent.Collector.Path)
		}
	}
}

// 监听离开事件， 这个事件脱离全局上下文的范围，独立存在
func (app *App) ListenCollectorClose() {
	for {

		collector := <-CloseChan

		// 很多情况都可以触发这个行为，必须更换日期了，或者退出程序
		shutdownLogAgent, ok := app.getAgent(collector.Path)
		if !ok {
			log.Println("close a unsafe Agent:", collector.Path)
			continue
		}
		// 停止Agent
		shutdownLogAgent.Stop()

		app.deleteAgent(shutdownLogAgent.Collector.Path)
		log.Println("already close Agent collector for ", shutdownLogAgent.Collector.Path)
	}
}

func (app *App) Run() {

	Ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
	}()

	etcd.Init()

	// 收集所有消息，每个消息中包含了应该对应Topic的Kafka-Producer
	go KafkaSender(Ctx)

	// 监听ETCD中Collector
	go watchEtcdConfig(Ctx)

	// 代理激活
	go app.ListenCollectorStart(Ctx)

	// 代理关闭
	go app.ListenCollectorClose()

	count, err := app.RegisterFirst()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	log.Printf("total start %d logagent\n", count)

	for s := range sign() {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
			log.Println("Safe Exit:", s)
			app.safeExit()
			return
		}
	}

}

func (app *App) safeExit() {
	AllAgents := app.allAgent()
	for _, logagent := range AllAgents {
		//没有保存的删除了
		CloseChan <- logagent.Collector
		time.Sleep(500 * time.Millisecond)
	}

	etcd.CloseEvent()
	os.Exit(0)
}

func sign() <-chan os.Signal {
	c := make(chan os.Signal, 2)

	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2}

	// 监听信号, 判断是否忽略 sighup 信号量
	if !signal.Ignored(syscall.SIGHUP) {
		signals = append(signals, syscall.SIGHUP)
	}

	signal.Notify(c, signals...)

	return c
}
