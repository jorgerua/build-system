# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OCI Build System — a distributed build system that receives GitHub push webhooks, performs git operations, runs NX builds, and produces OCI-compatible images via buildah. Services are written in Go and communicate via NATS.

## Commands

All commands use `npm exec nx` (not a globally-installed `nx`) to avoid version mismatches.

### Build
```bash
# All projects
make build                          # Linux/Mac
.\build.ps1 build                   # Windows

# Specific project
npm exec nx run api-service:build
npm exec nx run worker-service:build
npm exec nx run shared:build

# Only affected projects (CI-friendly)
npm exec nx affected -- --target=build
```

### Test
```bash
# All projects
make test

# Single project
npm exec nx run api-service:test
npm exec nx run git-service:test

# Single test function (within a lib/app directory)
cd libs/git-service && go test -run TestFunctionName
cd libs/git-service && go test -run Property    # property-based tests only

# With coverage
npm exec nx run api-service:test --codeCoverage
make test-coverage
```

### Lint & Format
```bash
make lint           # lint all via NX
make format         # gofmt all .go files
```

### Run locally
```bash
make run-nats       # start NATS only (Docker)
make run-api        # run api-service locally (nx serve)
make run-worker     # run worker-service locally (nx serve)
make run            # start all via docker-compose
make stop
```

### Utility
```bash
make clean          # rm dist/, .nx/cache, nx reset
npm exec nx reset   # clear NX cache only
npm exec nx graph   # dependency graph
make install        # npm install + go mod download for all modules
```

## Architecture

```
GitHub Webhook → API Service → NATS (builds.webhook)
                                    ↓
                              Worker Service
                              ├─ BuildOrchestrator
                              │   ├─ Phase 1: git-service  (clone/pull repo)
                              │   ├─ Phase 2: nx-service   (run NX build)
                              │   └─ Phase 3: image-service (buildah OCI image)
                              └─ Publishes: builds.status, builds.complete
```

### NATS Subjects
| Subject | Publisher | Subscriber | Purpose |
|---|---|---|---|
| `builds.webhook` | api-service | worker-service | New build jobs |
| `builds.status` | worker-service | api-service | Status updates |
| `builds.complete` | worker-service | api-service | Job completion |

### Module Structure

Each app and library has its **own `go.mod`** (not a workspace-level go.mod). The Go module paths follow `github.com/jorgerua/build-system/<apps|libs>/<name>`.

```
apps/
  api-service/     # HTTP server: Gin router, webhook validation, build status queries
  worker-service/  # Worker pool + BuildOrchestrator: git sync → NX build → image build
libs/
  shared/          # Shared types (BuildJob, JobStatus, BuildPhase, Language, Config)
  nats-client/     # NATS connection wrapper
  git-service/     # Git clone/pull operations, repos cached at BUILD_CODE_CACHE_PATH
  nx-service/      # Runs NX builds, returns BuildResult
  image-service/   # Builds OCI images via buildah, auto-detects Dockerfile location
  cache-service/   # Manages dependency caches per language (java/dotnet/go)
```

### Dependency Injection

Both services use `go.uber.org/fx` for DI. The pattern in `main.go` is:
- `fx.Provide(...)` for constructors
- `fx.Invoke(...)` for side-effecting startup functions
- Lifecycle hooks via `lc.Append(fx.Hook{OnStart, OnStop})` for graceful shutdown

### Configuration

Config loads from `config.yaml` (path overridable via `CONFIG_PATH` env var). All config values can be overridden with environment variables (via viper's `AutomaticEnv`). Key env vars:

**API Service**: `SERVER_PORT` (8080), `NATS_URL` (nats://localhost:4222), `GITHUB_WEBHOOK_SECRET`, `LOG_LEVEL`, `AUTH_TOKEN`

**Worker Service**: `NATS_URL`, `WORKER_POOL_SIZE` (5), `WORKER_TIMEOUT` (3600s), `WORKER_MAX_RETRIES` (3), `BUILD_CODE_CACHE_PATH`, `BUILD_BUILD_CACHE_PATH`

### Build Pipeline (worker-service/orchestrator.go)

`BuildOrchestrator.ExecuteBuild` runs three sequential phases, each tracked as a `PhaseMetric` on the `BuildJob`:
1. **git_sync** — `gitService.SyncRepository` with 3 retries + exponential backoff
2. **nx_build** — `nxService.Build` with 1 retry; language auto-detected by nx-service
3. **image_build** — `imageService.BuildImage` with 2 retries; Dockerfile auto-detected; tags derived from repo name, commit hash, and branch

### Integration Tests

Uses Robot Framework in `tests/integration/`:
```bash
pip install robotframework robotframework-requests
cd tests/integration && robot .
robot webhook.robot   # run single suite
```

## NX Notes

- NX caches `build`, `test`, `lint` — use `--skip-nx-cache` to bypass
- `build` targets depend on `^build` (dependencies must build first)
- `test` targets depend on `build`
- Add new projects to `workspace.json` under `"projects"`
- Each project needs a `project.json` with `build`, `test`, `lint` targets
- When `go mod tidy` is needed, run it inside the specific lib/app directory, not the repo root
