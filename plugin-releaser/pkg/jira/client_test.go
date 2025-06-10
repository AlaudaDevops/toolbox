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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andygrunwald/go-jira"
	"github.com/ankitpokhrel/jira-cli/pkg/jql"
	. "github.com/onsi/gomega"
)

func TestClient_FindOrCreateIssue(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name             string
		jql              *jql.JQL
		mockSearchResp   string
		mockSearchStatus int
		mockCreateResp   string
		mockCreateStatus int
		expectedIssueKey string
		wantErr          bool
	}{
		{
			name:             "finds existing issue",
			jql:              jql.NewJQL("TEST").FilterBy("status", "Done"),
			mockSearchResp:   `{"issues":[{"id":"10000","key":"TEST-1","fields":{"status":{"statusCategory":{"key":"new"}}}}]}`,
			mockSearchStatus: http.StatusOK,
			expectedIssueKey: "TEST-1",
			wantErr:          false,
		},
		{
			name:             "creates new issue when all existing are done",
			jql:              jql.NewJQL("TEST").FilterBy("summary", "~ 'Test Issue'"),
			mockSearchResp:   `{"issues":[{"id":"10000","key":"TEST-1","fields":{"status":{"statusCategory":{"key":"done"}}}}]}`,
			mockSearchStatus: http.StatusOK,
			mockCreateResp:   `{"id":"10001","key":"TEST-2"}`,
			mockCreateStatus: http.StatusCreated,
			expectedIssueKey: "TEST-2",
			wantErr:          false,
		},
		{
			name:             "search error",
			jql:              jql.NewJQL("TEST").FilterBy("summary", "~ 'Test Issue'"),
			mockSearchResp:   `{"errorMessages":["Error searching for issues"],"errors":{}}`,
			mockSearchStatus: http.StatusBadRequest,
			wantErr:          true,
		},
		{
			name:             "create error",
			jql:              jql.NewJQL("TEST").FilterBy("summary", "~ 'Test Issue'"),
			mockSearchResp:   `{"issues":[{"id":"10000","key":"TEST-1","fields":{"status":{"statusCategory":{"key":"done"}}}}]}`,
			mockSearchStatus: http.StatusOK,
			mockCreateResp:   `{"errorMessages":["Error creating issue"],"errors":{}}`,
			mockCreateStatus: http.StatusBadRequest,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && r.URL.Path == "/rest/api/2/search" {
					w.WriteHeader(tt.mockSearchStatus)
					fmt.Fprintln(w, tt.mockSearchResp)
					return
				}
				if r.Method == http.MethodPost && r.URL.Path == "/rest/api/2/issue" {
					w.WriteHeader(tt.mockCreateStatus)
					fmt.Fprintln(w, tt.mockCreateResp)
					return
				}
			}))
			defer ts.Close()

			client, err := NewClient(ts.URL, "user", "pass")
			g.Expect(err).ToNot(HaveOccurred())

			issue, err := client.FindOrCreateIssue(context.Background(), tt.jql,
				WithSummary("Test Issue"),
				WithProject("TEST"))

			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(issue).ToNot(BeNil())
				g.Expect(issue.Key).To(Equal(tt.expectedIssueKey))
			}
		})
	}
}

func TestClient_CreateOrUpdateIssue(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name             string
		jql              *jql.JQL
		mockSearchResp   string
		mockSearchStatus int
		mockCreateResp   string
		mockCreateStatus int
		mockUpdateResp   string
		mockUpdateStatus int
		expectedIssueKey string
		expectUpdate     bool
		wantErr          bool
	}{
		{
			name:             "creates new issue when none exist",
			jql:              jql.NewJQL("TEST").FilterBy("summary", "~ 'Test Issue'"),
			mockSearchResp:   `{"issues":[]}`,
			mockSearchStatus: http.StatusOK,
			mockCreateResp:   `{"id":"10001","key":"TEST-2"}`,
			mockCreateStatus: http.StatusCreated,
			expectedIssueKey: "TEST-2",
			expectUpdate:     false,
			wantErr:          false,
		},
		{
			name:             "updates existing issue",
			jql:              jql.NewJQL("TEST").FilterBy("summary", "~ 'Test Issue'"),
			mockSearchResp:   `{"issues":[{"id":"10000","key":"TEST-1","fields":{"status":{"statusCategory":{"key":"new"}}}}]}`,
			mockSearchStatus: http.StatusOK,
			mockUpdateResp:   `{"id":"10000","key":"TEST-1"}`,
			mockUpdateStatus: http.StatusOK,
			expectedIssueKey: "TEST-1",
			expectUpdate:     true,
			wantErr:          false,
		},
		{
			name:             "search error",
			jql:              jql.NewJQL("TEST").FilterBy("summary", "~ 'Test Issue'"),
			mockSearchResp:   `{"errorMessages":["Error searching for issues"],"errors":{}}`,
			mockSearchStatus: http.StatusBadRequest,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && r.URL.Path == "/rest/api/2/search" {
					w.WriteHeader(tt.mockSearchStatus)
					fmt.Fprintln(w, tt.mockSearchResp)
					return
				}
				if r.Method == http.MethodPost && r.URL.Path == "/rest/api/2/issue" {
					w.WriteHeader(tt.mockCreateStatus)
					fmt.Fprintln(w, tt.mockCreateResp)
					return
				}
				if r.Method == http.MethodPut && r.URL.Path == "/rest/api/2/issue/10000" {
					w.WriteHeader(tt.mockUpdateStatus)
					fmt.Fprintln(w, tt.mockUpdateResp)
					return
				}
			}))
			defer ts.Close()

			client, err := NewClient(ts.URL, "user", "pass")
			g.Expect(err).ToNot(HaveOccurred())

			issue, err := client.CreateOrUpdateIssue(context.Background(), tt.jql,
				WithSummary("Updated Test Issue"),
				WithProject("TEST"))

			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(issue).ToNot(BeNil())
				g.Expect(issue.Key).To(Equal(tt.expectedIssueKey))
			}
		})
	}
}

func TestClient_IsCompletedIssue(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name           string
		issue          jira.Issue
		expectComplete bool
	}{
		{
			name: "completed issue",
			issue: jira.Issue{
				Fields: &jira.IssueFields{
					Status: &jira.Status{
						StatusCategory: jira.StatusCategory{
							Key: StatusCategoryDone,
						},
					},
				},
			},
			expectComplete: true,
		},
		{
			name: "not completed issue",
			issue: jira.Issue{
				Fields: &jira.IssueFields{
					Status: &jira.Status{
						StatusCategory: jira.StatusCategory{
							Key: "new",
						},
					},
				},
			},
			expectComplete: false,
		},
		{
			name:           "nil fields",
			issue:          jira.Issue{},
			expectComplete: true,
		},
		{
			name: "nil status",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
			},
			expectComplete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup simple client for calling IsCompletedIssue
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			client, err := NewClient(ts.URL, "user", "pass")
			g.Expect(err).ToNot(HaveOccurred())

			result := client.IsCompletedIssue(tt.issue)
			g.Expect(result).To(Equal(tt.expectComplete))
		})
	}
}
