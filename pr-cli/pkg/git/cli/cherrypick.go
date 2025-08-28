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
	"regexp"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/sirupsen/logrus"
)

// cleanBranchNameForTempDir removes path separators from branch names for use in temp directory names
func cleanBranchNameForTempDir(branchName string) string {
	clean := strings.ReplaceAll(branchName, "/", "-")
	return strings.ReplaceAll(clean, "\\", "-")
}

// sanitizeErrorMessage removes sensitive information (tokens, credentials) from error messages
func sanitizeErrorMessage(message string) string {
	// First handle OAuth2 tokens in URLs (oauth2:token@host) before general URL pattern
	oauth2Pattern := regexp.MustCompile(`oauth2:[^@\s]+@`)
	message = oauth2Pattern.ReplaceAllString(message, "oauth2:[TOKEN_REDACTED]@")

	// Pattern to match GitHub tokens (ghp_, gho_, ghu_, ghs_, ghr_, ghco_)
	// GitHub tokens can vary in length but are typically 36-40 characters after the prefix
	githubTokenPattern := regexp.MustCompile(`gh[pousr][_a-zA-Z0-9]+`)
	message = githubTokenPattern.ReplaceAllString(message, "[TOKEN_REDACTED]")

	// Pattern to match GitLab tokens (glpat-)
	gitlabTokenPattern := regexp.MustCompile(`glpat-[a-zA-Z0-9_-]{20,}`)
	message = gitlabTokenPattern.ReplaceAllString(message, "[TOKEN_REDACTED]")

	// Don't replace any more credentials in URLs since we've already handled oauth2 format
	// The oauth2 format is now the standard for both GitHub and GitLab

	return message
}

// CherryPicker handles Git CLI cherry-pick operations
type CherryPicker struct {
	logger     *logrus.Logger
	repoURL    string
	token      string
	owner      string
	repo       string
	workingDir string
	currentDir string // Store original directory for cleanup
	prID       int    // Original PR ID
}

// NewCherryPicker creates a new CherryPicker instance
func NewCherryPicker(logger *logrus.Logger, repoURL, token, owner, repo string, prID int) *CherryPicker {
	currentDir, _ := os.Getwd()
	return &CherryPicker{
		logger:     logger,
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
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("cherrypick-%s-%s-x", cp.repo, cleanBranchNameForTempDir(targetBranch)))
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
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("cherrypick-%s-%s-x", cp.repo, cleanBranchNameForTempDir(targetBranch)))
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
	cp.logger.Debugf("Cloning repository %s to %s", sanitizeErrorMessage(cp.repoURL), cp.workingDir)
	// First, try cloning with token in URL (existing method)
	cloneCmd := exec.Command("git", "clone", cp.repoURL, cp.workingDir)

	// Set environment variables for Git authentication as a fallback
	env := os.Environ()
	env = append(env, "GIT_ASKPASS=echo")
	env = append(env, "GIT_USERNAME=token")
	env = append(env, fmt.Sprintf("GIT_PASSWORD=%s", cp.token))
	cloneCmd.Env = env

	output, err := cloneCmd.CombinedOutput()
	if err != nil {
		sanitizedOutput := sanitizeErrorMessage(string(output))

		// Try alternative clone method using git credential helper
		if err := cp.cloneWithCredentialHelper(); err != nil {
			return fmt.Errorf("failed to clone repository with both methods: original error: %w, output: %s", err, sanitizedOutput)
		}
		return nil
	} else {
		cp.logger.Debugf("Repository cloned successfully: %s", sanitizeErrorMessage(cp.repoURL))
	}
	return nil
}

