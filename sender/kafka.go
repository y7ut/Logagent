package sender

import (
	"strings"

	"github.com/segmentio/kafka-go"
	"github.com/y7ut/logagent/conf"
)

func InitWriter() *kafka.Writer {
	w := &kafka.Writer{
		Addr:     kafka.TCP(strings.Split(conf.APPConfig.Kafka.Address, ",")...),
		Balancer: &kafka.LeastBytes{},
	}
	return w
}
