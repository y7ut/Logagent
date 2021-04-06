package sender

import (
	"github.com/y7ut/logagent/conf"
	"github.com/segmentio/kafka-go"
)

func InitTopicWriter(topic string) *kafka.Writer {
	// make a writer that produces to topic-A, using the least-bytes distribution
	w := &kafka.Writer{
		Addr:     kafka.TCP(conf.APPConfig.Kafka.Address),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return w
}
