/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package jira

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AlaudaDevops/toolbox/roadmap-planner/backend/internal/models"
	"github.com/andygrunwald/go-jira"
	"go.uber.org/zap"
)

// SnapshotIssue is a Jira issue shaped for the team-analytics snapshot
// writer. It deliberately does not reuse models.Issue so we can evolve
// the snapshot fields (story points, sprint) without rippling through
// the roadmap UI's existing JSON contracts.
type SnapshotIssue struct {
	ID            string
	Key           string
	IssueType     string
	Status        string
	Components    []string
	Versions      []string
	Assignee      *models.User
	StoryPoints   float64
	SprintID      string
	SprintName    string
	CreatedAt     time.Time
	ResolvedAt    *time.Time
	StatusChanges []models.StatusChange // populated when ExpandChangelog=true
}

// SnapshotSearchOpts configures SearchSnapshots.
//
// CustomFields are optional: leave them empty to skip extraction. We do
// not hard-code Cloud-vs-Server defaults because every Jira instance has
// different IDs.
type SnapshotSearchOpts struct {
	JQL              string
	PageSize         int  // default 200
	MaxPages         int  // 0 = until empty page
	ExpandChangelog  bool // include status timeline (~+30% latency)
	StoryPointsField string
	SprintField      string
}

// SearchSnapshots runs a JQL query, pages through all results, and
// returns the data the team-analytics snapshot writer needs.
//
// Pagination relies on go-jira's StartAt/MaxResults — the inner library
// does the per-page fetch, we drive the loop. We stop on:
//   - empty page (natural end), or
//   - MaxPages reached, or
//   - context cancelled.
//
// The fields fetched are minimal-but-sufficient for IssueSnapshot rows:
// summary, status, assignee, components, fix versions, issuetype,
// created, resolutiondate, plus configured custom fields.
func (c *Client) SearchSnapshots(ctx context.Context, opts SnapshotSearchOpts) ([]SnapshotIssue, error) {
	if opts.JQL == "" {
		return nil, fmt.Errorf("SearchSnapshots: JQL is required")
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 200
	}

	fields := []string{
		"summary", "assignee", "components", "issuetype", "status",
		"fixVersions", "created", "resolutiondate", "priority",
	}
	if opts.StoryPointsField != "" {
		fields = append(fields, opts.StoryPointsField)
	}
	if opts.SprintField != "" {
		fields = append(fields, opts.SprintField)
	}

	expand := ""
	if opts.ExpandChangelog {
		expand = "changelog"
	}

	out := make([]SnapshotIssue, 0)
	for page := 0; opts.MaxPages == 0 || page < opts.MaxPages; page++ {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		searchOpts := &jira.SearchOptions{
			Fields:     fields,
			Expand:     expand,
			StartAt:    page * opts.PageSize,
			MaxResults: opts.PageSize,
		}
		issues, resp, err := c.inner.Issue.SearchWithContext(ctx, opts.JQL, searchOpts)
		if err != nil {
			return out, fmt.Errorf("snapshot search page %d: %s", page, c.handleError(resp, err))
		}
		if len(issues) == 0 {
			break
		}
		for i := range issues {
			out = append(out, c.toSnapshotIssue(&issues[i], opts))
		}
		if len(issues) < opts.PageSize {
			break // last page
		}
	}

	c.logger.Debug("SearchSnapshots done", zap.Int("count", len(out)), zap.String("jql", opts.JQL))
	return out, nil
}

