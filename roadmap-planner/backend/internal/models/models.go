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

package models

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
)

// Pillar represents a high-level domain in the roadmap
type Pillar struct {
	ID         string      `json:"id"`
	Key        string      `json:"key"`
	Name       string      `json:"name"`
	Priority   string      `json:"priority"`
	Component  string      `json:"component"`
	Sequence   int         `json:"sequence"`   // For ordering pillars
	Milestones []Milestone `json:"milestones"`
}

// Milestone represents a goal to be achieved in a quarter
type Milestone struct {
	ID       string `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Quarter  string `json:"quarter"` // 2025Q1, 2025Q2, etc.
	PillarID string `json:"pillar_id"`
	Sequence int    `json:"sequence"` // For ordering milestones within a pillar
	Epics    []Epic `json:"epics"`
	Status   string `json:"status"`
}

// Epic represents a product requirement or set of stories
type Epic struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Version     string `json:"version"`     // Fix version for sorting
	Component   string `json:"component"`
	MilestoneID string `json:"milestone_id"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
}

// User represents a Jira user
type User struct {
	AccountID    string `json:"account_id"`
	Name         string `json:"name"`
	DisplayName  string `json:"display_name"`
	EmailAddress string `json:"email_address"`
}

// CreateMilestoneRequest represents the request to create a new milestone
type CreateMilestoneRequest struct {
	Name     string `json:"name" binding:"required"`
	Quarter  string `json:"quarter" binding:"required"`
	PillarID string `json:"pillar_id" binding:"required"`
	Assignee User   `json:"assignee" binding:"required"`
}

// CreateEpicRequest represents the request to create a new epic
type CreateEpicRequest struct {
	Name        string `json:"name" binding:"required"`
	Component   string `json:"component" binding:"required"`
	Version     string `json:"version"`
	MilestoneID string `json:"milestone_id" binding:"required"`
	Priority    string `json:"priority"`
	Assignee    User   `json:"assignee" binding:"required"`
}

// UpdateEpicMilestoneRequest represents the request to move an epic to a different milestone
type UpdateEpicMilestoneRequest struct {
	MilestoneID string `json:"milestone_id" binding:"required"`
}

// UpdateMilestoneRequest represents the request to update a milestone
type UpdateMilestoneRequest struct {
	Name    string `json:"name" binding:"required"`
	Quarter string `json:"quarter" binding:"required"`
}

// AuthRequest represents the authentication request
type AuthRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	BaseURL  string `json:"base_url" binding:"required"`
}

// RoadmapData represents the complete roadmap data
type RoadmapData struct {
	Pillars  []Pillar `json:"pillars"`
	Quarters []string `json:"quarters"`
}

// Validate validates the quarter format (YYYYQX)
func (r *CreateMilestoneRequest) Validate() error {
	quarterRegex := regexp.MustCompile(`^\d{4}Q[1-4]$`)
	if !quarterRegex.MatchString(r.Quarter) {
		return fmt.Errorf("invalid quarter format, expected YYYYQX (e.g., 2025Q1)")
	}
	return nil
}

// ConvertJiraIssueToPillar converts a Jira issue to a Pillar model
func ConvertJiraIssueToPillar(issue *jira.Issue) *Pillar {
	pillar := &Pillar{
		ID:         issue.ID,
		Key:        issue.Key,
		Name:       issue.Fields.Summary,
		Priority:   extractPriorityFromIssue(issue),
		Component:  extractComponentFromIssue(issue),
		Sequence:   extractSequenceFromIssue(issue),
		Milestones: []Milestone{},
	}

	return pillar
}

// ConvertJiraIssueToMilestone converts a Jira issue to a Milestone model
func ConvertJiraIssueToMilestone(issue *jira.Issue, pillarID string) *Milestone {
	milestone := &Milestone{
		ID:       issue.ID,
		Key:      issue.Key,
		Name:     issue.Fields.Summary,
		Quarter:  extractQuarterFromIssue(issue),
		PillarID: pillarID,
		Sequence: extractSequenceFromIssue(issue),
		Epics:    []Epic{},
		Status:   extractStatusFromIssue(issue),
	}

	return milestone
}

// ConvertJiraIssueToEpic converts a Jira issue to an Epic model
func ConvertJiraIssueToEpic(issue *jira.Issue, milestoneID string) *Epic {
	epic := &Epic{
		ID:          issue.ID,
		Key:         issue.Key,
		Name:        issue.Fields.Summary,
		Version:     extractVersionFromIssue(issue),
		Component:   extractComponentFromIssue(issue),
		MilestoneID: milestoneID,
		Status:      extractStatusFromIssue(issue),
		Priority:    extractPriorityFromIssue(issue),
	}

	return epic
}

// GenerateQuarters generates a list of quarters for the roadmap
func GenerateQuarters() []string {
	currentYear := time.Now().Year()
	quarters := make([]string, 0, 8) // 2 years worth of quarters

	for year := currentYear; year <= currentYear+1; year++ {
		for quarter := 1; quarter <= 4; quarter++ {
			quarters = append(quarters, fmt.Sprintf("%dQ%d", year, quarter))
		}
	}

	// Sort quarters chronologically (older first)
	SortQuarters(quarters)

	return quarters
}

