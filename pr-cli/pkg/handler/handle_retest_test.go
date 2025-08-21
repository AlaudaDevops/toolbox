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
	"errors"
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	mock_git "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

// TestPRHandler_HandleRetest_AllChecksPassing tests HandleRetest when all checks are passing
func TestPRHandler_HandleRetest_AllChecksPassing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	// Mock CheckRunsStatus to return all checks passing
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(true, []git.CheckRun{}, nil).
		Times(1)

	// Expect a comment about all checks passing
	mockClient.EXPECT().
		PostComment("✅ All checks are passing. No failed tests to rerun.").
		Return(nil).
		Times(1)

	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: &config.Config{},
	}

	err := handler.HandleRetest([]string{})
	if err != nil {
		t.Errorf("HandleRetest() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleRetest_CheckRunsStatusError tests HandleRetest when CheckRunsStatus fails
func TestPRHandler_HandleRetest_CheckRunsStatusError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	expectedError := errors.New("API error")
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(false, nil, expectedError).
		Times(1)

	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: &config.Config{},
	}

	err := handler.HandleRetest([]string{})
	if err == nil {
		t.Errorf("HandleRetest() expected error but got nil")
	}
	if !errors.Is(err, expectedError) {
		t.Errorf("HandleRetest() error = %v, want %v", err, expectedError)
	}
}

