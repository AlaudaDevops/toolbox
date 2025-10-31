# PR CLI Webhook Service Design

## Executive Summary

This document outlines the design for extending PR CLI from a command-line tool (used in Tekton Tasks) to also function as a webhook-receiving webservice. This will enable direct webhook integration with GitHub and GitLab, eliminating the dependency on Tekton Pipelines for PR management automation.

## Table of Contents

1. [Current Architecture](#current-architecture)
2. [Proposed Architecture](#proposed-architecture)
3. [Design Principles](#design-principles)
4. [Implementation Plan](#implementation-plan)
5. [Security Considerations](#security-considerations)
6. [Testing Strategy](#testing-strategy)
7. [Deployment & Operations](#deployment--operations)
8. [Migration Path](#migration-path)

## Current Architecture

### CLI Mode (Existing)
```
GitHub/GitLab PR Comment
    ↓
Tekton Pipeline (triggered by comment pattern)
    ↓
Tekton Task runs pr-cli container
    ↓
pr-cli processes command via CLI flags/env vars
    ↓
GitHub/GitLab API (execute action)
```

**Key Components:**
- **Entry Point**: `main.go` → `cmd.Execute()` → Cobra CLI
- **Configuration**: Flags + Environment variables (PR_* prefix)
- **Command Parsing**: `cmd/parser.go` - parses trigger comments
- **Command Execution**: `pkg/handler/` - executes PR operations
- **Platform Clients**: `pkg/platforms/github/`, `pkg/platforms/gitlab/`
- **Results Output**: Writes to `/tekton/results/` for pipeline integration

### Current Flow
1. User posts comment (e.g., `/lgtm`) on PR
2. Tekton Pipelines as Code detects comment via regex pattern
3. Pipeline extracts webhook payload data into parameters
4. Task runs `pr-cli` with parameters as env vars
5. CLI validates, parses command, executes operation
6. Results written to Tekton results directory

## Proposed Architecture

### Webhook Service Mode (New)
```
GitHub/GitLab Webhook Event
    ↓
pr-cli webservice (HTTP server)
    ↓
Webhook validation & parsing
    ↓
Event filtering (PR comment events)
    ↓
Command extraction & validation
    ↓
Async job queue (optional) or direct execution
    ↓
Reuse existing handler logic
    ↓
GitHub/GitLab API (execute action)
```

### Dual-Mode Architecture
```
┌─────────────────────────────────────────────────────────┐
│                      pr-cli Binary                       │
├─────────────────────────────────────────────────────────┤
│  main.go                                                 │
│    ├── CLI Mode (existing): pr-cli [flags]              │
│    └── Server Mode (new): pr-cli serve [flags]          │
└─────────────────────────────────────────────────────────┘
         │                              │
         │ CLI Mode                     │ Server Mode
         ▼                              ▼
┌──────────────────┐          ┌──────────────────────────┐
│  cmd/root.go     │          │  cmd/serve.go (new)      │
│  cmd/options.go  │          │  pkg/webhook/ (new)      │
│  cmd/parser.go   │          │    ├── server.go         │
│  cmd/executor.go │          │    ├── handlers.go       │
└──────────────────┘          │    ├── validator.go      │
         │                    │    ├── parser.go         │
         │                    │    └── middleware.go     │
         │                    └──────────────────────────┘
         │                              │
         └──────────────┬───────────────┘
                        ▼
         ┌──────────────────────────────┐
         │  Shared Core Components      │
         │  ├── pkg/handler/            │
         │  ├── pkg/platforms/          │
         │  ├── pkg/config/             │
         │  ├── pkg/git/                │
         │  └── pkg/comment/            │
         └──────────────────────────────┘
```

## Design Principles

1. **Code Reuse**: Maximize reuse of existing handler logic
2. **Backward Compatibility**: CLI mode continues to work unchanged
3. **Security First**: Webhook signature validation, rate limiting, authentication
4. **Cloud Native**: Stateless, horizontally scalable, health checks
5. **Observability**: Structured logging, metrics, tracing
6. **Testability**: Comprehensive unit and integration tests
7. **Configuration**: 12-factor app principles (env vars, config files)

## Implementation Plan

### Phase 1: Core Webhook Infrastructure

#### 1.1 New Command Structure

**File: `cmd/serve.go`** (new)
```go
package cmd

import (
    "github.com/spf13/cobra"
    "github.com/AlaudaDevops/toolbox/pr-cli/pkg/webhook"
)

var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start webhook server for PR management",
    Long: `Start an HTTP server that receives webhooks from GitHub/GitLab
and processes PR comment commands automatically.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        return runWebhookServer(cmd, args)
    },
}

func init() {
    rootCmd.AddCommand(serveCmd)

    // Server configuration
    serveCmd.Flags().String("listen-addr", ":8080", "Server listen address")
    serveCmd.Flags().String("webhook-path", "/webhook", "Webhook endpoint path")
    serveCmd.Flags().String("health-path", "/health", "Health check endpoint path")
    serveCmd.Flags().String("metrics-path", "/metrics", "Metrics endpoint path")

    // Security configuration
    serveCmd.Flags().String("webhook-secret", "", "Webhook secret for signature validation")
    serveCmd.Flags().String("webhook-secret-file", "", "File containing webhook secret")
    serveCmd.Flags().StringSlice("allowed-repos", []string{}, "Allowed repositories (owner/repo format)")
    serveCmd.Flags().Bool("require-signature", true, "Require webhook signature validation")

    // TLS configuration
    serveCmd.Flags().Bool("tls-enabled", false, "Enable TLS")
    serveCmd.Flags().String("tls-cert-file", "", "TLS certificate file")
    serveCmd.Flags().String("tls-key-file", "", "TLS private key file")

    // Processing configuration
    serveCmd.Flags().Bool("async-processing", true, "Process webhooks asynchronously")
    serveCmd.Flags().Int("worker-count", 10, "Number of worker goroutines")
    serveCmd.Flags().Int("queue-size", 100, "Job queue size")
    serveCmd.Flags().Duration("request-timeout", 30*time.Second, "Request processing timeout")

    // Rate limiting
    serveCmd.Flags().Int("rate-limit-requests", 100, "Max requests per minute per IP")
    serveCmd.Flags().Bool("rate-limit-enabled", true, "Enable rate limiting")
}
```

#### 1.2 Webhook Server Package

**File: `pkg/webhook/server.go`** (new)
```go
package webhook

import (
    "context"
    "net/http"
    "time"

    "github.com/sirupsen/logrus"
)

type Server struct {
    config     *Config
    logger     *logrus.Logger
    httpServer *http.Server
    jobQueue   chan *WebhookJob
    workers    []*Worker
}

type Config struct {
    ListenAddr        string
    WebhookPath       string
    HealthPath        string
    MetricsPath       string
    WebhookSecret     string
    AllowedRepos      []string
    RequireSignature  bool
    TLSEnabled        bool
    TLSCertFile       string
    TLSKeyFile        string
    AsyncProcessing   bool
    WorkerCount       int
    QueueSize         int
    RequestTimeout    time.Duration
    RateLimitRequests int
    RateLimitEnabled  bool
}

func NewServer(config *Config, logger *logrus.Logger) *Server {
    return &Server{
        config:   config,
        logger:   logger,
        jobQueue: make(chan *WebhookJob, config.QueueSize),
    }
}

func (s *Server) Start(ctx context.Context) error {
    // Initialize workers
    if s.config.AsyncProcessing {
        s.startWorkers(ctx)
    }

    // Setup HTTP routes
    mux := s.setupRoutes()

    s.httpServer = &http.Server{
        Addr:         s.config.ListenAddr,
        Handler:      mux,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Start server
    if s.config.TLSEnabled {
        return s.httpServer.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
    }
    return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
    // Stop accepting new requests
    if err := s.httpServer.Shutdown(ctx); err != nil {
        return err
    }

    // Close job queue
    close(s.jobQueue)

    // Wait for workers to finish
    // (implementation details)

    return nil
}
```

#### 1.3 Webhook Event Parsing

**File: `pkg/webhook/parser.go`** (new)
```go
package webhook

import (
    "encoding/json"
    "fmt"

    "github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
)

// WebhookEvent represents a parsed webhook event
type WebhookEvent struct {
    Platform      string // "github" or "gitlab"
    EventType     string // "issue_comment", "merge_request_comment", etc.
    Action        string // "created", "edited", etc.
    Repository    Repository
    PullRequest   PullRequest
    Comment       Comment
    Sender        User
}

type Repository struct {
    Owner string
    Name  string
    URL   string
}

type PullRequest struct {
    Number int
    Author string
    State  string
}

type Comment struct {
    ID   int64
    Body string
    User User
}

type User struct {
    Login string
}

// ParseGitHubWebhook parses GitHub webhook payload
func ParseGitHubWebhook(payload []byte, eventType string) (*WebhookEvent, error) {
    // GitHub sends issue_comment events for PR comments
    if eventType != "issue_comment" {
        return nil, fmt.Errorf("unsupported event type: %s", eventType)
    }

    var event struct {
        Action string `json:"action"`
        Issue  struct {
            Number      int    `json:"number"`
            PullRequest *struct{} `json:"pull_request"` // Present if it's a PR
            User        struct {
                Login string `json:"login"`
            } `json:"user"`
            State string `json:"state"`
        } `json:"issue"`
        Comment struct {
            ID   int64  `json:"id"`
            Body string `json:"body"`
            User struct {
                Login string `json:"login"`
            } `json:"user"`
        } `json:"comment"`
        Repository struct {
            Name  string `json:"name"`
            Owner struct {
                Login string `json:"login"`
            } `json:"owner"`
            HTMLURL string `json:"html_url"`
        } `json:"repository"`
        Sender struct {
            Login string `json:"login"`
        } `json:"sender"`
    }

    if err := json.Unmarshal(payload, &event); err != nil {
        return nil, fmt.Errorf("failed to parse GitHub webhook: %w", err)
    }

    // Only process PR comments (not regular issues)
    if event.Issue.PullRequest == nil {
        return nil, fmt.Errorf("not a pull request comment")
    }

    return &WebhookEvent{
        Platform:  "github",
        EventType: eventType,
        Action:    event.Action,
        Repository: Repository{
            Owner: event.Repository.Owner.Login,
            Name:  event.Repository.Name,
            URL:   event.Repository.HTMLURL,
        },
        PullRequest: PullRequest{
            Number: event.Issue.Number,
            Author: event.Issue.User.Login,
            State:  event.Issue.State,
        },
        Comment: Comment{
            ID:   event.Comment.ID,
            Body: event.Comment.Body,
            User: User{Login: event.Comment.User.Login},
        },
        Sender: User{Login: event.Sender.Login},
    }, nil
}

// ParseGitLabWebhook parses GitLab webhook payload
func ParseGitLabWebhook(payload []byte, eventType string) (*WebhookEvent, error) {
    // GitLab sends "Note Hook" events for comments
    if eventType != "Note Hook" {
        return nil, fmt.Errorf("unsupported event type: %s", eventType)
    }

    var event struct {
        ObjectKind string `json:"object_kind"`
        User       struct {
            Username string `json:"username"`
        } `json:"user"`
        Project struct {
            Name      string `json:"name"`
            Namespace string `json:"namespace"`
            WebURL    string `json:"web_url"`
        } `json:"project"`
        MergeRequest struct {
            IID    int    `json:"iid"`
            State  string `json:"state"`
            Author struct {
                Username string `json:"username"`
            } `json:"author"`
        } `json:"merge_request"`
        ObjectAttributes struct {
            Note string `json:"note"`
            ID   int64  `json:"id"`
        } `json:"object_attributes"`
    }

    if err := json.Unmarshal(payload, &event); err != nil {
        return nil, fmt.Errorf("failed to parse GitLab webhook: %w", err)
    }

    if event.ObjectKind != "note" {
        return nil, fmt.Errorf("not a comment event")
    }

    return &WebhookEvent{
        Platform:  "gitlab",
        EventType: eventType,
        Repository: Repository{
            Owner: event.Project.Namespace,
            Name:  event.Project.Name,
            URL:   event.Project.WebURL,
        },
        PullRequest: PullRequest{
            Number: event.MergeRequest.IID,
            Author: event.MergeRequest.Author.Username,
            State:  event.MergeRequest.State,
        },
        Comment: Comment{
            ID:   event.ObjectAttributes.ID,
            Body: event.ObjectAttributes.Note,
            User: User{Login: event.User.Username},
        },
        Sender: User{Login: event.User.Username},
    }, nil
}

// ToConfig converts WebhookEvent to pr-cli Config
func (e *WebhookEvent) ToConfig(baseConfig *config.Config) *config.Config {
    cfg := *baseConfig // Copy base config

    cfg.Platform = e.Platform
    cfg.Owner = e.Repository.Owner
    cfg.Repo = e.Repository.Name
    cfg.PRNum = e.PullRequest.Number
    cfg.CommentSender = e.Comment.User.Login
    cfg.TriggerComment = e.Comment.Body

    return &cfg
}
```

#### 1.4 Webhook Validation & Security

**File: `pkg/webhook/validator.go`** (new)
```go
package webhook

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "strings"
)

// ValidateGitHubSignature validates GitHub webhook signature
func ValidateGitHubSignature(payload []byte, signature string, secret string) error {
    if signature == "" {
        return fmt.Errorf("missing signature")
    }

    // GitHub sends signature as "sha256=<hash>"
    if !strings.HasPrefix(signature, "sha256=") {
        return fmt.Errorf("invalid signature format")
    }

    expectedMAC := signature[7:] // Remove "sha256=" prefix

    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    actualMAC := hex.EncodeToString(mac.Sum(nil))

    if !hmac.Equal([]byte(actualMAC), []byte(expectedMAC)) {
        return fmt.Errorf("signature mismatch")
    }

    return nil
}

// ValidateGitLabToken validates GitLab webhook token
func ValidateGitLabToken(token string, secret string) error {
    if token == "" {
        return fmt.Errorf("missing token")
    }

    if token != secret {
        return fmt.Errorf("invalid token")
    }

    return nil
}

// ValidateRepository checks if repository is allowed
func ValidateRepository(owner, repo string, allowedRepos []string) error {
    if len(allowedRepos) == 0 {
        return nil // No restrictions
    }

    repoFullName := fmt.Sprintf("%s/%s", owner, repo)

    for _, allowed := range allowedRepos {
        if allowed == repoFullName || allowed == "*" {
            return nil
        }
        // Support wildcard patterns like "myorg/*"
        if strings.HasSuffix(allowed, "/*") {
            orgPrefix := strings.TrimSuffix(allowed, "/*")
            if owner == orgPrefix {
                return nil
            }
        }
    }

    return fmt.Errorf("repository %s not allowed", repoFullName)
}
```

#### 1.5 HTTP Handlers

**File: `pkg/webhook/handlers.go`** (new)
```go
package webhook

import (
    "encoding/json"
    "io"
    "net/http"

    "github.com/sirupsen/logrus"
)

func (s *Server) setupRoutes() *http.ServeMux {
    mux := http.NewServeMux()

    // Apply middleware
    webhookHandler := s.loggingMiddleware(
        s.rateLimitMiddleware(
            http.HandlerFunc(s.handleWebhook),
        ),
    )

    mux.Handle(s.config.WebhookPath, webhookHandler)
    mux.HandleFunc(s.config.HealthPath, s.handleHealth)
    mux.HandleFunc(s.config.MetricsPath, s.handleMetrics)

    return mux
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Read payload
    body, err := io.ReadAll(r.Body)
    if err != nil {
        s.logger.Errorf("Failed to read request body: %v", err)
        http.Error(w, "Failed to read request", http.StatusBadRequest)
        return
    }
    defer r.Body.Close()

    // Determine platform from headers
    platform := s.detectPlatform(r)
    if platform == "" {
        http.Error(w, "Unable to detect platform", http.StatusBadRequest)
        return
    }

    // Validate signature
    if s.config.RequireSignature {
        if err := s.validateSignature(r, body, platform); err != nil {
            s.logger.Warnf("Signature validation failed: %v", err)
            http.Error(w, "Invalid signature", http.StatusUnauthorized)
            return
        }
    }

    // Parse webhook event
    event, err := s.parseWebhook(body, r, platform)
    if err != nil {
        s.logger.Debugf("Failed to parse webhook: %v", err)
        // Return 200 to avoid retries for non-PR events
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "ignored",
            "reason": err.Error(),
        })
        return
    }

    // Validate repository
    if err := ValidateRepository(event.Repository.Owner, event.Repository.Name, s.config.AllowedRepos); err != nil {
        s.logger.Warnf("Repository validation failed: %v", err)
        http.Error(w, "Repository not allowed", http.StatusForbidden)
        return
    }

    // Filter events (only process "created" comments)
    if event.Action != "created" {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "ignored",
            "reason": "not a created comment",
        })
        return
    }

    // Check if comment matches command pattern
    if !s.isCommandComment(event.Comment.Body) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "ignored",
            "reason": "not a command comment",
        })
        return
    }

    // Process webhook
    if s.config.AsyncProcessing {
        // Queue for async processing
        job := &WebhookJob{
            Event:     event,
            Timestamp: time.Now(),
        }

        select {
        case s.jobQueue <- job:
            s.logger.Infof("Queued webhook job for PR #%d", event.PullRequest.Number)
            w.WriteHeader(http.StatusAccepted)
            json.NewEncoder(w).Encode(map[string]string{
                "status": "queued",
            })
        default:
            s.logger.Error("Job queue full")
            http.Error(w, "Server busy", http.StatusServiceUnavailable)
        }
    } else {
        // Process synchronously
        if err := s.processWebhook(r.Context(), event); err != nil {
            s.logger.Errorf("Failed to process webhook: %v", err)
            http.Error(w, "Processing failed", http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "processed",
        })
    }
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": "healthy",
        "queue_size": len(s.jobQueue),
        "queue_capacity": cap(s.jobQueue),
    })
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
    // TODO: Implement Prometheus metrics
    w.Header().Set("Content-Type", "text/plain")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("# Metrics endpoint - TODO: implement Prometheus metrics\n"))
}

