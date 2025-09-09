# Makefile for PR CLI project

# Build variables
BINARY_NAME=pr-cli
BUILD_DIR=bin
GO_FILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")
MAIN_FILE=main.go

# Version variables
VERSION?=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
GIT_COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION?=$(shell go version | awk '{print $$3}')

# Linker flags for version injection
LDFLAGS=-ldflags "-w -s -X github.com/AlaudaDevops/toolbox/pr-cli/internal/version.Version=$(VERSION) \
	-X github.com/AlaudaDevops/toolbox/pr-cli/internal/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/AlaudaDevops/toolbox/pr-cli/internal/version.BuildDate=$(BUILD_DATE)"

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build the binary
.PHONY: build
build: clean tidy
	@echo "Building $(BINARY_NAME) with version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)

## build-local: Build the binary in current directory (for development)
.PHONY: build-local
build-local: tidy
	@echo "Building $(BINARY_NAME) locally with version $(VERSION)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_FILE)

## run: Run the application with help
.PHONY: run
run: build-local
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME) --help

## version: Build and show version information
.PHONY: version
version: build-local
	@echo "Version information:"
	@./$(BINARY_NAME) version
	@echo ""
	@echo "Short version:"
	@./$(BINARY_NAME) --version

## clean: Clean build files
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)

## tidy: Tidy go modules
.PHONY: tidy
tidy:
	@echo "Tidying Go modules..."
	$(GOMOD) tidy

## download: Download go modules
.PHONY: download
download:
	@echo "Downloading Go modules..."
	$(GOMOD) download

## test: Run tests
.PHONY: test
test: tidy
	@echo "Running tests..."
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
.PHONY: test-coverage
test-coverage: tidy
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## fmt: Format Go code
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	$(GOFMT) -s -w $(GO_FILES)

## vet: Run go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## lint: Run golangci-lint (requires golangci-lint to be installed)
.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo >&2 "golangci-lint is required but not installed. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run

## install: Install the binary to GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install

## check: Run all checks (fmt, vet, test)
.PHONY: check
check: fmt vet test

## demo: Run demo commands (requires proper environment variables)
.PHONY: demo
demo: build-local
	@echo "Running PR CLI demo commands..."
	@echo "Testing GitHub platform with /help command..."
	@./test-pr-cli.sh || echo "Demo completed (expected to fail without real tokens)"

## generate: Generate mock files using go generate
.PHONY: generate
generate:
	@echo "Generating mock files..."
	@go generate ./...

## clean-coverage: Clean coverage files
.PHONY: clean-coverage
clean-coverage:
	@echo "Cleaning coverage files..."
	@rm -f coverage.out coverage.html

## clean-mocks: Clean generated mock files
.PHONY: clean-mocks
clean-mocks:
	@echo "Cleaning mock files..."
	@rm -rf testing/mock/

## all: Run all tasks (clean, build, test)
.PHONY: all
all: clean generate fmt vet lint test build

## dev: Development workflow with mock generation
.PHONY: dev
dev: clean-mocks generate fmt vet lint test build

# Docker variables
DOCKER_IMAGE=build-harbor.alauda.cn/devps/toolbox/pr-cli
DOCKER_TAG?=$(VERSION)
DOCKER_BUILD_ARGS=--build-arg VERSION=$(VERSION) --build-arg GIT_COMMIT=$(GIT_COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE)

## docker-build: Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build $(DOCKER_BUILD_ARGS) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker image built successfully: $(DOCKER_IMAGE):$(DOCKER_TAG)"
