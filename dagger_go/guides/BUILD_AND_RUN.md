# Dagger Go Build & Run Guide

Complete guide to building and running the PRT Services Simulator Dagger Go CI/CD pipeline.

## ⚡ Quick Reference

| Goal | Command | Time |
|------|---------|------|
| **Full pipeline** | `set -a && source credentials/.env && set +a && export CR_PAT USERNAME && ./run.sh` | 40-60s |
| **Corporate pipeline** | `set -a && source credentials/.env && set +a && export CR_PAT USERNAME DEBUG_CERTS && ./run-corporate.sh` | 40-60s |
| **Test code** | `go test -v` | 30-60s |
| **Build binary** | `go build -o railway-dagger-go main.go` | 5-10s |
| **Build corporate** | `go build -tags corporate -o railway-corporate-dagger-go corporate_main.go` | 5-10s |
| **Debug** | VSC F5 → Debug Dagger Go | Live |

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

**Key Notes:**
- ✅ Uses Dagger Go SDK (downloads automatically)
- ❌ Does NOT require Dagger CLI installed
- Requires Docker running (SDK uses Docker Engine)

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

**Success output:**
```
$ ls -lh railway-dagger-go
-rwxrwxr-x 20M railway-dagger-go
$ file railway-dagger-go
railway-dagger-go: ELF 64-bit LSB executable, x86-64
```

**Duration**: 5-10 seconds

**Key Notes:**
- ✅ Pure Go compilation (no dependencies needed after download)
- ❌ Does NOT require Docker
- Binary ready for server deployment
- Run with credentials: `set -a && source credentials/.env && set +a && ./railway-dagger-go`

---

### Workflow 3: Run Full CI/CD Pipeline

**Goal**: Build Docker image and deploy to GitHub Container Registry

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
7. Pushes to GitHub Container Registry

**Success output:**
```
🚀 Starting prt_services_simulator CI/CD Pipeline (Go SDK v0.19.7)...
🧪 Running all tests (Spring Boot MockMvc)...
  [INFO] Tests run: 8, Failures: 0, Errors: 0, Skipped: 0
  [INFO] BUILD SUCCESS
📦 Building Maven artifact (JAR file)...
🐳 Building Docker image...
📤 Publishing to: ghcr.io/username/prt-services-simulator:v0.1.0-abc1234-20260223-1200
✅ Pipeline completed successfully!
```

**Duration**:
- First run: 3-5 minutes (downloads dependencies)
- Cached run: 1-2 minutes (uses Maven + layer cache)

**Requirements:**
- ✅ Docker running (Dagger SDK uses it)
- ✅ CR_PAT and USERNAME in credentials/.env
- ❌ Dagger CLI NOT required

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

**Prerequisites:**

1. **Place CA certificates** (optional):
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

**Success output:**
```
🏢 CORPORATE MODE: MITM Proxy & Custom CA Support
   📜 Found 2 CA certificate path(s)
      - ca-certificates.crt ✅
      - corporate-ca.pem ✅

🚀 Starting prt_services_simulator CI/CD Pipeline (Go SDK v0.19.7 - Corporate Mode)...
🧪 Running tests (MockMvc, inside Dagger container)...
  [INFO] BUILD SUCCESS
📦 Building Maven artifact...
🐳 Building Docker image...
✅ Corporate pipeline completed successfully!
```

**Certificate Auto-Discovery Sources:**
1. `credentials/certs/` (user-provided `.pem` files)
2. System stores (`/etc/ssl/certs/`, `/etc/pki/ca-trust/`)
3. Docker/Rancher Desktop directories (`.docker/certs.d`, `.rancher/certs.d`)
4. macOS Docker Desktop Group Containers
5. Windows Certificate Store (via WSL)
6. Jenkins CI/CD environment (`$JENKINS_HOME/certs`)
7. GitHub Actions runner (`$RUNNER_TEMP/ca-certificates`)
8. `CA_CERTIFICATES_PATH` environment variable

**Documentation:**
- [CERTIFICATE_LOGGING.md](../CERTIFICATE_LOGGING.md) - Detailed logging guide
- [CERTIFICATE_QUICK_REFERENCE.md](../CERTIFICATE_QUICK_REFERENCE.md) - Setup guide

**Duration**: 40-60 seconds (same as standard pipeline)

**Key Notes:**
- ✅ Gracefully degrades if no certificates found
- ✅ Works on Linux, macOS, Windows (WSL), Jenkins, GitHub Actions
- ✅ Zero configuration needed (auto-discovery works automatically)
- ✅ Optional manual configuration via `credentials/certs/`

