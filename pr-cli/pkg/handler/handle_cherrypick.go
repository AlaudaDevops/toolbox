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

package handler

import (
	"fmt"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git/cli"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
)

// HandleCherrypick handles the cherrypick command to create a cherrypick PR to a different branch
func (h *PRHandler) HandleCherrypick(args []string) error {
	// Validate arguments
	if len(args) == 0 {
		return h.client.PostComment(messages.CherryPickInvalidCommandTemplate)
	}

	targetBranch := args[0]

	// Check user permissions - allow if user has write permission OR is the PR creator
	hasPermission, userPerm, err := h.client.CheckUserPermissions(h.config.CommentSender, h.config.LGTMPermissions)
	if err != nil {
		return fmt.Errorf("failed to check user permissions: %w", err)
	}

	// Allow cherrypick if user has write permission OR is the PR creator
	isPRCreator := h.config.CommentSender == h.prSender
	if !hasPermission && !isPRCreator {
		message := fmt.Sprintf(messages.CherryPickInsufficientPermissionsTemplate,
			h.config.CommentSender, userPerm, strings.Join(h.config.LGTMPermissions, ", "), h.prSender, strings.Join(h.config.LGTMPermissions, ", "))

		return h.client.PostComment(message)
	}

	// Get PR information
	prInfo, err := h.client.GetPR()
	if err != nil {
		return fmt.Errorf("failed to get PR information: %w", err)
	}

	// Handle different PR states
	switch prInfo.State {
	case "open":
		// PR is open, schedule cherrypick for when it merges
		return h.scheduleCherrypick(prInfo, targetBranch)
	case "closed":
		if prInfo.Merged {
			// PR is merged (GitHub API reports merged PRs as "closed" with Merged=true)
			return h.createCherrypickPR(prInfo, targetBranch)
		} else {
			// PR is closed but not merged, allow cherrypick of the last commit
			return h.createCherrypickPR(prInfo, targetBranch)
		}
	default:
		// Unknown state, notify user
		message := fmt.Sprintf(messages.CherryPickClosedPRTemplate, h.config.PRNum, prInfo.State)
		return h.client.PostComment(message)
	}
}

// createCherrypickPR creates a cherrypick PR for merged or closed PRs
func (h *PRHandler) createCherrypickPR(prInfo *git.PullRequest, targetBranch string) error {
	// For merged or closed PRs, we would typically:
	// 1. Create a new branch based on the target branch
	// 2. Cherry-pick the commits to the new branch
	// 3. Create a new PR from the new branch to the target branch

	// Perform the actual cherry-pick operation
	h.Logger.Infof("Creating cherrypick for PR %d, state: %s, merged: %t", h.config.PRNum, prInfo.State, prInfo.Merged)
	return h.performCherryPick(targetBranch)
}

// scheduleCherrypick schedules a cherrypick for when the PR merges
func (h *PRHandler) scheduleCherrypick(_ *git.PullRequest, targetBranch string) error {
	message := fmt.Sprintf(messages.CherryPickScheduledTemplate, targetBranch)
	return h.client.PostComment(message)
}

// performCherryPick executes a cherry-pick operation to the specified branch
func (h *PRHandler) performCherryPick(targetBranch string) error {
	h.Logger.Infof("Starting cherry-pick operation to %s", targetBranch)

	// Get the current PR information to get the merged commit
	prInfo, err := h.client.GetPR()
	if err != nil {
		h.Logger.Errorf("Failed to get PR info for cherry-pick: %v", err)
		return h.postCherryPickError(targetBranch, fmt.Sprintf("Failed to get PR information: %v", err))
	}

	// Get commits from the PR to cherry-pick
	commits, err := h.client.GetCommits()
	if err != nil {
		h.Logger.Errorf("Failed to get commits for cherry-pick: %v", err)
		return h.postCherryPickError(targetBranch, fmt.Sprintf("Failed to get PR commits: %v", err))
	}

	if len(commits) == 0 {
		h.Logger.Errorf("No commits found in PR for cherry-pick")
		return h.postCherryPickError(targetBranch, "No commits found in PR")
	}

	// Check if we should use Git CLI for cherry-pick (more reliable)
	useGitCLI := h.config.UseGitCLIForCherryPick
	if useGitCLI {
		return h.performCherryPickWithGitCLI(commits, targetBranch, prInfo)
	}

	// Use the original API-based approach
	// Note: API-based cherry-pick has several limitations:
	// 1. Can't cherry-pick a specific file's changes
	// 2. Less reliable - may fail due to API rate limits or network issues
	// 3. Platform dependency - behavior may vary between GitHub and GitLab
	return h.performCherryPickWithAPI(commits, targetBranch, prInfo)
}

