package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewCLIExecutionConfig tests CLI configuration creation
func TestNewCLIExecutionConfig(t *testing.T) {
	t.Run("with debugMode false", func(t *testing.T) {
		config := NewCLIExecutionConfig(false)

		assert.NotNil(t, config)
		assert.False(t, config.ValidateCommentSender, "CLI doesn't validate comment sender")
		assert.True(t, config.ValidatePRStatus, "should validate PR status")
		assert.False(t, config.DebugMode, "debug mode should be false")
		assert.True(t, config.PostErrorsAsPRComments, "should post errors as PR comments")
		assert.True(t, config.ReturnErrors, "CLI returns errors for proper exit codes")
		assert.False(t, config.StopOnFirstError, "should continue on error")
	})

	t.Run("with debugMode true", func(t *testing.T) {
		config := NewCLIExecutionConfig(true)

		assert.NotNil(t, config)
		assert.False(t, config.ValidateCommentSender, "CLI doesn't validate comment sender")
		assert.True(t, config.ValidatePRStatus, "should validate PR status")
		assert.True(t, config.DebugMode, "debug mode should be true")
		assert.True(t, config.PostErrorsAsPRComments, "should post errors as PR comments")
		assert.True(t, config.ReturnErrors, "CLI returns errors for proper exit codes")
		assert.False(t, config.StopOnFirstError, "should continue on error")
	})
}

// TestNewWebhookExecutionConfig tests webhook configuration creation
func TestNewWebhookExecutionConfig(t *testing.T) {
	config := NewWebhookExecutionConfig()

	assert.NotNil(t, config)
	assert.True(t, config.ValidateCommentSender, "webhook validates comment sender")
	assert.True(t, config.ValidatePRStatus, "webhook validates PR status")
	assert.False(t, config.DebugMode, "debug mode should be false")
	assert.True(t, config.PostErrorsAsPRComments, "webhook posts errors as PR comments")
	assert.False(t, config.ReturnErrors, "webhook doesn't return errors (already posted)")
	assert.False(t, config.StopOnFirstError, "should continue on error")
}

// TestConfigurationCombinations tests all configuration combinations with all command types
func TestConfigurationCombinations(t *testing.T) {
	testCases := []struct {
		name           string
		configFunc     func() *ExecutionConfig
		commandType    CommandType
		expectedBehavior string
	}{
		// CLI default configuration
		{
			name:           "CLI default - single command",
			configFunc:     func() *ExecutionConfig { return NewCLIExecutionConfig(false) },
			commandType:    SingleCommand,
			expectedBehavior: "validate sender and PR status, post errors as comments, don't return errors",
		},
		{
			name:           "CLI default - multi command",
			configFunc:     func() *ExecutionConfig { return NewCLIExecutionConfig(false) },
			commandType:    MultiCommand,
			expectedBehavior: "validate sender and PR status, post summary as comment, don't return errors",
		},
		{
			name:           "CLI default - built-in",
			configFunc:     func() *ExecutionConfig { return NewCLIExecutionConfig(false) },
			commandType:    BuiltInCommand,
			expectedBehavior: "no validation, execute directly",
		},
		// CLI debug configuration
		{
			name:           "CLI debug - single command",
			configFunc:     func() *ExecutionConfig { return NewCLIExecutionConfig(true) },
			commandType:    SingleCommand,
			expectedBehavior: "skip sender validation in debug mode, validate PR status, post errors as comments",
		},
		{
			name:           "CLI debug - multi command",
			configFunc:     func() *ExecutionConfig { return NewCLIExecutionConfig(true) },
			commandType:    MultiCommand,
			expectedBehavior: "skip sender validation in debug mode, validate PR status, post summary",
		},
		{
			name:           "CLI debug - built-in",
			configFunc:     func() *ExecutionConfig { return NewCLIExecutionConfig(true) },
			commandType:    BuiltInCommand,
			expectedBehavior: "no validation, execute directly",
		},
		// Webhook configuration
		{
			name:           "Webhook - single command",
			configFunc:     NewWebhookExecutionConfig,
			commandType:    SingleCommand,
			expectedBehavior: "no validation, return errors, don't post comments",
		},
		{
			name:           "Webhook - multi command",
			configFunc:     NewWebhookExecutionConfig,
			commandType:    MultiCommand,
			expectedBehavior: "no validation, return errors, post summary",
		},
		{
			name:           "Webhook - built-in",
			configFunc:     NewWebhookExecutionConfig,
			commandType:    BuiltInCommand,
			expectedBehavior: "no validation, execute directly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.configFunc()
			assert.NotNil(t, config)

			// Verify configuration properties based on the config type
			switch tc.name {
			case "CLI default - single command", "CLI default - multi command", "CLI default - built-in":
				assert.False(t, config.ValidateCommentSender)
				assert.True(t, config.ValidatePRStatus)
				assert.False(t, config.DebugMode)
				assert.True(t, config.PostErrorsAsPRComments)
				assert.True(t, config.ReturnErrors)
				assert.False(t, config.StopOnFirstError)

			case "CLI debug - single command", "CLI debug - multi command", "CLI debug - built-in":
				assert.False(t, config.ValidateCommentSender)
				assert.True(t, config.ValidatePRStatus)
				assert.True(t, config.DebugMode)
				assert.True(t, config.PostErrorsAsPRComments)
				assert.True(t, config.ReturnErrors)
				assert.False(t, config.StopOnFirstError)

			case "Webhook - single command", "Webhook - multi command", "Webhook - built-in":
				assert.True(t, config.ValidateCommentSender)
				assert.True(t, config.ValidatePRStatus)
				assert.False(t, config.DebugMode)
				assert.True(t, config.PostErrorsAsPRComments)
				assert.False(t, config.ReturnErrors)
				assert.False(t, config.StopOnFirstError)
			}
		})
	}
}

