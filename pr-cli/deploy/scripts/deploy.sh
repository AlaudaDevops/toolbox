#!/usr/bin/env bash
# Deploy PR CLI webhook service to Kubernetes

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
ENVIRONMENT="${1:-development}"
DRY_RUN="${DRY_RUN:-false}"
SKIP_SECRETS="${SKIP_SECRETS:-false}"

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(dirname "$SCRIPT_DIR")"

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl is not installed"
        exit 1
    fi
    
    if ! command -v kustomize &> /dev/null; then
        print_warn "kustomize is not installed, using kubectl kustomize"
    fi
    
    # Check kubectl connection
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    print_info "Prerequisites check passed"
}

# Function to validate environment
validate_environment() {
    local env=$1
    
    if [[ ! -d "$DEPLOY_DIR/overlays/$env" ]]; then
        print_error "Environment '$env' not found"
        print_info "Available environments: development, staging, production"
        exit 1
    fi
}

# Function to check secrets
check_secrets() {
    local env=$1
    local namespace
    
    case $env in
        development)
            namespace="pr-cli-dev"
            ;;
        staging)
            namespace="pr-cli-staging"
            ;;
        production)
            namespace="pr-cli"
            ;;
    esac
    
    print_info "Checking secrets in namespace: $namespace"
    
    # Check if namespace exists
    if ! kubectl get namespace "$namespace" &> /dev/null; then
        print_warn "Namespace $namespace does not exist yet"
        return 0
    fi
    
    # Check if secret exists
    if ! kubectl get secret pr-cli-secrets -n "$namespace" &> /dev/null 2>&1; then
        print_warn "Secret 'pr-cli-secrets' not found in namespace $namespace"
        print_warn "The deployment will use placeholder values from base/secret.yaml"
        print_warn "Make sure to update the secret with actual values before use!"
        
        if [[ "$SKIP_SECRETS" != "true" ]]; then
            read -p "Continue anyway? (y/N) " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                exit 1
            fi
        fi
    else
        print_info "Secret 'pr-cli-secrets' found"
    fi
}

# Function to build manifests
build_manifests() {
    local env=$1
    
    print_info "Building manifests for environment: $env"
    
    if command -v kustomize &> /dev/null; then
        kustomize build "$DEPLOY_DIR/overlays/$env"
    else
        kubectl kustomize "$DEPLOY_DIR/overlays/$env"
    fi
}

# Function to deploy
deploy() {
    local env=$1
    
    print_info "Deploying to environment: $env"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        print_info "DRY RUN - Showing manifests that would be applied:"
        build_manifests "$env"
    else
        kubectl apply -k "$DEPLOY_DIR/overlays/$env"
        
        # Get namespace
        local namespace
        case $env in
            development)
                namespace="pr-cli-dev"
                ;;
            staging)
                namespace="pr-cli-staging"
                ;;
            production)
                namespace="pr-cli"
                ;;
        esac
        
        print_info "Waiting for deployment to be ready..."
        kubectl rollout status deployment -n "$namespace" -l app.kubernetes.io/name=pr-cli --timeout=5m
        
        print_info "Deployment successful!"
        print_info "Check status with: kubectl get all -n $namespace"
    fi
}

# Function to show usage
usage() {
    cat <<EOF
Usage: $0 [ENVIRONMENT]

Deploy PR CLI webhook service to Kubernetes using Kustomize.

ENVIRONMENT:
    development     Deploy to development environment (default)
    staging         Deploy to staging environment
    production      Deploy to production environment

ENVIRONMENT VARIABLES:
    DRY_RUN=true        Show manifests without applying
    SKIP_SECRETS=true   Skip secret validation prompts

EXAMPLES:
    # Deploy to development
    $0 development

    # Deploy to production
    $0 production

    # Dry run for staging
    DRY_RUN=true $0 staging

    # Deploy without secret checks
    SKIP_SECRETS=true $0 development

EOF
}

# Main function
main() {
    if [[ "${1:-}" == "-h" ]] || [[ "${1:-}" == "--help" ]]; then
        usage
        exit 0
    fi
    
    print_info "PR CLI Webhook Service Deployment"
    print_info "=================================="
    
    check_prerequisites
    validate_environment "$ENVIRONMENT"
    
    if [[ "$SKIP_SECRETS" != "true" ]]; then
        check_secrets "$ENVIRONMENT"
    fi
    
    deploy "$ENVIRONMENT"
    
    print_info "Done!"
}

main "$@"