// TestPRHandler_HandleRetest_WithFailedChecks tests HandleRetest with failed checks
func TestPRHandler_HandleRetest_WithFailedChecks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	failedChecks := []git.CheckRun{
		{
			Name:       "Pipelines as Code CI / build-pipeline",
			Status:     "completed",
			Conclusion: "failure",
		},
		{
			Name:       "test-pipeline / unit-tests",
			Status:     "completed",
			Conclusion: "failure",
		},
	}

	mockClient.EXPECT().
		CheckRunsStatus().
		Return(false, failedChecks, nil).
		Times(1)

	// Expect individual /test comments for each pipeline
	// "Pipelines as Code CI / build-pipeline" -> "build-pipeline"
	// "test-pipeline / unit-tests" -> "unit-tests"
	mockClient.EXPECT().
		PostComment("/test build-pipeline").
		Return(nil).
		Times(1)

	mockClient.EXPECT().
		PostComment("/test unit-tests").
		Return(nil).
		Times(1)

	// Expect summary comment
	mockClient.EXPECT().
		PostComment("🔄 **Retesting failed pipelines**\n\nTriggered retests for:\n• build-pipeline\n• unit-tests").
		Return(nil).
		Times(1)

	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: &config.Config{},
	}

	err := handler.HandleRetest([]string{})
	if err != nil {
		t.Errorf("HandleRetest() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleRetest_WithSpecificPipelines tests HandleRetest with specific pipeline names
func TestPRHandler_HandleRetest_WithSpecificPipelines(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	// Even when specific pipelines are provided, HandleRetest still calls CheckRunsStatus first
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(false, []git.CheckRun{}, nil). // Return some failed checks but won't be used
		Times(1)

	// Expect individual /test comments for specified pipelines
	mockClient.EXPECT().
		PostComment("/test pipeline1").
		Return(nil).
		Times(1)

	mockClient.EXPECT().
		PostComment("/test pipeline2").
		Return(nil).
		Times(1)

	// Expect summary comment
	mockClient.EXPECT().
		PostComment("🔄 **Retesting failed pipelines**\n\nTriggered retests for:\n• pipeline1\n• pipeline2").
		Return(nil).
		Times(1)

	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: &config.Config{},
	}

	err := handler.HandleRetest([]string{"pipeline1", "pipeline2"})
	if err != nil {
		t.Errorf("HandleRetest() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleRetest_NoRetestablePipelines tests HandleRetest when no pipelines can be retested
func TestPRHandler_HandleRetest_NoRetestablePipelines(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	failedChecks := []git.CheckRun{
		{
			Name:       "codecov/project",
			Status:     "completed",
			Conclusion: "failure",
		},
		{
			Name:       "license/cla",
			Status:     "completed",
			Conclusion: "failure",
		},
	}

	mockClient.EXPECT().
		CheckRunsStatus().
		Return(false, failedChecks, nil).
		Times(1)

	// Expect comment about no retestable pipelines
	mockClient.EXPECT().
		PostComment("✅ No failed pipelines found that can be retested.\n\nSkipped checks (cannot extract pipeline name):\n• codecov/project\n• license/cla").
		Return(nil).
		Times(1)

	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: &config.Config{},
	}

	err := handler.HandleRetest([]string{})
	if err != nil {
		t.Errorf("HandleRetest() error = %v, wantErr false", err)
	}
}

// TestPRHandler_HandleRetest_PostCommentError tests HandleRetest when PostComment fails
func TestPRHandler_HandleRetest_PostCommentError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	// HandleRetest always calls CheckRunsStatus first
	mockClient.EXPECT().
		CheckRunsStatus().
		Return(false, []git.CheckRun{}, nil). // Return some failed checks but won't be used
		Times(1)

	expectedError := errors.New("failed to post comment")
	mockClient.EXPECT().
		PostComment("/test pipeline1").
		Return(expectedError).
		Times(1)

	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: &config.Config{},
	}

	err := handler.HandleRetest([]string{"pipeline1"})
	if err == nil {
		t.Errorf("HandleRetest() expected error but got nil")
	}
}

// Test_extractPipelineName tests the extractPipelineName function
func Test_extractPipelineName(t *testing.T) {
	tests := []struct {
		name      string
		checkName string
		want      string
	}{
		{
			name:      "PAC pattern",
			checkName: "Pipelines as Code CI / build-pipeline",
			want:      "build-pipeline",
		},
		{
			name:      "Pipeline with task",
			checkName: "test-pipeline / unit-tests",
			want:      "unit-tests",
		},
		{
			name:      "Pipeline with common task name",
			checkName: "my-pipeline / build",
			want:      "build",
		},
		{
			name:      "Simple pipeline name",
			checkName: "simple-pipeline",
			want:      "simple-pipeline",
		},
		{
			name:      "Codecov check (should be filtered)",
			checkName: "codecov/project",
			want:      "",
		},
		{
			name:      "License check (should be filtered)",
			checkName: "license/cla",
			want:      "",
		},
		{
			name:      "SonarCloud check (should be filtered)",
			checkName: "SonarCloud Code Analysis",
			want:      "",
		},
		{
			name:      "Merge conflict check (should be filtered)",
			checkName: "Merge conflict",
			want:      "",
		},
		{
			name:      "Dependabot check (should be filtered)",
			checkName: "dependabot/npm_and_yarn",
			want:      "",
		},
		{
			name:      "GitGuardian check (should be filtered)",
			checkName: "GitGuardian Security Checks",
			want:      "",
		},
		{
			name:      "Multiple separators",
			checkName: "CI / build-pipeline / compile",
			want:      "compile",
		},
		{
			name:      "Task at end with multiple parts",
			checkName: "namespace / pipeline-name / test",
			want:      "pipeline-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractPipelineName(tt.checkName); got != tt.want {
				t.Errorf("extractPipelineName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPRHandler_HandleRetest_MixedCheckStates tests HandleRetest with various check states
func TestPRHandler_HandleRetest_MixedCheckStates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_git.NewMockGitClient(ctrl)

	failedChecks := []git.CheckRun{
		{
			Name:       "pipeline1 / build",
			Status:     "completed",
			Conclusion: "failure",
		},
		{
			Name:       "pipeline2 / test",
			Status:     "in_progress",
			Conclusion: "",
		},
		{
			Name:       "pipeline3 / deploy",
			Status:     "completed",
			Conclusion: "success",
		},
		{
			Name:       "pipeline4 / scan",
			Status:     "completed",
			Conclusion: "timed_out",
		},
		{
			Name:       "pipeline5",
			Status:     "completed",
			Conclusion: "cancelled",
		},
	}

	mockClient.EXPECT().
		CheckRunsStatus().
		Return(false, failedChecks, nil).
		Times(1)

	// Based on extractPipelineName logic:
	// "pipeline1 / build" -> "build" (only 2 parts, return last part)
	// "pipeline4 / scan" -> "scan" (only 2 parts, return last part)
	// "pipeline5" -> "pipeline5" (direct name)

	// Expect /test comments only for failed, timed_out, and cancelled completed checks
	mockClient.EXPECT().
		PostComment("/test build").
		Return(nil).
		Times(1)

	mockClient.EXPECT().
		PostComment("/test scan").
		Return(nil).
		Times(1)

	mockClient.EXPECT().
		PostComment("/test pipeline5").
		Return(nil).
		Times(1)

	// Expect summary comment
	mockClient.EXPECT().
		PostComment("🔄 **Retesting failed pipelines**\n\nTriggered retests for:\n• build\n• scan\n• pipeline5").
		Return(nil).
		Times(1)

	handler := &PRHandler{
		Logger: logrus.New(),
		client: mockClient,
		config: &config.Config{},
	}

	err := handler.HandleRetest([]string{})
	if err != nil {
		t.Errorf("HandleRetest() error = %v, wantErr false", err)
	}
}
