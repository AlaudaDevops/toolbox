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
	"strings"

	"io"

	"github.com/andygrunwald/go-jira"
	"github.com/ankitpokhrel/jira-cli/pkg/jql"
)

// StatusCategoryDone represents the status category for completed issues
const StatusCategoryDone = "done"

// Client represents a Jira client that wraps the underlying Jira client
type Client struct {
	inner *jira.Client
}

// ClientOption represents a function that can modify a Client
type ClientOption func(*Client)

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

// LinkIssue creates a relationship between a parent and child issue
// If the link already exists, does nothing
func (c *Client) LinkIssue(ctx context.Context, parent *jira.Issue, child *jira.Issue) error {
	if child.Fields != nil {
		for _, link := range child.Fields.IssueLinks {
			if link.InwardIssue.ID == parent.ID {
				return nil
			}
		}
	}

	issueLink := &jira.IssueLink{
		Type: jira.IssueLinkType{
			Name: "Relate",
		},
		OutwardIssue: &jira.Issue{
			ID: child.ID,
		},
		InwardIssue: &jira.Issue{
			ID: parent.ID,
		},
	}

	resp, err := c.inner.Issue.AddLinkWithContext(ctx, issueLink)
	if err != nil {
		return fmt.Errorf(" failed to link issue: %s", c.handleError(resp, err))
	}

	return nil
}

// CompleteIssue transitions all issues matching the JQL query to a completed state
// Also completes any linked child issues
func (c *Client) CompleteIssue(ctx context.Context, jql *jql.JQL) error {
	options := &jira.SearchOptions{
		Fields: []string{"issuelinks", "status", "reporter"},
	}
	issues, resp, err := c.inner.Issue.Search(jql.String(), options)
	if err != nil {
		return fmt.Errorf("failed to create issue: %s", c.handleError(resp, err))
	}

	if len(issues) == 0 {
		return nil
	}

	for _, issue := range issues {
		if c.IsCompletedIssue(issue) {
			continue
		}

		transitionID, err := c.findCompletionTransitionID(ctx, issue)
		if err != nil {
			return err
		}

		if transitionID == "" {
			return fmt.Errorf("no transition ID found for issue %s", c.handleError(resp, err))
		}

		for _, link := range issue.Fields.IssueLinks {
			if c.IsCompletedIssue(*link.OutwardIssue) {
				continue
			}

			resp, err = c.inner.Issue.DoTransition(link.OutwardIssue.ID, transitionID)
			if err != nil {
				return fmt.Errorf("failed to complete child issue: %s, error: %s", link.OutwardIssue.ID, c.handleError(resp, err))
			}
		}

		resp, err = c.inner.Issue.DoTransition(issue.ID, transitionID)
		if err != nil {
			return fmt.Errorf("failed to complete parent issue: %s, error: %s", issue.ID, c.handleError(resp, err))
		}
	}

	return nil
}

// GetComponentVersion retrieves the version of a component in a project
// Returns the version name if found, or an error if not found
func (c *Client) GetComponentVersion(ctx context.Context, projectKey string, component string) (string, error) {
	project, resp, err := c.inner.Project.GetWithContext(ctx, strings.ToUpper(projectKey))
	if err != nil {
		return "", fmt.Errorf("failed to get project: %s", c.handleError(resp, err))
	}

	for _, version := range project.Versions {
		if strings.Contains(version.Name, component) && version.Released != nil && !*version.Released {
			return version.Name, nil
		}
	}

	return "", fmt.Errorf("no component version, project: %s, component: %s", projectKey, component)
}

// findCompletionTransitionID finds the transition ID that moves an issue to a completed state
func (c *Client) findCompletionTransitionID(ctx context.Context, issue jira.Issue) (string, error) {
	transitions, resp, err := c.inner.Issue.GetTransitionsWithContext(ctx, issue.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get transitions: %s", c.handleError(resp, err))
	}

	for _, transition := range transitions {
		if transition.To.StatusCategory.Key == StatusCategoryDone {
			return transition.ID, nil
		}
	}

	return "", nil
}

// IsCompletedIssue checks if an issue is in a completed state
func (c *Client) IsCompletedIssue(issue jira.Issue) bool {
	if issue.Fields == nil || issue.Fields.Status == nil {
		return true
	}

	return issue.Fields.Status.StatusCategory.Key == StatusCategoryDone
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
