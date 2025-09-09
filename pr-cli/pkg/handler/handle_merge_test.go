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
	"fmt"
	"strings"
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	mock_git "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

// createMockHandler creates a PRHandler with mock client for testing
func createMockHandler(ctrl *gomock.Controller, cfg *config.Config) (*PRHandler, *mock_git.MockGitClient) {
	mockClient := mock_git.NewMockGitClient(ctrl)
	handler := &PRHandler{
		Logger:   logrus.New(),
		client:   mockClient,
		config:   cfg,
		prSender: "pr-author",
	}
	return handler, mockClient
}

// TestHandleMerge_Success tests successful merge with all validations passing
func TestHandleMerge_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		LGTMThreshold:   2,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
		MergeMethod:     "squash",
		PRNum:           123,
		RobotAccounts:   []string{"bot1", "bot2"},
	}

	handler, mockClient := createMockHandler(ctrl, cfg)

	// Mock permission validation
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(true, "admin", nil).
		Times(1)

	// Mock check runs validation
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(true, []git.CheckRun{}, nil).
		Times(1)

	// Mock GetComments for LGTM validation
	mockClient.EXPECT().
		GetComments().
		Return([]git.Comment{}, nil).
		Times(1)

	// Mock LGTM votes validation
	lgtmUsers := map[string]string{
		"reviewer1": "admin",
		"reviewer2": "write",
	}
	mockClient.EXPECT().
		GetLGTMVotes([]git.Comment{}, []string{"admin", "write"}, false).
		Return(2, lgtmUsers, nil).
		Times(1)

	// Mock merge execution
	mockClient.EXPECT().
		MergePR("squash").
		Return(nil).
		Times(1)

	// Note: cherry-pick check will use cached comments from LGTM validation
	// so no additional GetComments() call is needed

	// Mock successful merge comment
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	err := handler.HandleMerge([]string{})
	if err != nil {
		t.Errorf("HandleMerge() error = %v, wantErr false", err)
	}
}

// TestHandleMerge_InsufficientPermissions tests merge failure due to insufficient permissions
func TestHandleMerge_InsufficientPermissions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		LGTMThreshold:   2,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
		MergeMethod:     "squash",
		PRNum:           123,
	}

	handler, mockClient := createMockHandler(ctrl, cfg)

	// Mock permission validation - user has read permission, not PR creator
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(false, "read", nil).
		Times(1)

	// Mock error comment post
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	err := handler.HandleMerge([]string{})
	if err == nil {
		t.Error("HandleMerge() expected error for insufficient permissions but got nil")
	}

	// Check that it's a CommentedError
	if _, ok := err.(*CommentedError); !ok {
		t.Errorf("Expected CommentedError, got %T", err)
	}
}

// TestHandleMerge_PRCreatorBypass tests PR creator can merge without permissions
func TestHandleMerge_PRCreatorBypass(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		LGTMThreshold:   2,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "pr-author", // Same as prSender
		MergeMethod:     "squash",
		PRNum:           123,
	}

	handler, mockClient := createMockHandler(ctrl, cfg)

	// Mock permission validation - user has read permission but is PR creator
	mockClient.EXPECT().
		CheckUserPermissions("pr-author", []string{"admin", "write"}).
		Return(false, "read", nil).
		Times(1)

	// Mock check runs validation
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(true, []git.CheckRun{}, nil).
		Times(1)

	// Mock GetComments for LGTM validation
	mockClient.EXPECT().
		GetComments().
		Return([]git.Comment{}, nil).
		Times(1)

	// Mock LGTM votes validation
	lgtmUsers := map[string]string{
		"reviewer1": "admin",
		"reviewer2": "write",
	}
	mockClient.EXPECT().
		GetLGTMVotes([]git.Comment{}, []string{"admin", "write"}, false).
		Return(2, lgtmUsers, nil).
		Times(1)

	// Mock merge execution
	mockClient.EXPECT().
		MergePR("squash").
		Return(nil).
		Times(1)

	// Note: cherry-pick check will use cached comments from LGTM validation

	// Mock successful merge comment
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	err := handler.HandleMerge([]string{})
	if err != nil {
		t.Errorf("HandleMerge() error = %v, wantErr false", err)
	}
}

