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

package git

import "testing"

func TestGenerateCherryPickBranchName(t *testing.T) {
	tests := []struct {
		name         string
		prID         int
		commitSHA    string
		targetBranch string
		expected     string
	}{
		{
			name:         "normal case with full SHA",
			prID:         123,
			commitSHA:    "abcdef123456789",
			targetBranch: "main",
			expected:     "cherry-pick-123-to-main-abcdef1",
		},
		{
			name:         "short SHA less than 7 characters",
			prID:         456,
			commitSHA:    "abc123",
			targetBranch: "develop",
			expected:     "cherry-pick-456-to-develop-abc123",
		},
		{
			name:         "exactly 7 character SHA",
			prID:         789,
			commitSHA:    "abcdef1",
			targetBranch: "feature",
			expected:     "cherry-pick-789-to-feature-abcdef1",
		},
		{
			name:         "target branch with forward slash",
			prID:         101,
			commitSHA:    "1234567890abcdef",
			targetBranch: "release/v1.0",
			expected:     "cherry-pick-101-to-release-v1-0-1234567",
		},
		{
			name:         "target branch with dots",
			prID:         202,
			commitSHA:    "fedcba987654321",
			targetBranch: "release-v1.2.3",
			expected:     "cherry-pick-202-to-release-v1-2-3-fedcba9",
		},
		{
			name:         "target branch with both slashes and dots",
			prID:         303,
			commitSHA:    "abcdef123456789fedcba",
			targetBranch: "feature/user.auth/v2.0",
			expected:     "cherry-pick-303-to-feature-user-auth-v2-0-abcdef1",
		},
		{
			name:         "single character SHA",
			prID:         404,
			commitSHA:    "a",
			targetBranch: "main",
			expected:     "cherry-pick-404-to-main-a",
		},
		{
			name:         "empty SHA",
			prID:         505,
			commitSHA:    "",
			targetBranch: "main",
			expected:     "cherry-pick-505-to-main-",
		},
		{
			name:         "empty target branch",
			prID:         606,
			commitSHA:    "abcdef123456789",
			targetBranch: "",
			expected:     "cherry-pick-606-to--abcdef1",
		},
		{
			name:         "zero PR ID",
			prID:         0,
			commitSHA:    "abcdef123456789",
			targetBranch: "main",
			expected:     "cherry-pick-0-to-main-abcdef1",
		},
		{
			name:         "negative PR ID",
			prID:         -1,
			commitSHA:    "abcdef123456789",
			targetBranch: "main",
			expected:     "cherry-pick--1-to-main-abcdef1",
		},
		{
			name:         "target branch with multiple consecutive special chars",
			prID:         707,
			commitSHA:    "1234567890",
			targetBranch: "feature//test..branch",
			expected:     "cherry-pick-707-to-feature--test--branch-1234567",
		},
		{
			name:         "target branch starting and ending with special chars",
			prID:         808,
			commitSHA:    "abcdef123456789",
			targetBranch: ".feature/branch.",
			expected:     "cherry-pick-808-to--feature-branch--abcdef1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateCherryPickBranchName(tt.prID, tt.commitSHA, tt.targetBranch)
			if result != tt.expected {
				t.Errorf("GenerateCherryPickBranchName() = %v, want %v", result, tt.expected)
			}
		})
	}
}
