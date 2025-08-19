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
	"slices"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
	"github.com/sirupsen/logrus"
)

// PRHandler encapsulates Git client and configuration for PR operations
type PRHandler struct {
	*logrus.Logger
	client   git.GitClient  // Git platform client interface
	config   *config.Config // Configuration
	prSender string         // Pull request author username (retrieved from API)
}

// NewPRHandler creates a new PR handler with Git client and configuration
func NewPRHandler(logger *logrus.Logger, cfg *config.Config) (*PRHandler, error) {
	// Create Git client configuration
	clientConfig := &git.Config{
		Platform:      cfg.Platform,
		Token:         cfg.Token,
		BaseURL:       cfg.BaseURL,
		Owner:         cfg.Owner,
		Repo:          cfg.Repo,
		PRNum:         cfg.PRNum,
		CommentSender: cfg.CommentSender,
		SelfCheckName: cfg.SelfCheckName,
		RobotAccounts: cfg.RobotAccounts,
	}

	// Create platform-specific client
	client, err := git.CreateClient(logger, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create git client: %w", err)
	}

	// Get PR information to retrieve the author
	prInfo, err := client.GetPR()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR information: %w", err)
	}

	// Update client config with the actual PR sender
	clientConfig.PRSender = prInfo.Author

	// Recreate client with complete configuration
	client, err = git.CreateClient(logger, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to recreate git client: %w", err)
	}

	return &PRHandler{
		Logger:   logger,
		client:   client,
		config:   cfg,
		prSender: prInfo.Author,
	}, nil
}

// CheckPRStatus verifies if the PR is in the expected state
func (h *PRHandler) CheckPRStatus(expectedState string) error {
	return h.client.CheckPRStatus(expectedState)
}

// isRobotUser checks if the user is a robot user that should be excluded from LGTM status
func (h *PRHandler) isRobotUser(user string) bool {
	return slices.Contains(h.config.RobotAccounts, user)
}

// hasLGTMPermission checks if the given permission is in the LGTM permissions list
func (h *PRHandler) hasLGTMPermission(userPerm string) bool {
	return slices.Contains(h.config.LGTMPermissions, userPerm)
}

// generateLGTMStatusMessage generates a formatted LGTM status message with check runs status
func (h *PRHandler) generateLGTMStatusMessage(validVotes int, lgtmUsers map[string]string, includeThreshold bool) string {
	// Build robot users map for filtering
	robotUsers := make(map[string]bool)
	for user := range lgtmUsers {
		if h.isRobotUser(user) {
			robotUsers[user] = true
		}
	}

	// Build users table for status message
	usersTable := messages.BuildLGTMUsersTable(lgtmUsers, h.config.LGTMPermissions, robotUsers)

	// Check check runs status
	allPassed, failedChecks, err := h.client.CheckRunsStatus()
	if err != nil {
		h.Logger.Errorf("Failed to check run status: %v", err)
		// Continue without check runs status if there's an error
	}

	// Build base LGTM status message
	var message string
	if validVotes >= h.config.LGTMThreshold {
		message = fmt.Sprintf(messages.LGTMStatusReadyTemplate, validVotes, h.config.LGTMThreshold, usersTable)
	} else {
		// Not enough LGTM votes
		message = fmt.Sprintf(messages.LGTMStatusPendingTemplate, validVotes, h.config.LGTMThreshold, h.config.LGTMThreshold-validVotes, usersTable, strings.Join(h.config.LGTMPermissions, ", "))

		if includeThreshold {
			message += messages.LGTMStatusTipTemplate
		}
	}

	// Add check runs status information
	if err == nil {
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

		message += messages.BuildCheckRunsStatusSection(allPassed, checkStatuses)
	}

	return message
}

// GetComments retrieves all comments from the pull request
func (h *PRHandler) GetComments() ([]git.Comment, error) {
	return h.client.GetComments()
}

// PostComment posts a comment to the pull request
func (h *PRHandler) PostComment(message string) error {
	return h.client.PostComment(message)
}
