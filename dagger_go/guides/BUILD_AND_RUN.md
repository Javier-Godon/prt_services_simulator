# Dagger Go Build & Run Guide

Complete guide to building and running the PRT Services Simulator Dagger Go CI/CD pipeline.

## ⚡ Quick Reference

 Goal  Command  Time 
---------------------
 **Full pipeline**  `set -a && source credentials/.env && set +a && export CR_PAT USERNAME && ./run.sh`  40-60s 
 **Corporate pipeline**  `set -a && source credentials/.env && set +a && export CR_PAT USERNAME DEBUG_CERTS && ./run-corporate.sh`  40-60s 
 **Test code**  `go test -v`  30-60s 
 **Build binary**  `go build -o railway-dagger-go main.go`  5-10s 
 **Build corporate**  `go build -tags corporate -o railway-corporate-dagger-go corporate_main.go`  5-10s 
 **Debug**  VSC F5 → Debug Dagger Go  Live 

**Key Points**:
- ❌ **Dagger CLI NOT required** - Uses Dagger Go SDK
- ✅ **Docker required** for running the pipeline
- ✅ **All tests run inside the Dagger container** (MockMvc via `mvn test`)
- ✅ **No Testcontainers / Cucumber** — pure Spring Boot MockMvc tests

> **Note**: All commands are run from inside the `dagger_go/` directory.

---

## Prerequisites

### What You Need

```bash
✅ Go 1.22+
✅ Docker running
✅ credentials/.env with CR_PAT and USERNAME
❌ Dagger CLI (NOT needed - SDK handles it)
```

### Verify Setup

```bash
go version                  # Should show go1.22+
docker ps                   # Should work
cat credentials/.env        # Should show CR_PAT=... USERNAME=...
```

---

## Environment Variables

### Required

| Variable | Description |
|---|---|
| `CR_PAT` | Personal Access Token with write access to your container registry |
| `USERNAME` | Your username on the git hosting platform |

### Optional — Git Hosting

| Variable | Default | Description |
|---|---|---|
| `GIT_HOST` | `github.com` | Git server hostname (e.g. `gitlab.com`, `bitbucket.org`, `gitea.mycompany.com`) |
| `GIT_AUTH_USERNAME` | `x-access-token` | HTTP auth username for clone. Use `oauth2` for GitLab PAT, `x-token-auth` for Bitbucket |
| `GIT_REPO` | auto-built from `GIT_HOST`/`USERNAME`/`REPO_NAME` | Override with a full URL if needed |
| `GIT_BRANCH` | `main` | Branch to build |
| `REPO_NAME` | `prt_services_simulator` | Repository name |

### Optional — Container Registry

| Variable | Default | Description |
|---|---|---|
| `REGISTRY` | `ghcr.io` | Container registry host (e.g. `docker.io`, `registry.gitlab.com`, `registry.mycompany.com`) |
| `REGISTRY_USERNAME` | same as `USERNAME` | Registry namespace / org (override when different from git username) |
| `IMAGE_NAME` | same as `REPO_NAME` | Docker image name |

### Optional — Pipeline Behaviour

| Variable | Default | Description |
|---|---|---|
| `DEPLOY_WEBHOOK` | _(unset)_ | URL to notify after a successful publish |
| `HTTP_PROXY` / `HTTPS_PROXY` | _(unset)_ | Corporate proxy (corporate pipeline only) |
| `DEBUG_CERTS` | `false` | Enable verbose certificate diagnostics (corporate pipeline) |
| `CA_CERTIFICATES_PATH` | _(unset)_ | Colon-separated paths to extra CA certificates |

### Minimal `credentials/.env` (GitHub + GHCR — defaults)

```bash
CR_PAT=ghp_your_github_pat
USERNAME=your_username
```

### Example — GitLab + GitLab Registry

```bash
CR_PAT=glpat_your_gitlab_token
USERNAME=your_gitlab_username
GIT_HOST=gitlab.com
GIT_AUTH_USERNAME=oauth2
REGISTRY=registry.gitlab.com
# REGISTRY_USERNAME defaults to USERNAME — override only if the namespace differs
```

### Example — Self-hosted Gitea + Docker Hub

```bash
CR_PAT=your_token
USERNAME=your_username
GIT_HOST=gitea.mycompany.com
GIT_AUTH_USERNAME=your_username   # Gitea uses the actual username for HTTP auth
REGISTRY=docker.io
REGISTRY_USERNAME=myorg           # Docker Hub org (different from git username)
```

