package etcd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/y7ut/logagent/conf"
	clientv3 "go.etcd.io/etcd/client/v3"
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
	return cli
}

func GetLogConfToEtcd() ([]byte, error) {
	key := configPath + conf.APPConfig.ID

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	resp, err := cli.Get(ctx, key)
	cancel()
	if err != nil {
		return []byte{}, fmt.Errorf("get failed, err:%s ", err)
	}

	// 如果没有这个节点 那就新增这个节点并且注册为空
	if len(resp.Kvs) == 0 {
		return []byte{}, fmt.Errorf("agent (%s) has not registed", conf.APPConfig.ID)
	}

	activeKey := statusPath + conf.APPConfig.ID

	// 注册激活状态
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	_, err = cli.Put(ctx, activeKey, "1")
	cancel()
	if err != nil {
		return []byte{}, fmt.Errorf("get failed, err:%s ", err)
	}

	return resp.Kvs[0].Value, nil
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

func GetDelRevValueFromEtcd(key string, rev int64) ([]byte, error) {
	var value []byte

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	resp, err := cli.Get(ctx, key, clientv3.WithRev(rev))
	cancel()
	if err != nil {
		log.Println(fmt.Sprintf("Get Etcd config failed, err:%s \n", err))
	}

	if len(resp.Kvs) == 0 {
		return value, fmt.Errorf("config get error")
	}

	return resp.Kvs[0].Value, nil
}
