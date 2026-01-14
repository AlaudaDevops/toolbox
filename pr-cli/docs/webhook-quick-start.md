# PR CLI Webhook Service - Quick Start Guide

## Overview

This guide provides a quick reference for implementing and deploying the PR CLI webhook service.

## Prerequisites

- Go 1.25+
- Kubernetes cluster (for production deployment)
- GitHub/GitLab repository with admin access
- GitHub Personal Access Token or GitLab Access Token

## Development Setup

### 1. Install Dependencies

```bash
cd pr-cli

# Add new dependencies
go get golang.org/x/time/rate
go get github.com/prometheus/client_golang/prometheus

# Update go.mod
go mod tidy
```

### 2. Run Locally

```bash
# Set required environment variables
export PR_TOKEN="ghp_your_github_token"
export WEBHOOK_SECRET="your-webhook-secret"
export ALLOWED_REPOS="your-org/*"
# enabling PR workflow triggering
export PR_EVENT_ENABLED=true
export WORKFLOW_FILE=kilo-pr-review.yaml # name of the file in the target repository
# export WORKFLOW_REPO=org/repo # org and repo to trigger workflow



# Build and run
make build-local
./pr-cli serve \
  --listen-addr=:8080 \
  --webhook-path=/webhook \
  --require-signature=false \
  --verbose

# In another terminal, test health endpoint
curl http://localhost:8080/health
```

### 3. Test with ngrok

```bash
# Install ngrok
brew install ngrok  # macOS
# or download from https://ngrok.com/

# Expose local server
ngrok http 8080

# Copy the HTTPS URL (e.g., https://abc123.ngrok.io)
# Configure GitHub webhook to use: https://abc123.ngrok.io/webhook
```

### 4. Configure GitHub Webhook

1. Go to your repository → **Settings** → **Webhooks** → **Add webhook**
2. **Payload URL**: `https://abc123.ngrok.io/webhook`
3. **Content type**: `application/json`
4. **Secret**: Same as `WEBHOOK_SECRET` environment variable
5. **Events**: Select "Issue comments" only
6. **Active**: ✓

### 5. Test Webhook

1. Create a test PR in your repository
2. Post a comment: `/help`
3. Check ngrok terminal for incoming webhook
4. Check pr-cli logs for processing
5. Verify bot response in PR comments

## Production Deployment

### 1. Build Docker Image

```bash
# Build with version info
make docker-build VERSION=v1.0.0

# Or use existing Dockerfile
docker build -t your-registry/pr-cli:v1.0.0 .
docker push your-registry/pr-cli:v1.0.0
```

### 2. Create Kubernetes Secrets

```bash
# Create namespace
kubectl create namespace pr-automation

# Create secrets
kubectl create secret generic pr-cli-secrets \
  --from-literal=webhook-secret='your-random-secret' \
  --from-literal=github-token='ghp_xxxxxxxxxxxx' \
  -n pr-automation
```

### 3. Deploy to Kubernetes

```bash
# Apply manifests
kubectl apply -f deploy/kubernetes/rbac.yaml
kubectl apply -f deploy/kubernetes/deployment.yaml
kubectl apply -f deploy/kubernetes/service.yaml
kubectl apply -f deploy/kubernetes/ingress.yaml

# Check deployment
kubectl get pods -n pr-automation
kubectl logs -f deployment/pr-cli-webhook -n pr-automation
```

### 4. Configure Production Webhook

1. Update GitHub webhook URL to production ingress
2. **Payload URL**: `https://pr-webhook.example.com/webhook`
3. Enable SSL verification
4. Test with a PR comment

## Configuration Reference

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LISTEN_ADDR` | Server listen address | `:8080` | No |
| `WEBHOOK_PATH` | Webhook endpoint path | `/webhook` | No |
| `WEBHOOK_SECRET` | Webhook signature secret | - | Yes |
| `REQUIRE_SIGNATURE` | Validate webhook signatures | `true` | No |
| `ALLOWED_REPOS` | Allowed repositories (comma-separated) | `*` | No |
| `PR_TOKEN` | GitHub/GitLab API token | - | Yes |
| `PR_PLATFORM` | Platform (github/gitlab) | `github` | No |
| `ASYNC_PROCESSING` | Enable async processing | `true` | No |
| `WORKER_COUNT` | Number of worker goroutines | `10` | No |
| `QUEUE_SIZE` | Job queue size | `100` | No |
| `RATE_LIMIT_ENABLED` | Enable rate limiting | `true` | No |
| `RATE_LIMIT_REQUESTS` | Max requests per minute per IP | `100` | No |
| `PR_EVENT_ENABLED` | Enable pull_request event handling | `false` | No |
| `PR_EVENT_ACTIONS` | PR actions to listen for | `opened,synchronize,reopened,ready_for_review,edited` | No |
| `WORKFLOW_FILE` | Workflow file to trigger | - | If PR_EVENT_ENABLED |
| `WORKFLOW_REPO` | Repository to trigger workflow file. If empty will use the same as event | - | No |
| `WORKFLOW_REF` | Git ref for workflow dispatch | `main` | No |
| `WORKFLOW_INPUTS` | Static workflow inputs (key=value,key=value) | - | No |

### Command-Line Flags

```bash
pr-cli serve --help

