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

package pr

import (
	"fmt"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/git"
	"github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GitLabPRCreator implements PRCreator interface for GitLab
type GitLabPRCreator struct {
	client     *gitlab.Client
	workingDir string
}

// NewGitLabPRCreator creates a new GitLab PR (Merge Request) creator
func NewGitLabPRCreator(baseURL, token string, workingDir string) (*GitLabPRCreator, error) {
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &GitLabPRCreator{
		client:     client,
		workingDir: workingDir,
	}, nil
}

func (g *GitLabPRCreator) getRepoID(repo *config.Repo) (int, error) {
	gitRepo, err := git.ParseRepoURL(repo.URL)
	if err != nil {
		return 0, fmt.Errorf("failed to parse repository URL: %w", err)
	}
	project, _, err := g.client.Projects.GetProject(gitRepo.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get project ID: %w", err)
	}
	return project.ID, nil
}

// CreatePR creates a merge request based on the update result
func (g *GitLabPRCreator) CreatePR(repo *config.Repo, sourceBranch string, option PRCreateOption) error {
	repoID, err := g.getRepoID(repo)
	if err != nil {
		return err
	}

	if len(option.UpdateSummary.SuccessfulUpdates) == 0 {
		return fmt.Errorf("no successful updates to create merge request for")
	}

	// Generate MR title and description
	title := generatePRTitle(&option.UpdateSummary)
	description := GeneratePRBody(&option.UpdateSummary)

	// Check if MR already exists
	existingMR, err := g.checkExistingMR(repoID, sourceBranch, repo.Branch)
	if err != nil {
		logrus.Warnf("Warning: failed to check existing merge request: %v", err)
		// Continue with creation attempt
	}

	if existingMR != nil {
		// Update existing MR
		logrus.Debugf("Found existing merge request !%d, updating...", existingMR.Number)
		return g.updateExistingMR(repoID, existingMR.Number, title, description, option)
	}

	// Create new merge request
	labels := append(gitlab.LabelOptions{}, option.Labels...)
	opts := &gitlab.CreateMergeRequestOptions{
		Title:        &title,
		Description:  &description,
		SourceBranch: &sourceBranch,
		TargetBranch: &repo.Branch,
		Labels:       &labels,
		AssigneeIDs:  g.getAssigneeIDs(option.Assignees),
	}

	mr, _, err := g.client.MergeRequests.CreateMergeRequest(repoID, opts)
	if err != nil {
		return fmt.Errorf("failed to create merge request: %w", err)
	}

	logrus.Debugf("Successfully created merge request !%d: %s", mr.IID, mr.WebURL)
	return nil
}

// GetPlatformType returns the type of platform
func (g *GitLabPRCreator) GetPlatformType() string {
	return "gitlab"
}

// checkExistingMR checks if a merge request already exists for the given source and target branches
func (g *GitLabPRCreator) checkExistingMR(repoID int, sourceBranch, targetBranch string) (*PRInfo, error) {
	expectState := "opened"
	opts := &gitlab.ListProjectMergeRequestsOptions{
		SourceBranch: &sourceBranch,
		TargetBranch: &targetBranch,
		State:        &expectState,
	}

	mrs, _, err := g.client.MergeRequests.ListProjectMergeRequests(repoID, opts)
	if err != nil {
		return nil, err
	}

	if len(mrs) == 0 {
		return nil, nil
	}

	// Return the first matching MR
	mr := mrs[0]
	return &PRInfo{
		Number: mr.IID,
		Title:  mr.Title,
		State:  string(mr.State),
	}, nil
}

// updateExistingMR updates an existing merge request with new title, description, and options
func (g *GitLabPRCreator) updateExistingMR(repoID, mrID int, title, description string, option PRCreateOption) error {
	labels := append(gitlab.LabelOptions{}, option.Labels...)
	opts := &gitlab.UpdateMergeRequestOptions{
		Title:       &title,
		Description: &description,
		Labels:      &labels,
		AssigneeIDs: g.getAssigneeIDs(option.Assignees),
	}

	_, _, err := g.client.MergeRequests.UpdateMergeRequest(repoID, mrID, opts)
	if err != nil {
		return fmt.Errorf("failed to update merge request: %w", err)
	}

	return nil
}

// getAssigneeIDs converts assignee usernames to user IDs
func (g *GitLabPRCreator) getAssigneeIDs(assignees []string) *[]int {
	if len(assignees) == 0 {
		return nil
	}

	var assigneeIDs []int
	for _, username := range assignees {
		users, _, err := g.client.Users.ListUsers(&gitlab.ListUsersOptions{
			Username: &username,
		})
		if err != nil && len(users) != 1 {
			logrus.Warnf("Failed to get user ID for username or the user is not found %s: %v", username, err)
			continue
		}
		assigneeIDs = append(assigneeIDs, users[0].ID)
	}

	return &assigneeIDs
}
