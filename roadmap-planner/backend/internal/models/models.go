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
	Components []string    `json:"components"`
	Sequence   int         `json:"sequence"` // For ordering pillars
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
	ID             string    `json:"id"`
	Key            string    `json:"key"`
	Name           string    `json:"name"`
	Versions       []string  `json:"versions"` // Fix version for sorting
	Components     []string  `json:"components"`
	MilestoneIDs   []string  `json:"milestone_ids"`
	Status         string    `json:"status"`
	Priority       string    `json:"priority"`
	Assignee       *User     `json:"assignee,omitempty"`
	ResolutionDate time.Time `json:"resolution_date,omitempty"`
	CreationDate   time.Time `json:"creation_date,omitempty"`
}

// Issue represents a product requirement or set of stories
type Issue struct {
	ID              string    `json:"id"`
	Key             string    `json:"key"`
	Type            string    `json:"type"`
	Name            string    `json:"name"`
	Versions        []string  `json:"versions"` // Fix version for sorting
	Components      []string  `json:"components"`
	AffectsVersions []string  `json:"affected_versions"`
	Status          string    `json:"status"`
	Priority        string    `json:"priority"`
	Assignee        *User     `json:"assignee,omitempty"`
	ResolutionDate  time.Time `json:"resolution_date,omitempty"`
	CreationDate    time.Time `json:"creation_date,omitempty"`
}

// User represents a Jira user
type User struct {
	AccountID    string `json:"account_id"`
	Name         string `json:"name"`
	DisplayName  string `json:"display_name"`
	EmailAddress string `json:"email_address"`
}

// StatusChange represents a status transition from Jira changelog
type StatusChange struct {
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	ChangedAt  time.Time `json:"changed_at"`
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
	Component   string `json:"component"`
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

// UpdateEpicRequest represents the request to update an epic
type UpdateEpicRequest struct {
	Name      string `json:"name" binding:"required"`
	Component string `json:"component"`
	Version   string `json:"version"`
	Priority  string `json:"priority"`
	Assignee  *User  `json:"assignee"`
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

// BasicData represents the basic/static roadmap data that doesn't change often
type BasicData struct {
	Pillars    []BasicPillar `json:"pillars"`
	Quarters   []string      `json:"quarters"`
	Components []string      `json:"components"`
	Versions   []string      `json:"versions"`
	Project    *Project      `json:"project"`
}

// BasicPillar represents a pillar without its milestones and epics
type BasicPillar struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	Priority  string `json:"priority"`
	Component string `json:"component"`
	Sequence  int    `json:"sequence"`
}

// Project is a jira project object
type Project struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Key        string      `json:"key"`
	Versions   []Version   `json:"versions,omitempty"`
	Components []Component `json:"components,omitempty"`
}

// Component in a Jira Project
type Component struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	Description string `json:"description"`
}

// Version represents a version in a Jira project
type Version struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Archieved       bool   `json:"archived"`
	Released        bool   `json:"released"`
	ReleaseDate     string `json:"releaseDate"`
	UserReleaseDate string `json:"userReleaseDate"`
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
		Components: extractComponentsFromIssue(issue),
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
	// issue.Transitions
	epic := &Epic{
		ID:             issue.ID,
		Key:            issue.Key,
		Name:           issue.Fields.Summary,
		Versions:       extractVersionsFromIssue(issue),
		Components:     extractComponentsFromIssue(issue),
		MilestoneIDs:   extractMilestoneIdsFromIssue(issue),
		Status:         extractStatusFromIssue(issue),
		Priority:       extractPriorityFromIssue(issue),
		Assignee:       convertJiraUserToUser(issue.Fields.Assignee),
		ResolutionDate: time.Time(issue.Fields.Resolutiondate),
		CreationDate:   time.Time(issue.Fields.Created),
	}

	return epic
}

// ConvertJiraIssueToIssue converts a Jira issue to an Epic model
func ConvertJiraIssueToIssue(issue *jira.Issue) *Issue {
	bug := &Issue{
		ID:             issue.ID,
		Key:            issue.Key,
		Name:           issue.Fields.Summary,
		Type:           issue.Fields.Type.Name,
		Versions:       extractVersionsFromIssue(issue),
		Components:     extractComponentsFromIssue(issue),
		Status:         extractStatusFromIssue(issue),
		Priority:       extractPriorityFromIssue(issue),
		Assignee:       convertJiraUserToUser(issue.Fields.Assignee),
		ResolutionDate: time.Time(issue.Fields.Resolutiondate),
		CreationDate:   time.Time(issue.Fields.Created),
	}

	return bug
}

// ConvertJiraProjectToProject converts a Jira project to a Project model
func ConvertJiraProjectToProject(project *jira.Project) *Project {
	p := &Project{
		ID:         project.ID,
		Name:       project.Name,
		Key:        project.Key,
		Versions:   make([]Version, 0, len(project.Versions)),
		Components: make([]Component, 0, len(project.Components)),
	}

	// Convert versions
	for _, version := range project.Versions {
		p.Versions = append(p.Versions, *ConvertJiraVersionToVersion(&version))
	}

	// Convert components
	for _, component := range project.Components {
		p.Components = append(p.Components, *ConvertJiraComponentToComponent(&component))
	}

	return p
}

// ConvertJiraComponentToComponent converts a Jira component to a Component model
func ConvertJiraComponentToComponent(component *jira.ProjectComponent) *Component {
	return &Component{
		ID:          component.ID,
		Name:        component.Name,
		Description: component.Description,
	}
}

