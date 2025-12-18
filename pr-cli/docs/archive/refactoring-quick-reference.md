# Command Execution Refactoring - Quick Reference

## Current vs Proposed: Side-by-Side Comparison

### Single Command Execution

#### Current (CLI Mode)
```go
// cmd/executor.go
func (p *PROption) executeSingleCommand(prHandler *handler.PRHandler, command string, cmdArgs []string) error {
    // Check PR status
    if !p.shouldSkipPRStatusCheck(command) {
        if err := prHandler.CheckPRStatus("open"); err != nil {
            return fmt.Errorf("PR status check failed: %w", err)
        }
    }
    
    // Validate comment sender
    if !p.Config.Debug {
        if err := p.validateCommentSender(prHandler); err != nil {
            return fmt.Errorf("comment sender validation failed: %w", err)
        }
    }
    
    // Execute command
    err := prHandler.ExecuteCommand(command, cmdArgs)
    
    // Handle error
    if err != nil {
        return p.handleCommandError(prHandler, command, err)
    }
    
    return nil
}
```

#### Current (Worker Mode)
```go
// pkg/webhook/worker.go
func (w *Worker) executeSingleCommand(logger *logrus.Entry, prHandler *handler.PRHandler, platform string, parsedCmd *executor.ParsedCommand, startTime time.Time) {
    command := parsedCmd.Command
    args := parsedCmd.Args
    
    // Execute command (NO VALIDATION)
    if err := prHandler.ExecuteCommand(command, args); err != nil {
        logger.Errorf("Failed to execute command: %v", err)
        CommandExecutionTotal.WithLabelValues(platform, command, "error").Inc()
        WebhookProcessingDuration.WithLabelValues(platform, command).Observe(time.Since(startTime).Seconds())
        return
    }
    
    CommandExecutionTotal.WithLabelValues(platform, command, "success").Inc()
    WebhookProcessingDuration.WithLabelValues(platform, command).Observe(time.Since(startTime).Seconds())
}
```

#### Proposed (All Modes)
```go
// pkg/executor/executor.go
func (e *CommandExecutor) ExecuteSingleCommand(command string, args []string) (*ExecutionResult, error) {
    startTime := time.Now()
    
    // Validate (configurable)
    if err := e.validator.ValidateSingleCommand(command); err != nil {
        return NewErrorResult(SingleCommand, err), err
    }
    
    // Execute
    err := e.context.PRHandler.ExecuteCommand(command, args)
    
    // Handle error (configurable)
    if err != nil {
        if handleErr := e.resultHandler.HandleSingleCommandError(command, err); handleErr != nil {
            return NewErrorResult(SingleCommand, handleErr), handleErr
        }
    }
    
    // Record metrics (optional)
    status := "success"
    if err != nil {
        status = "error"
    }
    e.context.MetricsRecorder.RecordCommandExecution(e.context.Platform, command, status)
    e.context.MetricsRecorder.RecordProcessingDuration(e.context.Platform, command, time.Since(startTime))
    
    return NewSuccessResult(SingleCommand), nil
}
```

### Multi-Command Execution

#### Current (CLI Mode)
```go
// cmd/multi_command.go
func (p *PROption) executeMultiCommand(prHandler *handler.PRHandler, commandLines []string, rawCommandLines []string) error {
    // Parse
    subCommands, err := p.parseMultiCommandLines(commandLines)
    if err != nil {
        return err
    }
    
    // Validate
    if err := p.validateMultiCommandExecution(prHandler, subCommands, rawCommandLines); err != nil {
        return err
    }
    
    // Execute
    return p.handleMultiCommandExecution(prHandler, subCommands)
}

func (p *PROption) handleMultiCommandExecution(prHandler *handler.PRHandler, subCommands []handler.SubCommand) error {
    var results []string
    var hasErrors bool
    
    for _, subCmd := range subCommands {
        result := p.processMultiCommand(prHandler, subCmd)
        results = append(results, result)
        if strings.HasPrefix(result, "❌") {
            hasErrors = true
        }
    }
    
    header := "**Multi-Command Execution Results:**"
    if hasErrors {
        header = fmt.Sprintf("%s (⚠️ Some commands failed)", header)
    }
    
    summary := fmt.Sprintf("%s\n\n%s", header, strings.Join(results, "\n"))
    return prHandler.PostComment(summary)
}
```

#### Current (Worker Mode)
```go
// pkg/webhook/worker.go
func (w *Worker) executeMultiCommand(logger *logrus.Entry, prHandler *handler.PRHandler, platform string, parsedCmd *executor.ParsedCommand, startTime time.Time) {
    // Parse
    subCommands, err := executor.ParseMultiCommandLines(parsedCmd.CommandLines)
    if err != nil {
        logger.Errorf("Failed to parse multi-command lines: %v", err)
        CommandExecutionTotal.WithLabelValues(platform, "multi", "error").Inc()
        return
    }
    
    // Execute (NO VALIDATION)
    var results []string
    var hasErrors bool
    
    for _, subCmd := range subCommands {
        result := w.processSubCommand(logger, prHandler, platform, subCmd)
        results = append(results, result)
        if strings.HasPrefix(result, "❌") {
            hasErrors = true
        }
    }
    
    // Post summary
    header := "**Multi-Command Execution Results:**"
    if hasErrors {
        header = fmt.Sprintf("%s (⚠️ Some commands failed)", header)
    }
    
    summary := fmt.Sprintf("%s\n\n%s", header, strings.Join(results, "\n"))
    if err := prHandler.PostComment(summary); err != nil {
        logger.Errorf("Failed to post multi-command summary: %v", err)
    }
    
    // Metrics
    status := "success"
    if hasErrors {
        status = "partial_error"
    }
    CommandExecutionTotal.WithLabelValues(platform, "multi", status).Inc()
    WebhookProcessingDuration.WithLabelValues(platform, "multi").Observe(time.Since(startTime).Seconds())
}
```

