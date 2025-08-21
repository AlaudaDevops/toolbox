#!/bin/bash

# CI Test Runner Script for Roadmap Planner
# This script mimics the CI pipeline for local testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in the right directory
if [[ ! -d "backend" ]] || [[ ! -d "frontend" ]]; then
    log_error "Please run this script from the roadmap-planner directory"
    exit 1
fi

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."

    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi

    # Check Node.js
    if ! command -v node &> /dev/null; then
        log_error "Node.js is not installed"
        exit 1
    fi

    # Check npm
    if ! command -v npm &> /dev/null; then
        log_error "npm is not installed"
        exit 1
    fi

    # Check Docker (optional)
    if ! command -v docker &> /dev/null; then
        log_warning "Docker is not installed - skipping Docker tests"
        SKIP_DOCKER=true
    fi

    log_success "All required dependencies are available"
}

# Backend tests
test_backend() {
    log_info "Running backend tests..."

    cd backend

    # Install dependencies
    log_info "Installing Go dependencies..."
    go mod download
    go mod verify

    # Check formatting
    log_info "Checking Go formatting..."
    if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
        log_error "Go files need formatting. Run: cd backend && gofmt -s -w ."
        gofmt -s -l .
        cd ..
        return 1
    fi

    # Run go mod tidy check
    log_info "Checking go mod tidy..."
    go mod tidy
    if ! git diff --quiet -- go.mod go.sum; then
        log_error "go.mod or go.sum is not tidy. Run: cd backend && go mod tidy"
        cd ..
        return 1
    fi

    # Run vet
    log_info "Running go vet..."
    go vet ./...

    # Run tests with coverage
    log_info "Running Go tests with coverage..."
    go test -v -race -coverprofile=coverage.out ./...

    # Generate coverage report
    go tool cover -html=coverage.out -o coverage.html

    # Check coverage
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
    log_info "Test coverage: ${COVERAGE}%"

    if (( $(echo "$COVERAGE < 60" | bc -l) )); then
        log_warning "Coverage is below 60%"
    fi

    cd ..
    log_success "Backend tests completed"
}

# Frontend tests
test_frontend() {
    log_info "Running frontend tests..."

    cd frontend

    # Install dependencies
    log_info "Installing npm dependencies..."
    npm ci

    # Security audit
    log_info "Running security audit..."
    npm audit --audit-level high || log_warning "Security vulnerabilities found"

    log_warning "Skipping lint and tests on frontend for now..."
    # TODO: fix all the tests and make possible to use lint and tests

    # Run linting if available
    # if npm run lint --if-present > /dev/null 2>&1; then
    #     log_info "Running ESLint..."
    #     npm run lint
    # else
    #     log_warning "No lint script found in package.json"
    # fi

    # Run tests
    # log_info "Running frontend tests..."
    # npm test -- --coverage --watchAll=false --passWithNoTests

    # Build frontend
    log_info "Building frontend..."
    npm run build

    cd ..
    log_success "Frontend tests completed"
}

# Docker tests
test_docker() {
    if [[ "$SKIP_DOCKER" == "true" ]]; then
        log_warning "Skipping Docker tests - Docker not available"
        return 0
    fi

    log_info "Running Docker tests..."

    # Build Docker image
    log_info "Building Docker image..."
    docker build -t roadmap-planner:test .

    # Test Docker image
    log_info "Testing Docker image..."

    # Start container
    docker run -d --name roadmap-planner-test \
        -p 8080:8080 \
        -e DEBUG=true \
        roadmap-planner:test

    # Wait for startup
    sleep 10

    # Test health endpoint
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        log_success "Docker health check passed"
    else
        log_error "Docker health check failed"
        docker logs roadmap-planner-test
        docker stop roadmap-planner-test
        docker rm roadmap-planner-test
        return 1
    fi

    # Clean up
    docker stop roadmap-planner-test
    docker rm roadmap-planner-test

    log_success "Docker tests completed"
}

# Main execution
main() {
    log_info "Starting CI test runner for Roadmap Planner..."

    check_dependencies

    # Run tests based on arguments
    if [[ "$1" == "backend" ]]; then
        test_backend
    elif [[ "$1" == "frontend" ]]; then
        test_frontend
    elif [[ "$1" == "docker" ]]; then
        test_docker
    else
        # Run all tests
        test_backend
        test_frontend
        test_docker
    fi

    log_success "All tests completed successfully! 🎉"
}

# Help message
if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
    echo "Usage: $0 [backend|frontend|docker|--help]"
    echo ""
    echo "Options:"
    echo "  backend     Run only backend tests"
    echo "  frontend    Run only frontend tests"
    echo "  docker      Run only Docker tests"
    echo "  --help, -h  Show this help message"
    echo ""
    echo "If no option is provided, all tests will run."
    exit 0
fi

# Run main function
main "$@"
