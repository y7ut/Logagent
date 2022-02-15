package agent

// 消息生成器
func generator(sourceAgent *LogAgent) {
	for line := range sourceAgent.Tail.Lines {
		// 将所有的消息发送到一个统一的频道用于处理消息和限流
		LogChannel <- &Log{Content: line.Text, Source: sourceAgent, CreatedAt: line.Time}
	}
}
