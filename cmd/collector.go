package cmd

import (
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
	"github.com/y7ut/logagent/pkg/collection"
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

	dataCollection := collection.New(collectors).Map(func(k int, item agent.Collector) agent.Collector {
		if item.Style == "Date" {
			item.Path = time.Now().Format(item.Path)
		}
		return item
	}).Map(func(k int, item agent.Collector) agent.Collector {
		_, err := os.Stat(item.Path)
		if err != nil && os.IsNotExist(err) {
			item.Exist = "❌"
			return item
		}
		item.Exist = "✅"
		return item
	})

	if cmd.Flag("filter").Value.String() != "" {
		dataCollection.Filter(func(item agent.Collector) bool {
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
