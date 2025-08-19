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

import (
	"fmt"
	"strings"
)

// GenerateCherryPickBranchName generates a consistent branch name for cherry-pick operations
// Format: cherry-pick-{PR_ID}-to-{target_branch}-{short_sha}
func GenerateCherryPickBranchName(prID int, commitSHA, targetBranch string) string {
	// Get short SHA (first 7 characters)
	shortSHA := commitSHA
	if len(commitSHA) > 7 {
		shortSHA = commitSHA[:7]
	}

	// Clean target branch name (replace special characters with hyphens)
	cleanTargetBranch := strings.ReplaceAll(targetBranch, "/", "-")
	cleanTargetBranch = strings.ReplaceAll(cleanTargetBranch, ".", "-")

	return fmt.Sprintf("cherry-pick-%d-to-%s-%s", prID, cleanTargetBranch, shortSHA)
}
