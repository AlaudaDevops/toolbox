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
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewCherryPicker(t *testing.T) {
	logger := logrus.New()
	repoURL := "https://oauth2:token@github.com/owner/repo.git"
	token := "test-token"
	owner := "owner"
	repo := "repo"

	cherryPicker := NewCherryPicker(logger, repoURL, token, owner, repo, 1)

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
			logger := logrus.New()
			cherryPicker, err := NewCherryPickerForPlatform(logger, tt.platform, tt.token, tt.owner, tt.repo, tt.baseURL, 1)

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
	logger := logrus.New()
	cherryPicker := NewCherryPicker(logger, "https://test.com", "token", "owner", "repo", 1)
	commitSHA := "abc123def456"

	expected := "cherry-pick-1-to-main-abc123d"
	result := cherryPicker.GetCherryPickBranchName(commitSHA, "main")

	if result != expected {
		t.Errorf("Expected branch name %s, got %s", expected, result)
	}
}

func TestGetRepositoryURLWithoutCredentials(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "GitHub URL with oauth2 token",
			repoURL:  "https://oauth2:ghs_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx@github.com/owner/repo.git",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "GitLab OAuth2 URL",
			repoURL:  "https://oauth2:glpat-xxxxxxxxxxxxxxxxxxxx@gitlab.com/owner/repo.git",
			expected: "https://gitlab.com/owner/repo.git",
		},
		{
			name:     "URL without credentials",
			repoURL:  "https://github.com/owner/repo.git",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "GitHub Enterprise with oauth2 token",
			repoURL:  "https://oauth2:ghs_token@github.enterprise.com/owner/repo.git",
			expected: "https://github.enterprise.com/owner/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := &CherryPicker{repoURL: tt.repoURL}
			result := cp.getRepositoryURLWithoutCredentials()
			if result != tt.expected {
				t.Errorf("getRepositoryURLWithoutCredentials() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetHostFromURL(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "GitHub URL",
			repoURL:  "https://oauth2:ghs_token@github.com/owner/repo.git",
			expected: "github.com",
		},
		{
			name:     "GitLab URL",
			repoURL:  "https://oauth2:token@gitlab.com/owner/repo.git",
			expected: "gitlab.com",
		},
		{
			name:     "GitHub Enterprise URL",
			repoURL:  "https://oauth2:token@github.enterprise.com/owner/repo.git",
			expected: "github.enterprise.com",
		},
		{
			name:     "URL without credentials",
			repoURL:  "https://github.com/owner/repo.git",
			expected: "github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := &CherryPicker{repoURL: tt.repoURL}
			result := cp.getHostFromURL()
			if result != tt.expected {
				t.Errorf("getHostFromURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "GitHub oauth2 token in URL",
			input:    "Cloning into '/tmp/test'... fatal: could not read Password for 'https://oauth2:ghs_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx@github.com': No such device",
			expected: "Cloning into '/tmp/test'... fatal: could not read Password for 'https://oauth2:[TOKEN_REDACTED]@github.com': No such device",
		},
		{
			name:     "GitLab OAuth2 token in URL",
			input:    "fatal: Authentication failed for 'https://oauth2:glpat-xxxxxxxxxxxxxxxxxxxx@gitlab.com/user/repo.git/'",
			expected: "fatal: Authentication failed for 'https://oauth2:[TOKEN_REDACTED]@gitlab.com/user/repo.git/'",
		},
		{
			name:     "GitHub personal access token",
			input:    "Error using token ghp_abcdefghijklmnopqrstuvwxyz1234567890AB",
			expected: "Error using token [TOKEN_REDACTED]",
		},
		{
			name:     "Multiple credentials in same message",
			input:    "Failed with https://oauth2:ghs_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx@github.com and oauth2:glpat-xxxxxxxxxxxxxxxxxxxx@gitlab.com",
			expected: "Failed with https://oauth2:[TOKEN_REDACTED]@github.com and oauth2:[TOKEN_REDACTED]@gitlab.com",
		},
		{
			name:     "No credentials to sanitize",
			input:    "normal error message without credentials",
			expected: "normal error message without credentials",
		},
		{
			name:     "GitHub Enterprise URL with oauth2 token",
			input:    "Failed to clone https://oauth2:ghs_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx@github.enterprise.com/org/repo.git",
			expected: "Failed to clone https://oauth2:[TOKEN_REDACTED]@github.enterprise.com/org/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeErrorMessage(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeErrorMessage() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsEmptyCommitError(t *testing.T) {
	logger := logrus.New()
	cherryPicker := NewCherryPicker(logger, "https://test.com", "token", "owner", "repo", 1)

	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name:     "Empty commit error",
			errMsg:   "The previous cherry-pick is now empty, possibly due to conflict resolution.",
			expected: true,
		},
		{
			name:     "Nothing to commit error",
			errMsg:   "nothing to commit, working tree clean",
			expected: true,
		},
		{
			name:     "Generic empty error",
			errMsg:   "commit is empty after applying changes",
			expected: true,
		},
		{
			name:     "Regular cherry-pick error",
			errMsg:   "error: could not apply abc123d... commit message",
			expected: false,
		},
		{
			name:     "Conflict error",
			errMsg:   "CONFLICT (content): Merge conflict in file.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fmt.Errorf("%s", tt.errMsg)
			result := cherryPicker.isEmptyCommitError(err)
			if result != tt.expected {
				t.Errorf("isEmptyCommitError() = %v, want %v for error: %s", result, tt.expected, tt.errMsg)
			}
		})
	}
}