// TestHandleMerge_ChecksNotPassing tests merge failure due to failed checks
func TestHandleMerge_ChecksNotPassing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		LGTMThreshold:   2,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
		MergeMethod:     "squash",
		PRNum:           123,
	}

	handler, mockClient := createMockHandler(ctrl, cfg)

	// Mock permission validation
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(true, "admin", nil).
		Times(1)

	// Mock check runs validation - some checks failed
	failedChecks := []git.CheckRun{
		{Name: "test-check", Status: "completed", Conclusion: "failure", URL: "https://example.com/check1"},
		{Name: "lint-check", Status: "completed", Conclusion: "failure", URL: "https://example.com/check2"},
	}
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(false, failedChecks, nil).
		Times(1)

	// Mock error comment post
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	err := handler.HandleMerge([]string{})
	if err == nil {
		t.Error("HandleMerge() expected error for failed checks but got nil")
	}

	// Check that it's a CommentedError
	if _, ok := err.(*CommentedError); !ok {
		t.Errorf("Expected CommentedError, got %T", err)
	}
}

// TestHandleMerge_NotEnoughLGTM tests merge failure due to insufficient LGTM votes
func TestHandleMerge_NotEnoughLGTM(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		LGTMThreshold:   2,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
		MergeMethod:     "squash",
		PRNum:           123,
	}

	handler, mockClient := createMockHandler(ctrl, cfg)

	// Mock permission validation
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(true, "admin", nil).
		Times(1)

	// Mock check runs validation
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(true, []git.CheckRun{}, nil).
		Times(1)

	// Mock GetComments for LGTM validation
	mockClient.EXPECT().
		GetComments().
		Return([]git.Comment{}, nil).
		Times(1)

	// Mock LGTM votes validation - not enough votes
	lgtmUsers := map[string]string{
		"reviewer1": "admin",
	}
	mockClient.EXPECT().
		GetLGTMVotes([]git.Comment{}, []string{"admin", "write"}, false).
		Return(1, lgtmUsers, nil).
		Times(1)

	// Mock error comment post
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	err := handler.HandleMerge([]string{})
	if err == nil {
		t.Error("HandleMerge() expected error for insufficient LGTM but got nil")
	}

	// Check that it's a CommentedError
	if _, ok := err.(*CommentedError); !ok {
		t.Errorf("Expected CommentedError, got %T", err)
	}
}

// TestHandleMerge_MergeFailed tests merge failure during actual merge operation
func TestHandleMerge_MergeFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		LGTMThreshold:   2,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
		MergeMethod:     "squash",
		PRNum:           123,
	}

	handler, mockClient := createMockHandler(ctrl, cfg)

	// Mock permission validation
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(true, "admin", nil).
		Times(1)

	// Mock check runs validation
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(true, []git.CheckRun{}, nil).
		Times(1)

	// Mock GetComments for LGTM validation
	mockClient.EXPECT().
		GetComments().
		Return([]git.Comment{}, nil).
		Times(1)

	// Mock LGTM votes validation
	lgtmUsers := map[string]string{
		"reviewer1": "admin",
		"reviewer2": "write",
	}
	mockClient.EXPECT().
		GetLGTMVotes([]git.Comment{}, []string{"admin", "write"}, false).
		Return(2, lgtmUsers, nil).
		Times(1)

	// Mock merge execution - fail
	mergeError := fmt.Errorf("merge conflict detected")
	mockClient.EXPECT().
		MergePR("squash").
		Return(mergeError).
		Times(1)

	// Mock error comment post
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	err := handler.HandleMerge([]string{})
	if err == nil {
		t.Error("HandleMerge() expected error for merge failure but got nil")
	}

	// Check that it's a CommentedError
	if commentedErr, ok := err.(*CommentedError); !ok {
		t.Errorf("Expected CommentedError, got %T", err)
	} else if commentedErr.Err != mergeError {
		t.Errorf("Expected wrapped error %v, got %v", mergeError, commentedErr.Err)
	}
}