---

## Workflows

### Workflow 1: Test Code

**Goal**: Verify code compiles and tests pass

**Command:**

```bash
set -a && source credentials/.env && set +a && go test -v
```

**What happens:**
1. Loads CR_PAT and USERNAME from credentials/.env
2. Downloads Dagger Go SDK v0.19.7 (automatically)
3. Runs unit tests

**Success output:**
```
go: downloading dagger.io/dagger v0.19.7
--- PASS: TestProjectRootDiscovery (1.234s)
--- PASS: TestEnvironmentVariables (0.567s)
PASS
ok      railway/dagger    2.345s
```

**Duration**: 30-60 seconds (first time), 5-10 seconds (cached)

---

### Workflow 2: Build Binary

**Goal**: Create executable for deployment

**Command:**

```bash
go mod download dagger.io/dagger && go mod tidy
go build -o railway-dagger-go main.go
```

**What happens:**
1. Downloads Dagger Go SDK and all dependencies
2. Compiles Go code to standalone executable
3. Creates 20MB binary: `railway-dagger-go`

**Duration**: 5-10 seconds

---

### Workflow 3: Run Full CI/CD Pipeline

**Goal**: Build Docker image and publish to your container registry

**Command:**

```bash
set -a && source credentials/.env && set +a && export CR_PAT USERNAME && ./run.sh
```

**What happens:**
1. Loads credentials from credentials/.env
2. Connects to Dagger Engine (via Docker)
3. Runs all MockMvc tests (`mvn test`) inside the container
4. Builds JAR (`mvn package -DskipTests`)
5. Creates Docker image
6. Tags with git commit SHA + timestamp
7. Pushes to your configured container registry

**Success output:**
```
🚀 Starting prt_services_simulator CI/CD Pipeline (Go SDK v0.19.7)...
   Repository: https://github.com/username/prt_services_simulator.git (branch: main)
   Registry:   ghcr.io/username
🧪 Running all tests (Spring Boot MockMvc)...
  [INFO] Tests run: 9, Failures: 0, Errors: 0, Skipped: 0
  [INFO] BUILD SUCCESS
📦 Building Maven artifact (JAR file)...
🐳 Building Docker image...
📤 Publishing to: ghcr.io/username/prt-services-simulator:v0.1.0-abc1234-20260316-1200
✅ Pipeline completed successfully!
```

**Duration**:
- First run: 3-5 minutes (downloads dependencies)
- Cached run: 1-2 minutes (uses Maven + layer cache)

**Requirements:**
- ✅ Docker running (Dagger SDK uses it)
- ✅ CR_PAT and USERNAME in credentials/.env

---

### Workflow 4: Run Corporate Pipeline (MITM Proxy + CA Certificates)

**Goal**: Build with corporate proxy and custom CA certificates support

**Command:**

```bash
set -a && source credentials/.env && set +a && export CR_PAT USERNAME DEBUG_CERTS && ./run-corporate.sh
```

**What's Different:**
- Auto-discovers CA certificates from 50+ locations
- Supports corporate MITM proxies (HTTP_PROXY, HTTPS_PROXY)
- Mounts certificates into containers automatically
- Enhanced logging with `DEBUG_CERTS=true`
- Probes registry-specific cert paths for the configured `REGISTRY`

**Prerequisites:**

1. **Place CA certificates** (optional — auto-discovery works without this):
   ```bash
   mkdir -p credentials/certs
   cp /path/to/corporate-ca.pem credentials/certs/
   ```

2. **Configure proxy** (optional - add to `credentials/.env`):
   ```bash
   HTTP_PROXY=http://proxy.company.com:8080
   HTTPS_PROXY=https://proxy.company.com:8080
   ```

3. **Enable debug logging** (optional):
   ```bash
   DEBUG_CERTS=true
   ```

**Certificate Auto-Discovery Sources:**
1. `credentials/certs/` (user-provided `.pem` files)
2. System stores (`/etc/ssl/certs/`, `/etc/pki/ca-trust/`)
3. Docker/Rancher Desktop directories (`.docker/certs.d/<REGISTRY>/`, `.rancher/certs.d/`)
4. macOS Docker Desktop Group Containers
5. Windows Certificate Store (via WSL)
6. `CA_CERTIFICATES_PATH` environment variable

