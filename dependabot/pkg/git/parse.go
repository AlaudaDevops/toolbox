/*
Copyright 2025 The example Authors.

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

import (
	"errors"
	"net/url"
	"strings"
)

// ParseRepoURL extracts the repository name and owner from a Git repository URL.
// It supports various formats including:
// - https://github.com/example/toolbox.git => group: example, repo: toolbox
// - https://gitlab.com/group/repo.git => group: group, repo: repo
// - https://gitlab.example.com/group/subgroup/repo.git => group: group/subgroup, repo: repo
func ParseRepoURL(repoURL string) (*Repository, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return nil, err
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) < 2 {
		return nil, errors.New("invalid repository URL: not enough path segments")
	}

	group := strings.Join(segments[:len(segments)-1], "/")
	repo := segments[len(segments)-1]
	repoPath := strings.TrimSuffix(repo, ".git")

	return &Repository{
		Group: group,
		Repo:  repoPath,
	}, nil
}
