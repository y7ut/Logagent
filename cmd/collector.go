package cmd

import (
	"container/list"
	"encoding/json"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/y7ut/logagent/agent"
	"github.com/y7ut/logagent/component/table"
	"github.com/y7ut/logagent/conf"
	"github.com/y7ut/logagent/etcd"
	"gopkg.in/ini.v1"
)

// VersionCmd represents the version command
var ListCollectorCmd = &cobra.Command{
	Use:   "list",
	Short: "Show collectors list of Bifrost",
	Run: func(cmd *cobra.Command, args []string) {
		listCollectors(cmd, args)
	},
}

func listCollectors(cmd *cobra.Command, args []string) {
	configPath := cmd.Flag("config").Value.String()
	checkconfig(configPath)

	if err := ini.MapTo(conf.APPConfig, configPath); err != nil {
		fmt.Printf("load ini file error: %s ", err)
		return
	}
	etcd.Init()

	collectorData, err := etcd.GetLogConfToEtcd()
	if err != nil {
		fmt.Printf("get etcd conf error: %s ", err)
		return
	}

	collectors := make([]agent.Collector, 0)

	err = json.Unmarshal(collectorData, &collectors)

	if err != nil {
		fmt.Printf("unmarshal error: %s ", err)
		return
	}

	dataCollection := collection(collectors).Each(func(item *agent.Collector) {
		if item.Style == "Date" {
			item.Path = time.Now().Format(item.Path)
		}
	}).Each(func(item *agent.Collector) {
		_, err := os.Stat(item.Path)
		if err != nil && os.IsNotExist(err) {
			item.Exist = "❌"
			return
		}
		item.Exist = "✅"
	})

	if cmd.Flag("filter").Value.String() != "" {
		dataCollection.Filter(func(item *agent.Collector) bool {
			return item.Style == cmd.Flag("filter").Value.String()
		})
	}

	oh := table.NewGrid(dataCollection.Value()).DefineHeader(map[string]string{
		"Exist": "状态",
	})

	tableDemo := table.Create(oh)
	if _, err := tea.NewProgram(tableDemo).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

}

func init() {
	ListCollectorCmd.Flags().StringP("config", "c", "./bifrost.conf", "config bifrost file")
	ListCollectorCmd.Flags().StringP("filter", "f", "", "filter collector")
	RootCmd.AddCommand(ListCollectorCmd)
}

type item interface {
	any
}

// jian
type Collection[T item] struct {
	data *list.List
}

func collection[T item](d []T) *Collection[T] {

	l := list.New()
	for _, item := range d {
		t := item
		l.PushBack(&t)
	}

	return &Collection[T]{data: l}
}

func (c *Collection[T]) Each(lambda func(item *T)) *Collection[T] {
	for current := c.data.Front(); current != nil; current = current.Next() {

		lambda(current.Value.(*T))
	}
	return c
}

func (c *Collection[T]) Filter(lambda func(item *T) bool) *Collection[T] {
	crash := list.New()
	for current := c.data.Back(); current != nil; current = current.Prev() {
		if !lambda(current.Value.(*T)) {
			continue
		}
		crash.PushBack(current.Value.(*T))
	}
	c.data = crash
	return c

}

func (c *Collection[T]) Len() int {
	return c.data.Len()
}

func (c *Collection[T]) Value() []*T {
	l := make([]*T, 0)
	for current := c.data.Front(); current != nil; current = current.Next() {
		l = append(l, current.Value.(*T))
	}
	return l
}
