# Roadmap Planner

A web-based Kanban application for managing product roadmaps using direct Jira integration. The application displays roadmap items organized by Pillars (vertical lanes) and Quarters (horizontal lanes), enabling Product Managers to visualize and manage their product roadmap efficiently.

## Features

- **Kanban Board**: Visual roadmap organized by pillars and quarters
- **Jira Integration**: Direct integration with Jira for real-time data
- **Milestone Management**: Create and manage milestones within quarters
- **Epic Management**: Create epics and link them to milestones
- **Drag & Drop**: Move epics between milestones easily
- **Component Versioning**: Filter and manage component versions

## Architecture

### Technology Stack
- **Backend**: Go with Gin framework
- **Frontend**: React with drag-and-drop support
- **Integration**: Jira REST API v8.4
- **Authentication**: Jira Basic Auth
- **Deployment**: Unified Docker container (backend serves frontend)

### Project Structure
```
roadmap-planner/
├── backend/                 # Go backend application
│   ├── cmd/server/         # Main application entry point
│   ├── internal/           # Internal packages
│   │   ├── api/           # REST API handlers and routes
│   │   ├── jira/          # Jira client and integration
│   │   ├── models/        # Data models
│   │   └── config/        # Configuration management
│   ├── go.mod             # Go module definition
│   └── go.sum             # Go module checksums
├── frontend/               # React frontend application
│   ├── src/               # Source code
│   │   ├── components/    # React components
│   │   ├── services/      # API service layer
│   │   └── hooks/         # Custom React hooks
│   ├── public/            # Static assets
│   └── package.json       # Node.js dependencies
└── docker-compose.yml     # Docker composition
```

## Getting Started

### Prerequisites
- Go 1.24.6 or later
- Node.js 18 or later
- Access to a Jira instance
- Jira credentials (username and API token)

### Quick Setup

1. **Clone and setup**:
   ```bash
   git clone <repository-url>
   cd roadmap-planner
   make setup
   ```

2. **Configure credentials**:
   Edit `.env` file with your Jira credentials:
   ```bash
   JIRA_BASE_URL=https://your-jira-instance.atlassian.net
   JIRA_USERNAME=your-username
   JIRA_PASSWORD=your-api-token
   ```

3. **Start development servers** (separate processes):
   ```bash
   make dev
   ```

4. **Access the application**:
   - Development Frontend: http://localhost:3000
   - Development Backend API: http://localhost:8080

### Production Deployment

For production, the application runs as a single container where the backend serves both the API and frontend files:

```bash
# Build unified production image
make build-unified

# Run production container
make docker-run
```

In production mode:
- **Application**: http://localhost:8080 (single endpoint for everything)
- No CORS issues as frontend and backend are served from the same origin

### Alternative Setup Methods

#### Using Docker Compose (Production)
```bash
# Set environment variables
cp .env.example .env
# Edit .env with your Jira credentials

# Start the unified application
make docker-run
# Access at: http://localhost:8080
```

#### Using Docker Compose (Development)
```bash
# Set environment variables
cp .env.example .env
# Edit .env with your Jira credentials

# Start separate development containers
make docker-run-dev
# Frontend: http://localhost:3000
# Backend: http://localhost:8080
```

#### Manual Setup

**Backend:**
```bash
cd backend
go mod tidy
go run cmd/server/main.go
```

**Frontend:**
```bash
cd frontend
npm install
npm start
```

## Configuration

### Backend Configuration
The backend can be configured using environment variables or a YAML configuration file:

```yaml
# config.yaml
jira:
  base_url: "https://your-jira-instance.atlassian.net"
  username: "your-username"
  password: "your-api-token"
  project: "DEVOPS"

server:
  port: 8080
  cors:
    allowed_origins: ["http://localhost:3000"]

cache:
  ttl: "5m"
  refresh_interval: "1m"
```

## API Endpoints

- `POST /api/auth/login` - Authenticate with Jira
- `POST /api/auth/logout` - Clear session
- `GET /api/roadmap` - Get complete roadmap data
- `GET /api/pillars` - Get all pillars
- `POST /api/milestones` - Create new milestone
- `PUT /api/epics/:id/milestone` - Move epic to different milestone
- `POST /api/epics` - Create new epic
- `GET /api/components/:name/versions` - Get versions for component

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for detailed development instructions.

### Quick Commands

```bash
make help             # Show all available commands
make setup            # Set up development environment
make dev              # Run development servers (separate)
make test             # Run all tests
make build            # Build for local development
make build-unified    # Build unified production Docker image
make docker-run       # Run production (unified container)
make docker-run-dev   # Run development (separate containers)
```

### Testing
```bash
# All tests
make test

# Backend only
cd backend && go test ./...

# Frontend only
cd frontend && npm test

# Local CI testing (mimics GitHub Actions)
./ci-test.sh              # Run all tests
./ci-test.sh backend      # Backend tests only
./ci-test.sh frontend     # Frontend tests only
./ci-test.sh docker       # Docker tests only
```

## CI/CD Pipeline

The project uses GitHub Actions for continuous integration with the following checks:

### Automated Testing
- **Backend**: Go unit tests, linting (golangci-lint), formatting, security scanning
- **Frontend**: Jest tests, ESLint, npm audit, build verification
- **Integration**: Docker build testing, health checks, API endpoint validation
- **Security**: Trivy vulnerability scanning for filesystem and Docker images

### Code Quality
- **Coverage**: Minimum 60% test coverage requirement
- **Linting**: Comprehensive linting for both Go and JavaScript/React
- **Formatting**: Automated code formatting checks
- **Dependencies**: Security vulnerability scanning

### Triggers
The CI pipeline runs when:
- Push to `main` or `develop` branches (only if roadmap-planner files change)
- Pull requests targeting `main` or `develop` branches
- Changes to the CI workflow file itself

### Local Testing
Use the provided script to run the same checks locally:
```bash
./ci-test.sh --help  # See available options
```

## API Documentation

See [API.md](API.md) for complete API documentation.

### Key Endpoints
- `GET /api/roadmap` - Get complete roadmap data
- `POST /api/milestones` - Create new milestone
- `POST /api/epics` - Create new epic
- `PUT /api/epics/:id/milestone` - Move epic between milestones

## Jira Integration

The application integrates directly with Jira using the following structure:

- **Pillars**: High-level parent issues in the DEVOPS project
- **Milestones**: Sub-tasks of pillars with quarter custom field
- **Epics**: Issues linked to milestones via "blocks" relationship

### Required Jira Setup

1. DEVOPS project with appropriate permissions
2. Custom field for quarters (or use description fallback)
3. "Blocks" issue link type enabled
4. API token for authentication

## Troubleshooting

### Common Issues

1. **Authentication fails**: Verify Jira URL and use API token
2. **CORS errors**: Check allowed origins in configuration
3. **No data shown**: Verify project permissions and issue structure

### Debug Mode

Enable debug logging:
```bash
DEBUG=true make dev
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Follow the development guide
4. Add tests for new functionality
5. Submit a pull request

## License

Licensed under the Apache License, Version 2.0. See LICENSE file for details.
