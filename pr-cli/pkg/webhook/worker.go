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
	"time"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
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

	// Parse and execute command
	command, args := parseSimpleCommand(cfg.TriggerComment)
	if command == "" {
		logger.Errorf("Failed to parse command from: %s", cfg.TriggerComment)
		CommandExecutionTotal.WithLabelValues(job.Event.Platform, "unknown", "error").Inc()
		return
	}

	// Execute command
	if err := prHandler.ExecuteCommand(command, args); err != nil {
		logger.Errorf("Failed to execute command: %v", err)
		CommandExecutionTotal.WithLabelValues(job.Event.Platform, command, "error").Inc()
		WebhookProcessingDuration.WithLabelValues(job.Event.Platform, command).Observe(time.Since(startTime).Seconds())
		return
	}

	logger.Info("Successfully processed webhook job")
	CommandExecutionTotal.WithLabelValues(job.Event.Platform, command, "success").Inc()
	WebhookProcessingDuration.WithLabelValues(job.Event.Platform, command).Observe(time.Since(startTime).Seconds())
}
