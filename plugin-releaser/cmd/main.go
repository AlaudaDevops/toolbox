package main

import (
	"fmt"
	"github.com/AlaudaDevops/toolbox/plugin-releaser/cmd/jira"
	"github.com/spf13/cobra"
	"os"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "plugin-releaser",
	Short: "A CLI tool for executing plugin release related tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	rootCmd.AddCommand(jira.NewJiraCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
