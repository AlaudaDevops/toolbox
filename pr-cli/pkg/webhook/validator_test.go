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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateGitHubSignature(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"test": "data"}`)

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name        string
		payload     []byte
		signature   string
		secret      string
		expectError bool
	}{
		{
			name:        "valid signature",
			payload:     payload,
			signature:   validSignature,
			secret:      secret,
			expectError: false,
		},
		{
			name:        "invalid signature",
			payload:     payload,
			signature:   "sha256=invalid",
			secret:      secret,
			expectError: true,
		},
		{
			name:        "wrong secret",
			payload:     payload,
			signature:   validSignature,
			secret:      "wrong-secret",
			expectError: true,
		},
		{
			name:        "missing sha256 prefix",
			payload:     payload,
			signature:   hex.EncodeToString(mac.Sum(nil)),
			secret:      secret,
			expectError: true,
		},
		{
			name:        "empty signature",
			payload:     payload,
			signature:   "",
			secret:      secret,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitHubSignature(tt.payload, tt.signature, tt.secret)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGitLabToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		secret      string
		expectError bool
	}{
		{
			name:        "valid token",
			token:       "test-token",
			secret:      "test-token",
			expectError: false,
		},
		{
			name:        "invalid token",
			token:       "wrong-token",
			secret:      "test-token",
			expectError: true,
		},
		{
			name:        "empty token",
			token:       "",
			secret:      "test-token",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitLabToken(tt.token, tt.secret)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRepository(t *testing.T) {
	tests := []struct {
		name         string
		owner        string
		repo         string
		allowedRepos []string
		expectError  bool
	}{
		{
			name:         "exact match",
			owner:        "myorg",
			repo:         "myrepo",
			allowedRepos: []string{"myorg/myrepo"},
			expectError:  false,
		},
		{
			name:         "wildcard match",
			owner:        "myorg",
			repo:         "myrepo",
			allowedRepos: []string{"myorg/*"},
			expectError:  false,
		},
		{
			name:         "multiple repos - match",
			owner:        "myorg",
			repo:         "repo2",
			allowedRepos: []string{"myorg/repo1", "myorg/repo2"},
			expectError:  false,
		},
		{
			name:         "no match",
			owner:        "otherorg",
			repo:         "myrepo",
			allowedRepos: []string{"myorg/myrepo"},
			expectError:  true,
		},
		{
			name:         "wildcard no match",
			owner:        "otherorg",
			repo:         "myrepo",
			allowedRepos: []string{"myorg/*"},
			expectError:  true,
		},
		{
			name:         "empty allowlist (allow all)",
			owner:        "anyorg",
			repo:         "anyrepo",
			allowedRepos: []string{},
			expectError:  false,
		},
		{
			name:         "nil allowlist (allow all)",
			owner:        "anyorg",
			repo:         "anyrepo",
			allowedRepos: nil,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepository(tt.owner, tt.repo, tt.allowedRepos)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateWebhookEvent(t *testing.T) {
	tests := []struct {
		name        string
		event       *WebhookEvent
		expectError bool
	}{
		{
			name: "valid event",
			event: &WebhookEvent{
				Platform: "github",
				Repository: Repository{
					Owner: "owner",
					Name:  "repo",
				},
				PullRequest: PullRequest{
					Number: 1,
				},
				Comment: Comment{
					Body: "/lgtm",
				},
				Sender: User{
					Login: "user",
				},
			},
			expectError: false,
		},
		{
			name: "missing platform",
			event: &WebhookEvent{
				Repository: Repository{
					Owner: "owner",
					Name:  "repo",
				},
			},
			expectError: true,
		},
		{
			name: "missing repository owner",
			event: &WebhookEvent{
				Platform: "github",
				Repository: Repository{
					Name: "repo",
				},
			},
			expectError: true,
		},
		{
			name: "missing repository name",
			event: &WebhookEvent{
				Platform: "github",
				Repository: Repository{
					Owner: "owner",
				},
			},
			expectError: true,
		},
		{
			name: "missing PR number",
			event: &WebhookEvent{
				Platform: "github",
				Repository: Repository{
					Owner: "owner",
					Name:  "repo",
				},
				PullRequest: PullRequest{
					Number: 0,
				},
			},
			expectError: true,
		},
		{
			name: "missing comment body",
			event: &WebhookEvent{
				Platform: "github",
				Repository: Repository{
					Owner: "owner",
					Name:  "repo",
				},
				PullRequest: PullRequest{
					Number: 1,
				},
				Comment: Comment{
					Body: "",
				},
			},
			expectError: true,
		},
		{
			name: "missing sender",
			event: &WebhookEvent{
				Platform: "github",
				Repository: Repository{
					Owner: "owner",
					Name:  "repo",
				},
				PullRequest: PullRequest{
					Number: 1,
				},
				Comment: Comment{
					Body: "/lgtm",
				},
				Sender: User{
					Login: "",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookEvent(tt.event)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
