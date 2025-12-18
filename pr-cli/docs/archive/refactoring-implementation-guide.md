# Command Execution Refactoring - Implementation Guide

## Quick Start

This guide provides concrete implementation examples for the refactoring described in `refactoring-command-execution.md`.

## Phase 1: Create Core Components

### 1.1 Create `pkg/executor/config.go`

```go
package executor

import (
    "github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
    "github.com/sirupsen/logrus"
)

// ExecutionConfig controls command execution behavior
type ExecutionConfig struct {
    // Validation settings
    ValidateCommentSender bool // Validate that comment sender posted the command
    ValidatePRStatus      bool // Validate PR is in correct state
    DebugMode            bool // Skip certain validations in debug mode
    
    // Error handling
    PostErrorsAsPRComments bool // Post errors as PR comments
    ReturnErrors          bool // Return errors to caller
    
    // Multi-command settings
    StopOnFirstError bool // Stop multi-command execution on first error
}

// ExecutionContext contains context for command execution
type ExecutionContext struct {
    PRHandler       *handler.PRHandler
    Logger          logrus.FieldLogger
    Config          *ExecutionConfig
    MetricsRecorder MetricsRecorder
    
    // Additional context
    Platform      string // For metrics
    CommentSender string // For validation
}

// NewCLIExecutionConfig creates config for CLI mode
func NewCLIExecutionConfig(debugMode bool) *ExecutionConfig {
    return &ExecutionConfig{
        ValidateCommentSender:  !debugMode,
        ValidatePRStatus:       true,
        DebugMode:             debugMode,
        PostErrorsAsPRComments: true,
        ReturnErrors:          false,
        StopOnFirstError:      false,
    }
}

// NewWebhookExecutionConfig creates config for webhook modes
func NewWebhookExecutionConfig() *ExecutionConfig {
    return &ExecutionConfig{
        ValidateCommentSender:  false, // Webhook already validated
        ValidatePRStatus:       false, // Trust webhook
        DebugMode:             false,
        PostErrorsAsPRComments: false, // Log only
        ReturnErrors:          true,
        StopOnFirstError:      false,
    }
}
```

### 1.2 Create `pkg/executor/metrics.go`

```go
package executor

import "time"

// MetricsRecorder is an interface for recording execution metrics
type MetricsRecorder interface {
    RecordCommandExecution(platform, command, status string)
    RecordProcessingDuration(platform, command string, duration time.Duration)
}

// NoOpMetricsRecorder is a no-op implementation for CLI mode
type NoOpMetricsRecorder struct{}

func (n *NoOpMetricsRecorder) RecordCommandExecution(platform, command, status string) {}
func (n *NoOpMetricsRecorder) RecordProcessingDuration(platform, command string, duration time.Duration) {}
```

### 1.3 Create `pkg/executor/result.go`

```go
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
```

### 1.4 Create `pkg/executor/validator.go`

```go
package executor

import (
    "fmt"
    "strings"
    
    "github.com/AlaudaDevops/toolbox/pr-cli/pkg/comment"
    "github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
)

// Validator handles command execution validation
type Validator struct {
    context *ExecutionContext
}

// NewValidator creates a new validator
func NewValidator(ctx *ExecutionContext) *Validator {
    return &Validator{context: ctx}
}

// ValidateSingleCommand validates a single command execution
func (v *Validator) ValidateSingleCommand(command string) error {
    // Validate PR status if required
    if v.context.Config.ValidatePRStatus && !v.shouldSkipPRStatusCheck(command) {
        if err := v.context.PRHandler.CheckPRStatus("open"); err != nil {
            return fmt.Errorf("PR status check failed: %w", err)
        }
    }
    
    // Validate comment sender if required
    if v.context.Config.ValidateCommentSender {
        if err := v.validateCommentSender(); err != nil {
            return fmt.Errorf("comment sender validation failed: %w", err)
        }
    }
    
    return nil
}

// ValidateMultiCommand validates multi-command execution
func (v *Validator) ValidateMultiCommand(subCommands []SubCommand, rawCommandLines []string) error {
    // Check if any command needs PR status validation
    needsPRCheck := false
    for _, subCmd := range subCommands {
        if !v.shouldSkipPRStatusCheck(subCmd.Command) && !handler.IsBuiltInCommand(subCmd.Command) {
            needsPRCheck = true
            break
        }
    }
    
    if v.context.Config.ValidatePRStatus && needsPRCheck {
        if err := v.context.PRHandler.CheckPRStatus("open"); err != nil {
            return fmt.Errorf("PR status check failed: %w", err)
        }
    }
    
    // Validate comment sender for multi-command
    if v.context.Config.ValidateCommentSender {
        if err := v.validateCommentSenderMulti(rawCommandLines); err != nil {
            return fmt.Errorf("comment sender validation failed: %w", err)
        }
    }
    
    return nil
}

// shouldSkipPRStatusCheck returns true if the command can work with closed PRs
func (v *Validator) shouldSkipPRStatusCheck(command string) bool {
    skipCommands := map[string]bool{
        "cherry-pick": true,
        "cherrypick":  true,
    }
    return skipCommands[command] || handler.IsBuiltInCommand(command)
}

// validateCommentSender validates that the comment sender posted the trigger comment
func (v *Validator) validateCommentSender() error {
    comments, err := v.context.PRHandler.GetCommentsWithCache()
    if err != nil {
        return fmt.Errorf("failed to get PR comments: %w", err)
    }
    
    // Get trigger comment from PR handler config
    triggerComment := v.context.PRHandler.GetConfig().TriggerComment
    normalizedTrigger := comment.Normalize(triggerComment)
    
    for _, commentObj := range comments {
        if strings.EqualFold(commentObj.User.Login, v.context.CommentSender) {
            normalizedBody := comment.Normalize(commentObj.Body)
            if normalizedBody == normalizedTrigger || strings.Contains(normalizedBody, normalizedTrigger) {
                v.context.Logger.Infof("Comment sender validation passed: %s posted the trigger", v.context.CommentSender)
                return nil
            }
        }
    }
    
    return fmt.Errorf("comment sender '%s' did not post a comment containing the trigger", v.context.CommentSender)
}

// validateCommentSenderMulti validates comment sender for multi-command
func (v *Validator) validateCommentSenderMulti(rawCommandLines []string) error {
    if len(rawCommandLines) == 0 {
        return nil
    }
    
    comments, err := v.context.PRHandler.GetCommentsWithCache()
    if err != nil {
        return fmt.Errorf("failed to get PR comments: %w", err)
    }
    
    var hasSenderComments bool
    var missingCommands []string
    
    for _, cmdLine := range rawCommandLines {
        normalizedCmdLine := comment.Normalize(cmdLine)
        found := false
        
        for _, commentObj := range comments {
            if strings.EqualFold(commentObj.User.Login, v.context.CommentSender) {
                hasSenderComments = true
                normalizedBody := comment.Normalize(commentObj.Body)
                if strings.Contains(normalizedBody, normalizedCmdLine) {
                    found = true
                    break
                }
            }
        }
        
        if !found {
            missingCommands = append(missingCommands, cmdLine)
        }
    }
    
    if !hasSenderComments {
        return fmt.Errorf("comment sender '%s' did not post any comment", v.context.CommentSender)
    }
    
    if len(missingCommands) > 0 {
        return fmt.Errorf("comment sender '%s' did not post commands: %s", 
            v.context.CommentSender, strings.Join(missingCommands, ", "))
    }
    
    v.context.Logger.Infof("Multi-command validation passed for sender: %s", v.context.CommentSender)
    return nil
}
```

