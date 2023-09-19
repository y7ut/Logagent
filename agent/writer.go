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
	writer := sender.InitWriter()
	bufferSize := conf.APPConfig.Kafka.QueueSize
	constHeaders := []kafka.Header{{Key: "source_agent", Value: []byte(conf.APPConfig.ID)}}

	MessageCollection := make([]kafka.Message, 0)
	MessageAlreadySend := make([]kafka.Message, 0, bufferSize)

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

				if currentLen > bufferSize {
					// 装不下切割一下
					// copy 的话需要提前准备好slice的长度，感觉不是很好
					// copy(MessageAlreadySend, MessageCollection[:bufferSize])
					MessageAlreadySend = append(MessageAlreadySend, MessageCollection[:bufferSize]...)
					MessageCollection = MessageCollection[bufferSize:]
				} else {
					// 如果缓冲区装得下，则全部写入
					MessageAlreadySend = append(MessageAlreadySend, MessageCollection...)
					MessageCollection = MessageCollection[:0]
				}

				log.Printf("Sender total %d\n", len(MessageAlreadySend))
				err := writer.WriteMessages(ctx, MessageAlreadySend...)
				if err != nil {
					log.Println("failed to write messages:", err)
				}
				MessageAlreadySend = MessageAlreadySend[:0]
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
				MessageAlreadySend = append(MessageAlreadySend, MessageCollection[:bufferSize]...)
				MessageCollection = MessageCollection[bufferSize:]
				log.Printf("Sender total %d\n", len(MessageAlreadySend))
				err := writer.WriteMessages(ctx, MessageAlreadySend...)
				if err != nil {
					log.Println("failed to write messages:", err)
				}
				MessageAlreadySend = MessageAlreadySend[:0]
			}
		}

	}
}