func (c *Client) toSnapshotIssue(issue *jira.Issue, opts SnapshotSearchOpts) SnapshotIssue {
	si := SnapshotIssue{
		ID:         issue.ID,
		Key:        issue.Key,
		Components: extractComponentsForSnapshot(issue),
		Versions:   extractVersionsForSnapshot(issue),
	}
	if issue.Fields == nil {
		return si
	}
	if issue.Fields.Type.Name != "" {
		si.IssueType = issue.Fields.Type.Name
	}
	if issue.Fields.Status != nil {
		si.Status = issue.Fields.Status.Name
	}
	if issue.Fields.Assignee != nil {
		si.Assignee = &models.User{
			AccountID:    issue.Fields.Assignee.AccountID,
			Name:         issue.Fields.Assignee.Name,
			DisplayName:  issue.Fields.Assignee.DisplayName,
			EmailAddress: issue.Fields.Assignee.EmailAddress,
			Active:       issue.Fields.Assignee.Active,
		}
	}
	si.CreatedAt = time.Time(issue.Fields.Created)
	if rd := time.Time(issue.Fields.Resolutiondate); !rd.IsZero() {
		si.ResolvedAt = &rd
	}

	// Custom fields — best effort. Story points is usually a float; sprint
	// is a list whose entries can be either Cloud-shaped objects or the
	// legacy GreenHopper string format.
	if opts.StoryPointsField != "" && issue.Fields.Unknowns != nil {
		if v, ok := issue.Fields.Unknowns[opts.StoryPointsField]; ok && v != nil {
			si.StoryPoints = asFloat(v)
		}
	}
	if opts.SprintField != "" && issue.Fields.Unknowns != nil {
		if v, ok := issue.Fields.Unknowns[opts.SprintField]; ok && v != nil {
			si.SprintID, si.SprintName = parseSprint(v)
		}
	}

	if opts.ExpandChangelog && issue.Changelog != nil {
		for _, h := range issue.Changelog.Histories {
			for _, item := range h.Items {
				if item.Field != "status" {
					continue
				}
				ts, _ := parseJiraDateTime(h.Created)
				si.StatusChanges = append(si.StatusChanges, models.StatusChange{
					FromStatus: item.FromString,
					ToStatus:   item.ToString,
					ChangedAt:  ts,
				})
			}
		}
	}

	return si
}

// asFloat coerces typical JSON-decoded numerics to float64.
func asFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

// parseSprint returns (id, name) of the *latest* sprint entry on an
// issue. Format-tolerant:
//
//   - Jira Cloud: list of objects with "id", "name", "state".
//   - Jira Server / GreenHopper: list of strings like
//     "com.atlassian.greenhopper.service.sprint.Sprint@hash[
//      id=123,rapidViewId=…,state=ACTIVE,name=Sprint 26.05,…]".
//
// We don't need to be authoritative — sprint id only fuels the rollup
// "this sprint" widget; if we cannot parse, we fall back to empty and
// the issue is excluded from sprint slices (no harm done).
func parseSprint(v interface{}) (string, string) {
	list, ok := v.([]interface{})
	if !ok || len(list) == 0 {
		return "", ""
	}
	last := list[len(list)-1]

	// Cloud shape.
	if obj, ok := last.(map[string]interface{}); ok {
		id := ""
		if v, ok := obj["id"]; ok {
			id = fmt.Sprint(v)
		}
		name := ""
		if v, ok := obj["name"].(string); ok {
			name = v
		}
		return id, name
	}

	// Server / GreenHopper shape.
	if s, ok := last.(string); ok {
		return parseGreenHopperSprint(s)
	}
	return "", ""
}

func parseGreenHopperSprint(s string) (string, string) {
	// extract everything between '[' and ']'
	open := strings.IndexByte(s, '[')
	close := strings.LastIndexByte(s, ']')
	if open < 0 || close < 0 || close <= open {
		return "", ""
	}
	body := s[open+1 : close]
	id, name := "", ""
	for _, kv := range strings.Split(body, ",") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "id":
			id = v
		case "name":
			name = v
		}
	}
	return id, name
}

func extractComponentsForSnapshot(issue *jira.Issue) []string {
	if issue.Fields == nil {
		return nil
	}
	out := make([]string, 0, len(issue.Fields.Components))
	for _, c := range issue.Fields.Components {
		out = append(out, c.Name)
	}
	return out
}

func extractVersionsForSnapshot(issue *jira.Issue) []string {
	if issue.Fields == nil {
		return nil
	}
	out := make([]string, 0, len(issue.Fields.FixVersions))
	for _, v := range issue.Fields.FixVersions {
		out = append(out, v.Name)
	}
	return out
}
