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
	"context"
	"fmt"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/git"
	"github.com/google/go-github/v58/github"
	"github.com/sirupsen/logrus"
)

// GitHubPRCreator implements PRCreator interface for GitHub using the GitHub SDK
type GitHubPRCreator struct {
	// workingDir is the working directory for git operations
	workingDir string
	// client is the GitHub API client
	client *github.Client
}

// NewGitHubPRCreator creates a new GitHub PR creator
func NewGitHubPRCreator(baseURL, token string, workingDir string) *GitHubPRCreator {
	// Create GitHub client
	client := github.NewClient(nil).WithAuthToken(token)
	if baseURL != "" {
		var err error
		client, err = client.WithEnterpriseURLs(baseURL, baseURL+"/api/v3")
		if err != nil {
			logrus.Fatalf("Failed to set enterprise URLs: %v", err)
		}
	}

	return &GitHubPRCreator{
		workingDir: workingDir,
		client:     client,
	}
}

// CreatePR creates a pull request based on the update result
// If PR already exists, it will update the existing PR instead of creating a new one
func (g *GitHubPRCreator) CreatePR(repo *config.RepoConfig, sourceBranch string, option PRCreateOption) error {
	if len(option.UpdateSummary.SuccessfulUpdates) == 0 {
		return fmt.Errorf("no successful updates to create PR for")
	}

	gitRepo, err := git.ParseRepoURL(repo.URL)
	if err != nil {
		return fmt.Errorf("failed to parse repository URL: %w", err)
	}

	ctx := context.Background()

	// Generate PR title and body
	title := generatePRTitle(&option.UpdateSummary)
	body := GeneratePRBody(&option.UpdateSummary)

	// Check if PR already exists
	existingPR, err := g.checkExistingPR(gitRepo, sourceBranch, repo.Branch)
	if err != nil {
		logrus.Warnf("Warning: failed to check existing PR: %v", err)
		// Continue with creation attempt
	}

	if existingPR != nil {
		// Update existing PR
		logrus.Debugf("Found existing PR #%d, updating...", existingPR.Number)
		return g.updateExistingPR(gitRepo, existingPR.Number, title, body, option)
	}

	// Create new pull request
	newPR := &github.NewPullRequest{
		Title:               github.String(title),
		Body:                github.String(body),
		Head:                github.String(sourceBranch),
		Base:                github.String(repo.Branch),
		MaintainerCanModify: github.Bool(true),
	}

	pr, _, err := g.client.PullRequests.Create(ctx, gitRepo.Group, gitRepo.Repo, newPR)
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	// Add labels if any
	if len(option.Labels) > 0 {
		_, _, err = g.client.Issues.AddLabelsToIssue(ctx, gitRepo.Group, gitRepo.Repo, pr.GetNumber(), option.Labels)
		if err != nil {
			logrus.Warnf("Failed to add labels to PR #%d: %v", pr.GetNumber(), err)
		}
	}

	// Add assignees if any
	if len(option.Assignees) > 0 {
		_, _, err = g.client.Issues.AddAssignees(ctx, gitRepo.Group, gitRepo.Repo, pr.GetNumber(), option.Assignees)
		if err != nil {
			logrus.Warnf("Failed to add assignees to PR #%d: %v", pr.GetNumber(), err)
		}
	}

	logrus.Debugf("Successfully created pull request #%d", pr.GetNumber())
	return nil
}

// checkExistingPR checks if a PR already exists for the given source and target branches
func (g *GitHubPRCreator) checkExistingPR(gitRepo *git.Repository, sourceBranch, targetBranch string) (*PRInfo, error) {
	ctx := context.Background()

	// List open PRs with the same head branch
	opts := &github.PullRequestListOptions{
		Head:  sourceBranch,
		Base:  targetBranch,
		State: "open",
	}

	prs, _, err := g.client.PullRequests.List(ctx, gitRepo.Group, gitRepo.Repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing PRs: %w", err)
	}

	if len(prs) == 0 {
		return nil, nil // No existing PR found
	}

	return &PRInfo{
		Number: prs[0].GetNumber(),
		Title:  prs[0].GetTitle(),
		State:  prs[0].GetState(),
	}, nil
}

// updateExistingPR updates an existing pull request
func (g *GitHubPRCreator) updateExistingPR(gitRepo *git.Repository, prNumber int, title, body string, option PRCreateOption) error {
	ctx := context.Background()

	// Update PR
	pr := &github.PullRequest{
		Title: github.String(title),
		Body:  github.String(body),
	}

	_, _, err := g.client.PullRequests.Edit(ctx, gitRepo.Group, gitRepo.Repo, prNumber, pr)
	if err != nil {
		return fmt.Errorf("failed to update PR #%d: %w", prNumber, err)
	}

	// Add labels if any
	if len(option.Labels) > 0 {
		_, _, err = g.client.Issues.AddLabelsToIssue(ctx, gitRepo.Group, gitRepo.Repo, prNumber, option.Labels)
		if err != nil {
			logrus.Warnf("Failed to add labels to PR #%d: %v", prNumber, err)
		}
	}

	// Add assignees if any
	if len(option.Assignees) > 0 {
		_, _, err = g.client.Issues.AddAssignees(ctx, gitRepo.Group, gitRepo.Repo, prNumber, option.Assignees)
		if err != nil {
			logrus.Warnf("Failed to add assignees to PR #%d: %v", prNumber, err)
		}
	}

	logrus.Debugf("Successfully updated pull request #%d", prNumber)
	return nil
}

// GetPlatformType returns the type of platform
func (g *GitHubPRCreator) GetPlatformType() string {
	return "github"
}

// Ensure GitHubPRCreator implements PRCreator interface
var _ PRCreator = (*GitHubPRCreator)(nil)
