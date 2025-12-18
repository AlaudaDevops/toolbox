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
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
)

func TestCommandExecutor_Execute(t *testing.T) {
	tests := []struct {
		name        string
		parsedCmd   *ParsedCommand
		setupMock   func(*MockPRHandler)
		wantSuccess bool
		wantErr     bool
	}{
		{
			name: "single command success",
			parsedCmd: &ParsedCommand{
				Type:    SingleCommand,
				Command: "rebase",
				Args:    []string{},
			},
			setupMock: func(m *MockPRHandler) {
				m.CheckPRStatusFunc = func(expectedStatus string) error {
					return nil
				}
				m.ExecuteCommandFunc = func(command string, args []string) error {
					return nil
				}
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name: "single command failure",
			parsedCmd: &ParsedCommand{
				Type:    SingleCommand,
				Command: "merge",
				Args:    []string{},
			},
			setupMock: func(m *MockPRHandler) {
				m.CheckPRStatusFunc = func(expectedStatus string) error {
					return nil
				}
				m.ExecuteCommandFunc = func(command string, args []string) error {
					return errors.New("merge failed")
				}
				m.PostCommentFunc = func(body string) error {
					return nil
				}
			},
			wantSuccess: false,
			wantErr:     true,
		},
		{
			name: "multi command success",
			parsedCmd: &ParsedCommand{
				Type:         MultiCommand,
				CommandLines: []string{"/rebase", "/lgtm"},
			},
			setupMock: func(m *MockPRHandler) {
				m.CheckPRStatusFunc = func(expectedStatus string) error {
					return nil
				}
				m.ExecuteCommandFunc = func(command string, args []string) error {
					return nil
				}
				m.PostCommentFunc = func(body string) error {
					return nil
				}
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name: "built-in command",
			parsedCmd: &ParsedCommand{
				Type:    BuiltInCommand,
				Command: "help",
				Args:    []string{},
			},
			setupMock: func(m *MockPRHandler) {
				m.ExecuteCommandFunc = func(command string, args []string) error {
					return nil
				}
			},
			wantSuccess: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPRHandler := &MockPRHandler{}
			tt.setupMock(mockPRHandler)

			config := NewWebhookExecutionConfig()
			config.ValidateCommentSender = false // Disable for simplicity in tests
			config.ReturnErrors = true           // Enable to see errors in tests

			ctx := &ExecutionContext{
				PRHandler:       mockPRHandler,
				Logger:          &MockLogger{},
				Config:          config,
				MetricsRecorder: &MockMetricsRecorder{},
				Platform:        "github",
			}

			executor := NewCommandExecutor(ctx)
			result, err := executor.Execute(tt.parsedCmd)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result != nil && result.Success != tt.wantSuccess {
				t.Errorf("Execute() success = %v, want %v", result.Success, tt.wantSuccess)
			}
		})
	}
}

func TestCommandExecutor_ExecuteSingleCommand(t *testing.T) {
	tests := []struct {
		name            string
		command         string
		args            []string
		setupMock       func(*MockPRHandler)
		config          *ExecutionConfig
		wantSuccess     bool
		wantErr         bool
		wantMetricsCall bool
		expectedStatus  string
	}{
		{
			name:    "successful execution",
			command: "rebase",
			args:    []string{},
			setupMock: func(m *MockPRHandler) {
				m.CheckPRStatusFunc = func(expectedStatus string) error {
					return nil
				}
				m.ExecuteCommandFunc = func(command string, args []string) error {
					return nil
				}
			},
			config: &ExecutionConfig{
				ValidateCommentSender: false,
				ValidatePRStatus:      false,
			},
			wantSuccess:     true,
			wantErr:         false,
			wantMetricsCall: true,
			expectedStatus:  "success",
		},
		{
			name:    "validation failure",
			command: "merge",
			args:    []string{},
			setupMock: func(m *MockPRHandler) {
				m.CheckPRStatusFunc = func(expectedStatus string) error {
					return errors.New("PR is closed")
				}
			},
			config: &ExecutionConfig{
				ValidatePRStatus: true,
			},
			wantSuccess:     false,
			wantErr:         true,
			wantMetricsCall: true,
			expectedStatus:  "validation_failed",
		},
		{
			name:    "execution failure",
			command: "lgtm",
			args:    []string{},
			setupMock: func(m *MockPRHandler) {
				m.CheckPRStatusFunc = func(expectedStatus string) error {
					return nil
				}
				m.ExecuteCommandFunc = func(command string, args []string) error {
					return errors.New("execution failed")
				}
				m.PostCommentFunc = func(body string) error {
					return nil
				}
			},
			config: &ExecutionConfig{
				ValidatePRStatus:       true,
				PostErrorsAsPRComments: true,
				ReturnErrors:           true,
			},
			wantSuccess:     false,
			wantErr:         true,
			wantMetricsCall: true,
			expectedStatus:  "failure",
		},
		{
			name:    "metrics recording",
			command: "approve",
			args:    []string{"test"},
			setupMock: func(m *MockPRHandler) {
				m.ExecuteCommandFunc = func(command string, args []string) error {
					return nil
				}
			},
			config: &ExecutionConfig{
				ValidateCommentSender: false,
				ValidatePRStatus:      false,
			},
			wantSuccess:     true,
			wantErr:         false,
			wantMetricsCall: true,
			expectedStatus:  "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPRHandler := &MockPRHandler{}
			tt.setupMock(mockPRHandler)

			metricsRecorder := &MockMetricsRecorder{}

			ctx := &ExecutionContext{
				PRHandler:       mockPRHandler,
				Logger:          &MockLogger{},
				Config:          tt.config,
				MetricsRecorder: metricsRecorder,
				Platform:        "github",
			}

			executor := NewCommandExecutor(ctx)
			result, err := executor.ExecuteSingleCommand(tt.command, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteSingleCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("ExecuteSingleCommand() success = %v, want %v", result.Success, tt.wantSuccess)
			}

			if tt.wantMetricsCall {
				if metricsRecorder.GetCommandExecutionCount() == 0 {
					t.Errorf("Expected metrics to be recorded, but no calls found")
				}

				lastCall := metricsRecorder.GetLastCommandExecution()
				if lastCall != nil && lastCall.Status != tt.expectedStatus {
					t.Errorf("Expected status %v, got %v", tt.expectedStatus, lastCall.Status)
				}
			}
		})
	}
}

