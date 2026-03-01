# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Status

This repo is in the **specification phase** — no implementation code exists yet. The OpenSpec framework in `openspec/` contains all design artifacts. Implementation follows the task checklist in `openspec/changes/container-build-service/tasks.md`.

## OpenSpec Workflow

Use these slash commands to work with the specification:

- `/opsx:explore` — Think through a design problem or investigate requirements
- `/opsx:propose` — Propose a new change (generates proposal, design, tasks, specs)
- `/opsx:apply` — Implement tasks from an existing change
- `/opsx:archive` — Archive a completed change

Artifacts live under `openspec/changes/<change-name>/`:
- `proposal.md` — Why, what, capabilities, impact
- `design.md` — Architectural decisions with rationale
- `tasks.md` — Implementation checklist (numbered, e.g. `10.3`)
- `specs/` — Per-capability requirement scenarios

## Target Architecture

Two Go binaries, same module, shared `internal/` packages:

```
GitHub push event
  → webhook-server (HTTP)
    → validates HMAC-SHA256 signature (webhook secret only)
    → extracts installation_id from payload (does NOT generate token)
    → publishes build job to NATS JetStream:
        { repo_url, sha_after, commit_messages[], installation_id, published_at }

  → worker (NATS consumer, StatefulSet)
    → starts msg.InProgress() heartbeat goroutine (every 2 min, prevents false redelivery)
    → generates fresh GitHub App installation token (from installation_id)
    → clones repo at push SHA to /tmp/repo-<job-id>
      ↳ on clone failure: nack NATS message, abort
    → runs `nx affected --base=<last_processed_sha> --head=<push_sha>`
      ↳ first run (no SHA in TiDB): uses `git rev-list --max-parents=0 HEAD` as base
    → filters to apps/* projects only
    → parallel build dispatcher (configurable concurrency semaphore)
      per project:
        → two-phase claim (atomic INSERT into build_records with UNIQUE(project, sha))
          ↳ duplicate key → skip (already claimed or completed)
          ↳ stale pending (> 30 min) → conditional re-claim UPDATE
        → language detection (go.mod > pom.xml > build.gradle > *.csproj)
          ↳ unknown language: log warning, skip project (not a build failure)
        → SemVer bump from Conventional Commits (feat→minor, fix→patch, !→major, default→patch)
        → generate Dockerfile from template → write to /tmp/dockerfile-<job-id>-<project>
        → exec `buildah bud --storage-driver overlay --root <pvc-mount> -f <dockerfile> -t <registry>/<project>:<version> /tmp/repo-<job-id>`
        → exec `buildah push --storage-driver overlay --root <pvc-mount> <image> --authfile <secret-mount>`
        → delete temp Dockerfile; update version in TiDB on success
        → application-level retry: up to 3 attempts, exponential backoff, never nack NATS
          ↳ 3rd failure: mark build_record as failure, continue to next project
    → update last_processed_sha in TiDB (always, regardless of per-project outcomes)
    → ack NATS message
    → emit DogStatsD metrics to Datadog
```

## Key Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Two binaries | webhook-server + worker | Webhook must respond < 10s; workers scale independently |
| No ephemeral pods | `buildah bud` as subprocess in worker pod | Eliminates client-go, RBAC for pod creation, second clone, ConfigMaps |
| Token generation | Worker generates token before clone | Tokens expire in 1h; jobs in queue for > 1h would fail at clone with webhook-server approach |
| NATS JetStream | Durable consumer, AckWait 5min, MaxDelivers 3 | At-least-once delivery; MaxDelivers covers worker crashes, not build retries |
| NATS heartbeat | msg.InProgress() every 2 min | Keeps AckWait short (5 min) while allowing multi-hour builds without false redelivery |
| Application retry | Per-project, max 3×, exponential backoff, never nack | Reuses local clone; avoids re-cloning and re-running nx affected for projects already succeeded |
| Idempotency | Two-phase claim: UNIQUE(project, commit_sha) + stale timeout (30 min) | Atomic INSERT prevents TOCTOU race; stale timeout recovers from crashed workers |
| SHA advancement | last_processed_sha always advances (even on partial build failure) | Prevents unbounded nx affected diff accumulation; failed projects rebuild on next code change |
| Build context | Monorepo root (/tmp/repo-<job-id>) reused from clone | Go projects may import from libs/; same clone used for nx affected and buildah bud |
| Generated Dockerfiles | Templates per language; ignores any Dockerfile in repo | Standardized builds; existing Dockerfiles may not be multi-stage or production-ready |
| Buildah storage driver | Overlay (CAP_SETUID + CAP_SETFCAP) with VFS fallback | Overlay gives layer deduplication; VFS for clusters that prohibit capabilities |
| Worker deployment | StatefulSet with volumeClaimTemplates | Each worker pod gets its own RWO buildah-storage PVC automatically |
| SemVer initial version | 0.1.0 | New projects start at pre-1.0; version stored per project in TiDB |

