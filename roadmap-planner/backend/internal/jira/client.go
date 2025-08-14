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
	"strings"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/models"
	"github.com/andygrunwald/go-jira"
	"go.uber.org/zap"
	"github.com/trivago/tgo/tcontainer"
)

// Client wraps the Jira client with roadmap-specific functionality
type Client struct {
	inner   *jira.Client
	project string
	logger  *zap.Logger
}

// NewClient creates a new Jira client with the given credentials
func NewClient(baseURL, username, password, project string) (*Client, error) {
	tp := jira.BasicAuthTransport{
		Username: username,
		Password: password,
	}

	clientLogger := logger.WithComponent("jira-client")
	clientLogger.Debug("Creating new Jira client", zap.String("base_url", baseURL))

	inner, err := jira.NewClient(tp.Client(), baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create jira client: %w", err)
	}

	return &Client{
		inner:   inner,
		project: project,
		logger:  clientLogger,
	}, nil
}

// TestConnection tests the connection to Jira
func (c *Client) TestConnection(ctx context.Context) error {
	_, resp, err := c.inner.User.GetSelfWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to jira: %s", c.handleError(resp, err))
	}
	return nil
}

// GetPillars fetches all active pillars (parent issues) from the specified project
func (c *Client) GetPillars(ctx context.Context) ([]models.Pillar, error) {
	// JQL to find all active pillars in the project
	jqlQuery := fmt.Sprintf("project = %s AND (issuetype in (Pillar) and resolution is empty) OR (issuetype in (Milestone, Epic) AND resolution is empty) ORDER BY priority DESC, created ASC", c.project)

	searchOptions := &jira.SearchOptions{
		Fields: []string{"summary", "priority", "components", "status", "issuetype", "parent", "quater", "issuelinks", "extras", "customfield_12242", "customfield_10020", "customfield_10021", "customfield_12801", "customfield_sequence", "customfield_rank", "created"},
		MaxResults: 2000,
	}

	issues, resp, err := c.inner.Issue.SearchWithContext(ctx, jqlQuery, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pillars: %s", c.handleError(resp, err))
	}

	c.logger.Info("Found issues", zap.Int("count", len(issues)))

	for _, childIssue := range issues {
			c.logger.Info("Processing issue",
				zap.String("type", childIssue.Fields.Type.Name),
				zap.String("key", childIssue.Key),
				zap.String("summary", childIssue.Fields.Summary))
			if len(childIssue.Fields.IssueLinks) > 0 {
				for _, link := range childIssue.Fields.IssueLinks {
					c.logger.Debug("Issue link found",
						zap.String("type", link.Type.Name),
						zap.Any("inward_issue", link.InwardIssue),
						zap.Any("outward_issue", link.OutwardIssue))
				}
			}
		}


	pillars := make([]models.Pillar, 0, len(issues))
	for _, issue := range issues {
		if issue.Fields.Type.Name != "Pillar" {
			continue
		}
		pillar := models.ConvertJiraIssueToPillar(&issue)
		c.logger.Debug("Processing pillar",
			zap.String("key", pillar.Key),
			zap.String("name", pillar.Name),
			zap.String("id", pillar.ID))

		// Fetch milestones for this pillar
		// milestones, err := c.GetMilestones(ctx, pillar.ID)

		milestoneIssues := FilterIsssuesByParent(issues, pillar.Key)
		milestones := make([]models.Milestone, 0, len(milestoneIssues))
		for _, milestoneIssue := range milestoneIssues {
			milestone := models.ConvertJiraIssueToMilestone(&milestoneIssue, pillar.ID)
			milestoneKey := milestone.Key



			c.logger.Debug("Processing milestone",
				zap.String("key", milestoneKey),
				zap.Int("total_issues", len(issues)))

			// for _, link := range milestoneIssue.Fields.IssueLinks {
			// 	c.logger.Infof("link: %#v %#v %#v", link.Type, link.InwardIssue, link.OutwardIssue)
			// 	if link.Type.Name == ""
			// }

			// Filter epics
			epicIssues := FilterIssuesByFunc(issues, func(childIssue jira.Issue) bool {
				c.logger.Debug("Processing child issue",
					zap.String("type", childIssue.Fields.Type.Name),
					zap.String("key", childIssue.Key),
					zap.String("summary", childIssue.Fields.Summary))
				if childIssue.Fields.Type.Name != "Epic" {
					return false
				}
				for _, link := range childIssue.Fields.IssueLinks {
					if link.Type.Name == "Blocks" && link.OutwardIssue != nil && link.OutwardIssue.Key == milestoneKey {
						return true
					}
				}
				return false
			})
			// epicIssues := FilterIssuesByIssueLink(issues, "Blocks", milestone.Key)
			c.logger.Debug("Found epics for milestone",
				zap.Int("epic_count", len(epicIssues)),
				zap.String("milestone_key", milestone.Key))
			milestone.Epics = make([]models.Epic, 0, len(epicIssues))
			for _, epicIssue := range epicIssues {
				epic := models.ConvertJiraIssueToEpic(&epicIssue, milestone.ID)
				milestone.Epics = append(milestone.Epics, *epic)
			}
			milestones = append(milestones, *milestone)
		}
		c.logger.Info("Found milestones for pillar",
			zap.Int("milestone_count", len(milestones)),
			zap.String("pillar_key", pillar.Key))
		pillar.Milestones = milestones

		pillars = append(pillars, *pillar)
	}

	// Sort pillars by sequence
	models.SortPillars(pillars)

	// Sort milestones within each pillar
	for i := range pillars {
		models.SortMilestones(pillars[i].Milestones)

		// Sort epics within each milestone
		for j := range pillars[i].Milestones {
			models.SortEpics(pillars[i].Milestones[j].Epics)
		}
	}

	c.logger.Info("Returning pillars", zap.Int("total_count", len(pillars)))
	return pillars, nil
}