func TestCommandExecutor_ExecuteMultiCommand(t *testing.T) {
	tests := []struct {
		name               string
		commandLines       []string
		setupMock          func(*MockPRHandler)
		config             *ExecutionConfig
		wantAllSuccess     bool
		wantErr            bool
		expectedResultsLen int
		expectSummary      bool
	}{
		{
			name:         "all commands succeed",
			commandLines: []string{"/rebase", "/lgtm", "/merge"},
			setupMock: func(m *MockPRHandler) {
				m.ExecuteCommandFunc = func(command string, args []string) error {
					return nil
				}
				m.PostCommentFunc = func(body string) error {
					return nil
				}
			},
			config: &ExecutionConfig{
				ValidateCommentSender: false,
				ValidatePRStatus:      false,
			},
			wantAllSuccess:     true,
			wantErr:            false,
			expectedResultsLen: 3,
			expectSummary:      true,
		},
		{
			name:         "continue on error",
			commandLines: []string{"/rebase", "/lgtm"},
			setupMock: func(m *MockPRHandler) {
				callCount := 0
				m.ExecuteCommandFunc = func(command string, args []string) error {
					callCount++
					if callCount == 2 {
						return errors.New("lgtm failed")
					}
					return nil
				}
				m.PostCommentFunc = func(body string) error {
					return nil
				}
			},
			config: &ExecutionConfig{
				StopOnFirstError:      false,
				ValidateCommentSender: false,
				ValidatePRStatus:      false,
			},
			wantAllSuccess:     false,
			wantErr:            false,
			expectedResultsLen: 2,
			expectSummary:      true,
		},
		{
			name:         "stop on first error",
			commandLines: []string{"/rebase", "/lgtm", "/merge"},
			setupMock: func(m *MockPRHandler) {
				callCount := 0
				m.ExecuteCommandFunc = func(command string, args []string) error {
					callCount++
					if callCount == 2 {
						return errors.New("lgtm failed")
					}
					return nil
				}
				m.PostCommentFunc = func(body string) error {
					return nil
				}
			},
			config: &ExecutionConfig{
				StopOnFirstError:      true,
				ValidateCommentSender: false,
				ValidatePRStatus:      false,
			},
			wantAllSuccess:     false,
			wantErr:            false,
			expectedResultsLen: 2,
			expectSummary:      true,
		},
		{
			name:         "summary posting",
			commandLines: []string{"/rebase"},
			setupMock: func(m *MockPRHandler) {
				m.ExecuteCommandFunc = func(command string, args []string) error {
					return nil
				}
				m.PostCommentFunc = func(body string) error {
					return nil
				}
			},
			config: &ExecutionConfig{
				ValidateCommentSender: false,
				ValidatePRStatus:      false,
			},
			wantAllSuccess:     true,
			wantErr:            false,
			expectedResultsLen: 1,
			expectSummary:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPRHandler := &MockPRHandler{}
			tt.setupMock(mockPRHandler)

			ctx := &ExecutionContext{
				PRHandler:       mockPRHandler,
				Logger:          &MockLogger{},
				Config:          tt.config,
				MetricsRecorder: &MockMetricsRecorder{},
				Platform:        "github",
			}

			executor := NewCommandExecutor(ctx)
			result, err := executor.ExecuteMultiCommand(tt.commandLines, tt.commandLines)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteMultiCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result.Results) != tt.expectedResultsLen {
				t.Errorf("ExecuteMultiCommand() results length = %v, want %v", len(result.Results), tt.expectedResultsLen)
			}

			if tt.wantAllSuccess {
				for i, r := range result.Results {
					if !r.Success {
						t.Errorf("Result %d should be successful but was not", i)
					}
				}
			}
		})
	}
}