// TestHandleMerge_WithRebaseMethod tests rebase merge with single commit
func TestHandleMerge_WithRebaseMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		LGTMThreshold:   1,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
		MergeMethod:     "squash",
		PRNum:           123,
	}

	handler, mockClient := createMockHandler(ctrl, cfg)

	// Mock permission validation
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(true, "admin", nil).
		Times(1)

	// Mock check runs validation
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(true, []git.CheckRun{}, nil).
		Times(1)

	// Mock GetComments for LGTM validation
	mockClient.EXPECT().
		GetComments().
		Return([]git.Comment{}, nil).
		Times(1)

	// Mock LGTM votes validation
	lgtmUsers := map[string]string{"reviewer1": "admin"}
	mockClient.EXPECT().
		GetLGTMVotes([]git.Comment{}, []string{"admin", "write"}, false).
		Return(1, lgtmUsers, nil).
		Times(1)

	// Mock commit validation for rebase
	mockClient.EXPECT().
		GetCommits().
		Return([]git.Commit{{SHA: "abc123", Message: "Test commit"}}, nil).
		Times(1)

	// Mock merge execution
	mockClient.EXPECT().
		MergePR("rebase").
		Return(nil).
		Times(1)

	// Note: cherry-pick check will use cached comments from LGTM validation

	// Mock successful merge comment
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	err := handler.HandleMerge([]string{"rebase"})
	if err != nil {
		t.Errorf("HandleMerge() error = %v, wantErr false", err)
	}
}

// TestHandleMerge_WithAutoMethod tests auto method selection
func TestHandleMerge_WithAutoMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		LGTMThreshold:   1,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
		MergeMethod:     "squash",
		PRNum:           123,
	}

	handler, mockClient := createMockHandler(ctrl, cfg)

	// Mock permission validation
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(true, "admin", nil).
		Times(1)

	// Mock check runs validation
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(true, []git.CheckRun{}, nil).
		Times(1)

	// Mock GetComments for LGTM validation
	mockClient.EXPECT().
		GetComments().
		Return([]git.Comment{}, nil).
		Times(1)

	// Mock LGTM votes validation
	lgtmUsers := map[string]string{"reviewer1": "admin"}
	mockClient.EXPECT().
		GetLGTMVotes([]git.Comment{}, []string{"admin", "write"}, false).
		Return(1, lgtmUsers, nil).
		Times(1)

	// Mock auto method selection - rebase is available and preferred
	mockClient.EXPECT().
		GetAvailableMergeMethods().
		Return([]string{"rebase", "squash", "merge"}, nil).
		Times(1)

	// Mock commit validation for rebase (auto-selected)
	mockClient.EXPECT().
		GetCommits().
		Return([]git.Commit{{SHA: "abc123", Message: "Test commit"}}, nil).
		Times(1)

	// Mock merge execution with auto-selected method (rebase)
	mockClient.EXPECT().
		MergePR("rebase").
		Return(nil).
		Times(1)

	// Note: cherry-pick check will use cached comments from LGTM validation

	// Mock successful merge comment
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	err := handler.HandleMerge([]string{"auto"})
	if err != nil {
		t.Errorf("HandleMerge() error = %v, wantErr false", err)
	}
}

