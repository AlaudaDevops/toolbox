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
)

// HandleLabel adds labels to the PR
func (h *PRHandler) HandleLabel(labels []string) error {
	if len(labels) == 0 {
		return fmt.Errorf("no labels specified")
	}

	h.Logger.Infof("Adding labels: %v", labels)

	// Get current labels for debugging
	currentLabels, err := h.client.GetLabels()
	if err != nil {
		h.Logger.Warnf("Failed to get current labels: %v", err)
	} else {
		h.Logger.Infof("Current labels: %v", currentLabels)
	}

	if err = h.client.AddLabels(labels); err != nil {
		return fmt.Errorf("failed to add labels: %w", err)
	}

	// Get updated labels for verification
	updatedLabels, err := h.client.GetLabels()
	if err != nil {
		h.Logger.Warnf("Failed to get updated labels: %v", err)
	} else {
		h.Logger.Infof("Updated labels: %v", updatedLabels)
	}

	// Create friendly message
	labelsStr := strings.Join(labels, ", ")
	message := fmt.Sprintf("🏷️ Labels `%s` have been added to this PR by @%s", labelsStr, h.config.CommentSender)

	return h.client.PostComment(message)
}
