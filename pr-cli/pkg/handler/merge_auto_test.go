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

package handler

import (
	"errors"
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	git_mock "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

func TestDetermineMergeMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := git_mock.NewMockGitClient(ctrl)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce log noise

	cfg := &config.Config{
		MergeMethod: "auto",
	}

	handler, err := NewPRHandlerWithClient(logger, cfg, mockClient, "test-sender")
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	tests := []struct {
		name      string
		args      []string
		setupMock func()
		expected  string
	}{
		{
			name:     "should use explicit squash argument",
			args:     []string{"squash"},
			expected: "squash",
		},
		{
			name:     "should use explicit rebase argument",
			args:     []string{"rebase"},
			expected: "rebase",
		},
		{
			name:     "should use explicit merge argument",
			args:     []string{"merge"},
			expected: "merge",
		},
		{
			name: "should ignore invalid arguments and use auto",
			args: []string{"invalid"},
			setupMock: func() {
				mockClient.EXPECT().GetAvailableMergeMethods().Return([]string{"rebase", "squash"}, nil)
			},
			expected: "rebase",
		},
		{
			name: "should use auto mode explicitly",
			args: []string{"auto"},
			setupMock: func() {
				mockClient.EXPECT().GetAvailableMergeMethods().Return([]string{"rebase", "squash", "merge"}, nil)
			},
			expected: "rebase",
		},
		{
			name: "should use auto mode as default",
			args: []string{},
			setupMock: func() {
				mockClient.EXPECT().GetAvailableMergeMethods().Return([]string{"squash", "merge"}, nil)
			},
			expected: "squash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			result := handler.determineMergeMethod(tt.args)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSelectAutoMergeMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := git_mock.NewMockGitClient(ctrl)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.Config{}

	handler, err := NewPRHandlerWithClient(logger, cfg, mockClient, "test-sender")
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	tests := []struct {
		name             string
		availableMethods []string
		apiError         error
		expected         string
	}{
		{
			name:             "should select rebase when available",
			availableMethods: []string{"merge", "rebase", "squash"},
			expected:         "rebase",
		},
		{
			name:             "should select squash when rebase not available",
			availableMethods: []string{"merge", "squash"},
			expected:         "squash",
		},
		{
			name:             "should select merge when only merge available",
			availableMethods: []string{"merge"},
			expected:         "merge",
		},
		{
			name:     "should fallback to squash when API fails",
			apiError: errors.New("API error"),
			expected: "squash",
		},
		{
			name:             "should fallback to squash when no methods available",
			availableMethods: []string{},
			expected:         "squash",
		},
		{
			name:             "should use first available method as fallback",
			availableMethods: []string{"custom"},
			expected:         "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.apiError != nil {
				mockClient.EXPECT().GetAvailableMergeMethods().Return(nil, tt.apiError)
			} else {
				mockClient.EXPECT().GetAvailableMergeMethods().Return(tt.availableMethods, nil)
			}

			result := handler.selectAutoMergeMethod()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