func TestCommandExecutor_StopOnFirstError(t *testing.T) {
	commandLines := []string{"/rebase", "/lgtm", "/merge"}

	tests := []struct {
		name           string
		stopOnFirst    bool
		failAtCommand  int
		expectedCalls  int
		expectedLength int
	}{
		{
			name:           "stop on first error - fail at command 2",
			stopOnFirst:    true,
			failAtCommand:  2,
			expectedCalls:  2,
			expectedLength: 2,
		},
		{
			name:           "continue on error - fail at command 2",
			stopOnFirst:    false,
			failAtCommand:  2,
			expectedCalls:  3,
			expectedLength: 3,
		},
		{
			name:           "stop on first error - fail at command 1",
			stopOnFirst:    true,
			failAtCommand:  1,
			expectedCalls:  1,
			expectedLength: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			mockPRHandler := &MockPRHandler{
				ExecuteCommandFunc: func(command string, args []string) error {
					callCount++
					if callCount == tt.failAtCommand {
						return errors.New("command failed")
					}
					return nil
				},
				PostCommentFunc: func(body string) error {
					return nil
				},
			}

			ctx := &ExecutionContext{
				PRHandler: mockPRHandler,
				Logger:    &MockLogger{},
				Config: &ExecutionConfig{
					StopOnFirstError:      tt.stopOnFirst,
					ValidateCommentSender: false,
					ValidatePRStatus:      false,
				},
				MetricsRecorder: &MockMetricsRecorder{},
				Platform:        "github",
			}

			executor := NewCommandExecutor(ctx)
			result, err := executor.ExecuteMultiCommand(commandLines, commandLines)

			if err != nil {
				t.Errorf("ExecuteMultiCommand() unexpected error: %v", err)
			}

			if callCount != tt.expectedCalls {
				t.Errorf("Expected %d command calls, got %d", tt.expectedCalls, callCount)
			}

			if len(result.Results) != tt.expectedLength {
				t.Errorf("Expected %d results, got %d", tt.expectedLength, len(result.Results))
			}

			// Verify that execution stopped at the right point
			if tt.stopOnFirst {
				hasFailure := false
				for i, r := range result.Results {
					if !r.Success {
						hasFailure = true
						if i != tt.failAtCommand-1 {
							t.Errorf("Expected failure at index %d, got failure at %d", tt.failAtCommand-1, i)
						}
					}
				}
				if !hasFailure {
					t.Errorf("Expected at least one failure")
				}
			}
		})
	}
}

