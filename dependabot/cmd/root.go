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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/git"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/pipeline"
)

var (
	// cfgFile is the external configuration file path
	cfgFile string
	// projectDir is the path to the project directory containing go.mod
	projectDir string
	// repo is the repository URL to clone and analyze
	repo string
	// branch is the branch to clone and create PR against (default: main)
	branch string
	// createPR enables automatic PR creation
	createPR bool
	// debug enables debug log output
	debug bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dependabot",
	Short: "Automated dependency vulnerability fixing tool",
	Long: `DependaBot is a tool that automatically updates vulnerable dependencies
in your projects based on Trivy security scan results.

It supports multiple programming languages and can automatically create
pull requests with the security fixes.

The tool can either use existing Trivy scan results or automatically run
Trivy scanning if no results file is provided.

Configuration can be provided via:
1. Repository configuration: .github/dependabot.yml or .github/dependabot.yaml
2. External configuration file (--config flag)
3. Command line flags (highest priority)

Example usage:
  # Local project mode (automatic scan)
  dependabot --dir /path/to/project
  dependabot --dir /path/to/project
  dependabot --dir /path/to/project --create-pr=false

  # Remote repository mode (clone + automatic scan)
  dependabot --repo https://github.com/user/repo.git
  dependabot --repo git@github.com:user/repo.git --branch main
  dependabot --repo https://github.com/user/repo.git --create-pr=false

  # Using external configuration file
  dependabot --repo https://github.com/user/repo.git --config my-config.yml`,
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

func init() {
	// Persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")

	// Optional flags
	rootCmd.Flags().StringVar(&projectDir, "dir", ".", "path to project directory containing go.mod (default: current directory)")
	rootCmd.Flags().StringVar(&repo, "repo", "", "repository URL to clone and analyze (alternative to dir)")
	rootCmd.Flags().StringVar(&branch, "branch", "main", "branch to clone and create PR against")
	rootCmd.Flags().BoolVar(&createPR, "create-pr", false, "enable automatic PR creation")
	rootCmd.Flags().BoolVar(&debug, "debug", false, "enable debug log output")

	cobra.OnInitialize(func() {
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
			logrus.Debug("Debug logging enabled")
		} else {
			logrus.SetLevel(logrus.InfoLevel)
		}
	})
}

// runDependaBot runs the main DependaBot pipeline
func runDependaBot() error {
	// Get values directly from cobra flags
	projectDirPath := projectDir
	repositoryURL := repo
	targetBranchName := branch
	enablePRCreation := createPR

	// Show config file being used if specified
	if cfgFile != "" {
		logrus.Infof("Using config file: %s", cfgFile)
	}

	// Validate input parameters
	if repositoryURL != "" && projectDirPath != "." {
		return fmt.Errorf("cannot specify both --repo and --dir. Use --repo for remote repositories or --dir for local projects")
	}

	if repositoryURL == "" && projectDirPath == "" {
		projectDirPath = "."
	}

	// Step 0: Handle repository cloning if needed
	var workingProjectPath string
	var gitCloner *git.GitCloner

	if repositoryURL != "" {
		logrus.Infof("Cloning remote repository: %s", repositoryURL)

		// Check if git is installed
		if err := git.CheckGitInstalled(); err != nil {
			return fmt.Errorf("git CLI check failed: %w", err)
		}

		// Clone repository with specified branch
		gitCloner = git.NewGitCloner(repositoryURL, targetBranchName)

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
		repositoryURL, err = g.GetRepoURL()
		if err != nil {
			return fmt.Errorf("failed to get repo info: %w", err)
		}
		targetBranchName, err = g.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get branch info: %w", err)
		}
		workingProjectPath = projectDirPath
	}

	// Step 1: Read and merge all configurations
	logrus.Info("Reading and merging configurations...")

	configReader := config.NewConfigReader()

	// 1.1: Read repository configuration
	repoConfig, err := configReader.ReadRepoConfig(workingProjectPath)
	if err != nil {
		logrus.Warnf("Warning: failed to read repository config: %v", err)
		repoConfig = &config.DependaBotConfig{}
	}

	// 1.2: Read external configuration if provided
	externalConfig, err := configReader.ReadExternalConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to read external config: %w", err)
	}

	// 1.3: Create CLI configuration from command line flags
	cliConfig := &config.DependaBotConfig{
		Repo: config.Repo{
			URL:    repositoryURL,
			Branch: targetBranchName,
		},
		PRConfig: config.PRConfig{
			AutoCreate: &enablePRCreation,
		},
	}

	// 1.4: Merge configurations (repo config -> external config -> CLI config)
	// CLI config has highest priority
	finalConfig := configReader.MergeConfigs(repoConfig, externalConfig, cliConfig)
	finalConfig = configReader.ApplyDefaults(finalConfig)

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
