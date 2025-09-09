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

package handler

import (
	"os"
	"path/filepath"
)

// writeTektonResult writes a result value for Tekton pipeline
func (h *PRHandler) writeTektonResult(name, value string) {
	// Check if the results directory exists
	if _, err := os.Stat(h.config.ResultsDir); os.IsNotExist(err) {
		h.Debugf("Results directory %s does not exist, skipping result writing", h.config.ResultsDir)
		return
	}

	resultPath := filepath.Join(h.config.ResultsDir, name)
	if err := os.WriteFile(resultPath, []byte(value), 0644); err != nil {
		h.Errorf("Failed to write result %s to %s: %v", name, resultPath, err)
		return
	}

	h.Infof("Wrote result %s=%s to %s", name, value, resultPath)
}
