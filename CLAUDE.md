# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is a monorepo containing multiple independent DevOps tools and utilities for building and maintaining projects. Each tool is a standalone Go application or web service with its own structure, dependencies, and build process.

## Repository Structure

The repository contains 8 main tools:

- **artifact-scanner**: Scans container images for vulnerabilities and syncs results to Jira
- **dependabot**: Automated dependency vulnerability scanner and PR creator
- **kube-bench-report**: Generates HTML reports from kube-bench output
- **plugin-releaser**: Creates Jira issues for plugin release management
- **pr-cli**: Comment-driven PR/MR management for GitHub and GitLab
- **pr-watcher**: Monitors and reports on old PRs/MRs across organizations
- **roadmap-planner**: Full-stack web app (Go backend + React frontend) for Jira roadmap planning
- **syncfiles**: File synchronization utility

Each tool has its own `go.mod` (and `package.json` for frontend code) at the tool's root directory.

## Common Development Commands

### Building Tools

Each Go tool can be built independently from its directory:

```bash
# Most tools with Makefiles
cd <tool-name>
make build

# Tools without Makefiles
cd <tool-name>
go build -o bin/<tool-name> main.go
```

### Testing

```bash
# Standard test command for all Go tools
cd <tool-name>
go test ./...

# With coverage (for tools with Makefile)
make test-coverage
```

### Code Quality

```bash
# Format code
go fmt ./...
# Or with Makefile
make fmt

# Run linter (requires golangci-lint)
golangci-lint run
# Or with Makefile
make lint

# Run go vet
go vet ./...
# Or with Makefile
make vet
```

## Tool-Specific Build Instructions

### pr-cli

Most feature-complete Makefile with versioning support:

```bash
# Build binary
make build

# Build locally for development (creates pr-cli in current directory)
make build-local

# Run all checks
make check  # Runs fmt, vet, test

# Full development workflow
make dev    # clean-mocks, generate, fmt, vet, lint, test, build

# Generate mocks
make generate

# Docker operations
make docker-build
make docker-push

# Kubernetes deployment
make k8s-deploy ENVIRONMENT=development
make k8s-status ENVIRONMENT=development
```

### pr-watcher

```bash
# Build
make build

# Build for multiple platforms
make build-all

# Run with arguments
make run ARGS="watch-prs --org myorg --days 7"

# Check CLI dependencies
make check-gh      # Check GitHub CLI
make check-glab    # Check GitLab CLI
make check-all     # Check both
```

### dependabot

```bash
# Build and run with example
make example

# Create example trivy file for testing
make create-example-trivy
```

### roadmap-planner

Full-stack application with backend and frontend:

```bash
# Setup development environment (creates .env and config files)
make setup

# Run development servers (backend + frontend concurrently)
make dev

# Build both
make build

# Build separately
make build-backend
make build-frontend

# Run tests
make test              # Both
make test-backend
make test-frontend

# Docker operations
make docker-build      # Production image
make docker-run        # Production with docker-compose
make docker-run-dev    # Development with docker-compose
```

### Other Tools

Tools without Makefiles (artifact-scanner, syncfiles, kube-bench-report, plugin-releaser):

```bash
cd <tool-name>
go build -o bin/<tool-name> main.go
go test ./...
```

## Architecture Patterns

### pr-cli Architecture