// GetMilestones fetches all milestones (sub-tasks) for a given pillar
func (c *Client) GetMilestones(ctx context.Context, pillarID string) ([]models.Milestone, error) {
	// JQL to find all sub-tasks of the pillar
	jqlQuery := fmt.Sprintf("parent = %s ORDER BY created ASC", pillarID)
	c.logger.Info("Fetching milestones with JQL", zap.String("query", jqlQuery))

	searchOptions := &jira.SearchOptions{
		Fields: []string{"summary", "status", "parent", "customfield_*", "customfield_10020", "customfield_10021", "customfield_sequence", "customfield_rank", "created"},
	}

	issues, resp, err := c.inner.Issue.SearchWithContext(ctx, jqlQuery, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch milestones for pillar %s: %s", pillarID, c.handleError(resp, err))
	}

	c.logger.Info("Found milestone issues for pillar",
		zap.Int("count", len(issues)),
		zap.String("pillar_id", pillarID))

	milestones := make([]models.Milestone, 0, len(issues))
	for _, issue := range issues {
		milestone := models.ConvertJiraIssueToMilestone(&issue, pillarID)

		// Fetch epics for this milestone
		epics, err := c.GetEpics(ctx, milestone.ID)
		if err != nil {
			c.logger.Warn("Failed to fetch epics for milestone",
				zap.String("milestone_key", milestone.Key),
				zap.Error(err))
			epics = []models.Epic{}
		}
		milestone.Epics = epics

		milestones = append(milestones, *milestone)
	}

	// Sort milestones by sequence
	models.SortMilestones(milestones)

	// Sort epics within each milestone
	for i := range milestones {
		models.SortEpics(milestones[i].Epics)
	}

	return milestones, nil
}

// GetEpics fetches all epics linked to a milestone via "blocks" relationship
func (c *Client) GetEpics(ctx context.Context, milestoneID string) ([]models.Epic, error) {
	// JQL to find all issues that block the milestone
	jqlQuery := fmt.Sprintf("issue in linkedIssues(%s, \"blocks\") ORDER BY priority DESC, created ASC", milestoneID)

	searchOptions := &jira.SearchOptions{
		Fields: []string{"summary", "status", "priority", "components", "fixVersions", "issuetype", "created"},
	}

	issues, resp, err := c.inner.Issue.SearchWithContext(ctx, jqlQuery, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch epics for milestone %s: %s", milestoneID, c.handleError(resp, err))
	}

	epics := make([]models.Epic, 0, len(issues))
	for _, issue := range issues {
		epic := models.ConvertJiraIssueToEpic(&issue, milestoneID)
		epics = append(epics, *epic)
	}

	// Sort epics by fix version (blanks first)
	models.SortEpics(epics)

	return epics, nil
}

