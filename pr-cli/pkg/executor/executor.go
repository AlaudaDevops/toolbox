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

// CommandExecutor orchestrates command execution with unified validation and error handling
type CommandExecutor struct {
	context       *ExecutionContext
	validator     *Validator
	resultHandler *ResultHandler
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(ctx *ExecutionContext) *CommandExecutor {
	return &CommandExecutor{
		context:       ctx,
		validator:     NewValidator(ctx),
		resultHandler: NewResultHandler(ctx),
	}
}

// Execute is the main entry point for command execution
func (e *CommandExecutor) Execute(parsedCmd *ParsedCommand) (*ExecutionResult, error) {
	switch parsedCmd.Type {
	case SingleCommand:
		return e.ExecuteSingleCommand(parsedCmd.Command, parsedCmd.Args)
	case BuiltInCommand:
		return e.ExecuteBuiltInCommand(parsedCmd.Command, parsedCmd.Args)
	case MultiCommand:
		return e.ExecuteMultiCommand(parsedCmd.CommandLines, parsedCmd.RawCommandLines)
	default:
		return nil, fmt.Errorf(ErrUnknownCommandType, parsedCmd.Type)
	}
}

// ExecuteSingleCommand executes a single regular command
func (e *CommandExecutor) ExecuteSingleCommand(command string, args []string) (*ExecutionResult, error) {
	startTime := time.Now()
	e.context.Logger.Infof("Executing single command: %s", command)

	// Validate
	if err := e.validator.ValidateSingleCommand(command); err != nil {
		e.recordMetrics(command, "error", startTime)
		return NewErrorResult(SingleCommand, err), err
	}

	// Execute
	if err := e.context.PRHandler.ExecuteCommand(command, args); err != nil {
		e.recordMetrics(command, "error", startTime)

		// Handle error based on config
		if handledErr := e.resultHandler.HandleSingleCommandError(command, err); handledErr != nil {
			return NewErrorResult(SingleCommand, handledErr), handledErr
		}

		// Error was handled (posted as comment), return success to avoid terminal error
		if e.context.Config.PostErrorsAsPRComments {
			return NewSuccessResult(SingleCommand), nil
		}

		return NewErrorResult(SingleCommand, err), err
	}

	e.recordMetrics(command, "success", startTime)
	return NewSuccessResult(SingleCommand), nil
}

// ExecuteBuiltInCommand executes a built-in system command
func (e *CommandExecutor) ExecuteBuiltInCommand(command string, args []string) (*ExecutionResult, error) {
	startTime := time.Now()
	e.context.Logger.Infof("Executing built-in command: %s", command)

	// Built-in commands skip validation
	if err := e.context.PRHandler.ExecuteCommand(command, args); err != nil {
		e.recordMetrics(command, "error", startTime)
		e.context.Logger.Errorf("Built-in command %s failed: %v", command, err)

		// For built-in commands, return errors directly (don't post to PR)
		return NewErrorResult(BuiltInCommand, err), err
	}

	e.recordMetrics(command, "success", startTime)
	return NewSuccessResult(BuiltInCommand), nil
}

// ExecuteMultiCommand executes multiple commands
func (e *CommandExecutor) ExecuteMultiCommand(commandLines []string, rawCommandLines []string) (*ExecutionResult, error) {
	startTime := time.Now()
	e.context.Logger.Infof("Executing multi-command with %d commands", len(commandLines))

	// Parse command lines into sub-commands
	subCommands, err := ParseMultiCommandLines(commandLines)
	if err != nil {
		e.recordMetrics("multi", "error", startTime)
		return NewErrorResult(MultiCommand, err), err
	}

	// Validate
	if err := e.validator.ValidateMultiCommand(subCommands, rawCommandLines); err != nil {
		e.recordMetrics("multi", "error", startTime)
		return NewErrorResult(MultiCommand, err), err
	}

	// Execute all sub-commands
	results := e.executeSubCommands(subCommands)

	// Post summary comment if configured
	if e.context.Config.PostErrorsAsPRComments {
		if err := e.resultHandler.HandleMultiCommandResults(results); err != nil {
			e.context.Logger.Errorf("Failed to post multi-command summary: %v", err)
		}
	}

	// Determine overall status
	hasErrors := false
	for _, result := range results {
		if !result.Success {
			hasErrors = true
		}
	}

	status := "success"
	if hasErrors {
		status = "partial_error"
	}
	e.recordMetrics("multi", status, startTime)

	return NewMultiCommandResult(results), nil
}

// executeSubCommands executes all sub-commands and collects results
func (e *CommandExecutor) executeSubCommands(subCommands []SubCommand) []SubCommandResult {
	var results []SubCommandResult

	for _, subCmd := range subCommands {
		result := e.executeSubCommand(subCmd)
		results = append(results, result)

		// Stop on first error if configured
		if !result.Success && e.context.Config.StopOnFirstError {
			e.context.Logger.Infof("Stopping multi-command execution due to error in: %s", subCmd.Command)
			break
		}
	}

	return results
}

// executeSubCommand executes a single sub-command
func (e *CommandExecutor) executeSubCommand(subCmd SubCommand) SubCommandResult {
	cmdDisplay := GetCommandDisplayName(subCmd)
	e.context.Logger.Infof("Executing sub-command: %s", cmdDisplay)

	err := e.context.PRHandler.ExecuteCommand(subCmd.Command, subCmd.Args)

	// Record individual command metrics
	status := "success"
	if err != nil {
		status = "error"
		e.context.Logger.Errorf("Sub-command '%s' failed: %v", subCmd.Command, err)
	}
	e.context.MetricsRecorder.RecordCommandExecution(e.context.Platform, subCmd.Command, status)

	return SubCommandResult{
		Command: subCmd.Command,
		Args:    subCmd.Args,
		Success: err == nil,
		Error:   err,
	}
}

// recordMetrics records execution metrics
func (e *CommandExecutor) recordMetrics(command, status string, startTime time.Time) {
	e.context.MetricsRecorder.RecordCommandExecution(e.context.Platform, command, status)
	e.context.MetricsRecorder.RecordProcessingDuration(e.context.Platform, command, time.Since(startTime))
}
