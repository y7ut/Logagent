package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Debug bool
)

var RootCmd = &cobra.Command{
	Use:   "./bifrost ",
	Short: "Use Bifrost to listen log and collect data",
	Long: `
	
     ____  _ ____                __
    / __ )(_) __/________  _____/ /_
   / __  / / /_/ ___/ __ \/ ___/ __/
  / /_/ / / __/ /  / /_/ (__  ) /_
 /_____/_/_/ /_/   \____/____/\__/
`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