func (s *Server) detectPlatform(r *http.Request) string {
    // GitHub sends X-GitHub-Event header
    if r.Header.Get("X-GitHub-Event") != "" {
        return "github"
    }

    // GitLab sends X-Gitlab-Event header
    if r.Header.Get("X-Gitlab-Event") != "" {
        return "gitlab"
    }

    return ""
}

func (s *Server) validateSignature(r *http.Request, body []byte, platform string) error {
    switch platform {
    case "github":
        signature := r.Header.Get("X-Hub-Signature-256")
        return ValidateGitHubSignature(body, signature, s.config.WebhookSecret)
    case "gitlab":
        token := r.Header.Get("X-Gitlab-Token")
        return ValidateGitLabToken(token, s.config.WebhookSecret)
    default:
        return fmt.Errorf("unknown platform: %s", platform)
    }
}

func (s *Server) parseWebhook(body []byte, r *http.Request, platform string) (*WebhookEvent, error) {
    switch platform {
    case "github":
        eventType := r.Header.Get("X-GitHub-Event")
        return ParseGitHubWebhook(body, eventType)
    case "gitlab":
        eventType := r.Header.Get("X-Gitlab-Event")
        return ParseGitLabWebhook(body, eventType)
    default:
        return nil, fmt.Errorf("unknown platform: %s", platform)
    }
}

