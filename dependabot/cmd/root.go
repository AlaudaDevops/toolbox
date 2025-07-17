/*
Copyright 2025 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package cmd provides command line interface for the DependaBot application
package cmd

import (
	"fmt"
	"os"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/git"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/pipeline"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dependabot",
	Short: "Automated dependency vulnerability fixing tool",
	Long: `DependaBot is a tool that automatically updates vulnerable dependencies
in your projects based on Trivy security scan results.

It supports multiple programming languages and can automatically create
pull requests with the security fixes.

Example usage:
  # Local project mode (automatic scan)
  dependabot --dir /path/to/project
  dependabot --dir /path/to/project
  dependabot --dir /path/to/project --create-pr=false

  # Remote repository mode (clone + automatic scan)
  dependabot --repo.url https://github.com/user/repo.git
  dependabot --repo.url git@github.com:user/repo.git --repo.branch main
  dependabot --repo.url https://github.com/user/repo.git --pr.autoCreate=false

  # Using external configuration file
  dependabot --repo.url https://github.com/user/repo.git --config my-config.yml`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return runDependaBot()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// runDependaBot runs the main DependaBot pipeline
func runDependaBot() error {
	if viper.GetBool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Debug logging enabled")
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	var cfg config.DependaBotConfig
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to unmarshal repository configuration: %w", err)
	}
	// Get values directly from cobra flags
	projectDirPath := viper.GetString("dir")

	// Validate input parameters
	if cfg.Repo.URL != "" && projectDirPath != "." {
		return fmt.Errorf("cannot specify both --repo and --dir. Use --repo for remote repositories or --dir for local projects")
	}

	if cfg.Repo.URL == "" && projectDirPath == "" {
		projectDirPath = "."
	}

	var workingProjectPath string
	var gitCloner *git.GitCloner

	if cfg.Repo.URL != "" {
		logrus.Infof("Cloning remote repository: %s", cfg.Repo.URL)

		// Check if git is installed
		if err := git.CheckGitInstalled(); err != nil {
			return fmt.Errorf("git CLI check failed: %w", err)
		}

		// Clone repository with specified branch and submodule configuration
		gitCloner = git.NewGitClonerFromConfig(&cfg.Repo)

		clonedPath, err := gitCloner.CloneRepository()
		if err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		workingProjectPath = clonedPath

		// Ensure cleanup on exit
		defer func() {
			if cleanupErr := gitCloner.Cleanup(); cleanupErr != nil {
				logrus.Warnf("Warning: failed to cleanup cloned repository: %v", cleanupErr)
			}
		}()
	} else {
		var err error
		g := git.NewGitOperator(projectDirPath)
		cfg.Repo.URL, err = g.GetRepoURL()
		if err != nil {
			return fmt.Errorf("failed to get repo info: %w", err)
		}
		cfg.Repo.Branch, err = g.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get branch info: %w", err)
		}
		workingProjectPath = projectDirPath
	}

	logrus.Info("Reading and merging configurations...")

	configReader := config.NewConfigReader()

	repoConfig, err := configReader.ReadRepoConfig(workingProjectPath)
	if err != nil {
		logrus.Warnf("Warning: failed to read repository config: %v", err)
		repoConfig = &config.DependaBotConfig{}
	}

	// CLI config has highest priority
	finalConfig := configReader.MergeConfigs(repoConfig, &cfg)
	finalConfig = configReader.ApplyDefaults(finalConfig)
	logrus.Debug("Final config:", finalConfig.String())

	// Step 2: Convert to pipeline configuration
	pipelineConfig := &pipeline.Config{
		ProjectPath:      workingProjectPath,
		DependaBotConfig: *finalConfig,
	}

	// Step 3: Create and run pipeline
	logrus.Info("Starting dependency update pipeline...")
	p := pipeline.NewPipeline(pipelineConfig)
	return p.Run()
}
