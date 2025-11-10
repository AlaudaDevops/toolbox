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

package webhook

import (
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/executor"
)

func TestMultiCommandParsing(t *testing.T) {
	tests := []struct {
		name          string
		comment       string
		wantType      executor.CommandType
		wantNumCmds   int
		wantErr       bool
	}{
		{
			name:        "single command",
			comment:     "/lgtm",
			wantType:    executor.SingleCommand,
			wantNumCmds: 0,
			wantErr:     false,
		},
		{
			name:        "multi-line command",
			comment:     "/lgtm\n/ready",
			wantType:    executor.MultiCommand,
			wantNumCmds: 2,
			wantErr:     false,
		},
		{
			name:        "multi-line with empty lines",
			comment:     "/lgtm\n\n/ready\n/merge squash",
			wantType:    executor.MultiCommand,
			wantNumCmds: 3,
			wantErr:     false,
		},
		{
			name:        "multi-line with args",
			comment:     "/assign user1 user2\n/label bug\n/merge squash",
			wantType:    executor.MultiCommand,
			wantNumCmds: 3,
			wantErr:     false,
		},
		{
			name:        "single command with args",
			comment:     "/assign user1 user2",
			wantType:    executor.SingleCommand,
			wantNumCmds: 0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedCmd, err := executor.ParseCommand(tt.comment)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if parsedCmd.Type != tt.wantType {
				t.Errorf("ParseCommand() Type = %v, want %v", parsedCmd.Type, tt.wantType)
			}

			if tt.wantType == executor.MultiCommand {
				if len(parsedCmd.CommandLines) != tt.wantNumCmds {
					t.Errorf("ParseCommand() CommandLines length = %v, want %v", len(parsedCmd.CommandLines), tt.wantNumCmds)
				}
			}
		})
	}
}

func TestParseMultiCommandLines(t *testing.T) {
	tests := []struct {
		name        string
		commandLines []string
		wantNumCmds int
		wantErr     bool
	}{
		{
			name:         "valid commands",
			commandLines: []string{"/lgtm", "/ready", "/merge squash"},
			wantNumCmds:  3,
			wantErr:      false,
		},
		{
			name:         "commands with args",
			commandLines: []string{"/assign user1 user2", "/label bug feature"},
			wantNumCmds:  2,
			wantErr:      false,
		},
		{
			name:         "empty command lines",
			commandLines: []string{},
			wantNumCmds:  0,
			wantErr:      true,
		},
		{
			name:         "invalid commands",
			commandLines: []string{"invalid", "also invalid"},
			wantNumCmds:  0,
			wantErr:      true,
		},
		{
			name:         "mix of valid and invalid",
			commandLines: []string{"/lgtm", "invalid", "/ready"},
			wantNumCmds:  2,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subCommands, err := executor.ParseMultiCommandLines(tt.commandLines)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMultiCommandLines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if len(subCommands) != tt.wantNumCmds {
				t.Errorf("ParseMultiCommandLines() got %d commands, want %d", len(subCommands), tt.wantNumCmds)
			}
		})
	}
}

func TestGetCommandDisplayName(t *testing.T) {
	tests := []struct {
		name    string
		subCmd  executor.SubCommand
		want    string
	}{
		{
			name:   "command without args",
			subCmd: executor.SubCommand{Command: "lgtm", Args: nil},
			want:   "/lgtm",
		},
		{
			name:   "command with single arg",
			subCmd: executor.SubCommand{Command: "merge", Args: []string{"squash"}},
			want:   "/merge squash",
		},
		{
			name:   "command with multiple args",
			subCmd: executor.SubCommand{Command: "assign", Args: []string{"user1", "user2"}},
			want:   "/assign user1 user2",
		},
		{
			name:   "command with complex args",
			subCmd: executor.SubCommand{Command: "label", Args: []string{"bug", "feature"}},
			want:   "/label bug feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.GetCommandDisplayName(tt.subCmd)
			if got != tt.want {
				t.Errorf("GetCommandDisplayName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebhookEventIsCommandComment(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		want    bool
	}{
		{
			name:    "single command",
			comment: "/lgtm",
			want:    true,
		},
		{
			name:    "multi-line command",
			comment: "/lgtm\n/ready",
			want:    true,
		},
		{
			name:    "command with leading whitespace",
			comment: "  /lgtm",
			want:    true,
		},
		{
			name:    "not a command",
			comment: "This is just a comment",
			want:    false,
		},
		{
			name:    "empty comment",
			comment: "",
			want:    false,
		},
		{
			name:    "slash in middle",
			comment: "This has a /slash but not at start",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &WebhookEvent{
				Comment: Comment{
					Body: tt.comment,
				},
			}
			got := event.IsCommandComment()
			if got != tt.want {
				t.Errorf("IsCommandComment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiCommandExecution(t *testing.T) {
	// Test that multi-command parsing works end-to-end
	multiComment := "/lgtm\n/ready\n/merge squash"

	parsedCmd, err := executor.ParseCommand(multiComment)
	if err != nil {
		t.Fatalf("ParseCommand() error = %v", err)
	}

	if parsedCmd.Type != executor.MultiCommand {
		t.Errorf("ParseCommand() Type = %v, want %v", parsedCmd.Type, executor.MultiCommand)
	}

	if len(parsedCmd.CommandLines) != 3 {
		t.Errorf("ParseCommand() CommandLines length = %v, want 3", len(parsedCmd.CommandLines))
	}

	// Parse the command lines into sub-commands
	subCommands, err := executor.ParseMultiCommandLines(parsedCmd.CommandLines)
	if err != nil {
		t.Fatalf("ParseMultiCommandLines() error = %v", err)
	}

	if len(subCommands) != 3 {
		t.Errorf("ParseMultiCommandLines() got %d commands, want 3", len(subCommands))
	}

	// Verify each command
	expectedCommands := []struct {
		command string
		args    []string
	}{
		{command: "lgtm", args: nil},
		{command: "ready", args: nil},
		{command: "merge", args: []string{"squash"}},
	}

	for i, expected := range expectedCommands {
		if subCommands[i].Command != expected.command {
			t.Errorf("subCommands[%d].Command = %v, want %v", i, subCommands[i].Command, expected.command)
		}

		if len(subCommands[i].Args) != len(expected.args) {
			t.Errorf("subCommands[%d].Args length = %v, want %v", i, len(subCommands[i].Args), len(expected.args))
			continue
		}

		for j, arg := range expected.args {
			if subCommands[i].Args[j] != arg {
				t.Errorf("subCommands[%d].Args[%d] = %v, want %v", i, j, subCommands[i].Args[j], arg)
			}
		}
	}
}