### 1.5 Create `pkg/executor/result_handler.go`

```go
package executor

import (
    "errors"
    "fmt"
    "strings"
    
    "github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
    "github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
)

// ResultHandler handles execution result formatting and posting
type ResultHandler struct {
    context *ExecutionContext
}

// NewResultHandler creates a new result handler
func NewResultHandler(ctx *ExecutionContext) *ResultHandler {
    return &ResultHandler{context: ctx}
}

// HandleSingleCommandError handles error from single command
func (r *ResultHandler) HandleSingleCommandError(command string, err error) error {
    // Check if this is a CommentedError (comment already posted)
    var commentedErr *handler.CommentedError
    if errors.As(err, &commentedErr) {
        r.context.Logger.Infof("Error comment already posted for command: %s", command)
        return nil
    }
    
    // Post error as PR comment if configured
    if r.context.Config.PostErrorsAsPRComments {
        errorMessage := fmt.Sprintf(messages.CommandErrorTemplate, command, err.Error())
        if commentErr := r.context.PRHandler.PostComment(errorMessage); commentErr != nil {
            r.context.Logger.Errorf("Failed to post error comment: %v", commentErr)
            return fmt.Errorf("command failed: %w (and failed to post error comment: %v)", err, commentErr)
        }
        r.context.Logger.Infof("Posted command error as PR comment for command: %s", command)
        return nil
    }
    
    // Return error if configured
    if r.context.Config.ReturnErrors {
        return err
    }
    
    // Otherwise just log
    r.context.Logger.Errorf("Command %s failed: %v", command, err)
    return nil
}

// HandleMultiCommandResults posts summary of multi-command execution
func (r *ResultHandler) HandleMultiCommandResults(results []SubCommandResult) error {
    var formattedResults []string
    var hasErrors bool
    
    for _, result := range results {
        formatted := r.FormatSubCommandResult(result)
        formattedResults = append(formattedResults, formatted)
        if !result.Success {
            hasErrors = true
        }
    }
    
    header := "**Multi-Command Execution Results:**"
    if hasErrors {
        header = fmt.Sprintf("%s (⚠️ Some commands failed)", header)
    }
    
    summary := fmt.Sprintf("%s\n\n%s", header, strings.Join(formattedResults, "\n"))
    return r.context.PRHandler.PostComment(summary)
}

// FormatSubCommandResult formats a sub-command result
func (r *ResultHandler) FormatSubCommandResult(result SubCommandResult) string {
    cmdDisplay := GetCommandDisplayName(SubCommand{
        Command: result.Command,
        Args:    result.Args,
    })
    
    if result.Success {
        return fmt.Sprintf("✅ Command `%s` executed successfully", cmdDisplay)
    }
    return fmt.Sprintf("❌ Command `%s` failed: %v", cmdDisplay, result.Error)
}
```

## Next Steps

Continue to Phase 2 in the main refactoring document to implement the CommandExecutor and integrate with existing code.

See `refactoring-command-execution.md` for:
- Complete architecture overview
- Migration plan
- Testing strategy
- Code examples for all phases

