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

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
)

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

// executeSingleCommand handles execution of a single regular command
func (p *PROption) executeSingleCommand(prHandler *handler.PRHandler, command string, cmdArgs []string) error {
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