func (s *Server) isCommandComment(body string) bool {
    // Reuse existing comment pattern matching
    return strings.HasPrefix(strings.TrimSpace(body), "/")
}
```

#### 1.6 Async Worker Pool

**File: `pkg/webhook/worker.go`** (new)
```go
package webhook

import (
    "context"
    "time"

    "github.com/AlaudaDevops/toolbox/pr-cli/pkg/config"
    "github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
    "github.com/sirupsen/logrus"
)

type WebhookJob struct {
    Event     *WebhookEvent
    Timestamp time.Time
}

type Worker struct {
    id         int
    jobQueue   <-chan *WebhookJob
    logger     *logrus.Logger
    baseConfig *config.Config
}

func (s *Server) startWorkers(ctx context.Context) {
    s.workers = make([]*Worker, s.config.WorkerCount)

    for i := 0; i < s.config.WorkerCount; i++ {
        worker := &Worker{
            id:         i,
            jobQueue:   s.jobQueue,
            logger:     s.logger,
            baseConfig: s.getBaseConfig(),
        }
        s.workers[i] = worker
        go worker.start(ctx)
    }

    s.logger.Infof("Started %d webhook workers", s.config.WorkerCount)
}

func (w *Worker) start(ctx context.Context) {
    w.logger.Infof("Worker %d started", w.id)

    for {
        select {
        case <-ctx.Done():
            w.logger.Infof("Worker %d stopping", w.id)
            return
        case job, ok := <-w.jobQueue:
            if !ok {
                w.logger.Infof("Worker %d: job queue closed", w.id)
                return
            }
            w.processJob(ctx, job)
        }
    }
}

func (w *Worker) processJob(ctx context.Context, job *WebhookJob) {
    logger := w.logger.WithFields(logrus.Fields{
        "worker":   w.id,
        "platform": job.Event.Platform,
        "repo":     fmt.Sprintf("%s/%s", job.Event.Repository.Owner, job.Event.Repository.Name),
        "pr":       job.Event.PullRequest.Number,
        "command":  job.Event.Comment.Body,
    })

    logger.Info("Processing webhook job")

    // Convert webhook event to config
    cfg := job.Event.ToConfig(w.baseConfig)

    // Create PR handler
    prHandler, err := handler.NewPRHandler(logger.Logger, cfg)
    if err != nil {
        logger.Errorf("Failed to create PR handler: %v", err)
        return
    }

    // Parse command (reuse existing CLI parser logic)
    parsedCmd, err := parseCommand(cfg.TriggerComment)
    if err != nil {
        logger.Errorf("Failed to parse command: %v", err)
        return
    }

    // Execute command (reuse existing execution logic)
    if err := executeCommand(prHandler, parsedCmd); err != nil {
        logger.Errorf("Failed to execute command: %v", err)
        return
    }

    logger.Info("Successfully processed webhook job")
}

func (s *Server) processWebhook(ctx context.Context, event *WebhookEvent) error {
    // Synchronous processing (similar to worker logic)
    cfg := event.ToConfig(s.getBaseConfig())

    prHandler, err := handler.NewPRHandler(s.logger, cfg)
    if err != nil {
        return fmt.Errorf("failed to create PR handler: %w", err)
    }

    parsedCmd, err := parseCommand(cfg.TriggerComment)
    if err != nil {
        return fmt.Errorf("failed to parse command: %w", err)
    }

    return executeCommand(prHandler, parsedCmd)
}

