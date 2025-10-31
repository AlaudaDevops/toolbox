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
	"fmt"
	"strings"
)

// ValidateGitHubSignature validates the GitHub webhook signature
func ValidateGitHubSignature(payload []byte, signature string, secret string) error {
	if signature == "" {
		return fmt.Errorf("missing signature header")
	}

	// GitHub sends signature as "sha256=<hex>"
	if !strings.HasPrefix(signature, "sha256=") {
		return fmt.Errorf("invalid signature format")
	}

	// Extract hex signature
	receivedSig := strings.TrimPrefix(signature, "sha256=")

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures using constant-time comparison
	if !hmac.Equal([]byte(receivedSig), []byte(expectedSig)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// ValidateGitLabToken validates the GitLab webhook token
func ValidateGitLabToken(token string, secret string) error {
	if token == "" {
		return fmt.Errorf("missing token header")
	}

	// GitLab uses simple token comparison
	if token != secret {
		return fmt.Errorf("token mismatch")
	}

	return nil
}

// ValidateRepository checks if the repository is allowed
func ValidateRepository(owner, repo string, allowedRepos []string) error {
	// If no allowlist is configured, allow all repositories
	if len(allowedRepos) == 0 {
		return nil
	}

	fullName := fmt.Sprintf("%s/%s", owner, repo)

	for _, allowed := range allowedRepos {
		// Exact match
		if allowed == fullName {
			return nil
		}

		// Wildcard match for organization: "org/*"
		if strings.HasSuffix(allowed, "/*") {
			orgPrefix := strings.TrimSuffix(allowed, "/*")
			if owner == orgPrefix {
				return nil
			}
		}

		// Global wildcard: "*"
		if allowed == "*" {
			return nil
		}
	}

	return fmt.Errorf("repository %s is not in the allowed list", fullName)
}

// ValidateWebhookEvent performs basic validation on the webhook event
func ValidateWebhookEvent(event *WebhookEvent) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	if event.Platform == "" {
		return fmt.Errorf("platform is required")
	}

	if event.Repository.Owner == "" {
		return fmt.Errorf("repository owner is required")
	}

	if event.Repository.Name == "" {
		return fmt.Errorf("repository name is required")
	}

	if event.PullRequest.Number <= 0 {
		return fmt.Errorf("invalid pull request number: %d", event.PullRequest.Number)
	}

	if event.Comment.Body == "" {
		return fmt.Errorf("comment body is empty")
	}

	if event.Sender.Login == "" {
		return fmt.Errorf("sender login is required")
	}

	return nil
}
