package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "3.0.0"

// VersionCmd represents the version command
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show current version of Bifrost",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func init() {
	RootCmd.AddCommand(VersionCmd)
}
