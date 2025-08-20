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
func (h *PRHandler) HandleRemoveLGTM(_ []string) error {
	h.Logger.Info("Processing remove LGTM")

	// Check permissions first
	userPermission, hasPermission, err := h.validateRemoveLGTMPermissions()
	if err != nil {
		return err
	}
	if !hasPermission {
		return nil // Permission denied message already posted
	}

	// Get current LGTM state and process removal
	removalResult, err := h.processLGTMRemoval(userPermission)
	if err != nil {
		return err
	}

	// Generate and post final status
	return h.postRemoveLGTMStatus(removalResult)
}

// RemovalResult holds the result of LGTM removal processing
type RemovalResult struct {
	CurrentUserHasVote bool
	BeforeVotes        int
	AfterVotes         int
	UserPermission     string
}

// validateRemoveLGTMPermissions checks if the user has permission to remove LGTM
func (h *PRHandler) validateRemoveLGTMPermissions() (string, bool, error) {
	hasPermission, userPermission, err := h.client.CheckUserPermissions(h.config.CommentSender, h.config.LGTMPermissions)
	if err != nil {
		return "", false, fmt.Errorf("failed to check user permissions: %w", err)
	}

	if !hasPermission {
		message := fmt.Sprintf(messages.RemoveLGTMPermissionDeniedTemplate,
			h.config.CommentSender, userPermission, strings.Join(h.config.LGTMPermissions, ", "))
		err := h.client.PostComment(message)
		if err != nil {
			return "", false, err
		}
		return userPermission, false, nil
	}

	return userPermission, true, nil
}

// processLGTMRemoval handles the core logic of LGTM removal
func (h *PRHandler) processLGTMRemoval(userPermission string) (*RemovalResult, error) {
	// Get LGTM status before processing current /remove-lgtm command
	beforeRemoveVotes, beforeRemoveUsers, err := h.client.GetLGTMVotes(h.config.LGTMPermissions, h.config.Debug, h.config.CommentSender)
	if err != nil {
		return nil, fmt.Errorf("failed to get votes before current remove LGTM: %w", err)
	}

	// Check if current user has a vote and calculate vote counts
	result := h.calculateRemovalResult(beforeRemoveVotes, beforeRemoveUsers, userPermission)

	// Handle approval dismissal if needed
	h.handleApprovalDismissal(result)

	return result, nil
}

// calculateRemovalResult determines the impact of removing the current user's vote
func (h *PRHandler) calculateRemovalResult(beforeVotes int, beforeUsers map[string]string, userPermission string) *RemovalResult {
	// Check if current user has a vote
	_, currentUserHasVote := beforeUsers[h.config.CommentSender]

	// Calculate vote count after removal
	afterVotes := beforeVotes
	if currentUserHasVote {
		afterVotes = beforeVotes - 1
	}

	h.Logger.Debugf("Before remove LGTM: %d votes, after remove LGTM: %d votes (current user has vote: %t)",
		beforeVotes, afterVotes, currentUserHasVote)

	return &RemovalResult{
		CurrentUserHasVote: currentUserHasVote,
		BeforeVotes:        beforeVotes,
		AfterVotes:         afterVotes,
		UserPermission:     userPermission,
	}
}

// handleApprovalDismissal dismisses approval if removing vote would drop below threshold
func (h *PRHandler) handleApprovalDismissal(result *RemovalResult) {
	shouldDismiss := result.CurrentUserHasVote &&
		result.BeforeVotes >= h.config.LGTMThreshold &&
		result.AfterVotes < h.config.LGTMThreshold

	if shouldDismiss {
		h.dismissApproval(result.UserPermission)
	} else {
		h.logRemovalReason(result)
	}
}

// dismissApproval attempts to dismiss the PR approval
func (h *PRHandler) dismissApproval(userPermission string) {
	dismissMessage := fmt.Sprintf(messages.RemoveLGTMDismissTemplate, h.config.CommentSender)
	if err := h.client.DismissApprove(dismissMessage); err != nil {
		if !strings.Contains(err.Error(), "no approval review found") {
			h.Logger.Errorf("Failed to dismiss approval: %v", err)
		} else {
			h.Logger.Debugf("No approval review found to dismiss for user %s with permission: %s",
				h.config.CommentSender, userPermission)
		}
	} else {
		h.Logger.Infof("✅ User %s successfully dismissed approval (removing vote would drop below threshold) with permission: %s",
			h.config.CommentSender, userPermission)
	}
}

// logRemovalReason logs why the approval was not dismissed
func (h *PRHandler) logRemovalReason(result *RemovalResult) {
	if !result.CurrentUserHasVote {
		h.Logger.Infof("User %s requested remove LGTM but has no existing vote with permission: %s",
			h.config.CommentSender, result.UserPermission)
	} else {
		h.Logger.Infof("User %s requested remove LGTM but removing vote would not drop below threshold (before: %d, after: %d, threshold: %d) with permission: %s",
			h.config.CommentSender, result.BeforeVotes, result.AfterVotes, h.config.LGTMThreshold, result.UserPermission)
	}
}

// postRemoveLGTMStatus generates and posts the final status message
func (h *PRHandler) postRemoveLGTMStatus(_ *RemovalResult) error {
	// Get final LGTM status after processing all comments
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
