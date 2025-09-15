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
	"slices"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/cherrypick"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
)

// preferredMergeMethods defines the priority order for auto merge method selection
var preferredMergeMethods = []string{"rebase", "squash", "merge"}

// HandleMerge merges the PR if conditions are met
func (h *PRHandler) HandleMerge(args []string) error {
	// Validate user permissions
	userPerm, err := h.validateMergePermissions()
	if err != nil {
		return err
	}

	// Validate check runs status
	if err = h.validateCheckRunsStatus(); err != nil {
		return err
	}

	// Validate and process LGTM votes
	validVotes, lgtmUsers, err := h.validateAndProcessLGTMVotes(userPerm)
	if err != nil {
		return err
	}

	// Determine merge method from arguments
	method := h.determineMergeMethod(args)

	// Execute the merge
	if err := h.executeMerge(method); err != nil {
		return err
	}

	// Post success message
	return h.postMergeSuccessMessage(method, validVotes, lgtmUsers)
}

// validateMergePermissions checks if user has permission to merge
func (h *PRHandler) validateMergePermissions() (string, error) {
	hasPermission, userPerm, err := h.client.CheckUserPermissions(h.config.CommentSender, h.config.LGTMPermissions)
	if err != nil {
		return "", fmt.Errorf("failed to check user permissions: %w", err)
	}

	isPRCreator := strings.EqualFold(h.config.CommentSender, h.prSender)
	if !hasPermission && !isPRCreator {
		return userPerm, h.postInsufficientPermissionsMessage(userPerm)
	}

	return userPerm, nil
}

// postInsufficientPermissionsMessage posts error message for insufficient permissions
func (h *PRHandler) postInsufficientPermissionsMessage(userPerm string) error {
	message := fmt.Sprintf(messages.MergeInsufficientPermissionsTemplate,
		h.config.CommentSender, userPerm, strings.Join(h.config.LGTMPermissions, ", "),
		h.prSender, strings.Join(h.config.LGTMPermissions, ", "))

	if err := h.client.PostComment(message); err != nil {
		h.Errorf("Failed to post permission error comment: %v", err)
		return fmt.Errorf("insufficient permissions")
	}
	return &CommentedError{Err: fmt.Errorf("insufficient permissions")}
}

// validateCheckRunsStatus validates that all check runs are passing
func (h *PRHandler) validateCheckRunsStatus() error {
	allPassed, failedChecks, err := h.client.CheckRunsStatus()
	if err != nil {
		return fmt.Errorf("failed to check run status: %w", err)
	}

	if !allPassed {
		return h.postCheckRunsNotPassingMessage(failedChecks)
	}

	return nil
}

// postCheckRunsNotPassingMessage posts error message when checks are not passing
func (h *PRHandler) postCheckRunsNotPassingMessage(failedChecks []git.CheckRun) error {
	checkStatuses := h.convertToMessageCheckStatuses(failedChecks)
	statusTable := messages.BuildCheckStatusTable(checkStatuses)
	message := fmt.Sprintf(messages.MergeChecksNotPassingTemplate, statusTable)

	if err := h.client.PostComment(message); err != nil {
		h.Errorf("Failed to post check status comment: %v", err)
		return fmt.Errorf("checks not passing")
	}
	return &CommentedError{Err: fmt.Errorf("checks not passing")}
}

// convertToMessageCheckStatuses converts git.CheckRun to messages.CheckStatus
func (h *PRHandler) convertToMessageCheckStatuses(failedChecks []git.CheckRun) []messages.CheckStatus {
	var checkStatuses []messages.CheckStatus
	for _, check := range failedChecks {
		checkStatuses = append(checkStatuses, messages.CheckStatus{
			Name:       check.Name,
			Status:     check.Status,
			Conclusion: check.Conclusion,
			URL:        check.URL,
		})
	}
	return checkStatuses
}

// validateAndProcessLGTMVotes validates LGTM votes and processes admin/write user logic
func (h *PRHandler) validateAndProcessLGTMVotes(_ string) (int, map[string]string, error) {
	validVotes, lgtmUsers, err := h.GetLGTMVotes(h.config.LGTMPermissions, h.config.Debug)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get LGTM votes: %w", err)
	}

	if validVotes < h.config.LGTMThreshold {
		return 0, nil, h.postNotEnoughLGTMMessage(validVotes)
	}

	return validVotes, lgtmUsers, nil
}

