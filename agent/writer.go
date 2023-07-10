package agent

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/y7ut/logagent/conf"
	"github.com/y7ut/logagent/sender"
)

// 队列生产者，将log转化成Kafka Message
func KafkaSender(ctx context.Context) {

	var start bool

	var SenderMu sync.Mutex

	tick := time.NewTicker(3 * time.Second)

	MessageCollection := make([]kafka.Message, 0)

	writer := sender.InitWriter()

	size := conf.APPConfig.Kafka.QueueSize
	AppId := conf.APPConfig.ID
	for {
		select {
		case <-ctx.Done():
			log.Println("closeing Kafka Sender ")
			tick.Stop()
			return

		case <-tick.C:
			var currentMessages []kafka.Message

			SenderMu.Lock()
			currentLen := len(MessageCollection)
			// 如果队列装得下
			if currentLen <= size {
				currentMessages = MessageCollection
				MessageCollection = []kafka.Message{}
			} else {
				// 否则就切割一下队列
				currentMessages = MessageCollection[:size]
				MessageCollection = MessageCollection[size:]
			}
			SenderMu.Unlock()

			if currentLen != 0 && start {
				log.Printf("Sender total %d\n", len(currentMessages))

				err := writer.WriteMessages(ctx, currentMessages...)
				if err != nil {
					log.Println("failed to write messages:", err)
				}
			}

		case logmsg := <-LogChannel:
			if logmsg == nil {
				continue
			}
			var headers = []kafka.Header{}
			headers = append(headers, kafka.Header{Key: "source_agent", Value: []byte(AppId)})

			MessageCollection = append(MessageCollection, kafka.Message{
				Key:     []byte(logmsg.Source.Collector.Path),
				Value:   []byte(logmsg.Content),
				Topic:   logmsg.Source.Collector.Topic,
				Headers: headers,
			})

			start = true

		}

	}
}
