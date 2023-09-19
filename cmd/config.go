package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

var InitConfigCommand = &cobra.Command{
	Use:   "config",
	Short: "A tool to generate configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		generateConfig(cmd, args)
	},
}

var createHelper bool

func generateConfig(cmd *cobra.Command, args []string) {
	cfg := ParseConfig(createHelper)

	confFile := cmd.Flag("file").Value.String()

	if err := cfg.SaveTo(confFile); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Configuration file saved")
}

func ParseConfig(helper bool) *ini.File {
	cfg := ini.Empty()

	appName := fmt.Sprintf("bifrost_%d", rand.Intn(1000))
	kafkaConn := ""
	etcdConn := ""
	queueSize := "1000"

	if helper {
		fmt.Println("let's generate a config file for you: ")

		appName = promptInput("1. Enter the name of the application: ")
		if strings.TrimSpace(appName) != "" {
			appName = strings.TrimSpace(appName)
		}

		kafkaConn = promptInput("2. Enter the kafka connection string:")
		kafkaConn = strings.TrimSpace(kafkaConn)

		queueSize = promptInput("3. Enter the kafka queue size:")
		queueSize = strings.TrimSpace(queueSize)

		etcdConn = promptInput("4. Enter the etcd connection string: ")
		etcdConn = strings.TrimSpace(etcdConn)
	}

	cfg.Section("app").Comment = "Application name"
	cfg.Section("app").NewKey("logagent_id", appName)

	cfg.Section("kafka").Comment = "Kafka connection string"
	cfg.Section("kafka").NewKey("address", kafkaConn)
	cfg.Section("kafka").NewKey("queue_size", queueSize)

	cfg.Section("etcd").Comment = "Etcd connection string"
	cfg.Section("etcd").NewKey("address", etcdConn)

	cfg.Section("runtime").Comment = "runtime config"
	cfg.Section("runtime").NewKey("path", "./runtime")

	cfg.Section("log").Comment = "log config"
	cfg.Section("log").NewKey("path", "./runtime/log")
	cfg.Section("log").NewKey("name", "bifrost.log")
	return cfg
}

func init() {
	InitConfigCommand.Flags().StringP("file", "f", "bifrost.conf", "output file name")
	InitConfigCommand.Flags().BoolVarP(&createHelper, "step", "s", false, "create with helper ")
	RootCmd.AddCommand(InitConfigCommand)
}

// promptInput 从用户的stdin中获取输入，并在每次输入后刷新缓冲区
func promptInput(prompt string) string {
	fmt.Print(prompt)
	var input string
	fmt.Scanln(&input)
	return input
}
