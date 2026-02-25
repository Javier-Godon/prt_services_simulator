#!/bin/bash
# ═══════════════════════════════════════════════════════════════
# PRT Services Simulator — Kubernetes Deployment Script
# ═══════════════════════════════════════════════════════════════
#
# This script automates the full deployment of prt-services-simulator:
#   1. Validates prerequisites (kubectl, fixture files, CR_PAT/USERNAME)
#   2. Creates the namespace
#   3. Creates the GHCR image pull Secret (ghcr-secret) from CR_PAT + USERNAME
#   4. Creates the fixture Secret (prt-simulator-fixture) from deployment/fixtures/
#   5. Applies all remaining Kubernetes resources
#   6. Waits for the pod to be ready
#   7. Performs a smoke test (optional)
#
# Required environment variables (or set in dagger_go/credentials/.env):
#   CR_PAT    — GitHub personal access token with read:packages scope
#   USERNAME  — GitHub username
#
# Usage:
#   ./deployment/deploy.sh [--smoke-test] [--namespace NAMESPACE]
#
# Examples:
#   ./deployment/deploy.sh
#   ./deployment/deploy.sh --smoke-test
#   ./deployment/deploy.sh --namespace custom-simulator
#
# ═══════════════════════════════════════════════════════════════

set -euo pipefail

# ─── Configuration ─────────────────────────────────────────────
NAMESPACE="prt-simulator"
SMOKE_TEST=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
FIXTURES_DIR="${SCRIPT_DIR}/fixtures"
CREDENTIALS_ENV="${PROJECT_ROOT}/dagger_go/credentials/.env"

# ─── Colors for output ─────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ─── Parse arguments ──────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case $1 in
    --smoke-test)
      SMOKE_TEST=true
      shift
      ;;
    --namespace)
      NAMESPACE="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

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

# ─── Validate prerequisites ───────────────────────────────────
log_info "Validating prerequisites..."

# Check kubectl
if ! command -v kubectl &> /dev/null; then
  log_error "kubectl not found. Please install kubectl."
  exit 1
fi
log_success "kubectl found"

# Check cluster connection
if ! kubectl cluster-info &> /dev/null; then
  log_error "Cannot connect to Kubernetes cluster. Please configure kubectl."
  exit 1
fi
log_success "Connected to Kubernetes cluster"

# Check fixture files
FIXTURES=(
  "masterlist_mock.bin"
  "masterlist_es.bin"
)

for fixture in "${FIXTURES[@]}"; do
  if [ ! -f "${FIXTURES_DIR}/${fixture}" ]; then
    log_error "Fixture file not found: ${FIXTURES_DIR}/${fixture}"
    exit 1
  fi
  log_success "Found fixture: ${fixture}"
done

# Check CR_PAT and USERNAME — load from credentials/.env if not already set
if [ -z "${CR_PAT:-}" ] || [ -z "${USERNAME:-}" ]; then
  if [ -f "$CREDENTIALS_ENV" ]; then
    log_info "Loading credentials from ${CREDENTIALS_ENV}..."
    set -a
    # shellcheck source=/dev/null
    source "$CREDENTIALS_ENV"
    set +a
    log_success "Credentials loaded"
  fi
fi

if [ -z "${CR_PAT:-}" ]; then
  log_error "CR_PAT is not set. Export it or add it to dagger_go/credentials/.env"
  exit 1
fi
if [ -z "${USERNAME:-}" ]; then
  log_error "USERNAME is not set. Export it or add it to dagger_go/credentials/.env"
  exit 1
fi
log_success "Credentials available (USERNAME=${USERNAME})"

# ─── Step 1: Create namespace ─────────────────────────────────
log_info "Step 1: Creating namespace '${NAMESPACE}'..."

if kubectl get namespace "$NAMESPACE" &> /dev/null; then
  log_warn "Namespace '${NAMESPACE}' already exists (skipping)"
else
  kubectl create namespace "$NAMESPACE"
  log_success "Namespace created"
fi

# ─── Step 2: Create GHCR image pull Secret ────────────────────
log_info "Step 2: Creating GHCR image pull Secret 'ghcr-secret'..."

if kubectl get secret ghcr-secret -n "$NAMESPACE" &> /dev/null; then
  log_warn "Secret 'ghcr-secret' already exists in namespace '${NAMESPACE}'"
  read -p "Do you want to recreate it? (y/n) " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    kubectl delete secret ghcr-secret -n "$NAMESPACE"
    log_info "Deleted existing ghcr-secret"
  fi
fi

if ! kubectl get secret ghcr-secret -n "$NAMESPACE" &> /dev/null; then
  kubectl create secret docker-registry ghcr-secret \
    --namespace "$NAMESPACE" \
    --docker-server=ghcr.io \
    --docker-username="${USERNAME}" \
    --docker-password="${CR_PAT}" \
    --docker-email="${USERNAME}@users.noreply.github.com"
  log_success "GHCR pull Secret created"
else
  log_warn "ghcr-secret kept (not recreated)"
fi

# ─── Step 3: Create fixture Secret ────────────────────────────
log_info "Step 3: Creating fixture Secret 'prt-simulator-fixture'..."

# Check if Secret already exists
if kubectl get secret prt-simulator-fixture -n "$NAMESPACE" &> /dev/null; then
  log_warn "Secret 'prt-simulator-fixture' already exists in namespace '${NAMESPACE}'"
  read -p "Do you want to recreate it? (y/n) " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    kubectl delete secret prt-simulator-fixture -n "$NAMESPACE"
    log_info "Deleted existing prt-simulator-fixture Secret"
  else
    log_warn "Keeping existing prt-simulator-fixture Secret"
  fi
