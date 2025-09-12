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

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
)

// HandleClose closes the PR without merging
func (h *PRHandler) HandleClose(args []string) error {
	h.Infof("Closing PR #%d", h.config.PRNum)

	// Check if PR is already closed
	prInfo, err := h.client.GetPR()
	if err != nil {
		return fmt.Errorf("failed to get PR information: %w", err)
	}

	if prInfo.State == "closed" {
		message := fmt.Sprintf("❌ **PR #%d is already closed**\n\nCannot close a PR that is already in closed state.", h.config.PRNum)
		return h.client.PostComment(message)
	}

	// Close the PR
	if err := h.client.ClosePR(); err != nil {
		return fmt.Errorf("failed to close PR: %w", err)
	}

	// Post success message
	message := fmt.Sprintf(messages.CloseSuccessTemplate, h.config.PRNum)
	return h.client.PostComment(message)
}
