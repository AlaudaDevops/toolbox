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

package cherrypick

import (
	"slices"
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
)

func TestCherryPickWithMultiLineCommands(t *testing.T) {
	tests := []struct {
		name             string
		commentBody      string
		expectedBranches []string
	}{
		{
			name:             "Single cherry-pick command",
			commentBody:      "/cherry-pick release-1.0",
			expectedBranches: []string{"release-1.0"},
		},
		{
			name: "Multi-line comment with LGTM and cherry-pick",
			commentBody: `/lgtm
/cherry-pick release-1.0`,
			expectedBranches: []string{"release-1.0"},
		},
		{
			name: "Multi-line comment with multiple cherry-picks",
			commentBody: `/lgtm
/cherry-pick release-1.0
/cherrypick hotfix-branch`,
			expectedBranches: []string{"release-1.0", "hotfix-branch"},
		},
		{
			name: "Cherry-pick mixed with other commands",
			commentBody: `/assign user1
/cherry-pick release-1.0
/merge`,
			expectedBranches: []string{"release-1.0"},
		},
		{
			name: "No cherry-pick commands",
			commentBody: `/lgtm
/merge
/assign user1`,
			expectedBranches: []string{},
		},
		{
			name: "Cherry-pick with text around it",
			commentBody: `This looks good to me!
/lgtm
/cherry-pick release-1.0
Thanks for the fix!`,
			expectedBranches: []string{"release-1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Split the comment into command lines
			commandLines := comment.SplitCommandLines(tt.commentBody)

			// Find all cherry-pick commands
			var foundBranches []string
			for _, cmdLine := range commandLines {
				matches := CherryPickPattern.FindStringSubmatch(cmdLine)
				if len(matches) > 1 {
					foundBranches = append(foundBranches, matches[1])
				}
			}

			// Verify the results
			if len(foundBranches) != len(tt.expectedBranches) {
				t.Errorf("Expected %d cherry-pick commands, got %d. Found: %v, Expected: %v",
					len(tt.expectedBranches), len(foundBranches), foundBranches, tt.expectedBranches)
				return
			}

			// Check each expected branch is found
			for _, expectedBranch := range tt.expectedBranches {
				if slices.Contains(foundBranches, expectedBranch) {
					continue
				}
				t.Errorf("Expected to find cherry-pick command for branch %q, but it was not found. Found: %v",
					expectedBranch, foundBranches)
			}
		})
	}
}
