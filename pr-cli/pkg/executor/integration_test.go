package executor

import (
	"errors"
	"testing"
	"time"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestIntegration_CLIModeExecution tests full CLI mode flow with mock PRHandler
func TestIntegration_CLIModeExecution(t *testing.T) {
	t.Run("single command with validation", func(t *testing.T) {
		mockPR := &MockPRHandler{
			CheckPRStatusFunc: func(expectedStatus string) error {
				return nil // Open PR
			},
		}
		mockMetrics := &MockMetricsRecorder{}

		ctx := &ExecutionContext{
			PRHandler:       mockPR,
			Logger:          logrus.New(),
			Config:          NewCLIExecutionConfig(false),
			MetricsRecorder: mockMetrics,
			Platform:        "github",
			CommentSender:   "user1",
		}

		executor := NewCommandExecutor(ctx)
		parsedCmd := &ParsedCommand{
			Type:    SingleCommand,
			Command: "rebase",
			Args:    []string{},
		}

		result, err := executor.Execute(parsedCmd)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Equal(t, SingleCommand, result.CommandType)
		assert.Equal(t, 1, mockMetrics.GetCommandExecutionCount())
		lastExec := mockMetrics.GetLastCommandExecution()
		assert.NotNil(t, lastExec)
		assert.Equal(t, "success", lastExec.Status)
	})

	t.Run("multi-command with summary posting", func(t *testing.T) {
		commentPosted := false
		lastComment := ""
		mockPR := &MockPRHandler{
			CheckPRStatusFunc: func(expectedStatus string) error {
				return nil // Open PR
			},
			PostCommentFunc: func(body string) error {
				commentPosted = true
				lastComment = body
				return nil
			},
		}
		mockMetrics := &MockMetricsRecorder{}

		ctx := &ExecutionContext{
			PRHandler:       mockPR,
			Logger:          logrus.New(),
			Config:          NewCLIExecutionConfig(false),
			MetricsRecorder: mockMetrics,
			Platform:        "github",
			CommentSender:   "user1",
		}

		executor := NewCommandExecutor(ctx)
		parsedCmd := &ParsedCommand{
			Type:         MultiCommand,
			CommandLines: []string{"/rebase", "/lgtm"},
			RawCommandLines: []string{"/rebase", "/lgtm"},
		}

		result, err := executor.Execute(parsedCmd)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Equal(t, MultiCommand, result.CommandType)
		assert.Len(t, result.Results, 2)
		assert.True(t, commentPosted)
		assert.Contains(t, lastComment, "✅")
		// Multi-command records metrics for each sub-command + overall multi-command
		assert.Equal(t, 3, mockMetrics.GetCommandExecutionCount())
	})

	t.Run("error handling with PR comment", func(t *testing.T) {
		commentPosted := false
		lastComment := ""
		mockPR := &MockPRHandler{
			CheckPRStatusFunc: func(expectedStatus string) error {
				return nil // Open PR
			},
			ExecuteCommandFunc: func(command string, args []string) error {
				return errors.New("execution failed")
			},
			PostCommentFunc: func(body string) error {
				commentPosted = true
				lastComment = body
				return nil
			},
		}
		mockMetrics := &MockMetricsRecorder{}

		ctx := &ExecutionContext{
			PRHandler:       mockPR,
			Logger:          logrus.New(),
			Config:          NewCLIExecutionConfig(false),
			MetricsRecorder: mockMetrics,
			Platform:        "github",
			CommentSender:   "user1",
		}

		executor := NewCommandExecutor(ctx)
		parsedCmd := &ParsedCommand{
			Type:    SingleCommand,
			Command: "rebase",
			Args:    []string{},
		}

		result, err := executor.Execute(parsedCmd)

		// CLI mode posts error and returns it
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Success)
		assert.True(t, commentPosted)
		assert.Contains(t, lastComment, "execution failed")
		lastExec := mockMetrics.GetLastCommandExecution()
		assert.NotNil(t, lastExec)
		assert.Equal(t, "failure", lastExec.Status)
	})

	t.Run("debug mode behavior", func(t *testing.T) {
		mockPR := &MockPRHandler{
			CheckPRStatusFunc: func(expectedStatus string) error {
				return nil // Open PR
			},
		}
		mockMetrics := &MockMetricsRecorder{}

		ctx := &ExecutionContext{
			PRHandler:       mockPR,
			Logger:          logrus.New(),
			Config:          NewCLIExecutionConfig(true), // Debug mode
			MetricsRecorder: mockMetrics,
			Platform:        "github",
			CommentSender:   "unknown-user",
		}

		executor := NewCommandExecutor(ctx)
		parsedCmd := &ParsedCommand{
			Type:    SingleCommand,
			Command: "rebase",
			Args:    []string{},
		}

		result, err := executor.Execute(parsedCmd)

		// Debug mode skips comment sender validation
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Equal(t, 1, mockMetrics.GetCommandExecutionCount())
	})
}

