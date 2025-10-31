# Kubernetes Deployment Quick Start

This guide will help you deploy the PR CLI webhook service to Kubernetes in under 5 minutes.

## Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured
- GitHub or GitLab API token
- Webhook secret (generate with: `openssl rand -hex 32`)

## Quick Deploy to Development

### 1. Create Secrets

```bash
# Generate a webhook secret
WEBHOOK_SECRET=$(openssl rand -hex 32)

# Set your GitHub token
GITHUB_TOKEN="ghp_your_token_here"

# Create namespace
kubectl create namespace pr-cli-dev

# Create secret
kubectl create secret generic pr-cli-secrets \
  --namespace=pr-cli-dev \
  --from-literal=WEBHOOK_SECRET="$WEBHOOK_SECRET" \
  --from-literal=PR_TOKEN="$GITHUB_TOKEN"
```

**PS: Update these values in the target overlay:** `deploy/overlay`, i.e development will have a `dev-` prefix and will be overwritten.

### 2. Deploy


```bash
# Using kubectl
kubectl apply -k deploy/overlays/development

# OR using Make
make k8s-deploy-dev

# OR using the deploy script
./deploy/scripts/deploy.sh development
```

### 3. Verify

```bash
# Check pods
kubectl get pods -n pr-cli-dev

# Check logs
kubectl logs -n pr-cli-dev -l app.kubernetes.io/name=pr-cli -f

# Port forward to test locally
kubectl port-forward -n pr-cli-dev svc/dev-pr-cli-webhook 8080:80

# Test health endpoint
curl http://localhost:8080/health
```

### 4. Configure GitHub Webhook

```bash
# Get the ingress URL (if configured)
kubectl get ingress -n pr-cli-dev

# Or use port-forward for testing
# In GitHub repo settings → Webhooks → Add webhook:
# - Payload URL: http://your-ingress-url/webhook
# - Content type: application/json
# - Secret: <your WEBHOOK_SECRET>
# - Events: Issue comments
```

## Quick Deploy to Production

### 1. Update Image Tag

Edit `deploy/overlays/production/kustomization.yaml`:

```yaml
images:
  - name: pr-cli
    newName: your-registry/pr-cli
    newTag: v1.0.0  # Use your actual version
```

### 2. Configure Secrets

**Option A: Using kubectl**

```bash
kubectl create namespace pr-cli

kubectl create secret generic pr-cli-secrets \
  --namespace=pr-cli \
  --from-literal=WEBHOOK_SECRET="$WEBHOOK_SECRET" \
  --from-literal=PR_TOKEN="$GITHUB_TOKEN" \
  --from-literal=ALLOWED_REPOS="myorg/*"
```

**Option B: Using External Secrets (Recommended)**

See `deploy/examples/external-secret.yaml`

### 3. Update Ingress

Edit `deploy/overlays/production/ingress.yaml`:

```yaml
spec:
  tls:
  - hosts:
    - pr-cli.your-domain.com  # Update this
    secretName: pr-cli-tls
  rules:
  - host: pr-cli.your-domain.com  # Update this
```

### 4. Deploy

```bash
# Review manifests first
kubectl kustomize deploy/overlays/production

# Deploy
kubectl apply -k deploy/overlays/production

# OR using Make
make k8s-deploy-prod

# Watch rollout
kubectl rollout status deployment -n pr-cli prod-pr-cli-webhook
```

### 5. Verify

```bash
# Check all resources
kubectl get all -n pr-cli

# Check HPA
kubectl get hpa -n pr-cli

# Check PDB
kubectl get pdb -n pr-cli

# View logs
kubectl logs -n pr-cli -l app.kubernetes.io/name=pr-cli --tail=100 -f

# Check metrics
kubectl port-forward -n pr-cli svc/prod-pr-cli-webhook 8080:80
curl http://localhost:8080/metrics
```

## Common Customizations

### Change Worker Count

Edit your overlay's `kustomization.yaml`:

```yaml
configMapGenerator:
  - name: pr-cli-config
    behavior: merge
    literals:
      - WORKER_COUNT=20
      - QUEUE_SIZE=200
```

### Change Resource Limits

Edit your overlay's `deployment-patch.yaml`:

```yaml
spec:
  template:
    spec:
      containers:
      - name: pr-cli
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 1000m
            memory: 1Gi
```

### Add Repository Allowlist

Add to your secret:

```bash
kubectl create secret generic pr-cli-secrets \
  --namespace=pr-cli \
  --from-literal=WEBHOOK_SECRET="$WEBHOOK_SECRET" \
  --from-literal=PR_TOKEN="$GITHUB_TOKEN" \
  --from-literal=ALLOWED_REPOS="myorg/*,anotherorg/specific-repo" \
  --dry-run=client -o yaml | kubectl apply -f -
```

## Makefile Commands

```bash
# Build manifests
make k8s-build ENVIRONMENT=development

# Deploy to development
make k8s-deploy-dev

# Deploy to staging
make k8s-deploy-staging

# Deploy to production
make k8s-deploy-prod

# Dry run (show manifests without applying)
make k8s-dry-run ENVIRONMENT=production

# Check status
make k8s-status ENVIRONMENT=production

# View logs
make k8s-logs ENVIRONMENT=production

# Delete deployment
make k8s-delete ENVIRONMENT=development
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl describe pod -n pr-cli-dev <pod-name>

# Common issues:
# 1. Image pull error - check image name/tag
# 2. Secret not found - create pr-cli-secrets
# 3. ConfigMap not found - check kustomization
```

### Webhook Not Working

```bash
# Check logs for errors
kubectl logs -n pr-cli-dev -l app.kubernetes.io/name=pr-cli --tail=100

# Check service
kubectl get svc -n pr-cli-dev

# Check ingress
kubectl get ingress -n pr-cli-dev
kubectl describe ingress -n pr-cli-dev dev-pr-cli-webhook

# Test locally
kubectl port-forward -n pr-cli-dev svc/dev-pr-cli-webhook 8080:80
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'
```

### High Memory Usage

```bash
# Check resource usage
kubectl top pods -n pr-cli

# Reduce worker count
kubectl edit configmap -n pr-cli dev-pr-cli-config
# Change WORKER_COUNT and QUEUE_SIZE

# Restart pods
kubectl rollout restart deployment -n pr-cli dev-pr-cli-webhook
```

## Next Steps

- Configure monitoring with Prometheus/Grafana
- Set up alerts for high queue size or error rates
- Configure GitOps with ArgoCD or Flux
- Set up external secret management
- Configure TLS certificates with cert-manager
- Review security policies and network policies

## Additional Resources

- [Full Deployment Guide](README.md)
- [Webhook Usage Guide](../docs/webhook-usage.md)
- [Design Document](../docs/webhook-service-design.md)
- [Examples](examples/)

