# Unified Command Executor - Implementation Tasks

## Overview

This document contains the implementation tasks for the Unified Command Executor refactoring. Tasks are organized in phases, with each task referencing specific sections in the design document.

**Design Document**: `DESIGN-unified-command-executor.md`

**Implementation Order**:
1. Phase 1: Create the new engine (pkg/executor)
2. Phase 2: Integrate engine in webhook service
3. Phase 3: Add comprehensive automated tests
4. Phase 4: Refactor CLI to use the new engine
5. Phase 5: Cleanup and finalization

---

## Phase 1: Create Core Engine Components

**Goal**: Build the unified command executor engine without breaking existing functionality.

**Reference**: Design Section 4 (Lines 136-298)

### Task 1.1: Create Configuration Types

**File**: `pkg/executor/config.go`

**Design Reference**: Section 4.1-4.2 (Lines 140-175)

- [ ] Define `ExecutionConfig` struct with all configuration fields:
  - `ValidateCommentSender bool`
  - `ValidatePRStatus bool`
  - `DebugMode bool`
  - `PostErrorsAsPRComments bool`
  - `ReturnErrors bool`
  - `StopOnFirstError bool`
- [ ] Define `ExecutionContext` struct:
  - `PRHandler *handler.PRHandler`
  - `Logger logrus.FieldLogger`
  - `Config *ExecutionConfig`
  - `MetricsRecorder MetricsRecorder`
  - `Platform string`
  - `CommentSender string`
- [ ] Implement `NewCLIExecutionConfig(debugMode bool) *ExecutionConfig`
- [ ] Implement `NewWebhookExecutionConfig() *ExecutionConfig`
- [ ] Add godoc comments for all exported types and functions

**Validation**:
```bash
go build ./pkg/executor/...
```

---

### Task 1.2: Create Result Types

**File**: `pkg/executor/result.go`

**Design Reference**: Section 4.3 (Lines 177-202)

- [ ] Define `ExecutionResult` struct:
  - `Success bool`
  - `Error error`
  - `CommandType CommandType`
  - `Results []SubCommandResult`
- [ ] Define `SubCommandResult` struct:
  - `Command string`
  - `Args []string`
  - `Success bool`
  - `Error error`
- [ ] Implement `NewSuccessResult(cmdType CommandType) *ExecutionResult`
- [ ] Implement `NewErrorResult(cmdType CommandType, err error) *ExecutionResult`
- [ ] Implement `NewMultiCommandResult(results []SubCommandResult) *ExecutionResult`

**Validation**:
```bash
go build ./pkg/executor/...
```

---

### Task 1.3: Create Metrics Interface

**File**: `pkg/executor/metrics.go`

**Design Reference**: Section 4.7 (Lines 275-298)

- [ ] Define `MetricsRecorder` interface:
  - `RecordCommandExecution(platform, command, status string)`
  - `RecordProcessingDuration(platform, command string, duration time.Duration)`
- [ ] Implement `NoOpMetricsRecorder` struct (empty implementation)
- [ ] Add godoc comments

**Validation**:
```bash
go build ./pkg/executor/...
```

---

### Task 1.4: Create Interfaces for Testing

**File**: `pkg/executor/interfaces.go`

**Design Reference**: Section 5.2 (Lines 325-355)

- [ ] Define `PRHandlerInterface` interface (subset of PRHandler methods needed):
  - `ExecuteCommand(command string, args []string) error`
  - `ExecuteBuiltInCommand(command string, args []string) error`
  - `PostComment(body string) error`
  - `CheckPRStatus(expectedStatus string) error`
  - `GetCommentsWithCache() ([]Comment, error)`
  - `GetConfig() *PRConfig`
- [ ] Define `Comment` struct for interface
- [ ] Define `PRConfig` struct for interface

**Note**: These interfaces enable testing without real PRHandler dependencies.

**Validation**:
```bash
go build ./pkg/executor/...
```

---

### Task 1.5: Create Validator Component

**File**: `pkg/executor/validator.go`

**Design Reference**: Section 4.5 (Lines 224-258)

