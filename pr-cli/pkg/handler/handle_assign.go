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

// HandleAssign assigns users as reviewers to the PR
func (h *PRHandler) HandleAssign(users []string) error {
	if len(users) == 0 {
		return fmt.Errorf("no users specified for assignment")
	}

	h.Infof("Assigning users: %v", users)

	// Get current reviewers for debugging
	currentReviewers, err := h.client.GetRequestedReviewers()
	if err != nil {
		h.Warnf("Failed to get current reviewers: %v", err)
	} else {
		h.Infof("Current requested reviewers: %v", currentReviewers)
	}

	if err = h.client.AssignReviewers(users); err != nil {
		return fmt.Errorf("failed to assign reviewers: %w", err)
	}

	// Get updated reviewers for verification
	updatedReviewers, err := h.client.GetRequestedReviewers()
	if err != nil {
		h.Warnf("Failed to get updated reviewers: %v", err)
	} else {
		h.Infof("Updated requested reviewers: %v", updatedReviewers)
	}

	// Create friendly message with @username format
	userMentions := messages.FormatUserMentions(users)
	greeting := "👋 Hello " + strings.Join(userMentions, ", ")
	message := fmt.Sprintf(messages.AssignmentGreetingTemplate, greeting, h.config.CommentSender)

	return h.client.PostComment(message)
}
