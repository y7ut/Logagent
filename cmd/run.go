package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/y7ut/logagent/agent"
	"github.com/y7ut/logagent/conf"
	"gopkg.in/ini.v1"
)

var RunCommand = &cobra.Command{
	Use:   "run",
	Short: "Run bifrost to guard logs and collect data",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	},
}

func checkconfig(path string) {
	_, err := os.Stat(path)

	if err != nil && os.IsNotExist(err) {
		fmt.Println("🧸 config not found")
		cfg := ParseConfig(true)

		if err := cfg.SaveTo(path); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Configuration file saved")
		return
	}
}

func run(cmd *cobra.Command, args []string) {
	configPath := cmd.Flag("config").Value.String()
	checkconfig(configPath)

	if err := ini.MapTo(conf.APPConfig, configPath); err != nil {
		log.Fatalf("load ini file error: %s ", err)
		return
	}
	fmt.Printf("use config file[%s] start... \n", configPath)

	agent.Init(conf.APPConfig.Runtime.Path, conf.APPConfig.Log.Path, conf.APPConfig.Log.Name)
	agent.Start()
}

func init() {
	RunCommand.Flags().StringP("config", "c", "./bifrost.conf", "config bifrost file")
	RootCmd.AddCommand(RunCommand)
}
