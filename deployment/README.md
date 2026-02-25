# Kubernetes Deployment — PRT Services Simulator

## Quick start

Use the automated deployment script:

```bash
./deployment/deploy.sh
```

For details and manual steps, see **[DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md)**.

---

## Structure

```
deployment/
├── deploy.sh               # 🚀 Automated deployment (recommended)
├── undeploy.sh             # 🗑️ Automated teardown
├── kustomization.yaml      # Wires all manifests together
├── namespace.yaml          # prt-simulator namespace
├── configmap.yaml          # application.yaml — all simulator parameters
├── secret-fixture.yaml     # Skeleton Secret manifest (data fields empty — see DEPLOYMENT_GUIDE.md)
├── deployment.yaml         # Spring Boot pod
├── service.yaml            # ClusterIP service on port 8087 (in-cluster)
├── service-nodeport.yaml   # NodePort service on port 30087 (testing only)
├── fixtures/               # Local fixture files (gitignored)
│   ├── masterlist_mock.bin # Synthetic mock Master List (safe for CI / public repos)
│   └── masterlist_es.bin   # Real Spanish Master List (staging only)
├── README.md               # This file
└── DEPLOYMENT_GUIDE.md     # Full step-by-step deployment guide
```

## How configuration works

```
ConfigMap (application.yaml)
  └── mounted at /config/application.yaml
        └── loaded by Spring Boot via SPRING_CONFIG_ADDITIONAL_LOCATION=file:/config/
              └── overrides every simulator.* value baked into the JAR

Secret (prt-simulator-fixture)
  ├── masterlist_mock.bin → mounted at /data/fixtures/masterlist_mock.bin
  └── masterlist_es.bin   → mounted at /data/fixtures/masterlist_es.bin
        └── active fixture selected by simulator.download.fixture-file in the ConfigMap
```

Changing **any** simulator parameter = edit `configmap.yaml` + `kubectl apply`.  
Changing **the active fixture** = change `simulator.download.fixture-file` in `configmap.yaml`.  
Changing **the fixture binary** = recreate the Secret from `deployment/fixtures/` + rollout restart.

> ℹ️ **Image pull policy**: `imagePullPolicy: Always` is set in `deployment.yaml`.  
> This ensures the `latest` tag is always pulled from the registry on each deploy, suitable for development.  
> For production with pinned versions, use `imagePullPolicy: IfNotPresent`.

---

## Fixture files

| File | Secret key | Mount path | Description |
|---|---|---|---|
| `deployment/fixtures/masterlist_mock.bin` | `masterlist_mock.bin` | `/data/fixtures/masterlist_mock.bin` | Synthetic mock — safe for CI / public repos |
| `deployment/fixtures/masterlist_es.bin` | `masterlist_es.bin` | `/data/fixtures/masterlist_es.bin` | Real Spanish Master List — staging only |

The active fixture served by `GET /certificates/csca` is controlled by `simulator.download.fixture-file` in `configmap.yaml`.

Default (classpath fallback, used by local tests): `fixtures/masterlist_mock.bin` from `src/main/resources/`.

---

## All configurable parameters

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
| `simulator.download.fixture-file` | Path to the binary fixture served by `GET /certificates/csca` | `/data/fixtures/masterlist_es.bin` |

---

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/protocol/openid-connect/token` | Step 1 — OpenID Connect password grant |
| `POST` | `/auth/v1/login` | Step 2 — SFC login |
| `GET` | `/certificates/csca` | Step 3 — Certificate download |

---

## See also

- **[QUICK_REFERENCE.md](QUICK_REFERENCE.md)** — Common commands and tasks at a glance
- **[DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md)** — Full step-by-step deployment walkthrough
