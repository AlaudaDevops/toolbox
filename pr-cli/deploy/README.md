# PR CLI Webhook Service - Kubernetes Deployment

This directory contains Kustomize-based Kubernetes manifests for deploying the PR CLI webhook service.

## Directory Structure

```
deploy/
├── base/                           # Base Kubernetes resources
│   ├── kustomization.yaml         # Base kustomization config
│   ├── namespace.yaml             # Namespace definition
│   ├── serviceaccount.yaml        # Service account
│   ├── deployment.yaml            # Deployment with security best practices
│   ├── service.yaml               # ClusterIP service
│   ├── configmap.yaml             # Configuration
│   ├── secret.yaml                # Secrets (placeholder values)
│   ├── servicemonitor.yaml        # Prometheus ServiceMonitor
│   └── networkpolicy.yaml         # Network policies
├── overlays/
│   ├── development/               # Development environment
│   │   ├── kustomization.yaml
│   │   ├── deployment-patch.yaml
│   │   └── ingress.yaml
│   ├── staging/                   # Staging environment
│   │   ├── kustomization.yaml
│   │   ├── deployment-patch.yaml
│   │   └── ingress.yaml
│   └── production/                # Production environment
│       ├── kustomization.yaml
│       ├── deployment-patch.yaml
│       ├── ingress.yaml
│       ├── hpa.yaml               # Horizontal Pod Autoscaler
│       └── pdb.yaml               # Pod Disruption Budget
└── README.md                      # This file
```

## Prerequisites

- Kubernetes cluster (1.24+)
- kubectl CLI tool
- kustomize (v4.0+) or kubectl with built-in kustomize support
- Ingress controller (nginx recommended)
- cert-manager (for TLS certificates in staging/production)
- Prometheus Operator (optional, for ServiceMonitor)

## Quick Start

### 1. Build and View Manifests

```bash
# View development manifests
kubectl kustomize deploy/overlays/development

# View staging manifests
kubectl kustomize deploy/overlays/staging

# View production manifests
kubectl kustomize deploy/overlays/production
```

### 2. Configure Secrets

**IMPORTANT**: Before deploying, you must configure the secrets with actual values.

#### Option A: Using kubectl (for testing)

```bash
# Create namespace first
kubectl create namespace pr-cli-dev

# Create secret with actual values
kubectl create secret generic pr-cli-secrets \
  --namespace=pr-cli-dev \
  --from-literal=WEBHOOK_SECRET='your-webhook-secret' \
  --from-literal=PR_TOKEN='your-github-token' \
  --dry-run=client -o yaml | kubectl apply -f -
```

#### Option B: Using Kustomize Secret Generator

Create a `.env` file (DO NOT commit this):

```bash
# secrets.env
WEBHOOK_SECRET=your-actual-webhook-secret
PR_TOKEN=ghp_your_actual_github_token
```

Update `kustomization.yaml` in your overlay:

```yaml
secretGenerator:
  - name: pr-cli-secrets
    behavior: replace
    envs:
      - secrets.env
```

#### Option C: Using External Secret Management (Recommended for Production)

Use tools like:
- **Sealed Secrets**: Encrypt secrets in Git
- **External Secrets Operator**: Sync from AWS Secrets Manager, HashiCorp Vault, etc.
- **SOPS**: Encrypt secrets with age or PGP

Example with External Secrets Operator:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: pr-cli-secrets
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: pr-cli-secrets
  data:
  - secretKey: WEBHOOK_SECRET
    remoteRef:
      key: pr-cli/webhook-secret
  - secretKey: PR_TOKEN
    remoteRef:
      key: pr-cli/github-token
```

### 3. Deploy to Development

```bash
# Deploy
kubectl apply -k deploy/overlays/development

# Verify deployment
kubectl get all -n pr-cli-dev

# Check logs
kubectl logs -n pr-cli-dev -l app.kubernetes.io/name=pr-cli -f

