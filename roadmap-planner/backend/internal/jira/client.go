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
	"net/url"
	"strings"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/logger"
	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/models"
	"github.com/andygrunwald/go-jira"
	"github.com/trivago/tgo/tcontainer"
	"go.uber.org/zap"
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

	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
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

// ListProjects lists all projects
func (c *Client) ListProjects(ctx context.Context) ([]models.Project, error) {
	projectList, resp, err := c.inner.Project.GetListWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %s", c.handleError(resp, err))
	}
	if projectList == nil {
		return nil, fmt.Errorf("response is nil")
	}
	array := *projectList

	projects := make([]models.Project, 0, len(array))
	for _, project := range array {
		projects = append(projects, models.Project{
			ID:   project.ID,
			Name: project.Name,
			Key:  project.Key,
		})
	}
	return projects, nil
}

// GetPillars fetches all active pillars (parent issues) from the specified project
func (c *Client) GetPillars(ctx context.Context) ([]models.Pillar, error) {
	// JQL to find all active pillars in the project
	jqlQuery := fmt.Sprintf("project = %s AND (issuetype in (Pillar) and resolution is empty) OR (issuetype in (Milestone, Epic) AND resolution is empty) ORDER BY priority DESC, created ASC", c.project)

	searchOptions := &jira.SearchOptions{
		Fields:     []string{"summary", "priority", "components", "status", "issuetype", "parent", "quater", "issuelinks", "extras", "customfield_12242", "customfield_10020", "customfield_10021", "customfield_12801", "customfield_sequence", "customfield_rank", "created"},
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
			Unknowns: tcontainer.MarshalMap{"customfield_10004": req.Name},
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

	c.logger.Sugar().Debugw("Creating epic issue", "issue", issue, "paylod", req)

	createdIssue, resp, err := c.inner.Issue.CreateWithContext(ctx, issue)
	if err != nil {
		return nil, fmt.Errorf("failed to create epic: %s", c.handleError(resp, err))
	}
	c.logger.Sugar().Infow("Created epic issue", "epic", createdIssue, "req", req)

	// Link the epic to the milestone using "blocks" relationship
	if err := c.LinkEpicToMilestone(ctx, createdIssue.ID, req.MilestoneID); err != nil {
		c.logger.Warn("Failed to link epic to milestone",
			zap.String("epic_key", createdIssue.Key),
			zap.String("milestone_id", req.MilestoneID),
			zap.Error(err))
	}

	// epic := models.ConvertJiraIssueToEpic(createdIssue, req.MilestoneID)
	epic := &models.Epic{
		ID:           createdIssue.ID,
		Key:          createdIssue.Key,
		Name:         req.Name,
		Versions:     []string{req.Version},
		Components:   []string{req.Component},
		MilestoneIDs: []string{req.MilestoneID},
	}

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

	updateFields := map[string]interface{}{}

	// Update the quarter custom field
	if req.Name != "" {
		updateFields["summary"] = req.Name
	}

	if req.Quarter != "" {
		updateFields["customfield_12242"] = map[string]interface{}{
			"value": req.Quarter,
		}
	}
	// Update request
	resp, err := c.inner.Issue.UpdateIssueWithContext(ctx, milestoneID, map[string]interface{}{"fields": updateFields})
	if err != nil {
		return fmt.Errorf("failed to update milestone %s: %s", milestoneID, c.handleError(resp, err))
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn("Failed to close response body", zap.Error(closeErr))
		}
	}()

	c.logger.Info("Updated milestone",
		zap.String("milestone_id", milestoneID),
		zap.String("name", req.Name),
		zap.String("quarter", req.Quarter))

	return nil
}

