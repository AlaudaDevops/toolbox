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
	"fmt"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
)

// WebhookEvent represents a parsed webhook event
type WebhookEvent struct {
	EventID     string
	Platform    string
	Action      string
	Repository  Repository
	PullRequest PullRequest
	Comment     Comment
	Sender      User
}

// Repository represents repository information
type Repository struct {
	Owner string
	Name  string
	URL   string
}

// PullRequest represents pull request information
type PullRequest struct {
	Number int
	State  string
	Author string
}

// Comment represents a comment
type Comment struct {
	ID   int64
	Body string
	User string
}

// User represents a user
type User struct {
	Login string
}

// GitHubWebhookPayload represents GitHub webhook payload structure
type GitHubWebhookPayload struct {
	Action string `json:"action"`
	Issue  struct {
		Number      int    `json:"number"`
		State       string `json:"state"`
		PullRequest *struct {
		} `json:"pull_request,omitempty"` // This field exists only for PRs
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"issue"`
	Comment struct {
		ID   int64  `json:"id"`
		Body string `json:"body"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"comment"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		HTMLURL string `json:"html_url"`
	} `json:"repository"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}

// GitHubPullRequestPayload represents GitHub pull_request webhook payload structure
type GitHubPullRequestPayload struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		Number int    `json:"number"`
		State  string `json:"state"`
		Title  string `json:"title"`
		Draft  bool   `json:"draft"`
		User   struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	} `json:"pull_request"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		HTMLURL string `json:"html_url"`
	} `json:"repository"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}

// PRWebhookEvent represents a parsed pull_request webhook event
type PRWebhookEvent struct {
	EventID     string
	Platform    string
	Action      string
	Repository  Repository
	PullRequest PRInfo
	Sender      User
}

// PRInfo represents pull request information from pull_request event
type PRInfo struct {
	Number  int
	State   string
	Title   string
	Draft   bool
	Author  string
	HeadRef string
	HeadSHA string
	BaseRef string
}

// GitLabWebhookPayload represents GitLab webhook payload structure
type GitLabWebhookPayload struct {
	ObjectKind       string `json:"object_kind"`
	ObjectAttributes struct {
		Action       string `json:"action"`
		Note         string `json:"note"`
		NoteableType string `json:"noteable_type"`
		ID           int64  `json:"id"`
	} `json:"object_attributes"`
	MergeRequest struct {
		IID    int    `json:"iid"`
		State  string `json:"state"`
		Author struct {
			Username string `json:"username"`
		} `json:"author"`
	} `json:"merge_request"`
	Project struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		WebURL    string `json:"web_url"`
	} `json:"project"`
	User struct {
		Username string `json:"username"`
	} `json:"user"`
}

// ParseGitHubWebhook parses a GitHub webhook payload
func ParseGitHubWebhook(payload []byte, eventType string) (*WebhookEvent, error) {
	// Only process issue_comment events
	if eventType != "issue_comment" {
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}

	var ghPayload GitHubWebhookPayload
	if err := json.Unmarshal(payload, &ghPayload); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub webhook payload: %w", err)
	}

	// Only process comments on pull requests
	if ghPayload.Issue.PullRequest == nil {
		return nil, fmt.Errorf("comment is not on a pull request")
	}

	// Only process "created" action
	if ghPayload.Action != "created" {
		return nil, fmt.Errorf("ignoring action: %s (only 'created' is processed)", ghPayload.Action)
	}

	event := &WebhookEvent{
		Platform: "github",
		Action:   ghPayload.Action,
		Repository: Repository{
			Owner: ghPayload.Repository.Owner.Login,
			Name:  ghPayload.Repository.Name,
			URL:   ghPayload.Repository.HTMLURL,
		},
		PullRequest: PullRequest{
			Number: ghPayload.Issue.Number,
			State:  ghPayload.Issue.State,
			Author: ghPayload.Issue.User.Login,
		},
		Comment: Comment{
			ID:   ghPayload.Comment.ID,
			Body: ghPayload.Comment.Body,
			User: ghPayload.Comment.User.Login,
		},
		Sender: User{
			Login: ghPayload.Sender.Login,
		},
	}

	return event, nil
}

// ParseGitHubPullRequestWebhook parses a GitHub pull_request webhook payload
func ParseGitHubPullRequestWebhook(payload []byte, allowedActions []string) (*PRWebhookEvent, error) {
	var ghPayload GitHubPullRequestPayload
	if err := json.Unmarshal(payload, &ghPayload); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub pull_request payload: %w", err)
	}

	// Check if action is in allowedActions
	actionAllowed := false
	for _, action := range allowedActions {
		if ghPayload.Action == action {
			actionAllowed = true
			break
		}
	}
	if !actionAllowed {
		return nil, fmt.Errorf("action %q not in allowed actions", ghPayload.Action)
	}

	// Skip draft PRs unless action is "ready_for_review"
	if ghPayload.PullRequest.Draft && ghPayload.Action != "ready_for_review" {
		return nil, fmt.Errorf("skipping draft PR")
	}

	event := &PRWebhookEvent{
		Platform: "github",
		Action:   ghPayload.Action,
		Repository: Repository{
			Owner: ghPayload.Repository.Owner.Login,
			Name:  ghPayload.Repository.Name,
			URL:   ghPayload.Repository.HTMLURL,
		},
		PullRequest: PRInfo{
			Number:  ghPayload.PullRequest.Number,
			State:   ghPayload.PullRequest.State,
			Title:   ghPayload.PullRequest.Title,
			Draft:   ghPayload.PullRequest.Draft,
			Author:  ghPayload.PullRequest.User.Login,
			HeadRef: ghPayload.PullRequest.Head.Ref,
			HeadSHA: ghPayload.PullRequest.Head.SHA,
			BaseRef: ghPayload.PullRequest.Base.Ref,
		},
		Sender: User{
			Login: ghPayload.Sender.Login,
		},
	}

	return event, nil
}

// ParseGitLabWebhook parses a GitLab webhook payload
func ParseGitLabWebhook(payload []byte, eventType string) (*WebhookEvent, error) {
	// Only process note events
	if eventType != "Note Hook" && eventType != "note" {
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}

	var glPayload GitLabWebhookPayload
	if err := json.Unmarshal(payload, &glPayload); err != nil {
		return nil, fmt.Errorf("failed to parse GitLab webhook payload: %w", err)
	}

	// Only process merge request notes
	if glPayload.ObjectAttributes.NoteableType != "MergeRequest" {
		return nil, fmt.Errorf("note is not on a merge request")
	}

	event := &WebhookEvent{
		Platform: "gitlab",
		Action:   glPayload.ObjectAttributes.Action,
		Repository: Repository{
			Owner: glPayload.Project.Namespace,
			Name:  glPayload.Project.Name,
			URL:   glPayload.Project.WebURL,
		},
		PullRequest: PullRequest{
			Number: glPayload.MergeRequest.IID,
			State:  glPayload.MergeRequest.State,
			Author: glPayload.MergeRequest.Author.Username,
		},
		Comment: Comment{
			ID:   glPayload.ObjectAttributes.ID,
			Body: glPayload.ObjectAttributes.Note,
			User: glPayload.User.Username,
		},
		Sender: User{
			Login: glPayload.User.Username,
		},
	}

	return event, nil
}

// ToConfig converts a WebhookEvent to a Config for PR handler
func (e *WebhookEvent) ToConfig(baseConfig *config.Config) *config.Config {
	cfg := *baseConfig // Copy base config

	// Override with event-specific values
	cfg.Platform = e.Platform
	cfg.Owner = e.Repository.Owner
	cfg.Repo = e.Repository.Name
	cfg.PRNum = e.PullRequest.Number
	cfg.CommentSender = e.Comment.User
	cfg.TriggerComment = e.Comment.Body

	return &cfg
}

// IsCommandComment checks if the comment body contains a command
func (e *WebhookEvent) IsCommandComment() bool {
	if e.Comment.Body == "" {
		return false
	}
	// Check if comment starts with "/"
	body := e.Comment.Body
	for i, r := range body {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		if i < len(body) && body[i] == '/' {
			return true
		}
		break
	}
	return false
}
