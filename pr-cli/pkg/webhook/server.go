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
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/executor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Server represents the webhook HTTP server
type Server struct {
	config    *config.WebhookConfig
	logger    *logrus.Logger
	server    *http.Server
	jobQueue  chan *WebhookJob
	workers   []*Worker
	startTime time.Time
}

// NewServer creates a new webhook server
func NewServer(cfg *config.WebhookConfig, logger *logrus.Logger) *Server {
	return &Server{
		config:    cfg,
		logger:    logger,
		jobQueue:  make(chan *WebhookJob, cfg.QueueSize),
		startTime: time.Now(),
	}
}

// Start starts the webhook server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Infof("Starting webhook server on %s", s.config.ListenAddr)
	s.logger.Infof("Configuration: %s", s.config.DebugString())

	// Start workers if async processing is enabled
	if s.config.AsyncProcessing {
		s.startWorkers(ctx)
	}

	// Create HTTP router
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc(s.config.WebhookPath, s.handleWebhook)
	mux.HandleFunc(s.config.HealthPath, s.handleHealth)
	mux.HandleFunc(s.config.HealthPath+"/ready", s.handleReadiness)
	mux.Handle(s.config.MetricsPath, promhttp.Handler())

	// Apply middleware
	handler := securityHeadersMiddleware(mux)
	handler = recoveryMiddleware(s.logger)(handler)
	handler = loggingMiddleware(s.logger)(handler)
	handler = rateLimitMiddleware(s.config.RateLimitEnabled, s.config.RateLimitRequests, s.logger)(handler)

	// Create HTTP server
	s.server = &http.Server{
		Addr:         s.config.ListenAddr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server
	errChan := make(chan error, 1)
	go func() {
		if s.config.TLSEnabled {
			s.logger.Infof("Starting HTTPS server with TLS")
			errChan <- s.server.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
		} else {
			s.logger.Infof("Starting HTTP server (TLS disabled)")
			errChan <- s.server.ListenAndServe()
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.logger.Info("Context cancelled, shutting down server")
		return s.Shutdown(context.Background())
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down webhook server")

	// Stop accepting new requests
	if s.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			s.logger.Errorf("Server shutdown error: %v", err)
			return err
		}
	}

	// Close job queue
	if s.jobQueue != nil {
		close(s.jobQueue)
	}

	// Wait for workers to finish (they will exit when queue is closed)
	// In production, you might want to add a timeout here
	time.Sleep(2 * time.Second)

	s.logger.Info("Server shutdown complete")
	return nil
}

// startWorkers starts the worker pool
func (s *Server) startWorkers(ctx context.Context) {
	s.workers = make([]*Worker, s.config.WorkerCount)

	for i := 0; i < s.config.WorkerCount; i++ {
		worker := newWorker(i, s.jobQueue, s.logger, s.config.BaseConfig)
		s.workers[i] = worker
		go worker.start(ctx)
	}

	s.logger.Infof("Started %d webhook workers", s.config.WorkerCount)
}

// processWebhookSync processes a webhook synchronously
func (s *Server) processWebhookSync(event *WebhookEvent) error {
	// Convert webhook event to config
	cfg := event.ToConfig(s.config.BaseConfig)

	// Create PR handler
	prHandler, err := handler.NewPRHandler(s.logger, cfg)
	if err != nil {
		CommandExecutionTotal.WithLabelValues(event.Platform, "unknown", "error").Inc()
		return fmt.Errorf("failed to create PR handler: %w", err)
	}

	// Parse command using shared executor
	parsedCmd, err := executor.ParseCommand(cfg.TriggerComment)
	if err != nil {
		s.logger.Errorf("Failed to parse command from %q: %v", cfg.TriggerComment, err)
		CommandExecutionTotal.WithLabelValues(event.Platform, "unknown", "error").Inc()
		return fmt.Errorf("failed to parse command: %w", err)
	}

	// Use unified executor - now supports multi-command!
	return s.executeWithUnifiedExecutor(prHandler, event.Platform, cfg, parsedCmd)
}

// executeWithUnifiedExecutor executes commands using the unified executor for sync mode
func (s *Server) executeWithUnifiedExecutor(prHandler *handler.PRHandler, platform string, cfg *config.Config, parsedCmd *executor.ParsedCommand) error {
	// Create execution config for webhook mode
	execConfig := executor.NewWebhookExecutionConfig()

	// Create execution context
	execContext := &executor.ExecutionContext{
		PRHandler:       prHandler,
		Logger:          s.logger,
		Config:          execConfig,
		MetricsRecorder: NewWebhookMetricsRecorder(),
		Platform:        platform,
		CommentSender:   cfg.CommentSender,
		TriggerComment:  cfg.TriggerComment,
	}

	// Create and execute with unified executor
	cmdExecutor := executor.NewCommandExecutor(execContext)
	result, err := cmdExecutor.Execute(parsedCmd)

	if err != nil {
		s.logger.Errorf("Command execution failed: %v", err)
		return err
	}

	if !result.Success {
		return fmt.Errorf("command execution completed with errors")
	}

	s.logger.Info("Successfully processed webhook synchronously")
	return nil
}

// parseSimpleCommand parses a command from a comment body
// Returns command name and arguments
func parseSimpleCommand(body string) (string, []string) {
	body = trimLeadingWhitespace(body)
	if len(body) == 0 || body[0] != '/' {
		return "", nil
	}

	// Remove leading "/"
	body = body[1:]

	// Split into words
	parts := strings.Fields(body)
	if len(parts) == 0 {
		return "", nil
	}

	command := parts[0]
	args := []string{}
	if len(parts) > 1 {
		args = parts[1:]
	}

	return command, args
}

// extractCommand extracts the command name from a comment body
func extractCommand(body string) string {
	command, _ := parseSimpleCommand(body)
	if command == "" {
		return "unknown"
	}
	return command
}

// trimLeadingWhitespace removes leading whitespace from a string
func trimLeadingWhitespace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	return s[start:]
}
