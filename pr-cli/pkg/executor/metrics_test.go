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

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNoOpMetricsRecorder(t *testing.T) {
	recorder := &NoOpMetricsRecorder{}

	// These should not panic
	assert.NotPanics(t, func() {
		recorder.RecordCommandExecution("github", "merge", "success")
	})

	assert.NotPanics(t, func() {
		recorder.RecordProcessingDuration("github", "merge", 100*time.Millisecond)
	})
}

func TestMetricsRecorderInterface(t *testing.T) {
	var recorder MetricsRecorder = &NoOpMetricsRecorder{}

	assert.NotNil(t, recorder)

	// Test that it implements the interface correctly
	recorder.RecordCommandExecution("gitlab", "lgtm", "error")
	recorder.RecordProcessingDuration("gitlab", "lgtm", 50*time.Millisecond)

	// No assertions needed - just ensuring the interface is satisfied
}
