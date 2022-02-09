package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/y7ut/logagent/etcd"
)

type Collector struct {
	Style string `json:"style"`
	Path  string `json:"path"`
	Topic string `json:"topic"`
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
func watchEtcdConfig(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Close Etcd Watching!")
			return
		case confResp := <-etcd.WatchLogConfToEtcd():
			for _, event := range confResp.Events {
				switch event.Type {
				case mvccpb.PUT:
					diff, status, err := getCollectorChangeWithEvent(event)
					fmt.Printf("%s is Happend", status)
					if err != nil {
						log.Println("Get Collector Chnage Event Info Error:", err)
						continue
					}
					switch status {
					default:
						continue
					case "DEL":
						Close <- &diff
					case "PUT":
						Start <- &diff
					}
				case mvccpb.DELETE:
					// 节点开启的时候不会出现突然删除的情况所以不考虑
					continue
				}
			}
		}

	}
}

// 获取Agent 中 Collector 的变更 有三种类型 CREATED  新增Agent PUT 新增Collector DEL 删除 Collector
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
