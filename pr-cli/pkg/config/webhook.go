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

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// WebhookConfig holds webhook server configuration
type WebhookConfig struct {
	// Server settings
	ListenAddr  string `json:"listen_addr" yaml:"listen_addr" mapstructure:"listen-addr"`
	WebhookPath string `json:"webhook_path" yaml:"webhook_path" mapstructure:"webhook-path"`
	HealthPath  string `json:"health_path" yaml:"health_path" mapstructure:"health-path"`
	MetricsPath string `json:"metrics_path" yaml:"metrics_path" mapstructure:"metrics-path"`

	// Security
	WebhookSecret    string   `json:"-" yaml:"-" mapstructure:"webhook-secret"`
	AllowedRepos     []string `json:"allowed_repos" yaml:"allowed_repos" mapstructure:"allowed-repos"`
	RequireSignature bool     `json:"require_signature" yaml:"require_signature" mapstructure:"require-signature"`

	// TLS
	TLSEnabled  bool   `json:"tls_enabled" yaml:"tls_enabled" mapstructure:"tls-enabled"`
	TLSCertFile string `json:"tls_cert_file" yaml:"tls_cert_file" mapstructure:"tls-cert-file"`
	TLSKeyFile  string `json:"tls_key_file" yaml:"tls_key_file" mapstructure:"tls-key-file"`

	// Processing
	AsyncProcessing bool `json:"async_processing" yaml:"async_processing" mapstructure:"async-processing"`
	WorkerCount     int  `json:"worker_count" yaml:"worker_count" mapstructure:"worker-count"`
	QueueSize       int  `json:"queue_size" yaml:"queue_size" mapstructure:"queue-size"`

	// Rate limiting
	RateLimitEnabled  bool `json:"rate_limit_enabled" yaml:"rate_limit_enabled" mapstructure:"rate-limit-enabled"`
	RateLimitRequests int  `json:"rate_limit_requests" yaml:"rate_limit_requests" mapstructure:"rate-limit-requests"`

	// Pull Request event configuration
	PREventEnabled bool     `json:"pr_event_enabled" yaml:"pr_event_enabled" mapstructure:"pr-event-enabled"`
	PREventActions []string `json:"pr_event_actions" yaml:"pr_event_actions" mapstructure:"pr-event-actions"`

	// Workflow dispatch configuration
	WorkflowFile   string            `json:"workflow_file" yaml:"workflow_file" mapstructure:"workflow-file"`
	WorkflowRef    string            `json:"workflow_ref" yaml:"workflow_ref" mapstructure:"workflow-ref"`
	WorkflowInputs map[string]string `json:"workflow_inputs" yaml:"workflow_inputs" mapstructure:"workflow-inputs"`

	// Base PR CLI configuration
	BaseConfig *Config `json:"-" yaml:"-"`
}

// NewDefaultWebhookConfig returns default webhook configuration
func NewDefaultWebhookConfig() *WebhookConfig {
	return &WebhookConfig{
		ListenAddr:        ":8080",
		WebhookPath:       "/webhook",
		HealthPath:        "/health",
		MetricsPath:       "/metrics",
		RequireSignature:  true,
		TLSEnabled:        false,
		AsyncProcessing:   true,
		WorkerCount:       10,
		QueueSize:         100,
		RateLimitEnabled:  true,
		RateLimitRequests: 100,
		PREventEnabled:    false,
		PREventActions:    []string{"opened", "synchronize", "reopened", "ready_for_review", "edited"},
		WorkflowRef:       "main",
		BaseConfig:        NewDefaultConfig(),
	}
}

