# PR CLI Webhook Service - Usage Guide

## Overview

The PR CLI webhook service allows you to receive webhooks directly from GitHub or GitLab and process PR comment commands without the need for Tekton Pipelines. This provides faster response times and lower resource usage.

## Quick Start

### 1. Start the Server

```bash
# Basic usage with environment variables
export PR_TOKEN="your-github-token"
export WEBHOOK_SECRET="your-webhook-secret"
./pr-cli serve
```

### 2. Configure GitHub Webhook

1. Go to your repository settings → Webhooks → Add webhook
2. Set Payload URL to: `https://your-server.com/webhook`
3. Set Content type to: `application/json`
4. Set Secret to: `your-webhook-secret` (same as WEBHOOK_SECRET)
5. Select "Let me select individual events" and choose:
   - Issue comments (for PR comment commands like `/lgtm`, `/merge`)
   - Pull requests (if using PR event handling to trigger workflows)
6. Save the webhook

### 3. Test the Webhook

Create a comment on a pull request with a command like `/lgtm` and the server will process it.

## Configuration

### Environment Variables

#### Server Configuration
- `LISTEN_ADDR` - Server listen address (default: `:8080`)
- `WEBHOOK_PATH` - Webhook endpoint path (default: `/webhook`)
- `HEALTH_PATH` - Health check endpoint path (default: `/health`)
- `METRICS_PATH` - Metrics endpoint path (default: `/metrics`)

#### Security
- `WEBHOOK_SECRET` - Webhook secret for signature validation
- `WEBHOOK_SECRET_FILE` - File containing webhook secret (for Kubernetes secrets)
- `ALLOWED_REPOS` - Comma-separated list of allowed repositories (e.g., `myorg/repo1,myorg/repo2,myorg/*`)
- `REQUIRE_SIGNATURE` - Require webhook signature validation (default: `true`)

#### TLS
- `TLS_ENABLED` - Enable TLS (default: `false`)
- `TLS_CERT_FILE` - TLS certificate file path
- `TLS_KEY_FILE` - TLS private key file path

#### Processing
- `ASYNC_PROCESSING` - Process webhooks asynchronously (default: `true`)
- `WORKER_COUNT` - Number of worker goroutines (default: `10`)
- `QUEUE_SIZE` - Job queue size (default: `100`)

#### Rate Limiting
- `RATE_LIMIT_ENABLED` - Enable rate limiting (default: `true`)
- `RATE_LIMIT_REQUESTS` - Max requests per minute per IP (default: `100`)

#### Pull Request Event Handling
- `PR_EVENT_ENABLED` - Enable pull_request event handling to trigger workflows (default: `false`)
- `PR_EVENT_ACTIONS` - Comma-separated PR actions to listen for (default: `opened,synchronize,reopened,ready_for_review,edited`)
- `WORKFLOW_FILE` - Workflow file to trigger for PR events (e.g., `.github/workflows/pr-check.yml`)
- `WORKFLOW_REPO` - Repository to trigger workflow file. If empty will use the same as event (e.g alaudadevops/toolbox)
- `WORKFLOW_REF` - Git ref to use for workflow dispatch (default: `main`)
- `WORKFLOW_INPUTS` - Static workflow inputs in `key=value,key=value` format

#### PR CLI Configuration
All standard PR CLI environment variables are supported:
- `PR_TOKEN` - GitHub/GitLab API token (required)
- `PR_PLATFORM` - Platform: `github` or `gitlab` (default: `github`)
- `PR_BASE_URL` - API base URL (optional)
- `PR_COMMENT_TOKEN` - Token for posting comments (optional, defaults to PR_TOKEN)
- `PR_LGTM_THRESHOLD` - Minimum LGTM approvals (default: `1`)
- `PR_LGTM_PERMISSIONS` - Required permissions for LGTM
- `PR_MERGE_METHOD` - Merge method: `auto`, `merge`, `squash`, `rebase` (default: `auto`)
- `PR_ROBOT_ACCOUNTS` - Comma-separated list of robot accounts to exclude from LGTM
- `PR_VERBOSE` - Enable verbose logging (default: `false`)