- [ ] Define `Validator` struct with `context *ExecutionContext`
- [ ] Implement `NewValidator(ctx *ExecutionContext) *Validator`
- [ ] Implement `ValidateSingleCommand(command string) error`:
  - Check PR status if `config.ValidatePRStatus` is true
  - Skip PR check for cherry-pick and built-in commands
  - Validate comment sender if `config.ValidateCommentSender` is true
  - Skip comment sender validation if `config.DebugMode` is true
- [ ] Implement `ValidateMultiCommand(subCommands []SubCommand, rawCommandLines []string) error`:
  - Check if any command needs PR status validation
  - Validate comment sender for all commands if enabled
- [ ] Implement `shouldSkipPRStatusCheck(command string) bool`:
  - Return true for: "cherry-pick", "cherrypick", and built-in commands
- [ ] Implement private `validateCommentSender() error`
- [ ] Implement private `validateCommentSenderMulti(rawCommandLines []string) error`

**Validation**:
```bash
go build ./pkg/executor/...
```

---

### Task 1.6: Create Result Handler Component

**File**: `pkg/executor/result_handler.go`

**Design Reference**: Section 4.6 (Lines 260-273)

- [ ] Define `ResultHandler` struct with `context *ExecutionContext`
- [ ] Implement `NewResultHandler(ctx *ExecutionContext) *ResultHandler`
- [ ] Implement `HandleSingleCommandError(command string, err error) error`:
  - Check for `handler.CommentedError` (skip if already posted)
  - Post error as PR comment if `config.PostErrorsAsPRComments` is true
  - Return error if `config.ReturnErrors` is true
  - Log error otherwise
- [ ] Implement `HandleMultiCommandResults(results []SubCommandResult) error`:
  - Format all results using `FormatSubCommandResult`
  - Create summary header (include warning if any failures)
  - Post summary as PR comment
- [ ] Implement `FormatSubCommandResult(result SubCommandResult) string`:
  - Success: `✅ Command \`/command args\` executed successfully`
  - Failure: `❌ Command \`/command args\` failed: error message`

**Validation**:
```bash
go build ./pkg/executor/...
```

---

### Task 1.7: Create Command Executor Component

**File**: `pkg/executor/executor.go`

**Design Reference**: Section 4.4 (Lines 204-222), Section 3.2 (Lines 92-128)

- [ ] Define `CommandExecutor` struct:
  - `context *ExecutionContext`
  - `validator *Validator`
  - `resultHandler *ResultHandler`
- [ ] Implement `NewCommandExecutor(ctx *ExecutionContext) *CommandExecutor`:
  - Initialize validator and result handler
- [ ] Implement `Execute(parsedCmd *ParsedCommand) (*ExecutionResult, error)`:
  - Route to appropriate method based on `parsedCmd.Type`
  - Handle SingleCommand, MultiCommand, BuiltIn
- [ ] Implement `ExecuteSingleCommand(command string, args []string) (*ExecutionResult, error)`:
  - Start time tracking
  - Call validator.ValidateSingleCommand
  - Execute via PRHandler.ExecuteCommand
  - Handle errors via resultHandler.HandleSingleCommandError
  - Record metrics
  - Return result
- [ ] Implement `ExecuteMultiCommand(commandLines, rawCommandLines []string) (*ExecutionResult, error)`:
  - Start time tracking
  - Parse command lines via ParseMultiCommandLines
  - Call validator.ValidateMultiCommand
  - Execute each sub-command
  - Collect results (continue on error unless StopOnFirstError)
  - Handle results via resultHandler.HandleMultiCommandResults
  - Record metrics for each sub-command
  - Return aggregate result
- [ ] Implement `ExecuteBuiltInCommand(command string, args []string) (*ExecutionResult, error)`:
  - Execute via PRHandler.ExecuteBuiltInCommand
  - Record metrics
  - Return result

**Validation**:
```bash
go build ./pkg/executor/...
```

---

### Task 1.8: Create Unit Tests for Validator

**File**: `pkg/executor/validator_test.go`

**Design Reference**: Section 5.3.1 (Lines 358-440)

