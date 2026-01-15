# Build stage
FROM registry.alauda.cn:60070/devops/nonroot/builder-go:latest AS builder

# Build the binary with version information
ARG VERSION=dev
ARG GIT_COMMIT
ARG BUILD_DATE

# Set working directory
WORKDIR /app

# Copy source code
COPY . .

ENV GOMAXPROCS=4 \
	GOFLAGS=-buildvcs=false \
	CGO_ENABLED=0

# Build with version injection (matching Makefile LDFLAGS)
RUN if [ -z "$BUILD_DATE" ]; then export BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ"); fi && \
    if [ -z "$GIT_COMMIT" ]; then export GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); fi && \
    go build -trimpath \
    -ldflags "-w -s \
               -X github.com/AlaudaDevops/toolbox/pr-cli/internal/version.Version=${VERSION} \
               -X github.com/AlaudaDevops/toolbox/pr-cli/internal/version.GitCommit=${GIT_COMMIT} \
               -X github.com/AlaudaDevops/toolbox/pr-cli/internal/version.BuildDate=${BUILD_DATE}" \
    -tags disable_gcp -o pr-cli .

RUN ./pr-cli version

FROM build-harbor.alauda.cn/ops/alpine:3.21.5-alauda-202512241200

RUN apk add --no-cache \
    bash \
    curl \
    tar \
    git \
    && rm -rf /var/cache/apk/*

RUN deluser --remove-home nonroot ; \
    adduser -u 65532 -h /home/nonroot -D nonroot && \
    chmod a+rx /docker-entrypoint.sh

COPY --from=builder --chown=65532:65532 --chmod=755 /app/pr-cli /usr/local/bin/pr-cli

USER 65532

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/pr-cli"]

# Default command
CMD ["--help"]
