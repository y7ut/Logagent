package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/y7ut/logagent/etcd"
)

type Collector struct {
	Style  string `json:"style"`
	Path   string `json:"path"`
	Topic  string `json:"topic"`
	offect int64
}

// 文件配置加载
func loadCollectorConfig() (collectors []Collector) {

	file, _ := os.Open("./collector.json")

	defer file.Close()

	decoder := json.NewDecoder(file)

	err := decoder.Decode(&collectors)

	if err != nil {
		fmt.Println("Load Collectors Error:", err)
	}

	return collectors
}

// etcd配置加载
func getEtcdCollectorConfig() (collectors []Collector) {

	logConfig := etcd.GetLogConfToEtcd()

	err := json.Unmarshal(logConfig, &collectors)

	if err != nil {
		fmt.Println("Load Collectors Error:", err)
	}
	log.Println("load config success!")
	return collectors
}
