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

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
)

// CommentedError represents an error where a comment has already been posted to the PR
type CommentedError struct {
	Err error
}

func (e *CommentedError) Error() string {
	return e.Err.Error()
}

func (e *CommentedError) Unwrap() error {
	return e.Err
}

// HandleMerge merges the PR if conditions are met
func (h *PRHandler) HandleMerge(args []string) error {
	// Check user permissions - allow if user has write permission OR is the PR creator
	hasPermission, userPerm, err := h.client.CheckUserPermissions(h.config.CommentSender, h.config.LGTMPermissions)
	if err != nil {
		return fmt.Errorf("failed to check user permissions: %w", err)
	}

	// Allow merge if user has write permission OR is the PR creator
	isPRCreator := h.config.CommentSender == h.prSender
	if !hasPermission && !isPRCreator {
		message := fmt.Sprintf(messages.MergeInsufficientPermissionsTemplate,
			h.config.CommentSender, userPerm, strings.Join(h.config.LGTMPermissions, ", "), h.prSender, strings.Join(h.config.LGTMPermissions, ", "))

		if err = h.client.PostComment(message); err != nil {
			h.Logger.Errorf("Failed to post permission error comment: %v", err)
			return fmt.Errorf("insufficient permissions")
		}
		return &CommentedError{Err: fmt.Errorf("insufficient permissions")}
	}

	// Check if all checks are passing
	allPassed, failedChecks, err := h.client.CheckRunsStatus()
	if err != nil {
		return fmt.Errorf("failed to check run status: %w", err)
	}

	if !allPassed {
		// Convert failedChecks to our message type
		var checkStatuses []messages.CheckStatus
		for _, check := range failedChecks {
			checkStatuses = append(checkStatuses, messages.CheckStatus{
				Name:       check.Name,
				Status:     check.Status,
				Conclusion: check.Conclusion,
				URL:        check.URL,
			})
		}

		statusTable := messages.BuildCheckStatusTable(checkStatuses)
		message := fmt.Sprintf(messages.MergeChecksNotPassingTemplate, statusTable)

		if err = h.client.PostComment(message); err != nil {
			h.Logger.Errorf("Failed to post check status comment: %v", err)
			return fmt.Errorf("checks not passing")
		}
		return &CommentedError{Err: fmt.Errorf("checks not passing")}
	}

	// Check LGTM votes
	validVotes, lgtmUsers, err := h.client.GetLGTMVotes(h.config.LGTMPermissions, h.config.Debug)
	if err != nil {
		return fmt.Errorf("failed to get LGTM votes: %w", err)
	}

	// Handle admin/write user direct merge logic - use config instead of hardcoded values
	if h.prSender != h.config.CommentSender && h.hasLGTMPermission(userPerm) {
		// Check if merger already voted
		_, alreadyVoted := lgtmUsers[h.config.CommentSender]
		if h.config.LGTMThreshold == 1 || (validVotes >= h.config.LGTMThreshold-1 && !alreadyVoted) {
			if !alreadyVoted {
				lgtmUsers[h.config.CommentSender] = userPerm
				validVotes++
			}
		}
	}

	if validVotes < h.config.LGTMThreshold {
		message := fmt.Sprintf(messages.MergeNotEnoughLGTMTemplate,
			validVotes, h.config.LGTMThreshold, h.config.LGTMThreshold-validVotes)

		if err := h.client.PostComment(message); err != nil {
			h.Logger.Errorf("Failed to post LGTM status comment: %v", err)
			return fmt.Errorf("not enough LGTM votes")
		}
		return &CommentedError{Err: fmt.Errorf("not enough LGTM votes")}
	}

	// Determine merge method
	method := h.config.MergeMethod
	if len(args) > 0 {
		if args[0] == "merge" || args[0] == "squash" || args[0] == "rebase" {
			method = args[0]
		}
	}

	h.Logger.Infof("Merging PR with method: %s", method)

	// Perform the merge
	if err := h.client.MergePR(method); err != nil {
		message := fmt.Sprintf(messages.MergeFailedTemplate, h.config.PRNum, err)

		if postErr := h.client.PostComment(message); postErr != nil {
			h.Logger.Errorf("Failed to post merge error comment: %v", postErr)
			return fmt.Errorf("merge failed: %w", err)
		}
		return &CommentedError{Err: err}
	}

	// After successful merge, handle any pending cherry-pick operations
	if err := h.handlePostMergeCherryPicks(); err != nil {
		h.Logger.Errorf("Failed to handle post-merge cherry-picks: %v", err)
		// Don't fail the merge operation for cherry-pick errors
	}

	// Build robot users map for filtering
	robotUsers := make(map[string]bool)
	for user := range lgtmUsers {
		if h.isRobotUser(user) {
			robotUsers[user] = true
		}
	}

	// Build success message with users table
	usersTable := messages.BuildUsersTable(lgtmUsers, robotUsers)
	successMessage := fmt.Sprintf(messages.MergeSuccessTemplate, method, h.config.CommentSender, validVotes, h.config.LGTMThreshold, usersTable)

	return h.client.PostComment(successMessage)
}

var (
	cherryPickPattern = regexp.MustCompile(`^/cherry-pick\s+(\S+)`)
)

// handlePostMergeCherryPicks processes any cherry-pick commands found in PR comments after merge
func (h *PRHandler) handlePostMergeCherryPicks() error {
	h.Logger.Info("Checking for cherry-pick commands after merge")

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
