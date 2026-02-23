# Kubernetes Deployment — PRT Services Simulator

## Structure

```
deployment/
├── kustomization.yaml      # Wires all manifests together
├── namespace.yaml          # prt-simulator namespace
├── configmap.yaml          # application.yaml — all simulator parameters
├── secret-fixture.yaml     # ml_sample.bin binary fixture
├── deployment.yaml         # Spring Boot pod
└── service.yaml            # ClusterIP service on port 8087
```

## How configuration works

```
ConfigMap (application.yaml)
  └── mounted at /config/application.yaml
        └── loaded by Spring Boot via SPRING_CONFIG_ADDITIONAL_LOCATION=file:/config/
              └── overrides every simulator.* value baked into the JAR

Secret (ml_sample.bin)
  └── mounted at /data/fixtures/ml_sample.bin
        └── referenced by simulator.download.fixture-file in the ConfigMap
```

Changing **any** simulator parameter = edit `configmap.yaml` + `kubectl apply`.  
Changing **the fixture file** = update the Secret + `kubectl rollout restart`.

---

## Quick deploy (synthetic fixture)

```bash
# 1. Create namespace + all resources
kubectl apply -k deployment/

# 2. Verify
kubectl get all -n prt-simulator

# 3. Test from inside the cluster
kubectl run -it --rm curl --image=curlimages/curl --restart=Never -n prt-simulator -- \
  curl -s -X POST http://prt-services-simulator:8087/protocol/openid-connect/token \
  -d "grant_type=password&client_id=cert-parser-client&client_secret=super-secret-123&username=operator@border.gov&password=operator-pass"
```

---

## Replacing ml_sample.bin with a real fixture

The Secret holds the binary fixture. Replace it imperatively — never commit
real certificate data to git:

```bash
# Delete the existing secret
kubectl delete secret prt-simulator-fixture -n prt-simulator

# Recreate it from your real file
kubectl create secret generic prt-simulator-fixture \
  --namespace prt-simulator \
  --from-file=ml_sample.bin=/path/to/your/real_ml_sample.bin

# Restart the pod to pick up the new file
kubectl rollout restart deployment/prt-services-simulator -n prt-simulator

# Watch rollout
kubectl rollout status deployment/prt-services-simulator -n prt-simulator
```

---

## Overriding simulator parameters

Edit `deployment/configmap.yaml` and apply:

```bash
kubectl apply -f deployment/configmap.yaml

# Restart pod to pick up new config
kubectl rollout restart deployment/prt-services-simulator -n prt-simulator
```

### All available parameters

| YAML key | Description | Default |
|---|---|---|
| `simulator.auth.expected-client-id` | Expected OAuth2 client_id | `cert-parser-client` |
| `simulator.auth.expected-client-secret` | Expected client_secret | `super-secret-123` |
| `simulator.auth.expected-username` | Expected username | `operator@border.gov` |
| `simulator.auth.expected-password` | Expected password | `operator-pass` |
| `simulator.auth.access-token` | Issued access token value | `simulated-access-token-abc123` |
| `simulator.login.expected-border-post-id` | Expected borderPostId | `1` |
| `simulator.login.expected-box-id` | Expected boxId | `XX/99/X` |
| `simulator.login.expected-passenger-control-type` | Expected passengerControlType | `1` |
| `simulator.login.sfc-token` | Issued SFC token value | `simulated-sfc-token-xyz789` |
| `simulator.download.fixture-file` | Path to the binary fixture served by GET /certificates/csca | `/data/fixtures/ml_sample.bin` |

---

## Updating the image

Edit `deployment/deployment.yaml` and change the `image:` field, then:

```bash
kubectl apply -f deployment/deployment.yaml
kubectl rollout status deployment/prt-services-simulator -n prt-simulator
```

Or set the image directly:

```bash
kubectl set image deployment/prt-services-simulator \
  prt-services-simulator=ghcr.io/your-github-username/prt-services-simulator:v0.1.0-abc1234 \
  -n prt-simulator
```

---

## Useful commands

```bash
# Logs
kubectl logs -f deployment/prt-services-simulator -n prt-simulator

# Shell into the pod
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- sh

# Verify the mounted fixture inside the pod
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- \
  ls -lh /data/fixtures/

# Verify the mounted config inside the pod
kubectl exec -it deployment/prt-services-simulator -n prt-simulator -- \
  cat /config/application.yaml

# Delete everything
kubectl delete namespace prt-simulator
```

---

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/protocol/openid-connect/token` | Step 1 — OpenID Connect password grant |
| `POST` | `/auth/v1/login` | Step 2 — SFC login |
| `GET` | `/certificates/csca` | Step 3 — Certificate download |

