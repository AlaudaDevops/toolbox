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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// WebhookRequestsTotal counts total webhook requests received
	WebhookRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pr_cli_webhook_requests_total",
			Help: "Total number of webhook requests received",
		},
		[]string{"platform", "event_type", "status"},
	)

	// WebhookProcessingDuration tracks webhook processing duration
	WebhookProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pr_cli_webhook_processing_duration_seconds",
			Help:    "Webhook processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"platform", "command"},
	)

	// CommandExecutionTotal counts total commands executed
	CommandExecutionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pr_cli_command_execution_total",
			Help: "Total number of commands executed",
		},
		[]string{"platform", "command", "status"},
	)

	// QueueSize tracks current job queue size
	QueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pr_cli_queue_size",
			Help: "Current size of the webhook job queue",
		},
	)

	// ActiveWorkers tracks number of active workers
	ActiveWorkers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pr_cli_active_workers",
			Help: "Number of active worker goroutines",
		},
	)

	// PREventProcessingTotal counts pull_request events processed
	PREventProcessingTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pr_cli_pr_event_total",
			Help: "Total number of pull_request events processed",
		},
		[]string{"platform", "action", "status"},
	)

	// WorkflowDispatchTotal counts workflow dispatch triggers
	WorkflowDispatchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pr_cli_workflow_dispatch_total",
			Help: "Total number of workflow dispatch triggers",
		},
		[]string{"platform", "workflow", "status"},
	)
)
