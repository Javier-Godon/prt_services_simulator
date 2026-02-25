# Deployment Guide — PRT Services Simulator

## Quick start (automated)

The fastest way to deploy: use the automated script!

```bash
./deployment/deploy.sh
```

This does everything:
- ✅ Validates prerequisites (kubectl, fixture files)
- ✅ Creates the namespace
- ✅ Creates the Secret from `deployment/fixtures/`
- ✅ Applies all Kubernetes manifests
- ✅ Waits for the pod to be ready
- ✅ Displays useful commands

### With smoke test

```bash
./deployment/deploy.sh --smoke-test
```

This also runs basic tests against all three endpoints to verify everything works.

### Custom namespace

```bash
./deployment/deploy.sh --namespace my-custom-namespace
```

### Undeploy

```bash
./deployment/undeploy.sh
```

---

## Overview

This guide covers the full lifecycle of deploying `prt-services-simulator` to a Kubernetes cluster:
first-time setup, fixture management, configuration changes, image updates, and teardown.

The deployment consists of:

| Resource | Name | Purpose |
|---|---|---|
| Namespace | `prt-simulator` | Isolation |
| ConfigMap | `prt-simulator-config` | All simulator parameters (Spring Boot `application.yaml`) |
| Secret | `prt-simulator-fixture` | Binary Master List fixture files |
| Deployment | `prt-services-simulator` | Spring Boot pod |
| Service | `prt-services-simulator` | ClusterIP on port 8087 |

---

## Fixture files

| File | Where | Purpose | Committed? |
|---|---|---|---|
| `masterlist_mock.bin` | `deployment/fixtures/` and `src/main/resources/fixtures/` | Synthetic mock — safe for CI and public repos | ✅ Yes (all branches) |
| `masterlist_es.bin` | `deployment/fixtures/` | Real Spanish Master List | ✅ Yes (staging branch only) |

- **`masterlist_mock.bin`** is a synthetic 256-byte blob with no real certificate data. It is what tests and CI pipelines use.
- **`masterlist_es.bin`** is the real staging fixture. It lives in `deployment/fixtures/` and is committed to the staging branch, but should not be merged into public branches.
- `src/main/resources/fixtures/ml_es.bin` (real ML, local only) is gitignored and never committed anywhere.

---

## Prerequisites

- `kubectl` configured and pointing at the target cluster
- Access to `ghcr.io` to pull the container image
- The fixture binaries available in `deployment/fixtures/` (committed on staging branch):
  - `deployment/fixtures/masterlist_mock.bin`
  - `deployment/fixtures/masterlist_es.bin`

---

## First-time deployment

### Step 1 — Create the namespace

The namespace must exist before the Secret can be created:

```bash
kubectl apply -f deployment/namespace.yaml
```

### Step 2 — Create the fixture Secret imperatively

```bash
kubectl create secret generic prt-simulator-fixture \
  --namespace prt-simulator \
  --from-file=masterlist_mock.bin=deployment/fixtures/masterlist_mock.bin \
  --from-file=masterlist_es.bin=deployment/fixtures/masterlist_es.bin
```

> ⚠️ Do **not** use `kubectl apply -f deployment/secret-fixture.yaml` to populate the Secret —
> that manifest is kept in git as a documentation skeleton only (data fields are intentionally empty).
> The Secret must always be created imperatively from `deployment/fixtures/`.

### Step 3 — Deploy all remaining resources via Kustomize

```bash
kubectl apply -k deployment/
```

This applies `namespace.yaml`, `configmap.yaml`, `deployment.yaml`, and `service.yaml`.
Since the Secret already exists from Step 2, Kustomize reconciles only its metadata —
it does **not** overwrite the binary data.

### Step 4 — Verify

```bash
# All resources in the namespace
kubectl get all -n prt-simulator

# Check the pod is Running
kubectl get pods -n prt-simulator

# Verify mounted fixture files inside the pod
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- ls -lh /data/fixtures/

# Verify active Spring Boot config
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- cat /config/application.yaml
```

### Step 5 — Smoke test

```bash
# Step 1: Get an access token
kubectl run -it --rm curl --image=curlimages/curl --restart=Never -n prt-simulator -- \
  curl -s -X POST http://prt-services-simulator:8087/protocol/openid-connect/token \
  -d "grant_type=password&client_id=cert-parser-client&client_secret=super-secret-123&username=operator@border.gov&password=operator-pass"

# Step 2: SFC login
kubectl run -it --rm curl --image=curlimages/curl --restart=Never -n prt-simulator -- \
  curl -s -X POST http://prt-services-simulator:8087/auth/v1/login \
  -H "Authorization: Bearer simulated-access-token-abc123" \
  -H "Content-Type: application/json" \
  -d '{"borderPostId":1,"boxId":1,"passengerControlType":1}'

# Step 3: Download certificate binary (expect HTTP 200 + binary body)
kubectl run -it --rm curl --image=curlimages/curl --restart=Never -n prt-simulator -- \
  curl -s -o /dev/null -w "%{http_code}" \
  http://prt-services-simulator:8087/certificates/csca \
  -H "Authorization: Bearer simulated-access-token-abc123" \
  -H "x-sfc-authorization: Bearer simulated-sfc-token-xyz789"
```

---

## Fixture management

### Switching the active fixture

The simulator serves whichever file `simulator.download.fixture-file` points to in `configmap.yaml`.
Edit that value and restart:

```bash
# Options:
#   /data/fixtures/masterlist_es.bin   — real Spanish Master List (staging)
#   /data/fixtures/masterlist_mock.bin — synthetic mock (CI / testing)

kubectl apply -f deployment/configmap.yaml
kubectl rollout restart deployment/prt-services-simulator -n prt-simulator
kubectl rollout status deployment/prt-services-simulator -n prt-simulator
```

