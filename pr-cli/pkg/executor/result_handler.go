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
	"errors"
	"fmt"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
)

// ResultHandler handles command execution results
type ResultHandler struct {
	context *ExecutionContext
}

// NewResultHandler creates a new result handler instance
func NewResultHandler(ctx *ExecutionContext) *ResultHandler {
	return &ResultHandler{
		context: ctx,
	}
}

// HandleSingleCommandError handles errors from single command execution
func (r *ResultHandler) HandleSingleCommandError(command string, err error) error {
	// Check if this is a CommentedError (comment already posted)
	var commentedErr *handler.CommentedError
	if errors.As(err, &commentedErr) {
		r.context.Logger.Infof("Error comment already posted for command: %s", command)
		// Don't return error if not configured to do so
		if !r.context.Config.ReturnErrors {
			return nil
		}
		return err
	}

	// Post error as PR comment if configured
	if r.context.Config.PostErrorsAsPRComments {
		errorMessage := fmt.Sprintf(messages.CommandErrorTemplate, command, err.Error())
		if commentErr := r.context.PRHandler.PostComment(errorMessage); commentErr != nil {
			r.context.Logger.Errorf("Failed to post error comment: %v", commentErr)
			// Return both errors if configured
			if r.context.Config.ReturnErrors {
				return fmt.Errorf("command failed: %w (and failed to post error comment: %v)", err, commentErr)
			}
		} else {
			r.context.Logger.Infof("Posted command error as PR comment for command: %s", command)
		}
	}

	// Return error if configured
	if r.context.Config.ReturnErrors {
		return err
	}

	// Just log the error otherwise
	r.context.Logger.Errorf("Command %s failed: %v", command, err)
	return nil
}

// HandleMultiCommandResults handles results from multi-command execution
func (r *ResultHandler) HandleMultiCommandResults(results []SubCommandResult) error {
	var formattedResults []string
	var hasErrors bool

	for _, result := range results {
		formatted := r.FormatSubCommandResult(result)
		formattedResults = append(formattedResults, formatted)
		
		if !result.Success {
			hasErrors = true
		}
	}

	// Create summary with appropriate header
	header := "**Multi-Command Execution Results:**"
	if hasErrors {
		header = fmt.Sprintf("%s (⚠️ Some commands failed)", header)
	}

	summary := fmt.Sprintf("%s\n\n%s", header, strings.Join(formattedResults, "\n"))
	return r.context.PRHandler.PostComment(summary)
}

// FormatSubCommandResult formats a single sub-command result
func (r *ResultHandler) FormatSubCommandResult(result SubCommandResult) string {
	cmdDisplay := GetCommandDisplayName(SubCommand{
		Command: result.Command,
		Args:    result.Args,
	})

	if result.Success {
		return fmt.Sprintf("✅ Command `%s` executed successfully", cmdDisplay)
	}

	return fmt.Sprintf("❌ Command `%s` failed: %v", cmdDisplay, result.Error)
}
