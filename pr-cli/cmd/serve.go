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

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/webhook"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start webhook server to receive GitHub/GitLab webhooks",
	Long: `Start an HTTP server that receives webhooks from GitHub or GitLab and processes
PR comment commands. This mode eliminates the need for Tekton Pipelines and provides
faster response times with lower resource usage.

The server listens for webhook events, validates signatures, and executes PR commands
using the same logic as the CLI mode. It can also handle pull_request events to trigger
GitHub Actions workflows via workflow_dispatch.

Example:
  # Start server with default settings
  pr-cli serve

  # Start with custom configuration
  pr-cli serve --listen-addr=:8080 --webhook-secret=mysecret --allowed-repos="myorg/*"

  # Start with TLS
  pr-cli serve --tls-enabled --tls-cert-file=/etc/certs/tls.crt --tls-key-file=/etc/certs/tls.key

  # Start with PR event handling to trigger workflows
  pr-cli serve --pr-event-enabled --workflow-file=.github/workflows/pr-check.yml

Environment Variables:
  LISTEN_ADDR              Server listen address (default: :8080)
  WEBHOOK_PATH             Webhook endpoint path (default: /webhook)
  WEBHOOK_SECRET           Webhook secret for signature validation
  WEBHOOK_SECRET_FILE      File containing webhook secret
  ALLOWED_REPOS            Comma-separated list of allowed repositories (owner/repo or owner/*)
  REQUIRE_SIGNATURE        Require webhook signature validation (default: true)
  TLS_ENABLED              Enable TLS (default: false)
  TLS_CERT_FILE            TLS certificate file
  TLS_KEY_FILE             TLS private key file
  ASYNC_PROCESSING         Process webhooks asynchronously (default: true)
  WORKER_COUNT             Number of worker goroutines (default: 10)
  QUEUE_SIZE               Job queue size (default: 100)
  RATE_LIMIT_ENABLED       Enable rate limiting (default: true)
  RATE_LIMIT_REQUESTS      Max requests per minute per IP (default: 100)
  PR_EVENT_ENABLED         Enable pull_request event handling (default: false)
  PR_EVENT_ACTIONS         Comma-separated PR actions to listen for (default: opened,synchronize,reopened,ready_for_review,edited)
  WORKFLOW_FILE            Workflow file to trigger for PR events
  WORKFLOW_REPO            Sets a fixed workflow repository to trigger. Defaults to the same repository as the event.
  WORKFLOW_REF             Git ref for workflow dispatch (default: main)
  WORKFLOW_INPUTS          Static workflow inputs (key=value,key=value format)

  Plus all PR CLI environment variables (PR_TOKEN, PR_PLATFORM, etc.)
`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Server configuration flags
	serveCmd.Flags().String("listen-addr", ":8080", "Server listen address")
	serveCmd.Flags().String("webhook-path", "/webhook", "Webhook endpoint path")
	serveCmd.Flags().String("health-path", "/health", "Health check endpoint path")
	serveCmd.Flags().String("metrics-path", "/metrics", "Metrics endpoint path")

	// Security flags
	serveCmd.Flags().String("webhook-secret", "", "Webhook secret for signature validation")
	serveCmd.Flags().String("webhook-secret-file", "", "File containing webhook secret")
	serveCmd.Flags().StringSlice("allowed-repos", []string{}, "Allowed repositories (owner/repo format, supports wildcards)")
	serveCmd.Flags().Bool("require-signature", true, "Require webhook signature validation")

	// TLS flags
	serveCmd.Flags().Bool("tls-enabled", false, "Enable TLS")
	serveCmd.Flags().String("tls-cert-file", "", "TLS certificate file")
	serveCmd.Flags().String("tls-key-file", "", "TLS private key file")

	// Processing flags
	serveCmd.Flags().Bool("async-processing", true, "Process webhooks asynchronously")
	serveCmd.Flags().Int("worker-count", 10, "Number of worker goroutines")
	serveCmd.Flags().Int("queue-size", 100, "Job queue size")

	// Rate limiting flags
	serveCmd.Flags().Bool("rate-limit-enabled", true, "Enable rate limiting")
	serveCmd.Flags().Int("rate-limit-requests", 100, "Max requests per minute per IP")

	// Pull request event flags
	serveCmd.Flags().Bool("pr-event-enabled", false, "Enable pull_request event handling to trigger workflows")
	serveCmd.Flags().StringSlice("pr-event-actions", []string{"opened", "synchronize", "reopened", "ready_for_review", "edited"}, "PR actions to listen for")

	// Workflow dispatch flags
	serveCmd.Flags().String("workflow-file", "", "Workflow file to trigger (e.g., .github/workflows/pr-check.yml)")
	serveCmd.Flags().String("workflow-repo", "", "Repository to trigger workflow file. If empty will use the same as event (e.g alaudadevops/toolbox)")
	serveCmd.Flags().String("workflow-ref", "main", "Git ref to use for workflow dispatch")
	serveCmd.Flags().StringToString("workflow-inputs", nil, "Static workflow inputs (key=value)")

	// Add PR CLI flags to serve command
	prOption.AddFlags(serveCmd.Flags())
}

