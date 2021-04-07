package agent

import (
	"encoding/json"
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
		log.Println("Load Collectors Error:", err)
	}

	return collectors
}

// etcd配置加载
func getEtcdCollectorConfig() (collectors []Collector) {

	logConfig := etcd.GetLogConfToEtcd()

	err := json.Unmarshal(logConfig, &collectors)

	if err != nil {
		log.Println("Load Collectors Error:", err)
	}
	log.Println("load config success!")
	return collectors
}

// 返回
func watchEtcdConfig() chan []Collector {
	handleCollector := make(chan []Collector)

	go func ()  {
		etcdConfChan := etcd.WatchLogConfToEtcd()

		for confResp := range etcdConfChan {
			// 这里我们只分析第一个事件就可以
			var collectors []Collector
	
			changedConf := confResp.Events[0].Kv.Value
			err := json.Unmarshal(changedConf, &collectors)
	
			if err != nil {
				log.Println("Load New Collectors Error:", err)
			}
			log.Println("load New config success!")
			
			handleCollector<- collectors
		}
	}()
	return handleCollector
}
