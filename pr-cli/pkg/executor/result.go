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

// ExecutionResult represents the result of a command execution
type ExecutionResult struct {
	// Success indicates if the execution was successful
	Success bool
	
	// Error contains the error if execution failed
	Error error
	
	// CommandType indicates the type of command that was executed
	CommandType CommandType
	
	// Results contains the results of sub-commands for multi-command execution
	Results []SubCommandResult
}

// SubCommandResult represents the result of a single sub-command in multi-command execution
type SubCommandResult struct {
	// Command is the command name
	Command string
	
	// Args are the command arguments
	Args []string
	
	// Success indicates if the sub-command was successful
	Success bool
	
	// Error contains the error if the sub-command failed
	Error error
}

// NewSuccessResult creates a new successful execution result
func NewSuccessResult(cmdType CommandType) *ExecutionResult {
	return &ExecutionResult{
		Success:     true,
		Error:       nil,
		CommandType: cmdType,
		Results:     nil,
	}
}

// NewErrorResult creates a new failed execution result
func NewErrorResult(cmdType CommandType, err error) *ExecutionResult {
	return &ExecutionResult{
		Success:     false,
		Error:       err,
		CommandType: cmdType,
		Results:     nil,
	}
}

// NewMultiCommandResult creates a new multi-command execution result
func NewMultiCommandResult(results []SubCommandResult) *ExecutionResult {
	// Determine overall success based on sub-command results
	success := true
	for _, result := range results {
		if !result.Success {
			success = false
			break
		}
	}
	
	return &ExecutionResult{
		Success:     success,
		Error:       nil,
		CommandType: MultiCommand,
		Results:     results,
	}
}
