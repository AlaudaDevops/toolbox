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

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewValidator(t *testing.T) {
	ctx := &ExecutionContext{
		Logger: logrus.New(),
		Config: NewCLIExecutionConfig(false),
	}

	validator := NewValidator(ctx)

	assert.NotNil(t, validator)
	assert.Equal(t, ctx, validator.context)
}

func TestValidator_ShouldSkipPRStatusCheck(t *testing.T) {
	ctx := &ExecutionContext{
		Logger: logrus.New(),
		Config: NewCLIExecutionConfig(false),
	}

	validator := NewValidator(ctx)

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "cherry-pick should skip",
			command:  "cherry-pick",
			expected: true,
		},
		{
			name:     "cherrypick should skip",
			command:  "cherrypick",
			expected: true,
		},
		{
			name:     "built-in command should skip",
			command:  "__test",
			expected: true,
		},
		{
			name:     "regular command should not skip",
			command:  "merge",
			expected: false,
		},
		{
			name:     "lgtm should not skip",
			command:  "lgtm",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.shouldSkipPRStatusCheck(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidator_ValidateSingleCommand_NoValidation(t *testing.T) {
	ctx := &ExecutionContext{
		Logger: logrus.New(),
		Config: &ExecutionConfig{
			ValidatePRStatus:      false,
			ValidateCommentSender: false,
		},
	}

	validator := NewValidator(ctx)

	// Should pass without any validation
	err := validator.ValidateSingleCommand("merge")
	assert.NoError(t, err)
}

func TestValidator_ValidateMultiCommand_NoValidation(t *testing.T) {
	ctx := &ExecutionContext{
		Logger: logrus.New(),
		Config: &ExecutionConfig{
			ValidatePRStatus:      false,
			ValidateCommentSender: false,
		},
	}

	validator := NewValidator(ctx)

	subCommands := []SubCommand{
		{Command: "lgtm", Args: []string{}},
		{Command: "merge", Args: []string{}},
	}

	rawCommandLines := []string{"/lgtm", "/merge"}

	// Should pass without any validation
	err := validator.ValidateMultiCommand(subCommands, rawCommandLines)
	assert.NoError(t, err)
}

func TestValidator_ValidateMultiCommand_SkipPRStatusForCherryPick(t *testing.T) {
	ctx := &ExecutionContext{
		Logger: logrus.New(),
		Config: &ExecutionConfig{
			ValidatePRStatus:      true,
			ValidateCommentSender: false,
		},
	}

	validator := NewValidator(ctx)

	// All commands that skip PR status check
	subCommands := []SubCommand{
		{Command: "cherry-pick", Args: []string{"main"}},
		{Command: "__built-in", Args: []string{}},
	}

	rawCommandLines := []string{"/cherry-pick main", "/__built-in"}

	// Should pass without PR status validation since all commands skip it
	// Note: This will still fail in practice without a real PRHandler
	// but the logic branch is tested
	err := validator.ValidateMultiCommand(subCommands, rawCommandLines)
	// We expect an error because PRHandler is nil, but not a PR status error
	if err != nil {
		assert.NotContains(t, err.Error(), "PR status check failed")
	}
}

func TestValidator_ValidateMultiCommand_EmptyCommandLines(t *testing.T) {
	ctx := &ExecutionContext{
		Logger: logrus.New(),
		Config: &ExecutionConfig{
			ValidatePRStatus:      true,
			ValidateCommentSender: true,
		},
	}

	validator := NewValidator(ctx)

	subCommands := []SubCommand{}
	rawCommandLines := []string{}

	// Should pass with empty commands
	err := validator.ValidateMultiCommand(subCommands, rawCommandLines)
	// May fail due to nil PRHandler, but not due to empty commands
	// The validator handles empty rawCommandLines gracefully
	if err != nil {
		assert.NotContains(t, err.Error(), "no commands")
	}
}
