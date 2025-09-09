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

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
	"github.com/sirupsen/logrus"
)

// PRHandler encapsulates Git client and configuration for PR operations
type PRHandler struct {
	*logrus.Logger
	client        git.GitClient  // Git platform client interface
	config        *config.Config // Configuration
	prSender      string         // Pull request author username (retrieved from API)
	commentsCache []git.Comment  // Cached comments to avoid multiple API calls
}

// NewPRHandler creates a new PR handler with Git client and configuration
func NewPRHandler(logger *logrus.Logger, cfg *config.Config) (*PRHandler, error) {
	// Create Git client configuration
	clientConfig := &git.Config{
		Platform:      cfg.Platform,
		Token:         cfg.Token,
		CommentToken:  cfg.CommentToken,
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

// NewPRHandlerWithClient creates a new PR handler with a provided client (for testing)
func NewPRHandlerWithClient(logger *logrus.Logger, cfg *config.Config, client git.GitClient, prSender string) (*PRHandler, error) {
	return &PRHandler{
		Logger:   logger,
		client:   client,
		config:   cfg,
		prSender: prSender,
	}, nil
}

// CheckPRStatus verifies if the PR is in the expected state
func (h *PRHandler) CheckPRStatus(expectedState string) error {
	return h.client.CheckPRStatus(expectedState)
}

// isRobotUser checks if the user is a robot user that should be excluded from LGTM status
func (h *PRHandler) isRobotUser(user string) bool {
	return h.config.IsRobotUser(user)
}

// generateLGTMStatusMessage generates a formatted LGTM status message with check runs status
func (h *PRHandler) generateLGTMStatusMessage(validVotes int, lgtmUsers map[string]string, includeThreshold bool) string {
	// Check check runs status
	allPassed, failedChecks, err := h.client.CheckRunsStatus()
	if err != nil {
		h.Errorf("Failed to check run status: %v", err)
		// Continue without check runs status if there's an error
		allPassed = true
		failedChecks = nil
	}

	// Convert failedChecks to message type
	var checkStatuses []messages.CheckStatus
	for _, check := range failedChecks {
		checkStatuses = append(checkStatuses, messages.CheckStatus{
			Name:       check.Name,
			Status:     check.Status,
			Conclusion: check.Conclusion,
			URL:        check.URL,
		})
	}

	opts := messages.LGTMStatusOptions{
		ValidVotes:       validVotes,
		LGTMUsers:        lgtmUsers,
		LGTMThreshold:    h.config.LGTMThreshold,
		LGTMPermissions:  h.config.LGTMPermissions,
		RobotAccounts:    h.config.RobotAccounts,
		IncludeThreshold: includeThreshold,
		ChecksPassed:     allPassed,
		FailedChecks:     checkStatuses,
	}

	return messages.BuildLGTMStatusMessage(opts)
}

// GetCommentsWithCache retrieves all comments from the pull request with caching
func (h *PRHandler) GetCommentsWithCache() ([]git.Comment, error) {
	if h.commentsCache != nil {
		h.Debugf("Using cached comments (%d comments)", len(h.commentsCache))
		return h.commentsCache, nil
	}

	comments, err := h.client.GetComments()
	if err != nil {
		return nil, err
	}

	h.commentsCache = comments
	h.Debugf("Cached comments (%d comments)", len(h.commentsCache))
	return comments, nil
}

// GetLGTMVotes retrieves and validates LGTM votes with optimized comments caching
func (h *PRHandler) GetLGTMVotes(requiredPerms []string, debugMode bool, ignoreUserRemove ...string) (int, map[string]string, error) {
	// Use cached comments for better performance
	comments, err := h.GetCommentsWithCache()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get comments: %w", err)
	}

	// Use the optimized method that accepts pre-fetched comments
	return h.client.GetLGTMVotes(comments, requiredPerms, debugMode, ignoreUserRemove...)
}

// PostComment posts a comment to the pull request
func (h *PRHandler) PostComment(message string) error {
	return h.client.PostComment(message)
}
