package cmd

import (
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const EnvPrefix = "DEPENDABOT"

var (
	// cfgFile is the external configuration file path
	cfgFile string
)

func init() {
	cobra.OnInitialize(initConfig)
	// Persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")

	// Optional flags
	rootCmd.Flags().String("dir", ".", "path to project directory containing go.mod (default: current directory)")
	rootCmd.Flags().String("repo.url", "", "repository URL to clone and analyze (alternative to dir)")
	rootCmd.Flags().String("repo.branch", "main", "branch to clone and create PR against")
	rootCmd.Flags().String("git.provider", "github", "Git provider type (e.g., github, gitlab)")
	rootCmd.Flags().String("git.token", "", "Access token for the Git provider (used for authentication and PR creation)")
	rootCmd.Flags().String("git.baseUrl", "https://api.github.com", "Base API URL of the Git provider (e.g., https://api.github.com for GitHub, https://gitlab.example.com for GitLab)")
	rootCmd.Flags().Bool("pr.autoCreate", false, "enable automatic PR creation")
	rootCmd.Flags().StringSlice("pr.labels", []string{}, "labels to add to the PR")
	rootCmd.Flags().StringSlice("pr.assignees", []string{}, "assignees to add to the PR")
	rootCmd.Flags().Bool("debug", false, "enable debug log output")
	viper.BindPFlags(rootCmd.Flags())
}

func initConfig() {
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".dependabot.yaml")
	}
	if err := viper.ReadInConfig(); err != nil {
		logrus.Warn("Can't read config:", err)
	}
}
