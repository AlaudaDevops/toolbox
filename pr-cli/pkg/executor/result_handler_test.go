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
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
)

func TestResultHandler_HandleSingleCommandError(t *testing.T) {
	tests := []struct {
		name                   string
		command                string
		err                    error
		postErrorsAsPRComments bool
		returnErrors           bool
		postCommentErr         error
		wantErr                bool
		wantCommentPosted      bool
		wantLoggedError        bool
	}{
		{
			name:                   "posts error as PR comment",
			command:                "rebase",
			err:                    errors.New("rebase failed"),
			postErrorsAsPRComments: true,
			returnErrors:           false,
			wantErr:                false,
			wantCommentPosted:      true,
			wantLoggedError:        false,
		},
		{
			name:                   "returns error when configured",
			command:                "merge",
			err:                    errors.New("merge failed"),
			postErrorsAsPRComments: false,
			returnErrors:           true,
			wantErr:                true,
			wantCommentPosted:      false,
			wantLoggedError:        false,
		},
		{
			name:                   "CommentedError skips posting",
			command:                "lgtm",
			err:                    &handler.CommentedError{Err: errors.New("already commented")},
			postErrorsAsPRComments: true,
			returnErrors:           false,
			wantErr:                false,
			wantCommentPosted:      false,
			wantLoggedError:        false,
		},
		{
			name:                   "CommentedError returns error when configured",
			command:                "lgtm",
			err:                    &handler.CommentedError{Err: errors.New("already commented")},
			postErrorsAsPRComments: true,
			returnErrors:           true,
			wantErr:                true,
			wantCommentPosted:      false,
			wantLoggedError:        false,
		},
		{
			name:                   "logs error when neither post nor return",
			command:                "approve",
			err:                    errors.New("approve failed"),
			postErrorsAsPRComments: false,
			returnErrors:           false,
			wantErr:                false,
			wantCommentPosted:      false,
			wantLoggedError:        true,
		},
		{
			name:                   "handles PostComment error with returnErrors=true",
			command:                "rebase",
			err:                    errors.New("rebase failed"),
			postErrorsAsPRComments: true,
			returnErrors:           true,
			postCommentErr:         errors.New("failed to post comment"),
			wantErr:                true,
			wantCommentPosted:      true,
			wantLoggedError:        false,
		},
		{
			name:                   "handles PostComment error with returnErrors=false",
			command:                "rebase",
			err:                    errors.New("rebase failed"),
			postErrorsAsPRComments: true,
			returnErrors:           false,
			postCommentErr:         errors.New("failed to post comment"),
			wantErr:                false,
			wantCommentPosted:      true,
			wantLoggedError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commentPosted := false
			var postedComment string

			mockPRHandler := &MockPRHandler{
				PostCommentFunc: func(body string) error {
					commentPosted = true
					postedComment = body
					return tt.postCommentErr
				},
			}

			mockLogger := &MockLogger{}

			ctx := &ExecutionContext{
				PRHandler: mockPRHandler,
				Logger:    mockLogger,
				Config: &ExecutionConfig{
					PostErrorsAsPRComments: tt.postErrorsAsPRComments,
					ReturnErrors:           tt.returnErrors,
				},
			}

			handler := NewResultHandler(ctx)
			err := handler.HandleSingleCommandError(tt.command, tt.err)

			// Check error return
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleSingleCommandError() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check comment posting
			if commentPosted != tt.wantCommentPosted {
				t.Errorf("comment posted = %v, want %v", commentPosted, tt.wantCommentPosted)
			}

			// Verify comment content if posted
			if commentPosted && !strings.Contains(postedComment, tt.command) {
				t.Errorf("posted comment doesn't contain command name: %s", postedComment)
			}

			// Check error logging
			if tt.wantLoggedError && len(mockLogger.ErrorMessages) == 0 {
				t.Errorf("expected error to be logged, but no error messages found")
			}
		})
	}
}