// performCherryPickWithGitCLI uses Git CLI for more reliable cherry-pick operations
func (h *PRHandler) performCherryPickWithGitCLI(commits []git.Commit, targetBranch string, prInfo *git.PullRequest) error {
	h.Logger.Infof("Using Git CLI for cherry-pick operation to %s with %d commits", targetBranch, len(commits))

	// Use the last commit SHA for branch naming (consistent with API approach)
	lastCommit := commits[len(commits)-1]

	// Create a platform-agnostic cherrypicker
	cherryPicker, err := cli.NewCherryPickerForPlatform(
		cli.Platform(h.config.Platform),
		h.config.Token,
		h.config.Owner,
		h.config.Repo,
		h.config.BaseURL,
		h.config.PRNum,
	)
	if err != nil {
		h.Logger.Errorf("Failed to create cherrypicker: %v", err)
		return h.postCherryPickError(targetBranch, fmt.Sprintf("Failed to create cherrypicker: %v", err))
	}

	// Perform the cherry-pick operation for all commits
	if err := cherryPicker.CherryPickCommits(commits, targetBranch); err != nil {
		h.Logger.Errorf("Failed to cherry-pick with Git CLI: %v", err)
		return h.postCherryPickError(targetBranch, fmt.Sprintf("Failed to cherry-pick with Git CLI: %v", err))
	}

	// Get the branch name that was created
	branchName := cherryPicker.GetCherryPickBranchName(lastCommit.SHA, targetBranch)

	// Create a PR for the cherry-pick using the platform-specific client
	title := fmt.Sprintf("[Cherry-pick] %s", prInfo.Title)
	body := fmt.Sprintf("Cherry-pick of PR #%d to %s\n\nOriginal PR: #%d\nRequested by: @%s",
		h.config.PRNum, targetBranch, h.config.PRNum, h.config.CommentSender)

	newPR, err := h.client.CreatePR(title, body, branchName, targetBranch)
	if err != nil {
		h.Logger.Errorf("Failed to create cherry-pick PR: %v", err)
		return h.postCherryPickError(targetBranch, fmt.Sprintf("Failed to create PR: %v", err))
	}

	h.Logger.Infof("Created cherry-pick PR #%d", newPR.Number)

	// Success message for Git CLI approach
	message := fmt.Sprintf(messages.CherryPickSuccessTemplate,
		h.config.PRNum, targetBranch, h.config.PRNum, newPR.Number, targetBranch, h.config.CommentSender, lastCommit.SHA)
	return h.client.PostComment(message)
}

// performCherryPickWithAPI uses the original API-based cherry-pick approach
func (h *PRHandler) performCherryPickWithAPI(commits []git.Commit, targetBranch string, prInfo *git.PullRequest) error {
	h.Logger.Infof("Using API method for cherry-pick operation to %s", targetBranch)

	// Generate a unique branch name for the cherry-pick
	// Use the last commit SHA for consistency with Git CLI approach
	lastCommit := commits[len(commits)-1]
	cherryPickBranch := h.generateCherryPickBranchName(lastCommit.SHA, targetBranch)

	// Create a new branch from the target branch
	if err := h.client.CreateBranch(cherryPickBranch, targetBranch); err != nil {
		h.Logger.Errorf("Failed to create cherry-pick branch: %v", err)
		return h.postCherryPickError(targetBranch, fmt.Sprintf("Failed to create branch %s: %v", cherryPickBranch, err))
	}

	h.Logger.Infof("Created cherry-pick branch: %s", cherryPickBranch)

	// Cherry-pick all commits from the PR to the new branch
	for _, commit := range commits {
		h.Logger.Infof("Cherry-picking commit %s", commit.SHA)
		if err := h.client.CherryPickCommit(commit.SHA, cherryPickBranch); err != nil {
			h.Logger.Errorf("Failed to cherry-pick commit %s: %v", commit.SHA, err)
			return h.postCherryPickError(targetBranch, fmt.Sprintf("Failed to cherry-pick commit %s: %v", commit.SHA, err))
		}
	}

	// Create a PR for the cherry-pick
	title := fmt.Sprintf("[Cherry-pick] %s", prInfo.Title)
	body := fmt.Sprintf("Cherry-pick of PR #%d to %s\n\nOriginal PR: #%d\nRequested by: @%s",
		h.config.PRNum, targetBranch, h.config.PRNum, h.config.CommentSender)

	newPR, err := h.client.CreatePR(title, body, cherryPickBranch, targetBranch)
	if err != nil {
		h.Logger.Errorf("Failed to create cherry-pick PR: %v", err)
		return h.postCherryPickError(targetBranch, fmt.Sprintf("Failed to create PR: %v", err))
	}

	h.Logger.Infof("Created cherry-pick PR #%d", newPR.Number)

	// Post success message
	message := fmt.Sprintf(messages.CherryPickSuccessTemplate,
		h.config.PRNum, targetBranch, h.config.PRNum, newPR.Number, targetBranch, h.config.CommentSender, commits[len(commits)-1].SHA)

	return h.client.PostComment(message)
}

// generateCherryPickBranchName generates a consistent branch name for cherry-pick operations
// Format: cherry-pick-{PR_ID}-to-{target_branch}-{short_sha}
func (h *PRHandler) generateCherryPickBranchName(commitSHA, targetBranch string) string {
	return git.GenerateCherryPickBranchName(h.config.PRNum, commitSHA, targetBranch)
}

// postCherryPickError posts an error message for cherry-pick failures
func (h *PRHandler) postCherryPickError(targetBranch, errorMsg string) error {
	message := fmt.Sprintf(messages.CherryPickErrorTemplate,
		h.config.PRNum, targetBranch, h.config.CommentSender, errorMsg)
	return h.client.PostComment(message)
}
