# Development Guide

This guide provides detailed instructions for setting up and developing the Roadmap Planner application.

## Prerequisites

- Go 1.24.6 or later
- Node.js 18 or later
- Docker and Docker Compose (optional)
- Access to a Jira instance
- Jira credentials (username and API token)

## Quick Start

1. **Clone and setup**:
   ```bash
   git clone <repository-url>
   cd roadmap-planner
   make setup
   ```

2. **Configure credentials**:
   - Edit `.env` file with your Jira credentials
   - Edit `backend/config.yaml` if needed

3. **Start development servers**:
   ```bash
   make dev
   ```

4. **Access the application**:
   - Frontend: http://localhost:3000
   - Backend API: http://localhost:8080

## Project Structure

```
roadmap-planner/
├── backend/                 # Go backend application
│   ├── cmd/server/         # Main application entry point
│   ├── internal/           # Internal packages
│   │   ├── api/           # REST API handlers and routes
│   │   ├── jira/          # Jira client and integration
│   │   ├── models/        # Data models
│   │   └── config/        # Configuration management
│   ├── config.yaml        # Configuration file
│   ├── Dockerfile         # Backend Docker image
│   ├── go.mod             # Go module definition
│   └── go.sum             # Go module checksums
├── frontend/               # React frontend application
│   ├── src/               # Source code
│   │   ├── components/    # React components
│   │   ├── services/      # API service layer
│   │   └── hooks/         # Custom React hooks
│   ├── public/            # Static assets
│   ├── Dockerfile         # Frontend Docker image
│   ├── nginx.conf         # Nginx configuration
│   └── package.json       # Node.js dependencies
├── docker-compose.yml     # Docker composition
├── Makefile              # Development automation
└── README.md             # Project documentation
```

## Development Workflow

### Backend Development

1. **Install dependencies**:
   ```bash
   cd backend
   go mod tidy
   ```

2. **Run the server**:
   ```bash
   go run cmd/server/main.go
   ```

3. **Run tests**:
   ```bash
   go test ./...
   ```

4. **Build**:
   ```bash
   go build -o bin/server cmd/server/main.go
   ```

### Frontend Development

1. **Install dependencies**:
   ```bash
   cd frontend
   npm install
   ```

2. **Start development server**:
   ```bash
   npm start
   ```

3. **Run tests**:
   ```bash
   npm test
   ```

4. **Build for production**:
   ```bash
   npm run build
   ```

## Configuration

### Environment Variables

The application uses environment variables for configuration:

- `JIRA_BASE_URL`: Your Jira instance URL
- `JIRA_USERNAME`: Your Jira username
- `JIRA_PASSWORD`: Your Jira API token
- `JIRA_PROJECT`: Jira project key (default: DEVOPS)
- `SERVER_PORT`: Backend server port (default: 8080)
- `DEBUG`: Enable debug mode (default: false)

### Configuration File

The backend also supports YAML configuration in `backend/config.yaml`:

```yaml
debug: false
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

## API Documentation

### Authentication

The API uses Jira credentials passed via headers:
- `X-Jira-Username`: Jira username
- `X-Jira-Password`: Jira password/API token
- `X-Jira-BaseURL`: Jira instance URL
- `X-Jira-Project`: Jira project key

### Endpoints

- `POST /api/auth/login` - Authenticate with Jira
- `GET /api/roadmap` - Get complete roadmap data
- `POST /api/milestones` - Create new milestone
- `POST /api/epics` - Create new epic
- `PUT /api/epics/:id/milestone` - Move epic to different milestone
- `GET /api/components/:name/versions` - Get component versions
- `GET /api/users/assignable?issueKey=<key>` - Get assignable users for an issue

## Testing

### Backend Tests

```bash
cd backend
go test ./...
```

### Frontend Tests

```bash
cd frontend
npm test
```

### Integration Tests

Integration tests require a test Jira instance. Configure test credentials and run:

```bash
make test
```

## Docker Development

### Build and run with Docker Compose

```bash
make docker-build
make docker-run
```

### View logs

```bash
make docker-logs
```

### Stop containers

```bash
make docker-stop
```

## Troubleshooting

### Common Issues

1. **Jira Authentication Fails**:
   - Verify your Jira URL is correct
   - Use API token instead of password
   - Check user permissions for the DEVOPS project

2. **CORS Errors**:
   - Ensure frontend URL is in `allowed_origins`
   - Check that backend is running on correct port

3. **Build Failures**:
   - Run `go mod tidy` in backend directory
   - Run `npm install` in frontend directory
   - Check Go and Node.js versions

### Debug Mode

Enable debug mode by setting `DEBUG=true` or updating config.yaml:

```yaml
debug: true
```

This will:
- Enable detailed logging
- Show SQL queries (if database is added)
- Provide more verbose error messages

## Contributing

1. Create a feature branch
2. Make your changes
3. Add tests for new functionality
4. Run the test suite
5. Submit a pull request

### Code Style

- **Go**: Follow standard Go formatting (`go fmt`)
- **JavaScript**: Use Prettier for formatting
- **Commit messages**: Use conventional commit format

### Adding New Features

1. **Backend**: Add handlers in `internal/api/handlers/`
2. **Frontend**: Add components in `src/components/`
3. **Models**: Update `internal/models/` for new data structures
4. **Tests**: Add corresponding test files

## Deployment

### Production Build

```bash
make build
```

### Docker Production

```bash
docker-compose -f docker-compose.prod.yml up -d
```

### Environment Setup

For production deployment:

1. Set up proper environment variables
2. Configure HTTPS/TLS
3. Set up monitoring and logging
4. Configure backup strategies
5. Set up CI/CD pipelines

## Performance Considerations

- **Caching**: Implement Redis for production caching
- **Rate Limiting**: Add rate limiting for Jira API calls
- **Database**: Consider adding database for local caching
- **Monitoring**: Add application performance monitoring

## Security

- Use API tokens instead of passwords
- Implement proper session management
- Add input validation and sanitization
- Use HTTPS in production
- Regular security updates
