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

	if r.Header.Get("X-GitHub-Event") != "" {
		platform = "github"
		eventType = r.Header.Get("X-GitHub-Event")
	} else if r.Header.Get("X-Gitlab-Event") != "" {
		platform = "gitlab"
		eventType = r.Header.Get("X-Gitlab-Event")
	} else {
		s.logger.Warn("Unknown webhook source (missing platform headers)")
		http.Error(w, "Unknown webhook source", http.StatusBadRequest)
		WebhookRequestsTotal.WithLabelValues("unknown", "unknown", "error").Inc()
		return
	}

	// Validate signature
	if s.config.RequireSignature {
		if platform == "github" {
			signature := r.Header.Get("X-Hub-Signature-256")
			if err := ValidateGitHubSignature(body, signature, s.config.WebhookSecret); err != nil {
				s.logger.Warnf("GitHub signature validation failed: %v", err)
				http.Error(w, "Signature validation failed", http.StatusUnauthorized)
				WebhookRequestsTotal.WithLabelValues(platform, eventType, "unauthorized").Inc()
				return
			}
		} else if platform == "gitlab" {
			token := r.Header.Get("X-Gitlab-Token")
			if err := ValidateGitLabToken(token, s.config.WebhookSecret); err != nil {
				s.logger.Warnf("GitLab token validation failed: %v", err)
				http.Error(w, "Token validation failed", http.StatusUnauthorized)
				WebhookRequestsTotal.WithLabelValues(platform, eventType, "unauthorized").Inc()
				return
			}
		}
	}

	// Parse webhook payload
	var event *WebhookEvent
	if platform == "github" {
		event, err = ParseGitHubWebhook(body, eventType)
	} else if platform == "gitlab" {
		event, err = ParseGitLabWebhook(body, eventType)
	}

	if err != nil {
		// This is expected for non-PR comments or non-created actions
		s.logger.Debugf("Webhook parsing skipped: %v", err)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK (skipped: %v)", err)
		WebhookRequestsTotal.WithLabelValues(platform, eventType, "skipped").Inc()
		return
	}

	// Validate event
	if err := ValidateWebhookEvent(event); err != nil {
		s.logger.Warnf("Invalid webhook event: %v", err)
		http.Error(w, fmt.Sprintf("Invalid webhook event: %v", err), http.StatusBadRequest)
		WebhookRequestsTotal.WithLabelValues(platform, eventType, "invalid").Inc()
		return
	}

	// Check if comment contains a command
	if !event.IsCommandComment() {
		s.logger.Debugf("Comment does not contain a command: %s", event.Comment.Body)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK (not a command)")
		WebhookRequestsTotal.WithLabelValues(platform, eventType, "not_command").Inc()
		return
	}

	// Validate repository
	if err := ValidateRepository(event.Repository.Owner, event.Repository.Name, s.config.AllowedRepos); err != nil {
		s.logger.Warnf("Repository not allowed: %v", err)
		http.Error(w, "Repository not allowed", http.StatusForbidden)
		WebhookRequestsTotal.WithLabelValues(platform, eventType, "forbidden").Inc()
		return
	}

	// Log the event
	s.logger.WithFields(logrus.Fields{
		"platform":   event.Platform,
		"repository": fmt.Sprintf("%s/%s", event.Repository.Owner, event.Repository.Name),
		"pr_number":  event.PullRequest.Number,
		"command":    event.Comment.Body,
		"sender":     event.Sender.Login,
	}).Info("Received webhook event")

	// Process webhook (async or sync)
	if s.config.AsyncProcessing {
		// Enqueue job
		job := &WebhookJob{
			Event:     event,
			Timestamp: time.Now(),
		}

		select {
		case s.jobQueue <- job:
			s.logger.Debug("Job enqueued successfully")
			QueueSize.Set(float64(len(s.jobQueue)))
		default:
			s.logger.Error("Job queue is full")
			http.Error(w, "Server busy, please try again later", http.StatusServiceUnavailable)
			WebhookRequestsTotal.WithLabelValues(platform, eventType, "queue_full").Inc()
			return
		}
	} else {
		// Process synchronously
		if err := s.processWebhookSync(event); err != nil {
			s.logger.Errorf("Failed to process webhook: %v", err)
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
