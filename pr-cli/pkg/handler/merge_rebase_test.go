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
	"strings"
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	git_mock "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

func TestValidateRebaseMerge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := git_mock.NewMockGitClient(ctrl)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce log noise

	cfg := &config.Config{}

	handler, err := NewPRHandlerWithClient(logger, cfg, mockClient, "test-sender")
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	tests := []struct {
		name          string
		commits       []git.Commit
		getCommitsErr error
		expectError   bool
		errorContains string
	}{
		{
			name: "should pass with single commit",
			commits: []git.Commit{
				{SHA: "abc123", Message: "Initial commit", Author: "user1"},
			},
			expectError: false,
		},
		{
			name: "should fail with multiple commits",
			commits: []git.Commit{
				{SHA: "abc123", Message: "Initial commit", Author: "user1"},
				{SHA: "def456", Message: "Second commit", Author: "user2"},
			},
			expectError:   true,
			errorContains: "rebase not allowed: PR has 2 commits",
		},
		{
			name: "should fail with many commits",
			commits: []git.Commit{
				{SHA: "abc123", Message: "First commit", Author: "user1"},
				{SHA: "def456", Message: "Second commit", Author: "user2"},
				{SHA: "ghi789", Message: "Third commit", Author: "user3"},
				{SHA: "jkl012", Message: "Fourth commit", Author: "user4"},
			},
			expectError:   true,
			errorContains: "rebase not allowed: PR has 4 commits",
		},
		{
			name:          "should pass when get commits fails (graceful fallback)",
			getCommitsErr: errors.New("API error"),
			expectError:   false,
		},
		{
			name:        "should pass with no commits (empty PR case)",
			commits:     []git.Commit{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.getCommitsErr != nil {
				mockClient.EXPECT().GetCommits().Return(nil, tt.getCommitsErr)
			} else {
				mockClient.EXPECT().GetCommits().Return(tt.commits, nil)
			}

			// For cases where we expect an error, mock PostComment
			if tt.expectError {
				mockClient.EXPECT().PostComment(gomock.Any()).Return(nil)
			}

			err := handler.validateRebaseMerge()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestPostMultipleCommitsRebaseMessage(t *testing.T) {
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
		name            string
		commits         []git.Commit
		postCommentErr  error
		expectError     bool
		checkMessage    bool
		expectedMessage string
	}{
		{
			name: "should post message with commit details",
			commits: []git.Commit{
				{SHA: "abc1234567890", Message: "First commit with a very long message that should be truncated", Author: "user1"},
				{SHA: "def456", Message: "Second commit", Author: "user2"},
			},
			checkMessage: true,
			expectError:  true,
		},
		{
			name: "should handle post comment error gracefully",
			commits: []git.Commit{
				{SHA: "abc123", Message: "First commit", Author: "user1"},
				{SHA: "def456", Message: "Second commit", Author: "user2"},
			},
			postCommentErr: errors.New("post comment failed"),
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.checkMessage {
				// Capture the message posted
				mockClient.EXPECT().PostComment(gomock.Any()).Do(func(message string) {
					// Verify message contains expected content
					if !strings.Contains(message, "Cannot rebase merge: Multiple commits detected") {
						t.Errorf("message should contain rebase error, got: %s", message)
					}
					if !strings.Contains(message, "This PR contains **2 commits**") {
						t.Errorf("message should contain commit count, got: %s", message)
					}
					if !strings.Contains(message, "`abc1234`") {
						t.Errorf("message should contain truncated SHA, got: %s", message)
					}
					if !strings.Contains(message, "First commit with a very long message that should be trun...") {
						t.Errorf("message should contain truncated commit message, got: %s", message)
					}
				}).Return(nil)
			} else {
				mockClient.EXPECT().PostComment(gomock.Any()).Return(tt.postCommentErr)
			}

			err := handler.postMultipleCommitsRebaseMessage(tt.commits)

			if !tt.expectError {
				t.Errorf("expected error but got nil")
			}
			if err == nil {
				t.Errorf("expected error but got nil")
			}

			// Check that it's a CommentedError when post succeeds
			if tt.postCommentErr == nil {
				if _, ok := err.(*CommentedError); !ok {
					t.Errorf("expected CommentedError when post succeeds, got: %T", err)
				}
			}
		})
	}
}

func TestExecuteMergeWithRebaseValidation(t *testing.T) {
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
		name           string
		method         string
		commits        []git.Commit
		expectValidate bool
		expectMerge    bool
		expectError    bool
	}{
		{
			name:   "should validate and merge with rebase method single commit",
			method: "rebase",
			commits: []git.Commit{
				{SHA: "abc123", Message: "Single commit", Author: "user1"},
			},
			expectValidate: true,
			expectMerge:    true,
			expectError:    false,
		},
		{
			name:   "should validate and fail with rebase method multiple commits",
			method: "rebase",
			commits: []git.Commit{
				{SHA: "abc123", Message: "First commit", Author: "user1"},
				{SHA: "def456", Message: "Second commit", Author: "user2"},
			},
			expectValidate: true,
			expectMerge:    false,
			expectError:    true,
		},
		{
			name:           "should not validate with squash method",
			method:         "squash",
			expectValidate: false,
			expectMerge:    true,
			expectError:    false,
		},
		{
			name:           "should not validate with merge method",
			method:         "merge",
			expectValidate: false,
			expectMerge:    true,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectValidate {
				mockClient.EXPECT().GetCommits().Return(tt.commits, nil)
				if len(tt.commits) > 1 {
					// Expect PostComment for validation failure
					mockClient.EXPECT().PostComment(gomock.Any()).Return(nil)
				}
			}

			if tt.expectMerge {
				mockClient.EXPECT().MergePR(tt.method).Return(nil)
			}

			err := handler.executeMerge(tt.method)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}
