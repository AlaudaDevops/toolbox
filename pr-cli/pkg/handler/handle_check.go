/*
Copyright 2025 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
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
)

// HandleCheck displays current LGTM status or executes sub-commands
func (h *PRHandler) HandleCheck(args []string) error {
	// Parse sub-commands with arguments
	subCommands := h.parseSubCommands(args)
	if len(subCommands) == 0 {
		// No valid sub-commands found, execute original check
		return h.handleOriginalCheck()
	}

	// Execute sub-commands with check-specific header
	return h.handleSubCommands(subCommands, "**Check Command Results:**")
}

// handleOriginalCheck executes the original check logic
func (h *PRHandler) handleOriginalCheck() error {
	h.Logger.Info("Checking LGTM status and check runs status")

	// Get all LGTM votes to check current status
	validVotes, lgtmUsers, err := h.client.GetLGTMVotes(h.config.LGTMPermissions, h.config.Debug)
	if err != nil {
		return fmt.Errorf("failed to get LGTM votes: %w", err)
	}

	// Use the common method to generate status message (now includes check runs status)
	message := h.generateLGTMStatusMessage(validVotes, lgtmUsers, true)

	return h.client.PostComment(message)
}
