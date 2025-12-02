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

package executor

import "time"

// MetricsRecorder is an interface for recording execution metrics
type MetricsRecorder interface {
	RecordCommandExecution(platform, command, status string)
	RecordProcessingDuration(platform, command string, duration time.Duration)
}

// NoOpMetricsRecorder is a no-op implementation for CLI mode
type NoOpMetricsRecorder struct{}

// RecordCommandExecution is a no-op implementation
func (n *NoOpMetricsRecorder) RecordCommandExecution(platform, command, status string) {}

// RecordProcessingDuration is a no-op implementation
func (n *NoOpMetricsRecorder) RecordProcessingDuration(platform, command string, duration time.Duration) {
}
