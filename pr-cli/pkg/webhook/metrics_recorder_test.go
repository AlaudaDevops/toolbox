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
	"testing"
	"time"
)

func TestNewWebhookMetricsRecorder(t *testing.T) {
	platform := "github"
	recorder := NewWebhookMetricsRecorder(platform)

	if recorder == nil {
		t.Fatal("Expected non-nil recorder")
	}

	if recorder.platform != platform {
		t.Errorf("Expected platform %s, got %s", platform, recorder.platform)
	}
}

func TestWebhookMetricsRecorder_RecordCommandExecution(t *testing.T) {
	recorder := NewWebhookMetricsRecorder("github")

	// Test recording success
	recorder.RecordCommandExecution("github", "test", "success")

	// Test recording error
	recorder.RecordCommandExecution("github", "test", "error")

	// Test recording partial_error
	recorder.RecordCommandExecution("github", "multi", "partial_error")
}

func TestWebhookMetricsRecorder_RecordProcessingDuration(t *testing.T) {
	recorder := NewWebhookMetricsRecorder("github")

	// Test recording duration
	duration := 100 * time.Millisecond
	recorder.RecordProcessingDuration("github", "test", duration)

	// Test recording another duration
	duration = 500 * time.Millisecond
	recorder.RecordProcessingDuration("github", "merge", duration)
}
