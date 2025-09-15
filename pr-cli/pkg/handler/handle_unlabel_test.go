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
	mock_git "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

func TestHandleUnlabel(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name        string
		labels      []string
		expectError bool
		errorMsg    string
		setupMocks  func(*mock_git.MockGitClient)
	}{
		{
			name:        "successful unlabel single label",
			labels:      []string{"bug"},
			expectError: false,
			setupMocks: func(mockClient *mock_git.MockGitClient) {
				mockClient.EXPECT().GetLabels().Return([]string{"bug", "enhancement"}, nil)
				mockClient.EXPECT().RemoveLabels([]string{"bug"}).Return(nil)
				mockClient.EXPECT().GetLabels().Return([]string{"enhancement"}, nil)
				mockClient.EXPECT().PostComment("🏷️ Labels `bug` have been removed from this PR by @testuser").Return(nil)
			},
		},
		{
			name:        "successful unlabel multiple labels",
			labels:      []string{"bug", "enhancement", "documentation"},
			expectError: false,
			setupMocks: func(mockClient *mock_git.MockGitClient) {
				mockClient.EXPECT().GetLabels().Return([]string{"bug", "enhancement", "documentation", "feature"}, nil)
				mockClient.EXPECT().RemoveLabels([]string{"bug", "enhancement", "documentation"}).Return(nil)
				mockClient.EXPECT().GetLabels().Return([]string{"feature"}, nil)
				mockClient.EXPECT().PostComment("🏷️ Labels `bug, enhancement, documentation` have been removed from this PR by @testuser").Return(nil)
			},
		},
		{
			name:        "empty labels list",
			labels:      []string{},
			expectError: true,
			errorMsg:    "no labels specified",
			setupMocks:  func(mockClient *mock_git.MockGitClient) {},
		},
		{
			name:        "nil labels list",
			labels:      nil,
			expectError: true,
			errorMsg:    "no labels specified",
			setupMocks:  func(mockClient *mock_git.MockGitClient) {},
		},
		{
			name:        "get current labels fails but continues",
			labels:      []string{"bug"},
			expectError: false,
			setupMocks: func(mockClient *mock_git.MockGitClient) {
				mockClient.EXPECT().GetLabels().Return(nil, errors.New("API error"))
				mockClient.EXPECT().RemoveLabels([]string{"bug"}).Return(nil)
				mockClient.EXPECT().GetLabels().Return([]string{"enhancement"}, nil)
				mockClient.EXPECT().PostComment("🏷️ Labels `bug` have been removed from this PR by @testuser").Return(nil)
			},
		},
		{
			name:        "remove labels fails",
			labels:      []string{"bug"},
			expectError: true,
			errorMsg:    "failed to remove labels",
			setupMocks: func(mockClient *mock_git.MockGitClient) {
				mockClient.EXPECT().GetLabels().Return([]string{"bug", "enhancement"}, nil)
				mockClient.EXPECT().RemoveLabels([]string{"bug"}).Return(errors.New("API error"))
			},
		},
		{
			name:        "get updated labels fails but continues",
			labels:      []string{"bug"},
			expectError: false,
			setupMocks: func(mockClient *mock_git.MockGitClient) {
				mockClient.EXPECT().GetLabels().Return([]string{"bug", "enhancement"}, nil)
				mockClient.EXPECT().RemoveLabels([]string{"bug"}).Return(nil)
				mockClient.EXPECT().GetLabels().Return(nil, errors.New("API error"))
				mockClient.EXPECT().PostComment("🏷️ Labels `bug` have been removed from this PR by @testuser").Return(nil)
			},
		},
		{
			name:        "post comment fails",
			labels:      []string{"bug"},
			expectError: true,
			setupMocks: func(mockClient *mock_git.MockGitClient) {
				mockClient.EXPECT().GetLabels().Return([]string{"bug", "enhancement"}, nil)
				mockClient.EXPECT().RemoveLabels([]string{"bug"}).Return(nil)
				mockClient.EXPECT().GetLabels().Return([]string{"enhancement"}, nil)
				mockClient.EXPECT().PostComment("🏷️ Labels `bug` have been removed from this PR by @testuser").Return(errors.New("comment API error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mock_git.NewMockGitClient(ctrl)
			tt.setupMocks(mockClient)

			cfg := &config.Config{
				PRNum:         123,
				CommentSender: "testuser",
			}

			handler := &PRHandler{
				Logger: logger,
				client: mockClient,
				config: cfg,
			}

			err := handler.HandleUnlabel(tt.labels)

			if tt.expectError {
				if err == nil {
					t.Errorf("HandleUnlabel() error = nil, wantErr %v", tt.expectError)
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					if err.Error()[:len(tt.errorMsg)] != tt.errorMsg {
						t.Errorf("HandleUnlabel() error = %v, want error containing %v", err, tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("HandleUnlabel() error = %v, wantErr %v", err, tt.expectError)
				}
			}
		})
	}
}