// Helper functions for extracting data from Jira issues

// extractPriorityFromIssue extracts the priority from a Jira issue
func extractPriorityFromIssue(issue *jira.Issue) string {
	if issue.Fields.Priority != nil {
		return issue.Fields.Priority.Name
	}
	return ""
}

// extractComponentFromIssue extracts the primary component from a Jira issue
func extractComponentFromIssue(issue *jira.Issue) string {
	if len(issue.Fields.Components) > 0 {
		return issue.Fields.Components[0].Name
	}
	return ""
}

// extractVersionFromIssue extracts the primary fix version from a Jira issue
func extractVersionFromIssue(issue *jira.Issue) string {
	if len(issue.Fields.FixVersions) > 0 {
		return issue.Fields.FixVersions[0].Name
	}
	return ""
}

// extractStatusFromIssue extracts the status from a Jira issue
func extractStatusFromIssue(issue *jira.Issue) string {
	if issue.Fields.Status != nil {
		return issue.Fields.Status.Name
	}
	return ""
}

// extractQuarterFromIssue extracts the quarter value from a Jira issue
func extractQuarterFromIssue(issue *jira.Issue) string {
	// Try to get from custom field first
	if issue.Fields != nil && issue.Fields.Unknowns != nil {
		if quarter, exists := issue.Fields.Unknowns["customfield_12242"]; exists {
			if quarterStr, ok := quarter.(map[string]interface{}); ok && quarterStr != nil && quarterStr["value"] != nil {
				return quarterStr["value"].(string)
			}
		}
	}

	// Fallback: try to extract from description
	if issue.Fields.Summary != "" {
		return strings.SplitN(issue.Fields.Summary, ":", 2)[0]
		// issue.Fields.Summary
		// lines := strings.Split(issue.Fields.Description, "\n")
		// for _, line := range lines {
		// 	if strings.HasPrefix(strings.ToLower(line), "quarter:") {
		// 		parts := strings.SplitN(line, ":", 2)
		// 		if len(parts) == 2 {
		// 			return strings.TrimSpace(parts[1])
		// 		}
		// 	}
		// }
	}

	return ""
}

// extractSequenceFromIssue extracts the sequence/rank from a Jira issue
func extractSequenceFromIssue(issue *jira.Issue) int {
	// Try to get from custom field first (common sequence field names)
	if issue.Fields.Unknowns != nil {
		// Try common sequence field names
		sequenceFields := []string{
			"customfield_10020", // Common rank field
			"customfield_10021", // Alternative rank field
			"customfield_12801",
			"customfield_sequence",
			"customfield_rank",
		}

		for _, fieldName := range sequenceFields {
			if sequence, exists := issue.Fields.Unknowns[fieldName]; exists && sequence != nil {
				if seqFloat, ok := sequence.(float64); ok {
					return int(seqFloat)
				}
				if seqInt, ok := sequence.(int); ok {
					return seqInt
				}
				if seqStr, ok := sequence.(string); ok {
					if seqInt, err := strconv.Atoi(seqStr); err == nil {
						return seqInt
					}
				}
			}
		}
	}

	// Fallback: return 0 if no sequence found
	return 0
}

// Sorting functions

// SortPillars sorts pillars by sequence, then by name
func SortPillars(pillars []Pillar) {
	sort.Slice(pillars, func(i, j int) bool {
		if pillars[i].Sequence != pillars[j].Sequence {
			return pillars[i].Sequence < pillars[j].Sequence
		}
		return pillars[i].Name < pillars[j].Name
	})
}

// SortMilestones sorts milestones by sequence, then by name
func SortMilestones(milestones []Milestone) {
	sort.Slice(milestones, func(i, j int) bool {
		if milestones[i].Sequence != milestones[j].Sequence {
			return milestones[i].Sequence < milestones[j].Sequence
		}
		return milestones[i].Name < milestones[j].Name
	})
}

// SortEpics sorts epics by fix version (blanks first), then by name
func SortEpics(epics []Epic) {
	sort.Slice(epics, func(i, j int) bool {
		// Blanks (empty versions) should come first
		if epics[i].Version == "" && epics[j].Version != "" {
			return true
		}
		if epics[i].Version != "" && epics[j].Version == "" {
			return false
		}

		// If both have versions or both are blank, sort by version then name
		if epics[i].Version != epics[j].Version {
			return epics[i].Version < epics[j].Version
		}
		return epics[i].Name < epics[j].Name
	})
}

// SortQuarters sorts quarters chronologically (older first)
func SortQuarters(quarters []string) {
	sort.Slice(quarters, func(i, j int) bool {
		return parseQuarter(quarters[i]) < parseQuarter(quarters[j])
	})
}

// parseQuarter converts a quarter string (e.g., "2025Q1") to a comparable integer
func parseQuarter(quarter string) int {
	if len(quarter) < 6 {
		return 0
	}

	yearStr := quarter[:4]
	quarterStr := quarter[5:]

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return 0
	}

	q, err := strconv.Atoi(quarterStr)
	if err != nil {
		return 0
	}

	// Convert to comparable integer: year * 10 + quarter
	// e.g., 2025Q1 -> 20251, 2025Q2 -> 20252
	return year*10 + q
}
