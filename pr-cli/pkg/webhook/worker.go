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
	"strings"
	"time"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/executor"
	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
	"github.com/sirupsen/logrus"
)

// WebhookJob represents a webhook processing job
type WebhookJob struct {
	Event     *WebhookEvent
	Timestamp time.Time
}

// Worker processes webhook jobs from a queue
type Worker struct {
	id         int
	jobQueue   <-chan *WebhookJob
	logger     *logrus.Logger
	baseConfig *config.Config
}

// newWorker creates a new worker
func newWorker(id int, jobQueue <-chan *WebhookJob, logger *logrus.Logger, baseConfig *config.Config) *Worker {
	return &Worker{
		id:         id,
		jobQueue:   jobQueue,
		logger:     logger,
		baseConfig: baseConfig,
	}
}

// start begins processing jobs
func (w *Worker) start(ctx context.Context) {
	w.logger.Infof("Worker %d started", w.id)
	ActiveWorkers.Inc()
	defer ActiveWorkers.Dec()

	for {
		select {
		case <-ctx.Done():
			w.logger.Infof("Worker %d stopping", w.id)
			return
		case job, ok := <-w.jobQueue:
			if !ok {
				w.logger.Infof("Worker %d: job queue closed", w.id)
				return
			}
			w.processJob(ctx, job)
		}
	}
}

// processJob processes a single webhook job
func (w *Worker) processJob(ctx context.Context, job *WebhookJob) {
	startTime := time.Now()

	logger := w.logger.WithFields(logrus.Fields{
		"worker":   w.id,
		"platform": job.Event.Platform,
		"repo":     fmt.Sprintf("%s/%s", job.Event.Repository.Owner, job.Event.Repository.Name),
		"pr":       job.Event.PullRequest.Number,
		"command":  job.Event.Comment.Body,
		"sender":   job.Event.Sender.Login,
	})

	logger.Info("Processing webhook job")

	// Convert webhook event to config
	cfg := job.Event.ToConfig(w.baseConfig)

	// Create PR handler
	prHandler, err := handler.NewPRHandler(logger.Logger, cfg)
	if err != nil {
		logger.Errorf("Failed to create PR handler: %v", err)
		CommandExecutionTotal.WithLabelValues(job.Event.Platform, "unknown", "error").Inc()
		return
	}

	// Parse command using shared executor
	parsedCmd, err := executor.ParseCommand(cfg.TriggerComment)
	if err != nil {
		logger.Errorf("Failed to parse command from %q: %v", cfg.TriggerComment, err)
		CommandExecutionTotal.WithLabelValues(job.Event.Platform, "unknown", "error").Inc()
		return
	}

	// Execute based on command type
	switch parsedCmd.Type {
	case executor.SingleCommand, executor.BuiltInCommand:
		w.executeSingleCommand(logger, prHandler, job.Event.Platform, parsedCmd, startTime)
	case executor.MultiCommand:
		w.executeMultiCommand(logger, prHandler, job.Event.Platform, parsedCmd, startTime)
	default:
		logger.Errorf("Unknown command type: %s", parsedCmd.Type)
		CommandExecutionTotal.WithLabelValues(job.Event.Platform, "unknown", "error").Inc()
	}
}

// executeSingleCommand executes a single command
func (w *Worker) executeSingleCommand(logger *logrus.Entry, prHandler *handler.PRHandler, platform string, parsedCmd *executor.ParsedCommand, startTime time.Time) {
	command := parsedCmd.Command
	args := parsedCmd.Args

	// Execute command
	if err := prHandler.ExecuteCommand(command, args); err != nil {
		logger.Errorf("Failed to execute command: %v", err)
		CommandExecutionTotal.WithLabelValues(platform, command, "error").Inc()
		WebhookProcessingDuration.WithLabelValues(platform, command).Observe(time.Since(startTime).Seconds())
		return
	}

	logger.Info("Successfully processed webhook job")
	CommandExecutionTotal.WithLabelValues(platform, command, "success").Inc()
	WebhookProcessingDuration.WithLabelValues(platform, command).Observe(time.Since(startTime).Seconds())
}

// executeMultiCommand executes multiple commands
func (w *Worker) executeMultiCommand(logger *logrus.Entry, prHandler *handler.PRHandler, platform string, parsedCmd *executor.ParsedCommand, startTime time.Time) {
	logger.Infof("Executing multi-command with %d commands", len(parsedCmd.CommandLines))

	// Parse command lines into sub-commands
	subCommands, err := executor.ParseMultiCommandLines(parsedCmd.CommandLines)
	if err != nil {
		logger.Errorf("Failed to parse multi-command lines: %v", err)
		CommandExecutionTotal.WithLabelValues(platform, "multi", "error").Inc()
		return
	}

	// Execute each sub-command and collect results
	var results []string
	var hasErrors bool

	for _, subCmd := range subCommands {
		result := w.processSubCommand(logger, prHandler, platform, subCmd)
		results = append(results, result)

		// Check if this command failed
		if strings.HasPrefix(result, "❌") {
			hasErrors = true
		}
	}

	// Post summary comment
	header := "**Multi-Command Execution Results:**"
	if hasErrors {
		header = fmt.Sprintf("%s (⚠️ Some commands failed)", header)
	}

	summary := fmt.Sprintf("%s\n\n%s", header, strings.Join(results, "\n"))
	if err := prHandler.PostComment(summary); err != nil {
		logger.Errorf("Failed to post multi-command summary: %v", err)
	}

	// Record metrics
	status := "success"
	if hasErrors {
		status = "partial_error"
	}
	CommandExecutionTotal.WithLabelValues(platform, "multi", status).Inc()
	WebhookProcessingDuration.WithLabelValues(platform, "multi").Observe(time.Since(startTime).Seconds())

	logger.Info("Successfully processed multi-command webhook job")
}

// processSubCommand executes a single sub-command in multi-command context
func (w *Worker) processSubCommand(logger *logrus.Entry, prHandler *handler.PRHandler, platform string, subCmd executor.SubCommand) string {
	cmdDisplay := executor.GetCommandDisplayName(subCmd)

	// Execute the command
	if err := prHandler.ExecuteCommand(subCmd.Command, subCmd.Args); err != nil {
		logger.Errorf("Multi-command '%s' failed: %v", subCmd.Command, err)
		CommandExecutionTotal.WithLabelValues(platform, subCmd.Command, "error").Inc()
		return fmt.Sprintf("❌ Command `%s` failed: %v", cmdDisplay, err)
	}

	CommandExecutionTotal.WithLabelValues(platform, subCmd.Command, "success").Inc()
	return fmt.Sprintf("✅ Command `%s` executed successfully", cmdDisplay)
}