func (s *Server) getBaseConfig() *config.Config {
    // Return base configuration with tokens, LGTM settings, etc.
    // This should be loaded from environment variables or config file
    return &config.Config{
        Token:           os.Getenv("PR_TOKEN"),
        CommentToken:    os.Getenv("PR_COMMENT_TOKEN"),
        LGTMThreshold:   getEnvInt("PR_LGTM_THRESHOLD", 1),
        LGTMPermissions: getEnvSlice("PR_LGTM_PERMISSIONS", []string{"admin", "write"}),
        MergeMethod:     getEnv("PR_MERGE_METHOD", "auto"),
        Debug:           getEnvBool("PR_DEBUG", false),
        Verbose:         getEnvBool("PR_VERBOSE", false),
        RobotAccounts:   getEnvSlice("PR_ROBOT_ACCOUNTS", []string{}),
    }
}
```

#### 1.7 Middleware

**File: `pkg/webhook/middleware.go`** (new)
```go
package webhook

import (
    "net/http"
    "sync"
    "time"

    "golang.org/x/time/rate"
)

// Rate limiter per IP
type rateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rate     int
}

func newRateLimiter(requestsPerMinute int) *rateLimiter {
    return &rateLimiter{
        limiters: make(map[string]*rate.Limiter),
        rate:     requestsPerMinute,
    }
}

func (rl *rateLimiter) getLimiter(ip string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    limiter, exists := rl.limiters[ip]
    if !exists {
        limiter = rate.NewLimiter(rate.Limit(rl.rate)/60, rl.rate)
        rl.limiters[ip] = limiter
    }

    return limiter
}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
    if !s.config.RateLimitEnabled {
        return next
    }

    limiter := newRateLimiter(s.config.RateLimitRequests)

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := getClientIP(r)

        if !limiter.getLimiter(ip).Allow() {
            s.logger.Warnf("Rate limit exceeded for IP: %s", ip)
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Wrap response writer to capture status code
        wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

        next.ServeHTTP(wrapped, r)

        s.logger.WithFields(logrus.Fields{
            "method":      r.Method,
            "path":        r.URL.Path,
            "status":      wrapped.statusCode,
            "duration_ms": time.Since(start).Milliseconds(),
            "ip":          getClientIP(r),
        }).Info("HTTP request")
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

func getClientIP(r *http.Request) string {
    // Check X-Forwarded-For header
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        return strings.Split(xff, ",")[0]
    }

    // Check X-Real-IP header
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }

    // Fall back to RemoteAddr
    return strings.Split(r.RemoteAddr, ":")[0]
}
```

### Phase 2: Integration & Refactoring

#### 2.1 Shared Command Execution Logic

To avoid code duplication, extract command parsing and execution into shared functions:

**File: `pkg/executor/executor.go`** (new)
```go
package executor

import (
    "github.com/AlaudaDevops/toolbox/pr-cli/cmd"
    "github.com/AlaudaDevops/toolbox/pr-cli/pkg/handler"
)

// ParseCommand parses a trigger comment into a structured command
// This wraps the existing cmd.parseCommand logic
func ParseCommand(comment string) (*cmd.ParsedCommand, error) {
    // Reuse existing parser from cmd package
    option := cmd.NewPROption()
    return option.ParseCommand(comment) // Need to expose this method
}

// ExecuteCommand executes a parsed command using the PR handler
func ExecuteCommand(prHandler *handler.PRHandler, parsedCmd *cmd.ParsedCommand) error {
    // Reuse existing execution logic from cmd package
    switch parsedCmd.Type {
    case cmd.BuiltInCommand:
        return prHandler.ExecuteCommand(parsedCmd.Command, parsedCmd.Args)
    case cmd.SingleCommand:
        return prHandler.ExecuteCommand(parsedCmd.Command, parsedCmd.Args)
    case cmd.MultiCommand:
        return executeMultiCommand(prHandler, parsedCmd)
    default:
        return fmt.Errorf("unknown command type: %s", parsedCmd.Type)
    }
}

func executeMultiCommand(prHandler *handler.PRHandler, parsedCmd *cmd.ParsedCommand) error {
    // Reuse multi-command execution logic
    // This may require exposing some methods from cmd package
    return nil // TODO: implement
}
```

#### 2.2 Configuration Management

Extend the config package to support webhook-specific settings:

**File: `pkg/config/webhook.go`** (new)
```go
package config

// WebhookConfig holds webhook server configuration
type WebhookConfig struct {
    // Server settings
    ListenAddr     string `json:"listen_addr" yaml:"listen_addr" mapstructure:"listen-addr"`
    WebhookPath    string `json:"webhook_path" yaml:"webhook_path" mapstructure:"webhook-path"`
    HealthPath     string `json:"health_path" yaml:"health_path" mapstructure:"health-path"`
    MetricsPath    string `json:"metrics_path" yaml:"metrics_path" mapstructure:"metrics-path"`

    // Security
    WebhookSecret    string   `json:"-" yaml:"-" mapstructure:"webhook-secret"`
    AllowedRepos     []string `json:"allowed_repos" yaml:"allowed_repos" mapstructure:"allowed-repos"`
    RequireSignature bool     `json:"require_signature" yaml:"require_signature" mapstructure:"require-signature"`

    // TLS
    TLSEnabled  bool   `json:"tls_enabled" yaml:"tls_enabled" mapstructure:"tls-enabled"`
    TLSCertFile string `json:"tls_cert_file" yaml:"tls_cert_file" mapstructure:"tls-cert-file"`
    TLSKeyFile  string `json:"tls_key_file" yaml:"tls_key_file" mapstructure:"tls-key-file"`

    // Processing
    AsyncProcessing bool `json:"async_processing" yaml:"async_processing" mapstructure:"async-processing"`
    WorkerCount     int  `json:"worker_count" yaml:"worker_count" mapstructure:"worker-count"`
    QueueSize       int  `json:"queue_size" yaml:"queue_size" mapstructure:"queue-size"`

    // Rate limiting
    RateLimitEnabled  bool `json:"rate_limit_enabled" yaml:"rate_limit_enabled" mapstructure:"rate-limit-enabled"`
    RateLimitRequests int  `json:"rate_limit_requests" yaml:"rate_limit_requests" mapstructure:"rate-limit-requests"`
}

// NewDefaultWebhookConfig returns default webhook configuration
func NewDefaultWebhookConfig() *WebhookConfig {
    return &WebhookConfig{
        ListenAddr:        ":8080",
        WebhookPath:       "/webhook",
        HealthPath:        "/health",
        MetricsPath:       "/metrics",
        RequireSignature:  true,
        TLSEnabled:        false,
        AsyncProcessing:   true,
        WorkerCount:       10,
        QueueSize:         100,
        RateLimitEnabled:  true,
        RateLimitRequests: 100,
    }
}
```

### Phase 3: Observability & Monitoring

#### 3.1 Structured Logging

Enhance logging with structured fields for better observability:

```go
// In webhook handlers
logger.WithFields(logrus.Fields{
    "event_id":    event.Comment.ID,
    "platform":    event.Platform,
    "repository":  fmt.Sprintf("%s/%s", event.Repository.Owner, event.Repository.Name),
    "pr_number":   event.PullRequest.Number,
    "command":     event.Comment.Body,
    "sender":      event.Sender.Login,
    "action":      event.Action,
}).Info("Processing webhook event")
```

#### 3.2 Prometheus Metrics

**File: `pkg/webhook/metrics.go`** (new)
```go
package webhook

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    webhookRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "pr_cli_webhook_requests_total",
            Help: "Total number of webhook requests received",
        },
        []string{"platform", "event_type", "status"},
    )

    webhookProcessingDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "pr_cli_webhook_processing_duration_seconds",
            Help:    "Webhook processing duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"platform", "command"},
    )

    commandExecutionTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "pr_cli_command_execution_total",
            Help: "Total number of commands executed",
        },
        []string{"platform", "command", "status"},
    )

    queueSize = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "pr_cli_queue_size",
            Help: "Current size of the webhook job queue",
        },
    )

    activeWorkers = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "pr_cli_active_workers",
            Help: "Number of active worker goroutines",
        },
    )
)

