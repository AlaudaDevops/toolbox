# Command Execution Refactoring Design

## Current State Analysis

### Overview
The PR CLI tool currently has **duplicated logic** across three execution contexts:
1. **CLI Mode** (`cmd/` package) - Direct command execution via CLI
2. **Webhook Worker Mode** (`pkg/webhook/worker.go`) - Async processing with worker pool
3. **Webhook Sync Mode** (`pkg/webhook/server.go`) - Synchronous webhook processing

### Current Architecture Issues

#### 1. **Duplicated Command Execution Logic**

**CLI Mode** (`cmd/executor.go`, `cmd/multi_command.go`):
- `executeSingleCommand()` - Validates PR status, comment sender, executes command, handles errors
- `executeBuiltInCommand()` - Executes built-in commands
- `executeMultiCommand()` - Parses, validates, executes multiple commands, posts summary
- `processMultiCommand()` - Processes individual commands in multi-command context
- `validateCommentSender()` - Validates comment sender
- `validateMultiCommandExecution()` - Validates multi-command execution

**Worker Mode** (`pkg/webhook/worker.go`):
- `executeSingleCommand()` - Executes single commands with metrics
- `executeMultiCommand()` - Executes multiple commands with metrics
- `processSubCommand()` - Processes individual sub-commands
- **Missing**: Comment sender validation, PR status checks, error handling with PR comments

**Sync Mode** (`pkg/webhook/server.go`):
- `processWebhookSync()` - Basic command execution
- **Missing**: Multi-command support (TODO comment), validation, error handling

#### 2. **Type Duplication**
- `executor.SubCommand` (in `pkg/executor/types.go`)
- `handler.SubCommand` (in `pkg/handler/handle_batch.go`)
- Both have identical structure: `Command string`, `Args []string`
- Requires conversion between types in multiple places

#### 3. **Inconsistent Validation**
- CLI mode has comprehensive validation (PR status, comment sender)
- Worker mode has NO validation
- Sync mode has NO validation
- Webhook modes rely on webhook validation only (repository, signature)

#### 4. **Inconsistent Error Handling**
- CLI mode posts errors as PR comments
- Worker mode logs errors but doesn't post to PR
- Sync mode returns errors to HTTP response

#### 5. **Inconsistent Multi-Command Support**
- CLI mode: Full support with validation and summary
- Worker mode: Full support but no validation
- Sync mode: **NOT IMPLEMENTED** (TODO comment)

## Proposed Solution

