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

// HandleLGTM processes LGTM votes and approves PR if threshold is met
func (h *PRHandler) HandleLGTM(_ []string) error {
	h.Info("Processing LGTM")

	// Check if the comment sender is the PR author (not in debug mode)
	if h.config.CommentSender == h.prSender && !h.config.Debug {
		// PR author is trying to LGTM their own PR - post informational message with status
		h.Infof("PR author %s attempted to LGTM their own PR", h.config.CommentSender)

		// Get current LGTM status to include in the message
		validVotes, lgtmUsers, err := h.GetLGTMVotes(h.config.LGTMPermissions, h.config.Debug)
		if err != nil {
			return fmt.Errorf("failed to get LGTM votes: %w", err)
		}

		// Create message with self-approval notice and current status
		selfApprovalMessage := fmt.Sprintf(messages.LGTMSelfApprovalTemplate, h.config.CommentSender)
		statusMessage := h.generateLGTMStatusMessage(validVotes, lgtmUsers, true)
		combinedMessage := selfApprovalMessage + "\n\n" + statusMessage

		return h.client.PostComment(combinedMessage)
	}

	// Not PR author (or in debug mode), proceed with normal LGTM logic

	// First, check if the current comment sender has permission to approve
	hasPermission, userPermission, err := h.client.CheckUserPermissions(h.config.CommentSender, h.config.LGTMPermissions)
	if err != nil {
		return fmt.Errorf("failed to check user permissions: %w", err)
	}

	// If user doesn't have permission, deny access
	if !hasPermission {
		message := fmt.Sprintf(messages.LGTMPermissionDeniedTemplate,
			h.config.CommentSender, userPermission, strings.Join(h.config.LGTMPermissions, ", "))

		return h.client.PostComment(message)
	}

	// User has permission, now get all LGTM votes to check if threshold is met
	validVotes, lgtmUsers, err := h.GetLGTMVotes(h.config.LGTMPermissions, h.config.Debug)
	if err != nil {
		return fmt.Errorf("failed to get LGTM votes: %w", err)
	}

	if validVotes >= h.config.LGTMThreshold {
		// Threshold met - approve the PR with comprehensive status message
		approvalMessage := h.generateLGTMStatusMessage(validVotes, lgtmUsers, false)
		if err := h.client.ApprovePR(approvalMessage); err != nil {
			return fmt.Errorf("failed to approve PR: %w", err)
		}

		h.Infof("✅ PR approved with LGTM votes from user %s (permission: %s)", h.config.CommentSender, userPermission)
		return nil
	}

	// Not enough LGTM votes - post status message with tip
	message := h.generateLGTMStatusMessage(validVotes, lgtmUsers, true)
	return h.client.PostComment(message)
}
