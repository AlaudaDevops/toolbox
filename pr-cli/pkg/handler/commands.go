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

// CommandHandler defines the interface for command handlers
type CommandHandler func(args []string) error

// IsBuiltInCommand returns true if the command is a built-in command (starts with __)
func IsBuiltInCommand(command string) bool {
	return strings.HasPrefix(command, "__")
}

// ExecuteCommand executes a command with the given arguments
// This is the unified command execution method used by regular, sub-command, and built-in command flows
func (h *PRHandler) ExecuteCommand(command string, args []string) error {
	h.Infof("Executing command: %s with args: %v", command, args)

	// Handle built-in commands (commands starting with __)
	if IsBuiltInCommand(command) {
		return h.executeBuiltInCommand(command, args)
	}

	// Handle regular commands using command registry
	handler := h.getCommandHandler(command)
	if handler == nil {
		return fmt.Errorf("unknown command: %s", command)
	}

	return handler(args)
}

// getCommandHandler returns the handler function for a given command
func (h *PRHandler) getCommandHandler(command string) CommandHandler {
	commandRegistry := map[string]CommandHandler{
		"help":        h.HandleHelp,
		"assign":      h.HandleAssign,
		"unassign":    h.HandleUnassign,
		"lgtm":        h.HandleLGTM,
		"remove-lgtm": h.HandleRemoveLGTM,
		"merge":       h.HandleMerge,
		"ready":       h.HandleMerge, // alias for merge
		"close":       h.HandleClose,
		"rebase":      h.HandleRebase,
		"check":       h.HandleCheck,
		"batch":       h.HandleBatch,
		"cherry-pick": h.HandleCherrypick,
		"cherrypick":  h.HandleCherrypick, // alias for cherry-pick
		"label":       h.HandleLabel,
		"unlabel":     h.HandleUnlabel,
		"retest":      h.HandleRetest,
	}

	return commandRegistry[command]
}

// executeBuiltInCommand handles execution of built-in commands (commands starting with __)
func (h *PRHandler) executeBuiltInCommand(command string, _ []string) error {
	switch command {
	case "__post-merge-cherry-pick":
		return h.HandlePostMergeCherryPick()
	default:
		return fmt.Errorf("unknown built-in command: %s", command)
	}
}
