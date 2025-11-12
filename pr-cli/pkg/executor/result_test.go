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

	"github.com/stretchr/testify/assert"
)

func TestNewSuccessResult(t *testing.T) {
	tests := []struct {
		name        string
		commandType CommandType
	}{
		{"SingleCommand", SingleCommand},
		{"MultiCommand", MultiCommand},
		{"BuiltInCommand", BuiltInCommand},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewSuccessResult(tt.commandType)

			assert.NotNil(t, result)
			assert.True(t, result.Success)
			assert.Nil(t, result.Error)
			assert.Equal(t, tt.commandType, result.CommandType)
			assert.Nil(t, result.Results)
		})
	}
}

func TestNewErrorResult(t *testing.T) {
	testError := errors.New("test error")

	tests := []struct {
		name        string
		commandType CommandType
		err         error
	}{
		{"SingleCommand with error", SingleCommand, testError},
		{"MultiCommand with error", MultiCommand, testError},
		{"BuiltInCommand with error", BuiltInCommand, testError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewErrorResult(tt.commandType, tt.err)

			assert.NotNil(t, result)
			assert.False(t, result.Success)
			assert.Equal(t, tt.err, result.Error)
			assert.Equal(t, tt.commandType, result.CommandType)
			assert.Nil(t, result.Results)
		})
	}
}

func TestNewMultiCommandResult(t *testing.T) {
	tests := []struct {
		name            string
		subResults      []SubCommandResult
		expectedSuccess bool
	}{
		{
			name: "all successful",
			subResults: []SubCommandResult{
				{Command: "lgtm", Success: true, Error: nil},
				{Command: "merge", Success: true, Error: nil},
			},
			expectedSuccess: true,
		},
		{
			name: "one failure",
			subResults: []SubCommandResult{
				{Command: "lgtm", Success: true, Error: nil},
				{Command: "merge", Success: false, Error: errors.New("merge failed")},
			},
			expectedSuccess: false,
		},
		{
			name: "all failures",
			subResults: []SubCommandResult{
				{Command: "lgtm", Success: false, Error: errors.New("lgtm failed")},
				{Command: "merge", Success: false, Error: errors.New("merge failed")},
			},
			expectedSuccess: false,
		},
		{
			name:            "empty results",
			subResults:      []SubCommandResult{},
			expectedSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewMultiCommandResult(tt.subResults)

			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedSuccess, result.Success)
			assert.Equal(t, MultiCommand, result.CommandType)
			assert.Equal(t, tt.subResults, result.Results)
		})
	}
}

func TestSubCommandResult(t *testing.T) {
	tests := []struct {
		name    string
		result  SubCommandResult
		wantErr bool
	}{
		{
			name: "successful result",
			result: SubCommandResult{
				Command: "lgtm",
				Args:    []string{},
				Success: true,
				Error:   nil,
			},
			wantErr: false,
		},
		{
			name: "failed result",
			result: SubCommandResult{
				Command: "merge",
				Args:    []string{"squash"},
				Success: false,
				Error:   errors.New("merge failed"),
			},
			wantErr: true,
		},
		{
			name: "result with multiple args",
			result: SubCommandResult{
				Command: "assign",
				Args:    []string{"user1", "user2"},
				Success: true,
				Error:   nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.result.Command, tt.result.Command)
			assert.Equal(t, tt.result.Args, tt.result.Args)
			assert.Equal(t, tt.result.Success, tt.result.Success)
			if tt.wantErr {
				assert.NotNil(t, tt.result.Error)
			} else {
				assert.Nil(t, tt.result.Error)
			}
		})
	}
}
