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
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
)

// executeMultiCommand handles execution of multiple commands from a multi-line comment
func (p *PROption) executeMultiCommand(prHandler *handler.PRHandler, commandLines []string, rawCommandLines []string) error {
	p.Infof("Executing multi-line command with %d commands", len(commandLines))

	// Parse command lines into sub-commands
	subCommands, err := p.parseMultiCommandLines(commandLines)
	if err != nil {
		return err
	}

	// Validate permissions and PR status
	if err := p.validateMultiCommandExecution(prHandler, subCommands, rawCommandLines); err != nil {
		return err
	}

	// Execute the commands
	return p.handleMultiCommandExecution(prHandler, subCommands)
}

// parseMultiCommandLines parses multiple command lines into SubCommand structs
func (p *PROption) parseMultiCommandLines(commandLines []string) ([]handler.SubCommand, error) {
	var subCommands []handler.SubCommand

	for _, cmdLine := range commandLines {
		parsedCmd, err := p.parseCommand(cmdLine)
		if err != nil {
			p.Errorf("Failed to parse command line '%s': %v", cmdLine, err)
			continue
		}

		// Skip nested multi-command recursion
		if parsedCmd.Type == MultiCommand {
			p.Warnf("Ignoring nested multi-command in line: %s", cmdLine)
			continue
		}

		subCommands = append(subCommands, handler.SubCommand{
			Command: parsedCmd.Command,
			Args:    parsedCmd.Args,
		})
	}

	if len(subCommands) == 0 {
		return nil, fmt.Errorf("no valid commands found in multi-line comment")
	}

	return subCommands, nil
}

// validateMultiCommandExecution validates permissions and PR status for multi-command execution
func (p *PROption) validateMultiCommandExecution(prHandler *handler.PRHandler, subCommands []handler.SubCommand, rawCommandLines []string) error {
	// Check PR status - use the most restrictive check for all commands
	needsPRCheck := false
	for _, subCmd := range subCommands {
		if !p.shouldSkipPRStatusCheck(subCmd.Command) && !handler.IsBuiltInCommand(subCmd.Command) {
			needsPRCheck = true
			break
		}
	}

	if needsPRCheck {
		if err := prHandler.CheckPRStatus("open"); err != nil {
			return fmt.Errorf("PR status check failed: %w", err)
		}
	}

	// Validate comment sender in non-debug mode
	if !p.Config.Debug {
		if err := p.validateCommentSenderForMultiCommand(prHandler, rawCommandLines); err != nil {
			return fmt.Errorf("comment sender validation failed: %w", err)
		}
	}

	return nil
}

// handleMultiCommandExecution executes multiple sub-commands and posts a summary
func (p *PROption) handleMultiCommandExecution(prHandler *handler.PRHandler, subCommands []handler.SubCommand) error {
	var results []string
	var hasErrors bool

	for _, subCmd := range subCommands {
		result := p.processMultiCommand(prHandler, subCmd)
		results = append(results, result)

		// Check if this command failed
		if strings.HasPrefix(result, "❌") {
			hasErrors = true
		}
	}

	// Post summary with appropriate header
	header := "**Multi-Command Execution Results:**"
	if hasErrors {
		header = fmt.Sprintf("%s (⚠️ Some commands failed)", header)
	}

	summary := fmt.Sprintf("%s\n\n%s", header, strings.Join(results, "\n"))
	return prHandler.PostComment(summary)
}

// processMultiCommand executes a single command in multi-command context
func (p *PROption) processMultiCommand(prHandler *handler.PRHandler, subCmd handler.SubCommand) string {
	cmdDisplay := p.getCommandDisplayName(subCmd)

	// Execute the command
	if err := prHandler.ExecuteCommand(subCmd.Command, subCmd.Args); err != nil {
		p.Errorf("Multi-command '%s' failed: %v", subCmd.Command, err)
		return fmt.Sprintf("❌ Command `%s` failed: %v", cmdDisplay, err)
	}

	return fmt.Sprintf("✅ Command `%s` executed successfully", cmdDisplay)
}

// getCommandDisplayName returns the display name for a command including its arguments
func (p *PROption) getCommandDisplayName(subCmd handler.SubCommand) string {
	if len(subCmd.Args) == 0 {
		return fmt.Sprintf("/%s", subCmd.Command)
	}
	return fmt.Sprintf("/%s %s", subCmd.Command, strings.Join(subCmd.Args, " "))
}

// validateCommentSenderForMultiCommand validates comment sender for multi-command execution
func (p *PROption) validateCommentSenderForMultiCommand(prHandler *handler.PRHandler, rawCommandLines []string) error {
	// Handle edge case: no commands to validate
	if len(rawCommandLines) == 0 {
		p.Infof("Multi-command comment sender validation passed: no commands to validate")
		return nil
	}

	// Get all comments from the PR
	comments, err := prHandler.GetCommentsWithCache()
	if err != nil {
		return fmt.Errorf("failed to get PR comments: %w", err)
	}

	// Collect all missing commands
	var hasSenderComments bool
	var missingCommands []string

	for _, cmdLine := range rawCommandLines {
		normalizedCmdLine := comment.Normalize(cmdLine)
		found := false

		for _, commentObj := range comments {
			if strings.EqualFold(commentObj.User.Login, p.Config.CommentSender) {
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

	// Return appropriate error message
	if !hasSenderComments {
		return fmt.Errorf("comment sender '%s' did not post any comment", p.Config.CommentSender)
	}

	// If no commands are missing, validation passes
	if len(missingCommands) == 0 {
		p.Infof("Multi-command comment sender validation passed: %s posted comments containing all commands", p.Config.CommentSender)
		return nil
	}

	return fmt.Errorf("comment sender '%s' did not post comments containing the following commands: %s", p.Config.CommentSender, strings.Join(missingCommands, ", "))
}