// postNotEnoughLGTMMessage posts error message when there are not enough LGTM votes
func (h *PRHandler) postNotEnoughLGTMMessage(validVotes int) error {
	message := fmt.Sprintf(messages.MergeNotEnoughLGTMTemplate,
		validVotes, h.config.LGTMThreshold, h.config.LGTMThreshold-validVotes)

	if err := h.client.PostComment(message); err != nil {
		h.Errorf("Failed to post LGTM status comment: %v", err)
		return fmt.Errorf("not enough LGTM votes")
	}
	return &CommentedError{Err: fmt.Errorf("not enough LGTM votes")}
}

// determineMergeMethod determines merge method from arguments or uses default
func (h *PRHandler) determineMergeMethod(args []string) string {
	method := h.config.MergeMethod
	if len(args) > 0 {
		if args[0] == "merge" || args[0] == "squash" || args[0] == "rebase" || args[0] == "auto" {
			method = args[0]
		}
	}

	// If method is "auto", determine the best available method
	if method == "auto" {
		return h.selectAutoMergeMethod()
	}

	return method
}

// selectAutoMergeMethod selects the best available merge method automatically
// Priority: rebase > squash > merge
func (h *PRHandler) selectAutoMergeMethod() string {
	availableMethods, err := h.client.GetAvailableMergeMethods()
	if err != nil {
		h.Warnf("Failed to get available merge methods, falling back to squash: %v", err)
		return "squash"
	}

	h.Debugf("Available merge methods: %v", availableMethods)

	// Check preferred methods in priority order
	for _, preferred := range preferredMergeMethods {
		if slices.Contains(availableMethods, preferred) {
			h.Infof("Auto-selected merge method: %s", preferred)
			return preferred
		}
	}

	// Fallback to the first available method if none of the preferred methods are available
	if len(availableMethods) > 0 {
		method := availableMethods[0]
		h.Infof("Auto-selected fallback merge method: %s", method)
		return method
	}

	// Final fallback
	h.Warn("No available merge methods found, falling back to squash")
	return "squash"
}

// executeMerge performs the actual merge operation
func (h *PRHandler) executeMerge(method string) error {
	h.Infof("Merging PR with method: %s", method)

	// Special validation for rebase merge method
	if method == "rebase" {
		if err := h.validateRebaseMerge(); err != nil {
			return err
		}
	}

	if err := h.client.MergePR(method); err != nil {
		return h.postMergeFailedMessage(err)
	}

	return nil
}

// validateRebaseMerge checks if PR has only one commit for rebase merge
func (h *PRHandler) validateRebaseMerge() error {
	commits, err := h.client.GetCommits()
	if err != nil {
		h.Errorf("Failed to get commits for rebase validation: %v", err)
		// Allow merge to proceed if we can't get commit count
		return nil
	}

	if len(commits) > 1 {
		return h.postMultipleCommitsRebaseMessage(commits)
	}

	h.Infof("Rebase validation passed: PR has %d commit(s)", len(commits))
	return nil
}

// postMultipleCommitsRebaseMessage posts error message when rebase is not allowed due to multiple commits
func (h *PRHandler) postMultipleCommitsRebaseMessage(commits []git.Commit) error {
	// Build commits summary table
	commitsTable := "\n| SHA | Message |\n|-----|----------|\n"
	for _, commit := range commits {
		// Truncate commit message to first line only for readability
		firstLine := strings.Split(commit.Message, "\n")[0]
		if len(firstLine) > 60 {
			firstLine = firstLine[:57] + "..."
		}
		// Truncate SHA to 7 characters
		shortSHA := commit.SHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}
		commitsTable += fmt.Sprintf("| `%s` | %s |\n", shortSHA, firstLine)
	}

	message := fmt.Sprintf(messages.MergeMultipleCommitsRebaseTemplate, len(commits), commitsTable)

	if err := h.client.PostComment(message); err != nil {
		h.Errorf("Failed to post multiple commits rebase error comment: %v", err)
		return fmt.Errorf("rebase not allowed: PR has %d commits, only single commit allowed", len(commits))
	}
	return &CommentedError{Err: fmt.Errorf("rebase not allowed: PR has %d commits, only single commit allowed", len(commits))}
}

