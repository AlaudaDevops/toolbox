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
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
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
	command, cmdArgs, err := p.parseCommand(p.Config.TriggerComment)
	if err != nil {
		return fmt.Errorf("failed to parse command: %w", err)
	}

	p.Infof("Processing command: %s with args: %v", command, cmdArgs)
	if p.Config.Verbose {
		p.Debugf("Processing PR %d, config: %s", p.Config.PRNum, p.Config.DebugString())
	}

	// Initialize PR handler
	prHandler, err := handler.NewPRHandler(p.Logger, p.Config)
	if err != nil {
		return fmt.Errorf("failed to initialize PR handler: %w", err)
	}

	// Handle built-in commands separately
	if handler.IsBuiltInCommand(command) {
		return p.executeBuiltInCommand(prHandler, command, cmdArgs)
	}

	// Handle regular commands
	return p.executeRegularCommand(prHandler, command, cmdArgs)
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
	p.Config.TriggerComment = strings.TrimSpace(p.Config.TriggerComment)
	p.Config.LGTMReviewEvent = strings.TrimSpace(p.Config.LGTMReviewEvent)
	p.Config.MergeMethod = strings.TrimSpace(p.Config.MergeMethod)
	p.Config.SelfCheckName = strings.TrimSpace(p.Config.SelfCheckName)
	p.Config.ResultsDir = strings.TrimSpace(p.Config.ResultsDir)

	// Handle special string fields that need to be read separately
	// since they're not directly mapped to Config struct fields
	if p.prNumStr == "" {
		p.prNumStr = strings.TrimSpace(viper.GetString("pr-num"))
	}
	if p.lgtmPermissionsStr == "" {
		p.lgtmPermissionsStr = strings.TrimSpace(viper.GetString("lgtm-permissions"))
	}
	if p.robotAccountsStr == "" {
		p.robotAccountsStr = strings.TrimSpace(viper.GetString("robot-accounts"))
	}
}

// parseStringFields converts string CLI fields to proper types in config
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
		permissions := strings.Split(p.lgtmPermissionsStr, ",")
		for i, perm := range permissions {
			permissions[i] = strings.TrimSpace(perm)
		}
		p.Config.LGTMPermissions = permissions
	} else if len(p.Config.LGTMPermissions) == 0 {
		// If no CLI flag and no environment variable, use default
		p.Config.LGTMPermissions = []string{"admin", "write"}
	}

	// Parse robot accounts
	if p.robotAccountsStr != "" {
		accounts := strings.Split(p.robotAccountsStr, ",")
		for i, account := range accounts {
			accounts[i] = strings.TrimSpace(account)
		}
		p.Config.RobotAccounts = accounts
	}

	return nil
}

var (
	// Match pattern: /command [args...] or /__built-in-command [args...]
	commentRegexp = regexp.MustCompile(`^/(help|rebase|lgtm|remove-lgtm|cherry-pick|cherrypick|assign|merge|ready|unassign|label|unlabel|check|retest|batch)\s*(.*)$`)
	// Match pattern for built-in commands: /__command [args...]
	builtInCommandRegexp = regexp.MustCompile(`^/(__[a-z-_]+)\s*(.*)$`)
)

// parseCommand parses the trigger comment to extract command and arguments
func (p *PROption) parseCommand(comment string) (string, []string, error) {
	comment = strings.TrimSpace(comment)
	if !strings.HasPrefix(comment, "/") {
		return "", nil, fmt.Errorf("comment must start with /")
	}

	// Try to match built-in commands first (/__command)
	if builtInMatches := builtInCommandRegexp.FindStringSubmatch(comment); len(builtInMatches) >= 2 {
		command := builtInMatches[1] // Built-in command with __ prefix already captured
		argsStr := strings.TrimSpace(builtInMatches[2])

		var args []string
		if argsStr != "" {
			args = strings.Fields(argsStr)
		}

		return command, args, nil
	}

	// Try to match regular commands (/command)
	matches := commentRegexp.FindStringSubmatch(comment)
	if len(matches) < 2 {
		return "", nil, fmt.Errorf("invalid command format")
	}

	command := matches[1]
	argsStr := strings.TrimSpace(matches[2])

	// Handle special case: /lgtm cancel
	if command == "lgtm" && argsStr == "cancel" {
		command = "remove-lgtm"
		argsStr = ""
	}

	var args []string
	if argsStr != "" {
		args = strings.Fields(argsStr)
	}

	return command, args, nil
}

// commandsSkipPRStatusCheck defines commands that can work with closed PRs
// and don't require the PR to be in "open" state
var commandsSkipPRStatusCheck = map[string]bool{
	"cherry-pick": true,
	"cherrypick":  true,
	// Add other commands here that should work with closed PRs
}

