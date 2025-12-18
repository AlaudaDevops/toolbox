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
	"fmt"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
)

// commandsSkipPRStatusCheck defines commands that can work with closed PRs
var commandsSkipPRStatusCheck = map[string]bool{
	"cherry-pick": true,
	"cherrypick":  true,
}

// Validator handles validation logic for command execution
type Validator struct {
	context *ExecutionContext
}

// NewValidator creates a new validator instance
func NewValidator(ctx *ExecutionContext) *Validator {
	return &Validator{
		context: ctx,
	}
}

// ValidateSingleCommand validates a single command before execution
func (v *Validator) ValidateSingleCommand(command string) error {
	// Check PR status if validation is enabled
	if v.context.Config.ValidatePRStatus && !v.shouldSkipPRStatusCheck(command) {
		if err := v.context.PRHandler.CheckPRStatus("open"); err != nil {
			return fmt.Errorf("PR status check failed: %w", err)
		}
	}

	// Validate comment sender if enabled and not in debug mode
	if v.context.Config.ValidateCommentSender && !v.context.Config.DebugMode {
		if err := v.validateCommentSender(); err != nil {
			return fmt.Errorf("comment sender validation failed: %w", err)
		}
	}

	return nil
}

// ValidateMultiCommand validates multiple commands before execution
func (v *Validator) ValidateMultiCommand(subCommands []SubCommand, rawCommandLines []string) error {
	// Check if any command needs PR status validation
	needsPRCheck := false
	if v.context.Config.ValidatePRStatus {
		for _, subCmd := range subCommands {
			if !v.shouldSkipPRStatusCheck(subCmd.Command) && !handler.IsBuiltInCommand(subCmd.Command) {
				needsPRCheck = true
				break
			}
		}
	}

	if needsPRCheck {
		if err := v.context.PRHandler.CheckPRStatus("open"); err != nil {
			return fmt.Errorf("PR status check failed: %w", err)
		}
	}

	// Validate comment sender if enabled and not in debug mode
	if v.context.Config.ValidateCommentSender && !v.context.Config.DebugMode {
		if err := v.validateCommentSenderMulti(rawCommandLines); err != nil {
			return fmt.Errorf("comment sender validation failed: %w", err)
		}
	}

	return nil
}

// shouldSkipPRStatusCheck returns true if the command can work with closed PRs
func (v *Validator) shouldSkipPRStatusCheck(command string) bool {
	return commandsSkipPRStatusCheck[command] || handler.IsBuiltInCommand(command)
}

// validateCommentSender verifies that the comment sender posted the trigger comment
func (v *Validator) validateCommentSender() error {
	comments, err := v.context.PRHandler.GetCommentsWithCache()
	if err != nil {
		return fmt.Errorf("failed to get PR comments: %w", err)
	}

	// Get trigger comment from context (stored in CommentSender field for now)
	// This will be refactored to use a proper TriggerComment field
	triggerComment := v.context.CommentSender
	if triggerComment == "" {
		return fmt.Errorf("trigger comment not provided")
	}

	normalizedTrigger := comment.Normalize(triggerComment)

	// Check if any comment from the comment sender contains the trigger
	found := false
	for _, commentObj := range comments {
		if strings.EqualFold(commentObj.User.Login, v.context.CommentSender) {
			normalizedCommentBody := comment.Normalize(commentObj.Body)
			if normalizedCommentBody == normalizedTrigger || strings.Contains(normalizedCommentBody, normalizedTrigger) {
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Errorf("comment sender '%s' did not post a comment containing the trigger", v.context.CommentSender)
	}

	v.context.Logger.Infof("Comment sender validation passed: %s posted a comment containing the trigger", v.context.CommentSender)
	return nil
}

// validateCommentSenderMulti validates comment sender for multi-command execution
func (v *Validator) validateCommentSenderMulti(rawCommandLines []string) error {
	if len(rawCommandLines) == 0 {
		v.context.Logger.Infof("Multi-command comment sender validation passed: no commands to validate")
		return nil
	}

	comments, err := v.context.PRHandler.GetCommentsWithCache()
	if err != nil {
		return fmt.Errorf("failed to get PR comments: %w", err)
	}

	var hasSenderComments bool
	var missingCommands []string

	for _, cmdLine := range rawCommandLines {
		normalizedCmdLine := comment.Normalize(cmdLine)
		found := false

		for _, commentObj := range comments {
			if strings.EqualFold(commentObj.User.Login, v.context.CommentSender) {
				hasSenderComments = true
				normalizedCommentBody := comment.Normalize(commentObj.Body)
				if strings.Contains(normalizedCommentBody, normalizedCmdLine) {
					found = true
					break
				}
			}
		}

		if !found {
			missingCommands = append(missingCommands, cmdLine)
		}
	}

	if !hasSenderComments {
		return fmt.Errorf("comment sender '%s' did not post any comment", v.context.CommentSender)
	}

	if len(missingCommands) == 0 {
		v.context.Logger.Infof("Multi-command comment sender validation passed: %s posted comments containing all commands", v.context.CommentSender)
		return nil
	}

	return fmt.Errorf("comment sender '%s' did not post comments containing the following commands: %s", 
		v.context.CommentSender, strings.Join(missingCommands, ", "))
}
