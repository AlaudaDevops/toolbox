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

// Package pr provides pull request creation functionality
package pr

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

// GitHubPRCreator implements PRCreator interface for GitHub using gh CLI
type GitHubPRCreator struct {
	// workingDir is the working directory for git operations
	workingDir string
	// defaultLabels are labels to add to all PRs
}

// NewGitHubPRCreator creates a new GitHub PR creator
func NewGitHubPRCreator(workingDir string) *GitHubPRCreator {
	return &GitHubPRCreator{
		workingDir: workingDir,
	}
}

// CreatePR creates a pull request based on the update result
// If PR already exists, it will update the existing PR instead of creating a new one
func (g *GitHubPRCreator) CreatePR(sourceBranch, targetBranch string, option PRCreateOption) error {
	if len(option.UpdateSummary.SuccessfulUpdates) == 0 {
		return fmt.Errorf("no successful updates to create PR for")
	}

	// Generate PR title and body
	title := generatePRTitle(&option.UpdateSummary)
	body := GeneratePRBody(&option.UpdateSummary)

	// Check if PR already exists
	existingPR, err := g.checkExistingPR(sourceBranch, targetBranch)
	if err != nil {
		logrus.Warnf("Warning: failed to check existing PR: %v", err)
		// Continue with creation attempt
	}

	if existingPR != nil {
		// Update existing PR
		logrus.Debugf("Found existing PR #%d, updating...", existingPR.Number)
		return g.updateExistingPR(existingPR.Number, title, body, option)
	}

	// Create new pull request using gh CLI
	args := []string{
		"pr", "create",
		"--title", title,
		"--body", body,
		"--base", targetBranch,
		"--head", sourceBranch,
	}

	// Add labels if any
	if len(option.Labels) > 0 {
		args = append(args, "--label", strings.Join(option.Labels, ","))
	}

	// Add assignees if any
	if len(option.Assignees) > 0 {
		args = append(args, "--assignee", strings.Join(option.Assignees, ","))
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir = g.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create PR: %w, output: %s", err, string(output))
	}

	logrus.Debugf("Successfully created pull request:%s", string(output))
	return nil
}

// PRInfo represents basic information about a pull request
type PRInfo struct {
	// Number is the pull request number
	Number int `json:"number"`
	// Title is the pull request title
	Title string `json:"title"`
	// State is the pull request state (open, closed, merged)
	State string `json:"state"`
}

// checkExistingPR checks if a PR already exists for the given source and target branches
func (g *GitHubPRCreator) checkExistingPR(sourceBranch, targetBranch string) (*PRInfo, error) {
	// List open PRs with the same head branch
	args := []string{
		"pr", "list",
		"--head", sourceBranch,
		"--base", targetBranch,
		"--json", "number,title,state",
		"--limit", "1",
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir = g.workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list existing PRs: %w", err)
	}

	var prs []PRInfo
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PR list response: %w", err)
	}

	if len(prs) == 0 {
		return nil, nil // No existing PR found
	}

	return &prs[0], nil
}

// updateExistingPR updates an existing pull request
func (g *GitHubPRCreator) updateExistingPR(prNumber int, title, body string, option PRCreateOption) error {
	// Update PR with all properties in a single command
	args := []string{
		"pr", "edit", fmt.Sprintf("%d", prNumber),
		"--title", title,
		"--body", body,
	}

	// Add labels if provided
	if len(option.Labels) > 0 {
		args = append(args, "--add-label", strings.Join(option.Labels, ","))
	}

	// Add assignees if provided
	if len(option.Assignees) > 0 {
		args = append(args, "--add-assignee", strings.Join(option.Assignees, ","))
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir = g.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update PR #%d: %w, output: %s", prNumber, err, string(output))
	}

	logrus.Debugf("Successfully updated pull request #%d:%s", prNumber, string(output))
	return nil
}

// GetPlatformType returns the type of platform
func (g *GitHubPRCreator) GetPlatformType() string {
	return "github"
}

// Ensure GitHubPRCreator implements PRCreator interface
var _ PRCreator = (*GitHubPRCreator)(nil)
