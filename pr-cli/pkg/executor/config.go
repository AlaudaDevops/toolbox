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

package executor

import (
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/sirupsen/logrus"
)

// PRHandlerInterface defines the interface for PR handler operations needed by the executor
type PRHandlerInterface interface {
	ExecuteCommand(command string, args []string) error
	PostComment(body string) error
	CheckPRStatus(expectedStatus string) error
	GetCommentsWithCache() ([]git.Comment, error)
}

// ExecutionConfig holds the configuration for command execution behavior
type ExecutionConfig struct {
	// ValidateCommentSender determines if the command sender should be validated
	ValidateCommentSender bool
	
	// ValidatePRStatus determines if PR status should be checked before execution
	ValidatePRStatus bool
	
	// DebugMode enables debug logging and skips some validations
	DebugMode bool
	
	// PostErrorsAsPRComments determines if errors should be posted as PR comments
	PostErrorsAsPRComments bool
	
	// ReturnErrors determines if errors should be returned to the caller
	ReturnErrors bool
	
	// StopOnFirstError determines if multi-command execution should stop on first error
	StopOnFirstError bool
}

// ExecutionContext holds the context for command execution
type ExecutionContext struct {
	// PRHandler is the handler for PR operations
	PRHandler PRHandlerInterface
	
	// Logger is the logger instance for logging
	Logger logrus.FieldLogger
	
	// Config is the execution configuration
	Config *ExecutionConfig
	
	// MetricsRecorder records execution metrics
	MetricsRecorder MetricsRecorder
	
	// Platform is the git platform (github, gitlab, gitea)
	Platform string
	
	// CommentSender is the user who triggered the command
	CommentSender string
	
	// TriggerComment is the original comment that triggered the command
	TriggerComment string
}

// NewCLIExecutionConfig creates a new execution configuration for CLI mode
func NewCLIExecutionConfig(debugMode bool) *ExecutionConfig {
	return &ExecutionConfig{
		ValidateCommentSender:  false, // CLI doesn't validate comment sender
		ValidatePRStatus:       true,  // CLI checks PR status
		DebugMode:              debugMode,
		PostErrorsAsPRComments: true,  // CLI posts errors as comments
		ReturnErrors:           true,  // CLI returns errors for proper exit codes
		StopOnFirstError:       false, // CLI continues executing all commands
	}
}

// NewWebhookExecutionConfig creates a new execution configuration for webhook mode
func NewWebhookExecutionConfig() *ExecutionConfig {
	return &ExecutionConfig{
		ValidateCommentSender:  true,  // Webhook validates comment sender
		ValidatePRStatus:       true,  // Webhook checks PR status
		DebugMode:              false, // Webhook runs in production mode
		PostErrorsAsPRComments: true,  // Webhook posts errors as comments
		ReturnErrors:           false, // Webhook doesn't return errors (already posted)
		StopOnFirstError:       false, // Webhook continues executing all commands
	}
}