// LoadFromEnv loads webhook configuration from environment variables
func (wc *WebhookConfig) LoadFromEnv() error {
	// Server settings
	if addr := os.Getenv("LISTEN_ADDR"); addr != "" {
		wc.ListenAddr = addr
	}
	if path := os.Getenv("WEBHOOK_PATH"); path != "" {
		wc.WebhookPath = path
	}
	if path := os.Getenv("HEALTH_PATH"); path != "" {
		wc.HealthPath = path
	}
	if path := os.Getenv("METRICS_PATH"); path != "" {
		wc.MetricsPath = path
	}

	// Security
	if secret := os.Getenv("WEBHOOK_SECRET"); secret != "" {
		wc.WebhookSecret = secret
	} else if secretFile := os.Getenv("WEBHOOK_SECRET_FILE"); secretFile != "" {
		data, err := os.ReadFile(secretFile)
		if err != nil {
			return fmt.Errorf("failed to read webhook secret file: %w", err)
		}
		wc.WebhookSecret = strings.TrimSpace(string(data))
	}

	if repos := os.Getenv("ALLOWED_REPOS"); repos != "" {
		wc.AllowedRepos = strings.Split(repos, ",")
		for i := range wc.AllowedRepos {
			wc.AllowedRepos[i] = strings.TrimSpace(wc.AllowedRepos[i])
		}
	}

	if requireSig := os.Getenv("REQUIRE_SIGNATURE"); requireSig != "" {
		wc.RequireSignature = requireSig == "true"
	}

	// TLS
	if tlsEnabled := os.Getenv("TLS_ENABLED"); tlsEnabled != "" {
		wc.TLSEnabled = tlsEnabled == "true"
	}
	if certFile := os.Getenv("TLS_CERT_FILE"); certFile != "" {
		wc.TLSCertFile = certFile
	}
	if keyFile := os.Getenv("TLS_KEY_FILE"); keyFile != "" {
		wc.TLSKeyFile = keyFile
	}

	// Processing
	if async := os.Getenv("ASYNC_PROCESSING"); async != "" {
		wc.AsyncProcessing = async == "true"
	}
	if workers := os.Getenv("WORKER_COUNT"); workers != "" {
		if count, err := strconv.Atoi(workers); err == nil {
			wc.WorkerCount = count
		}
	}
	if queueSize := os.Getenv("QUEUE_SIZE"); queueSize != "" {
		if size, err := strconv.Atoi(queueSize); err == nil {
			wc.QueueSize = size
		}
	}

	// Rate limiting
	if rateLimitEnabled := os.Getenv("RATE_LIMIT_ENABLED"); rateLimitEnabled != "" {
		wc.RateLimitEnabled = rateLimitEnabled == "true"
	}
	if rateLimitReqs := os.Getenv("RATE_LIMIT_REQUESTS"); rateLimitReqs != "" {
		if reqs, err := strconv.Atoi(rateLimitReqs); err == nil {
			wc.RateLimitRequests = reqs
		}
	}

	// Pull Request event configuration
	if prEventEnabled := os.Getenv("PR_EVENT_ENABLED"); prEventEnabled != "" {
		wc.PREventEnabled = prEventEnabled == "true"
	}
	if prEventActions := os.Getenv("PR_EVENT_ACTIONS"); prEventActions != "" {
		wc.PREventActions = strings.Split(prEventActions, ",")
		for i := range wc.PREventActions {
			wc.PREventActions[i] = strings.TrimSpace(wc.PREventActions[i])
		}
	}

	// Workflow dispatch configuration
	if workflowFile := os.Getenv("WORKFLOW_FILE"); workflowFile != "" {
		wc.WorkflowFile = workflowFile
	}
	if workflowRef := os.Getenv("WORKFLOW_REF"); workflowRef != "" {
		wc.WorkflowRef = workflowRef
	}
	if workflowInputs := os.Getenv("WORKFLOW_INPUTS"); workflowInputs != "" {
		// Parse as key=value,key=value format
		wc.WorkflowInputs = make(map[string]string)
		pairs := strings.Split(workflowInputs, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(kv) == 2 {
				wc.WorkflowInputs[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	return nil
}

// Validate checks if the webhook configuration is valid
func (wc *WebhookConfig) Validate() error {
	if wc.ListenAddr == "" {
		return fmt.Errorf("listen address is required")
	}
	if wc.WebhookPath == "" {
		return fmt.Errorf("webhook path is required")
	}
	if wc.RequireSignature && wc.WebhookSecret == "" {
		return fmt.Errorf("webhook secret is required when signature validation is enabled")
	}
	if wc.TLSEnabled {
		if wc.TLSCertFile == "" || wc.TLSKeyFile == "" {
			return fmt.Errorf("TLS cert and key files are required when TLS is enabled")
		}
	}
	if wc.WorkerCount < 1 {
		return fmt.Errorf("worker count must be at least 1")
	}
	if wc.QueueSize < 1 {
		return fmt.Errorf("queue size must be at least 1")
	}
	if wc.RateLimitEnabled && wc.RateLimitRequests < 1 {
		return fmt.Errorf("rate limit requests must be at least 1")
	}
	if wc.PREventEnabled {
		trimmedWorkflowFile := strings.TrimSpace(wc.WorkflowFile)
		if trimmedWorkflowFile == "" {
			return fmt.Errorf("workflow file is required when PR event handling is enabled")
		}
		if !isValidWorkflowPath(trimmedWorkflowFile) {
			return fmt.Errorf("workflow file path %q is invalid: must be a valid file path (e.g., .github/workflows/pr-check.yml)", wc.WorkflowFile)
		}
		wc.WorkflowFile = trimmedWorkflowFile
	}
	if wc.BaseConfig == nil {
		return fmt.Errorf("base config is required")
	}
	return nil
}

func isValidWorkflowPath(path string) bool {
	if path == "" {
		return false
	}
	if strings.Contains(path, "..") {
		return false
	}
	return strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml")
}

// DebugString returns a string representation with sensitive data redacted
func (wc *WebhookConfig) DebugString() string {
	return fmt.Sprintf("WebhookConfig{ListenAddr: %s, WebhookPath: %s, RequireSignature: %v, AsyncProcessing: %v, WorkerCount: %d, QueueSize: %d}",
		wc.ListenAddr, wc.WebhookPath, wc.RequireSignature, wc.AsyncProcessing, wc.WorkerCount, wc.QueueSize)
}