### Command-Line Flags

All environment variables can also be set via command-line flags:

```bash
./pr-cli serve \
  --listen-addr=:8080 \
  --webhook-secret=mysecret \
  --allowed-repos="myorg/*" \
  --token="github-token" \
  --verbose
```

## Endpoints

### Webhook Endpoint
- **Path**: `/webhook` (configurable)
- **Method**: POST
- **Description**: Receives webhooks from GitHub/GitLab

### Health Check
- **Path**: `/health` (configurable)
- **Method**: GET
- **Response**:
  ```json
  {
    "status": "healthy",
    "version": "dev",
    "uptime": "1h23m45s",
    "queue_size": 5
  }
  ```

### Readiness Check
- **Path**: `/health/ready` (configurable)
- **Method**: GET
- **Description**: Returns 200 if server is ready to accept requests (queue not full)

### Metrics
- **Path**: `/metrics` (configurable)
- **Method**: GET
- **Description**: Prometheus metrics endpoint

## Metrics

The webhook service exposes the following Prometheus metrics:

- `pr_cli_webhook_requests_total` - Total webhook requests (labels: platform, event_type, status)
- `pr_cli_webhook_processing_duration_seconds` - Webhook processing duration (labels: platform, command)
- `pr_cli_command_execution_total` - Total command executions (labels: platform, command, status)
- `pr_cli_queue_size` - Current job queue size
- `pr_cli_active_workers` - Number of active workers
- `pr_cli_pr_event_total` - Total pull_request events processed (labels: platform, action, status)
- `pr_cli_workflow_dispatch_total` - Total workflow dispatch triggers (labels: platform, workflow, status)

## Security

### Signature Validation

The webhook service validates webhook signatures to ensure requests are authentic:

- **GitHub**: Validates `X-Hub-Signature-256` header using HMAC-SHA256
- **GitLab**: Validates `X-Gitlab-Token` header using simple token comparison

To disable signature validation (not recommended for production):
```bash
./pr-cli serve --require-signature=false
```

### Repository Allowlist

Restrict which repositories can trigger commands:

```bash
# Allow specific repositories
export ALLOWED_REPOS="myorg/repo1,myorg/repo2"

# Allow all repositories in an organization
export ALLOWED_REPOS="myorg/*"

# Allow multiple organizations
export ALLOWED_REPOS="org1/*,org2/*,org3/specific-repo"
```

If `ALLOWED_REPOS` is empty or not set, all repositories are allowed.

### TLS

Enable TLS for secure communication:

```bash
./pr-cli serve \
  --tls-enabled \
  --tls-cert-file=/etc/certs/tls.crt \
  --tls-key-file=/etc/certs/tls.key
```

## Deployment

### Docker

```dockerfile
FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN go build -o pr-cli .

FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/pr-cli /pr-cli
ENTRYPOINT ["/pr-cli"]
CMD ["serve"]
```

### Kubernetes

See `docs/webhook-service-design.md` for complete Kubernetes deployment manifests including:
- Deployment
- Service
- Ingress
- ConfigMap
- Secret
- ServiceMonitor (for Prometheus)

### Docker Compose

```yaml
version: '3.8'
services:
  pr-cli:
    image: pr-cli:latest
    command: serve
    ports:
      - "8080:8080"
    environment:
      - PR_TOKEN=${GITHUB_TOKEN}
      - WEBHOOK_SECRET=${WEBHOOK_SECRET}
      - ALLOWED_REPOS=myorg/*
      - PR_VERBOSE=true
    restart: unless-stopped
```

## Monitoring

### Health Checks

```bash
# Check if server is healthy
curl http://localhost:8080/health

# Check if server is ready
curl http://localhost:8080/health/ready
```

### Metrics

```bash
# View Prometheus metrics
curl http://localhost:8080/metrics
```

### Logs

The server outputs structured JSON logs:

