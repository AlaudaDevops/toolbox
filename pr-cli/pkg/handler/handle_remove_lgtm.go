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

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
)

// HandleRemoveLGTM processes the removal of LGTM by dismissing approval if threshold was met
func (h *PRHandler) HandleRemoveLGTM() error {
	h.Logger.Info("Processing remove LGTM")

	// Check if the current comment sender has permission
	hasPermission, userPermission, err := h.client.CheckUserPermissions(h.config.CommentSender, h.config.LGTMPermissions)
	if err != nil {
		return fmt.Errorf("failed to check user permissions: %w", err)
	}

	if !hasPermission {
		// User doesn't have permission
		message := fmt.Sprintf(messages.RemoveLGTMPermissionDeniedTemplate,
			h.config.CommentSender, userPermission, strings.Join(h.config.LGTMPermissions, ", "))

		return h.client.PostComment(message)
	}

	// We need to check the LGTM status BEFORE processing the current /remove-lgtm command
	// Use the updated method that can ignore the current user's latest /remove-lgtm comment
	beforeRemoveVotes, beforeRemoveUsers, err := h.client.GetLGTMVotes(h.config.LGTMPermissions, h.config.Debug, h.config.CommentSender)
	if err != nil {
		return fmt.Errorf("failed to get votes before current remove LGTM: %w", err)
	}

	// Check if current user has a vote before this /remove-lgtm command
	currentUserHasVote := false
	if _, exists := beforeRemoveUsers[h.config.CommentSender]; exists {
		currentUserHasVote = true
	}

	// Calculate what the vote count would be after removing current user's vote
	afterRemoveVotes := beforeRemoveVotes
	if currentUserHasVote {
		afterRemoveVotes = beforeRemoveVotes - 1
	}
	h.Logger.Debugf("Before remove LGTM: %d votes, after remove LGTM: %d votes (current user has vote: %t)",
		beforeRemoveVotes, afterRemoveVotes, currentUserHasVote)

	// If current user has a vote and removing it would drop below threshold, dismiss approval
	if currentUserHasVote && beforeRemoveVotes >= h.config.LGTMThreshold && afterRemoveVotes < h.config.LGTMThreshold {
		dismissMessage := fmt.Sprintf(messages.RemoveLGTMDismissTemplate, h.config.CommentSender)
		if err = h.client.DismissApprove(dismissMessage); err != nil {
			// Check if the error is because no approval was found to dismiss
			if !strings.Contains(err.Error(), "no approval review found") {
				h.Logger.Errorf("Failed to dismiss approval: %v", err)
			} else {
				h.Logger.Debugf("No approval review found to dismiss for user %s with permission: %s", h.config.CommentSender, userPermission)
			}
		} else {
			h.Logger.Infof("✅ User %s successfully dismissed approval (removing vote would drop below threshold) with permission: %s", h.config.CommentSender, userPermission)
		}
	} else {
		if !currentUserHasVote {
			h.Logger.Infof("User %s requested remove LGTM but has no existing vote with permission: %s", h.config.CommentSender, userPermission)
		} else {
			h.Logger.Infof("User %s requested remove LGTM but removing vote would not drop below threshold (before: %d, after: %d, threshold: %d) with permission: %s",
				h.config.CommentSender, beforeRemoveVotes, afterRemoveVotes, h.config.LGTMThreshold, userPermission)
		}
	}

	// Get final LGTM status (after processing all comments including /remove-lgtm)
	validVotes, lgtmUsers, err := h.client.GetLGTMVotes(h.config.LGTMPermissions, h.config.Debug)
	if err != nil {
		return fmt.Errorf("failed to get final LGTM votes: %w", err)
	}

	// Generate status message
	statusMessage := fmt.Sprintf(messages.RemoveLGTMStatusTemplate,
		h.config.CommentSender, validVotes, h.config.LGTMThreshold, max(0, h.config.LGTMThreshold-validVotes))

	// Add the common LGTM status table
	statusMessage += h.generateLGTMStatusMessage(validVotes, lgtmUsers, false)

	return h.client.PostComment(statusMessage)
}

// Helper function to return max of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
