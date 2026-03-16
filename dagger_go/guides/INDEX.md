# Implementation Guides

Step-by-step guides for building, running, and configuring the Dagger pipeline.

## 📚 Contents

### Quick Start
- **[IMPLEMENTATION_QUICK_START.md](IMPLEMENTATION_QUICK_START.md)** - Copy-paste code examples for quick implementation

### Comprehensive Guides
- **[BUILD_AND_RUN.md](BUILD_AND_RUN.md)** ⭐ **START HERE**
  - Build & run commands
  - Full environment variable reference (including `GIT_HOST`, `REGISTRY`, `GIT_AUTH_USERNAME`, `REGISTRY_USERNAME`)
  - Platform examples: GitHub, GitLab, Bitbucket, self-hosted
  - Troubleshooting all common issues

- **[PIPELINE_INTERNALS.md](PIPELINE_INTERNALS.md)** - Deep technical details
  - `SimulatorPipeline` / `CorporatePipeline` struct fields
  - How Docker socket mounting works
  - Internal pipeline architecture

- **[TESTCONTAINERS_IMPLEMENTATION_GUIDE.md](TESTCONTAINERS_IMPLEMENTATION_GUIDE.md)** - For Java developers
  - How to write integration tests
  - Testcontainers setup

### Setup Guides
- **[INTELLIJ_SETUP.md](INTELLIJ_SETUP.md)** - IntelliJ IDEA configuration and debugging
- **[VSC_SETUP.md](VSC_SETUP.md)** - Visual Studio Code setup

## 🎯 Quick Navigation

**5-minute setup**: `../QUICKSTART.md`
**Environment variables / platform config**: `BUILD_AND_RUN.md#environment-variables`
**Corporate / proxy setup**: `../deployment/CORPORATE_PIPELINE.md`
**Platform reference table**: `../deployment/CORPORATE_QUICK_REFERENCE.md#platform-quick-reference`
**IDE-specific setup**: `INTELLIJ_SETUP.md` or `VSC_SETUP.md`

## 📋 Implementation Path

1. **Credentials** (2 min)
   - Create `credentials/.env` with `CR_PAT` and `USERNAME`
   - Add `GIT_HOST` / `REGISTRY` overrides if not using GitHub/GHCR

2. **Build & Run** (5 min)
   - Follow `BUILD_AND_RUN.md`

3. **Corporate / Proxy** (if needed)
   - Review `../deployment/CORPORATE_PIPELINE.md`

4. **Validation** (ongoing)
   - Use verification commands in `BUILD_AND_RUN.md`

## 🔧 IDE Setup

Before implementing:
- IntelliJ users: See `INTELLIJ_SETUP.md`
- VS Code users: See `VSC_SETUP.md`
- Command-line only: See `BUILD_AND_RUN.md`

## 📖 Related Documentation

- Investigation & analysis: See `../docs/`
- Deployment & pipeline: See `../deployment/`
- Quick reference: See `../reference/`
