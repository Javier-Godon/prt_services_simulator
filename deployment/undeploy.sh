#!/bin/bash
# ═══════════════════════════════════════════════════════════════
# PRT Services Simulator — Kubernetes Undeploy Script
# ═══════════════════════════════════════════════════════════════
#
# This script removes the prt-services-simulator deployment
# and all associated Kubernetes resources.
#
# Usage:
#   ./deployment/undeploy.sh [--namespace NAMESPACE]
#
# Examples:
#   ./deployment/undeploy.sh
#   ./deployment/undeploy.sh --namespace custom-simulator
#
# ═══════════════════════════════════════════════════════════════

set -euo pipefail

# ─── Configuration ─────────────────────────────────────────────
NAMESPACE="prt-simulator"
FORCE=false

# ─── Colors for output ─────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ─── Helper functions ─────────────────────────────────────────
log_info() {
  echo -e "${BLUE}ℹ${NC} $*"
}

log_success() {
  echo -e "${GREEN}✓${NC} $*"
}

log_warn() {
  echo -e "${YELLOW}⚠${NC} $*"
}

log_error() {
  echo -e "${RED}✗${NC} $*"
}

# ─── Parse arguments ──────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case $1 in
    --namespace)
      NAMESPACE="$2"
      shift 2
      ;;
    --force)
      FORCE=true
      shift
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

# ─── Check kubectl ─────────────────────────────────────────────
if ! command -v kubectl &> /dev/null; then
  log_error "kubectl not found. Please install kubectl."
  exit 1
fi

# ─── Confirm action ───────────────────────────────────────────
if [ "$FORCE" = false ]; then
  log_warn "This will delete the entire namespace '$NAMESPACE' and all resources in it."
  read -p "Are you sure? (type 'yes' to confirm): " -r
  if [ "$REPLY" != "yes" ]; then
    log_info "Cancelled."
    exit 0
  fi
fi

# ─── Check if namespace exists ─────────────────────────────────
if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
  log_warn "Namespace '$NAMESPACE' does not exist. Nothing to delete."
  exit 0
fi

# ─── Delete namespace (and all resources in it) ─────────────────
log_info "Deleting Secrets..."

if kubectl get secret ghcr-secret -n "$NAMESPACE" &> /dev/null; then
  kubectl delete secret ghcr-secret -n "$NAMESPACE"
  log_success "ghcr-secret deleted"
else
  log_warn "ghcr-secret not found (skipping)"
fi

if kubectl get secret prt-simulator-fixture -n "$NAMESPACE" &> /dev/null; then
  kubectl delete secret prt-simulator-fixture -n "$NAMESPACE"
  log_success "prt-simulator-fixture deleted"
else
  log_warn "prt-simulator-fixture not found (skipping)"
fi

log_info "Deleting namespace '$NAMESPACE' and remaining resources..."

kubectl delete namespace "$NAMESPACE"

log_success "Namespace '$NAMESPACE' deleted successfully"

# ─── Summary ──────────────────────────────────────────────────
echo ""
log_success "Undeploy complete!"
echo ""
log_info "To re-deploy, run:"
echo "  ./deployment/deploy.sh"
echo ""

