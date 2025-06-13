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

func TestRepository_String(t *testing.T) {
	tests := []struct {
		name  string
		group string
		repo  string
		want  string
	}{
		{
			name:  "simple alphanumeric",
			group: "owner",
			repo:  "project",
			want:  "owner/project",
		},
		{
			name:  "with dashes",
			group: "my-org",
			repo:  "test-repo",
			want:  "my-org/test-repo",
		},
		{
			name:  "with subgroup",
			group: "my-org/subgroup",
			repo:  "test-repo",
			want:  "my-org/subgroup/test-repo",
		},
		{
			name:  "with underscores",
			group: "my_org",
			repo:  "test_repo",
			want:  "my_org/test_repo",
		},
		{
			name:  "empty group",
			group: "",
			repo:  "repo",
			want:  "repo",
		},
		{
			name:  "empty repo",
			group: "group",
			repo:  "",
			want:  "group",
		},
		{
			name:  "both empty",
			group: "",
			repo:  "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Repository{
				Group: tt.group,
				Repo:  tt.repo,
			}
			if got := r.String(); got != tt.want {
				t.Errorf("Repository.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepository_Encode(t *testing.T) {
	tests := []struct {
		name  string
		group string
		repo  string
		want  string
	}{
		{
			name:  "simple path",
			group: "owner",
			repo:  "project",
			want:  "owner%2Fproject",
		},
		{
			name:  "with subgroup",
			group: "my-org/subgroup",
			repo:  "test-repo",
			want:  "my-org%2Fsubgroup%2Ftest-repo",
		},
		{
			name:  "empty values",
			group: "",
			repo:  "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Repository{
				Group: tt.group,
				Repo:  tt.repo,
			}
			if got := r.UrlEncode(); got != tt.want {
				t.Errorf("Repository.Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}