// TestIntegration_WebhookModeExecution tests full webhook mode flow with mock PRHandler
func TestIntegration_WebhookModeExecution(t *testing.T) {
	t.Run("single command without validation", func(t *testing.T) {
		mockPR := &MockPRHandler{
			CheckPRStatusFunc: func(expectedStatus string) error {
				return nil // Open PR
			},
			GetCommentsWithCacheFunc: func() ([]git.Comment, error) {
				return []git.Comment{
					{User: git.User{Login: "user1"}, Body: "/rebase"},
				}, nil
			},
		}
		mockMetrics := &MockMetricsRecorder{}

		ctx := &ExecutionContext{
			PRHandler:       mockPR,
			Logger:          logrus.New(),
			Config:          NewWebhookExecutionConfig(),
			MetricsRecorder: mockMetrics,
			Platform:        "github",
			CommentSender:   "user1",
			TriggerComment:  "/rebase",
		}

		executor := NewCommandExecutor(ctx)
		parsedCmd := &ParsedCommand{
			Type:    SingleCommand,
			Command: "rebase",
			Args:    []string{},
		}

		result, err := executor.Execute(parsedCmd)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Equal(t, SingleCommand, result.CommandType)
		assert.Equal(t, 1, mockMetrics.GetCommandExecutionCount())
		lastExec := mockMetrics.GetLastCommandExecution()
		assert.NotNil(t, lastExec)
		assert.Equal(t, "success", lastExec.Status)
	})

	t.Run("multi-command with metrics", func(t *testing.T) {
		mockPR := &MockPRHandler{
			CheckPRStatusFunc: func(expectedStatus string) error {
				return nil // Open PR
			},
			GetCommentsWithCacheFunc: func() ([]git.Comment, error) {
				return []git.Comment{
					{User: git.User{Login: "user1"}, Body: "/rebase\n/lgtm\n/merge"},
				}, nil
			},
		}
		mockMetrics := &MockMetricsRecorder{}

		ctx := &ExecutionContext{
			PRHandler:       mockPR,
			Logger:          logrus.New(),
			Config:          NewWebhookExecutionConfig(),
			MetricsRecorder: mockMetrics,
			Platform:        "github",
			CommentSender:   "user1",
			TriggerComment:  "/rebase\n/lgtm\n/merge",
		}

		executor := NewCommandExecutor(ctx)
		parsedCmd := &ParsedCommand{
			Type:         MultiCommand,
			CommandLines: []string{"/rebase", "/lgtm", "/merge"},
			RawCommandLines: []string{"/rebase", "/lgtm", "/merge"},
		}

		result, err := executor.Execute(parsedCmd)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Equal(t, MultiCommand, result.CommandType)
		assert.Len(t, result.Results, 3)
		// Multi-command records metrics for each sub-command + overall multi-command
		assert.Equal(t, 4, mockMetrics.GetCommandExecutionCount())
		assert.Equal(t, 4, mockMetrics.GetProcessingDurationCount())
	})

	t.Run("error handling without PR comment", func(t *testing.T) {
		commentPosted := false
		mockPR := &MockPRHandler{
			ExecuteCommandFunc: func(command string, args []string) error {
				return errors.New("webhook execution failed")
			},
			PostCommentFunc: func(body string) error {
				commentPosted = true
				return nil
			},
			CheckPRStatusFunc: func(expectedStatus string) error {
				return nil // Open PR
			},
			GetCommentsWithCacheFunc: func() ([]git.Comment, error) {
				return []git.Comment{
					{User: git.User{Login: "user1"}, Body: "/rebase"},
				}, nil
			},
		}
		mockMetrics := &MockMetricsRecorder{}

		ctx := &ExecutionContext{
			PRHandler:       mockPR,
			Logger:          logrus.New(),
			Config:          NewWebhookExecutionConfig(),
			MetricsRecorder: mockMetrics,
			Platform:        "github",
			CommentSender:   "user1",
			TriggerComment:  "/rebase",
		}

		executor := NewCommandExecutor(ctx)
		parsedCmd := &ParsedCommand{
			Type:    SingleCommand,
			Command: "rebase",
			Args:    []string{},
		}

		result, err := executor.Execute(parsedCmd)

		// Webhook mode posts errors and doesn't return them
		// The result Success is true because the error was successfully handled (posted)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success) // Success means error was handled properly
		assert.True(t, commentPosted) // Webhook posts errors as comments
		lastExec := mockMetrics.GetLastCommandExecution()
		assert.NotNil(t, lastExec)
		assert.Equal(t, "failure", lastExec.Status) // But metrics show failure
	})
}