# Check health
kubectl port-forward -n pr-cli-dev svc/dev-pr-cli-webhook 8080:80
curl http://localhost:8080/health
```

### 4. Deploy to Staging

```bash
# Update image tag in deploy/overlays/staging/kustomization.yaml
# Then deploy
kubectl apply -k deploy/overlays/staging

# Verify
kubectl get all -n pr-cli-staging
```

### 5. Deploy to Production

```bash
# Update image tag in deploy/overlays/production/kustomization.yaml
# Then deploy
kubectl apply -k deploy/overlays/production

# Verify
kubectl get all -n pr-cli
```

## Configuration

### Environment-Specific Settings

Each overlay can customize:

| Setting | Development | Staging | Production |
|---------|-------------|---------|------------|
| Replicas | 1 | 2 | 3 (HPA: 3-10) |
| CPU Request | 50m | 100m | 200m |
| Memory Request | 64Mi | 128Mi | 256Mi |
| Worker Count | 5 | 15 | 20 |
| Queue Size | 50 | 150 | 200 |
| Verbose Logging | true | true | false |

### Customizing Configuration

Edit the ConfigMap in your overlay's `kustomization.yaml`:

```yaml
configMapGenerator:
  - name: pr-cli-config
    behavior: merge
    literals:
      - WORKER_COUNT=25
      - QUEUE_SIZE=250
      - ALLOWED_REPOS=myorg/*
      - PR_LGTM_THRESHOLD=2
```

### Customizing Resources

Edit the deployment patch in your overlay:

```yaml
# deployment-patch.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pr-cli-webhook
spec:
  template:
    spec:
      containers:
      - name: pr-cli
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 2Gi
```

## Ingress Configuration

### Development (HTTP)

The development overlay uses HTTP without TLS for easier local testing.

Update the host in `deploy/overlays/development/ingress.yaml`:

```yaml
spec:
  rules:
  - host: pr-cli-dev.your-domain.com
```

### Staging/Production (HTTPS)

Update the host and TLS settings in the respective `ingress.yaml`:

```yaml
spec:
  tls:
  - hosts:
    - pr-cli.your-domain.com
    secretName: pr-cli-tls
  rules:
  - host: pr-cli.your-domain.com
```

The ingress uses cert-manager to automatically provision TLS certificates.

## Monitoring

### Prometheus Metrics

The deployment includes a ServiceMonitor for Prometheus Operator:

```bash
# Check if ServiceMonitor is created
kubectl get servicemonitor -n pr-cli

# View metrics endpoint
kubectl port-forward -n pr-cli svc/prod-pr-cli-webhook 8080:80
curl http://localhost:8080/metrics
```

### Available Metrics

- `webhook_requests_total` - Total webhook requests
- `webhook_processing_duration_seconds` - Processing duration
- `command_execution_total` - Command executions
- `webhook_queue_size` - Current queue size
- `webhook_active_workers` - Active workers

### Grafana Dashboard

Import the Grafana dashboard from `docs/grafana-dashboard.json` (if available) or create custom dashboards using the metrics above.

## Security

### Security Features Enabled

✅ **Pod Security**:
- Non-root user (UID 65532)
- Read-only root filesystem
- No privilege escalation
- Dropped all capabilities
- Seccomp profile

✅ **Network Security**:
- NetworkPolicy restricting ingress/egress
- TLS for external communication (staging/production)

✅ **Secret Management**:
- Secrets mounted as environment variables
- Support for external secret management

✅ **RBAC**:
- Minimal ServiceAccount with no permissions (automountServiceAccountToken: false)

### Network Policies

The NetworkPolicy allows:
- **Ingress**: From ingress-nginx and monitoring namespaces
- **Egress**: DNS, HTTPS (443), HTTP (80)

Adjust the NetworkPolicy in `base/networkpolicy.yaml` if needed.

## Scaling

### Manual Scaling

```bash
# Scale deployment
kubectl scale deployment -n pr-cli prod-pr-cli-webhook --replicas=5
```

### Horizontal Pod Autoscaler (Production)

Production includes HPA that scales based on CPU/memory:

```bash
# Check HPA status
kubectl get hpa -n pr-cli

# Describe HPA
kubectl describe hpa -n pr-cli prod-pr-cli-webhook
```

HPA Configuration:
- Min replicas: 3
- Max replicas: 10
- Target CPU: 70%
- Target Memory: 80%

## High Availability

### Pod Disruption Budget (Production)

Production includes a PDB to ensure at least 1 pod is always available during disruptions:

```bash
# Check PDB
kubectl get pdb -n pr-cli
```

### Pod Anti-Affinity

The deployment uses preferredDuringSchedulingIgnoredDuringExecution to spread pods across nodes.

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n pr-cli -l app.kubernetes.io/name=pr-cli
```

### View Logs

```bash
# All pods
kubectl logs -n pr-cli -l app.kubernetes.io/name=pr-cli --tail=100 -f

# Specific pod
kubectl logs -n pr-cli <pod-name> -f
```

### Check Events

```bash
kubectl get events -n pr-cli --sort-by='.lastTimestamp'
```

### Debug Pod

```bash
# Exec into pod (limited due to read-only filesystem)
kubectl exec -it -n pr-cli <pod-name> -- /bin/sh

# Port forward for local testing
kubectl port-forward -n pr-cli svc/prod-pr-cli-webhook 8080:80
```

### Common Issues

#### Pods Not Starting

Check:
1. Image pull errors: `kubectl describe pod -n pr-cli <pod-name>`
2. Secret exists: `kubectl get secret -n pr-cli pr-cli-secrets`
3. ConfigMap exists: `kubectl get configmap -n pr-cli pr-cli-config`

#### Webhook Not Receiving Requests

Check:
1. Ingress configuration: `kubectl get ingress -n pr-cli`
2. Service endpoints: `kubectl get endpoints -n pr-cli`
3. NetworkPolicy: `kubectl get networkpolicy -n pr-cli`
4. GitHub/GitLab webhook configuration

#### High Memory Usage

1. Check metrics: `kubectl top pods -n pr-cli`
2. Reduce worker count or queue size in ConfigMap
3. Increase memory limits in deployment patch

## Updating the Deployment

### Rolling Update

```bash
# Update image tag in kustomization.yaml
# Then apply
kubectl apply -k deploy/overlays/production

# Watch rollout
kubectl rollout status deployment -n pr-cli prod-pr-cli-webhook
```

### Rollback

```bash
# Rollback to previous version
kubectl rollout undo deployment -n pr-cli prod-pr-cli-webhook

# Rollback to specific revision
kubectl rollout undo deployment -n pr-cli prod-pr-cli-webhook --to-revision=2

# Check rollout history
kubectl rollout history deployment -n pr-cli prod-pr-cli-webhook
```

## Cleanup

### Delete Deployment

```bash
# Development
kubectl delete -k deploy/overlays/development

# Staging
kubectl delete -k deploy/overlays/staging

# Production
kubectl delete -k deploy/overlays/production
```

## CI/CD Integration

### GitOps with ArgoCD

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: pr-cli-webhook
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/pr-cli
    targetRevision: main
    path: deploy/overlays/production
  destination:
    server: https://kubernetes.default.svc
    namespace: pr-cli
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### GitOps with Flux

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: pr-cli-webhook
  namespace: flux-system
spec:
  interval: 5m
  path: ./deploy/overlays/production
  prune: true
  sourceRef:
    kind: GitRepository
    name: pr-cli
```

## Best Practices

1. **Never commit secrets** - Use external secret management
2. **Use specific image tags** - Avoid `latest` in production
3. **Test in development first** - Always test changes in dev before production
4. **Monitor metrics** - Set up alerts for high queue size, error rates
5. **Regular updates** - Keep dependencies and base images updated
6. **Backup configurations** - Store kustomize overlays in version control
7. **Use GitOps** - Automate deployments with ArgoCD or Flux

## Additional Resources

- [Kustomize Documentation](https://kustomize.io/)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [PR CLI Webhook Usage Guide](../docs/webhook-usage.md)
- [PR CLI Webhook Design](../docs/webhook-service-design.md)

