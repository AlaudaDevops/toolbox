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

package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/executor"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// PROption option for PR CLI command
type PROption struct {
	*logrus.Logger
	Config *config.Config

	// String fields for CLI parsing (will be converted to Config)
	prNumStr           string
	lgtmPermissionsStr string
	robotAccountsStr   string
}

// NewPROption creates a new PROption instance
func NewPROption() *PROption {
	return &PROption{
		Logger: logrus.New(),
		Config: config.NewDefaultConfig(),
	}
}

// AddFlags add flags to options
func (p *PROption) AddFlags(flags *pflag.FlagSet) {
	// Platform and authentication configuration
	flags.StringVar(&p.Config.Platform, "platform", p.Config.Platform, "Git platform (github or gitlab)")
	flags.StringVar(&p.Config.Token, "token", "", "Git platform API token for authentication")
	flags.StringVar(&p.Config.CommentToken, "comment-token", "", "Git platform API token for posting comments (optional, falls back to --token)")
	flags.StringVar(&p.Config.BaseURL, "base-url", "", "API base URL (optional, defaults per platform)")
	flags.StringVar(&p.Config.Owner, "repo-owner", "", "Repository owner (organization or user)")
	flags.StringVar(&p.Config.Repo, "repo-name", "", "Repository name")

	// PR information
	flags.StringVar(&p.prNumStr, "pr-num", "", "Pull request number")
	flags.StringVar(&p.Config.CommentSender, "comment-sender", "", "Comment author's username")
	flags.StringVar(&p.Config.TriggerComment, "trigger-comment", "", "The comment that triggered this command")

	// LGTM configuration
	flags.IntVar(&p.Config.LGTMThreshold, "lgtm-threshold", p.Config.LGTMThreshold, "Minimum number of LGTM approvals required")
	flags.StringVar(&p.lgtmPermissionsStr, "lgtm-permissions", "", "Required permissions for LGTM (comma-separated)")
	flags.StringVar(&p.Config.LGTMReviewEvent, "lgtm-review-event", p.Config.LGTMReviewEvent, "Review event type for LGTM")
	flags.StringVar(&p.robotAccountsStr, "robot-accounts", "", "Robot/bot account usernames (comma-separated) for managing bot approval reviews")

	// Merge configuration
	flags.StringVar(&p.Config.MergeMethod, "merge-method", p.Config.MergeMethod, "Merge method (default: auto, options: auto, merge, squash, rebase)")

	// Check configuration
	flags.StringVar(&p.Config.SelfCheckName, "self-check-name", p.Config.SelfCheckName, "Name of the tool's own check run to exclude from status checks")

	// Cherry-pick configuration
	flags.BoolVar(&p.Config.UseGitCLIForCherryPick, "use-git-cli-for-cherrypick", p.Config.UseGitCLIForCherryPick, "Use Git CLI for cherry-pick operations (default: true, more reliable than API method)")

	// Debug and logging flags
	flags.BoolVar(&p.Config.Verbose, "verbose", false, "Enable verbose logging (debug level logs)")
	flags.BoolVar(&p.Config.Debug, "debug", false, "Enable debug mode (skip comment sender validation and allow PR creators to approve their own PR)")
	flags.StringVar(&p.Config.ResultsDir, "results-dir", p.Config.ResultsDir, "Directory to write results files (default: /tekton/results)")
}

// Run executes the PR CLI logic
func (p *PROption) Run(cmd *cobra.Command, args []string) error {
	// Initialize and validate configuration
	if err := p.initialize(); err != nil {
		return err
	}

	// Parse the trigger comment to determine the command
	parsedCmd, err := p.parseCommand(p.Config.TriggerComment)
	if err != nil {
		return fmt.Errorf("failed to parse command %q: %w", p.Config.TriggerComment, err)
	}

	p.Infof("Processing command type: %s", parsedCmd.Type)
	if p.Config.Verbose {
		p.Debugf("Processing PR %d, config: %s", p.Config.PRNum, p.Config.DebugString())
	}

	// Initialize PR handler
	prHandler, err := handler.NewPRHandler(p.Logger, p.Config)
	if err != nil {
		return fmt.Errorf("failed to initialize PR handler: %w", err)
	}

	// Use unified executor
	return p.executeWithUnifiedExecutor(prHandler, parsedCmd)
}

