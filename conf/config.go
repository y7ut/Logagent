package conf

type LogAgentConf struct {
	App   `ini:"app"`
	Kafka `ini:"kafka"`
	Etcd  `ini:"etcd"`
}

// kafka 配置
type Kafka struct {
	Address   string `ini:"address"`
	QueueSize string `ini:"queue_size"`
}

// APP 属性
type App struct {
	ID string `ini:"logagent_id"`
}

// ETCD 配置
type Etcd struct {
	Address string `ini:"address"`
}

var (
	APPConfig = new(LogAgentConf)
)
