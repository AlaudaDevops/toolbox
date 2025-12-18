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

import "time"

// WebhookMetricsRecorder implements the MetricsRecorder interface for webhook execution.
type WebhookMetricsRecorder struct {
	platform string
}

// NewWebhookMetricsRecorder creates a new WebhookMetricsRecorder.
func NewWebhookMetricsRecorder(platform string) *WebhookMetricsRecorder {
	return &WebhookMetricsRecorder{
		platform: platform,
	}
}

// RecordCommandExecution records a command execution metric.
func (r *WebhookMetricsRecorder) RecordCommandExecution(platform, command, status string) {
	CommandExecutionTotal.WithLabelValues(platform, command, status).Inc()
}

// RecordProcessingDuration records the processing duration for a command.
func (r *WebhookMetricsRecorder) RecordProcessingDuration(platform, command string, duration time.Duration) {
	WebhookProcessingDuration.WithLabelValues(platform, command).Observe(duration.Seconds())
}
