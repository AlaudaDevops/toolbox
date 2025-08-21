# CI/CD Documentation for Roadmap Planner

This document describes the continuous integration and continuous deployment (CI/CD) setup for the Roadmap Planner project.

## Overview

The CI/CD pipeline is implemented using GitHub Actions and includes comprehensive testing, security scanning, and quality checks for both backend (Go) and frontend (React) components.

## Pipeline Structure

### Workflow Triggers

The CI pipeline runs on:
- **Push** to `main` or `develop` branches (only when roadmap-planner files change)
- **Pull Requests** targeting `main` or `develop` branches
- Changes to the workflow file itself

### Path-based Filtering

The pipeline uses intelligent path filtering to only run relevant jobs:
- **Backend changes**: `roadmap-planner/backend/**`, `roadmap-planner/Makefile`
- **Frontend changes**: `roadmap-planner/frontend/**`, `roadmap-planner/Makefile`
- **Docker changes**: `roadmap-planner/Dockerfile`, `roadmap-planner/docker-compose*.yml`

## Jobs and Stages

### 1. Change Detection (`changes`)
- Determines which parts of the codebase have changed
- Outputs flags used by subsequent jobs to decide if they should run
- Uses `dorny/paths-filter` action for accurate path detection

### 2. Backend Tests (`backend-tests`)
- **Triggers**: When backend files change
- **Go Version**: 1.24.6
- **Steps**:
  - Dependency installation and verification
  - Code formatting check (`gofmt`)
  - Module tidiness check (`go mod tidy`)
  - Static analysis (`go vet`)
  - Linting (`golangci-lint`)
  - Unit tests with race detection
  - Coverage analysis (minimum 60% threshold)
  - Coverage report generation

### 3. Frontend Tests (`frontend-tests`)
- **Triggers**: When frontend files change
- **Node Version**: 18
- **Steps**:
  - Dependency installation (`npm ci`)
  - Security vulnerability scan (`npm audit`)
  - ESLint linting
  - Unit tests with coverage
  - Build verification
  - Artifact upload for build files

### 4. Security Scanning (`security-scan`)
- **Triggers**: When backend or frontend files change
- **Tools**: Trivy vulnerability scanner
- **Scans**: Filesystem vulnerabilities
- **Integration**: Results uploaded to GitHub Security tab

### 5. Docker Build Test (`docker-build`)
- **Triggers**: After successful backend/frontend tests or Docker file changes
- **Steps**:
  - Multi-stage Docker image build
  - Container startup test
  - Health endpoint verification
  - Docker image vulnerability scanning
  - Cleanup

### 6. Integration Tests (`integration-tests`)
- **Triggers**: After successful backend and frontend tests
- **Steps**:
  - Build and start unified container
  - Health check verification
  - API endpoint testing
  - Static file serving verification
  - Service log collection

### 7. Code Quality (`code-quality`)
- **Tools**: SonarCloud (optional)
- **Analysis**: Code quality metrics, technical debt, security hotspots
- **Configuration**: `sonar-project.properties`

### 8. Final Status Check (`final-check`)
- **Purpose**: Aggregate status of all jobs
- **Behavior**: Fails if any critical job fails
- **Output**: Clear success/failure summary

## Configuration Files

### GitHub Actions Workflow
- **File**: `.github/workflows/roadmap-planner-ci.yaml`
- **Purpose**: Main CI/CD pipeline definition

### Go Linting Configuration
- **File**: `backend/.golangci.yml`
- **Purpose**: golangci-lint configuration with comprehensive rules
- **Rules**: 25+ enabled linters with project-specific settings

### SonarCloud Configuration
- **File**: `sonar-project.properties`
- **Purpose**: Code quality analysis configuration
- **Metrics**: Coverage, duplications, maintainability, reliability, security

### Enhanced .gitignore
- **File**: `.gitignore`
- **Purpose**: Comprehensive ignore patterns for build artifacts, dependencies, IDE files

## Local Testing

### CI Test Script
- **File**: `ci-test.sh`
- **Purpose**: Local execution of CI checks
- **Usage**:
  ```bash
  ./ci-test.sh           # Run all tests
  ./ci-test.sh backend   # Backend only
  ./ci-test.sh frontend  # Frontend only
  ./ci-test.sh docker    # Docker only
  ./ci-test.sh --help    # Show help
  ```

### Manual Testing Commands
```bash
# Backend
cd backend
go mod download && go mod verify
gofmt -s -l .
go mod tidy
go vet ./...
golangci-lint run
go test -v -race -coverprofile=coverage.out ./...

# Frontend
cd frontend
npm ci
npm audit --audit-level high
npm run lint
npm test -- --coverage --watchAll=false
npm run build

# Docker
docker build -t roadmap-planner:test .
docker run -d --name test -p 8080:8080 roadmap-planner:test
curl -f http://localhost:8080/health
docker stop test && docker rm test
```

## Quality Gates

### Coverage Requirements
- **Backend**: Minimum 60% test coverage
- **Frontend**: Coverage collected and reported (threshold configurable)

### Security Requirements
- **Dependencies**: No high-severity vulnerabilities
- **Code**: No critical security issues in static analysis
- **Docker**: No high/critical vulnerabilities in final image

### Code Quality Requirements
- **Formatting**: All code must be properly formatted
- **Linting**: No linting errors (warnings allowed)
- **Dependencies**: All modules must be tidy and verified

## Monitoring and Notifications

### GitHub Integration
- **Checks**: All jobs appear as GitHub status checks
- **Pull Requests**: Cannot be merged without passing checks
- **Security**: Vulnerabilities reported to Security tab

### Artifacts
- **Backend Coverage**: HTML and raw coverage reports
- **Frontend Build**: Production build artifacts
- **Retention**: 7 days for all artifacts

## Best Practices

### For Developers
1. **Run tests locally** before pushing using `./ci-test.sh`
2. **Keep dependencies updated** and security-audited
3. **Write meaningful tests** to maintain coverage thresholds
4. **Follow code formatting** standards enforced by linters

### For Maintainers
1. **Review security reports** regularly in GitHub Security tab
2. **Monitor coverage trends** to ensure quality maintenance
3. **Update dependencies** in CI workflow as needed
4. **Adjust quality gates** based on project maturity

## Troubleshooting

### Common Issues

1. **Coverage Below Threshold**
   - Add more unit tests
   - Check if new code is tested
   - Review coverage report artifacts

2. **Linting Failures**
   - Run `golangci-lint run` locally
   - Run `npm run lint` for frontend
   - Fix formatting with `gofmt -s -w .`

3. **Docker Build Failures**
   - Check if frontend builds successfully
   - Verify Dockerfile syntax
   - Test build locally with same commands

4. **Security Vulnerabilities**
   - Review Trivy scan results
   - Update vulnerable dependencies
   - Check GitHub Security tab for details

### Getting Help

- **CI Logs**: Check GitHub Actions logs for detailed error messages
- **Local Reproduction**: Use `./ci-test.sh` to reproduce issues locally
- **Coverage Reports**: Download artifacts to see detailed coverage analysis
