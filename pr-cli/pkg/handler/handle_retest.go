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
	"fmt"
	"strings"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
)

// HandleRetest reruns failed test jobs by commenting /test {pipeline-name}
func (h *PRHandler) HandleRetest(args []string) error {
	allPassed, failedChecks, err := h.getCheckRunsStatus()
	if err != nil {
		return err
	}

	if allPassed {
		return h.postAllChecksPassingMessage()
	}

	if len(args) > 0 {
		return h.retestSpecificPipelines(args)
	}

	return h.retestFailedPipelines(failedChecks)
}

// getCheckRunsStatus retrieves and validates check runs status
func (h *PRHandler) getCheckRunsStatus() (bool, []git.CheckRun, error) {
	allPassed, failedChecks, err := h.client.CheckRunsStatus()
	if err != nil {
		return false, nil, fmt.Errorf("failed to get check runs status: %w", err)
	}
	return allPassed, failedChecks, nil
}

// postAllChecksPassingMessage posts a message when all checks are passing
func (h *PRHandler) postAllChecksPassingMessage() error {
	message := "✅ All checks are passing. No failed tests to rerun."
	return h.client.PostComment(message)
}

// retestFailedPipelines processes failed checks and retests eligible pipelines
func (h *PRHandler) retestFailedPipelines(failedChecks []git.CheckRun) error {
	retestablePipelines, skippedPipelines := h.filterRetestablePipelines(failedChecks)

	if len(retestablePipelines) == 0 {
		return h.postNoRetestablePipelinesMessage(skippedPipelines)
	}

	return h.retestPipelines(retestablePipelines)
}

// filterRetestablePipelines filters failed checks to find pipelines that can be retested
func (h *PRHandler) filterRetestablePipelines(failedChecks []git.CheckRun) ([]string, []string) {
	var retestablePipelines []string
	var skippedPipelines []string

	for _, check := range failedChecks {
		if h.shouldSkipSelfCheck(check.Name) {
			continue
		}

		if h.isRetestableCheck(check) {
			pipelineName := extractPipelineName(check.Name)
			if pipelineName != "" {
				retestablePipelines = append(retestablePipelines, pipelineName)
			} else {
				skippedPipelines = append(skippedPipelines, check.Name)
			}
		}
	}

	return retestablePipelines, skippedPipelines
}

// shouldSkipSelfCheck determines if a check should be skipped as it's our own pipeline
func (h *PRHandler) shouldSkipSelfCheck(checkName string) bool {
	return h.config.SelfCheckName != "" &&
		strings.HasSuffix(strings.TrimSpace(checkName), "/ "+h.config.SelfCheckName)
}

// isRetestableCheck determines if a check run can be retested
func (h *PRHandler) isRetestableCheck(check git.CheckRun) bool {
	return check.Status == "completed" &&
		(check.Conclusion == "failure" || check.Conclusion == "timed_out" || check.Conclusion == "cancelled")
}

// postNoRetestablePipelinesMessage posts a message when no pipelines can be retested
func (h *PRHandler) postNoRetestablePipelinesMessage(skippedPipelines []string) error {
	message := "✅ No failed pipelines found that can be retested."
	if len(skippedPipelines) > 0 {
		message += fmt.Sprintf("\n\nSkipped checks (cannot extract pipeline name):\n• %s",
			strings.Join(skippedPipelines, "\n• "))
	}
	return h.client.PostComment(message)
}

// retestSpecificPipelines retests only the pipelines specified in args
func (h *PRHandler) retestSpecificPipelines(pipelineNames []string) error {
	return h.retestPipelines(pipelineNames)
}

// retestPipelines posts /test comments for the specified pipelines
func (h *PRHandler) retestPipelines(pipelineNames []string) error {
	var comments []string

	for _, pipeline := range pipelineNames {
		comment := fmt.Sprintf("/test %s", pipeline)
		comments = append(comments, comment)
	}

	// Post individual comments for each pipeline to trigger retests
	for _, comment := range comments {
		if err := h.client.PostComment(comment); err != nil {
			h.Errorf("Failed to post retest comment '%s': %v", comment, err)
			return fmt.Errorf("failed to trigger retest for pipeline: %w", err)
		}
		h.Infof("Posted retest comment: %s", comment)
	}

	// Post a summary comment
	message := fmt.Sprintf("🔄 **Retesting failed pipelines**\n\nTriggered retests for:\n• %s",
		strings.Join(pipelineNames, "\n• "))

	return h.client.PostComment(message)
}

// extractPipelineName extracts the pipeline name from a check run name
// This function handles common patterns in PAC (Pipelines as Code) check names
func extractPipelineName(checkName string) string {
	// Common patterns for PAC check names:
	// "Pipelines as Code CI / pipeline-name" -> pipeline-name
	// "pipeline-name / task-name" -> pipeline-name
	// "pipeline-name" -> pipeline-name

	name := strings.TrimSpace(checkName)

	// Filter out some common non-pipeline check names first
	lowerName := strings.ToLower(name)
	nonPipelineChecks := []string{
		"merge conflict",
		"codecov",
		"sonarcloud",
		"license",
		"cla",
		"semantic",
		"dependabot",
		"gitguardian",
		"security",
	}

	for _, nonPipeline := range nonPipelineChecks {
		if strings.Contains(lowerName, nonPipeline) {
			return "" // Skip non-pipeline checks
		}
	}

	// Handle patterns with " / " separator
	parts := strings.Split(name, " / ")
	if len(parts) >= 2 {
		// For PAC pattern: "Pipelines as Code CI / pipeline-name"
		// Return the last part as the pipeline name
		lastPart := strings.TrimSpace(parts[len(parts)-1])

		// If the last part looks like a task name, return the second-to-last part
		// Common task patterns: build, test, deploy, lint, etc.
		commonTasks := []string{"build", "test", "deploy", "lint", "check", "scan", "analyze"}
		for _, task := range commonTasks {
			if strings.EqualFold(lastPart, task) && len(parts) >= 3 {
				return strings.TrimSpace(parts[len(parts)-2])
			}
		}

		return lastPart
	}

	// Return the full name if it looks like a pipeline
	return name
}
