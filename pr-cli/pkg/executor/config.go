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
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
	"github.com/sirupsen/logrus"
)

// ExecutionConfig controls command execution behavior
type ExecutionConfig struct {
	// Validation settings
	ValidateCommentSender bool // Validate that comment sender posted the command
	ValidatePRStatus      bool // Validate PR is in correct state
	DebugMode             bool // Skip certain validations in debug mode

	// Error handling
	PostErrorsAsPRComments bool // Post errors as PR comments
	ReturnErrors           bool // Return errors to caller

	// Multi-command settings
	StopOnFirstError bool // Stop multi-command execution on first error
}

// ExecutionContext contains context for command execution
type ExecutionContext struct {
	PRHandler       *handler.PRHandler
	Logger          logrus.FieldLogger
	Config          *ExecutionConfig
	MetricsRecorder MetricsRecorder

	// Additional context
	Platform       string // For metrics
	CommentSender  string // For validation
	TriggerComment string // For validation
}

// NewCLIExecutionConfig creates config for CLI mode
func NewCLIExecutionConfig(debugMode bool) *ExecutionConfig {
	return &ExecutionConfig{
		ValidateCommentSender:  !debugMode,
		ValidatePRStatus:       true,
		DebugMode:              debugMode,
		PostErrorsAsPRComments: true,
		ReturnErrors:           false,
		StopOnFirstError:       false,
	}
}

// NewWebhookExecutionConfig creates config for webhook modes
func NewWebhookExecutionConfig() *ExecutionConfig {
	return &ExecutionConfig{
		ValidateCommentSender:  false, // Webhook already validated
		ValidatePRStatus:       false, // Trust webhook
		DebugMode:              false,
		PostErrorsAsPRComments: false, // Log only
		ReturnErrors:           true,
		StopOnFirstError:       false,
	}
}
