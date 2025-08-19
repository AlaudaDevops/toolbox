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

// TestPRHandler_HandleHelp demonstrates how to use the generated mock
func TestPRHandler_HandleHelp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock GitClient
	mockClient := mock_git.NewMockGitClient(ctrl)

	// Set expectations on the mock
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	// Create PRHandler with the mock client
	cfg := &config.Config{
		LGTMThreshold:   1,
		LGTMPermissions: []string{"admin", "write"},
		MergeMethod:     "rebase",
		CommentSender:   "commenter",
		PRNum:           123,
	}
	handler := &PRHandler{
		Logger:   logrus.New(),
		client:   mockClient,
		config:   cfg,
		prSender: "author",
	}

	// Test the HandleHelp method
	err := handler.HandleHelp()
	if err != nil {
		t.Errorf("HandleHelp() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleAssign demonstrates testing with mock expectations
func TestPRHandler_HandleAssign(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	// Set up expectations for GetRequestedReviewers (called twice - before and after)
	mockClient.EXPECT().
		GetRequestedReviewers().
		Return([]string{}, nil).
		Times(1)

	mockClient.EXPECT().
		AssignReviewers([]string{"user1", "user2"}).
		Return(nil).
		Times(1)

	mockClient.EXPECT().
		GetRequestedReviewers().
		Return([]string{"user1", "user2"}, nil).
		Times(1)

	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: &config.Config{CommentSender: "commenter"},
	}

	err := handler.HandleAssign([]string{"user1", "user2"})
	if err != nil {
		t.Errorf("HandleAssign() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleAssign_MultipleUsers 测试多个用户分配
func TestPRHandler_HandleAssign_MultipleUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	// Set up expectations for multiple users
	expectedUsers := []string{"user1", "user2", "user3", "user4"}

	mockClient.EXPECT().
		GetRequestedReviewers().
		Return([]string{}, nil).
		Times(1)

	mockClient.EXPECT().
		AssignReviewers(expectedUsers).
		Return(nil).
		Times(1)

	mockClient.EXPECT().
		GetRequestedReviewers().
		Return(expectedUsers, nil).
		Times(1)

	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: &config.Config{CommentSender: "commenter"},
	}

	err := handler.HandleAssign(expectedUsers)
	if err != nil {
		t.Errorf("HandleAssign() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleAssign_WithAtSymbol 测试带@符号的用户名
func TestPRHandler_HandleAssign_WithAtSymbol(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	expectedUsers := []string{"@user1", "@user2", "@user3"}

	mockClient.EXPECT().
		GetRequestedReviewers().
		Return([]string{}, nil).
		Times(1)

	// Set up expectations - note that @ symbols should be stripped by the client
	mockClient.EXPECT().
		AssignReviewers(expectedUsers).
		Return(nil).
		Times(1)

	mockClient.EXPECT().
		GetRequestedReviewers().
		Return([]string{"user1", "user2", "user3"}, nil). // GitHub API会返回没有@的用户名
		Times(1)

	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: &config.Config{CommentSender: "commenter"},
	}

	err := handler.HandleAssign(expectedUsers)
	if err != nil {
		t.Errorf("HandleAssign() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleAssign_PartialFailure 测试部分分配失败的情况
func TestPRHandler_HandleAssign_PartialFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	// 模拟部分失败的情况 - 假设 user2 分配失败
	mockClient.EXPECT().
		GetRequestedReviewers().
		Return([]string{}, nil).
		Times(1)

	expectedUsers := []string{"user1", "user2", "user3"}
	mockClient.EXPECT().
		AssignReviewers(expectedUsers).
		Return(fmt.Errorf("failed to assign some reviewers: [user2] (successfully assigned: [user1 user3])")).
		Times(1)

	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: &config.Config{CommentSender: "commenter"},
	}

	err := handler.HandleAssign(expectedUsers)
	if err == nil {
		t.Error("HandleAssign() expected error for partial failure but got nil")
	}

	if !strings.Contains(err.Error(), "user2") {
		t.Errorf("HandleAssign() error = %v, should contain 'user2'", err)
	}
}

// TestPRHandler_HandleLGTM demonstrates testing complex scenarios with mocks
func TestPRHandler_HandleLGTM(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	// Mock user permission check - user has admin permission
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(true, "admin", nil).
		Times(1)

	// Mock LGTM votes response - threshold is met (2 >= 2)
	lgtmUsers := map[string]string{
		"reviewer1": "admin",
		"reviewer2": "write",
	}

	mockClient.EXPECT().
		GetLGTMVotes([]string{"admin", "write"}, false).
		Return(2, lgtmUsers, nil).
		Times(1)

	// Mock CheckRunsStatus call from generateLGTMStatusMessage (when threshold is met)
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(true, []git.CheckRun{}, nil).
		Times(1)

	// Mock the ApprovePR call when threshold is met (only once now)
	mockClient.EXPECT().
		ApprovePR(gomock.Any()).
		Return(nil).
		Times(1)

	cfg := &config.Config{
		LGTMThreshold:   2,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
	}
	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: cfg,
	}

	err := handler.HandleLGTM()
	if err != nil {
		t.Errorf("HandleLGTM() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleLGTM_NoPermission tests LGTM when user lacks permission
func TestPRHandler_HandleLGTM_NoPermission(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	// Mock user permission check - user only has read permission
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(false, "read", nil).
		Times(1)

	// Mock PostComment call for permission denied message
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	cfg := &config.Config{
		LGTMThreshold:   2,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
	}
	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: cfg,
	}

	err := handler.HandleLGTM()
	if err != nil {
		t.Errorf("HandleLGTM() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleRemoveLGTM tests removing LGTM when user has permission
func TestPRHandler_HandleRemoveLGTM(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	// Mock user permission check - user has admin permission
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(true, "admin", nil).
		Times(1)

	// Mock the DismissApprove call
	mockClient.EXPECT().
		DismissApprove(gomock.Any()).
		Return(nil).
		Times(1)

	// Mock LGTM votes response after dismissal
	lgtmUsers := map[string]string{
		"reviewer1": "write",
	}

	mockClient.EXPECT().
		GetLGTMVotes([]string{"admin", "write"}, false).
		Return(1, lgtmUsers, nil).
		Times(1)

	// Mock CheckRunsStatus call from generateLGTMStatusMessage
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(true, []git.CheckRun{}, nil).
		Times(1)

	// Mock PostComment for status update
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	cfg := &config.Config{
		LGTMThreshold:   2,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
	}
	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: cfg,
	}

	err := handler.HandleRemoveLGTM()
	if err != nil {
		t.Errorf("HandleRemoveLGTM() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleRemoveLGTM_NoApproval tests removing LGTM when no approval exists
func TestPRHandler_HandleRemoveLGTM_NoApproval(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	// Mock user permission check - user has admin permission
	mockClient.EXPECT().
		CheckUserPermissions("commenter", []string{"admin", "write"}).
		Return(true, "admin", nil).
		Times(1)

	// Mock the DismissApprove call returning error for no approval found
	// The error message must exactly match what the code checks for
	mockClient.EXPECT().
		DismissApprove(gomock.Any()).
		Return(fmt.Errorf("no approval review found")).
		Times(1)

	// Mock PostComment for no approval message
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	cfg := &config.Config{
		LGTMThreshold:   2,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
	}
	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: cfg,
	}

	err := handler.HandleRemoveLGTM()
	if err != nil {
		t.Errorf("HandleRemoveLGTM() error = %v, wantErr false", err)
	}
}
