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

package cmd

import (
	"fmt"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
	"github.com/sirupsen/logrus"
)

// commandsSkipPRStatusCheck defines commands that can work with closed PRs
// and don't require the PR to be in "open" state
var commandsSkipPRStatusCheck = map[string]bool{
	"cherry-pick": true,
	"cherrypick":  true,
	// Add other commands here that should work with closed PRs
}

// shouldSkipPRStatusCheck returns true if the command can work with closed PRs
func (p *PROption) shouldSkipPRStatusCheck(command string) bool {
	return commandsSkipPRStatusCheck[command] || handler.IsBuiltInCommand(command)
}

// validateCommentSender verifies that the comment-sender actually posted the trigger-comment
func (p *PROption) validateCommentSender(prHandler *handler.PRHandler) error {
	// Get all comments from the PR
	comments, err := prHandler.GetCommentsWithCache()
	if err != nil {
		return fmt.Errorf("failed to get PR comments: %w", err)
	}

	// Normalize the trigger comment for comparison
	normalizedTrigger := comment.Normalize(p.Config.TriggerComment)

	// Check if any comment from the comment-sender contains the trigger-comment
	found := false
	for _, commentObj := range comments {
		if strings.EqualFold(commentObj.User.Login, p.Config.CommentSender) {
			// Normalize the comment body for comparison
			normalizedCommentBody := comment.Normalize(commentObj.Body)

			// Check for exact match or if the normalized trigger is contained in the normalized comment
			if normalizedCommentBody == normalizedTrigger || strings.Contains(normalizedCommentBody, normalizedTrigger) {
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Errorf("comment sender '%s' did not post a comment containing '%s'", p.Config.CommentSender, normalizedTrigger)
	}

	p.Infof("Comment sender validation passed: %s posted a comment containing the trigger", p.Config.CommentSender)
	return nil
}

// initialize initializes and validates the PROption configuration
func (p *PROption) initialize() error {
	// Read all values from viper (which includes environment variables)
	p.readAllFromViper()

	// Parse string fields into config
	if err := p.parseStringFields(); err != nil {
		return fmt.Errorf("failed to parse CLI fields: %w", err)
	}

	// Set log level based on verbose flag
	if p.Config.Verbose {
		p.SetLevel(logrus.DebugLevel)
		p.Debug("Verbose logging enabled")
	} else {
		p.SetLevel(logrus.InfoLevel)
	}

	// Validate configuration
	if err := p.Config.Validate(); err != nil {
		return err
		// // return fmt.Errorf("configuration validation failed: %w", err)
	}

	return nil
}