func TestCommandExecutor_ExecuteBuiltInCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		executeErr  error
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:        "help command success",
			command:     "help",
			args:        []string{},
			executeErr:  nil,
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:        "version command success",
			command:     "version",
			args:        []string{},
			executeErr:  nil,
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:        "built-in command failure",
			command:     "help",
			args:        []string{},
			executeErr:  errors.New("command failed"),
			wantSuccess: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPRHandler := &MockPRHandler{
				ExecuteCommandFunc: func(command string, args []string) error {
					return tt.executeErr
				},
			}

			metricsRecorder := &MockMetricsRecorder{}

			config := NewWebhookExecutionConfig()
			config.ValidateCommentSender = false

			ctx := &ExecutionContext{
				PRHandler:       mockPRHandler,
				Logger:          &MockLogger{},
				Config:          config,
				MetricsRecorder: metricsRecorder,
				Platform:        "github",
			}

			executor := NewCommandExecutor(ctx)
			result, err := executor.ExecuteBuiltInCommand(tt.command, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteBuiltInCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("ExecuteBuiltInCommand() success = %v, want %v", result.Success, tt.wantSuccess)
			}

			// Verify metrics were recorded
			if metricsRecorder.GetCommandExecutionCount() == 0 {
				t.Errorf("Expected metrics to be recorded")
			}
		})
	}
}

func TestCommandExecutor_MetricsRecording(t *testing.T) {
	mockPRHandler := &MockPRHandler{
		CheckPRStatusFunc: func(expectedStatus string) error {
			return nil
		},
		ExecuteCommandFunc: func(command string, args []string) error {
			return nil
		},
		GetCommentsWithCacheFunc: func() ([]git.Comment, error) {
			return []git.Comment{
				{Body: "test", User: git.User{Login: "test-user"}},
			}, nil
		},
	}

	metricsRecorder := &MockMetricsRecorder{}

	config := NewWebhookExecutionConfig()
	config.ValidateCommentSender = false // Disable comment sender validation for this test

	ctx := &ExecutionContext{
		PRHandler:       mockPRHandler,
		Logger:          &MockLogger{},
		Config:          config,
		MetricsRecorder: metricsRecorder,
		Platform:        "github",
		CommentSender:   "test-user",
	}

	executor := NewCommandExecutor(ctx)

	// Test single command metrics
	_, err := executor.ExecuteSingleCommand("rebase", []string{})
	if err != nil {
		t.Fatalf("ExecuteSingleCommand() unexpected error: %v", err)
	}

	if metricsRecorder.GetCommandExecutionCount() != 1 {
		t.Errorf("Expected 1 command execution metric, got %d", metricsRecorder.GetCommandExecutionCount())
	}

	if metricsRecorder.GetProcessingDurationCount() != 1 {
		t.Errorf("Expected 1 processing duration metric, got %d", metricsRecorder.GetProcessingDurationCount())
	}

	lastExec := metricsRecorder.GetLastCommandExecution()
	if lastExec == nil {
		t.Fatal("Expected last command execution to be recorded")
	}

	if lastExec.Platform != "github" {
		t.Errorf("Expected platform 'github', got '%s'", lastExec.Platform)
	}

	if lastExec.Command != "rebase" {
		t.Errorf("Expected command 'rebase', got '%s'", lastExec.Command)
	}

	if lastExec.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", lastExec.Status)
	}
}