// UpdateEpic updates an epic's details
func (c *Client) UpdateEpic(ctx context.Context, epicID string, req models.UpdateEpicRequest) error {
	// Get the epic issue first to check if it exists
	epic, resp, err := c.inner.Issue.GetWithContext(ctx, epicID, nil)
	if err != nil {
		return fmt.Errorf("failed to get epic %s: %s", epicID, c.handleError(resp, err))
	}

	updateFields := map[string]interface{}{}

	if req.Name != "" {
		updateFields["summary"] = req.Name
		updateFields["customfield_10004"] = req.Name
	}

	// Update the basic fields
	epic.Fields.Summary = req.Name

	// Update component if specified
	if req.Component != "" {
		// epic.Fields.Components = []*jira.Component{
		// 	{Name: req.Component},
		// }
		updateFields["components"] = []map[string]string{
			{"name": req.Component},
		}
	} else {
		updateFields["components"] = []map[string]string{}
	}

	// Update version if specified
	if req.Version != "" {
		updateFields["fixVersions"] = []map[string]string{
			{"name": req.Version},
		}
	} else {
		updateFields["fixVersions"] = []map[string]string{}
	}

	// Update priority if specified
	if req.Priority != "" {
		updateFields["priority"] = &jira.Priority{Name: req.Priority}
	}

	// Update assignee if specified
	if req.Assignee != nil {
		updateFields["assignee"] = &jira.User{
			Name: req.Assignee.Name,
		}
	}

	// Update the issue
	resp, err = c.inner.Issue.UpdateIssueWithContext(ctx, epicID, map[string]interface{}{"fields": updateFields})
	if err != nil {
		return fmt.Errorf("failed to update epic %s: %s", epicID, c.handleError(resp, err))
	}

	c.logger.Info("Updated epic",
		zap.String("epic_id", epicID),
		zap.String("name", req.Name),
		zap.String("component", req.Component),
		zap.String("version", req.Version),
		zap.String("priority", req.Priority))

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

// GetProjectDetails fetches details of the project
func (c *Client) GetProjectDetails(ctx context.Context, projectKey string) (*models.Project, error) {
	// Get project information to access versions
	if projectKey == "" {
		projectKey = c.project
	}
	project, resp, err := c.inner.Project.Get(projectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %s", c.handleError(resp, err))
	}
	return models.ConvertJiraProjectToProject(project), nil
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

// GetQuartersFromMilestones extracts quarters from existing milestone data
func (c *Client) GetQuartersFromMilestones(ctx context.Context) ([]string, error) {
	// Get all pillars to extract quarters from their milestones
	pillars, err := c.GetPillars(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pillars for quarter extraction: %w", err)
	}

	// Extract quarters from all milestones
	var allMilestones []models.Milestone
	for _, pillar := range pillars {
		allMilestones = append(allMilestones, pillar.Milestones...)
	}

	quarters := models.ExtractQuartersFromMilestones(allMilestones)

	// If no quarters found from milestones, fall back to generated quarters
	if len(quarters) == 0 {
		c.logger.Warn("No quarters found in milestone data, falling back to generated quarters")
		quarters = models.GenerateQuarters()
	}

	c.logger.Info("Extracted quarters from milestones",
		zap.Strings("quarters", quarters),
		zap.Int("count", len(quarters)))

	return quarters, nil
}

// GetBasicPillars fetches pillar information without milestones and epics
func (c *Client) GetBasicPillars(ctx context.Context) ([]models.BasicPillar, error) {
	// JQL to find only pillars (not milestones or epics)
	jqlQuery := fmt.Sprintf("project = %s AND issuetype = Pillar AND resolution is empty ORDER BY priority DESC, created ASC", c.project)

	searchOptions := &jira.SearchOptions{
		Fields:     []string{"summary", "priority", "components", "status", "customfield_10020", "customfield_10021", "customfield_12801", "created"},
		MaxResults: 1000,
	}

	issues, resp, err := c.inner.Issue.SearchWithContext(ctx, jqlQuery, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch basic pillars: %s", c.handleError(resp, err))
	}

	c.logger.Info("Found basic pillars", zap.Int("count", len(issues)))

	basicPillars := make([]models.BasicPillar, 0, len(issues))
	for _, issue := range issues {
		// Extract basic pillar information
		priority := ""
		if issue.Fields.Priority != nil {
			priority = issue.Fields.Priority.Name
		}

		component := ""
		if len(issue.Fields.Components) > 0 {
			component = issue.Fields.Components[0].Name
		}

		// Extract sequence from custom fields
		sequence := 0
		if issue.Fields.Unknowns != nil {
			sequenceFields := []string{"customfield_10020", "customfield_10021", "customfield_12801"}
			for _, fieldName := range sequenceFields {
				if seq, exists := issue.Fields.Unknowns[fieldName]; exists && seq != nil {
					if seqFloat, ok := seq.(float64); ok {
						sequence = int(seqFloat)
						break
					}
					if seqInt, ok := seq.(int); ok {
						sequence = seqInt
						break
					}
				}
			}
		}

		pillar := models.BasicPillar{
			ID:        issue.ID,
			Key:       issue.Key,
			Name:      issue.Fields.Summary,
			Priority:  priority,
			Component: component,
			Sequence:  sequence,
		}
		basicPillars = append(basicPillars, pillar)
	}

	// Sort pillars by sequence
	models.SortBasicPillars(basicPillars)

	return basicPillars, nil
}

// GetMilestonesWithFilter fetches milestones with optional filtering
func (c *Client) GetMilestonesWithFilter(ctx context.Context, pillarIDs []string, quarters []string) ([]models.Milestone, error) {
	// Build JQL query with filters
	var jqlParts []string
	jqlParts = append(jqlParts, fmt.Sprintf("project = %s", c.project))
	jqlParts = append(jqlParts, "issuetype = Milestone")
	jqlParts = append(jqlParts, "resolution is empty")

	// Add pillar ID filter if provided
	if len(pillarIDs) > 0 {
		pillarFilter := fmt.Sprintf("parent in (%s)", strings.Join(pillarIDs, ","))
		jqlParts = append(jqlParts, pillarFilter)
	}

	// Add quarter filter if provided (this would need custom field filtering)
	if len(quarters) > 0 {
		// For now, we'll filter quarters in post-processing since JQL custom field filtering is complex
		// TODO: Implement custom field filtering for quarters
		c.logger.Debug("Quarter filtering will be done in post-processing", zap.Strings("quarters", quarters))
	}

	jqlQuery := strings.Join(jqlParts, " AND ") + " ORDER BY created ASC"

	searchOptions := &jira.SearchOptions{
		Fields:     []string{"summary", "status", "parent", "customfield_*", "quater", "quater", "issuelinks", "extras", "customfield_12242", "customfield_10020", "customfield_10021", "customfield_12801", "customfield_sequence", "customfield_rank", "created"},
		MaxResults: 1000,
	}

	issues, resp, err := c.inner.Issue.SearchWithContext(ctx, jqlQuery, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch milestones: %s", c.handleError(resp, err))
	}

	c.logger.Info("Found milestones with filter",
		zap.Int("count", len(issues)),
		zap.Strings("pillar_ids", pillarIDs),
		zap.Strings("quarters", quarters))

	quarterFilter := map[string]struct{}{}
	for _, quarter := range quarters {
		quarterFilter[quarter] = struct{}{}
	}

	milestones := make([]models.Milestone, 0, len(issues))
	for _, issue := range issues {
		// Get the pillar ID from the parent
		pillarID := ""
		if issue.Fields.Parent != nil {
			pillarID = issue.Fields.Parent.ID
		}

		milestone := models.ConvertJiraIssueToMilestone(&issue, pillarID)

		// Post-process quarter filtering if needed
		if _, ok := quarterFilter[milestone.Quarter]; len(quarters) > 0 && !ok {
			continue
		}

		milestones = append(milestones, *milestone)
	}

	// Sort milestones
	models.SortMilestones(milestones)

	return milestones, nil
}

// GetEpicsWithFilter fetches epics with optional filtering
func (c *Client) GetEpicsWithFilter(ctx context.Context, milestoneIDs []string, pillarIDs []string, components []string, versions []string) ([]models.Epic, error) {
	// Build JQL query with filters
	var jqlParts []string
	jqlParts = append(jqlParts, fmt.Sprintf("project = %s", c.project))
	jqlParts = append(jqlParts, "issuetype = Epic")
	jqlParts = append(jqlParts, "status not in (Cancelled,已取消)")

	// Add milestone ID filter if provided
	if len(milestoneIDs) > 0 {
		milestoneFilter := fmt.Sprintf(`issueFunction in linkedIssuesOf("id in (%s)", "is blocked by")`, strings.Join(milestoneIDs, ","))
		jqlParts = append(jqlParts, milestoneFilter)
	}

	// Add pillar ID filter if provided (epics are grandchildren of pillars)
	if len(pillarIDs) > 0 {
		// This is more complex - we'd need to get milestones first, then filter epics
		// For now, we'll handle this in post-processing
		// TODO: Implement pillar filtering for epics by getting milestones first
		c.logger.Debug("Pillar filtering will be done in post-processing", zap.Strings("pillar_ids", pillarIDs))
	}

	// Add component filter if provided
	if len(components) > 0 {
		componentFilter := fmt.Sprintf(`component in (%q)`, strings.Join(components, ","))
		jqlParts = append(jqlParts, componentFilter)
	}

	// Add version filter if provided
	if len(versions) > 0 {
		versionFilter := fmt.Sprintf(`fixVersion in (%q)`, strings.Join(versions, ","))
		jqlParts = append(jqlParts, versionFilter)
	}

	jqlQuery := strings.Join(jqlParts, " AND ") + " ORDER BY created ASC"

	searchOptions := &jira.SearchOptions{
		Fields:     []string{"summary", "assignee", "priority", "components", "status", "parent", "fixVersions", "created", "issuelinks", "resolutiondate", "created", "customfield_12242", "customfield_10020", "customfield_10021", "customfield_12801", "customfield_sequence", "customfield_rank"},
		MaxResults: 2000,
	}

	issues, resp, err := c.inner.Issue.SearchWithContext(ctx, jqlQuery, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch epics: %s", c.handleError(resp, err))
	}

	c.logger.Info("Found epics with filter",
		zap.Int("count", len(issues)),
		zap.Strings("milestone_ids", milestoneIDs),
		zap.Strings("pillar_ids", pillarIDs),
		zap.Strings("components", components),
		zap.Strings("versions", versions))

	milestoneIDsIndex := map[string]struct{}{}

	for _, milestoneID := range milestoneIDs {
		milestoneIDsIndex[milestoneID] = struct{}{}
	}

	epics := make([]models.Epic, 0, len(issues))
	for _, issue := range issues {

		epic := models.ConvertJiraIssueToEpic(&issue, "")

		if len(milestoneIDsIndex) > 0 && !models.HasItem(milestoneIDsIndex, epic.MilestoneIDs) {
			continue
		}

		epics = append(epics, *epic)
	}

	// Sort epics
	models.SortEpics(epics)

	return epics, nil
}

// GetIssuesWithFilter fetches issues with optional filtering
func (c *Client) GetIssuesWithFilter(ctx context.Context, epicIDs []string, components []string, versions []string, issueTypes []string) ([]models.Issue, error) {
	// Build JQL query with filters
	var jqlParts []string
	jqlParts = append(jqlParts, fmt.Sprintf("project = %s", c.project))
	jqlParts = append(jqlParts, "status not in (Cancelled,已取消)")
	jqlParts = append(jqlParts, "created > startOfDay(-366)")

	if len(issueTypes) > 0 {
		typeFilter := fmt.Sprintf(`issuetype in (%s)`, strings.Join(issueTypes, ","))
		jqlParts = append(jqlParts, typeFilter)
	}
	// Add milestone ID filter if provided
	if len(epicIDs) > 0 {
		milestoneFilter := fmt.Sprintf(`"Epic Link" in (%s)`, strings.Join(epicIDs, ","))
		jqlParts = append(jqlParts, milestoneFilter)
	}

	// Add component filter if provided
	if len(components) > 0 {
		componentFilter := fmt.Sprintf(`component in (%q)`, strings.Join(components, ","))
		jqlParts = append(jqlParts, componentFilter)
	}

	// Add version filter if provided
	if len(versions) > 0 {
		versionFilter := fmt.Sprintf(`fixVersion in (%q)`, strings.Join(versions, ","))
		jqlParts = append(jqlParts, versionFilter)
	}

	jqlQuery := strings.Join(jqlParts, " AND ") + " ORDER BY created ASC"

	searchOptions := &jira.SearchOptions{
		Fields:     []string{"summary", "assignee", "priority", "components", "issuetype", "status", "parent", "fixVersions", "created", "issuelinks", "resolutiondate", "created", "customfield_12242", "customfield_10020", "customfield_10021", "customfield_12801", "customfield_sequence", "customfield_rank"},
		MaxResults: 2000,
	}

	jiraIssues, resp, err := c.inner.Issue.SearchWithContext(ctx, jqlQuery, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issues: %s", c.handleError(resp, err))
	}

	c.logger.Info("Found issues with filter",
		zap.Int("count", len(jiraIssues)),
		zap.Strings("epic_ids", epicIDs),
		zap.Strings("issueTypes", issueTypes),
		zap.Strings("components", components),
		zap.Strings("versions", versions))

	issues := make([]models.Issue, 0, len(jiraIssues))
	for _, issue := range jiraIssues {

		bug := models.ConvertJiraIssueToIssue(&issue)

		issues = append(issues, *bug)
	}
	return issues, nil
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

// ChangelogEntry represents a single changelog entry from Jira
type ChangelogEntry struct {
	ID      string          `json:"id"`
	Author  *jira.User      `json:"author"`
	Created string          `json:"created"`
	Items   []ChangelogItem `json:"items"`
}

// ChangelogItem represents a single field change in a changelog entry
type ChangelogItem struct {
	Field      string `json:"field"`
	FieldType  string `json:"fieldtype"`
	From       string `json:"from"`
	FromString string `json:"fromString"`
	To         string `json:"to"`
	ToString   string `json:"toString"`
}

// IssueChangelog represents the changelog of an issue
type IssueChangelog struct {
	StartAt    int              `json:"startAt"`
	MaxResults int              `json:"maxResults"`
	Total      int              `json:"total"`
	Histories  []ChangelogEntry `json:"histories"`
}

// GetIssueChangelog fetches the changelog for a specific issue
func (c *Client) GetIssueChangelog(ctx context.Context, issueID string) (*IssueChangelog, error) {
	c.logger.Debug("Fetching changelog for issue", zap.String("issue_id", issueID))

	// The changelog is included in the issue when using expand=changelog
	issue, resp, err := c.inner.Issue.GetWithContext(ctx, issueID, &jira.GetQueryOptions{
		Expand: "changelog",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get issue changelog: %s", c.handleError(resp, err))
	}

	if issue.Changelog == nil {
		return &IssueChangelog{
			Histories: []ChangelogEntry{},
		}, nil
	}

	// Convert Jira changelog to our type
	changelog := &IssueChangelog{
		Total:     len(issue.Changelog.Histories),
		Histories: make([]ChangelogEntry, 0, len(issue.Changelog.Histories)),
	}

	for _, history := range issue.Changelog.Histories {
		entry := ChangelogEntry{
			ID:      history.Id,
			Author:  &history.Author,
			Created: history.Created,
			Items:   make([]ChangelogItem, 0, len(history.Items)),
		}
		for _, item := range history.Items {
			// Convert interface{} to string for From and To fields
			fromStr := ""
			if item.From != nil {
				if s, ok := item.From.(string); ok {
					fromStr = s
				}
			}
			toStr := ""
			if item.To != nil {
				if s, ok := item.To.(string); ok {
					toStr = s
				}
			}
			entry.Items = append(entry.Items, ChangelogItem{
				Field:      item.Field,
				FieldType:  item.FieldType,
				From:       fromStr,
				FromString: item.FromString,
				To:         toStr,
				ToString:   item.ToString,
			})
		}
		changelog.Histories = append(changelog.Histories, entry)
	}

	c.logger.Debug("Fetched changelog",
		zap.String("issue_id", issueID),
		zap.Int("history_count", len(changelog.Histories)))

	return changelog, nil
}

// GetStatusChanges extracts status changes from an issue's changelog
func (c *Client) GetStatusChanges(ctx context.Context, issueID string) ([]models.StatusChange, error) {
	changelog, err := c.GetIssueChangelog(ctx, issueID)
	if err != nil {
		return nil, err
	}

	var statusChanges []models.StatusChange
	for _, history := range changelog.Histories {
		for _, item := range history.Items {
			if item.Field == "status" {
				changedAt, _ := parseJiraDateTime(history.Created)
				statusChanges = append(statusChanges, models.StatusChange{
					FromStatus: item.FromString,
					ToStatus:   item.ToString,
					ChangedAt:  changedAt,
				})
			}
		}
	}

	return statusChanges, nil
}

// parseJiraDateTime parses a Jira datetime string
func parseJiraDateTime(dateStr string) (time.Time, error) {
	// Jira uses format: 2024-01-15T10:30:00.000+0000
	formats := []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05-0700",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
