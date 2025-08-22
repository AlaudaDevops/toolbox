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
	"regexp"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
)

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

	isPRCreator := h.config.CommentSender == h.prSender
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
		h.Logger.Errorf("Failed to post permission error comment: %v", err)
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
		h.Logger.Errorf("Failed to post check status comment: %v", err)
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
func (h *PRHandler) validateAndProcessLGTMVotes(userPerm string) (int, map[string]string, error) {
	validVotes, lgtmUsers, err := h.client.GetLGTMVotes(h.config.LGTMPermissions, h.config.Debug)
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
		h.Logger.Errorf("Failed to post LGTM status comment: %v", err)
		return fmt.Errorf("not enough LGTM votes")
	}
	return &CommentedError{Err: fmt.Errorf("not enough LGTM votes")}
}

// determineMergeMethod determines merge method from arguments or uses default
func (h *PRHandler) determineMergeMethod(args []string) string {
	method := h.config.MergeMethod
	if len(args) > 0 {
		if args[0] == "merge" || args[0] == "squash" || args[0] == "rebase" {
			method = args[0]
		}
	}
	return method
}

// executeMerge performs the actual merge operation
func (h *PRHandler) executeMerge(method string) error {
	h.Logger.Infof("Merging PR with method: %s", method)

	if err := h.client.MergePR(method); err != nil {
		return h.postMergeFailedMessage(err)
	}

	return nil
}

// postMergeFailedMessage posts error message when merge fails
func (h *PRHandler) postMergeFailedMessage(mergeErr error) error {
	message := fmt.Sprintf(messages.MergeFailedTemplate, h.config.PRNum, mergeErr)

	if postErr := h.client.PostComment(message); postErr != nil {
		h.Logger.Errorf("Failed to post merge error comment: %v", postErr)
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
	comments, err := h.client.GetComments()
	if err != nil {
		h.Logger.Errorf("Failed to get comments for cherry-pick check: %v", err)
		return false
	}

	// Check if any comment contains cherry-pick commands
	for _, comment := range comments {
		if cherryPickPattern.MatchString(comment.Body) {
			h.Logger.Infof("Found cherry-pick comment: %s", comment.Body)
			return true
		}
	}

	return false
}

var (
	cherryPickPattern = regexp.MustCompile(`^/cherry-pick\s+(\S+)`)
)

// HandlePostMergeCherryPick processes any cherry-pick commands found in PR comments after merge
// This is a public method that can be called independently for post-merge operations
func (h *PRHandler) HandlePostMergeCherryPick() error {
	h.Logger.Info("Checking for cherry-pick commands after merge")

	// First check if PR is closed (merged or closed)
	prInfo, err := h.client.GetPR()
	if err != nil {
		return fmt.Errorf("failed to get PR info: %w", err)
	}

	if prInfo.State == "open" {
		h.Logger.Info("PR is still open, skipping post-merge cherry-pick operations")
		return nil
	}

	h.Logger.Infof("PR is %s, proceeding with cherry-pick operations", prInfo.State)

	// Get all comments from the PR
	comments, err := h.client.GetComments()
	if err != nil {
		return fmt.Errorf("failed to get comments: %w", err)
	}

	// Find all cherry-pick commands
	cherryPickBranches := make(map[string]bool)

	for _, comment := range comments {
		matches := cherryPickPattern.FindStringSubmatch(comment.Body)
		if len(matches) > 1 {
			targetBranch := matches[1]
			cherryPickBranches[targetBranch] = true
			h.Logger.Infof("Found cherry-pick command for branch: %s", targetBranch)
		}
	}

	// Perform cherry-picks for each target branch
	for targetBranch := range cherryPickBranches {
		h.Logger.Infof("Performing cherry-pick to branch: %s", targetBranch)
		if err := h.performCherryPick(targetBranch); err != nil {
			h.Logger.Errorf("Cherry-pick to %s failed: %v", targetBranch, err)
			// Continue with other cherry-picks even if one fails
			continue
		}
	}

	return nil
}