#### Current (Sync Mode)
```go
// pkg/webhook/server.go
case executor.MultiCommand:
    // TODO: Implement multicommand executor
    // NOT IMPLEMENTED!
```

#### Proposed (All Modes)
```go
// pkg/executor/executor.go
func (e *CommandExecutor) ExecuteMultiCommand(commandLines, rawCommandLines []string) (*ExecutionResult, error) {
    startTime := time.Now()
    
    // Parse
    subCommands, err := ParseMultiCommandLines(commandLines)
    if err != nil {
        return NewErrorResult(MultiCommand, err), err
    }
    
    // Validate (configurable)
    if err := e.validator.ValidateMultiCommand(subCommands, rawCommandLines); err != nil {
        return NewErrorResult(MultiCommand, err), err
    }
    
    // Execute each sub-command
    var results []SubCommandResult
    for _, subCmd := range subCommands {
        err := e.context.PRHandler.ExecuteCommand(subCmd.Command, subCmd.Args)
        
        result := SubCommandResult{
            Command: subCmd.Command,
            Args:    subCmd.Args,
            Success: err == nil,
            Error:   err,
        }
        results = append(results, result)
        
        // Record metrics
        status := "success"
        if err != nil {
            status = "error"
        }
        e.context.MetricsRecorder.RecordCommandExecution(e.context.Platform, subCmd.Command, status)
        
        // Stop on first error if configured
        if err != nil && e.context.Config.StopOnFirstError {
            break
        }
    }
    
    // Handle results (configurable)
    if err := e.resultHandler.HandleMultiCommandResults(results); err != nil {
        return NewErrorResult(MultiCommand, err), err
    }
    
    e.context.MetricsRecorder.RecordProcessingDuration(e.context.Platform, "multi", time.Since(startTime))
    
    return NewMultiCommandResult(results), nil
}
```

## Usage Examples

### CLI Mode Usage
```go
// cmd/options.go
func (p *PROption) Run(cmd *cobra.Command, args []string) error {
    // Parse command
    parsedCmd, err := executor.ParseCommand(p.Config.TriggerComment)
    if err != nil {
        return err
    }
    
    // Create execution context
    ctx := &executor.ExecutionContext{
        PRHandler:       prHandler,
        Logger:          p.Logger,
        Config:          executor.NewCLIExecutionConfig(p.Config.Debug),
        MetricsRecorder: &executor.NoOpMetricsRecorder{},
        Platform:        p.Config.Platform,
        CommentSender:   p.Config.CommentSender,
    }
    
    // Create executor
    exec := executor.NewCommandExecutor(ctx)
    
    // Execute
    result, err := exec.Execute(parsedCmd)
    if err != nil {
        return err
    }
    
    return nil
}
```

### Worker Mode Usage
```go
// pkg/webhook/worker.go
func (w *Worker) processJob(ctx context.Context, job *WebhookJob) {
    // Parse command
    parsedCmd, err := executor.ParseCommand(cfg.TriggerComment)
    if err != nil {
        logger.Errorf("Failed to parse command: %v", err)
        return
    }
    
    // Create execution context
    execCtx := &executor.ExecutionContext{
        PRHandler:       prHandler,
        Logger:          logger,
        Config:          executor.NewWebhookExecutionConfig(),
        MetricsRecorder: &WebhookMetricsRecorder{platform: job.Event.Platform},
        Platform:        job.Event.Platform,
        CommentSender:   job.Event.Sender.Login,
    }
    
    // Create executor
    exec := executor.NewCommandExecutor(execCtx)
    
    // Execute
    result, err := exec.Execute(parsedCmd)
    if err != nil {
        logger.Errorf("Command execution failed: %v", err)
    }
}
```

### Sync Mode Usage
```go
// pkg/webhook/server.go
func (s *Server) processWebhookSync(event *WebhookEvent) error {
    // Parse command
    parsedCmd, err := executor.ParseCommand(cfg.TriggerComment)
    if err != nil {
        return err
    }
    
    // Create execution context
    ctx := &executor.ExecutionContext{
        PRHandler:       prHandler,
        Logger:          s.logger,
        Config:          executor.NewWebhookExecutionConfig(),
        MetricsRecorder: &WebhookMetricsRecorder{platform: event.Platform},
        Platform:        event.Platform,
        CommentSender:   event.Sender.Login,
    }
    
    // Create executor
    exec := executor.NewCommandExecutor(ctx)
    
    // Execute
    result, err := exec.Execute(parsedCmd)
    return err
}
```

## Key Differences

| Aspect | Current | Proposed |
|--------|---------|----------|
| **Code Location** | 3 separate files | 1 unified package |
| **Validation** | Inconsistent | Configurable & consistent |
| **Error Handling** | 3 different ways | 1 configurable way |
| **Metrics** | Worker/Sync only | All modes (optional) |
| **Multi-command** | CLI + Worker only | All modes |
| **Type Conversion** | Required | Not required |
| **Lines of Code** | ~455 | ~200 |
| **Maintainability** | Low (3 copies) | High (1 source) |

## Migration Checklist

- [ ] Phase 1: Create new executor components
- [ ] Phase 2: Migrate CLI mode
- [ ] Phase 3: Migrate worker mode
- [ ] Phase 4: Migrate sync mode
- [ ] Phase 5: Remove type duplication
- [ ] Phase 6: Cleanup and documentation