// readAllFromViper reads all configuration values from viper
// This includes environment variables with PR_ prefix
func (p *PROption) readAllFromViper() {
	// Use viper.Unmarshal to automatically map all values to the config struct
	if err := viper.Unmarshal(p.Config); err != nil {
		// Log warning but continue - this shouldn't prevent the application from running
		p.Warnf("Failed to unmarshal config from viper: %v", err)
	}

	// Clean up string values by trimming whitespace and newlines
	p.Config.Platform = strings.TrimSpace(p.Config.Platform)
	p.Config.Token = strings.TrimSpace(p.Config.Token)
	p.Config.CommentToken = strings.TrimSpace(p.Config.CommentToken)
	p.Config.BaseURL = strings.TrimSpace(p.Config.BaseURL)
	p.Config.Owner = strings.TrimSpace(p.Config.Owner)
	p.Config.Repo = strings.TrimSpace(p.Config.Repo)
	p.Config.CommentSender = strings.TrimSpace(p.Config.CommentSender)
	p.Config.TriggerComment = comment.Normalize(p.Config.TriggerComment)
	p.Config.LGTMReviewEvent = strings.TrimSpace(p.Config.LGTMReviewEvent)
	p.Config.MergeMethod = strings.TrimSpace(p.Config.MergeMethod)
	p.Config.SelfCheckName = strings.TrimSpace(p.Config.SelfCheckName)
	p.Config.ResultsDir = strings.TrimSpace(p.Config.ResultsDir)

	// Handle special string fields that need to be read separately
	p.prNumStr = strings.TrimSpace(viper.GetString("pr-num"))
	p.lgtmPermissionsStr = strings.TrimSpace(viper.GetString("lgtm-permissions"))
	p.robotAccountsStr = strings.TrimSpace(viper.GetString("robot-accounts"))
}

// parseStringFields parses string fields and sets them in Config
func (p *PROption) parseStringFields() error {
	// Parse PR number
	if p.prNumStr != "" {
		prNum, err := strconv.Atoi(p.prNumStr)
		if err != nil {
			return fmt.Errorf("invalid PR number '%s': %w", p.prNumStr, err)
		}
		p.Config.PRNum = prNum
	}

	// Parse LGTM permissions
	if p.lgtmPermissionsStr != "" {
		var permissions []string
		for perm := range strings.SplitSeq(p.lgtmPermissionsStr, ",") {
			perm = strings.TrimSpace(perm)
			if perm != "" {
				permissions = append(permissions, perm)
			}
		}
		if len(permissions) > 0 {
			p.Config.LGTMPermissions = permissions
		}
	}

	// Parse robot accounts
	if p.robotAccountsStr != "" {
		var accounts []string
		for account := range strings.SplitSeq(p.robotAccountsStr, ",") {
			account = strings.TrimSpace(account)
			if account != "" {
				accounts = append(accounts, account)
			}
		}
		if len(accounts) > 0 {
			p.Config.RobotAccounts = accounts
		}
	}

	return nil
}

// executeWithUnifiedExecutor executes commands using the unified executor layer
func (p *PROption) executeWithUnifiedExecutor(prHandler *handler.PRHandler, parsedCmd *ParsedCommand) error {
	// Create execution config
	execConfig := executor.NewCLIExecutionConfig(p.Config.Debug)

	// Create execution context
	execContext := &executor.ExecutionContext{
		PRHandler:       prHandler,
		Logger:          p.Logger,
		Config:          execConfig,
		MetricsRecorder: &executor.NoOpMetricsRecorder{},
		Platform:        p.Config.Platform,
		CommentSender:   p.Config.CommentSender,
		TriggerComment:  p.Config.TriggerComment,
	}

	// Create executor
	cmdExecutor := executor.NewCommandExecutor(execContext)

	// Execute command
	result, err := cmdExecutor.Execute(parsedCmd)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("command execution failed")
	}

	return nil
}
