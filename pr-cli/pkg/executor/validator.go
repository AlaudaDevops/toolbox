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

// Validator handles command execution validation
type Validator struct {
	context *ExecutionContext
}

// NewValidator creates a new validator
func NewValidator(ctx *ExecutionContext) *Validator {
	return &Validator{context: ctx}
}

// ValidateSingleCommand validates a single command execution
func (v *Validator) ValidateSingleCommand(command string) error {
	// Validate PR status if required
	if v.context.Config.ValidatePRStatus && !v.shouldSkipPRStatusCheck(command) {
		if err := v.context.PRHandler.CheckPRStatus("open"); err != nil {
			return fmt.Errorf("PR status check failed: %w", err)
		}
	}

	// Validate comment sender if required
	if v.context.Config.ValidateCommentSender {
		if err := v.validateCommentSender(); err != nil {
			return fmt.Errorf("comment sender validation failed: %w", err)
		}
	}

	return nil
}

// ValidateMultiCommand validates multi-command execution
func (v *Validator) ValidateMultiCommand(subCommands []SubCommand, rawCommandLines []string) error {
	// Check if any command needs PR status validation
	needsPRCheck := false
	for _, subCmd := range subCommands {
		if !v.shouldSkipPRStatusCheck(subCmd.Command) && !handler.IsBuiltInCommand(subCmd.Command) {
			needsPRCheck = true
			break
		}
	}

	if v.context.Config.ValidatePRStatus && needsPRCheck {
		if err := v.context.PRHandler.CheckPRStatus("open"); err != nil {
			return fmt.Errorf("PR status check failed: %w", err)
		}
	}

	// Validate comment sender for multi-command
	if v.context.Config.ValidateCommentSender {
		if err := v.validateCommentSenderMulti(rawCommandLines); err != nil {
			return fmt.Errorf("comment sender validation failed: %w", err)
		}
	}

	return nil
}

// shouldSkipPRStatusCheck returns true if the command can work with closed PRs
func (v *Validator) shouldSkipPRStatusCheck(command string) bool {
	skipCommands := map[string]bool{
		"cherry-pick": true,
		"cherrypick":  true,
	}
	return skipCommands[command] || handler.IsBuiltInCommand(command)
}

// validateCommentSender validates that the comment sender posted the trigger comment
func (v *Validator) validateCommentSender() error {
	comments, err := v.context.PRHandler.GetCommentsWithCache()
	if err != nil {
		return fmt.Errorf("failed to get PR comments: %w", err)
	}

	// Get trigger comment from context
	normalizedTrigger := comment.Normalize(v.context.TriggerComment)

	for _, commentObj := range comments {
		if strings.EqualFold(commentObj.User.Login, v.context.CommentSender) {
			normalizedBody := comment.Normalize(commentObj.Body)
			if normalizedBody == normalizedTrigger || strings.Contains(normalizedBody, normalizedTrigger) {
				v.context.Logger.Infof("Comment sender validation passed: %s posted the trigger", v.context.CommentSender)
				return nil
			}
		}
	}

	return fmt.Errorf("comment sender '%s' did not post a comment containing the trigger", v.context.CommentSender)
}

// validateCommentSenderMulti validates comment sender for multi-command
func (v *Validator) validateCommentSenderMulti(rawCommandLines []string) error {
	if len(rawCommandLines) == 0 {
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
				normalizedBody := comment.Normalize(commentObj.Body)
				if strings.Contains(normalizedBody, normalizedCmdLine) {
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

	if len(missingCommands) > 0 {
		return fmt.Errorf("comment sender '%s' did not post commands: %s",
			v.context.CommentSender, strings.Join(missingCommands, ", "))
	}

	v.context.Logger.Infof("Multi-command validation passed for sender: %s", v.context.CommentSender)
	return nil
}
