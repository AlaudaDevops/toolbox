/*
Copyright 2024 The AlaudaDevops Authors.

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

package jira

import (
	"context"
	"fmt"

	"io"

	"github.com/AlaudaDevops/toolbox/plugin-releaser/pkg/types"
	"github.com/andygrunwald/go-jira"
	"github.com/ankitpokhrel/jira-cli/pkg/jql"
)

// StatusCategoryDone represents the status category for completed issues
const StatusCategoryDone = "done"
const StatusCategoryCancelled = "cancelled"

// Client represents a Jira client that wraps the underlying Jira client
type Client struct {
	inner *jira.Client
}

// ClientOption represents a function that can modify a Client
type ClientOption func(*Client)

// NewClientWithConfig creates a new Jira client with the given configuration
func NewClientWithConfig(config *types.JiraConfig) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	tp := jira.BasicAuthTransport{
		Username: config.Username,
		Password: config.Password,
	}

	inner, err := jira.NewClient(tp.Client(), config.BaseURL)
	if err != nil {
		return nil, err
	}

	return &Client{inner: inner}, nil
}

// NewClient creates a new Jira client with the given credentials
// baseURL: The base URL of the Jira instance
// username: The username for authentication
// password: The password for authentication
func NewClient(baseURL string, username, password string) (*Client, error) {
	tp := jira.BasicAuthTransport{
		Username: username,
		Password: password,
	}

	inner, err := jira.NewClient(tp.Client(), baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{inner: inner}, nil
}

// FindOrCreateIssue searches for an existing issue matching the JQL query
// If no matching issue is found or all matching issues are completed, creates a new issue
// Returns the found or created issue
func (c *Client) FindOrCreateIssue(ctx context.Context, jql *jql.JQL, options ...IssueOption) (*jira.Issue, error) {
	searchOptions := &jira.SearchOptions{
		Fields: []string{"status"},
	}
	issues, resp, err := c.inner.Issue.SearchWithContext(ctx, jql.String(), searchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find issue: %s", c.handleError(resp, err))
	}

	for _, issue := range issues {
		if !c.IsCompletedIssue(issue) {
			return &issue, nil
		}
	}

	newIssue := newIssue(options...)
	issue, resp, err := c.inner.Issue.Create(newIssue)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %s", c.handleError(resp, err))
	}

	return issue, nil
}

func (c *Client) GetActiveSprint(ctx context.Context, boardID int) (*jira.Sprint, error) {
	sprints, resp, err := c.inner.Board.GetAllSprintsWithOptionsWithContext(ctx, boardID, &jira.GetAllSprintsOptions{
		State: "active",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get sprints: %s", c.handleError(resp, err))
	}

	if len(sprints.Values) == 0 {
		return nil, fmt.Errorf("no active sprint found")
	}

	return &sprints.Values[0], nil
}

// CreateOrUpdateIssue searches for an existing issue matching the JQL query
// If found, updates it with the provided options; otherwise creates a new issue
// Returns the created or updated issue
func (c *Client) CreateOrUpdateIssue(ctx context.Context, jql *jql.JQL, options ...IssueOption) (*jira.Issue, error) {
	searchOptions := &jira.SearchOptions{
		Fields: []string{"status"},
	}
	issues, resp, err := c.inner.Issue.SearchWithContext(ctx, jql.String(), searchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find issue: %s", c.handleError(resp, err))
	}

	var found *jira.Issue
	for _, issue := range issues {
		if !c.IsCompletedIssue(issue) {
			found = &issue
			break
		}
	}

	if found == nil {
		newIssue := newIssue(options...)
		issue, resp, err := c.inner.Issue.Create(newIssue)
		if err != nil {
			return nil, fmt.Errorf("failed to create issue: %s", c.handleError(resp, err))
		}

		return issue, nil
	}

	found.Fields.Status = nil
	for _, option := range options {
		option(found)
	}

	issue, resp, err := c.inner.Issue.Update(found)
	if err != nil {
		return nil, fmt.Errorf("failed to update issue: %s", c.handleError(resp, err))
	}

	return issue, nil
}

// IsCompletedIssue checks if an issue is in a completed state
func (c *Client) IsCompletedIssue(issue jira.Issue) bool {
	if issue.Fields == nil || issue.Fields.Status == nil {
		return true
	}

	issueStatus := issue.Fields.Status.StatusCategory.Key
	return issueStatus == StatusCategoryDone || issueStatus == StatusCategoryCancelled
}

// handleError processes error responses from Jira API calls
// Returns a formatted error message including the response body if available
func (c *Client) handleError(resp *jira.Response, err error) string {
	message := err.Error()
	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		message = fmt.Sprintf("%s resp: %s", message, string(body))
	}
	return message
}