- **pkg/platforms/**: Platform-specific implementations (GitHub, GitLab)
- **pkg/git/**: Git platform abstraction layer
- **pkg/handler/**: Business logic handlers for commands
- **pkg/config/**: Configuration management
- **cmd/**: CLI command definitions using Cobra
- **internal/version/**: Version information injected at build time
- **testing/**: Test utilities and mocks

Key design: Platform abstraction through interfaces, allowing support for multiple git platforms (GitHub/GitLab) with shared business logic.

### pr-watcher Architecture

- Uses Cobra framework
- No global variables
- Function-based initialization
- Clean separation of concerns between GitHub and GitLab implementations
- Leverages external CLIs (`gh` and `glab`) rather than direct API calls

### dependabot Architecture

Three-tier configuration system (repository config → local config → CLI flags):
1. Repository configs: `.dependabot.yml`, `.github/dependabot.yml`
2. Local config: specified via `--config` or `.dependabot.yaml`
3. Command-line parameters (highest priority)

Pipeline execution order: Clone → Pre-scan Hook → Security Scan → Post-scan Hook → Package Updates → Pre-commit Hook → Commit Changes → Post-commit Hook → PR Creation → Notification

### roadmap-planner Architecture

**Backend** (Go):
- REST API serving frontend and Jira integration
- `/api/auth/login`: Authentication endpoint
- Jira credentials passed via HTTP headers (`X-Jira-Username`, `X-Jira-Password`, `X-Jira-BaseURL`, `X-Jira-Project`)
- Configuration via `backend/config.yaml`

**Frontend** (React):
- Create React App based
- Communicates with backend API
- Built and served via nginx in production

## Testing Approach

### Running Tests

For individual tools:
```bash
cd <tool-name>
go test ./...
```

For tools with Makefiles:
```bash
make test              # Standard tests
make test-coverage     # With coverage report
```

### Mock Generation (pr-cli)

pr-cli uses generated mocks:
```bash
cd pr-cli
make generate  # Generates mock files using go generate
```

Clean mocks before regeneration:
```bash
make clean-mocks
```

## Version Management

pr-cli uses ldflags for version injection at build time:
- Version info in `internal/version/`
- Set via `VERSION`, `GIT_COMMIT`, `BUILD_DATE` build variables
- Access via `pr-cli version` or `pr-cli --version`

## Docker and Kubernetes

### pr-cli Docker/K8s

```bash
# Docker
make docker-build DOCKER_TAG=v1.0.0
make docker-push

# Kubernetes deployment
make k8s-deploy ENVIRONMENT=development
make k8s-status ENVIRONMENT=development
make k8s-logs ENVIRONMENT=development
```

### roadmap-planner Docker

```bash
# Unified production image
make build-unified
# Or
make docker-build

# Run with docker-compose
make docker-run        # Production
make docker-run-dev    # Development
```

## Configuration Files

### pr-cli
- Environment variables or CLI flags
- No config files, all via flags/env vars

### dependabot
- `.dependabot.yml` or `.github/dependabot.yml` in repositories
- Local config at specified path or `.dependabot.yaml`
- Supports hooks (preScan, postScan, preCommit, postCommit)

### roadmap-planner
- `.env` file in root (copy from `.env.example`)
- `backend/config.yaml` (copy from `config.example.yaml`)
- Contains Jira credentials and configuration

### artifact-scanner
- `config.yaml` with registry, Jira, OPS API, and user mappings

### plugin-releaser
- `config.yaml` with Jira configuration and plugin owner information

## External Dependencies

### CLI Tools Required

- **pr-watcher**: Requires `gh` (GitHub CLI) and/or `glab` (GitLab CLI) installed and authenticated
- **pr-cli**: Can use Git CLI for cherry-pick operations (configurable)

### Optional Tools

- `golangci-lint`: For running linters
- `kustomize` or `kubectl`: For Kubernetes deployments

## Tekton Integration

pr-cli has Tekton Pipeline integration:
- Pipeline definitions in `pr-cli/pipeline/`
- Result files written to `/tekton/results` (configurable via `--results-dir`)
- Results: `merge-successful`, `has-cherry-pick-comments`

## Common Patterns

### Multi-Platform Support
pr-cli and pr-watcher both support GitHub and GitLab through platform abstraction.

### Comment-Driven Operations
pr-cli processes commands via PR/MR comments (e.g., `/lgtm`, `/merge`, `/assign`).

### Jira Integration
Multiple tools integrate with Jira: artifact-scanner, plugin-releaser, roadmap-planner.

### Security Scanning
dependabot and artifact-scanner both scan for vulnerabilities (Trivy, etc.).

## Go Version

All tools require Go 1.25.3 or later.

## Module Paths

Tools use different module path conventions:
- Most use: `github.com/AlaudaDevops/toolbox/<tool-name>`
- Some use: `github.com/alaudadevops/toolbox/<tool-name>`

When working with imports, check the specific tool's `go.mod` for the correct path.