// TestIntegration_MetricsRecording tests metrics recording
func TestIntegration_MetricsRecording(t *testing.T) {
	t.Run("metrics recorded correctly for success", func(t *testing.T) {
		mockPR := &MockPRHandler{
			CheckPRStatusFunc: func(expectedStatus string) error {
				return nil // Open PR
			},
			GetCommentsWithCacheFunc: func() ([]git.Comment, error) {
				return []git.Comment{
					{User: git.User{Login: "user1"}, Body: "/rebase"},
				}, nil
			},
		}
		mockMetrics := &MockMetricsRecorder{}

		ctx := &ExecutionContext{
			PRHandler:       mockPR,
			Logger:          logrus.New(),
			Config:          NewWebhookExecutionConfig(),
			MetricsRecorder: mockMetrics,
			Platform:        "github",
			CommentSender:   "user1",
			TriggerComment:  "/rebase",
		}

		executor := NewCommandExecutor(ctx)
		parsedCmd := &ParsedCommand{
			Type:    SingleCommand,
			Command: "rebase",
			Args:    []string{},
		}

		_, err := executor.Execute(parsedCmd)

		assert.NoError(t, err)
		assert.Equal(t, 1, mockMetrics.GetCommandExecutionCount())
		lastExec := mockMetrics.GetLastCommandExecution()
		assert.NotNil(t, lastExec)
		assert.Equal(t, "github", lastExec.Platform)
		assert.Equal(t, "rebase", lastExec.Command)
		assert.Equal(t, "success", lastExec.Status)
		assert.Equal(t, 1, mockMetrics.GetProcessingDurationCount())
		assert.True(t, mockMetrics.ProcessingDurationCalls[0].Duration > 0)
	})

	t.Run("metrics recorded correctly for failure", func(t *testing.T) {
		mockPR := &MockPRHandler{
			ExecuteCommandFunc: func(command string, args []string) error {
				return errors.New("command failed")
			},
			CheckPRStatusFunc: func(expectedStatus string) error {
				return nil // Open PR
			},
			GetCommentsWithCacheFunc: func() ([]git.Comment, error) {
				return []git.Comment{
					{User: git.User{Login: "user1"}, Body: "/merge --squash"},
				}, nil
			},
		}
		mockMetrics := &MockMetricsRecorder{}

		ctx := &ExecutionContext{
			PRHandler:       mockPR,
			Logger:          logrus.New(),
			Config:          NewWebhookExecutionConfig(),
			MetricsRecorder: mockMetrics,
			Platform:        "gitlab",
			CommentSender:   "user1",
			TriggerComment:  "/merge --squash",
		}

		executor := NewCommandExecutor(ctx)
		parsedCmd := &ParsedCommand{
			Type:    SingleCommand,
			Command: "merge",
			Args:    []string{"--squash"},
		}

		_, err := executor.Execute(parsedCmd)

		// Webhook mode posts errors and doesn't return them
		assert.NoError(t, err)
		assert.Equal(t, 1, mockMetrics.GetCommandExecutionCount())
		lastExec := mockMetrics.GetLastCommandExecution()
		assert.NotNil(t, lastExec)
		assert.Equal(t, "gitlab", lastExec.Platform)
		assert.Equal(t, "merge", lastExec.Command)
		assert.Equal(t, "failure", lastExec.Status)
		assert.Equal(t, 1, mockMetrics.GetProcessingDurationCount())
		assert.True(t, mockMetrics.ProcessingDurationCalls[0].Duration > 0)
	})

	t.Run("duration recording", func(t *testing.T) {
		mockPR := &MockPRHandler{
			ExecuteCommandFunc: func(cmd string, args []string) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			CheckPRStatusFunc: func(expectedStatus string) error {
				return nil // Open PR
			},
			GetCommentsWithCacheFunc: func() ([]git.Comment, error) {
				return []git.Comment{
					{User: git.User{Login: "user1"}, Body: "/rebase"},
				}, nil
			},
		}
		mockMetrics := &MockMetricsRecorder{}

		ctx := &ExecutionContext{
			PRHandler:       mockPR,
			Logger:          logrus.New(),
			Config:          NewWebhookExecutionConfig(),
			MetricsRecorder: mockMetrics,
			Platform:        "github",
			CommentSender:   "user1",
			TriggerComment:  "/rebase",
		}

		executor := NewCommandExecutor(ctx)
		parsedCmd := &ParsedCommand{
			Type:    SingleCommand,
			Command: "rebase",
			Args:    []string{},
		}

		_, err := executor.Execute(parsedCmd)

		assert.NoError(t, err)
		assert.Equal(t, 1, mockMetrics.GetProcessingDurationCount())
		assert.True(t, mockMetrics.ProcessingDurationCalls[0].Duration >= 10*time.Millisecond)
	})
}
