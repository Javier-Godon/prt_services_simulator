# Quick Reference — PRT Services Simulator Deployment

## One-line deploy

```bash
./deployment/deploy.sh
```

Done! The simulator is running.

---

## Available commands

| Command | Purpose |
|---|---|
| `./deployment/deploy.sh` | Deploy to Kubernetes (creates namespace, Secret, pods) |
| `./deployment/deploy.sh --smoke-test` | Deploy and run tests to verify all endpoints |
| `./deployment/deploy.sh --namespace custom-ns` | Deploy to a custom namespace |
| `./deployment/undeploy.sh` | Remove everything (delete namespace) |
| `./deployment/undeploy.sh --force` | Remove without confirmation |

---

## Common tasks

### View logs

```bash
kubectl logs -f deployment/prt-services-simulator -n prt-simulator
```

### Shell into pod

```bash
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- sh
```

### Port-forward to localhost

```bash
kubectl port-forward svc/prt-services-simulator 8087:8087 -n prt-simulator
```

Then access locally: `http://localhost:8087/...`

### Access via NodePort (testing only)

```bash
# Find your node IP
kubectl get nodes -o wide

# Access directly on port 30087 — no port-forward needed
curl http://<node-ip>:30087/protocol/openid-connect/token ...
```

### Restart deployment

```bash
kubectl rollout restart deployment/prt-services-simulator -n prt-simulator
```

### Update configuration

1. Edit `deployment/configmap.yaml`
2. Apply and restart:

```bash
kubectl apply -f deployment/configmap.yaml
kubectl rollout restart deployment/prt-services-simulator -n prt-simulator
```

### Switch fixture

Edit `deployment/configmap.yaml`:

```yaml
simulator:
  download:
    fixture-file: /data/fixtures/masterlist_es.bin   # Spanish ML
    # OR
    fixture-file: /data/fixtures/masterlist_mock.bin  # Synthetic mock
```

Then:

```bash
kubectl apply -f deployment/configmap.yaml
kubectl rollout restart deployment/prt-services-simulator -n prt-simulator
```

### Update fixtures

```bash
# Replace files in deployment/fixtures/
cp /new/path/masterlist_mock.bin deployment/fixtures/
cp /new/path/masterlist_es.bin deployment/fixtures/

# Recreate Secret
kubectl delete secret prt-simulator-fixture -n prt-simulator
kubectl create secret generic prt-simulator-fixture \
  --namespace prt-simulator \
  --from-file=masterlist_mock.bin=deployment/fixtures/masterlist_mock.bin \
  --from-file=masterlist_es.bin=deployment/fixtures/masterlist_es.bin

# Restart pod
kubectl rollout restart deployment/prt-services-simulator -n prt-simulator
```

---

## Endpoints

Test the three-step flow:

### Step 1: Get access token

```bash
curl -X POST http://localhost:8087/protocol/openid-connect/token \
  -d "grant_type=password&client_id=cert-parser-client&client_secret=super-secret-123&username=operator@border.gov&password=operator-pass"
```

### Step 2: SFC login

```bash
curl -X POST http://localhost:8087/auth/v1/login \
  -H "Authorization: Bearer simulated-access-token-abc123" \
  -H "Content-Type: application/json" \
  -d '{"borderPostId":1,"boxId":1,"passengerControlType":1}'
```

### Step 3: Download certificate

```bash
curl -X GET http://localhost:8087/certificates/csca \
  -H "Authorization: Bearer simulated-access-token-abc123" \
  -H "x-sfc-authorization: Bearer simulated-sfc-token-xyz789" \
  --output masterlist.bin
```

---

## Fixtures

| File | Used for | Notes |
|---|---|---|
| `deployment/fixtures/masterlist_mock.bin` | CI / testing | Synthetic, safe for public repos |
| `deployment/fixtures/masterlist_es.bin` | Production staging | Real Master List, staging branch only |
| `src/main/resources/fixtures/masterlist_mock.bin` | Local unit tests | Committed, same as above |
| `src/main/resources/fixtures/ml_es.bin` | Local testing | Real ML, gitignored, never committed |

---

## Troubleshooting

### Pods stuck in `Pending`

```bash
kubectl describe pod -l app.kubernetes.io/name=prt-services-simulator -n prt-simulator
# Check Events section for errors
```

### `GET /certificates/csca` returns 500

Check if fixture path exists:

```bash
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- ls -lh /data/fixtures/
```

### Need help?

See full guide: `deployment/DEPLOYMENT_GUIDE.md`

---

## Files

- `deploy.sh` — Automated deployment script
- `undeploy.sh` — Automated teardown script
- `README.md` — Overview
- `DEPLOYMENT_GUIDE.md` — Full step-by-step guide
- `secret-fixture.yaml` — K8s Secret manifest (data empty)
- `configmap.yaml` — K8s ConfigMap with simulator config
- `deployment.yaml` — K8s Deployment spec
- `service.yaml` — K8s Service spec
- `namespace.yaml` — K8s Namespace
- `kustomization.yaml` — Kustomize overlay

