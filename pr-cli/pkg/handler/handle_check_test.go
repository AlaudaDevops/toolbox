/*
Copyright 2025 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
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
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	mock_git "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

// TestPRHandler_HandleCheck_NoArgs tests HandleCheck with no arguments (original check behavior)
func TestPRHandler_HandleCheck_NoArgs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	lgtmUsers := map[string]string{
		"reviewer1": "admin",
		"reviewer2": "write",
	}

	// Mock GetComments for caching
	mockClient.EXPECT().
		GetComments().
		Return([]git.Comment{}, nil).
		Times(1)

	mockClient.EXPECT().
		GetLGTMVotes([]git.Comment{}, []string{"admin", "write"}, false).
		Return(2, lgtmUsers, nil).
		Times(1)

	mockClient.EXPECT().
		CheckRunsStatus().
		Return(true, []git.CheckRun{}, nil).
		Times(1)

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

	err := handler.HandleCheck([]string{})
	if err != nil {
		t.Errorf("HandleCheck() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleCheck_SingleSubCommand tests HandleCheck with one sub-command
func TestPRHandler_HandleCheck_SingleSubCommand(t *testing.T) {
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
		PostComment(gomock.Eq("**Check Command Results:**\n\n✅ Command `/assign user1` executed successfully")).
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

	err := handler.HandleCheck([]string{"/assign", "user1"})
	if err != nil {
		t.Errorf("HandleCheck() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleCheck_MultipleSubCommands tests HandleCheck with multiple sub-commands
func TestPRHandler_HandleCheck_MultipleSubCommands(t *testing.T) {
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

	expectedSummary := "**Check Command Results:**\n\n✅ Command `/assign user1 user2` executed successfully\n✅ Command `/label bug` executed successfully"
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

	err := handler.HandleCheck([]string{"/assign", "user1", "user2", "/label", "bug"})
	if err != nil {
		t.Errorf("HandleCheck() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleCheck_BuiltInCommandRejected tests HandleCheck rejects built-in commands
func TestPRHandler_HandleCheck_BuiltInCommandRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	expectedSummary := "**Check Command Results:** (⚠️ Some commands failed)\n\n❌ Command `/__builtin` is not allowed in batch execution"
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

	err := handler.HandleCheck([]string{"/__builtin"})
	if err != nil {
		t.Errorf("HandleCheck() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleCheck_SubCommandFailure tests HandleCheck with a sub-command that has internal success but external failure appearance
func TestPRHandler_HandleCheck_SubCommandFailure(t *testing.T) {
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

	// assign will post a success message
	mockClient.EXPECT().
		PostComment(gomock.Any()).
		Return(nil).
		Times(1)

	// HandleCheck should report this as successful since the command didn't return an error
	expectedSummary := "**Check Command Results:**\n\n✅ Command `/assign user1` executed successfully"
	mockClient.EXPECT().
		PostComment(gomock.Eq(expectedSummary)).
		Return(nil).
		Times(1)

	cfg := &config.Config{
		LGTMPermissions: []string{"admin", "write"},
		CommentSender:   "commenter",
	}
	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: cfg,
	}

	err := handler.HandleCheck([]string{"/assign", "user1"})
	if err != nil {
		t.Errorf("HandleCheck() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleCheck_ActualFailure tests HandleCheck with a command that returns an error
func TestPRHandler_HandleCheck_ActualFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	// For an unknown command, ExecuteCommand will return an error
	expectedSummary := "**Check Command Results:** (⚠️ Some commands failed)\n\n❌ Command `/unknowncommand` failed: unknown command: unknowncommand"
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

	err := handler.HandleCheck([]string{"/unknowncommand"})
	if err != nil {
		t.Errorf("HandleCheck() error = %v, wantErr false", err)
	}
}

// TestPRHandler_parseSubCommands tests the parseSubCommands method
func TestPRHandler_parseSubCommands(t *testing.T) {
	handler := &PRHandler{
		Logger: logrus.New(),
	}

	tests := []struct {
		name     string
		args     []string
		expected []SubCommand
	}{
		{
			name:     "empty args",
			args:     []string{},
			expected: []SubCommand{},
		},
		{
			name: "single command without args",
			args: []string{"/lgtm"},
			expected: []SubCommand{
				{Command: "lgtm", Args: []string{}},
			},
		},
		{
			name: "single command with args",
			args: []string{"/assign", "user1", "user2"},
			expected: []SubCommand{
				{Command: "assign", Args: []string{"user1", "user2"}},
			},
		},
		{
			name: "multiple commands",
			args: []string{"/lgtm", "/assign", "user1", "/label", "bug", "urgent"},
			expected: []SubCommand{
				{Command: "lgtm", Args: []string{}},
				{Command: "assign", Args: []string{"user1"}},
				{Command: "label", Args: []string{"bug", "urgent"}},
			},
		},
		{
			name: "args without command prefix ignored",
			args: []string{"orphan", "/lgtm", "valid"},
			expected: []SubCommand{
				{Command: "lgtm", Args: []string{"valid"}},
			},
		},
		{
			name: "merge with rebase arg",
			args: []string{"/merge", "rebase"},
			expected: []SubCommand{
				{Command: "merge", Args: []string{"rebase"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.parseSubCommands(tt.args)
			if len(result) != len(tt.expected) {
				t.Errorf("parseSubCommands() got %d commands, want %d", len(result), len(tt.expected))
				return
			}

			for i, cmd := range result {
				expected := tt.expected[i]
				if cmd.Command != expected.Command {
					t.Errorf("parseSubCommands()[%d].Command = %v, want %v", i, cmd.Command, expected.Command)
				}
				if len(cmd.Args) != len(expected.Args) {
					t.Errorf("parseSubCommands()[%d].Args length = %v, want %v", i, len(cmd.Args), len(expected.Args))
					continue
				}
				for j, arg := range cmd.Args {
					if arg != expected.Args[j] {
						t.Errorf("parseSubCommands()[%d].Args[%d] = %v, want %v", i, j, arg, expected.Args[j])
					}
				}
			}
		})
	}
}

// TestPRHandler_validateSubCommand tests the validateSubCommand method
func TestPRHandler_validateSubCommand(t *testing.T) {
	handler := &PRHandler{
		Logger: logrus.New(),
	}

	tests := []struct {
		name     string
		subCmd   SubCommand
		expected string
	}{
		{
			name:     "valid command",
			subCmd:   SubCommand{Command: "assign", Args: []string{}},
			expected: "",
		},
		{
			name:     "built-in command",
			subCmd:   SubCommand{Command: "__builtin", Args: []string{}},
			expected: "❌ Command `/__builtin` is not allowed in batch execution",
		},
		{
			name:     "command with underscore prefix",
			subCmd:   SubCommand{Command: "__internal", Args: []string{"arg1"}},
			expected: "❌ Command `/__internal arg1` is not allowed in batch execution",
		},
		{
			name:     "prohibited lgtm command",
			subCmd:   SubCommand{Command: "lgtm", Args: []string{}},
			expected: "❌ Command `/lgtm` is not allowed in batch execution",
		},
		{
			name:     "prohibited remove-lgtm command",
			subCmd:   SubCommand{Command: "remove-lgtm", Args: []string{}},
			expected: "❌ Command `/remove-lgtm` is not allowed in batch execution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.validateBatchCommand(tt.subCmd)
			if result != tt.expected {
				t.Errorf("validateBatchCommand() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPRHandler_formatResult tests the formatResult method
func TestPRHandler_formatResult(t *testing.T) {
	handler := &PRHandler{
		Logger: logrus.New(),
	}

	tests := []struct {
		name     string
		icon     string
		subCmd   SubCommand
		status   string
		expected string
	}{
		{
			name:     "success without args",
			icon:     "✅",
			subCmd:   SubCommand{Command: "lgtm", Args: []string{}},
			status:   "executed successfully",
			expected: "✅ Command `/lgtm` executed successfully",
		},
		{
			name:     "failure with args",
			icon:     "❌",
			subCmd:   SubCommand{Command: "assign", Args: []string{"user1", "user2"}},
			status:   "failed: permission denied",
			expected: "❌ Command `/assign user1 user2` failed: permission denied",
		},
		{
			name:     "success with single arg",
			icon:     "✅",
			subCmd:   SubCommand{Command: "merge", Args: []string{"rebase"}},
			status:   "executed successfully",
			expected: "✅ Command `/merge rebase` executed successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.formatResult(tt.icon, tt.subCmd, tt.status)
			if result != tt.expected {
				t.Errorf("formatResult() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPRHandler_HandleCheck_LGTMCommandRejected tests HandleCheck rejects LGTM commands
func TestPRHandler_HandleCheck_LGTMCommandRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	expectedSummary := "**Check Command Results:** (⚠️ Some commands failed)\n\n❌ Command `/lgtm` is not allowed in batch execution\n❌ Command `/remove-lgtm` is not allowed in batch execution"
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

	err := handler.HandleCheck([]string{"/lgtm", "/remove-lgtm"})
	if err != nil {
		t.Errorf("HandleCheck() error = %v, wantErr false", err)
	}
}

// TestPRHandler_getCommandDisplayName tests the getCommandDisplayName method
func TestPRHandler_getCommandDisplayName(t *testing.T) {
	handler := &PRHandler{
		Logger: logrus.New(),
	}

	tests := []struct {
		name     string
		subCmd   SubCommand
		expected string
	}{
		{
			name:     "command without args",
			subCmd:   SubCommand{Command: "lgtm", Args: []string{}},
			expected: "/lgtm",
		},
		{
			name:     "command with single arg",
			subCmd:   SubCommand{Command: "merge", Args: []string{"rebase"}},
			expected: "/merge rebase",
		},
		{
			name:     "command with multiple args",
			subCmd:   SubCommand{Command: "assign", Args: []string{"user1", "user2", "user3"}},
			expected: "/assign user1 user2 user3",
		},
		{
			name:     "remove-lgtm command",
			subCmd:   SubCommand{Command: "remove-lgtm", Args: []string{}},
			expected: "/remove-lgtm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getCommandDisplayName(tt.subCmd)
			if result != tt.expected {
				t.Errorf("getCommandDisplayName() = %v, want %v", result, tt.expected)
			}
		})
	}
}
