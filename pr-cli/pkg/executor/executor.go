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
	"fmt"
	"time"
)

// CommandExecutor orchestrates command execution with validation and result handling
type CommandExecutor struct {
	context       *ExecutionContext
	validator     *Validator
	resultHandler *ResultHandler
}

// NewCommandExecutor creates a new command executor instance
func NewCommandExecutor(ctx *ExecutionContext) *CommandExecutor {
	return &CommandExecutor{
		context:       ctx,
		validator:     NewValidator(ctx),
		resultHandler: NewResultHandler(ctx),
	}
}

// Execute executes a parsed command and returns the result
func (e *CommandExecutor) Execute(parsedCmd *ParsedCommand) (*ExecutionResult, error) {
	switch parsedCmd.Type {
	case SingleCommand:
		return e.ExecuteSingleCommand(parsedCmd.Command, parsedCmd.Args)
	case MultiCommand:
		return e.ExecuteMultiCommand(parsedCmd.CommandLines, parsedCmd.RawCommandLines)
	case BuiltInCommand:
		return e.ExecuteBuiltInCommand(parsedCmd.Command, parsedCmd.Args)
	default:
		return nil, fmt.Errorf(ErrUnsupportedCommandType, parsedCmd.Type)
	}
}

// ExecuteSingleCommand executes a single command
func (e *CommandExecutor) ExecuteSingleCommand(command string, args []string) (*ExecutionResult, error) {
	startTime := time.Now()
	
	e.context.Logger.Infof("Executing single command: %s with args: %v", command, args)

	// Validate command
	if err := e.validator.ValidateSingleCommand(command); err != nil {
		e.recordMetrics(command, "validation_failed", time.Since(startTime))
		return NewErrorResult(SingleCommand, err), err
	}

	// Execute command
	if err := e.context.PRHandler.ExecuteCommand(command, args); err != nil {
		e.recordMetrics(command, "failure", time.Since(startTime))
		
		// Handle error through result handler
		if handledErr := e.resultHandler.HandleSingleCommandError(command, err); handledErr != nil {
			return NewErrorResult(SingleCommand, handledErr), handledErr
		}
		
		// Error was handled (posted as comment), return success result
		return NewSuccessResult(SingleCommand), nil
	}

	e.recordMetrics(command, "success", time.Since(startTime))
	return NewSuccessResult(SingleCommand), nil
}

// ExecuteMultiCommand executes multiple commands
func (e *CommandExecutor) ExecuteMultiCommand(commandLines, rawCommandLines []string) (*ExecutionResult, error) {
	startTime := time.Now()
	
	e.context.Logger.Infof("Executing multi-command with %d commands", len(commandLines))

	// Parse command lines into sub-commands
	subCommands, err := ParseMultiCommandLines(commandLines)
	if err != nil {
		e.recordMetrics("multi-command", "parse_failed", time.Since(startTime))
		return NewErrorResult(MultiCommand, err), err
	}

	// Validate multi-command execution
	if err := e.validator.ValidateMultiCommand(subCommands, rawCommandLines); err != nil {
		e.recordMetrics("multi-command", "validation_failed", time.Since(startTime))
		return NewErrorResult(MultiCommand, err), err
	}

	// Execute each sub-command and collect results
	var results []SubCommandResult
	for _, subCmd := range subCommands {
		subStartTime := time.Now()
		result := e.executeSubCommand(subCmd)
		results = append(results, result)
		
		// Record metrics for each sub-command
		status := "success"
		if !result.Success {
			status = "failure"
		}
		e.recordMetrics(subCmd.Command, status, time.Since(subStartTime))

		// Stop on first error if configured
		if !result.Success && e.context.Config.StopOnFirstError {
			e.context.Logger.Infof("Stopping multi-command execution on first error")
			break
		}
	}

	// Handle results (post summary comment)
	if err := e.resultHandler.HandleMultiCommandResults(results); err != nil {
		e.context.Logger.Errorf("Failed to post multi-command results: %v", err)
	}

	e.recordMetrics("multi-command", "completed", time.Since(startTime))
	return NewMultiCommandResult(results), nil
}

// ExecuteBuiltInCommand executes a built-in command
func (e *CommandExecutor) ExecuteBuiltInCommand(command string, args []string) (*ExecutionResult, error) {
	startTime := time.Now()
	
	e.context.Logger.Infof("Executing built-in command: %s", command)

	// Execute built-in command (no validation needed)
	if err := e.context.PRHandler.ExecuteCommand(command, args); err != nil {
		e.recordMetrics(command, "failure", time.Since(startTime))
		e.context.Logger.Errorf("Built-in command %s failed: %v", command, err)
		return NewErrorResult(BuiltInCommand, err), err
	}

	e.recordMetrics(command, "success", time.Since(startTime))
	return NewSuccessResult(BuiltInCommand), nil
}

// executeSubCommand executes a single sub-command in multi-command context
func (e *CommandExecutor) executeSubCommand(subCmd SubCommand) SubCommandResult {
	e.context.Logger.Infof("Executing sub-command: %s with args: %v", subCmd.Command, subCmd.Args)

	if err := e.context.PRHandler.ExecuteCommand(subCmd.Command, subCmd.Args); err != nil {
		e.context.Logger.Errorf("Sub-command '%s' failed: %v", subCmd.Command, err)
		return SubCommandResult{
			Command: subCmd.Command,
			Args:    subCmd.Args,
			Success: false,
			Error:   err,
		}
	}

	return SubCommandResult{
		Command: subCmd.Command,
		Args:    subCmd.Args,
		Success: true,
		Error:   nil,
	}
}

// recordMetrics records execution metrics if a metrics recorder is configured
func (e *CommandExecutor) recordMetrics(command, status string, duration time.Duration) {
	if e.context.MetricsRecorder != nil {
		e.context.MetricsRecorder.RecordCommandExecution(e.context.Platform, command, status)
		e.context.MetricsRecorder.RecordProcessingDuration(e.context.Platform, command, duration)
	}
}