func runServe(cmd *cobra.Command, args []string) error {
	// Create webhook configuration
	webhookConfig := config.NewDefaultWebhookConfig()

	// Load from flags
	if addr, _ := cmd.Flags().GetString("listen-addr"); addr != "" {
		webhookConfig.ListenAddr = addr
	}
	if path, _ := cmd.Flags().GetString("webhook-path"); path != "" {
		webhookConfig.WebhookPath = path
	}
	if path, _ := cmd.Flags().GetString("health-path"); path != "" {
		webhookConfig.HealthPath = path
	}
	if path, _ := cmd.Flags().GetString("metrics-path"); path != "" {
		webhookConfig.MetricsPath = path
	}

	if secret, _ := cmd.Flags().GetString("webhook-secret"); secret != "" {
		webhookConfig.WebhookSecret = strings.TrimSpace(secret)
	}
	if secretFile, _ := cmd.Flags().GetString("webhook-secret-file"); secretFile != "" {
		data, err := os.ReadFile(secretFile)
		if err != nil {
			return fmt.Errorf("failed to read webhook secret file: %w", err)
		}
		webhookConfig.WebhookSecret = strings.TrimSpace(string(data))
	}

	if repos, _ := cmd.Flags().GetStringSlice("allowed-repos"); len(repos) > 0 {
		webhookConfig.AllowedRepos = repos
	}
	if requireSig, _ := cmd.Flags().GetBool("require-signature"); cmd.Flags().Changed("require-signature") {
		webhookConfig.RequireSignature = requireSig
	}

	if tlsEnabled, _ := cmd.Flags().GetBool("tls-enabled"); cmd.Flags().Changed("tls-enabled") {
		webhookConfig.TLSEnabled = tlsEnabled
	}
	if certFile, _ := cmd.Flags().GetString("tls-cert-file"); certFile != "" {
		webhookConfig.TLSCertFile = certFile
	}
	if keyFile, _ := cmd.Flags().GetString("tls-key-file"); keyFile != "" {
		webhookConfig.TLSKeyFile = keyFile
	}

	if async, _ := cmd.Flags().GetBool("async-processing"); cmd.Flags().Changed("async-processing") {
		webhookConfig.AsyncProcessing = async
	}
	if workers, _ := cmd.Flags().GetInt("worker-count"); cmd.Flags().Changed("worker-count") {
		webhookConfig.WorkerCount = workers
	}
	if queueSize, _ := cmd.Flags().GetInt("queue-size"); cmd.Flags().Changed("queue-size") {
		webhookConfig.QueueSize = queueSize
	}

	if rateLimitEnabled, _ := cmd.Flags().GetBool("rate-limit-enabled"); cmd.Flags().Changed("rate-limit-enabled") {
		webhookConfig.RateLimitEnabled = rateLimitEnabled
	}
	if rateLimitReqs, _ := cmd.Flags().GetInt("rate-limit-requests"); cmd.Flags().Changed("rate-limit-requests") {
		webhookConfig.RateLimitRequests = rateLimitReqs
	}

	// Pull request event configuration
	if prEventEnabled, _ := cmd.Flags().GetBool("pr-event-enabled"); cmd.Flags().Changed("pr-event-enabled") {
		webhookConfig.PREventEnabled = prEventEnabled
	}
	if prEventActions, _ := cmd.Flags().GetStringSlice("pr-event-actions"); cmd.Flags().Changed("pr-event-actions") {
		webhookConfig.PREventActions = prEventActions
	}

	// Workflow dispatch configuration
	if workflowFile, _ := cmd.Flags().GetString("workflow-file"); workflowFile != "" {
		webhookConfig.WorkflowFile = workflowFile
	}
	if workflowRef, _ := cmd.Flags().GetString("workflow-ref"); cmd.Flags().Changed("workflow-ref") {
		webhookConfig.WorkflowRef = workflowRef
	}
	if workflowRepo, _ := cmd.Flags().GetString("workflow-repo"); cmd.Flags().Changed("workflow-repo") {
		webhookConfig.WorkflowRepo = workflowRepo
	}
	if workflowInputs, _ := cmd.Flags().GetStringToString("workflow-inputs"); len(workflowInputs) > 0 {
		webhookConfig.WorkflowInputs = workflowInputs
	}

	// Load from environment variables (overrides flags)
	if err := webhookConfig.LoadFromEnv(); err != nil {
		return fmt.Errorf("failed to load webhook config from environment: %w", err)
	}

	// Initialize base PR CLI configuration
	if err := prOption.initialize(); err != nil && !isPRCommandError(err) {
		return fmt.Errorf("failed to initialize PR CLI config: %w", err)
	}
	webhookConfig.BaseConfig = prOption.Config

	// Validate configuration
	if err := webhookConfig.Validate(); err != nil {
		return fmt.Errorf("invalid webhook configuration: %w", err)
	}

	// Configure logging
	logger := logrus.New()
	if prOption.Config.Verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
	switch prOption.Config.LogFormat {
	case "console":
		logger.SetFormatter(&logrus.TextFormatter{})
	default:
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	// printing config
	logger.Debugf("Webhook config: %v", webhookConfig)
	// if data, err := json.Marshal(webhookConfig); err != nil {

	// }

	// Create and start server
	server := webhook.NewServer(webhookConfig, logger)

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Infof("Received signal: %v", sig)
		cancel()
	}()

	// Start server
	logger.Info("Starting PR CLI webhook server")
	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	logger.Info("Server stopped")
	return nil
}

// returns true if this is an error only necessary during PR commands
func isPRCommandError(err error) bool {
	switch err {
	case config.ErrMissingPlatform, config.ErrMissingToken, config.ErrMissingOwner, config.ErrMissingRepo,
		config.ErrInvalidPRNum, config.ErrMissingCommentSender, config.ErrMissingTriggerComment:
		return true
	default:
		return false
	}
}
