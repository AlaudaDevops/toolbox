# Command Execution Refactoring - Action Plan

## Overview

This document provides a step-by-step action plan for implementing the command execution refactoring.

## Prerequisites

- [ ] Review `REFACTORING_SUMMARY.md` for executive overview
- [ ] Review `refactoring-command-execution.md` for detailed design
- [ ] Review `refactoring-implementation-guide.md` for code examples
- [ ] Review `refactoring-quick-reference.md` for current vs proposed comparison
- [ ] Team approval to proceed

## Phase 1: Create Core Components (Week 1, Days 1-4)

### Day 1: Setup and Types

**Tasks:**
1. Create `pkg/executor/config.go`
   - [ ] Define `ExecutionConfig` struct
   - [ ] Define `ExecutionContext` struct
   - [ ] Add `NewCLIExecutionConfig()` helper
   - [ ] Add `NewWebhookExecutionConfig()` helper
   - [ ] Write unit tests

2. Create `pkg/executor/result.go`
   - [ ] Define `ExecutionResult` struct
   - [ ] Define `SubCommandResult` struct
   - [ ] Add helper constructors
   - [ ] Write unit tests

3. Create `pkg/executor/metrics.go`
   - [ ] Define `MetricsRecorder` interface
   - [ ] Implement `NoOpMetricsRecorder`
   - [ ] Write unit tests

**Deliverables:**
- 3 new files with full test coverage
- No breaking changes to existing code

**Validation:**
```bash
go test ./pkg/executor/... -v
go test ./pkg/executor/... -cover
```

### Day 2: Validator Component

**Tasks:**
1. Create `pkg/executor/validator.go`
   - [ ] Define `Validator` struct
   - [ ] Implement `NewValidator()`
   - [ ] Implement `ValidateSingleCommand()`
   - [ ] Implement `ValidateMultiCommand()`
   - [ ] Implement `ValidatePRStatus()`
   - [ ] Implement `ValidateCommentSender()`
   - [ ] Implement `ValidateCommentSenderMulti()`
   - [ ] Write comprehensive unit tests with mocks

**Deliverables:**
- Validator with full validation logic
- 80%+ test coverage

**Validation:**
```bash
go test ./pkg/executor/validator_test.go -v
```

### Day 3: Result Handler Component

**Tasks:**
1. Create `pkg/executor/result_handler.go`
   - [ ] Define `ResultHandler` struct
   - [ ] Implement `NewResultHandler()`
   - [ ] Implement `HandleSingleCommandError()`
   - [ ] Implement `HandleMultiCommandResults()`
   - [ ] Implement `FormatSubCommandResult()`
   - [ ] Write comprehensive unit tests

**Deliverables:**
- Result handler with error handling logic
- 80%+ test coverage

**Validation:**
```bash
go test ./pkg/executor/result_handler_test.go -v
```

### Day 4: Command Executor Component

**Tasks:**
1. Create `pkg/executor/executor.go`
   - [ ] Define `CommandExecutor` struct
   - [ ] Implement `NewCommandExecutor()`
   - [ ] Implement `Execute()` (main entry point)
   - [ ] Implement `ExecuteSingleCommand()`
   - [ ] Implement `ExecuteMultiCommand()`
   - [ ] Implement `ExecuteBuiltInCommand()`
   - [ ] Write comprehensive integration tests

**Deliverables:**
- Complete executor implementation
- Integration tests with mock PRHandler
- 80%+ test coverage

**Validation:**
```bash
go test ./pkg/executor/executor_test.go -v
go test ./pkg/executor/... -cover
```

**Milestone:** Phase 1 Complete - All new components created and tested

## Phase 2: Migrate CLI Mode (Week 1-2, Days 5-7)

### Day 5: Update CLI to Use Executor

