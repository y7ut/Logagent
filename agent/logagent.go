package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hpcloud/tail"
)

type LogAgent struct {
	Offset    int64         // 偏移值
	done      chan struct{} // 结束信号
	Tail      *tail.Tail    // 这个代理的tail
	Collector Collector     // 所服务的收集任务
	cycle     time.Duration // 周期

}

func NewAgent(c Collector) (*LogAgent, error) {
	var fileName string
	var LifeCycle time.Duration

	switch c.Style {
	case "File":
		fileName = c.Path
	case "Date":
		if !(strings.Contains(c.Path, "2006-01-02") || strings.Contains(c.Path, "20060102")) {
			return nil, fmt.Errorf("logagent file name(%s) format error", c.Path)
		}
		now := time.Now()
		fileName = now.Format(c.Path)
		yyyy, mm, dd := now.Date()
		LifeCycle = time.Date(yyyy, mm, dd+1, 0, 0, 0, 0, now.Location()).Sub(now)
	default:
		return nil, fmt.Errorf("logagent Type(%s) format error", c.Style)
	}

	offset, err := getLogFileOffset(app.runtimePath, fileName)

	if err != nil {
		log.Printf("can not load offset num from %s", fileName)
		offset = 0
	}
	log.Printf("load offset num from %s is %d", fileName, offset)
	config := tail.Config{
		ReOpen:    true, // true则文件被删掉阻塞等待新建该文件，false则文件被删掉时程序结束
		Follow:    true, // true则一直阻塞并监听指定文件，false则一次读完就结束程序
		Location:  &tail.SeekInfo{Offset: offset, Whence: 1},
		MustExist: false, // true则没有找到文件就报错并结束，false则没有找到文件就阻塞保持住
		Poll:      true,  // 使用Linux的Poll函数，poll的作用是把当前的文件指针挂到等待队列
		Logger:    tail.DiscardingLogger,
	}

	tailer, err := tail.TailFile(fileName, config)
	if err != nil {
		return nil, err
	}
	return &LogAgent{Tail: tailer, Collector: c, Offset: 0, cycle: LifeCycle, done: make(chan struct{})}, nil
}

func (l *LogAgent) tailLines() <-chan *tail.Line {
	return l.Tail.Lines
}

// Start 启动
func (l *LogAgent) Start(ctx context.Context) {
	go func(ctx context.Context) {
		defer func() {
			if err := l.exitTail(); err != nil {
				log.Printf("failed to close tailer: %v", err)
			}
			log.Printf("success to close %s tailer", l.Collector.Path)
		}()
		for {
			select {
			case <-ctx.Done():
				// 退出(Cancel)
				return
			case <-l.done:
				// 退出
				return
			case line := <-l.tailLines():
				// 将所有的消息发送到一个统一的频道用于处理消息和限流
				LogChannel <- NewLog(line.Text, l, line.Time)
			}
		}
	}(ctx)
	// 如果是滚动类型的日志，会每隔一段时间就会重启一次
	if l.cycle > 0 && l.Collector.Style == "Date" {
		go func(ctx context.Context) {
			select {
			case <-time.After(l.cycle):
				// 正常的日期格式任务，有一个一天的倒计时
				// 退出当天的任务，然后重新启动
				// 注意这里不需要去退出内部的tailer啥的，统一交给上面协程中的defer去处理
				CloseChan <- l.Collector
				time.Sleep(500 * time.Millisecond)
				StartChan <- l.Collector
				log.Printf("success refresh %s cycle", l.Collector.Path)
				return
			case <-l.done:
				// 退出
				log.Printf("stop refresh %s cycle", l.Collector.Path)
				return
			case <-ctx.Done():
				// 不触发倒计时:
				log.Printf("stop refresh %s cycle", l.Collector.Path)
				return
			}
		}(ctx)
	}
}

// exitTail 取消这个任务中的监听tailer
func (l *LogAgent) exitTail() error {
	// 退出之前，记录 offset
	offset, err := l.Tail.Tell()
	if err != nil {
		return err
	}
	if err := l.Tail.Stop(); err != nil {
		return err
	}

	putLogFileOffset(app.runtimePath, l.Tail.Filename, offset)
	return nil
}

// Stop 停止任务
func (l *LogAgent) Stop() error {
	l.done <- struct{}{}
	return nil
}
