# Makefile for DependaBot project

# Build variables
BINARY_NAME=dependabot
BUILD_DIR=bin
GO_FILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")
MAIN_FILE=main.go

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
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)

## run: Run the application with sample arguments
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME) --help

## clean: Clean build files
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)

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

## example: Run with example trivy file
.PHONY: example
example: build create-example-trivy
	@echo "Running with example trivy results..."
	@$(BUILD_DIR)/$(BINARY_NAME) -t examples/trivy-results.json -p examples/sample-project

## create-example-trivy: Create example trivy results file
.PHONY: create-example-trivy
create-example-trivy:
	@mkdir -p examples/sample-project
	@echo 'module github.com/example/sample' > examples/sample-project/go.mod
	@echo 'go 1.21' >> examples/sample-project/go.mod
	@echo '{"Results":[{"Target":"go.mod","Class":"lang-pkgs","Type":"gomod","Vulnerabilities":[{"VulnerabilityID":"CVE-2023-1234","PkgName":"github.com/example/vulnerable-package","InstalledVersion":"v1.0.0","FixedVersion":"v1.1.0","Severity":"HIGH","Title":"Example vulnerability","Description":"This is an example vulnerability"}]}]}' > examples/trivy-results.json

## clean-examples: Clean example files
.PHONY: clean-examples
clean-examples:
	@rm -rf examples/

## all: Run all tasks (clean, build, test)
.PHONY: all
all: clean fmt vet test build
