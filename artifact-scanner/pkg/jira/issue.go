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

// IssueTypeVulnerability represents the issue type for vulnerability tracking
const IssueTypeVulnerability = "Vulnerability"

// IssueTypeJob represents the issue type for job tracking
const IssueTypeJob = "Job"

// CustomFieldCVEScore represents the custom field ID for CVE score
const CustomFieldCVEScore = "customfield_11802"

// CustomFieldCVEID represents the custom field ID for CVE ID
const CustomFieldCVEID = "customfield_11801"

// IssueOption represents a function that can modify a Jira issue
type IssueOption func(issue *jira.Issue)

// newIssue creates a new Jira issue with the given options
// Returns a new Jira issue with the specified options applied
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
// project: The project key to set
func WithProject(project string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Project = jira.Project{Key: strings.ToUpper(project)}
	}
}

// WithAssignee sets the assignee for a Jira issue
// username: The username of the assignee
func WithAssignee(username string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Assignee = &jira.User{
			Name: username,
		}
	}
}

// WithType sets the issue type for a Jira issue
// issueType: The type of the issue
func WithType(issueType string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Type = jira.IssueType{Name: issueType}
	}
}

// WithAffectsVersion adds an affected version to a Jira issue
// version: The version that is affected by the issue
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
// description: The description text to set
func WithDescription(description string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Description = description
	}
}

// WithSummary sets the summary for a Jira issue
// summary: The summary text to set
func WithSummary(summary string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Summary = summary
	}
}

// WithPriority sets the priority for a Jira issue
// priority: The priority level to set
func WithPriority(priority string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Priority = &jira.Priority{Name: priority}
	}
}

// WithLabels sets the labels for a Jira issue
// labels: The labels to set for the issue
func WithLabels(labels ...string) IssueOption {
	return func(issue *jira.Issue) {
		issue.Fields.Labels = labels
	}
}

// WithCustomField sets custom fields for a Jira issue
// fields: A map of custom field IDs to their values
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