// postMergeFailedMessage posts error message when merge fails
func (h *PRHandler) postMergeFailedMessage(mergeErr error) error {
	message := fmt.Sprintf(messages.MergeFailedTemplate, h.config.PRNum, mergeErr)

	if postErr := h.client.PostComment(message); postErr != nil {
		h.Errorf("Failed to post merge error comment: %v", postErr)
		return fmt.Errorf("merge failed: %w", mergeErr)
	}
	return &CommentedError{Err: mergeErr}
}

// postMergeSuccessMessage posts success message and writes Tekton result
func (h *PRHandler) postMergeSuccessMessage(method string, validVotes int, lgtmUsers map[string]string) error {
	robotUsers := h.buildRobotUsersMap(lgtmUsers)
	usersTable := messages.BuildUsersTable(lgtmUsers, robotUsers)
	successMessage := fmt.Sprintf(messages.MergeSuccessTemplate,
		method, h.config.CommentSender, validVotes, h.config.LGTMThreshold, usersTable)

	// Check for cherry-pick comments and write result before merge
	hasCherryPickComments := h.checkForCherryPickComments()
	h.writeTektonResult("has-cherry-pick-comments", fmt.Sprintf("%t", hasCherryPickComments))

	h.writeTektonResult("merge-successful", "true")

	if err := h.client.PostComment(successMessage); err != nil {
		return err
	}

	return nil
}

// buildRobotUsersMap builds a map of robot users for filtering
func (h *PRHandler) buildRobotUsersMap(lgtmUsers map[string]string) map[string]bool {
	robotUsers := make(map[string]bool)
	for user := range lgtmUsers {
		if h.isRobotUser(user) {
			robotUsers[user] = true
		}
	}
	return robotUsers
}

// checkForCherryPickComments checks if there are any cherry-pick comments in the PR
func (h *PRHandler) checkForCherryPickComments() bool {
	// Get all comments from the PR
	comments, err := h.GetCommentsWithCache()
	if err != nil {
		h.Errorf("Failed to get comments for cherry-pick check: %v", err)
		return false
	}

	// Check if any comment contains cherry-pick commands
	for _, commentObj := range comments {
		// Split multi-line comments into individual command lines
		commandLines := comment.SplitCommandLines(commentObj.Body)

		// Check each command line for cherry-pick commands
		for _, cmdLine := range commandLines {
			if cherrypick.CherryPickPattern.MatchString(cmdLine) {
				h.Infof("Found cherry-pick command: %s", cmdLine)
				return true
			}
		}
	}

	return false
}

// HandlePostMergeCherryPick processes any cherry-pick commands found in PR comments after merge
// This is a public method that can be called independently for post-merge operations
func (h *PRHandler) HandlePostMergeCherryPick() error {
	h.Info("Checking for cherry-pick commands after merge")

	// First check if PR is closed (merged or closed)
	prInfo, err := h.client.GetPR()
	if err != nil {
		return fmt.Errorf("failed to get PR info: %w", err)
	}

	if prInfo.State == "open" {
		h.Info("PR is still open, skipping post-merge cherry-pick operations")
		return nil
	}

	h.Infof("PR is %s, proceeding with cherry-pick operations", prInfo.State)

	// Get all comments from the PR
	comments, err := h.GetCommentsWithCache()
	if err != nil {
		return fmt.Errorf("failed to get comments: %w", err)
	}

	// Find all cherry-pick commands
	cherryPickBranches := make(map[string]bool)

	for _, commentObj := range comments {
		// Split multi-line comments into individual command lines
		commandLines := comment.SplitCommandLines(commentObj.Body)

		// Check each command line for cherry-pick commands
		for _, cmdLine := range commandLines {
			matches := cherrypick.CherryPickPattern.FindStringSubmatch(cmdLine)
			if len(matches) > 1 {
				targetBranch := matches[1]
				cherryPickBranches[targetBranch] = true
				h.Infof("Found cherry-pick command for branch: %s", targetBranch)
			}
		}
	}

	// Perform cherry-picks for each target branch
	for targetBranch := range cherryPickBranches {
		h.Infof("Performing cherry-pick to branch: %s", targetBranch)
		if err := h.performCherryPick(targetBranch); err != nil {
			h.Errorf("Cherry-pick to %s failed: %v", targetBranch, err)
			// Continue with other cherry-picks even if one fails
			continue
		}
	}

	return nil
}
