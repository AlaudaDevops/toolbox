# Unified Architecture Changes

This document outlines the changes made to enable a unified deployment architecture where the backend serves both API endpoints and frontend static files.

## Changes Made

### 1. Backend Static File Serving

Modified `backend/internal/api/routes.go` to:
- Serve static assets from `/static/*filepath`
- Serve PWA files (`manifest.json`, `asset-manifest.json`)
- Use `NoRoute` handler to serve `index.html` for SPA routing
- Handle API routes vs frontend routes appropriately

### 2. Unified Dockerfile

Created `Dockerfile` at project root with multi-stage build:
- **Stage 1**: Build frontend with Node.js
- **Stage 2**: Build backend with Go
- **Stage 3**: Create final Alpine image with both components

### 3. Docker Compose Configuration

**Production (`docker-compose.yml`)**:
- Single `app` service running unified container
- Exposes only port 8080
- Uses `--profile production` for isolation

**Development (`docker-compose.dev.yml`)**:
- Separate `backend` and `frontend` services
- Frontend on port 3000, backend on port 8080
- Maintains existing development workflow

### 4. Frontend API Configuration

Updated `frontend/src/services/api.js`:
- Uses relative URLs in production (empty baseURL)
- Falls back to `localhost:8080` in development
- Eliminates CORS issues in production

### 5. CORS Middleware Updates

Modified `backend/internal/api/middleware/cors.go`:
- Allows same-origin requests (no Origin header)
- Maintains cross-origin support for development

### 6. Build System Updates

Updated `Makefile` with new targets:
- `make build-unified` - Build production Docker image
- `make docker-run` - Run production (unified)
- `make docker-run-dev` - Run development (separate)
- `make docker-build-dev` - Build development images

## Benefits

1. **No CORS Issues**: Frontend and API served from same origin
2. **Simplified Deployment**: Single container for production
3. **Better Performance**: Eliminates cross-origin overhead
4. **Same Dev Experience**: Development workflow unchanged
5. **Security**: Reduced attack surface with single endpoint

## Usage

### Development (Separate Services)
```bash
make dev                    # Local development
make docker-run-dev        # Docker development
```

### Production (Unified Service)
```bash
make build-unified         # Build production image
make docker-run           # Run production container
```

## Environment Variables

- `STATIC_FILES_PATH`: Path to frontend build files (default: `./frontend/build`)
- `SERVER_PORT`: Backend server port (default: `8080`)

## Migration Notes

- Production deployments now use single port (8080)
- No changes needed for existing development workflows
- Frontend build must complete before backend can serve files
- All existing API endpoints remain unchanged
