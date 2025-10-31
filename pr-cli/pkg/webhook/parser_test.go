/*
Copyright 2025 The AlaudaDevops Authors.

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

package webhook

import (
	"encoding/json"
	"testing"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitHubWebhook(t *testing.T) {
	tests := []struct {
		name        string
		eventType   string
		payload     string
		expectError bool
		validate    func(t *testing.T, event *WebhookEvent)
	}{
		{
			name:      "valid issue_comment event",
			eventType: "issue_comment",
			payload: `{
				"action": "created",
				"issue": {
					"number": 123,
					"pull_request": {},
					"user": {
						"login": "pr-author"
					}
				},
				"comment": {
					"id": 456,
					"body": "/lgtm",
					"user": {
						"login": "testuser"
					}
				},
				"repository": {
					"name": "test-repo",
					"html_url": "https://github.com/test-owner/test-repo",
					"owner": {
						"login": "test-owner"
					}
				},
				"sender": {
					"login": "testuser"
				}
			}`,
			expectError: false,
			validate: func(t *testing.T, event *WebhookEvent) {
				assert.Equal(t, "github", event.Platform)
				assert.Equal(t, "created", event.Action)
				assert.Equal(t, 123, event.PullRequest.Number)
				assert.Equal(t, "/lgtm", event.Comment.Body)
				assert.Equal(t, "testuser", event.Sender.Login)
				assert.Equal(t, "test-repo", event.Repository.Name)
				assert.Equal(t, "test-owner", event.Repository.Owner)
			},
		},
		{
			name:      "non-PR comment (should fail)",
			eventType: "issue_comment",
			payload: `{
				"action": "created",
				"issue": {
					"number": 123
				},
				"comment": {
					"body": "/lgtm"
				}
			}`,
			expectError: true,
		},
		{
			name:      "edited comment (should fail)",
			eventType: "issue_comment",
			payload: `{
				"action": "edited",
				"issue": {
					"number": 123,
					"pull_request": {}
				},
				"comment": {
					"body": "/lgtm"
				}
			}`,
			expectError: true,
		},
		{
			name:        "unsupported event type",
			eventType:   "push",
			payload:     `{}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseGitHubWebhook([]byte(tt.payload), tt.eventType)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, event)
				if tt.validate != nil {
					tt.validate(t, event)
				}
			}
		})
	}
}

func TestParseGitLabWebhook(t *testing.T) {
	tests := []struct {
		name        string
		eventType   string
		payload     string
		expectError bool
		validate    func(t *testing.T, event *WebhookEvent)
	}{
		{
			name:      "valid note event",
			eventType: "note",
			payload: `{
				"object_kind": "note",
				"merge_request": {
					"iid": 123,
					"state": "opened",
					"author": {
						"username": "pr-author"
					}
				},
				"object_attributes": {
					"id": 456,
					"note": "/lgtm",
					"noteable_type": "MergeRequest"
				},
				"user": {
					"username": "testuser"
				},
				"project": {
					"name": "test-repo",
					"namespace": "test-owner",
					"web_url": "https://gitlab.com/test-owner/test-repo"
				}
			}`,
			expectError: false,
			validate: func(t *testing.T, event *WebhookEvent) {
				assert.Equal(t, "gitlab", event.Platform)
				assert.Equal(t, 123, event.PullRequest.Number)
				assert.Equal(t, "/lgtm", event.Comment.Body)
				assert.Equal(t, "testuser", event.Sender.Login)
				assert.Equal(t, "test-repo", event.Repository.Name)
				assert.Equal(t, "test-owner", event.Repository.Owner)
			},
		},
		{
			name:      "non-MR note (should fail)",
			eventType: "note",
			payload: `{
				"object_kind": "note",
				"object_attributes": {
					"note": "/lgtm",
					"noteable_type": "Issue"
				}
			}`,
			expectError: true,
		},
		{
			name:        "unsupported event type",
			eventType:   "push",
			payload:     `{}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseGitLabWebhook([]byte(tt.payload), tt.eventType)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, event)
				if tt.validate != nil {
					tt.validate(t, event)
				}
			}
		})
	}
}

func TestIsCommandComment(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{"valid command", "/lgtm", true},
		{"command with args", "/assign @user", true},
		{"not a command", "This is a regular comment", false},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"command with leading whitespace", "  /lgtm", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &WebhookEvent{
				Comment: Comment{
					Body: tt.body,
				},
			}
			result := event.IsCommandComment()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToConfig(t *testing.T) {
	event := &WebhookEvent{
		Platform: "github",
		Action:   "created",
		Repository: Repository{
			Owner: "test-owner",
			Name:  "test-repo",
		},
		PullRequest: PullRequest{
			Number: 123,
		},
		Comment: Comment{
			ID:   456,
			Body: "/lgtm",
			User: "testuser",
		},
		Sender: User{
			Login: "testuser",
		},
	}

	baseConfig := &config.Config{
		Platform: "github",
		Token:    "test-token",
		Verbose:  true,
	}

	cfg := event.ToConfig(baseConfig)

	assert.Equal(t, "github", cfg.Platform)
	assert.Equal(t, "test-token", cfg.Token)
	assert.Equal(t, "test-owner", cfg.Owner)
	assert.Equal(t, "test-repo", cfg.Repo)
	assert.Equal(t, 123, cfg.PRNum)
	assert.Equal(t, "/lgtm", cfg.TriggerComment)
	assert.Equal(t, "testuser", cfg.CommentSender)
	assert.True(t, cfg.Verbose)
}

func TestParseSimpleCommand(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		expectedCommand string
		expectedArgs    []string
	}{
		{
			name:            "simple command",
			body:            "/lgtm",
			expectedCommand: "lgtm",
			expectedArgs:    []string{},
		},
		{
			name:            "command with args",
			body:            "/assign @user1 @user2",
			expectedCommand: "assign",
			expectedArgs:    []string{"@user1", "@user2"},
		},
		{
			name:            "command with leading whitespace",
			body:            "  /merge squash",
			expectedCommand: "merge",
			expectedArgs:    []string{"squash"},
		},
		{
			name:            "not a command",
			body:            "This is not a command",
			expectedCommand: "",
			expectedArgs:    nil,
		},
		{
			name:            "empty string",
			body:            "",
			expectedCommand: "",
			expectedArgs:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, args := parseSimpleCommand(tt.body)
			assert.Equal(t, tt.expectedCommand, command)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestExtractCommand(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{"simple command", "/lgtm", "lgtm"},
		{"command with args", "/assign @user", "assign"},
		{"not a command", "regular comment", "unknown"},
		{"empty", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCommand(tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWebhookEventJSON(t *testing.T) {
	event := &WebhookEvent{
		Platform: "github",
		Action:   "created",
		Repository: Repository{
			Owner: "owner",
			Name:  "repo",
		},
		PullRequest: PullRequest{
			Number: 1,
		},
		Comment: Comment{
			ID:   100,
			Body: "/test",
			User: "user",
		},
		Sender: User{
			Login: "user",
		},
	}

	// Test marshaling
	data, err := json.Marshal(event)
	require.NoError(t, err)

	// Test unmarshaling
	var decoded WebhookEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.Platform, decoded.Platform)
	assert.Equal(t, event.Action, decoded.Action)
	assert.Equal(t, event.Repository.Owner, decoded.Repository.Owner)
	assert.Equal(t, event.PullRequest.Number, decoded.PullRequest.Number)
}
