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
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
)

func TestValidator_ValidateSingleCommand(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		config         *ExecutionConfig
		prStatus       error
		comments       []git.Comment
		wantErr        bool
		errContains    string
	}{
		{
			name:    "valid command with open PR",
			command: "merge",
			config: &ExecutionConfig{
				ValidatePRStatus:      true,
				ValidateCommentSender: false,
			},
			prStatus: nil,
			wantErr:  false,
		},
		{
			name:    "cherry-pick allows closed PR",
			command: "cherry-pick",
			config: &ExecutionConfig{
				ValidatePRStatus:      true,
				ValidateCommentSender: false,
			},
			prStatus: nil,
			wantErr:  false,
		},
		{
			name:    "command fails on closed PR",
			command: "merge",
			config: &ExecutionConfig{
				ValidatePRStatus:      true,
				ValidateCommentSender: false,
			},
			prStatus:    fmt.Errorf("PR is closed"),
			wantErr:     true,
			errContains: "PR status check failed",
		},
		{
			name:    "comment sender validation passes",
			command: "merge",
			config: &ExecutionConfig{
				ValidatePRStatus:      false,
				ValidateCommentSender: true,
				DebugMode:             false,
			},
			comments: []git.Comment{
				{
					Body: "/merge",
					User: git.User{Login: "test-user"},
				},
			},
			wantErr: false,
		},
		{
			name:    "comment sender validation fails",
			command: "merge",
			config: &ExecutionConfig{
				ValidatePRStatus:      false,
				ValidateCommentSender: true,
				DebugMode:             false,
			},
			comments: []git.Comment{
				{
					Body: "/rebase",
					User: git.User{Login: "test-user"},
				},
			},
			wantErr:     true,
			errContains: "comment sender validation failed",
		},
		{
			name:    "debug mode skips comment sender validation",
			command: "merge",
			config: &ExecutionConfig{
				ValidatePRStatus:      false,
				ValidateCommentSender: true,
				DebugMode:             true,
			},
			comments: []git.Comment{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := &MockPRHandler{
				CheckPRStatusFunc: func(expectedStatus string) error {
					return tt.prStatus
				},
				GetCommentsWithCacheFunc: func() ([]git.Comment, error) {
					return tt.comments, nil
				},
			}

			ctx := &ExecutionContext{
				PRHandler:     mockHandler,
				Logger:        &MockLogger{},
				Config:        tt.config,
				CommentSender: "test-user",
			}

			validator := NewValidator(ctx)
			err := validator.ValidateSingleCommand(tt.command)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSingleCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" && err != nil {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateSingleCommand() error = %v, want error containing %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidator_ValidateMultiCommand(t *testing.T) {
	tests := []struct {
		name            string
		subCommands     []SubCommand
		rawCommandLines []string
		config          *ExecutionConfig
		prStatus        error
		comments        []git.Comment
		wantErr         bool
		errContains     string
	}{
		{
			name: "all commands valid",
			subCommands: []SubCommand{
				{Command: "lgtm", Args: nil},
				{Command: "merge", Args: nil},
			},
			rawCommandLines: []string{"/lgtm", "/merge"},
			config: &ExecutionConfig{
				ValidatePRStatus:      true,
				ValidateCommentSender: false,
			},
			prStatus: nil,
			wantErr:  false,
		},
		{
			name: "skip PR check when all commands are cherry-pick",
			subCommands: []SubCommand{
				{Command: "cherry-pick", Args: []string{"branch1"}},
				{Command: "cherrypick", Args: []string{"branch2"}},
			},
			rawCommandLines: []string{"/cherry-pick branch1", "/cherrypick branch2"},
			config: &ExecutionConfig{
				ValidatePRStatus:      true,
				ValidateCommentSender: false,
			},
			prStatus: fmt.Errorf("PR is closed"),
			wantErr:  false,
		},
		{
			name: "partial validation failure - PR closed",
			subCommands: []SubCommand{
				{Command: "lgtm", Args: nil},
				{Command: "merge", Args: nil},
			},
			rawCommandLines: []string{"/lgtm", "/merge"},
			config: &ExecutionConfig{
				ValidatePRStatus:      true,
				ValidateCommentSender: false,
			},
			prStatus:    fmt.Errorf("PR is closed"),
			wantErr:     true,
			errContains: "PR status check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := &MockPRHandler{
				CheckPRStatusFunc: func(expectedStatus string) error {
					return tt.prStatus
				},
				GetCommentsWithCacheFunc: func() ([]git.Comment, error) {
					return tt.comments, nil
				},
			}

			ctx := &ExecutionContext{
				PRHandler:     mockHandler,
				Logger:        &MockLogger{},
				Config:        tt.config,
				CommentSender: "test-user",
			}

			validator := NewValidator(ctx)
			err := validator.ValidateMultiCommand(tt.subCommands, tt.rawCommandLines)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMultiCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" && err != nil {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateMultiCommand() error = %v, want error containing %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidator_shouldSkipPRStatusCheck(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{"cherry-pick command", "cherry-pick", true},
		{"cherrypick command", "cherrypick", true},
		{"built-in command", "__post-merge-cherry-pick", true},
		{"rebase command", "rebase", false},
		{"merge command", "merge", false},
		{"lgtm command", "lgtm", false},
	}

	ctx := &ExecutionContext{
		Config: &ExecutionConfig{},
		Logger: &MockLogger{},
	}
	validator := NewValidator(ctx)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validator.shouldSkipPRStatusCheck(tt.command); got != tt.want {
				t.Errorf("shouldSkipPRStatusCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) > 0 && len(s) > 0 && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