// CreateMilestone creates a new milestone as a sub-task of a pillar
func (c *Client) CreateMilestone(ctx context.Context, req models.CreateMilestoneRequest) (*models.Milestone, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Create the milestone issue
	issue := &jira.Issue{
		Fields: &jira.IssueFields{
			Project:  jira.Project{Key: c.project},
			Type:     jira.IssueType{Name: "Milestone"},
			Summary:  req.Name,
			Parent:   &jira.Parent{ID: req.PillarID},
			Assignee: &jira.User{Name: req.Assignee.Name},
		},
	}

	// Add quarter custom field if available
	issue.Fields.Description = fmt.Sprintf("Quarter: %s", req.Quarter)
	issue.Fields.Unknowns = make(tcontainer.MarshalMap)
	issue.Fields.Unknowns.Set("customfield_12242", map[string]interface{}{
		"value": req.Quarter,
	})

	createdIssue, resp, err := c.inner.Issue.CreateWithContext(ctx, issue)
	if err != nil {
		return nil, fmt.Errorf("failed to create milestone: %s", c.handleError(resp, err))
	}
	c.logger.Sugar().Infow("Milestone created", "issue", createdIssue)

	milestone := models.ConvertJiraIssueToMilestone(issue, req.PillarID)
	milestone.Quarter = req.Quarter

	return milestone, nil
}

// CreateEpic creates a new epic and links it to a milestone
func (c *Client) CreateEpic(ctx context.Context, req models.CreateEpicRequest) (*models.Epic, error) {
	// Create the epic issue
	issue := &jira.Issue{
		Fields: &jira.IssueFields{
			Project:  jira.Project{Key: c.project},
			Type:     jira.IssueType{Name: "Epic"},
			Summary:  req.Name,
			Assignee: &jira.User{Name: req.Assignee.Name},
		},
	}

	// Add component if specified
	if req.Component != "" {
		issue.Fields.Components = []*jira.Component{
			{Name: req.Component},
		}
	}

	// Add version if specified
	if req.Version != "" {
		issue.Fields.FixVersions = []*jira.FixVersion{
			{Name: req.Version},
		}
	}

	// Add priority if specified
	if req.Priority != "" {
		issue.Fields.Priority = &jira.Priority{Name: req.Priority}
	}

	createdIssue, resp, err := c.inner.Issue.CreateWithContext(ctx, issue)
	if err != nil {
		return nil, fmt.Errorf("failed to create epic: %s", c.handleError(resp, err))
	}

	// Link the epic to the milestone using "blocks" relationship
	if err := c.LinkEpicToMilestone(ctx, createdIssue.ID, req.MilestoneID); err != nil {
		c.logger.Warn("Failed to link epic to milestone",
			zap.String("epic_key", createdIssue.Key),
			zap.String("milestone_id", req.MilestoneID),
			zap.Error(err))
	}

	epic := models.ConvertJiraIssueToEpic(createdIssue, req.MilestoneID)

	return epic, nil
}

// UpdateEpicMilestone moves an epic from one milestone to another
func (c *Client) UpdateEpicMilestone(ctx context.Context, epicID, newMilestoneID string) error {
	// First, get the current epic to find its current milestone links
	epic, resp, err := c.inner.Issue.GetWithContext(ctx, epicID, &jira.GetQueryOptions{
		Fields: "issuelinks",
	})
	if err != nil {
		return fmt.Errorf("failed to get epic %s: %s", epicID, c.handleError(resp, err))
	}

	// Remove existing "blocks" links to milestones
	for _, link := range epic.Fields.IssueLinks {
		if link.Type.Name == "Blocks" && link.OutwardIssue != nil && link.OutwardIssue.Fields.Type.Name == "Milestone" {
			if err := c.removeIssueLink(ctx, link.ID); err != nil {
				c.logger.Warn("Failed to remove link",
					zap.String("link_id", link.ID),
					zap.Error(err))
			}
		}
	}

	// Create new link to the new milestone
	return c.LinkEpicToMilestone(ctx, epicID, newMilestoneID)
}

// LinkEpicToMilestone creates a "blocks" link between an epic and a milestone
func (c *Client) LinkEpicToMilestone(ctx context.Context, epicID, milestoneID string) error {
	issueLink := &jira.IssueLink{
		Type: jira.IssueLinkType{Name: "Blocks"},
		InwardIssue: &jira.Issue{
			ID: epicID,
		},
		OutwardIssue: &jira.Issue{
			ID: milestoneID,
		},
	}

	resp, err := c.inner.Issue.AddLinkWithContext(ctx, issueLink)
	if err != nil {
		return fmt.Errorf("failed to link epic %s to milestone %s: %s", epicID, milestoneID, c.handleError(resp, err))
	}

	return nil
}

