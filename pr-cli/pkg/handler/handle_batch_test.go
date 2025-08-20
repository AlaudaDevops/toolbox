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
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	mock_git "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

// TestPRHandler_HandleBatch_SingleCommand tests HandleBatch with one command
func TestPRHandler_HandleBatch_SingleCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	mockClient.EXPECT().
		GetRequestedReviewers().
		Return([]string{}, nil).
		Times(1)

	mockClient.EXPECT().
		AssignReviewers([]string{"user1"}).
		Return(nil).
		Times(1)

	mockClient.EXPECT().
		GetRequestedReviewers().
		Return([]string{"user1"}, nil).
		Times(1)

	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	mockClient.EXPECT().
		PostComment(gomock.Eq("**Batch Execution Results:**\n\n✅ Command `/assign user1` executed successfully")).
		Return(nil).
		Times(1)

	cfg := &config.Config{
		LGTMThreshold:   1,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
	}
	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: cfg,
	}

	err := handler.HandleBatch([]string{"/assign", "user1"})
	if err != nil {
		t.Errorf("HandleBatch() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleBatch_MultipleCommands tests HandleBatch with multiple commands
func TestPRHandler_HandleBatch_MultipleCommands(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

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

	mockClient.EXPECT().
		GetLabels().
		Return([]string{}, nil).
		Times(2)

	mockClient.EXPECT().
		AddLabels([]string{"bug"}).
		Return(nil).
		Times(1)

	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	expectedSummary := "**Batch Execution Results:**\n\n✅ Command `/assign user1 user2` executed successfully\n✅ Command `/label bug` executed successfully"
	mockClient.EXPECT().
		PostComment(gomock.Eq(expectedSummary)).
		Return(nil).
		Times(1)

	cfg := &config.Config{
		LGTMThreshold:   1,
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
	}
	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: cfg,
	}

	err := handler.HandleBatch([]string{"/assign", "user1", "user2", "/label", "bug"})
	if err != nil {
		t.Errorf("HandleBatch() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleBatch_LGTMCommandRejected tests HandleBatch rejects LGTM commands
func TestPRHandler_HandleBatch_LGTMCommandRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	expectedSummary := "**Batch Execution Results:** (⚠️ Some commands failed)\n\n❌ Command `/lgtm` is not allowed in batch execution\n❌ Command `/remove-lgtm` is not allowed in batch execution"
	mockClient.EXPECT().
		PostComment(gomock.Eq(expectedSummary)).
		Return(nil).
		Times(1)

	cfg := &config.Config{
		CommentSender: "commenter",
	}
	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: cfg,
	}

	err := handler.HandleBatch([]string{"/lgtm", "/remove-lgtm"})
	if err != nil {
		t.Errorf("HandleBatch() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleBatch_RecursiveBatchRejected tests HandleBatch rejects recursive batch calls
func TestPRHandler_HandleBatch_RecursiveBatchRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	expectedSummary := "**Batch Execution Results:** (⚠️ Some commands failed)\n\n❌ Command `/batch` is not allowed in batch execution"
	mockClient.EXPECT().
		PostComment(gomock.Eq(expectedSummary)).
		Return(nil).
		Times(1)

	cfg := &config.Config{
		CommentSender: "commenter",
	}
	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: cfg,
	}

	err := handler.HandleBatch([]string{"/batch"})
	if err != nil {
		t.Errorf("HandleBatch() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleBatch_NoCommands tests HandleBatch with no commands
func TestPRHandler_HandleBatch_NoCommands(t *testing.T) {
	handler := &PRHandler{
		Logger: logrus.New(),
	}

	err := handler.HandleBatch([]string{})
	if err == nil {
		t.Errorf("HandleBatch() error = nil, wantErr true")
	}

	expectedError := "no valid commands provided for batch execution"
	if err.Error() != expectedError {
		t.Errorf("HandleBatch() error = %v, want %v", err.Error(), expectedError)
	}
}

// TestPRHandler_HandleBatch_BuiltInCommandRejected tests HandleBatch rejects built-in commands
func TestPRHandler_HandleBatch_BuiltInCommandRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	expectedSummary := "**Batch Execution Results:** (⚠️ Some commands failed)\n\n❌ Command `/__builtin` is not allowed in batch execution"
	mockClient.EXPECT().
		PostComment(gomock.Eq(expectedSummary)).
		Return(nil).
		Times(1)

	cfg := &config.Config{
		CommentSender: "commenter",
	}
	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: cfg,
	}

	err := handler.HandleBatch([]string{"/__builtin"})
	if err != nil {
		t.Errorf("HandleBatch() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleBatch_CommandFailure tests HandleBatch with a command that returns an error
func TestPRHandler_HandleBatch_CommandFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	// For an unknown command, ExecuteCommand will return an error
	expectedSummary := "**Batch Execution Results:** (⚠️ Some commands failed)\n\n❌ Command `/unknowncommand` failed: unknown command: unknowncommand"
	mockClient.EXPECT().
		PostComment(gomock.Eq(expectedSummary)).
		Return(nil).
		Times(1)

	cfg := &config.Config{
		CommentSender: "commenter",
	}
	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: cfg,
	}

	err := handler.HandleBatch([]string{"/unknowncommand"})
	if err != nil {
		t.Errorf("HandleBatch() error = %v, wantErr false", err)
	}
}
