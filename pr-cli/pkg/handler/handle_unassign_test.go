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

func TestHandleUnassign(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name        string
		users       []string
		expectError bool
		errorMsg    string
		setupMocks  func(*mock_git.MockGitClient)
	}{
		{
			name:        "successful unassignment single user",
			users:       []string{"user1"},
			expectError: false,
			setupMocks: func(mockClient *mock_git.MockGitClient) {
				mockClient.EXPECT().RemoveReviewers([]string{"user1"}).Return(nil)
				mockClient.EXPECT().PostComment("♻️ Removed @user1 from the review list. Thanks for your time!").Return(nil)
			},
		},
		{
			name:        "successful unassignment multiple users",
			users:       []string{"user1", "user2", "user3"},
			expectError: false,
			setupMocks: func(mockClient *mock_git.MockGitClient) {
				mockClient.EXPECT().RemoveReviewers([]string{"user1", "user2", "user3"}).Return(nil)
				mockClient.EXPECT().PostComment("♻️ Removed @user1, @user2, @user3 from the review list. Thanks for your time!").Return(nil)
			},
		},
		{
			name:        "empty users list",
			users:       []string{},
			expectError: true,
			errorMsg:    "no users specified for unassignment",
			setupMocks:  func(mockClient *mock_git.MockGitClient) {},
		},
		{
			name:        "nil users list",
			users:       nil,
			expectError: true,
			errorMsg:    "no users specified for unassignment",
			setupMocks:  func(mockClient *mock_git.MockGitClient) {},
		},
		{
			name:        "remove reviewers fails",
			users:       []string{"user1"},
			expectError: true,
			errorMsg:    "failed to remove reviewers",
			setupMocks: func(mockClient *mock_git.MockGitClient) {
				mockClient.EXPECT().RemoveReviewers([]string{"user1"}).Return(errors.New("API error"))
			},
		},
		{
			name:        "post comment fails",
			users:       []string{"user1"},
			expectError: true,
			setupMocks: func(mockClient *mock_git.MockGitClient) {
				mockClient.EXPECT().RemoveReviewers([]string{"user1"}).Return(nil)
				mockClient.EXPECT().PostComment("♻️ Removed @user1 from the review list. Thanks for your time!").Return(errors.New("comment API error"))
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
				PRNum: 123,
			}

			handler := &PRHandler{
				Logger: logger,
				client: mockClient,
				config: cfg,
			}

			err := handler.HandleUnassign(tt.users)

			if tt.expectError {
				if err == nil {
					t.Errorf("HandleUnassign() error = nil, wantErr %v", tt.expectError)
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					if err.Error()[:len(tt.errorMsg)] != tt.errorMsg {
						t.Errorf("HandleUnassign() error = %v, want error containing %v", err, tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("HandleUnassign() error = %v, wantErr %v", err, tt.expectError)
				}
			}
		})
	}
}
