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
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	mock_git "github.com/AlaudaDevops/toolbox/pr-cli/testing/mock/github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

func TestHandleClose(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name        string
		args        []string
		prState     string
		expectError bool
		setupMocks  func(*mock_git.MockGitClient)
	}{
		{
			name:        "successful close",
			args:        []string{},
			prState:     "open",
			expectError: false,
			setupMocks: func(mockClient *mock_git.MockGitClient) {
				prInfo := &git.PullRequest{
					Number: 123,
					Title:  "Test PR",
					State:  "open",
				}
				mockClient.EXPECT().GetPR().Return(prInfo, nil)
				mockClient.EXPECT().ClosePR().Return(nil)
				mockClient.EXPECT().PostComment(gomock.Any()).Return(nil)
			},
		},
		{
			name:        "already closed PR",
			args:        []string{},
			prState:     "closed",
			expectError: false,
			setupMocks: func(mockClient *mock_git.MockGitClient) {
				prInfo := &git.PullRequest{
					Number: 123,
					Title:  "Test PR",
					State:  "closed",
				}
				mockClient.EXPECT().GetPR().Return(prInfo, nil)
				mockClient.EXPECT().PostComment(gomock.Any()).Return(nil)
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

			err := handler.HandleClose(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("HandleClose() error = nil, wantErr %v", tt.expectError)
				}
			} else {
				if err != nil {
					t.Errorf("HandleClose() error = %v, wantErr %v", err, tt.expectError)
				}
			}
		})
	}
}
