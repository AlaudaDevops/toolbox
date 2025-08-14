# Roadmap Planner Makefile

.PHONY: help build run test clean docker-build docker-run dev setup

# Default target
help:
	@echo "Available targets:"
	@echo "  setup          - Set up development environment"
	@echo "  dev            - Run development servers"
	@echo "  build          - Build both backend and frontend"
	@echo "  test           - Run tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  docker-build   - Build Docker images"
	@echo "  docker-run     - Run with Docker Compose"
	@echo "  docker-stop    - Stop Docker containers"

# Setup development environment
setup:
	@echo "Setting up development environment..."
	@if [ ! -f .env ]; then cp .env.example .env; echo "Created .env file from example"; fi
	@if [ ! -f backend/config.yaml ]; then cp backend/config.example.yaml backend/config.yaml; echo "Created config.yaml from example"; fi
	@echo "Installing backend dependencies..."
	@cd backend && go mod tidy
	@echo "Installing frontend dependencies..."
	@cd frontend && npm install
	@echo "Setup complete! Please update .env and backend/config.yaml with your Jira credentials."

# Run development servers
dev:
	@echo "Starting development servers..."
	@trap 'kill %1; kill %2' INT; \
	cd backend && go run cmd/server/main.go & \
	cd frontend && npm start & \
	wait

# Build both backend and frontend
build: build-backend build-frontend

build-backend:
	@echo "Building backend..."
	@cd backend && go build -o bin/server cmd/server/main.go

build-frontend:
	@echo "Building frontend..."
	@cd frontend && npm run build

# Run tests
test: test-backend test-frontend

test-backend:
	@echo "Running backend tests..."
	@cd backend && go test ./...

test-frontend:
	@echo "Running frontend tests..."
	@cd frontend && npm test -- --watchAll=false

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf backend/bin
	@rm -rf frontend/build
	@rm -rf frontend/node_modules/.cache

# Docker targets
docker-build:
	@echo "Building Docker images..."
	@docker-compose build

docker-run:
	@echo "Starting with Docker Compose..."
	@docker-compose up -d

docker-stop:
	@echo "Stopping Docker containers..."
	@docker-compose down

docker-logs:
	@echo "Showing Docker logs..."
	@docker-compose logs -f

# Development helpers
backend-run:
	@echo "Running backend only..."
	@cd backend && go run cmd/server/main.go

frontend-run:
	@echo "Running frontend only..."
	@cd frontend && npm start

# Linting and formatting
lint:
	@echo "Running linters..."
	@cd backend && go vet ./...
	@cd backend && go fmt ./...
	@cd frontend && npm run lint --if-present

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Database/Migration helpers (if needed in future)
migrate-up:
	@echo "Running migrations..."
	@echo "No migrations defined yet"

migrate-down:
	@echo "Rolling back migrations..."
	@echo "No migrations defined yet"

# Production deployment helpers
deploy-staging:
	@echo "Deploying to staging..."
	@echo "Deployment scripts not implemented yet"

deploy-production:
	@echo "Deploying to production..."
	@echo "Deployment scripts not implemented yet"
