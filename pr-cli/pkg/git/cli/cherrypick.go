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
	"os"
	"os/exec"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
)

// CherryPicker handles Git CLI cherry-pick operations
type CherryPicker struct {
	repoURL    string
	token      string
	owner      string
	repo       string
	workingDir string
	currentDir string // Store original directory for cleanup
	prID       int    // Original PR ID
}

// NewCherryPicker creates a new CherryPicker instance
func NewCherryPicker(repoURL, token, owner, repo string, prID int) *CherryPicker {
	currentDir, _ := os.Getwd()
	return &CherryPicker{
		repoURL:    repoURL,
		token:      token,
		owner:      owner,
		repo:       repo,
		currentDir: currentDir,
		prID:       prID,
	}
}

// CherryPickCommit performs a cherry-pick operation using Git CLI
func (cp *CherryPicker) CherryPickCommit(commitSHA, targetBranch string) error {
	// Create a temporary directory for the repository
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("cherrypick-%s-%s", cp.repo, targetBranch))
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		// Cleanup: restore original directory and remove temp directory
		if cp.currentDir != "" {
			os.Chdir(cp.currentDir)
		}
		os.RemoveAll(tempDir)
	}()

	cp.workingDir = tempDir

	// Clone the repository
	if err := cp.cloneRepository(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Configure git user for commits (required for cherry-pick)
	if err := cp.configureGit(); err != nil {
		return fmt.Errorf("failed to configure git: %w", err)
	}

	// Fetch and checkout the target branch
	branchName := cp.GetCherryPickBranchName(commitSHA, targetBranch)
	if err := cp.checkoutTargetBranch(targetBranch, branchName); err != nil {
		return fmt.Errorf("failed to checkout target branch: %w", err)
	}

	// Fetch the commit to cherry-pick
	if err := cp.fetchCommit(commitSHA); err != nil {
		return fmt.Errorf("failed to fetch commit: %w", err)
	}

	// Perform the cherry-pick
	if err := cp.performCherryPick(commitSHA); err != nil {
		return fmt.Errorf("failed to perform cherry-pick: %w", err)
	}

	// Push the new branch
	if err := cp.pushBranch(branchName); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	return nil
}

// CherryPickCommits performs cherry-pick operations for multiple commits using Git CLI
func (cp *CherryPicker) CherryPickCommits(commits []git.Commit, targetBranch string) error {
	if len(commits) == 0 {
		return fmt.Errorf("no commits provided for cherry-pick")
	}

	// Create a temporary directory for the repository
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("cherrypick-%s-%s", cp.repo, targetBranch))
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		// Cleanup: restore original directory and remove temp directory
		if cp.currentDir != "" {
			os.Chdir(cp.currentDir)
		}
		os.RemoveAll(tempDir)
	}()

	cp.workingDir = tempDir

	// Clone the repository
	if err := cp.cloneRepository(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Configure git user for commits (required for cherry-pick)
	if err := cp.configureGit(); err != nil {
		return fmt.Errorf("failed to configure git: %w", err)
	}

	// Use the last commit SHA for branch naming (consistent with API approach)
	lastCommit := commits[len(commits)-1]
	branchName := cp.GetCherryPickBranchName(lastCommit.SHA, targetBranch)

	// Fetch and checkout the target branch
	if err := cp.checkoutTargetBranch(targetBranch, branchName); err != nil {
		return fmt.Errorf("failed to checkout target branch: %w", err)
	}

	// Fetch all commits to cherry-pick
	for _, commit := range commits {
		if err := cp.fetchCommit(commit.SHA); err != nil {
			return fmt.Errorf("failed to fetch commit %s: %w", commit.SHA, err)
		}
	}

	// Perform cherry-pick for all commits in order
	for i, commit := range commits {
		fmt.Printf("Cherry-picking commit %d/%d: %s\n", i+1, len(commits), commit.SHA)
		if err := cp.performCherryPick(commit.SHA); err != nil {
			return fmt.Errorf("failed to cherry-pick commit %s (%d/%d): %w", commit.SHA, i+1, len(commits), err)
		}
	}

	// Push the new branch
	if err := cp.pushBranch(branchName); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	return nil
}

// cloneRepository clones the repository to the temporary directory
func (cp *CherryPicker) cloneRepository() error {
	cloneCmd := exec.Command("git", "clone", cp.repoURL, cp.workingDir)
	output, err := cloneCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w, output: %s", err, string(output))
	}
	return nil
}

// configureGit configures git user settings required for cherry-pick commits
func (cp *CherryPicker) configureGit() error {
	// Change to the repository directory
	if err := os.Chdir(cp.workingDir); err != nil {
		return fmt.Errorf("failed to change to working directory: %w", err)
	}

	// Configure git user for commits
	if err := cp.runGitCommand("config", "user.email", "pr-cli@alaudadevops.com"); err != nil {
		return fmt.Errorf("failed to configure git user email: %w", err)
	}

	if err := cp.runGitCommand("config", "user.name", "PR CLI Bot"); err != nil {
		return fmt.Errorf("failed to configure git user name: %w", err)
	}

	return nil
}