- [ ] Create mock PRHandler implementing PRHandlerInterface
- [ ] Implement `TestValidator_ValidateSingleCommand`:
  - Test valid command with open PR
  - Test cherry-pick allows closed PR
  - Test command fails on closed PR
  - Test comment sender validation passes
  - Test comment sender validation fails
  - Test debug mode skips comment sender validation
- [ ] Implement `TestValidator_ValidateMultiCommand`:
  - Test all commands valid
  - Test partial validation failure
  - Test skip PR check when all commands are cherry-pick
- [ ] Implement `TestValidator_shouldSkipPRStatusCheck`:
  - Test cherry-pick, cherrypick, help, version return true
  - Test rebase, merge, lgtm return false

**Validation**:
```bash
go test ./pkg/executor/... -v -run TestValidator
```

---

### Task 1.9: Create Unit Tests for Result Handler

**File**: `pkg/executor/result_handler_test.go`

**Design Reference**: Section 5.3.1 (Lines 442-490)

- [ ] Implement `TestResultHandler_HandleSingleCommandError`:
  - Test posts error as PR comment
  - Test returns error when configured
  - Test CommentedError skips posting
  - Test logs error when neither post nor return
- [ ] Implement `TestResultHandler_HandleMultiCommandResults`:
  - Test all successful commands
  - Test partial failures
  - Test all failed commands
  - Test empty results
- [ ] Implement `TestResultHandler_FormatSubCommandResult`:
  - Test success with no args
  - Test success with args
  - Test failure with error message

**Validation**:
```bash
go test ./pkg/executor/... -v -run TestResultHandler
```

---

### Task 1.10: Create Unit Tests for Command Executor

**File**: `pkg/executor/executor_test.go`

**Design Reference**: Section 5.3.1 (Lines 492-560)

- [ ] Implement `TestCommandExecutor_Execute`:
  - Test single command success
  - Test single command failure
  - Test multi command success
  - Test multi command partial failure
  - Test built-in command
- [ ] Implement `TestCommandExecutor_ExecuteSingleCommand`:
  - Test successful execution
  - Test validation failure
  - Test execution failure
  - Test metrics recording
- [ ] Implement `TestCommandExecutor_ExecuteMultiCommand`:
  - Test all commands succeed
  - Test continue on error
  - Test stop on first error
  - Test summary posting
- [ ] Implement `TestCommandExecutor_StopOnFirstError`:
  - Test execution stops after first failure
  - Test remaining commands not executed

**Validation**:
```bash
go test ./pkg/executor/... -v -run TestCommandExecutor
go test ./pkg/executor/... -cover
```

---

### Task 1.11: Create Mock Implementations

**File**: `pkg/executor/mocks_test.go`

**Design Reference**: Section 5.4 (Lines 634-690)

- [ ] Implement `MockPRHandler` struct:
  - Use testify/mock or manual implementation
  - Implement all PRHandlerInterface methods
  - Support method call expectations
- [ ] Implement `MockMetricsRecorder` struct:
  - Track call counts
  - Track last recorded values
  - Support verification in tests

**Validation**:
```bash
go test ./pkg/executor/... -v
```

---

## Phase 2: Integrate Engine in Webhook Service

**Goal**: Update webhook worker and sync modes to use the new engine.

**Reference**: Design Section 6.2-6.3 (Lines 710-763)

### Task 2.1: Create Webhook Metrics Recorder

**File**: `pkg/webhook/metrics_recorder.go`

**Design Reference**: Section 4.7 (Lines 287-298)

- [ ] Define `WebhookMetricsRecorder` struct with `platform string`
- [ ] Implement `RecordCommandExecution(platform, command, status string)`:
  - Use existing `CommandExecutionTotal.WithLabelValues(...).Inc()`
- [ ] Implement `RecordProcessingDuration(platform, command string, d time.Duration)`:
  - Use existing `WebhookProcessingDuration.WithLabelValues(...).Observe(...)`
- [ ] Implement `NewWebhookMetricsRecorder(platform string) *WebhookMetricsRecorder`

