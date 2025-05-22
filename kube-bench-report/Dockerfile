FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o kube-bench-report

# Use a minimal alpine image for the final stage
FROM alpine:3.19

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/kube-bench-report /app/kube-bench-report

# Create a directory for input/output files
RUN mkdir -p /data

# Set the entrypoint
ENTRYPOINT ["/app/kube-bench-report"]

# Default command
CMD ["--help"]