**Documentation:**
- [CERTIFICATE_LOGGING.md](../CERTIFICATE_LOGGING.md) - Detailed logging guide
- [CERTIFICATE_QUICK_REFERENCE.md](../CERTIFICATE_QUICK_REFERENCE.md) - Setup guide

---

## Debug Your Code (VSC)

### Setup

1. Open `dagger_go/main.go`
2. Click gutter (left margin) next to line number to set breakpoint
3. Red circle ⭕ appears

### Run Debugger

Press `F5` and select "Debug Dagger Go"

**Debug Controls:**

 Key  Action 
-------------
 F10  Step over 
 F11  Step into 
 Shift+F11  Step out 
 F5  Continue 
 Shift+F5  Stop 

---

## File Structure

```
prt_services_simulator/
├── dagger_go/                  # ← You work here (run all commands from here)
│   ├── main.go                 # Standard pipeline
│   ├── corporate_main.go       # Corporate pipeline (proxy + CA certs)
│   ├── main_test.go            # Pipeline unit tests
│   ├── go.mod                  # Go module definition
│   ├── go.sum                  # Dependency checksums
│   ├── run.sh                  # Standard pipeline runner
│   ├── run-corporate.sh        # Corporate pipeline runner
│   ├── railway-dagger-go       # Binary (after `go build`)
│   ├── railway-corporate-dagger-go  # Corporate binary (after build)
│   └── credentials/            # ← Secrets live here
│       ├── .env                # CR_PAT, USERNAME, optional overrides
│       └── certs/              # Optional: corporate CA .pem files
│
├── src/
│   ├── main/java/com/border/simulator/   # Spring Boot application
│   └── test/java/com/border/simulator/   # MockMvc tests
│
├── pom.xml                     # Maven build descriptor
└── Dockerfile                  # Container image definition
```

---

## Troubleshooting

### Error: "Cannot connect to Docker daemon"

```bash
# - macOS/Windows: Open Docker Desktop app
# - Linux: sudo systemctl start docker
docker ps
```

### Error: "credentials/.env not found"

```bash
mkdir -p credentials
cat > credentials/.env << EOF
CR_PAT=your_token
USERNAME=your_username
EOF
```

### Error: "go: command not found"

```bash
export PATH=$PATH:/usr/local/go/bin
```

### Error: "Permission denied: ./run.sh"

```bash
chmod +x run.sh run-corporate.sh test.sh
```

### Error: "go: unknown module: dagger.io/dagger"

```bash
go mod download dagger.io/dagger
go mod tidy
go test -v
```

### Error: "No POM in this directory"

Source mount path mismatch — ensure the pipeline binary is rebuilt after any path changes:

```bash
go build -o railway-dagger-go main.go
```

---

## Common Issues & Quick Fixes

 Problem  Quick Fix 
--------------------
 "Command not found: go"  Install Go from golang.org 
 "Cannot connect to Docker"  Start Docker Desktop or daemon 
 "Permission denied: ./run.sh"  `chmod +x run.sh run-corporate.sh` 
 ".env not found"  Create `credentials/.env` with CR_PAT and USERNAME 
 "Module not found"  `go mod tidy` 
 "dagger: command not found"  Don't use Dagger CLI - use `go test -v` instead 
 "No POM in this directory"  Rebuild binary after any path changes 

---

## Next Steps

1. ✅ Verify prerequisites (Go, Docker, credentials/.env)
2. ✅ Test code: `set -a && source credentials/.env && set +a && go test -v`
3. ✅ Build binary: `go mod download dagger.io/dagger && go build -o railway-dagger-go main.go`
4. ✅ Run pipeline: `set -a && source credentials/.env && set +a && export CR_PAT USERNAME && ./run.sh`
5. ✅ Monitor logs in Dagger output
6. ✅ Check published image in your container registry

---

## Resources

- 📖 [Go Documentation](https://golang.org/doc)
- 🐳 [Docker Documentation](https://docs.docker.com)
- 🔧 [Dagger SDK](https://docs.dagger.io)

---

**Summary**: No Dagger CLI needed. Just Go + Docker. All credentials and registry/hosting overrides are loaded from `credentials/.env`. Defaults target GitHub + GHCR; set `GIT_HOST`, `REGISTRY`, etc. for other platforms.

**Last Updated**: March 16, 2026
**Status**: ✅ Ready to use