---

## Debug Your Code (VSC)

### Setup

1. Open `dagger_go/main.go`
2. Click gutter (left margin) next to line number to set breakpoint
3. Red circle ⭕ appears

### Run Debugger

Press `F5` and select "Debug Dagger Go"

**Debug Controls:**

| Key | Action |
|-----|--------|
| F10 | Step over |
| F11 | Step into |
| Shift+F11 | Step out |
| F5 | Continue |
| Shift+F5 | Stop |

**Inspect Variables:**
- Left panel shows locals, watch expressions, call stack
- Hover over variables to inspect values

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
│       ├── .env                # CR_PAT, USERNAME, optional proxy
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

### Error: "dagger: command not found"

**Cause**: You tried to use Dagger CLI

**Solution**: Don't use Dagger CLI! Use Go commands instead:

```bash
# ❌ Wrong:
dagger run

# ✅ Right:
go test -v
./run.sh
```

### Error: "Cannot connect to Docker daemon"

**Cause**: Docker not running

**Solution**:

```bash
docker ps
# If error:
# - macOS/Windows: Open Docker Desktop app
# - Linux: sudo systemctl start docker
```

### Error: "credentials/.env not found"

**Cause**: Missing credentials file

**Solution**:

```bash
mkdir -p credentials
cat > credentials/.env << EOF
CR_PAT=ghp_your_github_token
USERNAME=your_github_username
EOF
```

### Error: "go: command not found"

**Cause**: Go not in PATH

**Solution**:

```bash
which go
# Should show: /usr/local/go/bin/go

# Add to PATH if needed:
export PATH=$PATH:/usr/local/go/bin
```

### Error: "Permission denied: ./run.sh"

**Cause**: Script doesn't have execute permissions

**Solution**:

```bash
chmod +x run.sh run-corporate.sh test.sh
```

### Error: "go: unknown module: dagger.io/dagger"

**Cause**: Dependencies not downloaded

**Solution**:

```bash
go mod download dagger.io/dagger
go mod tidy
go test -v
```

### Error: "missing go.sum entry for module providing package dagger.io/dagger"

**Cause**: go.sum file not synchronized with go.mod

**Solution** (run these in order):

```bash
# Step 1: Download the Dagger module
go mod download dagger.io/dagger

# Step 2: Tidy up go.mod and go.sum
go mod tidy

# Step 3: Try building again
go build -o railway-dagger-go main.go
```

---

## Common Issues & Quick Fixes

| Problem | Quick Fix |
|---------|-----------|
| "Command not found: go" | Install Go from golang.org |
| "Cannot connect to Docker" | Start Docker Desktop or daemon |
| "Permission denied: ./run.sh" | `chmod +x run.sh run-corporate.sh` |
| ".env not found" | Create `credentials/.env` with CR_PAT and USERNAME |
| "Module not found" | `go mod tidy` |
| "dagger: command not found" | Don't use Dagger CLI - use `go test -v` instead |
| "No POM in this directory" | Source mount path mismatch — ensure pipeline binary is rebuilt |

---

## Performance Tips

### Faster Builds

1. **Keep Docker running** - Reuses containers
2. **Run tests only** - `go test -v` (faster than full pipeline)
3. **Use layer cache** - Docker caches previous layers

### Faster Development

1. **Build once** - `go build -o railway-dagger-go main.go`
2. **Deploy binary** - Run binary on servers
3. **Debug locally** - F5 for breakpoints

---

## Next Steps

1. ✅ Verify prerequisites (Go, Docker, credentials/.env)
2. ✅ Test code: `set -a && source credentials/.env && set +a && go test -v`
3. ✅ Build binary: `go mod download dagger.io/dagger && go build -o railway-dagger-go main.go`
4. ✅ Run pipeline: `set -a && source credentials/.env && set +a && export CR_PAT USERNAME && ./run.sh`
5. ✅ Monitor logs in Dagger output
6. ✅ Check image in GitHub Container Registry

---

## Resources

- 📖 [Go Documentation](https://golang.org/doc)
- 🐳 [Docker Documentation](https://docs.docker.com)
- 🔧 [Dagger SDK](https://docs.dagger.io)

---

**Summary**: No Dagger CLI needed. Just Go + Docker. Run `go test -v` to verify, `go build` to create binary, `./run.sh` to deploy. All credentials loaded from `credentials/.env` automatically.

**Last Updated**: February 23, 2026
**Status**: ✅ Ready to use
