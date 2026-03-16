# Corporate Pipeline - Quick Reference

## Two Separate Implementations

### Original Pipeline (Working - UNCHANGED)
```bash
cd dagger_go
./run.sh          # Runs original main.go
```

### Corporate Pipeline (New - ADDED)
```bash
cd dagger_go
./run-corporate.sh    # Runs corporate_main.go with CA + proxy support
```

---

## How Auto-Discovery Works

The corporate pipeline **automatically discovers CA certificates** from:

1. **User-provided directory** (`credentials/certs/*.pem`) - Highest priority
2. **Docker Desktop & Rancher Desktop certificates** (if running/installed):
   - Scans: `~/.docker/certs.d/`, `~/.rancher/certs.d/` (all subdirectories recursively)
   - Finds: All `.pem` and `.crt` files in Docker/Rancher certificate stores
   - Works on: **Windows**, **macOS**, **Linux**
   - **Supports BOTH**: Docker Desktop AND Rancher Desktop
3. **System certificate stores** (auto-detected by OS):
   - **Linux/Debian**: `/etc/ssl/certs/ca-bundle.crt` or `/etc/ssl/certs/ca-certificates.crt`, `/etc/rancher/k3s/certs.d`
   - **Linux/RHEL**: `/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem`, `/etc/docker/certs.d`, `/var/lib/docker/certs.d`, `/etc/rancher/`
   - **macOS**: `/etc/ssl/cert.pem` or `/usr/local/etc/openssl/cert.pem`, `~/.docker/certs.d/`, `~/.rancher/certs.d/`
   - **Windows 11 (Docker Desktop)**: `C:\ProgramData\Microsoft\Windows\Certificates\ca-certificates.pem`, `C:\Users\{USERNAME}\.docker\certs.d\`
   - **Windows 11 (Rancher Desktop)**: `C:\Users\{USERNAME}\AppData\Local\Rancher Desktop\certs`, `C:\Users\{USERNAME}\.rancher\certs.d\`
   - **Windows via WSL**: `/mnt/c/ProgramData/Microsoft/Windows/Certificates/ca-certificates.pem`
4. **Environment variable** `CA_CERTIFICATES_PATH` (colon or semicolon-separated paths)

**No manual extraction needed** - if Docker Desktop OR Rancher Desktop is running, we use their certificates automatically!

---

## Platform Quick Reference

| Platform | `GIT_HOST` | `GIT_AUTH_USERNAME` | `REGISTRY` |
|---|---|---|---|
| **GitHub** _(default)_ | `github.com` | `x-access-token` | `ghcr.io` |
| **GitLab.com** | `gitlab.com` | `oauth2` | `registry.gitlab.com` |
| **Self-hosted GitLab** | `gitlab.myco.com` | `oauth2` | `gitlab.myco.com:5050` |
| **Bitbucket Cloud** | `bitbucket.org` | `x-token-auth` | `docker.io` |
| **Gitea** | `gitea.myco.com` | _(your username)_ | `docker.io` or custom |
| **Docker Hub** | any | any | `docker.io` |

> `CR_PAT` is always your personal access token for both git clone auth and registry push.  
> Set `REGISTRY_USERNAME` only when your registry namespace differs from your git `USERNAME`.

---



### 1. Create directories
```bash
mkdir -p credentials/certs
```

### 2. Add your CA certificates (OPTIONAL - auto-discovery works too!)
```bash
# If you have extracted .pem files, copy them here
cp ~/company-ca.pem credentials/certs/
cp ~/proxy-ca.pem credentials/certs/

# Or: Use system certificates (auto-discovered from /etc/ssl/certs/, etc.)
# Or: Set CA_CERTIFICATES_PATH environment variable

# Verify what will be used
ls credentials/certs/
```

### 3. (Optional) Add proxy settings
```bash
cat >> credentials/.env << 'EOF'
HTTP_PROXY=http://proxy.company.com:8080
HTTPS_PROXY=https://proxy.company.com:8080
EOF
```

### 4. Run corporate pipeline
```bash
cd dagger_go
set -a && source credentials/.env && set +a
./run-corporate.sh
```

---

## Diagnose Certificate Issues

```bash
# Run with diagnostics enabled
cd dagger_go
DEBUG_CERTS=true ./run-corporate.sh 2>&1 | tee debug.log

# View certificate chains
grep -A 10 "Certificate Chain" debug.log
grep "subject=\|issuer=" debug.log
```

---

## Extract Missing Certificates

### From Windows (PowerShell)
```powershell
# Export all trusted CAs
Get-ChildItem cert:\CurrentUser\Root | ForEach-Object {
    $cert = $_
    $path = "certs\$($cert.Thumbprint).cer"
    [IO.File]::WriteAllBytes($path, $cert.Export([Security.Cryptography.X509Certificates.X509ContentType]::Cert))
}