**Validation**:
```bash
go build ./pkg/webhook/...
```

---

### Task 2.2: Update Worker to Use Command Executor

**File**: `pkg/webhook/worker.go`

**Design Reference**: Section 6.2 (Lines 716-746)

- [ ] Import `executor` package
- [ ] In `processJob()`:
  - [ ] Create `ExecutionContext` with:
    - PRHandler from existing code
    - Logger from existing code
    - `executor.NewWebhookExecutionConfig()`
    - `NewWebhookMetricsRecorder(platform)`
    - Platform from job
    - CommentSender from job event
  - [ ] Create `CommandExecutor` via `executor.NewCommandExecutor(ctx)`
  - [ ] Replace execution logic with `exec.Execute(parsedCmd)`
- [ ] **Keep old methods commented out** for rollback (do not delete yet)
- [ ] Remove duplicate execution methods after validation:
  - `executeSingleCommand()`
  - `executeMultiCommand()`
  - `processSubCommand()`

**Validation**:
```bash
go test ./pkg/webhook/... -v
go build ./...
```

---

### Task 2.3: Update Sync Mode to Use Command Executor

**File**: `pkg/webhook/server.go`

**Design Reference**: Section 6.3 (Lines 748-763)

- [ ] Import `executor` package
- [ ] In `processWebhookSync()`:
  - [ ] Create `ExecutionContext` with:
    - PRHandler from existing code
    - Logger from existing code
    - `executor.NewWebhookExecutionConfig()`
    - `NewWebhookMetricsRecorder(platform)`
    - Platform from event
    - CommentSender from event
  - [ ] Create `CommandExecutor`
  - [ ] Replace `switch parsedCmd.Type` with `exec.Execute(parsedCmd)`
  - [ ] **Multi-command now supported** (removes TODO)
- [ ] Return error from execution

**Validation**:
```bash
go test ./pkg/webhook/... -v
go build ./...
```

---

### Task 2.4: Update Webhook Tests

**Files**: `pkg/webhook/worker_test.go`, `pkg/webhook/server_test.go`

- [ ] Update worker tests to expect new execution flow
- [ ] Add tests for webhook metrics recorder
- [ ] Add tests for sync mode multi-command support
- [ ] Ensure all existing tests pass

**Validation**:
```bash
go test ./pkg/webhook/... -v -cover
```

---

## Phase 3: Comprehensive Automated Testing

**Goal**: Ensure complete test coverage and integration tests.

**Reference**: Design Section 5 (Lines 300-690)

### Task 3.1: Create Integration Tests

**File**: `pkg/executor/integration_test.go`

**Design Reference**: Section 5.3.2 (Lines 562-632)

- [ ] Implement `TestIntegration_CLIModeExecution`:
  - Full CLI mode flow with mock PRHandler
  - Test single command with validation
  - Test multi-command with summary posting
  - Test error handling with PR comment
  - Test debug mode behavior
- [ ] Implement `TestIntegration_WebhookModeExecution`:
  - Full webhook mode flow with mock PRHandler
  - Test single command without validation
  - Test multi-command with metrics
  - Test error handling without PR comment
- [ ] Implement `TestIntegration_MetricsRecording`:
  - Verify metrics recorded correctly for success
  - Verify metrics recorded correctly for failure
  - Verify duration recording

**Validation**:
```bash
go test ./pkg/executor/... -v -run TestIntegration
```

---

### Task 3.2: Create Configuration Combination Tests

**File**: `pkg/executor/config_test.go`

**Design Reference**: Section 5.3.3 (Lines 634-660)

- [ ] Implement `TestConfigurationCombinations`:
  - Test all config combinations (CLI default, CLI debug, Webhook)
  - Test all command types (single, multi, built-in)
  - Use table-driven tests
- [ ] Implement `TestNewCLIExecutionConfig`:
  - Test with debugMode=false
  - Test with debugMode=true
- [ ] Implement `TestNewWebhookExecutionConfig`:
  - Verify all webhook defaults

**Validation**:
```bash
go test ./pkg/executor/... -v -run TestConfiguration
```

