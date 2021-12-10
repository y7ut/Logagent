package sender

import (
	"strings"

	"github.com/segmentio/kafka-go"
	"github.com/y7ut/logagent/conf"
)

func InitTopicWriter(topic string) *kafka.Writer {
	// make a writer that produces to topic-A, using the least-bytes distribution
	w := &kafka.Writer{
		Addr:     kafka.TCP(strings.Split(conf.APPConfig.Kafka.Address, ",")...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return w
}
