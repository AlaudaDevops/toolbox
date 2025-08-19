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

// Config holds the configuration for PR CLI operations
type Config struct {
	// Platform configuration
	Platform string `json:"platform" yaml:"platform" mapstructure:"platform"`
	Token    string `json:"token" yaml:"token" mapstructure:"token"`
	BaseURL  string `json:"base_url,omitempty" yaml:"base_url,omitempty" mapstructure:"base_url"`

	// Repository configuration
	Owner string `json:"owner" yaml:"owner" mapstructure:"owner"`
	Repo  string `json:"repo" yaml:"repo" mapstructure:"repo"`

	// PR configuration
	PRNum         int    `json:"pr_num" yaml:"pr_num" mapstructure:"pr_num"`
	CommentSender string `json:"comment_sender" yaml:"comment_sender" mapstructure:"comment_sender"`

	// CLI-specific configuration
	TriggerComment string `json:"trigger_comment,omitempty" yaml:"trigger_comment,omitempty" mapstructure:"trigger_comment"`
	Debug          bool   `json:"debug,omitempty" yaml:"debug,omitempty" mapstructure:"debug"`

	// LGTM configuration
	LGTMThreshold   int      `json:"lgtm_threshold" yaml:"lgtm_threshold" mapstructure:"lgtm_threshold"`
	LGTMPermissions []string `json:"lgtm_permissions" yaml:"lgtm_permissions" mapstructure:"lgtm_permissions"`
	LGTMReviewEvent string   `json:"lgtm_review_event,omitempty" yaml:"lgtm_review_event,omitempty" mapstructure:"lgtm_review_event"`

	// Merge configuration
	MergeMethod string `json:"merge_method" yaml:"merge_method" mapstructure:"merge_method"`

	// Check configuration
	SelfCheckName string `json:"self_check_name,omitempty" yaml:"self_check_name,omitempty" mapstructure:"self_check_name"`

	// Cherry-pick configuration
	UseGitCLIForCherryPick bool `json:"use_git_cli_for_cherrypick,omitempty" yaml:"use_git_cli_for_cherrypick,omitempty" mapstructure:"use_git_cli_for_cherrypick"`

	// Logging configuration
	LogLevel string `json:"log_level" yaml:"log_level" mapstructure:"log_level"`
}

// NewDefaultConfig returns a new Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		Platform:               "github",
		LGTMThreshold:          1,
		LGTMPermissions:        []string{"admin", "write"},
		LGTMReviewEvent:        "APPROVE",
		MergeMethod:            "rebase",
		SelfCheckName:          "pr-cli",
		LogLevel:               "info",
		UseGitCLIForCherryPick: true, // Default to Git CLI method for backward compatibility
	}
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