// shouldSkipPRStatusCheck returns true if the command can work with closed PRs
func (p *PROption) shouldSkipPRStatusCheck(command string) bool {
	return commandsSkipPRStatusCheck[command] || handler.IsBuiltInCommand(command)
}

// validateCommentSender verifies that the comment-sender actually posted the trigger-comment
func (p *PROption) validateCommentSender(prHandler *handler.PRHandler) error {
	// Get all comments from the PR
	comments, err := prHandler.GetCommentsWithCache()
	if err != nil {
		return fmt.Errorf("failed to get PR comments: %w", err)
	}

	// Check if any comment from the comment-sender contains the trigger-comment
	found := false
	for _, comment := range comments {
		if strings.EqualFold(comment.User.Login, p.Config.CommentSender) &&
			(comment.Body == p.Config.TriggerComment || strings.Contains(comment.Body, p.Config.TriggerComment)) {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("comment sender '%s' did not post a comment containing '%s'", p.Config.CommentSender, p.Config.TriggerComment)
	}

	p.Infof("Comment sender validation passed: %s posted a comment containing the trigger", p.Config.CommentSender)
	return nil
}

// initialize initializes and validates the PROption configuration
func (p *PROption) initialize() error {
	// Read all values from viper (which includes environment variables)
	p.readAllFromViper()

	// Parse string fields into config
	if err := p.parseStringFields(); err != nil {
		return fmt.Errorf("failed to parse CLI fields: %w", err)
	}

	// Set log level based on verbose flag
	if p.Config.Verbose {
		p.SetLevel(logrus.DebugLevel)
		p.Debug("Verbose logging enabled")
	} else {
		p.SetLevel(logrus.InfoLevel)
	}

	// Validate configuration
	if err := p.Config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	return nil
}

// executeBuiltInCommand handles execution of built-in commands
func (p *PROption) executeBuiltInCommand(prHandler *handler.PRHandler, command string, cmdArgs []string) error {
	p.Infof("Executing built-in command: %s", command)

	// Built-in commands skip comment sender validation
	// Execute built-in command using unified method
	err := prHandler.ExecuteCommand(command, cmdArgs)

	// Handle error for built-in commands (may not want to post comments for internal commands)
	if err != nil {
		p.Errorf("Built-in command %s failed: %v", command, err)
		// For built-in commands, we typically don't post error comments to PR
		// as they are internal system operations
		return fmt.Errorf("built-in command failed: %w", err)
	}

	return nil
}

// executeRegularCommand handles execution of regular user commands
func (p *PROption) executeRegularCommand(prHandler *handler.PRHandler, command string, cmdArgs []string) error {
	p.Infof("Executing regular command: %s", command)

	// Check if PR is open and get PR information to retrieve the author
	// Skip status check for commands that can work with closed PRs
	if !p.shouldSkipPRStatusCheck(command) {
		if err := prHandler.CheckPRStatus("open"); err != nil {
			return fmt.Errorf("PR status check failed: %w", err)
		}
	}

	// Validate comment sender in non-debug mode
	if !p.Config.Debug {
		if err := p.validateCommentSender(prHandler); err != nil {
			return fmt.Errorf("comment sender validation failed: %w", err)
		}
	}

	// Execute regular command using unified method
	err := prHandler.ExecuteCommand(command, cmdArgs)

	// If command execution failed, try to post error as comment
	if err != nil {
		p.Errorf("Command %s failed: %v", command, err)
		return p.handleCommandError(prHandler, command, err)
	}

	return nil
}

// handleCommandError handles command errors by posting them as PR comments when possible
func (p *PROption) handleCommandError(prHandler *handler.PRHandler, command string, err error) error {
	// Check if this is a CommentedError (comment already posted)
	var commentedErr *handler.CommentedError
	if errors.As(err, &commentedErr) {
		// Comment already posted, just log and return nil to avoid terminal error
		p.Infof("Error comment already posted for command: %s", command)
		return nil
	}

	errorMessage := fmt.Sprintf(messages.CommandErrorTemplate, command, err.Error())

	// Try to post error message as PR comment
	if commentErr := prHandler.PostComment(errorMessage); commentErr != nil {
		// If we can't post the comment, return the original error plus comment error
		p.Errorf("Failed to post error comment: %v", commentErr)
		return fmt.Errorf("command failed: %w (and failed to post error comment: %v)", err, commentErr)
	}

	// Successfully posted error as comment, return nil to avoid terminal error
	p.Infof("Posted command error as PR comment for command: %s", command)
	return nil
}