---

### Task 3.3: Achieve Coverage Targets

**Reference**: Section 5.1 (Lines 308-315)

- [ ] Run coverage report: `go test ./pkg/executor/... -cover -coverprofile=coverage.out`
- [ ] Verify coverage targets:
  - CommandExecutor: 90%+
  - Validator: 95%+
  - ResultHandler: 90%+
  - Config: 100%
- [ ] Add missing tests if below targets
- [ ] Generate HTML coverage report: `go tool cover -html=coverage.out`

**Validation**:
```bash
go test ./pkg/executor/... -cover
```

---

### Task 3.4: Add Race Detection Tests

- [ ] Run all tests with race detector: `go test ./pkg/executor/... -race`
- [ ] Run all tests with race detector: `go test ./pkg/webhook/... -race`
- [ ] Fix any race conditions discovered

**Validation**:
```bash
go test ./... -race
```

---

## Phase 4: Refactor CLI to Use New Engine

**Goal**: Update CLI mode to use the unified command executor.

**Reference**: Design Section 6.1 (Lines 700-714)

### Task 4.1: Update CLI Options

**File**: `cmd/options.go`

**Design Reference**: Section 6.1 (Lines 700-714)

- [ ] Import `executor` package
- [ ] In `Run()` method:
  - [ ] Create `ExecutionContext` with:
    - PRHandler from existing setup
    - Logger (p.Logger)
    - `executor.NewCLIExecutionConfig(p.Config.Debug)`
    - `&executor.NoOpMetricsRecorder{}`
    - Platform (p.Config.Platform)
    - CommentSender (p.Config.CommentSender)
  - [ ] Create `CommandExecutor`
  - [ ] Replace execution logic with `exec.Execute(parsedCmd)`
- [ ] Update error handling (CLI returns nil, errors posted to PR)
- [ ] **Keep old code commented out** for rollback

**Validation**:
```bash
go test ./cmd/... -v
go build ./...
```

---

### Task 4.2: Update CLI Parser

**File**: `cmd/parser.go`

- [ ] Ensure `executor.ParseCommand()` is used consistently
- [ ] Remove any duplicate parsing logic
- [ ] Update imports if needed

**Validation**:
```bash
go test ./cmd/... -v
```

---

### Task 4.3: Update CLI Tests

**Files**: `cmd/options_test.go`, `cmd/parser_command_args_test.go`

- [ ] Update tests to expect new execution flow
- [ ] Ensure all existing behavior is preserved
- [ ] Add tests for debug mode with new executor

**Validation**:
```bash
go test ./cmd/... -v -cover
```

---

### Task 4.4: Manual CLI Testing

- [ ] Test single command: `./pr-cli --platform github --repo-owner test --repo-name test --pr-num 1 --trigger-comment "/rebase" --token xxx`
- [ ] Test multi-command: `./pr-cli --platform github --repo-owner test --repo-name test --pr-num 1 --trigger-comment "/rebase\n/lgtm" --token xxx`
- [ ] Test built-in command: `./pr-cli --platform github --repo-owner test --repo-name test --pr-num 1 --trigger-comment "/help" --token xxx`
- [ ] Test debug mode: `./pr-cli --debug --platform github ...`
- [ ] Test error handling: Trigger a command that will fail

**Validation**: Manual verification of expected behavior

---

## Phase 5: Cleanup and Finalization

**Goal**: Remove deprecated code and finalize documentation.

### Task 5.1: Remove Type Duplication

**File**: `pkg/handler/handle_batch.go`

**Design Reference**: Section 7.3 (Lines 785-800)

- [ ] Replace `handler.SubCommand` with `executor.SubCommand`
- [ ] Add import for executor package
- [ ] Update all methods using SubCommand
- [ ] Search codebase for remaining `handler.SubCommand` usages
- [ ] Remove `handler.SubCommand` definition

**Validation**:
```bash
go build ./...
go test ./pkg/handler/... -v
```

---

### Task 5.2: Remove Deprecated CLI Code

**Files**: `cmd/executor.go`, `cmd/multi_command.go`, `cmd/validator.go`

