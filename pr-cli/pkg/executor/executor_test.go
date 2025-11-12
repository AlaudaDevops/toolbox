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
	"time"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGitClient is a mock implementation of git.GitClient
type MockGitClient struct {
	mock.Mock
}

func (m *MockGitClient) GetPR() (*git.PullRequest, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*git.PullRequest), args.Error(1)
}

func (m *MockGitClient) GetComments() ([]git.Comment, error) {
	args := m.Called()
	return args.Get(0).([]git.Comment), args.Error(1)
}

func (m *MockGitClient) PostComment(body string) error {
	args := m.Called(body)
	return args.Error(0)
}

func (m *MockGitClient) Merge(method string) error {
	args := m.Called(method)
	return args.Error(0)
}

func (m *MockGitClient) GetReviews() ([]git.Review, error) {
	args := m.Called()
	return args.Get(0).([]git.Review), args.Error(1)
}

func (m *MockGitClient) GetCombinedStatus() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) AddLabel(label string) error {
	args := m.Called(label)
	return args.Error(0)
}

func (m *MockGitClient) RemoveLabel(label string) error {
	args := m.Called(label)
	return args.Error(0)
}

func (m *MockGitClient) AddAssignees(assignees []string) error {
	args := m.Called(assignees)
	return args.Error(0)
}

func (m *MockGitClient) RemoveAssignees(assignees []string) error {
	args := m.Called(assignees)
	return args.Error(0)
}

func (m *MockGitClient) ClosePR() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGitClient) CreatePR(title, body, head, base string) (int, error) {
	args := m.Called(title, body, head, base)
	return args.Int(0), args.Error(1)
}

func (m *MockGitClient) UpdatePR(title, body string) error {
	args := m.Called(title, body)
	return args.Error(0)
}

func (m *MockGitClient) Rebase() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGitClient) GetIssue(issueNumber int) (*git.Issue, error) {
	args := m.Called(issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*git.Issue), args.Error(1)
}

func (m *MockGitClient) UpdateIssue(issueNumber int, body string) error {
	args := m.Called(issueNumber, body)
	return args.Error(0)
}

// MockMetricsRecorder is a mock implementation of MetricsRecorder
type MockMetricsRecorder struct {
	CommandExecutions []struct {
		Platform string
		Command  string
		Status   string
	}
}

func (m *MockMetricsRecorder) RecordCommandExecution(platform, command, status string) {
	m.CommandExecutions = append(m.CommandExecutions, struct {
		Platform string
		Command  string
		Status   string
	}{platform, command, status})
}

func (m *MockMetricsRecorder) RecordProcessingDuration(platform, command string, duration time.Duration) {
	// No-op for testing
}

func createTestContext(t *testing.T, cfg *ExecutionConfig, mockClient *MockGitClient) *ExecutionContext {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	// Create a config
	prConfig := &config.Config{
		Platform:       "github",
		Owner:          "test-org",
		Repo:           "test-repo",
		PRNum:          1,
		CommentSender:  "test-user",
		TriggerComment: "/test",
	}

	// Create PR handler with mock client
	prHandler := &handler.PRHandler{}
	// We'll need to inject the mock client somehow - for now, skip actual handler operations

	metricsRecorder := &MockMetricsRecorder{}

	return &ExecutionContext{
		PRHandler:       prHandler,
		Logger:          logger,
		Config:          cfg,
		MetricsRecorder: metricsRecorder,
		Platform:        prConfig.Platform,
		CommentSender:   prConfig.CommentSender,
		TriggerComment:  prConfig.TriggerComment,
	}
}

func TestNewCommandExecutor(t *testing.T) {
	cfg := NewCLIExecutionConfig(false)
	ctx := &ExecutionContext{
		Logger:          logrus.New(),
		Config:          cfg,
		MetricsRecorder: &NoOpMetricsRecorder{},
		Platform:        "github",
		CommentSender:   "test-user",
		TriggerComment:  "/test",
	}

	executor := NewCommandExecutor(ctx)

	assert.NotNil(t, executor)
	assert.NotNil(t, executor.validator)
	assert.NotNil(t, executor.resultHandler)
	assert.Equal(t, ctx, executor.context)
}

// Note: Testing Execute methods requires properly mocked PRHandler
// which is complex. Instead, we test the individual components separately.

func TestExecute_UnknownCommandType(t *testing.T) {
	cfg := NewCLIExecutionConfig(false)
	ctx := &ExecutionContext{
		Logger:          logrus.New(),
		Config:          cfg,
		MetricsRecorder: &NoOpMetricsRecorder{},
		Platform:        "github",
		CommentSender:   "test-user",
		TriggerComment:  "/test",
	}

	executor := NewCommandExecutor(ctx)

	parsedCmd := &ParsedCommand{
		Type:    CommandType(999), // Invalid type
		Command: "test",
		Args:    []string{},
	}

	result, err := executor.Execute(parsedCmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unknown command type")
}

func TestNewCLIExecutionConfig(t *testing.T) {
	tests := []struct {
		name      string
		debugMode bool
		expected  *ExecutionConfig
	}{
		{
			name:      "CLI config with debug mode off",
			debugMode: false,
			expected: &ExecutionConfig{
				ValidateCommentSender:  true,
				ValidatePRStatus:       true,
				DebugMode:              false,
				PostErrorsAsPRComments: true,
				ReturnErrors:           false,
				StopOnFirstError:       false,
			},
		},
		{
			name:      "CLI config with debug mode on",
			debugMode: true,
			expected: &ExecutionConfig{
				ValidateCommentSender:  false,
				ValidatePRStatus:       true,
				DebugMode:              true,
				PostErrorsAsPRComments: true,
				ReturnErrors:           false,
				StopOnFirstError:       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewCLIExecutionConfig(tt.debugMode)
			assert.Equal(t, tt.expected, config)
		})
	}
}

func TestNewWebhookExecutionConfig(t *testing.T) {
	config := NewWebhookExecutionConfig()

	expected := &ExecutionConfig{
		ValidateCommentSender:  false,
		ValidatePRStatus:       false,
		DebugMode:              false,
		PostErrorsAsPRComments: false,
		ReturnErrors:           true,
		StopOnFirstError:       false,
	}

	assert.Equal(t, expected, config)
}

// Note: executeSubCommands requires a functioning PRHandler for full testing.
// These methods are tested indirectly through integration tests.
