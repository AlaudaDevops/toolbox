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
	"reflect"
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/executor"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
)

// TestParseMultiCommandLines tests the parseMultiCommandLines function
func TestParseMultiCommandLines(t *testing.T) {
	option := NewPROption()

	tests := []struct {
		name             string
		commandLines     []string
		expectedCommands []handler.SubCommand
		expectError      bool
		errorMessage     string
	}{
		{
			name:         "valid single command",
			commandLines: []string{"/ready"},
			expectedCommands: []handler.SubCommand{
				{Command: "ready", Args: nil},
			},
			expectError: false,
		},
		{
			name:         "valid multiple commands",
			commandLines: []string{"/ready", "/assign user1", "/lgtm"},
			expectedCommands: []handler.SubCommand{
				{Command: "ready", Args: nil},
				{Command: "assign", Args: []string{"user1"}},
				{Command: "lgtm", Args: nil},
			},
			expectError: false,
		},
		{
			name:         "command with multiple args",
			commandLines: []string{"/assign user1 user2", "/merge squash"},
			expectedCommands: []handler.SubCommand{
				{Command: "assign", Args: []string{"user1", "user2"}},
				{Command: "merge", Args: []string{"squash"}},
			},
			expectError: false,
		},
		{
			name:         "nested multi-command should be skipped",
			commandLines: []string{"/ready", "/multi", "/lgtm"},
			expectedCommands: []handler.SubCommand{
				{Command: "ready", Args: nil},
				{Command: "lgtm", Args: nil},
			},
			expectError: false,
		},
		{
			name:         "empty command lines",
			commandLines: []string{},
			expectError:  true,
			errorMessage: "no valid commands found in multi-line comment",
		},
		{
			name:         "only invalid commands",
			commandLines: []string{"invalid", "also invalid"},
			expectError:  true,
			errorMessage: "no valid commands found in multi-line comment",
		},
		{
			name:         "mix of valid and invalid commands",
			commandLines: []string{"invalid", "/ready", "/lgtm"},
			expectedCommands: []handler.SubCommand{
				{Command: "ready", Args: nil},
				{Command: "lgtm", Args: nil},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := option.parseMultiCommandLines(tt.commandLines)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseMultiCommandLines() expected error, got none")
					return
				}
				if tt.errorMessage != "" && err.Error() != tt.errorMessage {
					t.Errorf("parseMultiCommandLines() error message = %v, want %v", err.Error(), tt.errorMessage)
				}
				return
			}

			if err != nil {
				t.Errorf("parseMultiCommandLines() unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expectedCommands) {
				t.Errorf("parseMultiCommandLines() returned %d commands, want %d", len(result), len(tt.expectedCommands))
				return
			}

			for i, cmd := range result {
				expected := tt.expectedCommands[i]
				if cmd.Command != expected.Command {
					t.Errorf("parseMultiCommandLines() command[%d] = %s, want %s", i, cmd.Command, expected.Command)
				}
				if !reflect.DeepEqual(cmd.Args, expected.Args) {
					t.Errorf("parseMultiCommandLines() args[%d] = %v, want %v", i, cmd.Args, expected.Args)
				}
			}
		})
	}
}

// TestGetCommandDisplayName tests the GetCommandDisplayName function
func TestGetCommandDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		subCmd   handler.SubCommand
		expected string
	}{
		{
			name:     "command without args",
			subCmd:   handler.SubCommand{Command: "ready", Args: []string{}},
			expected: "/ready",
		},
		{
			name:     "command with nil args",
			subCmd:   handler.SubCommand{Command: "ready", Args: nil},
			expected: "/ready",
		},
		{
			name:     "command with single arg",
			subCmd:   handler.SubCommand{Command: "assign", Args: []string{"user1"}},
			expected: "/assign user1",
		},
		{
			name:     "command with multiple args",
			subCmd:   handler.SubCommand{Command: "assign", Args: []string{"user1", "user2"}},
			expected: "/assign user1 user2",
		},
		{
			name:     "command with complex args",
			subCmd:   handler.SubCommand{Command: "merge", Args: []string{"squash", "--no-edit"}},
			expected: "/merge squash --no-edit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert handler.SubCommand to executor.SubCommand
			execSubCmd := executor.SubCommand{
				Command: tt.subCmd.Command,
				Args:    tt.subCmd.Args,
			}
			result := executor.GetCommandDisplayName(execSubCmd)
			if result != tt.expected {
				t.Errorf("GetCommandDisplayName() = %s, want %s", result, tt.expected)
			}
		})
	}
}
