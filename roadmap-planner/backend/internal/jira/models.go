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
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/trivago/tgo/tcontainer"
)

// IssueOption represents a function that can modify a Jira issue
type IssueOption func(issue *jira.Issue)

// newIssue creates a new Jira issue with the given options
func newIssue(options ...IssueOption) *jira.Issue {
	issue := &jira.Issue{
		Fields: &jira.IssueFields{
			Unknowns: make(tcontainer.MarshalMap),
		},
	}

	for _, option := range options {
		option(issue)
	}

	return issue
}

// WithProject sets the project for a Jira issue
func WithProject(project string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Project = jira.Project{Key: strings.ToUpper(project)}
	}
}

// WithAssignee sets the assignee for a Jira issue
func WithAssignee(username string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Assignee = &jira.User{
			Name: username,
		}
	}
}

// WithType sets the issue type for a Jira issue
func WithType(issueType string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Type = jira.IssueType{Name: issueType}
	}
}

// WithParent sets the parent issue for a Jira issue
func WithParent(parentID string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Parent = &jira.Parent{ID: parentID}
	}
}

// WithComponent adds a component to a Jira issue
func WithComponent(component string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Components = append(issue.Fields.Components, &jira.Component{
			Name: component,
		})
	}
}

// WithFixVersion adds a fix version to a Jira issue
func WithFixVersion(version string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.FixVersions = append(issue.Fields.FixVersions, &jira.FixVersion{
			Name: version,
		})
	}
}

// WithAffectsVersion adds an affected version to a Jira issue
func WithAffectsVersion(version string) IssueOption {
	return func(issue *jira.Issue) {
		for _, affectVersion := range issue.Fields.AffectsVersions {
			if affectVersion.Name == version {
				return
			}
		}
		issue.Fields.AffectsVersions = append(issue.Fields.AffectsVersions, &jira.AffectsVersion{
			Name: version,
		})
	}
}

// WithDescription sets the description for a Jira issue
func WithDescription(description string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Description = description
	}
}

// WithSummary sets the summary for a Jira issue
func WithSummary(summary string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Summary = summary
	}
}

// WithPriority sets the priority for a Jira issue
func WithPriority(priority string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Priority = &jira.Priority{Name: priority}
	}
}

// WithLabels sets the labels for a Jira issue
func WithLabels(labels ...string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Labels = labels
	}
}

// WithCustomField sets custom fields for a Jira issue
func WithCustomField(fields map[string]interface{}) IssueOption {
	return func(issue *jira.Issue) {
		if issue.Fields.Unknowns == nil {
			issue.Fields.Unknowns = tcontainer.MarshalMap{}
		}

		for k, v := range fields {
			issue.Fields.Unknowns.Set(k, v)
		}
	}
}

// WithQuarter sets the quarter custom field for a Jira issue
func WithQuarter(quarter string) IssueOption {
	return func(issue *jira.Issue) {
		if issue.Fields.Unknowns == nil {
			issue.Fields.Unknowns = tcontainer.MarshalMap{}
		}
		// This would need to be adapted based on the actual custom field ID
		issue.Fields.Unknowns.Set("customfield_quarter", quarter)
	}
}

// IsCompletedIssue checks if an issue is in a completed state
func IsCompletedIssue(issue *jira.Issue) bool {
	if issue.Fields.Status == nil {
		return false
	}

	completedStatuses := []string{
		"Done", "Closed", "Resolved", "Complete", "Completed",
		"Fixed", "Verified", "Released", "Deployed",
	}

	status := strings.ToLower(issue.Fields.Status.Name)
	for _, completedStatus := range completedStatuses {
		if strings.ToLower(completedStatus) == status {
			return true
		}
	}

	return false
}

// GetCustomFieldValue extracts a custom field value from a Jira issue
func GetCustomFieldValue(issue *jira.Issue, fieldID string) interface{} {
	if issue.Fields.Unknowns == nil {
		return nil
	}

	value, exists := issue.Fields.Unknowns[fieldID]
	if !exists {
		return nil
	}

	return value
}

// GetQuarterFromIssue extracts the quarter value from a Jira issue
func GetQuarterFromIssue(issue *jira.Issue) string {
	// Try to get from custom field first
	if quarter := GetCustomFieldValue(issue, "customfield_quarter"); quarter != nil {
		if quarterStr, ok := quarter.(string); ok {
			return quarterStr
		}
	}

	// Fallback: try to extract from description
	if issue.Fields.Description != "" {
		lines := strings.Split(issue.Fields.Description, "\n")
		for _, line := range lines {
			if strings.HasPrefix(strings.ToLower(line), "quarter:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	return ""
}

// ExtractComponentFromIssue extracts the primary component from a Jira issue
func ExtractComponentFromIssue(issue *jira.Issue) string {
	if len(issue.Fields.Components) > 0 {
		return issue.Fields.Components[0].Name
	}
	return ""
}

// ExtractVersionFromIssue extracts the primary fix version from a Jira issue
func ExtractVersionFromIssue(issue *jira.Issue) string {
	if len(issue.Fields.FixVersions) > 0 {
		return issue.Fields.FixVersions[0].Name
	}
	return ""
}

// ExtractPriorityFromIssue extracts the priority from a Jira issue
func ExtractPriorityFromIssue(issue *jira.Issue) string {
	if issue.Fields.Priority != nil {
		return issue.Fields.Priority.Name
	}
	return ""
}

// ExtractStatusFromIssue extracts the status from a Jira issue
func ExtractStatusFromIssue(issue *jira.Issue) string {
	if issue.Fields.Status != nil {
		return issue.Fields.Status.Name
	}
	return ""
}

// FilterIsssuesByParent filters a list of Jira issues by parent ID
func FilterIsssuesByParent(issues []jira.Issue, parentID string) []jira.Issue {
	// filtered := []jira.Issue{}
	// for _, issue := range issues {
	// 	if issue.Fields.Parent != nil && issue.Fields.Parent.Key == parentID {
	// 		filtered = append(filtered, issue)
	// 	}
	// }
	// return filtered

	return FilterIssuesByFunc(issues, func(issue jira.Issue) bool {
		return issue.Fields.Parent != nil && issue.Fields.Parent.Key == parentID
	})
}

// FilterIssuesByIssueLink filters a list of Jira issues by issue link type and inward issue ID
func FilterIssuesByIssueLink(issues []jira.Issue, linkType string, inwardIssueID string) []jira.Issue {
	linkType = strings.ToLower(linkType)
	return FilterIssuesByFunc(issues, func(issue jira.Issue) bool {
		for _, link := range issue.Fields.IssueLinks {
			if strings.ToLower(link.Type.Name) == linkType {
				if link.InwardIssue != nil && link.InwardIssue.Key == inwardIssueID {
					return true
				}
			}
		}
		return false
	})

	// filtered := []jira.Issue{}
	// for _, issue := range issues {
	// 	for _, link := range issue.Fields.IssueLinks {
	// 		if link.Type.Name == linkType {
	// 			if len(inwardIssueID) == 0 || (link.InwardIssue != nil && link.InwardIssue.Key == inwardIssueID[0]) {
	// 				filtered = append(filtered, issue)
	// 				break
	// 			}
	// 		}
	// 	}
	// }
	// return filtered
}

// FilterIssuesByFunc filters a list of Jira issues by a custom function
// the filterFunc will keep items that return true and discard those that return false
func FilterIssuesByFunc(issues []jira.Issue, filterFunc func(jira.Issue) bool) []jira.Issue {
	filtered := []jira.Issue{}
	for _, issue := range issues {
		if filterFunc(issue) {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}
