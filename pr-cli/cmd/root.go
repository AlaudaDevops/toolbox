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

// Package cmd provides command line interface for the PR CLI application
package cmd

import (
	"os"

	"github.com/AlaudaDevops/toolbox/pr-cli/internal/version"
	"github.com/spf13/cobra"

	// Import platform implementations to register them
	_ "github.com/AlaudaDevops/toolbox/pr-cli/pkg/platforms/github"
	_ "github.com/AlaudaDevops/toolbox/pr-cli/pkg/platforms/gitlab"
)

// prOption is the global instance of PROption
var prOption *PROption

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pr-cli",
	Short: "GitHub Pull Request command line tool",
	Long: `PR CLI is a tool for managing GitHub Pull Requests through commands.
It supports operations like assigning reviewers, LGTM voting, merging, and more.

The tool processes GitHub comment commands and executes corresponding actions
on pull requests using the GitHub API.

Example usage:
  # Process a comment command (PR author automatically detected)
  pr-cli --platform github --repo-owner owner --repo-name repo --pr-num 123 --comment-sender user --token $TOKEN --trigger-comment "/assign user1 user2"

  # Use specific LGTM threshold
  pr-cli --lgtm-threshold 2 --trigger-comment "/lgtm" --platform github --repo-owner owner --repo-name repo --pr-num 123 --comment-sender user --token $TOKEN

  # Custom merge method
  pr-cli --merge-method squash --trigger-comment "/merge" --platform github --repo-owner owner --repo-name repo --pr-num 123 --comment-sender user --token $TOKEN

  # Use ready command (alias for merge)
  pr-cli --trigger-comment "/ready" --platform github --repo-owner owner --repo-name repo --pr-num 123 --comment-sender user --token $TOKEN

Supported Comment Commands:
  Regular commands:
  - /help, /assign, /unassign, /lgtm, /remove-lgtm, /merge, /ready, /rebase
  - /check, /batch, /cherry-pick, /cherrypick, /label, /unlabel, /retest

  Built-in commands (system internal):
  - __post-merge-cherry-pick

Tekton Results:
  When used in Tekton pipelines, pr-cli writes result files to the configured results directory.
  Available results:
  - merge-successful: Written with value "true" when a merge operation completes successfully
  - has-cherry-pick-comments: Written with value "true"/"false" during /merge or /ready operations to indicate if cherry-pick comments exist in the PR`,

	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle --version flag
		if showVersion, _ := cmd.Flags().GetBool("version"); showVersion {
			versionInfo := version.Get()

			// Use existing version output methods
			if outputFormat == "json" {
				return printVersionJSON(versionInfo)
			}
			return printVersionText(versionInfo)
		}
		return prOption.Run(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
