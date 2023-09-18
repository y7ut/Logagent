package conf

type LogAgentConf struct {
	App     `ini:"app"`
	Kafka   `ini:"kafka"`
	Etcd    `ini:"etcd"`
	Runtime `ini:"runtime"`
	Log     `ini:"log"`
}

// kafka 配置
type Kafka struct {
	Address   string `ini:"address"`
	QueueSize int    `ini:"queue_size"`
}

// APP 属性
type App struct {
	ID string `ini:"logagent_id"`
}

// ETCD 配置
type Etcd struct {
	Address string `ini:"address"`
}

type Runtime struct {
	Path string `ini:"path"`
}

type Log struct {
	Path string `ini:"path"`
	Name string `ini:"name"`
}

var (
	APPConfig = new(LogAgentConf)
)