// Update metrics in handlers
func (s *Server) recordMetrics(platform, eventType, status string, duration time.Duration) {
    webhookRequestsTotal.WithLabelValues(platform, eventType, status).Inc()
    queueSize.Set(float64(len(s.jobQueue)))
}
```

#### 3.3 Health Checks

Enhanced health check with detailed status:

```go
type HealthStatus struct {
    Status        string            `json:"status"`
    Version       string            `json:"version"`
    Uptime        string            `json:"uptime"`
    QueueSize     int               `json:"queue_size"`
    QueueCapacity int               `json:"queue_capacity"`
    Workers       int               `json:"workers"`
    Checks        map[string]string `json:"checks"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    status := HealthStatus{
        Status:        "healthy",
        Version:       version.Get().Version,
        Uptime:        time.Since(s.startTime).String(),
        QueueSize:     len(s.jobQueue),
        QueueCapacity: cap(s.jobQueue),
        Workers:       s.config.WorkerCount,
        Checks:        make(map[string]string),
    }

    // Check queue capacity
    if float64(len(s.jobQueue))/float64(cap(s.jobQueue)) > 0.9 {
        status.Checks["queue"] = "warning: queue nearly full"
    } else {
        status.Checks["queue"] = "ok"
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(status)
}
```

## Security Considerations

### 1. Webhook Signature Validation

**Critical**: Always validate webhook signatures to prevent unauthorized requests.

- **GitHub**: Validates `X-Hub-Signature-256` header using HMAC-SHA256
- **GitLab**: Validates `X-Gitlab-Token` header using secret token
- **Implementation**: See `pkg/webhook/validator.go`

**Configuration**:
```bash
# Set webhook secret (must match GitHub/GitLab webhook configuration)
export WEBHOOK_SECRET="your-secret-here"

# Or use file-based secret (Kubernetes secret mount)
export WEBHOOK_SECRET_FILE="/etc/secrets/webhook-secret"
```

### 2. Repository Allowlist

Restrict which repositories can trigger commands:

```bash
# Allow specific repositories
export ALLOWED_REPOS="myorg/repo1,myorg/repo2"

# Allow all repos in an organization
export ALLOWED_REPOS="myorg/*"

# Allow all repositories (not recommended for production)
export ALLOWED_REPOS="*"
```

### 3. Rate Limiting

Protect against abuse and DoS attacks:

- **Per-IP rate limiting**: Default 100 requests/minute
- **Configurable limits**: Adjust based on your needs
- **Automatic cleanup**: Old limiters are periodically cleaned up

### 4. TLS/HTTPS

**Production requirement**: Always use TLS in production.

```bash
# Enable TLS
export TLS_ENABLED=true
export TLS_CERT_FILE=/etc/certs/tls.crt
export TLS_KEY_FILE=/etc/certs/tls.key
```

**Recommendation**: Use a reverse proxy (nginx, Traefik, Ingress) for TLS termination.

### 5. Authentication & Authorization

**Token Security**:
- Store GitHub/GitLab tokens in Kubernetes secrets
- Use separate tokens for read vs. write operations if possible
- Rotate tokens regularly
- Use minimal required permissions

**Permission Checks**:
- Existing LGTM permission checks still apply
- Comment sender validation remains in place
- PR author restrictions are enforced

### 6. Input Validation

- Validate all webhook payloads
- Sanitize comment bodies before processing
- Reject malformed requests early
- Log suspicious activity

### 7. Secrets Management

**Best Practices**:
```yaml
# Kubernetes Secret
apiVersion: v1
kind: Secret
metadata:
  name: pr-cli-secrets
type: Opaque
stringData:
  webhook-secret: "your-webhook-secret"
  github-token: "ghp_xxxxxxxxxxxx"
  gitlab-token: "glpat-xxxxxxxxxxxx"
```

**Environment Variables**:
```bash
# Load from secret files (Kubernetes secret mounts)
export WEBHOOK_SECRET_FILE=/etc/secrets/webhook-secret
export PR_TOKEN_FILE=/etc/secrets/github-token
export PR_COMMENT_TOKEN_FILE=/etc/secrets/github-comment-token
```

### 8. Network Security

- **Firewall rules**: Restrict access to webhook endpoint
- **IP allowlisting**: Allow only GitHub/GitLab webhook IPs
- **Private networks**: Deploy in private VPC when possible
- **Service mesh**: Use Istio/Linkerd for mTLS

### 9. Audit Logging

Log all webhook events for security auditing:

```go
logger.WithFields(logrus.Fields{
    "event_id":       event.Comment.ID,
    "platform":       event.Platform,
    "repository":     fmt.Sprintf("%s/%s", event.Repository.Owner, event.Repository.Name),
    "pr_number":      event.PullRequest.Number,
    "command":        event.Comment.Body,
    "sender":         event.Sender.Login,
    "sender_ip":      getClientIP(r),
    "signature_valid": true,
    "timestamp":      time.Now().UTC(),
}).Info("Webhook event processed")
```

### 10. Security Headers

Add security headers to all responses:

```go
func securityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        next.ServeHTTP(w, r)
    })
}
```

## Testing Strategy

### 1. Unit Tests

#### 1.1 Webhook Parser Tests

**File: `pkg/webhook/parser_test.go`**
```go
func TestParseGitHubWebhook(t *testing.T) {
    tests := []struct {
        name        string
        payload     string
        eventType   string
        wantEvent   *WebhookEvent
        wantErr     bool
    }{
        {
            name:      "valid PR comment",
            payload:   loadFixture("github_pr_comment.json"),
            eventType: "issue_comment",
            wantEvent: &WebhookEvent{
                Platform: "github",
                PullRequest: PullRequest{Number: 123},
                Comment: Comment{Body: "/lgtm"},
            },
            wantErr: false,
        },
        {
            name:      "non-PR issue comment",
            payload:   loadFixture("github_issue_comment.json"),
            eventType: "issue_comment",
            wantErr:   true, // Should reject non-PR comments
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            event, err := ParseGitHubWebhook([]byte(tt.payload), tt.eventType)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseGitHubWebhook() error = %v, wantErr %v", err, tt.wantErr)
            }
            // Assert event fields...
        })
    }
}
```

#### 1.2 Signature Validation Tests

**File: `pkg/webhook/validator_test.go`**
```go
func TestValidateGitHubSignature(t *testing.T) {
    secret := "test-secret"
    payload := []byte(`{"test": "data"}`)

    // Generate valid signature
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

    tests := []struct {
        name      string
        payload   []byte
        signature string
        secret    string
        wantErr   bool
    }{
        {"valid signature", payload, validSig, secret, false},
        {"invalid signature", payload, "sha256=invalid", secret, true},
        {"missing signature", payload, "", secret, true},
        {"wrong secret", payload, validSig, "wrong-secret", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateGitHubSignature(tt.payload, tt.signature, tt.secret)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateGitHubSignature() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

#### 1.3 Repository Validation Tests

**File: `pkg/webhook/validator_test.go`**
```go
func TestValidateRepository(t *testing.T) {
    tests := []struct {
        name         string
        owner        string
        repo         string
        allowedRepos []string
        wantErr      bool
    }{
        {"exact match", "myorg", "repo1", []string{"myorg/repo1"}, false},
        {"wildcard org", "myorg", "repo1", []string{"myorg/*"}, false},
        {"wildcard all", "anyorg", "anyrepo", []string{"*"}, false},
        {"not allowed", "otherorg", "repo1", []string{"myorg/*"}, true},
        {"empty allowlist", "anyorg", "anyrepo", []string{}, false}, // No restrictions
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateRepository(tt.owner, tt.repo, tt.allowedRepos)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateRepository() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 2. Integration Tests

#### 2.1 End-to-End Webhook Tests

**File: `pkg/webhook/server_test.go`**
```go
func TestWebhookServer_HandleGitHubWebhook(t *testing.T) {
    // Setup test server
    config := &Config{
        ListenAddr:       ":0", // Random port
        WebhookPath:      "/webhook",
        WebhookSecret:    "test-secret",
        RequireSignature: true,
        AsyncProcessing:  false, // Synchronous for testing
    }

    server := NewServer(config, logrus.New())

    // Start server in background
    go server.Start(context.Background())
    defer server.Shutdown(context.Background())

    // Create test webhook payload
    payload := loadFixture("github_pr_comment_lgtm.json")

    // Sign payload
    mac := hmac.New(sha256.New, []byte(config.WebhookSecret))
    mac.Write(payload)
    signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

    // Send webhook request
    req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payload))
    req.Header.Set("X-GitHub-Event", "issue_comment")
    req.Header.Set("X-Hub-Signature-256", signature)
    req.Header.Set("Content-Type", "application/json")

    rr := httptest.NewRecorder()
    server.handleWebhook(rr, req)

    // Assert response
    assert.Equal(t, http.StatusOK, rr.Code)

    // Assert command was executed (mock GitHub API)
    // ...
}
```

### 3. Load Testing

**File: `testing/load/webhook_load_test.go`**
```go
func TestWebhookServer_LoadTest(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }

    // Setup server with realistic config
    config := &Config{
        WorkerCount:     10,
        QueueSize:       100,
        AsyncProcessing: true,
    }

    server := NewServer(config, logrus.New())

    // Simulate 1000 concurrent webhook requests
    concurrency := 100
    totalRequests := 1000

    var wg sync.WaitGroup
    errors := make(chan error, totalRequests)

    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < totalRequests/concurrency; j++ {
                if err := sendWebhookRequest(server); err != nil {
                    errors <- err
                }
            }
        }()
    }

    wg.Wait()
    close(errors)

    // Check for errors
    errorCount := 0
    for err := range errors {
        t.Logf("Request error: %v", err)
        errorCount++
    }

    // Allow some failures but not too many
    if errorCount > totalRequests/10 {
        t.Errorf("Too many errors: %d/%d", errorCount, totalRequests)
    }
}
```

### 4. Security Tests

**File: `pkg/webhook/security_test.go`**
```go
func TestWebhookServer_SecurityChecks(t *testing.T) {
    tests := []struct {
        name           string
        setupRequest   func(*http.Request)
        expectedStatus int
    }{
        {
            name: "missing signature",
            setupRequest: func(r *http.Request) {
                // Don't set signature header
            },
            expectedStatus: http.StatusUnauthorized,
        },
        {
            name: "invalid signature",
            setupRequest: func(r *http.Request) {
                r.Header.Set("X-Hub-Signature-256", "sha256=invalid")
            },
            expectedStatus: http.StatusUnauthorized,
        },
        {
            name: "rate limit exceeded",
            setupRequest: func(r *http.Request) {
                // Send many requests from same IP
            },
            expectedStatus: http.StatusTooManyRequests,
        },
        {
            name: "repository not allowed",
            setupRequest: func(r *http.Request) {
                // Use payload with non-allowed repo
            },
            expectedStatus: http.StatusForbidden,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 5. Test Fixtures

Create realistic webhook payloads for testing:

**File: `pkg/webhook/testdata/github_pr_comment_lgtm.json`**
```json
{
  "action": "created",
  "issue": {
    "number": 123,
    "state": "open",
    "pull_request": {},
    "user": {
      "login": "pr-author"
    }
  },
  "comment": {
    "id": 987654321,
    "body": "/lgtm",
    "user": {
      "login": "reviewer"
    }
  },
  "repository": {
    "name": "test-repo",
    "owner": {
      "login": "test-org"
    },
    "html_url": "https://github.com/test-org/test-repo"
  },
  "sender": {
    "login": "reviewer"
  }
}
```

### 6. Continuous Integration

**File: `.github/workflows/test-webhook.yml`**
```yaml
name: Webhook Tests

on:
  pull_request:
    paths:
      - 'pkg/webhook/**'
      - 'cmd/serve.go'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'

      - name: Run webhook unit tests
        run: go test -v -race -coverprofile=coverage.out ./pkg/webhook/...

      - name: Run integration tests
        run: go test -v -tags=integration ./pkg/webhook/...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

## Deployment & Operations

### 1. Kubernetes Deployment

**File: `deploy/kubernetes/deployment.yaml`**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pr-cli-webhook
  namespace: pr-automation
  labels:
    app: pr-cli-webhook
spec:
  replicas: 3
  selector:
    matchLabels:
      app: pr-cli-webhook
  template:
    metadata:
      labels:
        app: pr-cli-webhook
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: pr-cli-webhook
      securityContext:
        runAsNonRoot: true
        runAsUser: 65532
        fsGroup: 65532
      containers:
      - name: pr-cli
        image: registry.alauda.cn:60070/devops/toolbox/pr-cli:latest
        imagePullPolicy: Always
        command: ["/usr/local/bin/pr-cli"]
        args: ["serve"]
        ports:
        - name: http
          containerPort: 8080
          protocol: TCP
        env:
        # Server configuration
        - name: LISTEN_ADDR
          value: ":8080"
        - name: WEBHOOK_PATH
          value: "/webhook"
        - name: HEALTH_PATH
          value: "/health"
        - name: METRICS_PATH
          value: "/metrics"

        # Security configuration
        - name: WEBHOOK_SECRET
          valueFrom:
            secretKeyRef:
              name: pr-cli-secrets
              key: webhook-secret
        - name: REQUIRE_SIGNATURE
          value: "true"
        - name: ALLOWED_REPOS
          value: "myorg/*"

        # GitHub/GitLab tokens
        - name: PR_TOKEN
          valueFrom:
            secretKeyRef:
              name: pr-cli-secrets
              key: github-token
        - name: PR_COMMENT_TOKEN
          valueFrom:
            secretKeyRef:
              name: pr-cli-secrets
              key: github-comment-token
              optional: true

        # PR CLI configuration
        - name: PR_PLATFORM
          value: "github"
        - name: PR_LGTM_THRESHOLD
          value: "1"
        - name: PR_LGTM_PERMISSIONS
          value: "admin,write"
        - name: PR_MERGE_METHOD
          value: "auto"
        - name: PR_ROBOT_ACCOUNTS
          value: "dependabot,renovate"
        - name: PR_VERBOSE
          value: "true"

        # Worker configuration
        - name: ASYNC_PROCESSING
          value: "true"
        - name: WORKER_COUNT
          value: "10"
        - name: QUEUE_SIZE
          value: "100"

        # Rate limiting
        - name: RATE_LIMIT_ENABLED
          value: "true"
        - name: RATE_LIMIT_REQUESTS
          value: "100"

        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 1000m
            memory: 512Mi

        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3

        readinessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 2

        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          capabilities:
            drop:
            - ALL
---
apiVersion: v1
kind: Service
metadata:
  name: pr-cli-webhook
  namespace: pr-automation
  labels:
    app: pr-cli-webhook
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: http
    protocol: TCP
    name: http
  selector:
    app: pr-cli-webhook
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: pr-cli-webhook
  namespace: pr-automation
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/rate-limit: "100"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - pr-webhook.example.com
    secretName: pr-cli-webhook-tls
  rules:
  - host: pr-webhook.example.com
    http:
      paths:
      - path: /webhook
        pathType: Prefix
        backend:
          service:
            name: pr-cli-webhook
            port:
              number: 80
```

### 2. Secrets Management

**File: `deploy/kubernetes/secrets.yaml`**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pr-cli-secrets
  namespace: pr-automation
type: Opaque
stringData:
  # Webhook secret (must match GitHub/GitLab webhook configuration)
  webhook-secret: "CHANGE_ME_RANDOM_SECRET"

  # GitHub Personal Access Token with repo permissions
  github-token: "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

  # Optional: Separate token for posting comments
  github-comment-token: "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

  # GitLab Personal Access Token (if using GitLab)
  gitlab-token: "glpat-xxxxxxxxxxxxxxxxxxxx"
```

**Note**: In production, use external secret management:
- **Sealed Secrets**: Encrypt secrets in Git
- **External Secrets Operator**: Sync from Vault, AWS Secrets Manager, etc.
- **SOPS**: Encrypt secrets with age or PGP

### 3. RBAC Configuration

**File: `deploy/kubernetes/rbac.yaml`**
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: pr-cli-webhook
  namespace: pr-automation
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pr-cli-webhook
  namespace: pr-automation
rules:
- apiGroups: [""]
  resources: ["secrets"]
  resourceNames: ["pr-cli-secrets"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: pr-cli-webhook
  namespace: pr-automation
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: pr-cli-webhook
subjects:
- kind: ServiceAccount
  name: pr-cli-webhook
  namespace: pr-automation
```

### 4. Monitoring & Alerting

**File: `deploy/kubernetes/servicemonitor.yaml`**
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: pr-cli-webhook
  namespace: pr-automation
  labels:
    app: pr-cli-webhook
spec:
  selector:
    matchLabels:
      app: pr-cli-webhook
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

**File: `deploy/kubernetes/prometheusrule.yaml`**
```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: pr-cli-webhook-alerts
  namespace: pr-automation
spec:
  groups:
  - name: pr-cli-webhook
    interval: 30s
    rules:
    - alert: PRCliWebhookDown
      expr: up{job="pr-cli-webhook"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "PR CLI webhook service is down"
        description: "PR CLI webhook has been down for more than 5 minutes"

    - alert: PRCliQueueFull
      expr: pr_cli_queue_size / pr_cli_queue_capacity > 0.9
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "PR CLI job queue is nearly full"
        description: "Queue is {{ $value | humanizePercentage }} full"

    - alert: PRCliHighErrorRate
      expr: |
        rate(pr_cli_webhook_requests_total{status="error"}[5m]) /
        rate(pr_cli_webhook_requests_total[5m]) > 0.1
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High error rate in PR CLI webhook"
        description: "Error rate is {{ $value | humanizePercentage }}"
```

### 5. Horizontal Pod Autoscaling

**File: `deploy/kubernetes/hpa.yaml`**
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: pr-cli-webhook
  namespace: pr-automation
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: pr-cli-webhook
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: pr_cli_queue_size
      target:
        type: AverageValue
        averageValue: "50"
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 30
```

### 6. GitHub Webhook Configuration

Configure webhook in GitHub repository settings:

1. Go to **Settings** → **Webhooks** → **Add webhook**
2. **Payload URL**: `https://pr-webhook.example.com/webhook`
3. **Content type**: `application/json`
4. **Secret**: Same as `WEBHOOK_SECRET` in Kubernetes secret
5. **Events**: Select "Issue comments" only
6. **Active**: ✓ Enabled

### 7. GitLab Webhook Configuration

Configure webhook in GitLab project settings:

1. Go to **Settings** → **Webhooks**
2. **URL**: `https://pr-webhook.example.com/webhook`
3. **Secret token**: Same as `WEBHOOK_SECRET` in Kubernetes secret
4. **Trigger**: Select "Comments" only
5. **Enable SSL verification**: ✓ Enabled

### 8. Local Development

**File: `docker-compose.yml`**
```yaml
version: '3.8'

services:
  pr-cli-webhook:
    build:
      context: .
      dockerfile: Dockerfile
    command: ["serve"]
    ports:
      - "8080:8080"
    environment:
      - LISTEN_ADDR=:8080
      - WEBHOOK_SECRET=dev-secret
      - REQUIRE_SIGNATURE=false  # Disable for local testing
      - PR_TOKEN=${GITHUB_TOKEN}
      - PR_PLATFORM=github
      - PR_LGTM_THRESHOLD=1
      - PR_VERBOSE=true
      - ASYNC_PROCESSING=true
      - WORKER_COUNT=5
    volumes:
      - ./logs:/var/log/pr-cli
```

**Run locally**:
```bash
# Set GitHub token
export GITHUB_TOKEN=ghp_your_token_here

# Start service
docker-compose up

# Test webhook (use ngrok for GitHub to reach local service)
ngrok http 8080

# Configure GitHub webhook to use ngrok URL
```

## Migration Path

### Option 1: Gradual Migration (Recommended)

Run both Tekton and webhook service in parallel:

**Phase 1: Deploy Webhook Service**
1. Deploy webhook service to Kubernetes
2. Configure webhooks for a test repository
3. Monitor and validate functionality
4. Keep Tekton pipeline as fallback

**Phase 2: Expand Coverage**
1. Add more repositories to webhook service
2. Monitor performance and reliability
3. Tune worker count and queue size
4. Keep Tekton for critical repositories

**Phase 3: Full Migration**
1. Migrate all repositories to webhook service
2. Disable Tekton pipeline triggers
3. Keep Tekton pipeline definition for emergency fallback
4. Monitor for 2-4 weeks

**Phase 4: Cleanup**
1. Remove Tekton pipeline if no issues
2. Archive Tekton configuration
3. Update documentation

### Option 2: Direct Migration

For smaller deployments or new installations:

1. Deploy webhook service
2. Configure all webhooks
3. Remove Tekton pipeline
4. Monitor closely for first week

### Rollback Plan

If issues occur with webhook service:

1. **Immediate**: Disable webhook in GitHub/GitLab settings
2. **Quick**: Re-enable Tekton pipeline triggers
3. **Investigation**: Review logs, metrics, and errors
4. **Fix**: Address issues in webhook service
5. **Retry**: Re-enable webhooks after fixes

### Comparison: Tekton vs Webhook Service

| Aspect | Tekton Pipeline | Webhook Service |
|--------|----------------|-----------------|
| **Latency** | 10-30 seconds (pipeline startup) | <1 second (direct processing) |
| **Resource Usage** | High (new pod per event) | Low (shared pods) |
| **Scalability** | Limited by cluster capacity | Horizontal pod autoscaling |
| **Cost** | Higher (compute per event) | Lower (shared resources) |
| **Complexity** | Higher (pipeline + task definitions) | Lower (single service) |
| **Observability** | Tekton dashboard + logs | Prometheus + logs |
| **Debugging** | Pipeline logs, task logs | Structured logs, metrics |
| **Maintenance** | Pipeline YAML updates | Code updates + deployment |
| **Flexibility** | Limited to Tekton features | Full control over logic |

### Coexistence Strategy

Both modes can coexist indefinitely:

**Use Tekton for**:
- Complex multi-step workflows
- Integration with other Tekton pipelines
- Repositories requiring special handling
- Compliance/audit requirements

**Use Webhook Service for**:
- Simple PR comment commands
- High-volume repositories
- Low-latency requirements
- Cost optimization

## Implementation Roadmap

### Milestone 1: Foundation (Week 1-2)

**Goals**: Basic webhook server with GitHub support

**Tasks**:
- [ ] Create `cmd/serve.go` command
- [ ] Implement `pkg/webhook/server.go`
- [ ] Implement `pkg/webhook/parser.go` (GitHub only)
- [ ] Implement `pkg/webhook/validator.go`
- [ ] Implement `pkg/webhook/handlers.go`
- [ ] Add basic unit tests
- [ ] Update `go.mod` with new dependencies

**Dependencies**:
```bash
go get golang.org/x/time/rate
go get github.com/prometheus/client_golang/prometheus
```

**Deliverables**:
- Working webhook server for GitHub
- Basic signature validation
- Synchronous processing
- Health check endpoint

### Milestone 2: Async Processing (Week 3)

**Goals**: Add worker pool and async processing

**Tasks**:
- [ ] Implement `pkg/webhook/worker.go`
- [ ] Add job queue management
- [ ] Implement graceful shutdown
- [ ] Add worker pool tests
- [ ] Add load testing

**Deliverables**:
- Async webhook processing
- Configurable worker count
- Queue size monitoring
- Load test results

### Milestone 3: Security & Observability (Week 4)

**Goals**: Production-ready security and monitoring

**Tasks**:
- [ ] Implement `pkg/webhook/middleware.go`
- [ ] Add rate limiting
- [ ] Add security headers
- [ ] Implement `pkg/webhook/metrics.go`
- [ ] Add Prometheus metrics
- [ ] Enhanced health checks
- [ ] Structured logging improvements

**Deliverables**:
- Rate limiting per IP
- Prometheus metrics
- Security headers
- Comprehensive logging

### Milestone 4: GitLab Support (Week 5)

**Goals**: Add GitLab webhook support

**Tasks**:
- [ ] Implement GitLab webhook parsing
- [ ] Add GitLab signature validation
- [ ] Update tests for GitLab
- [ ] Add GitLab test fixtures
- [ ] Update documentation

**Deliverables**:
- Full GitLab support
- GitLab integration tests
- Updated documentation

### Milestone 5: Deployment & Documentation (Week 6)

**Goals**: Production deployment and documentation

**Tasks**:
- [ ] Create Kubernetes manifests
- [ ] Create Helm chart (optional)
- [ ] Write deployment guide
- [ ] Write operations runbook
- [ ] Create monitoring dashboards
- [ ] Write migration guide

**Deliverables**:
- Kubernetes deployment manifests
- Complete documentation
- Grafana dashboards
- Migration guide

### Milestone 6: Testing & Validation (Week 7-8)

**Goals**: Comprehensive testing and validation

**Tasks**:
- [ ] Integration tests with real GitHub API
- [ ] Security penetration testing
- [ ] Load testing (1000+ req/min)
- [ ] Chaos engineering tests
- [ ] Documentation review
- [ ] Code review

**Deliverables**:
- >80% code coverage
- Load test results
- Security audit report
- Production-ready service

## File Structure Summary

New files to be created:

```
pr-cli/
├── cmd/
│   └── serve.go                          # New: serve command
├── pkg/
│   ├── webhook/                          # New: webhook package
│   │   ├── server.go                     # HTTP server
│   │   ├── handlers.go                   # HTTP handlers
│   │   ├── parser.go                     # Webhook parsing
│   │   ├── validator.go                  # Security validation
│   │   ├── worker.go                     # Async workers
│   │   ├── middleware.go                 # HTTP middleware
│   │   ├── metrics.go                    # Prometheus metrics
│   │   ├── server_test.go                # Tests
│   │   ├── parser_test.go                # Tests
│   │   ├── validator_test.go             # Tests
│   │   └── testdata/                     # Test fixtures
│   │       ├── github_pr_comment_lgtm.json
│   │       ├── github_pr_comment_merge.json
│   │       └── gitlab_mr_comment_lgtm.json
│   ├── executor/                         # New: shared execution logic
│   │   ├── executor.go                   # Command execution
│   │   └── executor_test.go              # Tests
│   └── config/
│       └── webhook.go                    # New: webhook config
├── deploy/                               # New: deployment configs
│   └── kubernetes/
│       ├── deployment.yaml               # Deployment manifest
│       ├── service.yaml                  # Service manifest
│       ├── ingress.yaml                  # Ingress manifest
│       ├── secrets.yaml                  # Secrets template
│       ├── rbac.yaml                     # RBAC configuration
│       ├── servicemonitor.yaml           # Prometheus monitoring
│       ├── prometheusrule.yaml           # Alerting rules
│       └── hpa.yaml                      # Autoscaling
├── docs/
│   ├── webhook-service-design.md         # This document
│   ├── webhook-deployment.md             # New: deployment guide
│   └── webhook-operations.md             # New: operations runbook
└── docker-compose.yml                    # New: local development
```

## Dependencies

### New Go Dependencies

```go
// go.mod additions
require (
    golang.org/x/time v0.14.0                          // Rate limiting
    github.com/prometheus/client_golang v1.19.0        // Metrics
)
```

### External Dependencies

- **Kubernetes**: 1.25+ (for deployment)
- **Prometheus**: For metrics collection
- **Grafana**: For dashboards (optional)
- **Ingress Controller**: nginx, Traefik, or similar
- **Cert Manager**: For TLS certificates (optional)

## Success Criteria

### Functional Requirements
- ✅ Receives and validates GitHub webhooks
- ✅ Receives and validates GitLab webhooks
- ✅ Parses PR comment events correctly
- ✅ Executes all existing PR commands
- ✅ Maintains backward compatibility with CLI mode
- ✅ Handles errors gracefully

### Non-Functional Requirements
- ✅ <1 second webhook processing latency (p95)
- ✅ >99.9% uptime
- ✅ Handles 1000+ webhooks/minute
- ✅ <100MB memory per pod
- ✅ Horizontal scaling to 10+ pods
- ✅ Zero data loss (at-least-once processing)

### Security Requirements
- ✅ Webhook signature validation
- ✅ Repository allowlist enforcement
- ✅ Rate limiting per IP
- ✅ TLS/HTTPS support
- ✅ Secrets management
- ✅ Audit logging

### Operational Requirements
- ✅ Health check endpoint
- ✅ Prometheus metrics
- ✅ Structured logging
- ✅ Graceful shutdown
- ✅ Kubernetes-ready
- ✅ Comprehensive documentation

## Risks & Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Webhook signature bypass** | High | Low | Mandatory signature validation, security testing |
| **DoS attack** | High | Medium | Rate limiting, queue size limits, autoscaling |
| **Token compromise** | High | Low | Secrets rotation, minimal permissions, audit logs |
| **Queue overflow** | Medium | Medium | Queue size monitoring, alerts, backpressure |
| **GitHub API rate limits** | Medium | Medium | Token rotation, caching, retry logic |
| **Memory leaks** | Medium | Low | Load testing, memory profiling, limits |
| **Breaking changes** | Low | Low | Comprehensive tests, gradual rollout |

## Future Enhancements

### Phase 2 Features (Post-MVP)

1. **Multi-tenancy**: Support multiple organizations with separate configs
2. **Plugin System**: Allow custom command handlers
3. **Webhook Replay**: Replay failed webhooks from persistent queue
4. **Advanced Routing**: Route webhooks to different handlers based on rules
5. **GraphQL API**: Query webhook processing status
6. **Webhook Forwarding**: Forward webhooks to other services
7. **A/B Testing**: Test new features on subset of repositories
8. **Machine Learning**: Predict merge conflicts, suggest reviewers

### Integration Opportunities

1. **Slack/Teams**: Post notifications to chat
2. **Jira**: Update tickets based on PR status
3. **SonarQube**: Trigger code quality checks
4. **ArgoCD**: Trigger deployments on merge
5. **PagerDuty**: Alert on critical PR issues

## Conclusion

This design provides a comprehensive plan for extending PR CLI with webhook service capabilities while maintaining backward compatibility with the existing Tekton-based approach. The implementation follows cloud-native best practices with emphasis on security, scalability, and observability.

**Key Benefits**:
- ⚡ **10-30x faster** response time vs Tekton
- 💰 **Lower cost** through resource sharing
- 📈 **Better scalability** with horizontal autoscaling
- 🔒 **Enhanced security** with signature validation and rate limiting
- 📊 **Improved observability** with Prometheus metrics
- 🔄 **Gradual migration** path with zero downtime

**Next Steps**:
1. Review and approve this design document
2. Create GitHub issues for each milestone
3. Begin implementation with Milestone 1
4. Set up CI/CD for automated testing
5. Deploy to staging environment for validation
6. Gradual rollout to production

---

**Document Version**: 1.0
**Last Updated**: 2025-10-31
**Author**: PR CLI Development Team
**Status**: Draft - Pending Review