# Convert .cer to .pem
openssl x509 -inform DER -in cert.cer -out cert.pem
```

### From Linux/Ubuntu
```bash
# Export from system
sudo cp /etc/ssl/certs/ca-bundle.crt ~/company-ca.pem

# Or capture from failing connection
echo | openssl s_client -showcerts -servername docker.io \
  -connect registry-1.docker.io:443 2>&1 | \
  sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > ca-chain.pem
```

---

## File Reference

| File | Purpose | Status |
|------|---------|--------|
| `main.go` | Original pipeline | ✅ UNCHANGED |
| `corporate_main.go` | Corporate version | ✅ ADDED |
| `run.sh` | Original runner | ✅ UNCHANGED |
| `run-corporate.sh` | Corporate runner | ✅ ADDED |

---

## What Corporate Pipeline Adds

✅ **Auto-discovery** of CA certificates from system, Docker Desktop, AND Rancher Desktop
✅ Custom CA certificate support
✅ HTTP/HTTPS proxy configuration
✅ Certificate diagnostics
✅ Full CA chain validation
✅ Automatic CA installation in containers
✅ **Works on Linux, macOS, Windows 11, and WSL**
✅ **Supports Docker Desktop AND Rancher Desktop**## Troubleshooting Quick Fixes

| Issue | Fix |
|-------|-----|
| "No certificates found" | `mkdir -p credentials/certs && cp *.pem credentials/certs/` |
| "Still getting x509 error" | Run `DEBUG_CERTS=true ./run-corporate.sh` to see what's needed |
| "Proxy not working" | Add `HTTP_PROXY=...` to credentials/.env |
| "Can't read certificate" | Convert to PEM: `openssl x509 -inform DER -in cert.cer -out cert.pem` |

---

## Environment Variables (Optional)

```bash
# ── Always required ──────────────────────────────────────────────────────────
CR_PAT=your_token          # Personal Access Token (write access to registry)
USERNAME=your_username     # Username on the git hosting platform

# ── Git hosting (defaults work for GitHub) ───────────────────────────────────
# GIT_HOST=github.com               # Change to gitlab.com, bitbucket.org, etc.
# GIT_AUTH_USERNAME=x-access-token  # oauth2 for GitLab PAT, x-token-auth for Bitbucket
# GIT_REPO=https://...              # Full URL override (auto-built if unset)
# GIT_BRANCH=main
# REPO_NAME=prt_services_simulator

# ── Container registry (defaults work for GHCR) ──────────────────────────────
# REGISTRY=ghcr.io                  # docker.io, registry.gitlab.com, etc.
# REGISTRY_USERNAME=                # Overrides USERNAME for the registry namespace/org
# IMAGE_NAME=prt-services-simulator

# ── CA certificate discovery (optional) ──────────────────────────────────────
# CA_CERTIFICATES_PATH=/path/cert1.pem:/path/cert2.pem

# ── Proxy (optional, only if a corporate proxy is present) ───────────────────
# HTTP_PROXY=http://proxy.company.com:8080
# HTTPS_PROXY=https://proxy.company.com:8080
# NO_PROXY=localhost,127.0.0.1,.local

# ── Debug mode (optional, for troubleshooting) ───────────────────────────────
# DEBUG_CERTS=true
```

---

## Quick Decision Tree

```
Does original ./run.sh work?
├─ YES → Don't need corporate pipeline ✅
└─ NO → Your laptop has corporate MITM
   ├─ Error: "x509: certificate signed by unknown authority"?
   │  └─ Use corporate pipeline ✅
   └─ Error: "Cannot connect to proxy"?
      └─ Add proxy settings to credentials/.env ✅
```

---

## One-Liner Commands

```bash
# Create setup
mkdir -p credentials/certs

# Run diagnostics
DEBUG_CERTS=true ./run-corporate.sh 2>&1 | grep -i "certificate\|issuer"

# Extract certificate chain from docker.io
echo | openssl s_client -showcerts -servername registry-1.docker.io \
  -connect registry-1.docker.io:443 2>&1 | \
  sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > chain.pem

# Validate certificate format
openssl x509 -in cert.pem -text -noout | head -20

# List your certificates
ls -lh credentials/certs/
```

---

## Files Created

```
✅ dagger_go/corporate_main.go           (180+ lines - corporate pipeline)
✅ dagger_go/run-corporate.sh            (shell script - runner)
✅ dagger_go/CORPORATE_PIPELINE.md       (full documentation)
✅ CERTIFICATE_DISCOVERY.md              (finding certificates guide)
✅ dagger_go/CORPORATE_QUICK_REFERENCE.md (this file)

❌ main.go                               (UNTOUCHED)
❌ run.sh                                (UNTOUCHED)
```

---

## Test Both Versions

```bash
# Original (if it works)
./run.sh

# Corporate (new version)
./run-corporate.sh

# Both are independent - use whichever works for your environment
```

---

**Created**: November 22, 2025
**Last Updated**: March 16, 2026
**Status**: Ready to use
**Original pipeline**: Completely safe ✅
