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
	"regexp"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
	"github.com/google/shlex"
)

var (
	// Match pattern: /command [args...] or /__built-in-command [args...]
	commentRegexp = regexp.MustCompile(`^/(help|rebase|lgtm|remove-lgtm|cherry-?pick|assign|merge|ready|unassign|label|unlabel|check|retest|close|batch|checkbox|checkbox-issue)($|\s.*)`)
	// Match pattern for built-in commands: /__command [args...]
	builtInCommandRegexp = regexp.MustCompile(`^/(__[a-z-_]+)\s*(.*)$`)
)

// ParseCommand parses a trigger comment and returns a structured command object
func ParseCommand(commentStr string) (*ParsedCommand, error) {
	commentStr = comment.Normalize(commentStr)
	if !strings.HasPrefix(commentStr, "/") {
		return nil, fmt.Errorf("comment must start with /")
	}

	// Check if this is a multi-line command
	if comment.IsMultiLineCommand(commentStr) {
		commandLines := comment.SplitCommandLines(commentStr)
		rawCommandLines := comment.SplitRawCommandLines(commentStr)
		return &ParsedCommand{
			Type:            MultiCommand,
			CommandLines:    commandLines,
			RawCommandLines: rawCommandLines,
		}, nil
	}

	// Try to match built-in commands first (/__command)
	if builtInMatches := builtInCommandRegexp.FindStringSubmatch(commentStr); len(builtInMatches) >= 2 {
		command := builtInMatches[1] // Built-in command with __ prefix already captured
		argsStr := strings.TrimSpace(builtInMatches[2])

		args, err := splitArgs(argsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid command arguments: %w", err)
		}

		return &ParsedCommand{
			Type:    BuiltInCommand,
			Command: command,
			Args:    args,
		}, nil
	}

	// Try to match regular commands (/command)
	matches := commentRegexp.FindStringSubmatch(commentStr)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid command format")
	}

	// Handle special command transformations on the full command
	commentStr = comment.PreprocessSpecialCommands(commentStr)

	// Re-parse after transformation
	matches = commentRegexp.FindStringSubmatch(commentStr)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid command format after transformation")
	}

	command := matches[1]
	argsStr := strings.TrimSpace(matches[2])

	args, err := splitArgs(argsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid command arguments: %w", err)
	}

	return &ParsedCommand{
		Type:    SingleCommand,
		Command: command,
		Args:    args,
	}, nil
}

// ParseMultiCommandLines parses multiple command lines into SubCommand structs
func ParseMultiCommandLines(commandLines []string) ([]SubCommand, error) {
	var subCommands []SubCommand

	for _, cmdLine := range commandLines {
		parsedCmd, err := ParseCommand(cmdLine)
		if err != nil {
			// Skip invalid commands but continue processing
			continue
		}

		// Skip nested multi-command recursion
		if parsedCmd.Type == MultiCommand {
			continue
		}

		subCommands = append(subCommands, SubCommand{
			Command: parsedCmd.Command,
			Args:    parsedCmd.Args,
		})
	}

	if len(subCommands) == 0 {
		return nil, fmt.Errorf("no valid commands found in multi-line comment")
	}

	return subCommands, nil
}

// splitArgs splits argument string into individual arguments using shell-like parsing
func splitArgs(argsStr string) ([]string, error) {
	if strings.TrimSpace(argsStr) == "" {
		return nil, nil
	}
	return shlex.Split(argsStr)
}

// GetCommandDisplayName returns a display name for a command with its arguments
func GetCommandDisplayName(subCmd SubCommand) string {
	if len(subCmd.Args) > 0 {
		return fmt.Sprintf("/%s %s", subCmd.Command, strings.Join(subCmd.Args, " "))
	}
	return fmt.Sprintf("/%s", subCmd.Command)
}

