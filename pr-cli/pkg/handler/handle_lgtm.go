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
func (h *PRHandler) HandleLGTM() error {
	h.Logger.Info("Processing LGTM")

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
	validVotes, lgtmUsers, err := h.client.GetLGTMVotes(h.config.LGTMPermissions, h.config.Debug)
	if err != nil {
		return fmt.Errorf("failed to get LGTM votes: %w", err)
	}

	if validVotes >= h.config.LGTMThreshold {
		// Threshold met - approve the PR with comprehensive status message
		approvalMessage := h.generateLGTMStatusMessage(validVotes, lgtmUsers, false)
		if err := h.client.ApprovePR(approvalMessage); err != nil {
			return fmt.Errorf("failed to approve PR: %w", err)
		}

		h.Logger.Infof("✅ PR approved with LGTM votes from user %s (permission: %s)", h.config.CommentSender, userPermission)
		return nil
	}

	// Not enough LGTM votes - post status message with tip
	message := h.generateLGTMStatusMessage(validVotes, lgtmUsers, true)
	return h.client.PostComment(message)
}
