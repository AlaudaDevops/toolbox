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

// Package config provides configuration management for the PR CLI application
package config

import (
	"encoding/json"
	"fmt"
	"slices"
)

// Config holds the configuration for PR CLI operations
type Config struct {
	// Platform configuration
	Platform string `json:"platform" yaml:"platform" mapstructure:"platform"`
	Token    string `json:"token" yaml:"token" mapstructure:"token"`
	BaseURL  string `json:"base_url,omitempty" yaml:"base_url,omitempty" mapstructure:"base-url"`

	// Repository configuration
	Owner string `json:"owner" yaml:"owner" mapstructure:"repo-owner"`
	Repo  string `json:"repo" yaml:"repo" mapstructure:"repo-name"`

	// PR configuration
	PRNum         int    `json:"pr_num" yaml:"pr_num" mapstructure:"pr-num"`
	CommentSender string `json:"comment_sender" yaml:"comment_sender" mapstructure:"comment-sender"`

	// CLI-specific configuration
	TriggerComment string `json:"trigger_comment,omitempty" yaml:"trigger_comment,omitempty" mapstructure:"trigger-comment"`
	Debug          bool   `json:"debug,omitempty" yaml:"debug,omitempty" mapstructure:"debug"`
	Verbose        bool   `json:"verbose,omitempty" yaml:"verbose,omitempty" mapstructure:"verbose"`
	ResultsDir     string `json:"results_dir,omitempty" yaml:"results_dir,omitempty" mapstructure:"results-dir"`

	// LGTM configuration
	LGTMThreshold   int      `json:"lgtm_threshold" yaml:"lgtm_threshold" mapstructure:"lgtm-threshold"`
	LGTMPermissions []string `json:"lgtm_permissions" yaml:"lgtm_permissions" mapstructure:"lgtm-permissions"`
	LGTMReviewEvent string   `json:"lgtm_review_event,omitempty" yaml:"lgtm_review_event,omitempty" mapstructure:"lgtm-review-event"`
	RobotAccounts   []string `json:"robot_accounts,omitempty" yaml:"robot_accounts,omitempty" mapstructure:"robot-accounts"`

	// Merge configuration
	MergeMethod string `json:"merge_method" yaml:"merge_method" mapstructure:"merge-method"`

	// Check configuration
	SelfCheckName string `json:"self_check_name,omitempty" yaml:"self_check_name,omitempty" mapstructure:"self-check-name"`

	// Cherry-pick configuration
	UseGitCLIForCherryPick bool `json:"use_git_cli_for_cherrypick,omitempty" yaml:"use_git_cli_for_cherrypick,omitempty" mapstructure:"use-git-cli-for-cherrypick"`

	// Logging configuration
	LogLevel string `json:"log_level" yaml:"log_level" mapstructure:"log-level"`
}

// NewDefaultConfig returns a new Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		Platform:               "github",
		LGTMThreshold:          1,
		LGTMPermissions:        []string{"admin", "write"},
		LGTMReviewEvent:        "APPROVE",
		MergeMethod:            "squash",
		SelfCheckName:          "pr-cli",
		LogLevel:               "info",
		UseGitCLIForCherryPick: true, // Default to Git CLI method for backward compatibility
		ResultsDir:             "/tekton/results",
	}
}

// DebugString returns a JSON representation of the config with sensitive information redacted
func (c *Config) DebugString() string {
	debugConfig := *c
	debugConfig.Token = "[REDACTED]"

	data, err := json.MarshalIndent(debugConfig, "", "  ")
	if err != nil {
		return fmt.Sprintf("failed to marshal config: %v", err)
	}
	return string(data)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Platform == "" {
		return ErrMissingPlatform
	}
	if c.Token == "" {
		return ErrMissingToken
	}
	if c.Owner == "" {
		return ErrMissingOwner
	}
	if c.Repo == "" {
		return ErrMissingRepo
	}
	if c.PRNum <= 0 {
		return ErrInvalidPRNum
	}
	if c.CommentSender == "" {
		return ErrMissingCommentSender
	}
	if c.TriggerComment == "" {
		return ErrMissingTriggerComment
	}
	return nil
}

// IsRobotUser checks if the user is a robot user that should be excluded from LGTM status
func (c *Config) IsRobotUser(user string) bool {
	return slices.Contains(c.RobotAccounts, user)
}

// HasLGTMPermission checks if the given permission is in the LGTM permissions list
func (c *Config) HasLGTMPermission(userPerm string) bool {
	return slices.Contains(c.LGTMPermissions, userPerm)
}