Flags:
  --listen-addr string          Server listen address (default ":8080")
  --webhook-path string         Webhook endpoint path (default "/webhook")
  --health-path string          Health check endpoint path (default "/health")
  --metrics-path string         Metrics endpoint path (default "/metrics")
  --webhook-secret string       Webhook secret for signature validation
  --webhook-secret-file string  File containing webhook secret
  --allowed-repos strings       Allowed repositories (owner/repo format)
  --require-signature           Require webhook signature validation (default true)
  --tls-enabled                 Enable TLS
  --tls-cert-file string        TLS certificate file
  --tls-key-file string         TLS private key file
  --async-processing            Process webhooks asynchronously (default true)
  --worker-count int            Number of worker goroutines (default 10)
  --queue-size int              Job queue size (default 100)
  --rate-limit-enabled          Enable rate limiting (default true)
  --rate-limit-requests int     Max requests per minute per IP (default 100)
  --pr-event-enabled            Enable pull_request event handling (default false)
  --pr-event-actions strings    PR actions to listen for (default [opened,synchronize,reopened,ready_for_review,edited])
  --workflow-file string        Workflow file to trigger (e.g., .github/workflows/pr-check.yml)
  --workflow-repo string        Repository to trigger workflow file. If empty will use the same as event (e.g alaudadevops/toolbox)
  --workflow-ref string         Git ref for workflow dispatch (default "main")
  --workflow-inputs strings     Static workflow inputs (key=value format)
```

## Monitoring

### Health Check

```bash
curl http://localhost:8080/health

# Response:
{
  "status": "healthy",
  "version": "v1.0.0",
  "uptime": "2h30m15s",
  "queue_size": 5,
  "queue_capacity": 100,
  "workers": 10
}
```

### Metrics

```bash
curl http://localhost:8080/metrics

# Prometheus metrics:
# pr_cli_webhook_requests_total{platform="github",event_type="issue_comment",status="success"} 150
# pr_cli_webhook_requests_total{platform="github",event_type="pull_request",status="success"} 75
# pr_cli_webhook_processing_duration_seconds{platform="github",command="lgtm"} 0.234
# pr_cli_command_execution_total{platform="github",command="merge",status="success"} 45
# pr_cli_pr_event_total{platform="github",action="opened",status="success"} 30
# pr_cli_pr_event_total{platform="github",action="synchronize",status="success"} 45
# pr_cli_workflow_dispatch_total{platform="github",workflow=".github/workflows/pr-check.yml",status="success"} 75
# pr_cli_queue_size 5
# pr_cli_active_workers 10
```

### Logs

```bash
# Kubernetes logs
kubectl logs -f deployment/pr-cli-webhook -n pr-automation

# Docker logs
docker logs -f pr-cli-webhook

# Example log entry:
{
  "level": "info",
  "msg": "Processing webhook event",
  "event_id": 987654321,
  "platform": "github",
  "repository": "myorg/myrepo",
  "pr_number": 123,
  "command": "/lgtm",
  "sender": "reviewer",
  "timestamp": "2025-10-31T10:30:00Z"
}
```

## Troubleshooting

### Webhook Not Received

1. Check GitHub webhook delivery status
2. Verify ingress/service configuration
3. Check firewall rules
4. Verify DNS resolution
5. Check server logs for errors

### Signature Validation Failed

1. Verify `WEBHOOK_SECRET` matches GitHub configuration
2. Check webhook payload is not modified by proxy
3. Verify Content-Type is `application/json`
4. Check for clock skew issues

### Commands Not Executing

1. Check `PR_TOKEN` has correct permissions
2. Verify repository is in `ALLOWED_REPOS`
3. Check command syntax in comment
4. Review handler logs for errors
5. Verify PR is in correct state (open)

### High Latency

1. Check queue size: `curl http://localhost:8080/health`
2. Increase worker count if queue is full
3. Check GitHub API rate limits
4. Review resource limits (CPU/memory)
5. Enable async processing if disabled

### Memory Issues

1. Check for memory leaks with profiling
2. Reduce queue size
3. Reduce worker count
4. Increase memory limits
5. Review log retention settings

## Security Checklist

- [ ] Webhook signature validation enabled
- [ ] Strong random webhook secret configured
- [ ] Repository allowlist configured
- [ ] TLS/HTTPS enabled in production
- [ ] Rate limiting enabled
- [ ] GitHub token has minimal required permissions
- [ ] Secrets stored in Kubernetes secrets (not env vars)
- [ ] Network policies configured
- [ ] Audit logging enabled
- [ ] Regular security updates applied

## Performance Tuning

### For High-Volume Repositories

```bash
# Increase workers and queue size
export WORKER_COUNT=20
export QUEUE_SIZE=500

# Adjust rate limits
export RATE_LIMIT_REQUESTS=500

# Enable horizontal autoscaling
kubectl apply -f deploy/kubernetes/hpa.yaml
```

### For Low-Latency Requirements

```bash
# Disable async processing for immediate execution
export ASYNC_PROCESSING=false

# Reduce worker count (not needed for sync mode)
export WORKER_COUNT=1
```

### For Resource-Constrained Environments

```bash
# Reduce workers and queue
export WORKER_COUNT=3
export QUEUE_SIZE=20

# Set resource limits
# See deploy/kubernetes/deployment.yaml
```

## Next Steps

1. Review full design document: [webhook-service-design.md](./webhook-service-design.md)
2. Implement Phase 1 (Foundation)
3. Set up CI/CD pipeline
4. Deploy to staging environment
5. Run load tests
6. Gradual production rollout
7. Monitor and optimize

## Support

- **Documentation**: [docs/](../docs/)
- **Issues**: GitHub Issues
- **Logs**: Check application logs for detailed error messages
- **Metrics**: Use Prometheus/Grafana for monitoring

---

**Last Updated**: 2025-10-31  
**Version**: 1.0