// TestHandleMerge_RebaseWithMultipleCommits tests rebase merge with multiple commits
func TestHandleMerge_RebaseWithMultipleCommits(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		LGTMThreshold:   1,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
		MergeMethod:     "rebase",
		PRNum:           123,
	}

	handler, mockClient := createMockHandler(ctrl, cfg)

	// Mock permission validation
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(true, "admin", nil).
		Times(1)

	// Mock check runs validation
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(true, []git.CheckRun{}, nil).
		Times(1)

	// Mock GetComments for LGTM validation
	mockClient.EXPECT().
		GetComments().
		Return([]git.Comment{}, nil).
		Times(1)

	// Mock LGTM votes validation
	lgtmUsers := map[string]string{"reviewer1": "admin"}
	mockClient.EXPECT().
		GetLGTMVotes([]git.Comment{}, []string{"admin", "write"}, false).
		Return(1, lgtmUsers, nil).
		Times(1)

	// Mock commit validation - multiple commits
	commits := []git.Commit{
		{SHA: "abc123", Message: "First commit"},
		{SHA: "def456", Message: "Second commit"},
		{SHA: "ghi789", Message: "Third commit"},
	}
	mockClient.EXPECT().
		GetCommits().
		Return(commits, nil).
		Times(1)

	// Mock error comment post for multiple commits
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		DoAndReturn(func(message string) error {
			// Verify the message contains expected content
			if !strings.Contains(message, "3 commits") {
				t.Errorf("Expected message to contain '3 commits', got: %s", message)
			}
			if !strings.Contains(message, "abc123") && !strings.Contains(message, "def456") {
				t.Errorf("Expected message to contain commit SHAs, got: %s", message)
			}
			return nil
		}).
		Times(1)

	err := handler.HandleMerge([]string{"rebase"})
	if err == nil {
		t.Error("HandleMerge() expected error for multiple commits in rebase but got nil")
	}

	// Check that it's a CommentedError
	if _, ok := err.(*CommentedError); !ok {
		t.Errorf("Expected CommentedError, got %T", err)
	}
}

// TestHandleMerge_WithCherryPickComments tests merge success with cherry-pick comments detected
func TestHandleMerge_WithCherryPickComments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		LGTMThreshold:   1,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
		MergeMethod:     "squash",
		PRNum:           123,
	}

	handler, mockClient := createMockHandler(ctrl, cfg)

	// Mock permission validation
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(true, "admin", nil).
		Times(1)

	// Mock check runs validation
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(true, []git.CheckRun{}, nil).
		Times(1)

	// Mock GetComments for LGTM validation - return comments with cherry-pick
	// This will be cached and used later for cherry-pick check
	cherryPickComments := []git.Comment{
		{Body: "/cherry-pick release-1.0", User: git.User{Login: "user1"}},
		{Body: "/cherrypick release-2.0", User: git.User{Login: "user2"}},
		{Body: "Regular comment", User: git.User{Login: "user3"}},
	}
	mockClient.EXPECT().
		GetComments().
		Return(cherryPickComments, nil).
		Times(1)

	// Mock LGTM votes validation
	lgtmUsers := map[string]string{"reviewer1": "admin"}
	mockClient.EXPECT().
		GetLGTMVotes(cherryPickComments, []string{"admin", "write"}, false).
		Return(1, lgtmUsers, nil).
		Times(1)

	// Mock merge execution
	mockClient.EXPECT().
		MergePR("squash").
		Return(nil).
		Times(1)

	// Note: cherry-pick check will use cached comments (cherryPickComments)

	// Mock successful merge comment
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	err := handler.HandleMerge([]string{})
	if err != nil {
		t.Errorf("HandleMerge() error = %v, wantErr false", err)
	}
}

// TestHandleMergeDetermineMergeMethod tests the determineMergeMethod function
func TestHandleMergeDetermineMergeMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{MergeMethod: "squash"}
	handler, _ := createMockHandler(ctrl, cfg)

	testCases := []struct {
		name     string
		args     []string
		expected string
	}{
		{"no args - use config default", []string{}, "squash"},
		{"merge method", []string{"merge"}, "merge"},
		{"squash method", []string{"squash"}, "squash"},
		{"rebase method", []string{"rebase"}, "rebase"},
		{"auto method", []string{"auto"}, "auto"}, // Will trigger selectAutoMergeMethod
		{"invalid method", []string{"invalid"}, "squash"},
		{"multiple args - use first", []string{"rebase", "extra"}, "rebase"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// For auto method test, we need to mock GetAvailableMergeMethods
			if tc.expected == "auto" {
				// Actually, when args is "auto", determineMergeMethod calls selectAutoMergeMethod
				// which returns the actual selected method, not "auto"
				// So we need to mock that and expect the selected method
				mockClient := handler.client.(*mock_git.MockGitClient)
				mockClient.EXPECT().
					GetAvailableMergeMethods().
					Return([]string{"rebase", "squash", "merge"}, nil).
					Times(1)

				result := handler.ExposedDetermineMergeMethod(tc.args)
				// Should return "rebase" (first preferred method)
				if result != "rebase" {
					t.Errorf("determineMergeMethod(%v) = %v, want %v", tc.args, result, "rebase")
				}
			} else {
				result := handler.ExposedDetermineMergeMethod(tc.args)
				if result != tc.expected {
					t.Errorf("determineMergeMethod(%v) = %v, want %v", tc.args, result, tc.expected)
				}
			}
		})
	}
}