## Tech Stack

| Concern | Library |
|---|---|
| DI | `go.uber.org/fx` |
| Logging | `go.uber.org/zap` (JSON to stdout) |
| Config | `spf13/viper` (config.yaml + env overrides) |
| Messaging | `github.com/nats-io/nats.go` (JetStream) |
| Database | `go-sql-driver/mysql` (TiDB) |
| Metrics | `github.com/DataDog/datadog-go` (DogStatsD) |

**Not used**: `k8s.io/client-go` — builds run via `buildah bud` subprocess, no K8s pods are created.

## Module Structure (to be created)

```
cmd/webhook-server/main.go   # fx app wiring for HTTP service
cmd/worker/main.go           # fx app wiring for NATS consumer
internal/
  config/                    # Viper setup, shared config struct
  nats/                      # JetStream publisher + durable consumer + heartbeat
  tidb/                      # Repository layer (versions, SHA, build_records two-phase claim)
  github/                    # App JWT, installation token generation, webhook HMAC validation
  detection/                 # Language detection by marker files (Go > Java > .NET)
  templates/                 # Dockerfile templates + renderer (text/template)
  semver/                    # Conventional Commits parser + SemVer bump + aggregation
  buildah/                   # buildah bud/push executor + storage driver detection
  orchestrator/              # Build pipeline coordinator (claim → detect → version → build → retry)
templates/                   # Dockerfile templates (Go, Java/Maven, Java/Gradle, .NET)
deploy/
  worker.Dockerfile          # Multi-stage: Go builder + Node.js runtime + buildah + fuse-overlayfs + git + Nx CLI
  webhook-server.Dockerfile  # Multi-stage: Go builder + distroless/static runtime
  k8s/                       # StatefulSet (worker), Deployment (webhook-server), PVCs, RBAC, Secrets, ConfigMap
```

## TiDB Schema (planned)

```sql
project_versions (project VARCHAR, version VARCHAR, updated_at TIMESTAMP)
build_state      (repo VARCHAR, last_processed_sha CHAR(40), updated_at TIMESTAMP)
build_records    (id BIGINT PK, project VARCHAR, commit_sha CHAR(40),
                  status ENUM('pending','success','failure'),
                  claimed_at TIMESTAMP, updated_at TIMESTAMP,
                  UNIQUE KEY uk_project_sha (project, commit_sha))
```

## DogStatsD Metrics

| Metric | Type | Tags |
|---|---|---|
| `build.duration` | histogram | project, language, status=success\|failure |
| `build.status` | count | project, status=success\|failure |
| `build.queue_wait_time` | histogram | — (uses `published_at` from NATS payload) |
| `build.projects_affected` | gauge | — |
| `build.retry_count` | count | project, attempt=1\|2\|3 |

## Commands (once implemented)

```bash
# Run tests for a single package
cd internal/<pkg> && go test -run TestName

# Run all tests
go test ./...

# Build binaries
go build ./cmd/webhook-server
go build ./cmd/worker
```

All `go mod tidy` and `go test` commands must run from within the module directory (not repo root — there is no workspace-level go.mod).
