package agent

import (
	"context"
	"encoding/json"
	"log"

	"github.com/y7ut/logagent/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// getEtcdCollectorConfig etcd配置加载
func getEtcdCollectorConfig() (collectors []Collector, err error) {

	logConfig, err := etcd.GetLogConfToEtcd()

	if err != nil {
		return collectors, err
	}

	err = json.Unmarshal(logConfig, &collectors)

	if err != nil {
		return collectors, err
	}
	log.Println("load config success!")
	return collectors, nil
}

// watchEtcdConfig 监听etcd中的事件，会通过下面的 Get Change 计算出事件的变更
func watchEtcdConfig(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Close Etcd Watching!")
			return
		case confResp := <-etcd.WatchLogConfToEtcd():
			for _, event := range confResp.Events {
				switch event.Type {
				case clientv3.EventTypePut:
					diff, status, err := getCollectorChangeWithEvent(event)
					log.Printf("ETCD watch %s is Happend", status)
					if err != nil {
						log.Println("Get Collector Chnage Event Info Error:", err)
						continue
					}
					switch status {
					default:
						continue
					case "DEL":
						CloseChan <- diff
					case "PUT":
						StartChan <- diff
					}
				case clientv3.EventTypeDelete:
					// 节点开启的时候不会出现突然删除的情况所以不考虑
					continue
				}
			}
		}

	}
}

// getCollectorChangeWithEvent 获取Agent 中 Collector 的变更
// 有三种类型 changeType
// 1: CREATED  初始化, 一般指新增了 Agent，还没有注册 Collector
// 2: PUT      新增了 Collector
// 3: DEL      删除了 Collector
func getCollectorChangeWithEvent(event *clientv3.Event) (different Collector, changeType string, err error) {

	var currentCollectors []Collector

	changedConf := event.Kv.Value

	err = json.Unmarshal(changedConf, &currentCollectors)
	if err != nil {
		return different, changeType, err
	}

	var oldCollector []Collector

	oldKey := event.Kv.Key
	rev := event.Kv.ModRevision - 1

	oldValue, err := etcd.GetDelRevValueFromEtcd(string(oldKey), rev)

	if err != nil {
		if len(currentCollectors) == 0 {
			changeType = "CREATED"
			err = nil
		}
		return different, changeType, err
	}

	err = json.Unmarshal(oldValue, &oldCollector)

	if err != nil {
		return different, changeType, err
	}

	var set = make(map[Collector]bool)

	if len(oldCollector)-len(currentCollectors) < 0 {
		changeType = "PUT"

		for _, item := range oldCollector {
			set[item] = true
		}

		for _, item := range currentCollectors {
			if !set[item] {
				different = item
			}
		}
	} else {
		changeType = "DEL"
		for _, item := range currentCollectors {
			set[item] = true
		}

		for _, item := range oldCollector {
			if !set[item] {
				different = item
			}
		}
	}

	return different, changeType, err
}