// TestHandleMergeSelectAutoMergeMethod tests the selectAutoMergeMethod function
func TestHandleMergeSelectAutoMergeMethod(t *testing.T) {
	testCases := []struct {
		name             string
		availableMethods []string
		apiError         error
		expected         string
		expectedLog      string
	}{
		{
			name:             "rebase preferred and available",
			availableMethods: []string{"rebase", "squash", "merge"},
			expected:         "rebase",
		},
		{
			name:             "squash preferred when rebase not available",
			availableMethods: []string{"squash", "merge"},
			expected:         "squash",
		},
		{
			name:             "merge fallback when only merge available",
			availableMethods: []string{"merge"},
			expected:         "merge",
		},
		{
			name:             "first available when no preferred methods",
			availableMethods: []string{"unknown", "custom"},
			expected:         "unknown",
		},
		{
			name:             "empty available methods",
			availableMethods: []string{},
			expected:         "squash",
		},
		{
			name:        "API error fallback",
			apiError:    fmt.Errorf("API error"),
			expected:    "squash",
			expectedLog: "Failed to get available merge methods",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg := &config.Config{MergeMethod: "auto"}
			handler, mockClient := createMockHandler(ctrl, cfg)

			if tc.apiError != nil {
				mockClient.EXPECT().
					GetAvailableMergeMethods().
					Return(nil, tc.apiError).
					Times(1)
			} else {
				mockClient.EXPECT().
					GetAvailableMergeMethods().
					Return(tc.availableMethods, nil).
					Times(1)
			}

			result := handler.ExposedSelectAutoMergeMethod()
			if result != tc.expected {
				t.Errorf("selectAutoMergeMethod() = %v, want %v", result, tc.expected)
			}
		})
	}
}

// TestHandleMergeValidateRebaseMerge tests the validateRebaseMerge function
func TestHandleMergeValidateRebaseMerge(t *testing.T) {
	testCases := []struct {
		name        string
		commits     []git.Commit
		apiError    error
		expectError bool
		errorType   interface{}
	}{
		{
			name: "single commit - valid",
			commits: []git.Commit{
				{SHA: "abc123", Message: "Single commit message"},
			},
			expectError: false,
		},
		{
			name: "multiple commits - invalid",
			commits: []git.Commit{
				{SHA: "abc123", Message: "First commit"},
				{SHA: "def456", Message: "Second commit"},
			},
			expectError: true,
			errorType:   &CommentedError{},
		},
		{
			name:        "API error - allow merge to proceed",
			apiError:    fmt.Errorf("failed to get commits"),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg := &config.Config{MergeMethod: "rebase"}
			handler, mockClient := createMockHandler(ctrl, cfg)

			if tc.apiError != nil {
				mockClient.EXPECT().
					GetCommits().
					Return(nil, tc.apiError).
					Times(1)
			} else {
				mockClient.EXPECT().
					GetCommits().
					Return(tc.commits, nil).
					Times(1)

				// If multiple commits, expect PostComment call for error message
				if len(tc.commits) > 1 {
					mockClient.EXPECT().
						PostComment(gomock.Any()).
						Return(nil).
						Times(1)
				}
			}

			err := handler.ExposedValidateRebaseMerge()

			if tc.expectError && err == nil {
				t.Errorf("validateRebaseMerge() expected error but got nil")
			} else if !tc.expectError && err != nil {
				t.Errorf("validateRebaseMerge() unexpected error: %v", err)
			}

			if tc.expectError && tc.errorType != nil {
				if _, ok := err.(*CommentedError); !ok {
					t.Errorf("Expected CommentedError, got %T", err)
				}
			}
		})
	}
}
