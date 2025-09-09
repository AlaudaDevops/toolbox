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

package handler

import (
	"fmt"
	"strings"
)

// SubCommand represents a parsed sub-command with its arguments
type SubCommand struct {
	Command string
	Args    []string
}

// HandleBatch executes multiple commands in batch mode
func (h *PRHandler) HandleBatch(args []string) error {
	// Parse sub-commands with arguments (reuse existing logic)
	subCommands := h.parseSubCommands(args)
	if len(subCommands) == 0 {
		return fmt.Errorf("no valid commands provided for batch execution")
	}

	// Execute batch commands with batch-specific validation
	return h.handleBatchCommands(subCommands)
}

// handleBatchCommands executes batch commands and returns a summary
func (h *PRHandler) handleBatchCommands(subCommands []SubCommand) error {
	return h.handleSubCommands(subCommands, "**Batch Execution Results:**")
}

// handleSubCommands executes sub-commands and returns a summary with the specified header
func (h *PRHandler) handleSubCommands(subCommands []SubCommand, headerText string) error {
	var results []string
	var hasErrors bool

	for _, subCmd := range subCommands {
		result := h.processBatchCommand(subCmd)
		results = append(results, result)

		// Check if this command failed
		if strings.HasPrefix(result, "❌") {
			hasErrors = true
		}
	}

	// Post summary with appropriate header
	header := headerText
	if hasErrors {
		header = fmt.Sprintf("%s (⚠️ Some commands failed)", headerText)
	}

	summary := fmt.Sprintf("%s\n\n%s", header, strings.Join(results, "\n"))
	return h.client.PostComment(summary)
}

// processBatchCommand validates and executes a single batch command, returning the result message
func (h *PRHandler) processBatchCommand(subCmd SubCommand) string {
	// Validate batch command with batch-specific rules
	if validationErr := h.validateBatchCommand(subCmd); validationErr != "" {
		return validationErr
	}

	// Execute the batch command (reuse existing logic)
	if err := h.executeSubCommand(subCmd.Command, subCmd.Args); err != nil {
		h.Errorf("Batch command '%s' failed: %v", subCmd.Command, err)
		return h.formatResult("❌", subCmd, fmt.Sprintf("failed: %v", err))
	}

	return h.formatResult("✅", subCmd, "executed successfully")
}

// batchProhibitedCommands defines commands that are not allowed in batch execution
var batchProhibitedCommands = map[string]bool{
	"batch":       true, // Prevent recursive batch calls
	"lgtm":        true, // LGTM commands not supported in batch
	"remove-lgtm": true, // LGTM commands not supported in batch
}

// validateBatchCommand validates if a command can be executed via batch
func (h *PRHandler) validateBatchCommand(subCmd SubCommand) string {
	cmdDisplay := h.getCommandDisplayName(subCmd)

	// Filter out built-in commands (starting with __)
	if IsBuiltInCommand(subCmd.Command) {
		h.Warnf("Built-in command '%s' is not allowed via batch", subCmd.Command)
		return fmt.Sprintf("❌ Command `%s` is not allowed in batch execution", cmdDisplay)
	}

	// Filter out prohibited commands for batch execution
	if batchProhibitedCommands[subCmd.Command] {
		h.Warnf("Command '%s' is prohibited in batch execution", subCmd.Command)
		return fmt.Sprintf("❌ Command `%s` is not allowed in batch execution", cmdDisplay)
	}

	return "" // Valid command - ExecuteCommand will handle unknown commands
}

// formatResult formats a sub-command execution result
func (h *PRHandler) formatResult(icon string, subCmd SubCommand, status string) string {
	cmdDisplay := h.getCommandDisplayName(subCmd)
	return fmt.Sprintf("%s Command `%s` %s", icon, cmdDisplay, status)
}

// getCommandDisplayName returns the display name for a command including its arguments
func (h *PRHandler) getCommandDisplayName(subCmd SubCommand) string {
	if len(subCmd.Args) == 0 {
		return fmt.Sprintf("/%s", subCmd.Command)
	}
	return fmt.Sprintf("/%s %s", subCmd.Command, strings.Join(subCmd.Args, " "))
}

// parseSubCommands parses arguments into sub-commands with their arguments
// Example: ["/merge", "rebase", "/lgtm", "/assign", "user1", "user2"]
// Returns: [{merge, [rebase]}, {lgtm, []}, {assign, [user1, user2]}]
func (h *PRHandler) parseSubCommands(args []string) []SubCommand {
	var subCommands []SubCommand
	var currentCmd *SubCommand

	for _, arg := range args {
		if strings.HasPrefix(arg, "/") {
			// Start a new command
			if currentCmd != nil {
				subCommands = append(subCommands, *currentCmd)
			}

			command := strings.TrimPrefix(arg, "/")
			currentCmd = &SubCommand{
				Command: command,
				Args:    []string{},
			}
		} else if currentCmd != nil {
			// Add argument to current command
			currentCmd.Args = append(currentCmd.Args, arg)
		}
		// Ignore arguments that don't belong to any command
	}

	// Add the last command
	if currentCmd != nil {
		subCommands = append(subCommands, *currentCmd)
	}

	return subCommands
}

// executeSubCommand executes a single sub-command with arguments
func (h *PRHandler) executeSubCommand(command string, args []string) error {
	return h.ExecuteCommand(command, args)
}
