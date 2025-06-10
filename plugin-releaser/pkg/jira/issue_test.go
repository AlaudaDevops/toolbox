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
	"testing"

	. "github.com/onsi/gomega"
)

func TestNewIssue(t *testing.T) {
	g := NewGomegaWithT(t)

	issue := newIssue()
	g.Expect(issue).ToNot(BeNil())
	g.Expect(issue.Fields).ToNot(BeNil())
	g.Expect(issue.Fields.AffectsVersions).To(HaveLen(0))
}

func TestWithProject(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name        string
		projectKey  string
		expectedKey string
	}{
		{
			name:        "lowercase project key",
			projectKey:  "test",
			expectedKey: "TEST",
		},
		{
			name:        "uppercase project key",
			projectKey:  "TEST",
			expectedKey: "TEST",
		},
		{
			name:        "mixed case project key",
			projectKey:  "TeSt",
			expectedKey: "TEST",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issue := newIssue(WithProject(tc.projectKey))
			g.Expect(issue.Fields.Project.Key).To(Equal(tc.expectedKey))
		})
	}
}

func TestWithAssignee(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name     string
		username string
	}{
		{
			name:     "valid username",
			username: "johndoe",
		},
		{
			name:     "empty username",
			username: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issue := newIssue(WithAssignee(tc.username))
			g.Expect(issue.Fields.Assignee).ToNot(BeNil())
			g.Expect(issue.Fields.Assignee.Name).To(Equal(tc.username))
		})
	}
}

func TestWithType(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name      string
		issueType string
	}{
		{
			name:      "vulnerability issue type",
			issueType: IssueTypeVulnerability,
		},
		{
			name:      "job issue type",
			issueType: IssueTypeJob,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issue := newIssue(WithType(tc.issueType))
			g.Expect(issue.Fields.Type.Name).To(Equal(tc.issueType))
		})
	}
}

func TestWithAffectsVersion(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name           string
		versions       []string
		expectedLength int
	}{
		{
			name:           "single version",
			versions:       []string{"1.0.0"},
			expectedLength: 1,
		},
		{
			name:           "multiple versions",
			versions:       []string{"1.0.0", "2.0.0"},
			expectedLength: 2,
		},
		{
			name:           "duplicate versions",
			versions:       []string{"1.0.0", "1.0.0"},
			expectedLength: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			options := make([]IssueOption, 0, len(tc.versions))
			for _, v := range tc.versions {
				options = append(options, WithAffectsVersion(v))
			}

			issue := newIssue(options...)
			g.Expect(issue.Fields.AffectsVersions).To(HaveLen(tc.expectedLength))

			// Verify versions are set correctly
			for _, v := range tc.versions[:tc.expectedLength] {
				found := false
				for _, av := range issue.Fields.AffectsVersions {
					if av.Name == v {
						found = true
						break
					}
				}
				g.Expect(found).To(BeTrue(), "Version %s should be in AffectsVersions", v)
			}
		})
	}
}

func TestWithDescription(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "empty description",
			description: "",
		},
		{
			name:        "non-empty description",
			description: "This is a test description",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issue := newIssue(WithDescription(tc.description))
			g.Expect(issue.Fields.Description).To(Equal(tc.description))
		})
	}
}

func TestWithSummary(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name    string
		summary string
	}{
		{
			name:    "empty summary",
			summary: "",
		},
		{
			name:    "non-empty summary",
			summary: "Test summary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issue := newIssue(WithSummary(tc.summary))
			g.Expect(issue.Fields.Summary).To(Equal(tc.summary))
		})
	}
}

func TestWithPriority(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name     string
		priority string
	}{
		{
			name:     "high priority",
			priority: "High",
		},
		{
			name:     "low priority",
			priority: "Low",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issue := newIssue(WithPriority(tc.priority))
			g.Expect(issue.Fields.Priority).ToNot(BeNil())
			g.Expect(issue.Fields.Priority.Name).To(Equal(tc.priority))
		})
	}
}

func TestWithLabels(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name         string
		labels       []string
		expectedSize int
	}{
		{
			name:         "no labels",
			labels:       []string{},
			expectedSize: 0,
		},
		{
			name:         "single label",
			labels:       []string{"test"},
			expectedSize: 1,
		},
		{
			name:         "multiple labels",
			labels:       []string{"test", "security"},
			expectedSize: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issue := newIssue(WithLabels(tc.labels...))
			g.Expect(issue.Fields.Labels).To(HaveLen(tc.expectedSize))
			for i, label := range tc.labels {
				g.Expect(issue.Fields.Labels[i]).To(Equal(label))
			}
		})
	}
}
