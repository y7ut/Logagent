package agent

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/y7ut/logagent/etcd"
)

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
	app := &App{Agents: make(map[string]*LogAgent)}
	return app
}

func (app *App) Run() {

	Ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 收集所有消息，每个消息中包含了应该对应Topic的Kafka-Producer
	go KafkaSender(Ctx)

	// 监听ETCD中Collector
	go watchEtcdConfig(Ctx)

	// 代理激活
	go app.AgentRegister(Ctx)

	// 代理关闭
	go app.AgentLeave()

	count := app.RegisterFirst()

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
		Close <- &logagent.Collector
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