// runGitCommand runs a git command and returns error with output
func (cp *CherryPicker) runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s failed: %w, output: %s", strings.Join(args, " "), err, string(output))
	}
	return nil
}

// checkoutTargetBranch checks out the target branch for cherry-pick
func (cp *CherryPicker) checkoutTargetBranch(targetBranch, branchName string) error {
	// Fetch all branches to ensure we have the latest refs
	if err := cp.runGitCommand("fetch", "origin"); err != nil {
		return fmt.Errorf("failed to fetch from origin: %w", err)
	}

	// Fetch the specific target branch
	if err := cp.runGitCommand("fetch", "origin", targetBranch); err != nil {
		return fmt.Errorf("failed to fetch target branch %s: %w", targetBranch, err)
	}

	// Check if the target branch exists remotely
	if err := cp.runGitCommand("rev-parse", "--verify", fmt.Sprintf("origin/%s", targetBranch)); err != nil {
		return fmt.Errorf("target branch %s does not exist on remote: %w", targetBranch, err)
	}

	// Create and checkout the new branch based on the target branch
	if err := cp.runGitCommand("checkout", "-b", branchName, fmt.Sprintf("origin/%s", targetBranch)); err != nil {
		return fmt.Errorf("failed to checkout new branch %s from %s: %w", branchName, targetBranch, err)
	}

	return nil
}

// fetchCommit fetches the specific commit to cherry-pick
func (cp *CherryPicker) fetchCommit(commitSHA string) error {
	// Try to fetch the specific commit
	if err := cp.runGitCommand("fetch", "origin", commitSHA); err != nil {
		// If fetching specific commit fails, try to fetch all refs
		// This handles cases where the commit might be in a PR branch
		if err := cp.runGitCommand("fetch", "origin", "+refs/*:refs/remotes/origin/*"); err != nil {
			return fmt.Errorf("failed to fetch commit %s: %w", commitSHA, err)
		}
	}

	// Verify the commit exists
	if err := cp.runGitCommand("rev-parse", "--verify", commitSHA); err != nil {
		return fmt.Errorf("commit %s not found after fetch: %w", commitSHA, err)
	}

	return nil
}

// performCherryPick executes the cherry-pick operation
func (cp *CherryPicker) performCherryPick(commitSHA string) error {
	// Try cherry-pick with mainline parent (helpful for merge commits)
	err := cp.runGitCommand("cherry-pick", "-m", "1", commitSHA)
	if err != nil {
		// If mainline cherry-pick fails, try normal cherry-pick
		err = cp.runGitCommand("cherry-pick", commitSHA)
		if err != nil {
			// Check if it's a conflict
			if strings.Contains(err.Error(), "CONFLICT") || strings.Contains(err.Error(), "conflict") {
				// Abort the current cherry-pick
				cp.runGitCommand("cherry-pick", "--abort")

				// Try with strategy options for automatic conflict resolution
				err = cp.runGitCommand("cherry-pick", "--strategy=recursive", "--strategy-option=theirs", commitSHA)
				if err != nil {
					// If conflict resolution fails, try ours strategy
					cp.runGitCommand("cherry-pick", "--abort")
					err = cp.runGitCommand("cherry-pick", "--strategy=recursive", "--strategy-option=ours", commitSHA)
					if err != nil {
						return fmt.Errorf("failed to cherry-pick commit %s with automatic conflict resolution: %w", commitSHA, err)
					}
				}
			} else {
				return fmt.Errorf("failed to cherry-pick commit %s: %w", commitSHA, err)
			}
		}
	}
	return nil
}

// pushBranch pushes the cherry-pick branch to the remote
func (cp *CherryPicker) pushBranch(branchName string) error {
	// Check if there are any changes to push
	if err := cp.runGitCommand("diff-index", "--quiet", "HEAD"); err != nil {
		// There are changes, proceed with push
		if err := cp.runGitCommand("push", "-u", "origin", branchName); err != nil {
			return fmt.Errorf("failed to push branch %s: %w", branchName, err)
		}
	} else {
		// No changes to push, this might indicate cherry-pick was a no-op
		// Still try to push the branch for consistency
		if err := cp.runGitCommand("push", "-u", "origin", branchName); err != nil {
			// If push fails and there are no changes, it might be expected
			return fmt.Errorf("failed to push branch %s (no changes detected): %w", branchName, err)
		}
	}
	return nil
}

// GetCherryPickBranchName returns the name of the branch that was created for cherry-pick
// Format: cherry-pick-{PR_ID}-to-{target_branch}-{short_sha}
func (cp *CherryPicker) GetCherryPickBranchName(commitSHA, targetBranch string) string {
	return git.GenerateCherryPickBranchName(cp.prID, commitSHA, targetBranch)
}
