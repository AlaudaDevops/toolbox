# Unified Command Executor - Design Document

## Table of Contents

1. [Overview](#1-overview)
2. [Problem Statement](#2-problem-statement)
3. [Architecture Design](#3-architecture-design)
4. [Component Specifications](#4-component-specifications)
5. [Testing Strategy](#5-testing-strategy)
6. [Configuration Examples](#6-configuration-examples)
7. [Migration Considerations](#7-migration-considerations)

---

## 1. Overview

### 1.1 Purpose

This document describes the design for a unified command execution engine that consolidates duplicated logic across CLI, Webhook Worker, and Webhook Sync modes into a single, configurable component.

### 1.2 Goals

- **Unify** command execution logic into a single reusable engine
- **Eliminate** code duplication (~56% reduction)
- **Enable** consistent validation and error handling across all modes
- **Achieve** complete feature parity (multi-command support in all modes)
- **Improve** testability with comprehensive automated tests

### 1.3 Scope

| In Scope | Out of Scope |
|----------|--------------|
| Command parsing and execution | Webhook signature validation |
| PR status validation | Repository permissions |
| Comment sender validation | Platform-specific APIs |
| Error handling and result formatting | CLI argument parsing |
| Metrics recording interface | UI/UX changes |

---

## 2. Problem Statement

### 2.1 Current Architecture Issues

The PR CLI tool has **three execution contexts** with duplicated logic:

| Context | Location | Issues |
|---------|----------|--------|
| CLI Mode | `cmd/executor.go`, `cmd/multi_command.go` | Full validation, posts errors to PR |
| Worker Mode | `pkg/webhook/worker.go` | No validation, logs only |
| Sync Mode | `pkg/webhook/server.go` | No validation, missing multi-command |

### 2.2 Code Duplication Analysis

| Component | CLI Mode | Worker Mode | Sync Mode | Total |
|-----------|----------|-------------|-----------|-------|
| Single Command Execution | ~25 lines | ~15 lines | ~10 lines | ~50 lines |
| Multi Command Execution | ~90 lines | ~50 lines | Missing | ~140 lines |
| Sub-command Processing | ~20 lines | ~15 lines | Missing | ~35 lines |
| Comment Sender Validation | ~30 lines | Missing | Missing | ~30 lines |
| Multi-command Validation | ~50 lines | Missing | Missing | ~50 lines |
| Error Handling | ~25 lines | Missing | Missing | ~25 lines |
| **Total** | **~240 lines** | **~80 lines** | **~10 lines** | **~330 lines** |

**Target**: Reduce to ~150 lines in unified engine.

### 2.3 Inconsistencies

#### Validation Differences
```
CLI Mode:        PR Status ✓   Comment Sender ✓   Debug Mode ✓
Worker Mode:     PR Status ✗   Comment Sender ✗   Debug Mode ✗
Sync Mode:       PR Status ✗   Comment Sender ✗   Debug Mode ✗
```

#### Error Handling Differences
```
CLI Mode:        Posts error as PR comment, returns nil
Worker Mode:     Logs error, records metric
Sync Mode:       Returns error to HTTP response
```

#### Feature Gaps
```
Multi-command:   CLI ✓   Worker ✓   Sync ✗ (TODO in code)
```

---

## 3. Architecture Design

### 3.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Entry Points                              │
├───────────────────┬───────────────────┬─────────────────────────┤
│    CLI Mode       │   Worker Mode     │      Sync Mode          │
│   cmd/options.go  │ webhook/worker.go │   webhook/server.go     │
└─────────┬─────────┴─────────┬─────────┴───────────┬─────────────┘
          │                   │                     │
          │     ┌─────────────┴─────────────┐       │
          └─────┤   ExecutionContext        ├───────┘
                │   (Config + Dependencies) │
                └─────────────┬─────────────┘
                              │
                              ▼
          ┌───────────────────────────────────────────┐
          │        CommandExecutor (NEW)              │
          │        pkg/executor/executor.go           │
          │                                           │
          │  ┌─────────────┐  ┌──────────────────┐   │
          │  │  Validator  │  │  ResultHandler   │   │
          │  └─────────────┘  └──────────────────┘   │
          └───────────────────────────────────────────┘
                              │
                              ▼
          ┌───────────────────────────────────────────┐
          │          Existing Components              │
          │  PRHandler │ GitClient │ Parser          │
          └───────────────────────────────────────────┘
```

### 3.2 Component Interaction Flow

```
Execute() Entry Point
         │
         ▼
   ┌─────────────┐
   │ Parse Input │ ─────► Already parsed (ParsedCommand)
   └─────────────┘
         │
         ▼
   ┌─────────────────────┐
   │ Determine Type      │
   │ Single/Multi/BuiltIn│
   └─────────────────────┘
         │
    ┌────┴────┬──────────┐
    ▼         ▼          ▼
┌────────┐ ┌────────┐ ┌────────┐
│ Single │ │ Multi  │ │BuiltIn │
│Command │ │Command │ │Command │
└────┬───┘ └────┬───┘ └────┬───┘
     │          │          │
     └────┬─────┴──────────┘
          ▼
   ┌─────────────────────┐
   │ Validator           │
   │ (if config enabled) │
   │ - PR Status         │
   │ - Comment Sender    │
   └─────────────────────┘
          │
          ▼
   ┌─────────────────────┐
   │ Execute Command(s)  │
   │ via PRHandler       │
   └─────────────────────┘
          │
          ▼
   ┌─────────────────────┐
   │ ResultHandler       │
   │ - Format results    │
   │ - Post to PR/Log    │
   │ - Record metrics    │
   └─────────────────────┘
          │
          ▼
   ┌─────────────────────┐
   │ Return Result       │
   └─────────────────────┘
```

### 3.3 File Structure

```
pkg/executor/
├── types.go              # Existing: ParsedCommand, SubCommand, CommandType
├── parser.go             # Existing: ParseCommand, ParseMultiCommandLines
├── config.go             # NEW: ExecutionConfig, ExecutionContext
├── executor.go           # NEW: CommandExecutor main logic
├── validator.go          # NEW: Validator component
├── result_handler.go     # NEW: ResultHandler component
├── result.go             # NEW: ExecutionResult, SubCommandResult
├── metrics.go            # NEW: MetricsRecorder interface
├── executor_test.go      # NEW: Unit tests
├── validator_test.go     # NEW: Unit tests
├── result_handler_test.go# NEW: Unit tests
└── integration_test.go   # NEW: Integration tests
```

---

## 4. Component Specifications

### 4.1 ExecutionConfig

**Purpose**: Configure execution behavior per mode

**File**: `pkg/executor/config.go`

```go
// ExecutionConfig controls command execution behavior
type ExecutionConfig struct {
    // Validation settings
    ValidateCommentSender bool   // Check if comment sender posted the command
    ValidatePRStatus      bool   // Check if PR is in correct state
    DebugMode             bool   // Skip certain validations in debug mode
    
    // Error handling
    PostErrorsAsPRComments bool  // Post errors as PR comments
    ReturnErrors           bool  // Return errors to caller (for sync mode)
    
    // Multi-command settings
    StopOnFirstError bool        // Stop multi-command on first error
}
```

### 4.2 ExecutionContext

**Purpose**: Hold dependencies and runtime context

**File**: `pkg/executor/config.go`

```go
// ExecutionContext contains runtime context for command execution
type ExecutionContext struct {
    PRHandler       *handler.PRHandler
    Logger          logrus.FieldLogger
    Config          *ExecutionConfig
    MetricsRecorder MetricsRecorder
    
    // Runtime context
    Platform      string // For metrics: "github", "gitlab", "gitee"
    CommentSender string // User who triggered the command
}
```

### 4.3 ExecutionResult

**Purpose**: Standardized result type for all executions

**File**: `pkg/executor/result.go`

```go
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
```

### 4.4 CommandExecutor

**Purpose**: Main orchestrator for command execution

**File**: `pkg/executor/executor.go`

```go
// CommandExecutor handles unified command execution
type CommandExecutor struct {
    context       *ExecutionContext
    validator     *Validator
    resultHandler *ResultHandler
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(ctx *ExecutionContext) *CommandExecutor

// Execute executes a parsed command with full lifecycle
func (e *CommandExecutor) Execute(parsedCmd *ParsedCommand) (*ExecutionResult, error)

// ExecuteSingleCommand executes a single command
func (e *CommandExecutor) ExecuteSingleCommand(command string, args []string) (*ExecutionResult, error)

// ExecuteMultiCommand executes multiple commands
func (e *CommandExecutor) ExecuteMultiCommand(commandLines, rawCommandLines []string) (*ExecutionResult, error)

// ExecuteBuiltInCommand executes a built-in command (help, version)
func (e *CommandExecutor) ExecuteBuiltInCommand(command string, args []string) (*ExecutionResult, error)
```

### 4.5 Validator

**Purpose**: Centralized validation logic

**File**: `pkg/executor/validator.go`

```go
// Validator handles command execution validation
type Validator struct {
    context *ExecutionContext
}

// NewValidator creates a new validator
func NewValidator(ctx *ExecutionContext) *Validator

// ValidateSingleCommand validates a single command execution
func (v *Validator) ValidateSingleCommand(command string) error

// ValidateMultiCommand validates multi-command execution
func (v *Validator) ValidateMultiCommand(subCommands []SubCommand, rawCommandLines []string) error

// shouldSkipPRStatusCheck returns true for commands that work on closed PRs
func (v *Validator) shouldSkipPRStatusCheck(command string) bool
```

**Skip PR Status Check Commands**:
- `cherry-pick` / `cherrypick`
- Built-in commands (`help`, `version`)

### 4.6 ResultHandler

**Purpose**: Format and post execution results

**File**: `pkg/executor/result_handler.go`

```go
// ResultHandler handles execution result formatting and posting
type ResultHandler struct {
    context *ExecutionContext
}

// NewResultHandler creates a new result handler
func NewResultHandler(ctx *ExecutionContext) *ResultHandler

// HandleSingleCommandError handles error from single command
func (r *ResultHandler) HandleSingleCommandError(command string, err error) error

// HandleMultiCommandResults posts summary of multi-command execution
func (r *ResultHandler) HandleMultiCommandResults(results []SubCommandResult) error

// FormatSubCommandResult formats a sub-command result for display
func (r *ResultHandler) FormatSubCommandResult(result SubCommandResult) string
```

**Multi-command Result Format**:
```
**Multi-Command Execution Results:**

✅ Command `/rebase` executed successfully
✅ Command `/lgtm` executed successfully
❌ Command `/merge` failed: PR not approved
```

### 4.7 MetricsRecorder Interface

**Purpose**: Allow webhook modes to record metrics without coupling

**File**: `pkg/executor/metrics.go`

```go
// MetricsRecorder is an interface for recording execution metrics
type MetricsRecorder interface {
    RecordCommandExecution(platform, command, status string)
    RecordProcessingDuration(platform, command string, duration time.Duration)
}

// NoOpMetricsRecorder is a no-op implementation for CLI mode
type NoOpMetricsRecorder struct{}

func (n *NoOpMetricsRecorder) RecordCommandExecution(platform, command, status string) {}
func (n *NoOpMetricsRecorder) RecordProcessingDuration(platform, command string, d time.Duration) {}
```

**Webhook Implementation** (in `pkg/webhook/metrics_recorder.go`):
```go
// WebhookMetricsRecorder implements MetricsRecorder for webhook modes
type WebhookMetricsRecorder struct {
    platform string
}

func (w *WebhookMetricsRecorder) RecordCommandExecution(platform, command, status string) {
    CommandExecutionTotal.WithLabelValues(platform, command, status).Inc()
}

func (w *WebhookMetricsRecorder) RecordProcessingDuration(platform, command string, d time.Duration) {
    WebhookProcessingDuration.WithLabelValues(platform, command).Observe(d.Seconds())
}
```

---

## 5. Testing Strategy

### 5.1 Test Requirements

| Component | Coverage Target | Test Types |
|-----------|-----------------|------------|
| CommandExecutor | 90% | Unit, Integration |
| Validator | 95% | Unit |
| ResultHandler | 90% | Unit |
| Config | 100% | Unit |

### 5.2 Mock Interfaces

To enable testing without external dependencies, define mock interfaces:

**File**: `pkg/executor/interfaces.go`

```go
// PRHandlerInterface abstracts PRHandler for testing
type PRHandlerInterface interface {
    ExecuteCommand(command string, args []string) error
    ExecuteBuiltInCommand(command string, args []string) error
    PostComment(body string) error
    CheckPRStatus(expectedStatus string) error
    GetCommentsWithCache() ([]Comment, error)
    GetConfig() *PRConfig
}

// Comment represents a PR comment
type Comment struct {
    User struct {
        Login string
    }
    Body string
}

// PRConfig represents PR handler configuration
type PRConfig struct {
    TriggerComment string
}
```

### 5.3 Test Categories

#### 5.3.1 Unit Tests

**Validator Tests** (`validator_test.go`):

```go
func TestValidator_ValidateSingleCommand(t *testing.T) {
    tests := []struct {
        name           string
        command        string
        config         *ExecutionConfig
        prStatus       string
        comments       []Comment
        commentSender  string
        wantErr        bool
        errContains    string
    }{
        {
            name:    "valid command with open PR",
            command: "rebase",
            config:  &ExecutionConfig{ValidatePRStatus: true},
            prStatus: "open",
            wantErr: false,
        },
        {
            name:    "cherry-pick allows closed PR",
            command: "cherry-pick",
            config:  &ExecutionConfig{ValidatePRStatus: true},
            prStatus: "closed",
            wantErr: false,
        },
        {
            name:    "command fails on closed PR",
            command: "rebase",
            config:  &ExecutionConfig{ValidatePRStatus: true},
            prStatus: "closed",
            wantErr: true,
            errContains: "PR status",
        },
        {
            name:    "comment sender validation passes",
            command: "rebase",
            config:  &ExecutionConfig{ValidateCommentSender: true},
            comments: []Comment{{User: struct{Login string}{Login: "user1"}, Body: "/rebase"}},
            commentSender: "user1",
            wantErr: false,
        },
        {
            name:    "comment sender validation fails",
            command: "rebase",
            config:  &ExecutionConfig{ValidateCommentSender: true},
            comments: []Comment{{User: struct{Login string}{Login: "other"}, Body: "/rebase"}},
            commentSender: "user1",
            wantErr: true,
            errContains: "comment sender",
        },
        {
            name:    "debug mode skips comment sender validation",
            command: "rebase",
            config:  &ExecutionConfig{ValidateCommentSender: true, DebugMode: true},
            commentSender: "user1",
            wantErr: false,
        },
    }
    // ... test implementation
}

func TestValidator_ValidateMultiCommand(t *testing.T) {
    // Test multi-command validation scenarios
}

func TestValidator_shouldSkipPRStatusCheck(t *testing.T) {
    tests := []struct {
        command string
        want    bool
    }{
        {"cherry-pick", true},
        {"cherrypick", true},
        {"help", true},
        {"version", true},
        {"rebase", false},
        {"merge", false},
        {"lgtm", false},
    }
    // ... test implementation
}
```

**ResultHandler Tests** (`result_handler_test.go`):

```go
func TestResultHandler_HandleSingleCommandError(t *testing.T) {
    tests := []struct {
        name                   string
        command                string
        err                    error
        postErrorsAsPRComments bool
        returnErrors           bool
        wantPosted             bool
        wantErr                bool
    }{
        {
            name:                   "posts error as PR comment",
            command:                "rebase",
            err:                    errors.New("rebase failed"),
            postErrorsAsPRComments: true,
            wantPosted:             true,
            wantErr:                false,
        },
        {
            name:                   "returns error when configured",
            command:                "rebase",
            err:                    errors.New("rebase failed"),
            returnErrors:           true,
            wantErr:                true,
        },
        {
            name:                   "CommentedError skips posting",
            command:                "rebase",
            err:                    &handler.CommentedError{Err: errors.New("already posted")},
            postErrorsAsPRComments: true,
            wantPosted:             false,
            wantErr:                false,
        },
    }
    // ... test implementation
}

func TestResultHandler_HandleMultiCommandResults(t *testing.T) {
    // Test multi-command result formatting and posting
}

func TestResultHandler_FormatSubCommandResult(t *testing.T) {
    tests := []struct {
        result SubCommandResult
        want   string
    }{
        {
            result: SubCommandResult{Command: "rebase", Success: true},
            want:   "✅ Command `/rebase` executed successfully",
        },
        {
            result: SubCommandResult{Command: "merge", Args: []string{"--squash"}, Success: true},
            want:   "✅ Command `/merge --squash` executed successfully",
        },
        {
            result: SubCommandResult{Command: "merge", Success: false, Error: errors.New("not approved")},
            want:   "❌ Command `/merge` failed: not approved",
        },
    }
    // ... test implementation
}
```

**CommandExecutor Tests** (`executor_test.go`):

```go
func TestCommandExecutor_Execute(t *testing.T) {
    tests := []struct {
        name       string
        parsedCmd  *ParsedCommand
        config     *ExecutionConfig
        mockSetup  func(*MockPRHandler)
        wantResult *ExecutionResult
        wantErr    bool
    }{
        {
            name: "single command success",
            parsedCmd: &ParsedCommand{
                Type:    SingleCommand,
                Command: "rebase",
            },
            config: &ExecutionConfig{},
            mockSetup: func(m *MockPRHandler) {
                m.On("ExecuteCommand", "rebase", []string(nil)).Return(nil)
            },
            wantResult: &ExecutionResult{Success: true, CommandType: SingleCommand},
        },
        {
            name: "multi command partial failure",
            parsedCmd: &ParsedCommand{
                Type:         MultiCommand,
                CommandLines: []string{"/rebase", "/merge"},
            },
            config: &ExecutionConfig{},
            mockSetup: func(m *MockPRHandler) {
                m.On("ExecuteCommand", "rebase", []string(nil)).Return(nil)
                m.On("ExecuteCommand", "merge", []string(nil)).Return(errors.New("not approved"))
                m.On("PostComment", mock.Anything).Return(nil)
            },
            wantResult: &ExecutionResult{
                Success:     false,
                CommandType: MultiCommand,
                Results: []SubCommandResult{
                    {Command: "rebase", Success: true},
                    {Command: "merge", Success: false, Error: errors.New("not approved")},
                },
            },
        },
        {
            name: "built-in command",
            parsedCmd: &ParsedCommand{
                Type:    BuiltIn,
                Command: "help",
            },
            config: &ExecutionConfig{},
            mockSetup: func(m *MockPRHandler) {
                m.On("ExecuteBuiltInCommand", "help", []string(nil)).Return(nil)
            },
            wantResult: &ExecutionResult{Success: true, CommandType: BuiltIn},
        },
    }
    // ... test implementation
}

func TestCommandExecutor_ExecuteSingleCommand(t *testing.T) {
    // Detailed single command tests
}

func TestCommandExecutor_ExecuteMultiCommand(t *testing.T) {
    // Detailed multi-command tests
}

func TestCommandExecutor_StopOnFirstError(t *testing.T) {
    // Test StopOnFirstError config behavior
}
```

#### 5.3.2 Integration Tests

**File**: `pkg/executor/integration_test.go`

```go
func TestIntegration_CLIModeExecution(t *testing.T) {
    // Test full CLI mode flow with mock PRHandler
    ctx := &ExecutionContext{
        PRHandler: mockPRHandler,
        Logger:    logrus.NewEntry(logrus.New()),
        Config:    NewCLIExecutionConfig(false),
        MetricsRecorder: &NoOpMetricsRecorder{},
        Platform:  "github",
        CommentSender: "testuser",
    }
    
    executor := NewCommandExecutor(ctx)
    
    // Test scenarios:
    // 1. Single command with validation
    // 2. Multi-command with summary posting
    // 3. Error handling with PR comment
    // 4. Debug mode behavior
}

func TestIntegration_WebhookModeExecution(t *testing.T) {
    // Test full webhook mode flow with mock PRHandler
    ctx := &ExecutionContext{
        PRHandler: mockPRHandler,
        Logger:    logrus.NewEntry(logrus.New()),
        Config:    NewWebhookExecutionConfig(),
        MetricsRecorder: &mockMetricsRecorder{},
        Platform:  "github",
        CommentSender: "webhookuser",
    }
    
    executor := NewCommandExecutor(ctx)
    
    // Test scenarios:
    // 1. Single command without validation
    // 2. Multi-command with metrics
    // 3. Error handling without PR comment
}

func TestIntegration_MetricsRecording(t *testing.T) {
    // Test metrics are recorded correctly
    recorder := &mockMetricsRecorder{}
    
    ctx := &ExecutionContext{
        // ...
        MetricsRecorder: recorder,
    }
    
    executor := NewCommandExecutor(ctx)
    executor.Execute(parsedCmd)
    
    // Verify metrics were recorded
    assert.Equal(t, 1, recorder.commandExecutionCount)
    assert.Equal(t, "success", recorder.lastStatus)
}
```

#### 5.3.3 Table-Driven Tests

Use table-driven tests for comprehensive coverage:

```go
func TestConfigurationCombinations(t *testing.T) {
    configs := []struct {
        name   string
        config *ExecutionConfig
    }{
        {"CLI default", NewCLIExecutionConfig(false)},
        {"CLI debug", NewCLIExecutionConfig(true)},
        {"Webhook", NewWebhookExecutionConfig()},
    }
    
    commands := []struct {
        name    string
        command *ParsedCommand
    }{
        {"single rebase", &ParsedCommand{Type: SingleCommand, Command: "rebase"}},
        {"single cherrypick", &ParsedCommand{Type: SingleCommand, Command: "cherry-pick"}},
        {"multi command", &ParsedCommand{Type: MultiCommand, CommandLines: []string{"/rebase", "/lgtm"}}},
        {"built-in help", &ParsedCommand{Type: BuiltIn, Command: "help"}},
    }
    
    for _, cfg := range configs {
        for _, cmd := range commands {
            t.Run(fmt.Sprintf("%s/%s", cfg.name, cmd.name), func(t *testing.T) {
                // Test each configuration-command combination
            })
        }
    }
}
```

### 5.4 Mock Implementation

**File**: `pkg/executor/mocks_test.go`

```go
// MockPRHandler implements PRHandlerInterface for testing
type MockPRHandler struct {
    mock.Mock
}

func (m *MockPRHandler) ExecuteCommand(command string, args []string) error {
    args := m.Called(command, args)
    return args.Error(0)
}

func (m *MockPRHandler) ExecuteBuiltInCommand(command string, args []string) error {
    args := m.Called(command, args)
    return args.Error(0)
}

func (m *MockPRHandler) PostComment(body string) error {
    args := m.Called(body)
    return args.Error(0)
}

func (m *MockPRHandler) CheckPRStatus(expectedStatus string) error {
    args := m.Called(expectedStatus)
    return args.Error(0)
}

func (m *MockPRHandler) GetCommentsWithCache() ([]Comment, error) {
    args := m.Called()
    return args.Get(0).([]Comment), args.Error(1)
}

func (m *MockPRHandler) GetConfig() *PRConfig {
    args := m.Called()
    return args.Get(0).(*PRConfig)
}

// MockMetricsRecorder implements MetricsRecorder for testing
type MockMetricsRecorder struct {
    commandExecutionCount int
    durationRecordCount   int
    lastPlatform          string
    lastCommand           string
    lastStatus            string
}

func (m *MockMetricsRecorder) RecordCommandExecution(platform, command, status string) {
    m.commandExecutionCount++
    m.lastPlatform = platform
    m.lastCommand = command
    m.lastStatus = status
}

func (m *MockMetricsRecorder) RecordProcessingDuration(platform, command string, d time.Duration) {
    m.durationRecordCount++
}
```

### 5.5 Test Commands

```bash
# Run all executor tests
go test ./pkg/executor/... -v

# Run with coverage
go test ./pkg/executor/... -cover -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Run specific test
go test ./pkg/executor/... -v -run TestValidator_ValidateSingleCommand

# Run integration tests only
go test ./pkg/executor/... -v -run TestIntegration

# Run with race detector
go test ./pkg/executor/... -race
```

---

## 6. Configuration Examples

### 6.1 CLI Mode Configuration

```go
// cmd/options.go
func (p *PROption) Run(cmd *cobra.Command, args []string) error {
    // ... setup PRHandler ...
    
    ctx := &executor.ExecutionContext{
        PRHandler:       prHandler,
        Logger:          p.Logger,
        Config:          executor.NewCLIExecutionConfig(p.Config.Debug),
        MetricsRecorder: &executor.NoOpMetricsRecorder{},
        Platform:        p.Config.Platform,
        CommentSender:   p.Config.CommentSender,
    }
    
    exec := executor.NewCommandExecutor(ctx)
    result, err := exec.Execute(parsedCmd)
    
    // CLI doesn't return errors for command failures
    // (errors are posted as PR comments)
    return nil
}
```

### 6.2 Webhook Worker Mode Configuration

```go
// pkg/webhook/worker.go
func (w *Worker) processJob(ctx context.Context, job *WebhookJob) {
    // ... setup PRHandler ...
    
    execCtx := &executor.ExecutionContext{
        PRHandler:       prHandler,
        Logger:          logger,
        Config:          executor.NewWebhookExecutionConfig(),
        MetricsRecorder: NewWebhookMetricsRecorder(job.Event.Platform),
        Platform:        job.Event.Platform,
        CommentSender:   job.Event.Sender.Login,
    }
    
    exec := executor.NewCommandExecutor(execCtx)
    result, err := exec.Execute(parsedCmd)
    
    if err != nil {
        logger.Errorf("Command execution failed: %v", err)
    }
}
```

### 6.3 Webhook Sync Mode Configuration

```go
// pkg/webhook/server.go
func (s *Server) processWebhookSync(event *WebhookEvent) error {
    // ... setup PRHandler ...
    
    ctx := &executor.ExecutionContext{
        PRHandler:       prHandler,
        Logger:          s.logger,
        Config:          executor.NewWebhookExecutionConfig(),
        MetricsRecorder: NewWebhookMetricsRecorder(event.Platform),
        Platform:        event.Platform,
        CommentSender:   event.Sender.Login,
    }
    
    exec := executor.NewCommandExecutor(ctx)
    result, err := exec.Execute(parsedCmd)
    
    return err // Return error for HTTP response
}
```

---

## 7. Migration Considerations

### 7.1 Backward Compatibility

- All changes are internal refactoring
- No changes to CLI flags or webhook API
- Existing behavior is preserved
- Tests ensure no regressions

### 7.2 Rollback Strategy

Each phase is independently reversible:
1. **Phase 1-2**: Don't proceed to next phase
2. **Phase 3-4**: Revert commits, uncomment old code
3. **Phase 5-6**: Revert specific commits

### 7.3 Type Consolidation

Remove `handler.SubCommand`, use only `executor.SubCommand`:

```go
// Before (pkg/handler/handle_batch.go)
type SubCommand struct {
    Command string
    Args    []string
}

// After: Remove handler.SubCommand, use executor.SubCommand everywhere
import "github.com/AlaudaDevops/toolbox/pr-cli/pkg/executor"

// Update all usages to executor.SubCommand
```

### 7.4 Error Type Handling

The `handler.CommentedError` type indicates an error comment was already posted:

```go
// pkg/executor/result_handler.go
func (r *ResultHandler) HandleSingleCommandError(command string, err error) error {
    var commentedErr *handler.CommentedError
    if errors.As(err, &commentedErr) {
        // Comment already posted, don't post again
        return nil
    }
    // ... handle other errors
}
```

---

## Appendix A: Existing Code References

### Current Files to Modify

| File | Changes |
|------|---------|
| `cmd/options.go` | Use CommandExecutor |
| `cmd/executor.go` | Remove (consolidated into pkg/executor) |
| `cmd/multi_command.go` | Remove (consolidated into pkg/executor) |
| `cmd/validator.go` | Remove (consolidated into pkg/executor) |
| `pkg/webhook/worker.go` | Use CommandExecutor |
| `pkg/webhook/server.go` | Use CommandExecutor, add multi-command |
| `pkg/handler/handle_batch.go` | Use executor.SubCommand |

### New Files to Create

| File | Purpose |
|------|---------|
| `pkg/executor/config.go` | ExecutionConfig, ExecutionContext |
| `pkg/executor/executor.go` | CommandExecutor |
| `pkg/executor/validator.go` | Validator |
| `pkg/executor/result_handler.go` | ResultHandler |
| `pkg/executor/result.go` | ExecutionResult, SubCommandResult |
| `pkg/executor/metrics.go` | MetricsRecorder interface |
| `pkg/executor/interfaces.go` | PRHandlerInterface for testing |
| `pkg/webhook/metrics_recorder.go` | WebhookMetricsRecorder |