**Tasks:**
1. Update `cmd/options.go`
   - [ ] Import new executor package
   - [ ] Modify `Run()` to create `ExecutionContext`
   - [ ] Replace old execution calls with `CommandExecutor.Execute()`
   - [ ] Keep old code commented out for rollback

2. Update `cmd/parser.go`
   - [ ] Use `executor.ParseCommand()` directly
   - [ ] Remove any duplicate parsing logic

**Deliverables:**
- CLI using new executor
- Old code preserved as comments

**Validation:**
```bash
# Run existing CLI tests
go test ./cmd/... -v

# Manual testing
./pr-cli --help
./pr-cli --platform github --repo-owner test --repo-name test --pr-num 1 --trigger-comment "/help" --token xxx
```

### Day 6: Remove Old CLI Execution Code

**Tasks:**
1. Remove deprecated files (after validation)
   - [ ] Remove `cmd/executor.go` (old implementation)
   - [ ] Remove `cmd/multi_command.go` (old implementation)
   - [ ] Remove `cmd/validator.go` (old implementation)
   - [ ] Keep only `cmd/options.go` and `cmd/parser.go`

2. Update tests
   - [ ] Update `cmd/options_test.go`
   - [ ] Update `cmd/parser_test.go`
   - [ ] Remove obsolete test files

**Deliverables:**
- Cleaned up cmd package
- All tests passing

**Validation:**
```bash
go test ./cmd/... -v
go test ./... -v  # Full test suite
```

### Day 7: CLI Integration Testing

**Tasks:**
1. Run comprehensive CLI tests
   - [ ] Test single commands
   - [ ] Test multi-commands
   - [ ] Test built-in commands
   - [ ] Test error handling
   - [ ] Test validation (with/without debug mode)

2. Update documentation
   - [ ] Update CLI usage docs if needed
   - [ ] Update code comments

**Deliverables:**
- Fully tested CLI mode
- Updated documentation

**Milestone:** Phase 2 Complete - CLI migrated to new executor

## Phase 3: Migrate Worker Mode (Week 2, Days 8-10)

### Day 8: Create Webhook Metrics Recorder

**Tasks:**
1. Create `pkg/webhook/metrics_recorder.go`
   - [ ] Define `WebhookMetricsRecorder` struct
   - [ ] Implement `MetricsRecorder` interface
   - [ ] Use existing Prometheus metrics
   - [ ] Write unit tests

**Deliverables:**
- Webhook metrics recorder implementation

**Validation:**
```bash
go test ./pkg/webhook/metrics_recorder_test.go -v
```

### Day 9: Update Worker to Use Executor

**Tasks:**
1. Update `pkg/webhook/worker.go`
   - [ ] Import executor package
   - [ ] Modify `processJob()` to create `ExecutionContext`
   - [ ] Replace old execution methods with `CommandExecutor.Execute()`
   - [ ] Keep old code commented for rollback

2. Remove old worker execution methods
   - [ ] Remove `executeSingleCommand()`
   - [ ] Remove `executeMultiCommand()`
   - [ ] Remove `processSubCommand()`

**Deliverables:**
- Worker using new executor
- Metrics still working

**Validation:**
```bash
go test ./pkg/webhook/... -v
```

### Day 10: Worker Integration Testing

**Tasks:**
1. Test worker mode
   - [ ] Test async job processing
   - [ ] Test metrics recording
   - [ ] Test error handling
   - [ ] Test multi-command support

2. Load testing (optional)
   - [ ] Verify no performance regression
   - [ ] Check metrics accuracy

**Deliverables:**
- Fully tested worker mode

**Milestone:** Phase 3 Complete - Worker migrated to new executor

## Phase 4: Migrate Sync Mode (Week 2, Days 11-12)

### Day 11: Update Sync Mode

**Tasks:**
1. Update `pkg/webhook/server.go`
   - [ ] Modify `processWebhookSync()` to create `ExecutionContext`
   - [ ] Replace old execution with `CommandExecutor.Execute()`
   - [ ] **Implement multi-command support** (was TODO)
   - [ ] Use `WebhookMetricsRecorder`