// TestExecutionConfigDefaults verifies that default values are set correctly
func TestExecutionConfigDefaults(t *testing.T) {
	t.Run("CLI configurations", func(t *testing.T) {
		configs := []*ExecutionConfig{
			NewCLIExecutionConfig(false),
			NewCLIExecutionConfig(true),
		}

		for i, config := range configs {
			assert.NotNil(t, config, "config %d should not be nil", i)
			assert.False(t, config.ValidateCommentSender, "config %d: CLI doesn't validate comment sender", i)
			assert.True(t, config.ValidatePRStatus, "config %d: ValidatePRStatus should be true", i)
			assert.True(t, config.PostErrorsAsPRComments, "config %d: PostErrorsAsPRComments should be true", i)
			assert.True(t, config.ReturnErrors, "config %d: CLI returns errors", i)
			assert.False(t, config.StopOnFirstError, "config %d: StopOnFirstError should be false", i)
		}
	})

	t.Run("Webhook configuration", func(t *testing.T) {
		config := NewWebhookExecutionConfig()

		assert.NotNil(t, config)
		assert.True(t, config.ValidateCommentSender, "Webhook validates comment sender")
		assert.True(t, config.ValidatePRStatus, "Webhook validates PR status")
		assert.False(t, config.DebugMode, "DebugMode should be false")
		assert.True(t, config.PostErrorsAsPRComments, "Webhook posts errors as comments")
		assert.False(t, config.ReturnErrors, "Webhook doesn't return errors (already posted)")
		assert.False(t, config.StopOnFirstError, "StopOnFirstError should be false")
	})
}

// TestExecutionConfigModification tests that configurations can be modified after creation
func TestExecutionConfigModification(t *testing.T) {
	config := NewCLIExecutionConfig(false)

	// Modify configuration
	config.StopOnFirstError = true
	config.ValidatePRStatus = false

	assert.True(t, config.StopOnFirstError)
	assert.False(t, config.ValidatePRStatus)
	assert.False(t, config.ValidateCommentSender) // CLI doesn't validate comment sender
	assert.True(t, config.PostErrorsAsPRComments)
}

// TestExecutionConfigValidation tests validation behavior based on config
func TestExecutionConfigValidation(t *testing.T) {
	t.Run("CLI validates PR status but not sender", func(t *testing.T) {
		config := NewCLIExecutionConfig(false)
		assert.False(t, config.ValidateCommentSender)
		assert.True(t, config.ValidatePRStatus)
	})

	t.Run("CLI debug validates PR status but not sender", func(t *testing.T) {
		config := NewCLIExecutionConfig(true)
		assert.False(t, config.ValidateCommentSender)
		assert.True(t, config.ValidatePRStatus)
		assert.True(t, config.DebugMode)
	})

	t.Run("Webhook validates both sender and PR status", func(t *testing.T) {
		config := NewWebhookExecutionConfig()
		assert.True(t, config.ValidateCommentSender)
		assert.True(t, config.ValidatePRStatus)
	})
}

// TestExecutionConfigErrorHandling tests error handling behavior based on config
func TestExecutionConfigErrorHandling(t *testing.T) {
	t.Run("CLI posts errors and returns them", func(t *testing.T) {
		config := NewCLIExecutionConfig(false)
		assert.True(t, config.PostErrorsAsPRComments)
		assert.True(t, config.ReturnErrors)
	})

	t.Run("Webhook posts errors without returning", func(t *testing.T) {
		config := NewWebhookExecutionConfig()
		assert.True(t, config.PostErrorsAsPRComments)
		assert.False(t, config.ReturnErrors)
	})
}