### Design Principles
1. **Single Responsibility**: Separate parsing, validation, execution, and result handling
2. **DRY (Don't Repeat Yourself)**: Extract common logic into reusable components
3. **Consistency**: Same behavior across CLI and webhook modes
4. **Testability**: Easy to unit test each component
5. **Maintainability**: Clear separation of concerns

### New Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Entry Points                             │
├─────────────────┬─────────────────┬─────────────────────────┤
│   CLI Mode      │  Worker Mode    │   Sync Mode             │
│  (cmd/options)  │ (webhook/worker)│ (webhook/server)        │
└────────┬────────┴────────┬────────┴────────┬────────────────┘
         │                 │                 │
         └─────────────────┼─────────────────┘
                           │
                           ▼
         ┌─────────────────────────────────────┐
         │   Command Executor (NEW)            │
         │   pkg/executor/executor.go          │
         │                                     │
         │  - ExecuteCommand()                 │
         │  - ExecuteMultiCommand()            │
         │  - ExecuteSingleCommand()           │
         │  - ExecuteBuiltInCommand()          │
         └──────────────┬──────────────────────┘
                        │
         ┌──────────────┼──────────────────────┐
         │              │                      │
         ▼              ▼                      ▼
┌────────────────┐ ┌──────────────┐ ┌─────────────────┐
│   Validator    │ │   Handler    │ │ Result Handler  │
│   (NEW)        │ │  (existing)  │ │     (NEW)       │
│                │ │              │ │                 │
│ - ValidatePR   │ │ - Execute    │ │ - FormatResult  │
│ - ValidateSender│ │   Command   │ │ - PostSummary   │
└────────────────┘ └──────────────┘ └─────────────────┘
```

### Component Design

#### 1. **Unified Command Executor** (`pkg/executor/executor.go`)

**Purpose**: Central command execution logic used by all modes

```go
package executor

// ExecutionContext contains context for command execution
type ExecutionContext struct {
    PRHandler      *handler.PRHandler
    Logger         logrus.FieldLogger
    Config         *ExecutionConfig
    MetricsRecorder MetricsRecorder // Optional, for webhook modes
}

// ExecutionConfig controls execution behavior
type ExecutionConfig struct {
    // Validation settings
    ValidateCommentSender bool
    ValidatePRStatus      bool
    DebugMode            bool

    // Error handling
    PostErrorsAsPRComments bool

    // Multi-command settings
    StopOnFirstError bool
}

// ExecutionResult represents the result of command execution
type ExecutionResult struct {
    Success      bool
    Error        error
    CommandType  CommandType
    Results      []SubCommandResult // For multi-command
}

// SubCommandResult represents result of a single sub-command
type SubCommandResult struct {
    Command string
    Args    []string
    Success bool
    Error   error
}

// CommandExecutor handles unified command execution
type CommandExecutor struct {
    context *ExecutionContext
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(ctx *ExecutionContext) *CommandExecutor

// Execute executes a parsed command with validation and error handling
func (e *CommandExecutor) Execute(parsedCmd *ParsedCommand) (*ExecutionResult, error)

// ExecuteSingleCommand executes a single command
func (e *CommandExecutor) ExecuteSingleCommand(command string, args []string) error

// ExecuteMultiCommand executes multiple commands
func (e *CommandExecutor) ExecuteMultiCommand(commandLines, rawCommandLines []string) (*ExecutionResult, error)

// ExecuteBuiltInCommand executes a built-in command
func (e *CommandExecutor) ExecuteBuiltInCommand(command string, args []string) error
```

#### 2. **Unified Validator** (`pkg/executor/validator.go`)

**Purpose**: Centralized validation logic

```go
package executor

// Validator handles command execution validation
type Validator struct {
    prHandler *handler.PRHandler
    config    *ExecutionConfig
    logger    logrus.FieldLogger
}

// ValidateSingleCommand validates a single command execution
func (v *Validator) ValidateSingleCommand(command string, commentSender string) error

// ValidateMultiCommand validates multi-command execution
func (v *Validator) ValidateMultiCommand(subCommands []SubCommand, rawCommandLines []string, commentSender string) error

// ValidatePRStatus validates PR status for command
func (v *Validator) ValidatePRStatus(command string) error

// ValidateCommentSender validates that sender posted the command
func (v *Validator) ValidateCommentSender(triggerComment, commentSender string) error

// ValidateCommentSenderMulti validates sender for multi-command
func (v *Validator) ValidateCommentSenderMulti(rawCommandLines []string, commentSender string) error
```

#### 3. **Result Handler** (`pkg/executor/result_handler.go`)

**Purpose**: Format and post execution results

```go
package executor

// ResultHandler handles execution result formatting and posting
type ResultHandler struct {
    prHandler *handler.PRHandler
    logger    logrus.FieldLogger
}

// HandleSingleCommandError handles error from single command
func (r *ResultHandler) HandleSingleCommandError(command string, err error) error

// HandleMultiCommandResults posts summary of multi-command execution
func (r *ResultHandler) HandleMultiCommandResults(results []SubCommandResult) error

// FormatSubCommandResult formats a sub-command result
func (r *ResultHandler) FormatSubCommandResult(result SubCommandResult) string
```

#### 4. **Remove Type Duplication**

**Action**: Remove `handler.SubCommand`, use only `executor.SubCommand`

- Update `pkg/handler/handle_batch.go` to use `executor.SubCommand`
- Remove conversion logic in `cmd/multi_command.go`
- Update all references

#### 5. **Metrics Interface** (`pkg/executor/metrics.go`)

**Purpose**: Allow webhook modes to record metrics without coupling

```go
package executor

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

### Migration Plan

#### Phase 1: Create New Components (No Breaking Changes)
1. Create `pkg/executor/executor.go` with `CommandExecutor`
2. Create `pkg/executor/validator.go` with `Validator`
3. Create `pkg/executor/result_handler.go` with `ResultHandler`
4. Create `pkg/executor/metrics.go` with metrics interface
5. Add comprehensive unit tests

#### Phase 2: Migrate CLI Mode
1. Update `cmd/options.go` to use `CommandExecutor`
2. Remove old execution methods from `cmd/executor.go` and `cmd/multi_command.go`
3. Update tests

#### Phase 3: Migrate Worker Mode
1. Create webhook metrics recorder implementing `MetricsRecorder`
2. Update `pkg/webhook/worker.go` to use `CommandExecutor`
3. Remove old execution methods
4. Update tests

#### Phase 4: Migrate Sync Mode
1. Update `pkg/webhook/server.go` to use `CommandExecutor`
2. Implement multi-command support
3. Update tests

#### Phase 5: Remove Type Duplication
1. Update `pkg/handler/handle_batch.go` to use `executor.SubCommand`
2. Remove conversion logic
3. Update all tests

#### Phase 6: Cleanup
1. Remove deprecated code
2. Update documentation
3. Final integration tests

### Benefits

1. **Reduced Code Duplication**: ~60% reduction in command execution code
2. **Consistent Behavior**: Same validation and error handling across all modes
3. **Better Testability**: Each component can be tested independently
4. **Easier Maintenance**: Changes in one place affect all modes
5. **Feature Parity**: Multi-command support in all modes
6. **Better Error Handling**: Consistent error reporting
7. **Metrics Support**: Optional metrics without coupling

### Configuration Examples

#### CLI Mode
```go
executor := executor.NewCommandExecutor(&executor.ExecutionContext{
    PRHandler: prHandler,
    Logger:    logger,
    Config: &executor.ExecutionConfig{
        ValidateCommentSender:  !debugMode,
        ValidatePRStatus:       true,
        PostErrorsAsPRComments: true,
        StopOnFirstError:       false,
    },
    MetricsRecorder: &executor.NoOpMetricsRecorder{},
})
```

#### Worker Mode
```go
executor := executor.NewCommandExecutor(&executor.ExecutionContext{
    PRHandler: prHandler,
    Logger:    logger,
    Config: &executor.ExecutionConfig{
        ValidateCommentSender:  false, // Webhook already validated
        ValidatePRStatus:       false, // Trust webhook
        PostErrorsAsPRComments: false, // Log only
        StopOnFirstError:       false,
    },
    MetricsRecorder: &WebhookMetricsRecorder{platform: platform},
})
```

### Testing Strategy

1. **Unit Tests**: Each component (executor, validator, result handler)
2. **Integration Tests**: Full flow with mock PRHandler
3. **Regression Tests**: Ensure existing behavior is preserved
4. **E2E Tests**: Test all three modes with real scenarios

### Backward Compatibility

- All changes are internal refactoring
- No changes to CLI flags or webhook API
- Existing behavior is preserved
- Tests ensure no regressions

## Current State Comparison

### Feature Matrix

| Feature | CLI Mode | Worker Mode | Sync Mode | Proposed (All Modes) |
|---------|----------|-------------|-----------|---------------------|
| Single Command | ✅ Full | ✅ Full | ✅ Full | ✅ Full |
| Multi Command | ✅ Full | ✅ Full | ❌ TODO | ✅ Full |
| Built-in Command | ✅ Full | ✅ Full | ✅ Full | ✅ Full |
| PR Status Validation | ✅ Yes | ❌ No | ❌ No | ⚙️ Configurable |
| Comment Sender Validation | ✅ Yes | ❌ No | ❌ No | ⚙️ Configurable |
| Error Posting to PR | ✅ Yes | ❌ No | ❌ No | ⚙️ Configurable |
| Metrics Recording | ❌ No | ✅ Yes | ✅ Yes | ⚙️ Configurable |
| Debug Mode | ✅ Yes | ❌ No | ❌ No | ⚙️ Configurable |

### Code Duplication Analysis

| Component | CLI Mode | Worker Mode | Sync Mode | Lines of Code | Proposed |
|-----------|----------|-------------|-----------|---------------|----------|
| Single Command Execution | `cmd/executor.go:47-72` | `worker.go:123-137` | `server.go:176-182` | ~60 lines | 1 implementation |
| Multi Command Execution | `cmd/multi_command.go:28-117` | `worker.go:140-186` | ❌ Missing | ~150 lines | 1 implementation |
| Sub-command Processing | `cmd/multi_command.go:119-135` | `worker.go:188-201` | ❌ Missing | ~30 lines | 1 implementation |
| Comment Sender Validation | `cmd/validator.go:42-73` | ❌ Missing | ❌ Missing | ~30 lines | 1 implementation |
| Multi-command Validation | `cmd/multi_command.go:137-187` | ❌ Missing | ❌ Missing | ~50 lines | 1 implementation |
| Error Handling | `cmd/executor.go:74-96` | ❌ Missing | ❌ Missing | ~25 lines | 1 implementation |
| **Total Duplicated** | **~345 lines** | **~100 lines** | **~10 lines** | **~455 lines** | **~200 lines** |

**Reduction**: ~56% code reduction (455 → 200 lines)

### Current Issues Summary

#### 1. **Inconsistent Validation**
```go
// CLI Mode - Full validation
if !p.shouldSkipPRStatusCheck(command) {
    prHandler.CheckPRStatus("open")
}
if !p.Config.Debug {
    p.validateCommentSender(prHandler)
}

// Worker Mode - NO validation
// Just executes directly
prHandler.ExecuteCommand(command, args)

// Sync Mode - NO validation
// Just executes directly
prHandler.ExecuteCommand(parsedCmd.Command, parsedCmd.Args)
```

**Problem**: Security and consistency issues. Webhook modes trust webhook validation but don't verify PR state or comment sender.

#### 2. **Inconsistent Error Handling**
```go
// CLI Mode - Posts errors to PR
if err := prHandler.ExecuteCommand(command, args); err != nil {
    errorMessage := fmt.Sprintf(messages.CommandErrorTemplate, command, err.Error())
    prHandler.PostComment(errorMessage)
    return nil // Don't fail the pipeline
}

// Worker Mode - Only logs
if err := prHandler.ExecuteCommand(command, args); err != nil {
    logger.Errorf("Failed to execute command: %v", err)
    // No PR comment posted
}

// Sync Mode - Returns error to HTTP
if err = prHandler.ExecuteCommand(parsedCmd.Command, parsedCmd.Args); err != nil {
    return fmt.Errorf("failed to execute command: %w", err)
}
```

**Problem**: Users get different feedback depending on execution mode.

#### 3. **Type Conversion Overhead**
```go
// cmd/multi_command.go - Converting executor.SubCommand to handler.SubCommand
for _, subCmd := range subCommands {
    handlerSubCommands = append(handlerSubCommands, handler.SubCommand{
        Command: subCmd.Command,
        Args:    subCmd.Args,
    })
}

// Later converting back for display
execSubCmd := executor.SubCommand{
    Command: subCmd.Command,
    Args:    subCmd.Args,
}
```

**Problem**: Unnecessary conversions, maintenance burden, potential for bugs.

#### 4. **Missing Multi-Command in Sync Mode**
```go
// pkg/webhook/server.go:183-186
case executor.MultiCommand:
    // TODO: Implement multicommand executor
    //CommandExecutionTotal.WithLabelValues(event.Platform, parsedCmd.Command, "success").Inc()
    // WebhookProcessingDuration.WithLabelValues(event.Platform, parsedCmd.Command).Observe(time.Since(startTime).Seconds())
```

**Problem**: Feature gap - sync mode doesn't support multi-command execution.

### Proposed Solution Benefits

#### 1. **Unified Validation**
```go
// All modes use the same executor with different configs
executor := executor.NewCommandExecutor(&executor.ExecutionContext{
    PRHandler: prHandler,
    Logger:    logger,
    Config: &executor.ExecutionConfig{
        ValidateCommentSender:  configurable,
        ValidatePRStatus:       configurable,
        DebugMode:             configurable,
    },
})
```

#### 2. **Consistent Error Handling**
```go
// Single error handling logic with configurable behavior
if config.PostErrorsAsPRComments {
    resultHandler.HandleSingleCommandError(command, err)
} else if config.ReturnErrors {
    return err
} else {
    logger.Errorf("Command failed: %v", err)
}
```

#### 3. **No Type Conversion**
```go
// Use executor.SubCommand everywhere
subCommands := executor.ParseMultiCommandLines(commandLines)
for _, subCmd := range subCommands {
    // Direct use, no conversion
    result := executor.ExecuteSubCommand(subCmd)
}
```

#### 4. **Complete Feature Parity**
All modes support:
- ✅ Single commands
- ✅ Multi commands
- ✅ Built-in commands
- ✅ Configurable validation
- ✅ Configurable error handling
- ✅ Configurable metrics

