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

// ExecutionResult represents the result of command execution
type ExecutionResult struct {
	Success     bool
	Error       error
	CommandType CommandType
	Results     []SubCommandResult // For multi-command
}

// SubCommandResult represents result of a single sub-command
type SubCommandResult struct {
	Command string
	Args    []string
	Success bool
	Error   error
}

// NewSuccessResult creates a successful execution result
func NewSuccessResult(cmdType CommandType) *ExecutionResult {
	return &ExecutionResult{
		Success:     true,
		CommandType: cmdType,
	}
}

// NewErrorResult creates a failed execution result
func NewErrorResult(cmdType CommandType, err error) *ExecutionResult {
	return &ExecutionResult{
		Success:     false,
		Error:       err,
		CommandType: cmdType,
	}
}

// NewMultiCommandResult creates a multi-command result
func NewMultiCommandResult(results []SubCommandResult) *ExecutionResult {
	success := true
	for _, r := range results {
		if !r.Success {
			success = false
			break
		}
	}

	return &ExecutionResult{
		Success:     success,
		CommandType: MultiCommand,
		Results:     results,
	}
}
