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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name        string
		comment     string
		wantType    CommandType
		wantCommand string
		wantMultiCommand []string
		wantArgs    []string
		wantErr     bool
	}{
		{
			name:        "single command without args",
			comment:     "/lgtm",
			wantType:    SingleCommand,
			wantCommand: "lgtm",
			wantArgs:    nil,
			wantErr:     false,
		},
		{
			name:        "single command with args",
			comment:     "/assign user1 user2",
			wantType:    SingleCommand,
			wantCommand: "assign",
			wantArgs:    []string{"user1", "user2"},
			wantErr:     false,
		},
		{
			name:        "built-in command",
			comment:     "/__test-command arg1",
			wantType:    BuiltInCommand,
			wantCommand: "__test-command",
			wantArgs:    []string{"arg1"},
			wantErr:     false,
		},
		{
			name:     "multi-line command",
			comment:  "/lgtm\n/ready",
			wantType: MultiCommand,
			wantMultiCommand: []string{"/lgtm", "/ready"},
			wantErr:  false,
		},
		{
			name:    "invalid command - no slash",
			comment: "lgtm",
			wantErr: true,
		},
		{
			name:    "invalid command format",
			comment: "/invalid-command",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCommand(tt.comment)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if got.Type != tt.wantType {
				t.Errorf("ParseCommand() Type = %v, want %v", got.Type, tt.wantType)
			}

			if tt.wantType == MultiCommand {
				// For multi-command, just check that we have command lines
				if len(got.CommandLines) == 0 {
					t.Errorf("ParseCommand() MultiCommand has no CommandLines")
				}
				diff := cmp.Diff(tt.wantMultiCommand, got.CommandLines)
				if diff != "" {
					t.Errorf("ParseCommand() MultiCommand CommandLines mismatch (-want +got):\n%s", diff)
				}
				return
			}

			if got.Command != tt.wantCommand {
				t.Errorf("ParseCommand() Command = %v, want %v", got.Command, tt.wantCommand)
			}

			if len(got.Args) != len(tt.wantArgs) {
				t.Errorf("ParseCommand() Args length = %v, want %v", len(got.Args), len(tt.wantArgs))
				return
			}

			for i, arg := range got.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("ParseCommand() Args[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestParseMultiCommandLines(t *testing.T) {
	tests := []struct {
		name             string
		commandLines     []string
		expectedCommands []SubCommand
		expectError      bool
		errorMessage     string
	}{
		{
			name:         "valid single command",
			commandLines: []string{"/lgtm"},
			expectedCommands: []SubCommand{
				{Command: "lgtm", Args: nil},
			},
			expectError: false,
		},
		{
			name:         "valid multiple commands",
			commandLines: []string{"/lgtm", "/ready", "/merge"},
			expectedCommands: []SubCommand{
				{Command: "lgtm", Args: nil},
				{Command: "ready", Args: nil},
				{Command: "merge", Args: nil},
			},
			expectError: false,
		},
		{
			name:         "command with multiple args",
			commandLines: []string{"/assign user1 user2", "/merge squash"},
			expectedCommands: []SubCommand{
				{Command: "assign", Args: []string{"user1", "user2"}},
				{Command: "merge", Args: []string{"squash"}},
			},
			expectError: false,
		},
		{
			name:         "nested multi-command should be skipped",
			commandLines: []string{"/ready", "/lgtm\n/merge", "/assign user1"},
			expectedCommands: []SubCommand{
				{Command: "ready", Args: nil},
				{Command: "assign", Args: []string{"user1"}},
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
			commandLines: []string{"/lgtm", "invalid", "/ready"},
			expectedCommands: []SubCommand{
				{Command: "lgtm", Args: nil},
				{Command: "ready", Args: nil},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseMultiCommandLines(tt.commandLines)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseMultiCommandLines() expected error but got none")
					return
				}
				if tt.errorMessage != "" && err.Error() != tt.errorMessage {
					t.Errorf("ParseMultiCommandLines() error = %v, want %v", err.Error(), tt.errorMessage)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseMultiCommandLines() unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expectedCommands) {
				t.Errorf("ParseMultiCommandLines() got %d commands, want %d", len(result), len(tt.expectedCommands))
				return
			}

			for i, cmd := range result {
				expected := tt.expectedCommands[i]
				if cmd.Command != expected.Command {
					t.Errorf("ParseMultiCommandLines()[%d].Command = %v, want %v", i, cmd.Command, expected.Command)
				}

				if len(cmd.Args) != len(expected.Args) {
					t.Errorf("ParseMultiCommandLines()[%d].Args length = %v, want %v", i, len(cmd.Args), len(expected.Args))
					continue
				}

				for j, arg := range cmd.Args {
					if arg != expected.Args[j] {
						t.Errorf("ParseMultiCommandLines()[%d].Args[%d] = %v, want %v", i, j, arg, expected.Args[j])
					}
				}
			}
		})
	}
}

func TestGetCommandDisplayName(t *testing.T) {
	tests := []struct {
		name    string
		subCmd  SubCommand
		want    string
	}{
		{
			name:   "command without args",
			subCmd: SubCommand{Command: "lgtm", Args: nil},
			want:   "/lgtm",
		},
		{
			name:   "command with single arg",
			subCmd: SubCommand{Command: "merge", Args: []string{"squash"}},
			want:   "/merge squash",
		},
		{
			name:   "command with multiple args",
			subCmd: SubCommand{Command: "assign", Args: []string{"user1", "user2"}},
			want:   "/assign user1 user2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCommandDisplayName(tt.subCmd)
			if got != tt.want {
				t.Errorf("GetCommandDisplayName() = %v, want %v", got, tt.want)
			}
		})
	}
}