- [ ] Verify all tests pass with new implementation
- [ ] Remove or delete `cmd/executor.go` (execution logic moved to pkg/executor)
- [ ] Remove or delete `cmd/multi_command.go` (logic moved to pkg/executor)
- [ ] Remove or delete `cmd/validator.go` (logic moved to pkg/executor)
- [ ] Update imports in remaining cmd files

**Validation**:
```bash
go build ./...
go test ./cmd/... -v
go test ./... -v
```

---

### Task 5.3: Remove Deprecated Webhook Code

**File**: `pkg/webhook/worker.go`

- [ ] Remove commented-out old code from Task 2.2
- [ ] Remove unused imports
- [ ] Run linter: `golangci-lint run ./pkg/webhook/...`

**Validation**:
```bash
go build ./...
go test ./pkg/webhook/... -v
```

---

### Task 5.4: Code Quality Checks

- [ ] Run linter: `golangci-lint run ./...`
- [ ] Run vet: `go vet ./...`
- [ ] Fix any issues discovered
- [ ] Verify all tests pass: `go test ./... -v`
- [ ] Verify coverage maintained: `go test ./... -cover`

**Validation**:
```bash
golangci-lint run ./...
go vet ./...
go test ./... -v -cover
```

---

### Task 5.5: Update Documentation

- [ ] Update `README.md` if any user-facing changes
- [ ] Archive old refactoring documents (move to `docs/archive/`)
- [ ] Update code comments where needed
- [ ] Add CHANGELOG entry for this refactoring

**Files to archive**:
- `docs/refactoring-action-plan.md`
- `docs/refactoring-implementation-guide.md`
- `docs/refactoring-command-execution.md`
- `docs/REFACTORING_SUMMARY.md`
- `docs/refactoring-quick-reference.md`

---

### Task 5.6: Final Integration Testing

- [ ] Run full test suite: `go test ./... -v -race`
- [ ] Test CLI mode end-to-end (if possible)
- [ ] Test webhook worker mode (if possible)
- [ ] Test webhook sync mode (if possible)
- [ ] Verify metrics recording in webhook modes

**Validation**:
```bash
go test ./... -v -race
go test ./... -cover
```

---

## Success Criteria

- [ ] All three modes use unified executor
- [ ] Multi-command support in all modes (including sync)
- [ ] Code reduction achieved (~50%+ in execution code)
- [ ] All existing tests pass
- [ ] New tests cover 90%+ of new code
- [ ] No race conditions
- [ ] Linter passes
- [ ] Documentation updated

---

## Appendix: Quick Reference

### Test Commands

```bash
# All tests
go test ./... -v

# Specific package
go test ./pkg/executor/... -v

# With coverage
go test ./pkg/executor/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out

# With race detection
go test ./... -race

# Specific test
go test ./pkg/executor/... -v -run TestValidator_ValidateSingleCommand

# Build check
go build ./...

# Lint
golangci-lint run ./...
```

### Key Files

| New File | Purpose |
|----------|---------|
| `pkg/executor/config.go` | Configuration types |
| `pkg/executor/executor.go` | Main executor |
| `pkg/executor/validator.go` | Validation logic |
| `pkg/executor/result_handler.go` | Result formatting |
| `pkg/executor/result.go` | Result types |
| `pkg/executor/metrics.go` | Metrics interface |
| `pkg/webhook/metrics_recorder.go` | Webhook metrics |

### Design Document Line References

| Topic | Lines |
|-------|-------|
| Architecture Overview | 64-98 |
| Component Flow | 92-128 |
| ExecutionConfig | 140-157 |
| ExecutionContext | 159-175 |
| ExecutionResult | 177-202 |
| CommandExecutor | 204-222 |
| Validator | 224-258 |
| ResultHandler | 260-273 |
| MetricsRecorder | 275-298 |
| Testing Strategy | 300-320 |
| Mock Interfaces | 325-355 |
| Unit Tests | 358-560 |
| Integration Tests | 562-632 |
| Configuration Examples | 692-763 |
| Migration Considerations | 765-810 |
