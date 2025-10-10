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
	"strconv"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
)

const (
	defaultIssueTitle  = "Dependency Dashboard"
	defaultIssueAuthor = "alaudaa-renovate[bot]"
)

// HandleCheckboxIssue updates unchecked checkboxes in the specified or default GitHub issue.
func (h *PRHandler) HandleCheckboxIssue(args []string) error {
	h.Infof("Handling checkbox-issue command with args: %v", args)

	opts, argErr := parseCheckboxIssueArgs(args)
	if argErr != nil {
		return h.postCommandFailure(argErr.template, argErr.label)
	}

	var (
		targetIssue *git.Issue
		err         error
	)

	if opts.hasIssueNumber {
		targetIssue, err = h.client.GetIssue(opts.issueNumber)
	} else {
		targetIssue, err = h.findIssue(opts.title, opts.author)
	}

	if err != nil {
		h.Warnf("Failed to locate issue for checkbox update: %v", err)
		return h.postCommandFailure(messages.CheckboxIssueNotFoundTemplate, "")
	}

	if targetIssue == nil {
		return h.postCommandFailure(messages.CheckboxIssueNotFoundTemplate, "")
	}

	if strings.TrimSpace(targetIssue.Body) == "" {
		_, issueFailureLabel := buildIssueLabels(targetIssue)
		return h.postCommandFailure(messages.CheckboxIssueBodyMissingTemplate, issueFailureLabel)
	}

	issueSuccessLabel, issueFailureLabel := buildIssueLabels(targetIssue)
	if !comment.HasUncheckedCheckbox(targetIssue.Body) {
		return h.postCommandFailure(messages.CheckboxAlreadyCheckedTemplate, issueFailureLabel)
	}

	updatedBody, toggled := comment.ToggleAllUncheckedCheckboxes(targetIssue.Body)
	if toggled == 0 {
		return h.postCommandFailure(messages.CheckboxAlreadyCheckedTemplate, issueFailureLabel)
	}

	if err := h.client.UpdateIssueBody(targetIssue.Number, updatedBody); err != nil {
		return fmt.Errorf("failed to update issue #%d body: %w", targetIssue.Number, err)
	}

	successMessage := fmt.Sprintf(messages.CheckboxUpdateSuccessTemplate, h.config.CommentSender, issueSuccessLabel)
	if err := h.client.PostComment(successMessage); err != nil {
		return fmt.Errorf("failed to post checkbox issue success comment: %w", err)
	}

	h.Infof("Updated %d checkbox entries for issue #%d", toggled, targetIssue.Number)
	return nil
}

type checkboxIssueOptions struct {
	issueNumber    int
	hasIssueNumber bool
	title          string
	author         string
}

type checkboxIssueArgError struct {
	template string
	label    string
}

func parseCheckboxIssueArgs(args []string) (checkboxIssueOptions, *checkboxIssueArgError) {
	opts := checkboxIssueOptions{
		title:  defaultIssueTitle,
		author: defaultIssueAuthor,
	}

	for i := 0; i < len(args); i++ {
		token := strings.TrimSpace(args[i])
		if token == "" {
			continue
		}

		lower := strings.ToLower(token)
		switch {
		case lower == "--title" || lower == "-t":
			i++
			if i >= len(args) {
				return opts, &checkboxIssueArgError{
					template: messages.CheckboxIssueInvalidOptionTemplate,
					label:    "--title",
				}
			}
			opts.title = trimQuotes(args[i])
		case strings.HasPrefix(lower, "--title="):
			opts.title = trimQuotes(token[len("--title="):])
		case lower == "--author" || lower == "-a":
			i++
			if i >= len(args) {
				return opts, &checkboxIssueArgError{
					template: messages.CheckboxIssueInvalidOptionTemplate,
					label:    "--author",
				}
			}
			opts.author = trimQuotes(args[i])
		case strings.HasPrefix(lower, "--author="):
			opts.author = trimQuotes(token[len("--author="):])
		case strings.HasPrefix(token, "--") || strings.HasPrefix(token, "-"):
			return opts, &checkboxIssueArgError{
				template: messages.CheckboxIssueInvalidOptionTemplate,
				label:    token,
			}
		default:
			if opts.hasIssueNumber {
				return opts, &checkboxIssueArgError{
					template: messages.CheckboxIssueInvalidOptionTemplate,
					label:    token,
				}
			}
			number, err := strconv.Atoi(token)
			if err != nil {
				return opts, &checkboxIssueArgError{
					template: messages.CheckboxIssueInvalidNumberTemplate,
					label:    token,
				}
			}
			opts.issueNumber = number
			opts.hasIssueNumber = true
		}
	}

	return opts, nil
}

func trimQuotes(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"'")
	return value
}

func (h *PRHandler) findIssue(title, author string) (*git.Issue, error) {
	searchOpts := git.IssueSearchOptions{
		Title:  title,
		Author: author,
		State:  "open",
		Sort:   "created",
		Order:  "asc",
	}
	return h.client.FindIssue(searchOpts)
}

func buildIssueLabels(issue *git.Issue) (string, string) {
	if issue == nil {
		return "the issue", " in the issue"
	}

	issueRef := fmt.Sprintf("issue #%d", issue.Number)
	if issue.Title != "" {
		issueRef = fmt.Sprintf("%s \"%s\"", issueRef, issue.Title)
	}

	success := issueRef
	failure := fmt.Sprintf(" in %s", issueRef)
	if issue.URL != "" {
		success = fmt.Sprintf("[%s](%s)", issueRef, issue.URL)
		failure = fmt.Sprintf(" in [%s](%s)", issueRef, issue.URL)
	}
	return success, failure
}
