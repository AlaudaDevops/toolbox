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

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
)

// HandleCheckbox updates unchecked checkboxes in the PR description.
func (h *PRHandler) HandleCheckbox(args []string) error {
	h.Infof("Handling checkbox command with args: %v", args)

	prInfo, prErr := h.client.GetPR()
	if prErr != nil {
		h.Warnf("Failed to fetch pull request before checkbox update: %v", prErr)
	}

	if prInfo == nil || strings.TrimSpace(prInfo.Body) == "" {
		return h.postCommandFailure(messages.CheckboxDescriptionNotFoundTemplate, "")
	}

	prSuccessLabel, prFailureLabel := buildPRLabels(prInfo)
	if !comment.HasUncheckedCheckbox(prInfo.Body) {
		return h.postCommandFailure(messages.CheckboxAlreadyCheckedTemplate, prFailureLabel)
	}

	updatedBody, toggled := comment.ToggleAllUncheckedCheckboxes(prInfo.Body)
	if toggled == 0 {
		return h.postCommandFailure(messages.CheckboxAlreadyCheckedTemplate, prFailureLabel)
	}

	if err := h.client.UpdatePRBody(updatedBody); err != nil {
		return fmt.Errorf("failed to update pull request body: %w", err)
	}

	successMessage := fmt.Sprintf(messages.CheckboxUpdateSuccessTemplate, h.config.CommentSender, prSuccessLabel)
	if err := h.client.PostComment(successMessage); err != nil {
		return fmt.Errorf("failed to post checkbox success comment: %w", err)
	}

	h.Infof("Updated %d checkbox entries for pull request description", toggled)
	return nil
}

func (h *PRHandler) postCommandFailure(template, label string) error {
	message := template
	if strings.Contains(template, "%s") {
		message = fmt.Sprintf(template, label)
	}
	if err := h.client.PostComment(message); err != nil {
		return fmt.Errorf("failed to post checkbox status comment: %w", err)
	}
	return &CommentedError{Err: fmt.Errorf("checkbox command failed: %s", message)}
}

func buildPRLabels(pr *git.PullRequest) (string, string) {
	if pr == nil || pr.URL == "" {
		return "the pull request description", " in the pull request description"
	}
	return fmt.Sprintf("[the pull request description](%s)", pr.URL),
		fmt.Sprintf(" in [the pull request description](%s)", pr.URL)
}