fi

# Create Secret (if it doesn't exist or user confirmed deletion)
if ! kubectl get secret prt-simulator-fixture -n "$NAMESPACE" &> /dev/null; then
  kubectl create secret generic prt-simulator-fixture \
    --namespace "$NAMESPACE" \
    --from-file=masterlist_mock.bin="${FIXTURES_DIR}/masterlist_mock.bin" \
    --from-file=masterlist_es.bin="${FIXTURES_DIR}/masterlist_es.bin"
  log_success "Secret created"
else
  log_warn "Secret already exists (not recreated)"
fi

# ─── Step 4: Apply all remaining resources via Kustomize ──────
log_info "Step 4: Applying Kubernetes manifests via Kustomize..."

kubectl apply -k "$SCRIPT_DIR"
log_success "Manifests applied"

# ─── Step 5: Wait for deployment to be ready ──────────────────
log_info "Step 5: Waiting for deployment to be ready (timeout: 60s)..."

kubectl rollout status deployment/prt-services-simulator \
  -n "$NAMESPACE" \
  --timeout=60s

log_success "Deployment is ready!"

# ─── Step 6: Verify ──────────────────────────────────────────
log_info "Step 6: Verifying deployment..."

log_info "Pod status:"
kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=prt-services-simulator

log_info "Service info:"
kubectl get service prt-services-simulator -n "$NAMESPACE"

log_info "NodePort service info (testing):"
kubectl get service prt-services-simulator-nodeport -n "$NAMESPACE"

NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null || echo "<node-ip>")
log_info "NodePort access: http://${NODE_IP}:30087"

log_info "Mounted fixtures inside pod:"
kubectl exec -it deployment/prt-services-simulator -n "$NAMESPACE" -- ls -lh /data/fixtures/

log_info "Active config (fixture-file path):"
kubectl exec -it deployment/prt-services-simulator -n "$NAMESPACE" -- \
  grep -A2 'download:' /config/application.yaml

log_success "Deployment verification complete"

# ─── Step 7: Smoke test (optional) ────────────────────────────
if [ "$SMOKE_TEST" = true ]; then
  log_info "Step 7: Running smoke test..."

  # Get pod name
  POD_NAME=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=prt-services-simulator \
    -o jsonpath='{.items[0].metadata.name}')

  # Create a test pod with curl
  TEST_POD="prt-simulator-test-$$"

  # Step 1: Get token
  log_info "  Testing Step 1: OpenID Connect token endpoint..."
  TOKEN_RESPONSE=$(kubectl run -it --rm "$TEST_POD" \
    --image=curlimages/curl:latest \
    --restart=Never \
    -n "$NAMESPACE" \
    -- curl -s -X POST http://prt-services-simulator:8087/protocol/openid-connect/token \
    -d "grant_type=password&client_id=cert-parser-client&client_secret=super-secret-123&username=operator@border.gov&password=operator-pass" \
    2>/dev/null || echo "{}")

  if echo "$TOKEN_RESPONSE" | grep -q "access_token"; then
    log_success "  ✓ Token endpoint responded correctly"
  else
    log_warn "  Could not verify token endpoint (check pod logs for details)"
  fi

  # Step 2: SFC login
  log_info "  Testing Step 2: SFC login endpoint..."
  SFC_RESPONSE=$(kubectl run -it --rm "$TEST_POD" \
    --image=curlimages/curl:latest \
    --restart=Never \
    -n "$NAMESPACE" \
    -- curl -s -X POST http://prt-services-simulator:8087/auth/v1/login \
    -H "Authorization: Bearer simulated-access-token-abc123" \
    -H "Content-Type: application/json" \
    -d '{"borderPostId":1,"boxId":1,"passengerControlType":1}' \
    2>/dev/null || echo "")

  if [ -n "$SFC_RESPONSE" ] && [ "$SFC_RESPONSE" != "null" ]; then
    log_success "  ✓ SFC login endpoint responded"
  else
    log_warn "  Could not verify SFC login endpoint"
  fi

  # Step 3: Certificate download
  log_info "  Testing Step 3: Certificate download endpoint..."
  HTTP_CODE=$(kubectl run -it --rm "$TEST_POD" \
    --image=curlimages/curl:latest \
    --restart=Never \
    -n "$NAMESPACE" \
    -- curl -s -o /dev/null -w "%{http_code}" \
    http://prt-services-simulator:8087/certificates/csca \
    -H "Authorization: Bearer simulated-access-token-abc123" \
    -H "x-sfc-authorization: Bearer simulated-sfc-token-xyz789" \
    2>/dev/null || echo "000")

  if [ "$HTTP_CODE" = "200" ]; then
    log_success "  ✓ Certificate download endpoint responded with HTTP 200"
  else
    log_warn "  Certificate download endpoint responded with HTTP $HTTP_CODE (expected 200)"
  fi

  log_success "Smoke test complete"
fi

# ─── Summary ──────────────────────────────────────────────────
echo ""
log_success "Deployment complete!"
echo ""
echo "  Namespace:   $NAMESPACE"
echo "  ClusterIP:   prt-services-simulator:8087       (in-cluster)"
echo "  NodePort:    <node-ip>:30087                   (external, testing only)"
echo ""
log_info "Find node IP:  kubectl get nodes -o wide"
log_info "Useful commands:"
echo "  kubectl logs -f deployment/prt-services-simulator -n $NAMESPACE"
echo "  kubectl exec -it deployment/prt-services-simulator -n $NAMESPACE -- sh"
echo "  kubectl port-forward svc/prt-services-simulator 8087:8087 -n $NAMESPACE"
echo "  kubectl delete namespace $NAMESPACE"
echo ""

