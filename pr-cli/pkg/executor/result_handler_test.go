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

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewResultHandler(t *testing.T) {
	ctx := &ExecutionContext{
		Logger: logrus.New(),
		Config: NewCLIExecutionConfig(false),
	}

	handler := NewResultHandler(ctx)

	assert.NotNil(t, handler)
	assert.Equal(t, ctx, handler.context)
}

func TestResultHandler_FormatSubCommandResult(t *testing.T) {
	ctx := &ExecutionContext{
		Logger: logrus.New(),
		Config: NewCLIExecutionConfig(false),
	}

	handler := NewResultHandler(ctx)

	tests := []struct {
		name     string
		result   SubCommandResult
		expected string
	}{
		{
			name: "successful command without args",
			result: SubCommandResult{
				Command: "lgtm",
				Args:    []string{},
				Success: true,
				Error:   nil,
			},
			expected: "✅ Command `/lgtm` executed successfully",
		},
		{
			name: "successful command with args",
			result: SubCommandResult{
				Command: "assign",
				Args:    []string{"user1", "user2"},
				Success: true,
				Error:   nil,
			},
			expected: "✅ Command `/assign user1 user2` executed successfully",
		},
		{
			name: "failed command without args",
			result: SubCommandResult{
				Command: "merge",
				Args:    []string{},
				Success: false,
				Error:   errors.New("PR not ready"),
			},
			expected: "❌ Command `/merge` failed: PR not ready",
		},
		{
			name: "failed command with args",
			result: SubCommandResult{
				Command: "merge",
				Args:    []string{"squash"},
				Success: false,
				Error:   errors.New("conflicts detected"),
			},
			expected: "❌ Command `/merge squash` failed: conflicts detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := handler.FormatSubCommandResult(tt.result)
			assert.Equal(t, tt.expected, formatted)
		})
	}
}

func TestResultHandler_HandleSingleCommandError_NoConfig(t *testing.T) {
	ctx := &ExecutionContext{
		Logger: logrus.New(),
		Config: &ExecutionConfig{
			PostErrorsAsPRComments: false,
			ReturnErrors:           false,
		},
	}

	handler := NewResultHandler(ctx)

	err := handler.HandleSingleCommandError("merge", errors.New("test error"))

	// Should just log and return nil when no config is set
	assert.NoError(t, err)
}

func TestResultHandler_HandleSingleCommandError_ReturnErrors(t *testing.T) {
	ctx := &ExecutionContext{
		Logger: logrus.New(),
		Config: &ExecutionConfig{
			PostErrorsAsPRComments: false,
			ReturnErrors:           true,
		},
	}

	handler := NewResultHandler(ctx)

	testErr := errors.New("test error")
	err := handler.HandleSingleCommandError("merge", testErr)

	// Should return the error when ReturnErrors is true
	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}
