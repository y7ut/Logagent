package main

import (
	"log"
	"os"
	"runtime/pprof"

	"github.com/y7ut/logagent/agent"
)

func main() {
	//CPU 性能分析
	f, err := os.OpenFile("./cpu.prof", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
		return
	}
	pprof.StartCPUProfile(f)
	defer f.Close()

	app := agent.Init()
	app.Run()


}
