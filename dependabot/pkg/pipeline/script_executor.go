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

// Package pipeline provides a comprehensive pipeline for dependency updates and PR creation
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/sirupsen/logrus"
)

// ScriptExecutor handles execution of custom scripts
type ScriptExecutor struct {
	// projectPath is the base project path
	projectPath string
}

// NewScriptExecutor creates a new script executor
func NewScriptExecutor(projectPath string) *ScriptExecutor {
	return &ScriptExecutor{
		projectPath: projectPath,
	}
}

// ExecuteScript executes a custom script with the given configuration
func (e *ScriptExecutor) ExecuteScript(stage string, scriptConfig *config.ScriptConfig) error {
	if scriptConfig == nil || scriptConfig.Script == "" {
		return nil
	}

	// Create temporary script file
	tempScriptFile, err := e.createTempScriptFile(stage, scriptConfig.Script)
	if err != nil {
		return fmt.Errorf("failed to create temporary script file for %s: %w", stage, err)
	}
	defer e.cleanupTempFile(tempScriptFile)

	logrus.Infof("Executing %s script using bash from temporary file: %s", stage, tempScriptFile)

	// Set up context - use background context for infinite wait if no timeout specified
	ctx := context.Background()
	var cancel context.CancelFunc

	if scriptConfig.Timeout != "" {
		timeout, err := time.ParseDuration(scriptConfig.Timeout)
		if err != nil {
			logrus.Warnf("Invalid timeout format '%s' for %s script, continuing without timeout: %v", scriptConfig.Timeout, stage, err)
		} else if timeout > 0 {
			// Only set timeout if it's greater than 0
			ctx, cancel = context.WithTimeout(context.Background(), timeout)
			defer cancel()
		}
	}

	// Create command with context (works for both timeout and infinite wait cases)
	cmd := exec.CommandContext(ctx, "bash", tempScriptFile)
	cmd.Dir = e.projectPath

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	// Log script output
	if len(output) > 0 {
		logrus.Debugf("%s script output:\n%s", stage, string(output))
	}

	if err != nil {
		errorMsg := fmt.Sprintf("%s script failed: %v, output:\n%s", stage, err, string(output))

		if scriptConfig.ContinueOnError {
			logrus.Warnf("Warning: %s", errorMsg)
			return nil
		}

		return errors.New(errorMsg)
	}

	logrus.Infof("✅ %s script completed successfully", stage)
	return nil
}

// createTempScriptFile creates a temporary script file with the given content
func (e *ScriptExecutor) createTempScriptFile(stage, scriptContent string) (string, error) {
	// Create temporary file with .sh extension
	tempFile, err := os.CreateTemp("/tmp", fmt.Sprintf("dependabot_%s_*.sh", strings.ToLower(stage)))
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}

	// Write script content to temporary file
	if _, err := tempFile.WriteString(scriptContent); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write script content to temporary file: %w", err)
	}

	// Close the file to ensure content is written
	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Make the script executable
	if err := os.Chmod(tempFile.Name(), 0755); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to make script executable: %w", err)
	}

	return tempFile.Name(), nil
}

// cleanupTempFile removes the temporary script file
func (e *ScriptExecutor) cleanupTempFile(tempFilePath string) {
	if err := os.Remove(tempFilePath); err != nil {
		logrus.Debugf("Failed to cleanup temporary file %s: %v", tempFilePath, err)
	}
}
