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

package comment

import "strings"

// Normalize normalizes a comment for consistent processing by trimming whitespace
// and handling common escape sequences that might appear in command line inputs
func Normalize(comment string) string {
	// Trim whitespace first
	comment = strings.TrimSpace(comment)

	// Only handle escaped newlines at the end of commands (common in CLI contexts)
	// This prevents unintended replacements in the middle of user content
	// Loop to handle multiple consecutive escape sequences at the end
	for {
		trimmed := false
		if strings.HasSuffix(comment, "\\n") {
			comment = strings.TrimSuffix(comment, "\\n")
			trimmed = true
		}
		if strings.HasSuffix(comment, "\\r") {
			comment = strings.TrimSuffix(comment, "\\r")
			trimmed = true
		}
		if !trimmed {
			break
		}
	}

	// Trim again after suffix removal
	comment = strings.TrimSpace(comment)

	return comment
}

// IsMultiLineCommand checks if a comment contains multiple command lines
func IsMultiLineCommand(comment string) bool {
	commands := SplitCommandLines(comment)
	return len(commands) > 1
}

// SplitCommandLines splits a multi-line comment into individual command lines
func SplitCommandLines(comment string) []string {
	var commands []string

	// Normalize line endings - convert \r\n and \r to \n
	comment = strings.ReplaceAll(comment, "\r\n", "\n")
	comment = strings.ReplaceAll(comment, "\r", "\n")

	lines := strings.Split(strings.TrimSpace(comment), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines or non-command lines
		if line == "" || !strings.HasPrefix(line, "/") {
			continue
		}
		// Preprocess special commands before adding to the list
		line = PreprocessSpecialCommands(line)
		commands = append(commands, line)
	}

	return commands
}

// SplitRawCommandLines splits a multi-line comment into individual command lines without preprocessing
func SplitRawCommandLines(comment string) []string {
	var commands []string

	// Normalize line endings - convert \r\n and \r to \n
	comment = strings.ReplaceAll(comment, "\r\n", "\n")
	comment = strings.ReplaceAll(comment, "\r", "\n")

	lines := strings.Split(strings.TrimSpace(comment), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines or non-command lines
		if line == "" || !strings.HasPrefix(line, "/") {
			continue
		}
		// Do NOT preprocess special commands - keep original form
		commands = append(commands, line)
	}

	return commands
}

// processSpecialCommandTransformation handles the core logic for special command transformations
func processSpecialCommandTransformation(command, args string) (string, string) {
	// Handle special case: lgtm cancel -> remove-lgtm
	if command == "lgtm" && args == "cancel" {
		return "remove-lgtm", ""
	}
	// Add other special command transformations here as needed
	return command, args
}

// PreprocessSpecialCommands handles special command transformations for complete command strings
func PreprocessSpecialCommands(commandStr string) string {
	if !strings.HasPrefix(commandStr, "/") {
		return commandStr
	}

	// Split into command and args
	parts := strings.SplitN(commandStr[1:], " ", 2)
	command := parts[0]
	var args string
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	// Transform using core logic
	newCommand, newArgs := processSpecialCommandTransformation(command, args)

	// Reconstruct command string
	if newArgs == "" {
		return "/" + newCommand
	}
	return "/" + newCommand + " " + newArgs
}