// ConvertJiraVersionToVersion converts a Jira version to a Version model
func ConvertJiraVersionToVersion(version *jira.Version) *Version {
	v := &Version{
		ID:              version.ID,
		Name:            version.Name,
		ReleaseDate:     version.ReleaseDate,
		UserReleaseDate: version.UserReleaseDate,
	}

	// Handle boolean pointers
	if version.Archived != nil {
		v.Archieved = *version.Archived
	}
	if version.Released != nil {
		v.Released = *version.Released
	}

	return v
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
	if issue.Fields != nil && issue.Fields.Priority != nil {
		return issue.Fields.Priority.Name
	}
	return ""
}

// extractComponentsFromIssue extracts the components from a Jira issue
func extractComponentsFromIssue(issue *jira.Issue) (components []string) {
	if issue.Fields != nil {
		for _, component := range issue.Fields.Components {
			components = append(components, component.Name)
		}
	}
	return
}

// extractVersionsFromIssue extracts fix versions from a Jira issue
func extractVersionsFromIssue(issue *jira.Issue) (fixVersion []string) {
	if issue.Fields != nil {
		for _, version := range issue.Fields.FixVersions {
			fixVersion = append(fixVersion, version.Name)
		}
	}
	return
}

// extractStatusFromIssue extracts the status from a Jira issue
func extractStatusFromIssue(issue *jira.Issue) string {
	if issue.Fields != nil && issue.Fields.Status != nil {
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
		return strings.SplitN(issue.Fields.Summary, "：", 2)[0]
	}

	return ""
}

// extractMilestoneIdsFromIssue extracts the milestone IDs from a Jira issue
func extractMilestoneIdsFromIssue(issue *jira.Issue) (milestoneIDs []string) {
	if issue.Fields != nil && len(issue.Fields.IssueLinks) > 0 {
		for _, link := range issue.Fields.IssueLinks {
			if link.Type.Name == "Blocks" && link.OutwardIssue != nil && link.OutwardIssue.Fields.Type.Name == "Milestone" {
				milestoneIDs = append(milestoneIDs, link.OutwardIssue.ID)
			}
		}
	}
	return
}

// extractSequenceFromIssue extracts the sequence/rank from a Jira issue
func extractSequenceFromIssue(issue *jira.Issue) int {
	// Try to get from custom field first (common sequence field names)
	if issue.Fields.Unknowns != nil {
		// Try common sequence field names
		sequenceFields := []string{
			"customfield_10020", // Common rank field
			"customfield_10021", // Alternative rank field
			"customfield_12801", // custom sequence field
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

// extractQuartersFromMilestones extracts unique quarters from milestone data
func ExtractQuartersFromMilestones(milestones []Milestone) []string {
	quarterSet := make(map[string]bool)

	for _, milestone := range milestones {
		if milestone.Quarter != "" {
			quarterSet[milestone.Quarter] = true
		}
	}

	quarters := make([]string, 0, len(quarterSet))
	for quarter := range quarterSet {
		quarters = append(quarters, quarter)
	}

	// Sort quarters chronologically
	SortQuarters(quarters)

	return quarters
}

// // extractComponentsFromPillars extracts unique components from pillar data
// func ExtractComponentsFromPillars(pillars []Pillar) []string {
// 	componentSet := make(map[string]bool)

// 	for _, pillar := range pillars {
// 		if pillar.Component != "" {
// 			componentSet[pillar.Component] = true
// 		}

// 		// Also extract from milestones and epics
// 		for _, milestone := range pillar.Milestones {
// 			for _, epic := range milestone.Epics {
// 				if epic.Component != "" {
// 					componentSet[epic.Component] = true
// 				}
// 			}
// 		}
// 	}

// 	components := make([]string, 0, len(componentSet))
// 	for component := range componentSet {
// 		components = append(components, component)
// 	}

// 	// Sort components alphabetically
// 	sort.Strings(components)

// 	return components
// }

// // extractVersionsFromEpics extracts unique versions from epic data
// func ExtractVersionsFromEpics(pillars []Pillar) []string {
// 	versionSet := make(map[string]bool)

// 	for _, pillar := range pillars {
// 		for _, milestone := range pillar.Milestones {
// 			for _, epic := range milestone.Epics {
// 				if epic.Version != "" {
// 					versionSet[epic.Version] = true
// 				}
// 			}
// 		}
// 	}

// 	versions := make([]string, 0, len(versionSet))
// 	for version := range versionSet {
// 		versions = append(versions, version)
// 	}

// 	// Sort versions alphabetically
// 	sort.Strings(versions)

// 	return versions
// }

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

// SortBasicPillars sorts basic pillars by sequence, then by name
func SortBasicPillars(pillars []BasicPillar) {
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
		// if epics[i].Priority < epics[j].Priority {
		// 	return true
		// }

		// return epics[i].Name < epics[j].Name
		return epics[i].Priority < epics[j].Priority && epics[i].Name < epics[j].Name
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

// HasItem checks if any item in the slice exists in the index
func HasItem(index map[string]struct{}, item []string) bool {
	for _, i := range item {
		if _, exists := index[i]; exists {
			return true
		}
	}
	return false
}

// convertJiraUserToUser converts a Jira user to our User model
func convertJiraUserToUser(jiraUser *jira.User) *User {
	if jiraUser == nil {
		return nil
	}

	return &User{
		AccountID:    jiraUser.AccountID,
		Name:         jiraUser.Name,
		DisplayName:  jiraUser.DisplayName,
		EmailAddress: jiraUser.EmailAddress,
	}
}