// UpdateMilestone updates a milestone's name and quarter
func (c *Client) UpdateMilestone(ctx context.Context, milestoneID string, req models.UpdateMilestoneRequest) error {
	// Validate quarter format
	if err := (&models.CreateMilestoneRequest{Quarter: req.Quarter}).Validate(); err != nil {
		return fmt.Errorf("invalid quarter format: %w", err)
	}

	// Get the milestone issue first to check if it exists
	// milestone, resp, err := c.inner.Issue.GetWithContext(ctx, milestoneID, nil)
	// if err != nil {
	// 	return fmt.Errorf("failed to get milestone %s: %s", milestoneID, c.handleError(resp, err))
	// }

	// issue := jira.Issue{
	// 	ID: milestoneID,
	// 	Fields: &jira.IssueFields{},
	// }

	// Prepare the update fields
	updateFields := map[string]interface{}{}

	// Update the quarter custom field
	// if issue.Fields.Unknowns == nil {
	// 	issue.Fields.Unknowns = make(map[string]interface{})
	// }
	// updateFields["customfield_12242"] = map[string]interface{}{
	// 	"value": req.Quarter,
	// }

	if req.Name != "" {
		updateFields["summary"] = req.Name
	}

	if req.Quarter != "" {
		updateFields["customfield_12242"] = map[string]interface{}{
			"value": req.Quarter,
		}
	}

	// // Update the milestone fields directly
	// milestone.Fields.Summary = req.Name
	// issue.Fields.Unknowns["customfield_12242"] = map[string]interface{}{
	// 	"value": req.Quarter,
	// }



	resp, err := c.inner.Issue.UpdateIssueWithContext(ctx, milestoneID, map[string]interface{}{"fields": updateFields})

	// Update the issue
	// _, resp, err = c.inner.Issue.UpdateWithContext(ctx, milestone)
	if err != nil {
		return fmt.Errorf("failed to update milestone %s: %s", milestoneID, c.handleError(resp, err))
	}
	defer resp.Body.Close()

	c.logger.Info("Updated milestone",
		zap.String("milestone_id", milestoneID),
		zap.String("name", req.Name),
		zap.String("quarter", req.Quarter))

	return nil
}

// GetComponentVersions fetches available versions for a component
func (c *Client) GetComponentVersions(ctx context.Context, component string) ([]string, error) {
	// Get project information to access versions
	project, resp, err := c.inner.Project.Get(c.project)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %s", c.handleError(resp, err))
	}

	// Filter versions by component name prefix
	var componentVersions []string
	for _, version := range project.Versions {
		if strings.HasPrefix(version.Name, component) {
			componentVersions = append(componentVersions, version.Name)
		}
	}

	return componentVersions, nil
}

// GetAssignableUsers fetches users that can be assigned to issues in the project
func (c *Client) GetAssignableUsers(ctx context.Context, query string, projectKey string, issueKey string) ([]models.User, error) {
	// Use the user search endpoint to get users


	req, err := c.inner.NewRequestWithContext(ctx, "GET", "rest/api/latest/user/assignable/search", nil)
	if err != nil {
		// error creating the request
		return nil, err
	}
	urlQuery := req.URL.Query()
	urlQuery.Add("username", query)
	urlQuery.Add("projectKeys", projectKey)
	urlQuery.Add("issueKey", issueKey)
	req.URL.RawQuery = urlQuery.Encode()

	c.logger.Debug("Querying users", zap.String("url", req.URL.String()))

	// jira.Actor
	users := make([]jira.User, 0, 10)
	resp, err := c.inner.Do(req, &users)
	// c.inner.User.Find(property string, tweaks ...jira.userSearchF)
	// users, resp, err := c.inner.User.Find("", jira.WithUsername(query), jira.WithActive(true))
	if err != nil {
		return nil, fmt.Errorf("failed to get assignable users: %s", c.handleError(resp, err))
	}

	var assignableUsers []models.User
	// Limit to first 50 users to avoid too many results
	limit := len(users)
	// if limit > 100 {
	// 	limit = 100
	// }

	for i := 0; i < limit; i++ {
		user := users[i]
		assignableUsers = append(assignableUsers, models.User{
			AccountID:    user.Key,
			Name:         user.Name,
			DisplayName:  user.DisplayName,
			EmailAddress: user.EmailAddress,
		})
	}

	return assignableUsers, nil
}

// removeIssueLink removes an issue link
func (c *Client) removeIssueLink(ctx context.Context, linkID string) error {
	resp, err := c.inner.Issue.DeleteLinkWithContext(ctx, linkID)
	if err != nil {
		return fmt.Errorf("failed to remove link %s: %s", linkID, c.handleError(resp, err))
	}
	return nil
}

// handleError handles Jira API errors and returns a formatted error message
func (c *Client) handleError(resp *jira.Response, err error) string {
	if resp != nil && resp.Body != nil {
		body, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			return fmt.Sprintf("%v (Response: %s)", err, string(body))
		}
	}
	return err.Error()
}
