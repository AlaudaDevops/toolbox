# Variables
BINARY_NAME=pr-watcher
BUILD_DIR=bin
MAIN_FILE=main.go

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)

	# Linux amd64
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_FILE)

	# macOS amd64
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_FILE)

	# macOS arm64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_FILE)

	# Windows amd64
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_FILE)

	@echo "Multi-platform build complete"

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Install the binary to GOPATH/bin
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(MAIN_FILE)

# Run the application (requires org flag)
.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	go run $(MAIN_FILE) $(ARGS)

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run

# Check for security issues
.PHONY: security
security:
	@echo "Running security checks..."
	gosec ./...

# Development build (with debug symbols)
.PHONY: dev-build
dev-build:
	@echo "Building development version..."
	@mkdir -p $(BUILD_DIR)
	go build -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME)-dev $(MAIN_FILE)

# Check if GitHub CLI is available
.PHONY: check-gh
check-gh:
	@echo "Checking GitHub CLI availability..."
	@which gh > /dev/null || (echo "GitHub CLI (gh) is not installed. Please install it from https://cli.github.com/" && exit 1)
	@gh auth status > /dev/null || (echo "GitHub CLI is not authenticated. Please run 'gh auth login'" && exit 1)
	@echo "GitHub CLI is available and authenticated"

# Check if GitLab CLI is available
.PHONY: check-glab
check-glab:
	@echo "Checking GitLab CLI availability..."
	@which glab > /dev/null || (echo "GitLab CLI (glab) is not installed. Please install it from https://gitlab.com/gitlab-org/cli" && exit 1)
	@glab auth status > /dev/null || (echo "GitLab CLI is not authenticated. Please run 'glab auth login'" && exit 1)
	@echo "GitLab CLI is available and authenticated"

# Check both CLIs
.PHONY: check-all
check-all: check-gh check-glab
	@echo "Both GitHub and GitLab CLIs are ready!"

# Example usage
.PHONY: example
example: build check-gh
	@echo "Running example commands..."
	@echo "Note: Replace 'YOUR_ORG' and 'YOUR_GROUP' with actual values"
	@echo ""
	@echo "GitHub example:"
	@echo "  ./$(BUILD_DIR)/$(BINARY_NAME) watch-prs --org YOUR_ORG --days 7"
	@echo ""
	@echo "GitLab example (requires glab):"
	@echo "  ./$(BUILD_DIR)/$(BINARY_NAME) watch-mrs --group YOUR_GROUP --days 7"
	@echo ""
	@echo "Run './scripts/examples.sh' for more detailed examples"

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  deps          - Install dependencies"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  run           - Run the application (use ARGS='...' for arguments)"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code (requires golangci-lint)"
	@echo "  security      - Run security checks (requires gosec)"
	@echo "  dev-build     - Build with debug symbols"
	@echo "  check-gh      - Check GitHub CLI availability"
	@echo "  check-glab    - Check GitLab CLI availability"
	@echo "  check-all     - Check both GitHub and GitLab CLIs"
	@echo "  example       - Show example usage for both platforms"
	@echo "  help          - Show this help message"
