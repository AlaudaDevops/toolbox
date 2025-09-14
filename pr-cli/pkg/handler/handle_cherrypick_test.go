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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
	mockgit "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
)

func TestHandleCherrypick_BranchNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockgit.NewMockGitClient(ctrl)
	logger := logrus.New()

	handler := &PRHandler{
		client: mockClient,
		config: &config.Config{
			PRNum:         123,
			CommentSender: "testuser",
		},
		Logger:   logger,
		prSender: "prauthor",
	}

	targetBranch := "nonexistent-branch"

	// Mock PR information
	prInfo := &git.PullRequest{
		Number: 123,
		Title:  "Test PR",
		State:  "open",
		Merged: false,
	}

	// Mock user permissions check - user has permission
	mockClient.EXPECT().
		CheckUserPermissions("testuser", gomock.Any()).
		Return(true, "admin", nil)

	// Mock GetPR call
	mockClient.EXPECT().
		GetPR().
		Return(prInfo, nil)

	// Mock BranchExists call - branch doesn't exist
	mockClient.EXPECT().
		BranchExists(targetBranch).
		Return(false, nil)

	// Mock PostComment call with expected error message
	expectedMessage := fmt.Sprintf(messages.CherryPickBranchNotFoundTemplate, targetBranch)
	mockClient.EXPECT().
		PostComment(expectedMessage).
		Return(nil)

	// Test the function
	err := handler.HandleCherrypick([]string{targetBranch})
	if err != nil {
		t.Errorf("HandleCherrypick() error = %v, wantErr false", err)
	}
}

func TestHandleCherrypick_BranchExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockgit.NewMockGitClient(ctrl)
	logger := logrus.New()

	handler := &PRHandler{
		client: mockClient,
		config: &config.Config{
			PRNum:         123,
			CommentSender: "testuser",
		},
		Logger:   logger,
		prSender: "prauthor",
	}

	targetBranch := "release/v1.0"

	// Mock PR information
	prInfo := &git.PullRequest{
		Number: 123,
		Title:  "Test PR",
		State:  "open",
		Merged: false,
	}

	// Mock user permissions check - user has permission
	mockClient.EXPECT().
		CheckUserPermissions("testuser", gomock.Any()).
		Return(true, "admin", nil)

	// Mock GetPR call
	mockClient.EXPECT().
		GetPR().
		Return(prInfo, nil)

	// Mock BranchExists call - branch exists
	mockClient.EXPECT().
		BranchExists(targetBranch).
		Return(true, nil)

	// Mock PostComment call with expected success message
	expectedMessage := fmt.Sprintf(messages.CherryPickScheduledTemplate, targetBranch)
	mockClient.EXPECT().
		PostComment(expectedMessage).
		Return(nil)

	// Test the function
	err := handler.HandleCherrypick([]string{targetBranch})
	if err != nil {
		t.Errorf("HandleCherrypick() error = %v, wantErr false", err)
	}
}

func TestHandleCherrypick_BranchExistsCheckFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockgit.NewMockGitClient(ctrl)
	logger := logrus.New()

	handler := &PRHandler{
		client: mockClient,
		config: &config.Config{
			PRNum:         123,
			CommentSender: "testuser",
		},
		Logger:   logger,
		prSender: "prauthor",
	}

	targetBranch := "release/v1.0"

	// Mock PR information
	prInfo := &git.PullRequest{
		Number: 123,
		Title:  "Test PR",
		State:  "open",
		Merged: false,
	}

	// Mock user permissions check - user has permission
	mockClient.EXPECT().
		CheckUserPermissions("testuser", gomock.Any()).
		Return(true, "admin", nil)

	// Mock GetPR call
	mockClient.EXPECT().
		GetPR().
		Return(prInfo, nil)

	// Mock BranchExists call - check fails
	mockClient.EXPECT().
		BranchExists(targetBranch).
		Return(false, fmt.Errorf("API error"))

	// Mock PostComment call with expected success message (fallback behavior)
	expectedMessage := fmt.Sprintf(messages.CherryPickScheduledTemplate, targetBranch)
	mockClient.EXPECT().
		PostComment(expectedMessage).
		Return(nil)

	// Test the function - should proceed despite check failure
	err := handler.HandleCherrypick([]string{targetBranch})
	if err != nil {
		t.Errorf("HandleCherrypick() error = %v, wantErr false", err)
	}
}
