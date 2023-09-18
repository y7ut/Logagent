package agent

import (
	"sync"
	"time"
)

var logPool = sync.Pool{
	New: func() interface{} {
		return &Log{}
	},
}

// 日志消息
type Log struct {
	Content   string
	Source    *LogAgent
	CreatedAt time.Time
}

// 从日志池中获取一个
func NewLog(content string, source *LogAgent, createdAt time.Time) *Log {
	log := logPool.Get().(*Log)
	log.Content = content
	log.Source = source
	log.CreatedAt = createdAt
	return log
}

// 放回日志池
func (log *Log) Reset() {
	logPool.Put(log)
}
