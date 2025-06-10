package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pr-watcher",
	Short: "A CLI tool for repository and organization maintenance",
	Long: `pr-watcher is a CLI tool designed to help with repository and organization maintenance tasks.
It supports both GitHub and GitLab platforms and performs operations such as:

GitHub:
- Listing repositories in an organization
- Checking open pull requests
- Finding stale PRs that have been open for extended periods

GitLab:
- Listing projects in a group
- Checking open merge requests
- Finding stale MRs that have been open for extended periods

Both platforms:
- Generating comprehensive reports in JSON format
- Filtering by age, state, and draft status
- Exporting results to files for automation`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pr-watcher.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
