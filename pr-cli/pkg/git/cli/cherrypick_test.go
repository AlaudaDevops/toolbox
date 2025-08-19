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

package cli

import (
	"testing"
)

func TestNewCherryPicker(t *testing.T) {
	repoURL := "https://token@github.com/owner/repo.git"
	token := "test-token"
	owner := "owner"
	repo := "repo"

	cherryPicker := NewCherryPicker(repoURL, token, owner, repo, 1)

	if cherryPicker.repoURL != repoURL {
		t.Errorf("Expected repoURL %s, got %s", repoURL, cherryPicker.repoURL)
	}

	if cherryPicker.token != token {
		t.Errorf("Expected token %s, got %s", token, cherryPicker.token)
	}

	if cherryPicker.owner != owner {
		t.Errorf("Expected owner %s, got %s", owner, cherryPicker.owner)
	}

	if cherryPicker.repo != repo {
		t.Errorf("Expected repo %s, got %s", repo, cherryPicker.repo)
	}
}

func TestNewCherryPickerForPlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		token    string
		owner    string
		repo     string
		baseURL  string
		wantErr  bool
	}{
		{
			name:     "GitHub with default URL",
			platform: PlatformGitHub,
			token:    "test-token",
			owner:    "owner",
			repo:     "repo",
			baseURL:  "",
			wantErr:  false,
		},
		{
			name:     "GitHub Enterprise",
			platform: PlatformGitHub,
			token:    "test-token",
			owner:    "owner",
			repo:     "repo",
			baseURL:  "https://github.example.com",
			wantErr:  false,
		},
		{
			name:     "GitLab with default URL",
			platform: PlatformGitLab,
			token:    "test-token",
			owner:    "owner",
			repo:     "repo",
			baseURL:  "",
			wantErr:  false,
		},
		{
			name:     "GitLab self-hosted",
			platform: PlatformGitLab,
			token:    "test-token",
			owner:    "owner",
			repo:     "repo",
			baseURL:  "https://gitlab.example.com",
			wantErr:  false,
		},
		{
			name:     "Unsupported platform",
			platform: "unsupported",
			token:    "test-token",
			owner:    "owner",
			repo:     "repo",
			baseURL:  "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cherryPicker, err := NewCherryPickerForPlatform(tt.platform, tt.token, tt.owner, tt.repo, tt.baseURL, 1)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cherryPicker == nil {
				t.Error("Expected cherryPicker, got nil")
			}
		})
	}
}

func TestGetCherryPickBranchName(t *testing.T) {
	cherryPicker := NewCherryPicker("https://test.com", "token", "owner", "repo", 1)
	commitSHA := "abc123def456"

	expected := "cherry-pick-1-to-main-abc123d"
	result := cherryPicker.GetCherryPickBranchName(commitSHA, "main")

	if result != expected {
		t.Errorf("Expected branch name %s, got %s", expected, result)
	}
}
