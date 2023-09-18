package agent

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/y7ut/logagent/conf"
	"github.com/y7ut/logagent/sender"
)

var (
	LogChannel = make(chan *Log, 50)
)

// 队列生产者，将log转化成Kafka Message
func KafkaSender(ctx context.Context) {

	// 用来判断是否已经启动
	var start bool

	tick := time.NewTicker(3 * time.Second)
	MessageCollection := make([]kafka.Message, 0)
	writer := sender.InitWriter()
	bufferSize := conf.APPConfig.Kafka.QueueSize
	constHeaders := []kafka.Header{{Key: "source_agent", Value: []byte(conf.APPConfig.ID)}}

	for {
		select {
		case <-ctx.Done():
			log.Println("closeing Kafka Sender ")
			tick.Stop()
			return

		case <-tick.C:
			// 按照时间来判断缓冲区队列是否已满

			currentLen := len(MessageCollection)

			if currentLen != 0 && start {
				var MessageBox = make([]kafka.Message, currentLen)
				if currentLen > bufferSize {
					// 装不下切割一下
					copy(MessageBox, MessageCollection[:bufferSize])
					MessageCollection = MessageCollection[bufferSize:]
				} else {
					// 如果缓冲区装得下，则全部写入
					copy(MessageBox, MessageCollection)
					MessageCollection = MessageCollection[:0]
				}

				log.Printf("Sender total %d\n", len(MessageBox))
				err := writer.WriteMessages(ctx, MessageBox...)
				if err != nil {
					log.Println("failed to write messages:", err)
				}
			}

		case logmsg := <-LogChannel:
			if logmsg == nil {
				continue
			}
			MessageCollection = append(MessageCollection, kafka.Message{
				Key:     []byte(logmsg.Source.Collector.Path),
				Value:   []byte(logmsg.Content),
				Topic:   logmsg.Source.Collector.Topic,
				Headers: constHeaders,
			})

			// 释放一下日志对象
			logmsg.Reset()
			start = true

			// 如果缓冲区装不下了，就触发写入
			if len(MessageCollection) > bufferSize {
				// 装不下切割一下
				var MessageBox = make([]kafka.Message, bufferSize)
				copy(MessageBox, MessageCollection[:bufferSize])
				MessageCollection = MessageCollection[bufferSize:]
				log.Printf("Sender total %d\n", len(MessageBox))
				err := writer.WriteMessages(ctx, MessageBox...)
				if err != nil {
					log.Println("failed to write messages:", err)
				}
			}
		}

	}
}
