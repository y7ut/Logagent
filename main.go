package main

import (
	"github.com/y7ut/logagent/agent"
)

func main() {
	app := agent.Init()
	app.Run()
}
