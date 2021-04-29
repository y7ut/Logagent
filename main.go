package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/y7ut/logagent/agent"
	"github.com/y7ut/logagent/conf"
	"gopkg.in/ini.v1"
)

const version = "1.0.2"

var c = flag.String("c", "./logagent.conf", "使用配置文件启动")
var v = flag.Bool("v", false, "查看当前程序版本")

func main() {
	// 加载配置文件
	flag.Parse()

	if *v {
		fmt.Println("Jiwei Logagent \nversion: ", version)
		return
	}

	if err := ini.MapTo(conf.APPConfig, *c); err != nil {
		log.Println("load ini file error: ", err)
		return
	}

	app := agent.Init()
	app.Run()
}
