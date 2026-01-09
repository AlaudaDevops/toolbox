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
	"io"
	"net/http"
	"time"

	"github.com/AlaudaDevops/toolbox/pr-cli/internal/version"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/platforms/github"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// handleWebhook processes incoming webhook requests
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}


	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Errorf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Determine platform from headers
	platform := ""
	eventType := ""
	eventID := ""

	if r.Header.Get("X-GitHub-Event") != "" {
		platform = "github"
		eventType = r.Header.Get("X-GitHub-Event")
		eventID = r.Header.Get("X-GitHub-Delivery")
	} else if r.Header.Get("X-Gitlab-Event") != "" {
		platform = "gitlab"
		eventType = r.Header.Get("X-Gitlab-Event")
		eventID = r.Header.Get("X-Gitlab-Delivery")
	} else {
		s.logger.Warn("Unknown webhook source (missing platform headers)")
		http.Error(w, "Unknown webhook source", http.StatusBadRequest)
		WebhookRequestsTotal.WithLabelValues("unknown", "unknown", "error").Inc()
		return
	}
	if eventID == "" {
		eventID = uuid.NewString()
		s.logger.Infof("No eventID found in webhook headers, generated a new one: %q", eventID)
	}
	logger := s.logger.WithFields(logrus.Fields{
		"event_id": eventID,
		"platform": platform,
		"event_type": eventType,
	})
	s.logger.Infof("Received webhook event %q of type %q from %s", eventID, eventType, platform)

	// Validate signature
	if s.config.RequireSignature {
		switch platform {
		case "github":
			signature := r.Header.Get("X-Hub-Signature-256")
			if err := ValidateGitHubSignature(body, signature, s.config.WebhookSecret); err != nil {
				logger.Warnf("GitHub signature validation failed: %v", err)
				http.Error(w, "Signature validation failed", http.StatusUnauthorized)
				WebhookRequestsTotal.WithLabelValues(platform, eventType, "unauthorized").Inc()
				return
			}
		case "gitlab":
			token := r.Header.Get("X-Gitlab-Token")
			if err := ValidateGitLabToken(token, s.config.WebhookSecret); err != nil {
				logger.Warnf("GitLab token validation failed: %v", err)
				http.Error(w, "Token validation failed", http.StatusUnauthorized)
				WebhookRequestsTotal.WithLabelValues(platform, eventType, "unauthorized").Inc()
				return
			}
		}
	}

	// Handle pull_request events separately
	if platform == "github" && eventType == "pull_request" {
		s.handlePullRequestEvent(w, r, body, logger, platform, eventType, startTime)
		return
	}

	// Parse webhook payload for issue_comment events
	var event *WebhookEvent
	switch platform {
	case "github":
		event, err = ParseGitHubWebhook(body, eventType)
	case "gitlab":
		event, err = ParseGitLabWebhook(body, eventType)
	}
	if err != nil {
		// This is expected for non-PR comments or non-created actions
		logger.Debugf("Webhook parsing skipped: %v", err)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK (skipped: %v)", err)
		WebhookRequestsTotal.WithLabelValues(platform, eventType, "skipped").Inc()
		return
	}
	//attributing the event ID
	event.EventID = eventID

	// Validate event
	if err := ValidateWebhookEvent(event); err != nil {
		logger.Warnf("Invalid webhook event: %v", err)
		http.Error(w, fmt.Sprintf("Invalid webhook event: %v", err), http.StatusBadRequest)
		WebhookRequestsTotal.WithLabelValues(platform, eventType, "invalid").Inc()
		return
	}

	// Check if comment contains a command
	if !event.IsCommandComment() {
		logger.Debugf("Comment does not contain a command: %s", event.Comment.Body)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK (not a command)")
		WebhookRequestsTotal.WithLabelValues(platform, eventType, "not_command").Inc()
		return
	}

	// Validate repository
	if err := ValidateRepository(event.Repository.Owner, event.Repository.Name, s.config.AllowedRepos); err != nil {
		logger.Warnf("Repository not allowed: %v", err)
		http.Error(w, "Repository not allowed", http.StatusForbidden)
		WebhookRequestsTotal.WithLabelValues(platform, eventType, "forbidden").Inc()
		return
	}

	// Log the event
	logger = logger.WithFields(logrus.Fields{
		"platform":   event.Platform,
		"repository": fmt.Sprintf("%s/%s", event.Repository.Owner, event.Repository.Name),
		"pr_number":  event.PullRequest.Number,
		"command":    event.Comment.Body,
		"sender":     event.Sender.Login,
	})
	logger.Info("Received webhook event")

	// Process webhook (async or sync)
	if s.config.AsyncProcessing {
		// Enqueue job
		job := &WebhookJob{
			Event:     event,
			Timestamp: time.Now(),
		}

		select {
		case s.jobQueue <- job:
			logger.Debug("Job enqueued successfully")
			QueueSize.Set(float64(len(s.jobQueue)))
		default:
			logger.Error("Job queue is full")
			http.Error(w, "Server busy, please try again later", http.StatusServiceUnavailable)
			WebhookRequestsTotal.WithLabelValues(platform, eventType, "queue_full").Inc()
			return
		}
	} else {
		// Process synchronously
		if err := s.processWebhookSync(event); err != nil {
			logger.Errorf("Failed to process webhook: %v", err)
			http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
			WebhookRequestsTotal.WithLabelValues(platform, eventType, "error").Inc()
			return
		}
	}

	// Success response
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
	WebhookRequestsTotal.WithLabelValues(platform, eventType, "success").Inc()
	WebhookProcessingDuration.WithLabelValues(platform, extractCommand(event.Comment.Body)).Observe(time.Since(startTime).Seconds())
}

