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
	"strings"
)

// Platform represents the Git platform type
type Platform string

const (
	PlatformGitHub Platform = "github"
	PlatformGitLab Platform = "gitlab"
)

// NewCherryPickerForPlatform creates a CherryPicker instance for the specified platform
func NewCherryPickerForPlatform(platform Platform, token, owner, repo, baseURL string, prID int) (*CherryPicker, error) {
	var repoURL string

	switch platform {
	case PlatformGitHub:
		if baseURL != "" && baseURL != "https://api.github.com" {
			// GitHub Enterprise - extract hostname from API URL
			gitHost := strings.Replace(baseURL, "https://api.", "https://", 1)
			gitHost = strings.Replace(gitHost, "/api/v3", "", 1)
			repoURL = fmt.Sprintf("https://%s@%s/%s/%s.git", token, strings.TrimPrefix(gitHost, "https://"), owner, repo)
		} else {
			// GitHub.com
			repoURL = fmt.Sprintf("https://%s@github.com/%s/%s.git", token, owner, repo)
		}
	case PlatformGitLab:
		if baseURL != "" && baseURL != "https://gitlab.com" {
			// GitLab self-hosted - extract hostname from API URL
			gitHost := strings.Replace(baseURL, "/api/v4", "", 1)
			repoURL = fmt.Sprintf("https://oauth2:%s@%s/%s/%s.git", token, strings.TrimPrefix(gitHost, "https://"), owner, repo)
		} else {
			// GitLab.com
			repoURL = fmt.Sprintf("https://oauth2:%s@gitlab.com/%s/%s.git", token, owner, repo)
		}
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	return NewCherryPicker(repoURL, token, owner, repo, prID), nil
}
