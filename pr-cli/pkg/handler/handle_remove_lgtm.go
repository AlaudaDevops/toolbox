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

// HandleRemoveLGTM processes the removal of LGTM by providing information about manual dismissal
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

	// User has permission, show current LGTM status and provide information
	validVotes, lgtmUsers, err := h.client.GetLGTMVotes(h.config.LGTMPermissions, h.config.Debug)
	if err != nil {
		return fmt.Errorf("failed to get LGTM votes: %w", err)
	}

	h.Logger.Infof("User %s requested remove LGTM information with permission: %s", h.config.CommentSender, userPermission)

	// Generate informational message about manual dismissal
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
