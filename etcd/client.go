package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/y7ut/logagent/conf"
)

var (
	configPath = "/logagent/config/"
	statusPath = "/logagent/active/"
)

var cli *clientv3.Client

func Init() {
	cli = connect()
}

func connect() *clientv3.Client {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(conf.APPConfig.Etcd.Address, ","),
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(fmt.Sprintf("connect failed, err:%s \n", err))
	}

	log.Println("connect etcd succ")
	return cli
}

func GetLogConfToEtcd() []byte {
	key := configPath + conf.APPConfig.ID
	activeKey := statusPath + conf.APPConfig.ID

	// 注册激活状态
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err := cli.Put(ctx, activeKey, "1")
	cancel()
	if err != nil {
		panic(fmt.Sprintf("get failed, err:%s \n", err))
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	resp, err := cli.Get(ctx, key)
	cancel()
	if err != nil {
		panic(fmt.Sprintf("get failed, err:%s \n", err))
	}

	// 如果没有这个节点 那就新增这个节点并且注册为空
	if len(resp.Kvs) == 0 {
		// Json打包
		var emptyLogConfArr []struct{}

		data, err := json.Marshal(emptyLogConfArr)
		if err != nil {

			panic(fmt.Sprintf("json failed, err:%s \n", err))

		}
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
		_, err = cli.Put(ctx, key, string(data))
		cancel()

		if err != nil {
			panic(fmt.Sprintf("put failed, err:%s \n", err))
		}
		return data
	}

	return resp.Kvs[0].Value
}

func CloseEvent() {
	activeKey := statusPath + conf.APPConfig.ID

	defer func() {
		err := cli.Close()
		if err != nil {
			panic(fmt.Sprintf("close failed, err:%s \n", err))
		}
		log.Println("close etcd succ")
	}()

	// 注册激活状态
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err := cli.Put(ctx, activeKey, "0")
	cancel()
	if err != nil {
		panic(fmt.Sprintf("get failed, err:%s \n", err))
	}
}

func WatchLogConfToEtcd() clientv3.WatchChan {

	key := configPath + conf.APPConfig.ID

	wch := cli.Watch(context.Background(), key)

	return wch
}