```json
{
  "level": "info",
  "msg": "Processing webhook job",
  "platform": "github",
  "repo": "myorg/myrepo",
  "pr": 123,
  "command": "/lgtm",
  "sender": "username",
  "time": "2025-10-31T12:00:00Z"
}
```

## Troubleshooting

### Webhook Not Received

1. Check GitHub/GitLab webhook delivery logs
2. Verify firewall/network configuration
3. Check server logs for errors
4. Verify webhook secret matches

### Signature Validation Fails

1. Verify `WEBHOOK_SECRET` matches the secret configured in GitHub/GitLab
2. Check that the webhook is sending the correct signature header
3. For debugging, temporarily disable signature validation (not recommended for production)

### Commands Not Executing

1. Check repository allowlist configuration
2. Verify PR_TOKEN has correct permissions
3. Check server logs for detailed error messages
4. Ensure comment starts with `/` and is a valid command

### Queue Full

If you see "queue full" errors:
1. Increase `QUEUE_SIZE`
2. Increase `WORKER_COUNT`
3. Check if workers are stuck (review logs)

## Examples

### Basic GitHub Setup

```bash
export PR_TOKEN="ghp_xxxxxxxxxxxx"
export WEBHOOK_SECRET="my-secret-key"
export ALLOWED_REPOS="myorg/*"
./pr-cli serve --verbose
```

### Production Setup with TLS

```bash
./pr-cli serve \
  --listen-addr=:8443 \
  --tls-enabled \
  --tls-cert-file=/etc/certs/tls.crt \
  --tls-key-file=/etc/certs/tls.key \
  --webhook-secret-file=/etc/secrets/webhook-secret \
  --allowed-repos="myorg/*" \
  --token-file=/etc/secrets/github-token \
  --worker-count=20 \
  --queue-size=200
```

### PR Event Handling (Trigger Workflows)

Enable the webhook to trigger GitHub Actions workflows when PRs are opened or updated:

```bash
export PR_TOKEN="ghp_xxxxxxxxxxxx"
export WEBHOOK_SECRET="my-secret-key"
export ALLOWED_REPOS="myorg/*"
export PR_EVENT_ENABLED="true"
export WORKFLOW_FILE=".github/workflows/pr-check.yml"
export WORKFLOW_REF="main"
./pr-cli serve --verbose
```

Or with CLI flags:

```bash
./pr-cli serve \
  --pr-event-enabled \
  --workflow-file=.github/workflows/pr-check.yml \
  --workflow-ref=main \
  --pr-event-actions=opened,synchronize,reopened \
  --allowed-repos="myorg/*"
```

The triggered workflow will receive these inputs:
- `pr_number` - Pull request number
- `pr_action` - The action (opened, synchronize, etc.)
- `head_ref` - Source branch name
- `head_sha` - Head commit SHA
- `base_ref` - Target branch name
- `sender` - User who triggered the event

Example workflow that can be triggered:

```yaml
# .github/workflows/pr-check.yml
name: PR Check
on:
  workflow_dispatch:
    inputs:
      pr_number:
        description: 'Pull request number'
        required: true
      pr_action:
        description: 'PR action'
        required: true
      head_ref:
        description: 'Head branch'
        required: true
      head_sha:
        description: 'Head SHA'
        required: true

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ inputs.head_sha }}
      - name: Run checks
        run: |
          echo "Checking PR #${{ inputs.pr_number }}"
          echo "Action: ${{ inputs.pr_action }}"
          # Add your checks here
```

### GitLab Setup

```bash
export PR_PLATFORM="gitlab"
export PR_TOKEN="glpat-xxxxxxxxxxxx"
export PR_BASE_URL="https://gitlab.example.com"
export WEBHOOK_SECRET="my-gitlab-secret"
./pr-cli serve
```

## Migration from Tekton

If you're currently using PR CLI with Tekton Pipelines:

1. Deploy the webhook service alongside your existing Tekton setup
2. Configure webhooks to point to the new service
3. Test with a subset of repositories first
4. Gradually migrate all repositories
5. Decommission Tekton Pipelines once migration is complete

See `docs/webhook-service-design.md` for detailed migration guide.