// cloneWithCredentialHelper attempts to clone using environment variables for authentication
func (cp *CherryPicker) cloneWithCredentialHelper() error {
	// Extract the repository URL without credentials
	repoURLWithoutCreds := cp.getRepositoryURLWithoutCredentials()

	// Use a more direct approach with environment variables
	cloneCmd := exec.Command("git", "clone", repoURLWithoutCreds, cp.workingDir)

	// Set up environment for Git authentication
	env := os.Environ()

	// Method 1: Use GIT_ASKPASS with echo for non-interactive authentication
	env = append(env, "GIT_ASKPASS=echo")
	env = append(env, "GIT_TERMINAL_PROMPT=0")

	// Method 2: Use credential.helper.username and password via config
	// This is more reliable than askpass in some environments
	host := cp.getHostFromURL()

	// For GitHub/GitLab, username should be "token" or "oauth2"
	username := "token"
	if strings.Contains(cp.repoURL, "oauth2:") {
		username = "oauth2"
	}

	// Set credential environment variables
	env = append(env, fmt.Sprintf("GIT_CONFIG_KEY_0=credential.https://%s.username", host))
	env = append(env, fmt.Sprintf("GIT_CONFIG_VALUE_0=%s", username))
	env = append(env, fmt.Sprintf("GIT_CONFIG_KEY_1=credential.https://%s.password", host))
	env = append(env, fmt.Sprintf("GIT_CONFIG_VALUE_1=%s", cp.token))
	env = append(env, "GIT_CONFIG_COUNT=2")

	cloneCmd.Env = env

	output, err := cloneCmd.CombinedOutput()
	if err != nil {
		sanitizedOutput := sanitizeErrorMessage(string(output))
		return fmt.Errorf("credential helper clone failed: %w, output: %s", err, sanitizedOutput)
	} else {
		cp.logger.Debugf("Repository cloned successfully: %s", sanitizeErrorMessage(cp.repoURL))
	}

	return nil
}

// getRepositoryURLWithoutCredentials returns the repository URL without embedded credentials
func (cp *CherryPicker) getRepositoryURLWithoutCredentials() string {
	// Remove credentials from URL
	if strings.Contains(cp.repoURL, "@") {
		parts := strings.Split(cp.repoURL, "@")
		if len(parts) >= 2 {
			// Reconstruct URL: https:// + host/path
			return "https://" + strings.Join(parts[1:], "@")
		}
	}
	return cp.repoURL
}

// getHostFromURL extracts the host from the repository URL
func (cp *CherryPicker) getHostFromURL() string {
	url := cp.getRepositoryURLWithoutCredentials()
	// Remove https:// or http:// prefix
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	// Get host part (before first /)
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return "github.com" // fallback
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
		sanitizedOutput := sanitizeErrorMessage(string(output))
		return fmt.Errorf("git %s failed: %w, output: %s", strings.Join(args, " "), err, sanitizedOutput)
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
			if strings.Contains(err.Error(), "CONFLICT") || strings.Contains(err.Error(), "conflict") ||
				strings.Contains(err.Error(), "unmerged files") {
				cp.logger.Errorf("❌ CHERRY-PICK CONFLICT DETECTED for commit %s", commitSHA)
				cp.logger.Errorf("💡 This may be caused by: fork PR, merge conflicts, or missing dependencies")
				cp.logger.Infof("🔧 Attempting automatic conflict resolution...")

				// Abort the current cherry-pick
				cp.runGitCommand("cherry-pick", "--abort")

				// Try with strategy options for automatic conflict resolution
				err = cp.runGitCommand("cherry-pick", "--strategy=recursive", "--strategy-option=theirs", commitSHA)
				if err != nil {
					// If conflict resolution fails, try ours strategy
					cp.runGitCommand("cherry-pick", "--abort")
					err = cp.runGitCommand("cherry-pick", "--strategy=recursive", "--strategy-option=ours", commitSHA)
					if err != nil {
						// Check if this is an empty commit error
						if cp.isEmptyCommitError(err) {
							cp.logger.Warnf("Cherry-pick resulted in empty commit for %s, skipping with --allow-empty", commitSHA)
							// Try to skip the empty commit
							if skipErr := cp.runGitCommand("cherry-pick", "--skip"); skipErr != nil {
								cp.logger.Warnf("Failed to skip empty commit, continuing anyway: %v", skipErr)
							}
							return nil
						}
						cp.logger.Errorf("❌ CHERRY-PICK FAILED: All conflict resolution strategies failed for commit %s", commitSHA)
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

// isEmptyCommitError checks if the error is related to empty commits
func (cp *CherryPicker) isEmptyCommitError(err error) bool {
	errorMsg := strings.ToLower(err.Error())
	return strings.Contains(errorMsg, "empty") ||
		strings.Contains(errorMsg, "nothing to commit") ||
		strings.Contains(errorMsg, "the previous cherry-pick is now empty")
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