**Deliverables:**
- Sync mode using new executor
- Multi-command support added

**Validation:**
```bash
go test ./pkg/webhook/... -v
```

### Day 12: Sync Mode Integration Testing

**Tasks:**
1. Test sync mode
   - [ ] Test single commands
   - [ ] Test multi-commands (NEW!)
   - [ ] Test built-in commands
   - [ ] Test error handling
   - [ ] Test metrics

**Deliverables:**
- Fully tested sync mode with multi-command support

**Milestone:** Phase 4 Complete - Sync mode migrated with feature parity

## Phase 5: Remove Type Duplication (Week 3, Days 13-14)

### Day 13: Update Handler Package

**Tasks:**
1. Update `pkg/handler/handle_batch.go`
   - [ ] Replace `handler.SubCommand` with `executor.SubCommand`
   - [ ] Update all methods using SubCommand
   - [ ] Update tests

2. Remove conversion logic
   - [ ] Search for SubCommand conversions
   - [ ] Remove unnecessary conversions
   - [ ] Update imports

**Deliverables:**
- Single SubCommand type used everywhere

**Validation:**
```bash
go test ./pkg/handler/... -v
```

### Day 14: Final Type Cleanup

**Tasks:**
1. Verify no type duplication remains
   - [ ] Search codebase for `handler.SubCommand`
   - [ ] Ensure all use `executor.SubCommand`
   - [ ] Update all tests

**Deliverables:**
- No type duplication

**Milestone:** Phase 5 Complete - Type duplication removed

## Phase 6: Cleanup and Documentation (Week 3, Days 15-16)

### Day 15: Code Cleanup

**Tasks:**
1. Remove all deprecated code
   - [ ] Remove commented-out old code
   - [ ] Remove unused imports
   - [ ] Run linters

2. Code review
   - [ ] Self-review all changes
   - [ ] Check for TODO comments
   - [ ] Verify error messages are clear

**Deliverables:**
- Clean, production-ready code

**Validation:**
```bash
go vet ./...
golangci-lint run
```

### Day 16: Documentation and Final Testing

**Tasks:**
1. Update documentation
   - [ ] Update README if needed
   - [ ] Update architecture docs
   - [ ] Add migration notes to CHANGELOG

2. Final integration tests
   - [ ] Run full test suite
   - [ ] Test all three modes end-to-end
   - [ ] Verify metrics
   - [ ] Check logs

3. Create PR
   - [ ] Write comprehensive PR description
   - [ ] Link to design docs
   - [ ] Request reviews

**Deliverables:**
- Complete documentation
- PR ready for review

**Validation:**
```bash
go test ./... -v -race
go test ./... -cover
```

**Milestone:** Phase 6 Complete - Refactoring complete!

## Success Criteria Checklist

- [ ] All three modes use unified executor
- [ ] Multi-command support in all modes (including sync)
- [ ] Code reduction of 50%+ achieved
- [ ] All existing tests pass
- [ ] New tests cover 80%+ of new code
- [ ] No performance regression
- [ ] Documentation updated
- [ ] PR approved and merged

## Rollback Plan

If issues are discovered at any phase:

1. **Phase 1-2**: Simply don't proceed to next phase
2. **Phase 3-4**: Revert commits, uncomment old code
3. **Phase 5-6**: Revert specific commits

Each phase is designed to be independently reversible.

## Communication Plan

- **Daily**: Update progress in team chat
- **Weekly**: Demo progress in team meeting
- **Blockers**: Immediately communicate any blockers
- **Completion**: Present final results to team

## Resources

- Design docs in `pr-cli/docs/refactoring-*.md`
- Existing tests in `*_test.go` files
- Git history for reference implementations

## Notes

- Keep old code commented until phase is validated
- Write tests before removing old code
- Run full test suite after each phase
- Document any deviations from plan

