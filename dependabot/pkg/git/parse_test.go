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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantRepo    *Repository
		wantErr     bool
		errContains string
	}{
		{
			name:     "simple GitHub URL with .git",
			url:      "https://github.com/example/toolbox.git",
			wantRepo: &Repository{Group: "example", Repo: "toolbox"},
			wantErr:  false,
		},
		{
			name:     "GitHub URL without .git",
			url:      "https://github.com/example/toolbox",
			wantRepo: &Repository{Group: "example", Repo: "toolbox"},
			wantErr:  false,
		},
		{
			name:     "GitLab URL with subgroups",
			url:      "https://gitlab.com/group/subgroup/repo.git",
			wantRepo: &Repository{Group: "group/subgroup", Repo: "repo"},
			wantErr:  false,
		},
		{
			name:    "URL with insufficient segments",
			url:     "https://github.com/only-owner",
			wantErr: true,
		},
		{
			name:     "URL with special characters",
			url:      "https://github.com/org-name/repo-name.git",
			wantRepo: &Repository{Group: "org-name", Repo: "repo-name"},
			wantErr:  false,
		},
		{
			name:    "Empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRepo, err := ParseRepoURL(tt.url)

			// Check error cases
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRepoURL() error = nil, wantErr = true")
					return
				}
				return
			}

			// Check non-error cases
			if err != nil {
				t.Errorf("ParseRepoURL() unexpected error = %v", err)
				return
			}

			if diff := cmp.Diff(gotRepo, tt.wantRepo); diff != "" {
				t.Errorf("ParseRepoURL() diff = %v", diff)
			}
		})
	}
}