func TestResultHandler_HandleMultiCommandResults(t *testing.T) {
	tests := []struct {
		name              string
		results           []SubCommandResult
		wantErr           bool
		wantCommentPosted bool
		expectWarning     bool
	}{
		{
			name: "all successful commands",
			results: []SubCommandResult{
				{Command: "rebase", Args: []string{}, Success: true, Error: nil},
				{Command: "lgtm", Args: []string{}, Success: true, Error: nil},
				{Command: "merge", Args: []string{}, Success: true, Error: nil},
			},
			wantErr:           false,
			wantCommentPosted: true,
			expectWarning:     false,
		},
		{
			name: "partial failures",
			results: []SubCommandResult{
				{Command: "rebase", Args: []string{}, Success: true, Error: nil},
				{Command: "lgtm", Args: []string{}, Success: false, Error: errors.New("validation failed")},
				{Command: "merge", Args: []string{}, Success: true, Error: nil},
			},
			wantErr:           false,
			wantCommentPosted: true,
			expectWarning:     true,
		},
		{
			name: "all failed commands",
			results: []SubCommandResult{
				{Command: "rebase", Args: []string{}, Success: false, Error: errors.New("rebase failed")},
				{Command: "merge", Args: []string{}, Success: false, Error: errors.New("merge failed")},
			},
			wantErr:           false,
			wantCommentPosted: true,
			expectWarning:     true,
		},
		{
			name:              "empty results",
			results:           []SubCommandResult{},
			wantErr:           false,
			wantCommentPosted: true,
			expectWarning:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var postedComment string
			commentPosted := false

			mockPRHandler := &MockPRHandler{
				PostCommentFunc: func(body string) error {
					commentPosted = true
					postedComment = body
					return nil
				},
			}

			mockLogger := &MockLogger{}

			ctx := &ExecutionContext{
				PRHandler: mockPRHandler,
				Logger:    mockLogger,
				Config:    &ExecutionConfig{},
			}

			handler := NewResultHandler(ctx)
			err := handler.HandleMultiCommandResults(tt.results)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleMultiCommandResults() error = %v, wantErr %v", err, tt.wantErr)
			}

			if commentPosted != tt.wantCommentPosted {
				t.Errorf("comment posted = %v, want %v", commentPosted, tt.wantCommentPosted)
			}

			if commentPosted {
				// Check for warning in header
				hasWarning := strings.Contains(postedComment, "⚠️")
				if hasWarning != tt.expectWarning {
					t.Errorf("comment has warning = %v, want %v. Comment: %s", hasWarning, tt.expectWarning, postedComment)
				}

				// Check for header
				if !strings.Contains(postedComment, "Multi-Command Execution Results") {
					t.Errorf("comment missing header. Comment: %s", postedComment)
				}

				// Check that all results are included
				for _, result := range tt.results {
					if !strings.Contains(postedComment, result.Command) {
						t.Errorf("comment missing command %s. Comment: %s", result.Command, postedComment)
					}
				}
			}
		})
	}
}

func TestResultHandler_FormatSubCommandResult(t *testing.T) {
	tests := []struct {
		name           string
		result         SubCommandResult
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "success with no args",
			result: SubCommandResult{
				Command: "rebase",
				Args:    []string{},
				Success: true,
				Error:   nil,
			},
			wantContains:   []string{"✅", "rebase", "executed successfully"},
			wantNotContain: []string{"❌", "failed"},
		},
		{
			name: "success with args",
			result: SubCommandResult{
				Command: "cherry-pick",
				Args:    []string{"abc123", "def456"},
				Success: true,
				Error:   nil,
			},
			wantContains:   []string{"✅", "cherry-pick", "abc123", "def456", "executed successfully"},
			wantNotContain: []string{"❌", "failed"},
		},
		{
			name: "failure with error message",
			result: SubCommandResult{
				Command: "merge",
				Args:    []string{},
				Success: false,
				Error:   errors.New("merge conflict detected"),
			},
			wantContains:   []string{"❌", "merge", "failed", "merge conflict detected"},
			wantNotContain: []string{"✅", "executed successfully"},
		},
		{
			name: "failure with args",
			result: SubCommandResult{
				Command: "lgtm",
				Args:    []string{"approve"},
				Success: false,
				Error:   fmt.Errorf("permission denied"),
			},
			wantContains:   []string{"❌", "lgtm", "approve", "failed", "permission denied"},
			wantNotContain: []string{"✅", "executed successfully"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ExecutionContext{
				PRHandler: &MockPRHandler{},
				Logger:    &MockLogger{},
				Config:    &ExecutionConfig{},
			}

			handler := NewResultHandler(ctx)
			formatted := handler.FormatSubCommandResult(tt.result)

			for _, want := range tt.wantContains {
				if !strings.Contains(formatted, want) {
					t.Errorf("FormatSubCommandResult() = %v, want to contain %v", formatted, want)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(formatted, notWant) {
					t.Errorf("FormatSubCommandResult() = %v, should not contain %v", formatted, notWant)
				}
			}
		})
	}
}
