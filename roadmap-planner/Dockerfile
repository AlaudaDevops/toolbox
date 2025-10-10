# Multi-stage build for roadmap planner
# Stage 1: Build frontend
ARG NODE_IMAGE=node:24-alpine
ARG GO_IMAGE=golang:1.25.2-alpine
ARG RUNTIME_IMAGE=alpine:latest
FROM ${NODE_IMAGE} AS frontend-builder

WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy frontend source
COPY frontend/ ./

# Build the frontend
RUN npm run build

ARG GO_IMAGE=golang:1.25-alpine
# Stage 2: Build backend
FROM ${GO_IMAGE} AS backend-builder

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy backend go mod files
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source
COPY backend/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

ARG RUNTIME_IMAGE=alpine:latest
# Stage 3: Final production image
FROM ${RUNTIME_IMAGE}

# Install ca-certificates for HTTPS requests and curl for health checks
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy the backend binary
COPY --from=backend-builder /app/main .

# Copy the frontend build files
COPY --from=frontend-builder /app/frontend/build ./frontend/build

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Set environment variable for static files path
ENV STATIC_FILES_PATH=/app/frontend/build

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application
CMD ["./main"]