### Updating fixture binaries

If the binary files change, recreate the Secret:

```bash
kubectl delete secret prt-simulator-fixture -n prt-simulator

kubectl create secret generic prt-simulator-fixture \
  --namespace prt-simulator \
  --from-file=masterlist_mock.bin=deployment/fixtures/masterlist_mock.bin \
  --from-file=masterlist_es.bin=deployment/fixtures/masterlist_es.bin

kubectl rollout restart deployment/prt-services-simulator -n prt-simulator
kubectl rollout status deployment/prt-services-simulator -n prt-simulator
```

### Verifying what is mounted

```bash
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- ls -lh /data/fixtures/
```

---

## Updating simulator parameters

All simulator parameters live in `deployment/configmap.yaml` under the `application.yaml` key.
Edit that file and apply:

```bash
kubectl apply -f deployment/configmap.yaml
kubectl rollout restart deployment/prt-services-simulator -n prt-simulator
kubectl rollout status deployment/prt-services-simulator -n prt-simulator
```

### Parameter reference

| YAML key | Description | Default |
|---|---|---|
| `simulator.auth.expected-client-id` | Expected OAuth2 `client_id` | `cert-parser-client` |
| `simulator.auth.expected-client-secret` | Expected `client_secret` | `super-secret-123` |
| `simulator.auth.expected-username` | Expected username | `operator@border.gov` |
| `simulator.auth.expected-password` | Expected password | `operator-pass` |
| `simulator.auth.access-token` | Issued access token value | `simulated-access-token-abc123` |
| `simulator.login.expected-border-post-id` | Expected `borderPostId` | `1` |
| `simulator.login.expected-box-id` | Expected `boxId` | `XX/99/X` |
| `simulator.login.expected-passenger-control-type` | Expected `passengerControlType` | `1` |
| `simulator.login.sfc-token` | Issued SFC token value | `simulated-sfc-token-xyz789` |
| `simulator.download.fixture-file` | Active fixture path inside the pod | `/data/fixtures/masterlist_es.bin` |

---

## Updating the container image

### Via manifest (persistent)

Edit the `image:` field in `deployment/deployment.yaml`, then:

```bash
kubectl apply -f deployment/deployment.yaml
kubectl rollout status deployment/prt-services-simulator -n prt-simulator
```

### Via kubectl set image (ad-hoc)

```bash
kubectl set image deployment/prt-services-simulator \
  prt-services-simulator=ghcr.io/Javier-Godon/prt-services-simulator:<tag> \
  -n prt-simulator
kubectl rollout status deployment/prt-services-simulator -n prt-simulator
```

---

## Re-deploying from scratch

Use this after a fresh clone (staging branch) to bring up the full stack:

```bash
# 1. Create the namespace
kubectl apply -f deployment/namespace.yaml

# 2. Create the Secret from committed fixture files
kubectl create secret generic prt-simulator-fixture \
  --namespace prt-simulator \
  --from-file=masterlist_mock.bin=deployment/fixtures/masterlist_mock.bin \
  --from-file=masterlist_es.bin=deployment/fixtures/masterlist_es.bin

# 3. Apply everything else
kubectl apply -k deployment/

# 4. Wait for rollout
kubectl rollout status deployment/prt-services-simulator -n prt-simulator
```

---

## Useful commands

```bash
# Live logs
kubectl logs -f deployment/prt-services-simulator -n prt-simulator

# Shell into the pod
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- sh

# Verify mounted fixtures
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- ls -lh /data/fixtures/

# Verify active config
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- cat /config/application.yaml

# Describe pod (useful for volume mount issues)
kubectl describe pod -l app.kubernetes.io/name=prt-services-simulator -n prt-simulator

# Check Secret keys (without printing binary data)
kubectl get secret prt-simulator-fixture -n prt-simulator -o jsonpath='{.data}' | \
  python3 -c "import sys,json; [print(k, len(v), 'chars (base64)') for k,v in json.load(sys.stdin).items()]"

# Rollout history
kubectl rollout history deployment/prt-services-simulator -n prt-simulator

# Rollback to previous version
kubectl rollout undo deployment/prt-services-simulator -n prt-simulator

# Tear down everything
kubectl delete namespace prt-simulator
```

---

## Troubleshooting

### Pod stuck in `Pending` — Secret not found

The Secret must exist before the pod can start. Create it as shown in Step 2 above.

```bash
# Check the error
kubectl describe pod -l app.kubernetes.io/name=prt-services-simulator -n prt-simulator
# Look for: MountVolume.SetUp failed ... secret "prt-simulator-fixture" not found
```

### `GET /certificates/csca` returns 500 — fixture not found at path

The path configured in `simulator.download.fixture-file` does not match what is mounted.

```bash
# What path is configured?
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- \
  cat /config/application.yaml | grep fixture-file

# What is actually mounted?
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- ls -lh /data/fixtures/
```

Valid values for `simulator.download.fixture-file`:
- `/data/fixtures/masterlist_es.bin` — real Spanish ML (default in staging)
- `/data/fixtures/masterlist_mock.bin` — synthetic mock

### `GET /certificates/csca` returns 500 — Secret key mismatch

The key names inside the Secret must match the `key:` fields in `deployment.yaml`.

```bash
kubectl get secret prt-simulator-fixture -n prt-simulator -o yaml | grep -A5 'data:'
```

Expected keys: `masterlist_mock.bin` and `masterlist_es.bin`.
If they differ, delete the Secret and recreate it as shown in [Fixture management](#fixture-management).