// handleHealth returns health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":         "healthy",
		"version":        version.Get().Version,
		"uptime":         time.Since(s.startTime).String(),
		"queue_size":     len(s.jobQueue),
		"queue_capacity": cap(s.jobQueue),
		"workers":        s.config.WorkerCount,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// handleReadiness returns readiness status
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	// Check if queue is not full
	queueUsage := float64(len(s.jobQueue)) / float64(cap(s.jobQueue))
	if queueUsage > 0.95 {
		http.Error(w, "Queue nearly full", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// handlePullRequestEvent handles GitHub pull_request webhook events
func (s *Server) handlePullRequestEvent(w http.ResponseWriter, r *http.Request, body []byte, logger *logrus.Entry, platform, eventType string, startTime time.Time) {
	// Check if PR event handling is enabled
	if !s.config.PREventEnabled {
		logger.Debug("pull_request events disabled, skipping")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK (pull_request events disabled)")
		WebhookRequestsTotal.WithLabelValues(platform, eventType, "disabled").Inc()
		return
	}

	// Parse pull_request webhook payload
	prEvent, err := ParseGitHubPullRequestWebhook(body, s.config.PREventActions)
	if err != nil {
		logger.Debugf("PR webhook parsing skipped: %v", err)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK (skipped: %v)", err)
		WebhookRequestsTotal.WithLabelValues(platform, eventType, "skipped").Inc()
		return
	}

	// Validate repository
	if err := ValidateRepository(prEvent.Repository.Owner, prEvent.Repository.Name, s.config.AllowedRepos); err != nil {
		logger.Warnf("Repository not allowed: %v", err)
		http.Error(w, "Repository not allowed", http.StatusForbidden)
		WebhookRequestsTotal.WithLabelValues(platform, eventType, "forbidden").Inc()
		return
	}

	// Log the event
	logger = logger.WithFields(logrus.Fields{
		"repository": fmt.Sprintf("%s/%s", prEvent.Repository.Owner, prEvent.Repository.Name),
		"pr_number":  prEvent.PullRequest.Number,
		"pr_action":  prEvent.Action,
		"sender":     prEvent.Sender.Login,
	})
	logger.Info("Received pull_request webhook event")

	// Process the PR event (trigger workflow)
	if err := s.processPullRequestEvent(prEvent); err != nil {
		logger.Errorf("Failed to process pull_request event: %v", err)
		http.Error(w, "Failed to process pull_request event", http.StatusInternalServerError)
		PREventProcessingTotal.WithLabelValues(platform, prEvent.Action, "error").Inc()
		return
	}

	// Success response
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
	PREventProcessingTotal.WithLabelValues(platform, prEvent.Action, "success").Inc()
	WebhookRequestsTotal.WithLabelValues(platform, eventType, "success").Inc()
	WebhookProcessingDuration.WithLabelValues(platform, "pull_request").Observe(time.Since(startTime).Seconds())
}

// processPullRequestEvent triggers a workflow dispatch for a pull_request event
func (s *Server) processPullRequestEvent(event *PRWebhookEvent) error {
	// Create GitHub client configuration
	cfg := &git.Config{
		Platform: event.Platform,
		Token:    s.config.BaseConfig.Token,
		BaseURL:  s.config.BaseConfig.BaseURL,
		Owner:    event.Repository.Owner,
		Repo:     event.Repository.Name,
		PRNum:    event.PullRequest.Number,
	}

	// Create GitHub client using factory
	factory := &github.Factory{}
	client, err := factory.CreateClient(s.logger, cfg)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Cast to GitHub client to access TriggerWorkflowDispatch
	ghClient, ok := client.(*github.Client)
	if !ok {
		return fmt.Errorf("expected GitHub client, got different type")
	}

	// Build workflow inputs from PR event
	inputs := map[string]interface{}{
		"pr_number": fmt.Sprintf("%d", event.PullRequest.Number),
		"pr_action": event.Action,
		"head_ref":  event.PullRequest.HeadRef,
		"head_sha":  event.PullRequest.HeadSHA,
		"base_ref":  event.PullRequest.BaseRef,
		"sender":    event.Sender.Login,
	}

	// Merge with configured static inputs
	for k, v := range s.config.WorkflowInputs {
		inputs[k] = v
	}

	// Trigger workflow dispatch
	if err := ghClient.TriggerWorkflowDispatch(s.config.WorkflowFile, s.config.WorkflowRef, inputs); err != nil {
		WorkflowDispatchTotal.WithLabelValues(event.Platform, s.config.WorkflowFile, "error").Inc()
		return fmt.Errorf("failed to trigger workflow dispatch: %w", err)
	}

	WorkflowDispatchTotal.WithLabelValues(event.Platform, s.config.WorkflowFile, "success").Inc()
	s.logger.Infof("Successfully triggered workflow %s for PR #%d", s.config.WorkflowFile, event.PullRequest.Number)
	return nil
}
