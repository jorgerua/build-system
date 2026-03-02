## 1. Project Scaffolding

- [x] 1.1 Initialize Go module (`go mod init`) with project structure: `cmd/webhook-server/`, `cmd/worker/`, `internal/`, `templates/`, `deploy/k8s/`
- [x] 1.2 Add core dependencies: fx, zap, viper, nats.go, go-sql-driver/mysql, datadog-go (note: client-go is NOT required — builds run via buildah subprocess, not K8s pods)
- [x] 1.3 Create shared fx module for config (Viper), logging (zap), and Datadog client setup
- [x] 1.4 Create `deploy/worker.Dockerfile`: multi-stage build with (a) Go builder stage compiling `cmd/worker` binary, and (b) runtime stage based on a Node.js image (e.g., `node:20-bookworm-slim`) with `buildah`, `fuse-overlayfs`, and `git` installed via apt, and Nx CLI installed globally via npm; copy the compiled worker binary from the Go builder stage
- [x] 1.5 Create `deploy/webhook-server.Dockerfile`: multi-stage build with (a) Go builder stage compiling `cmd/webhook-server` binary, and (b) distroless runtime stage (`gcr.io/distroless/static-debian12`); copy the compiled binary from the Go builder stage

## 2. NATS Queue Integration

- [x] 2.1 Implement NATS JetStream connection module with stream and durable consumer setup; configure consumer with `AckWait: 5min` and `MaxDelivers: 3` (crash recovery only — build retries are application-level)
- [x] 2.2 Implement publisher (used by webhook-server) that publishes build job JSON messages
- [x] 2.3 Implement subscriber (used by worker) with ack/nack handling, message deserialization, and periodic `msg.InProgress()` heartbeat (every 2 minutes) to prevent false redelivery during long-running jobs

## 3. TiDB Persistence

- [x] 3.1 Define schema: `project_versions` table (project name, version, updated_at) and `build_state` table (repo, last_processed_sha, updated_at)
- [x] 3.2 Implement repository layer for version CRUD (get current version, update version, initialize at 0.1.0)
- [x] 3.3 Implement repository layer for last processed SHA (get, update)
- [x] 3.4 Implement build record storage for two-phase claim idempotency: schema with UNIQUE constraint on `(project, commit_sha)`, status ENUM `('pending','success','failure')`, `claimed_at` timestamp; repository methods: atomic INSERT (returning affected rows), status lookup, conditional re-claim UPDATE for stale records (claimed_at older than configurable threshold, default 30 min), and final status UPDATE to success/failure

## 4. GitHub App Authentication

- [x] 4.1 Implement GitHub App JWT generation from private key and app ID
- [x] 4.2 Implement installation token generation via GitHub API
- [x] 4.3 Implement webhook HMAC-SHA256 signature validation

## 5. Webhook Server

- [x] 5.1 Create HTTP server (fx lifecycle) with health check and webhook endpoint
- [x] 5.2 Implement push event handler: validate signature, filter branch (main only), parse payload (repo, SHA after, commit messages)
- [x] 5.3 Extract `installation_id` from webhook payload and publish build job to NATS JetStream (payload: repo URL, SHA after, commit messages, installation_id, published_at RFC3339 UTC timestamp — no token generation in webhook server)
- [x] 5.4 Wire up fx app in `cmd/webhook-server/main.go`

## 6. Language Detection

- [x] 6.1 Implement detector that scans project directory for marker files (go.mod, pom.xml, build.gradle, build.gradle.kts, *.csproj)
- [x] 6.2 Implement priority resolution (Go > Java > .NET) and build tool sub-detection (Maven vs Gradle)
- [x] 6.3 Return structured result (language, build tool) or error for unknown language

## 7. Dockerfile Template Engine

- [x] 7.1 Create Go multi-stage Dockerfile template (golang → distroless): COPY monorepo root, build via `go build ./apps/<project>/...`
- [x] 7.2 Create Java Maven multi-stage Dockerfile template (maven → eclipse-temurin JRE): COPY project directory, run `mvn package -DskipTests`
- [x] 7.3 Create Java Gradle multi-stage Dockerfile template (gradle → eclipse-temurin JRE): COPY project directory, run `gradle build -x test`
- [x] 7.4 Create .NET multi-stage Dockerfile template (dotnet SDK → aspnet runtime): COPY project directory, run `dotnet restore` and `dotnet publish`
- [x] 7.5 Implement template renderer that injects project variables (name, project subpath within monorepo, output artifact name) into templates via Go text/template

## 8. SemVer Version Calculator

- [x] 8.1 Implement Conventional Commits parser: extract type, scope, bang, and BREAKING CHANGE footer
- [x] 8.2 Implement bump type resolution from parsed commit (feat→minor, fix→patch, bang/BREAKING→major, default→patch)
- [x] 8.3 Implement highest-bump aggregation across multiple commits in a push
- [x] 8.4 Implement SemVer increment logic (apply bump to current version, reset lower components)

## 9. Buildah Builder

- [x] 9.1 Implement buildah build executor: write generated Dockerfile to `/tmp/dockerfile-<job-id>-<project>`; exec `buildah bud --storage-driver overlay --root <buildah-storage-mount> -f <dockerfile-path> -t <registry>/<project>:<version> /tmp/repo-<job-id>`; capture stdout/stderr; delete temp Dockerfile on completion or failure
- [x] 9.2 Implement buildah push: exec `buildah push --storage-driver overlay --root <buildah-storage-mount> <registry>/<project>:<version> --authfile <registry-secret-mount>`; capture stdout/stderr for logging
- [x] 9.3 Implement storage driver fallback: detect overlay capability at startup; use `--storage-driver vfs` if overlay unavailable; log selected driver
- [x] 9.4 Configure buildah storage root to point to the buildah-storage PVC mount path (configurable via Viper)

## 10. Build Orchestrator (Worker)

- [x] 10.1 Implement job consumer: subscribe to NATS, deserialize build job, validate payload; start `msg.InProgress()` heartbeat goroutine (every 2 minutes) immediately upon receipt
- [x] 10.2 Implement local repo clone for analysis: generate a fresh installation token from the job's `installation_id` using GitHub App credentials immediately before cloning; clone repo to a temporary directory; checkout the specific SHA; on clone failure, stop the heartbeat goroutine, nack the NATS message, and abort job processing — this clone is reused as buildah build context
- [x] 10.3 Implement `nx affected` execution: resolve base SHA (use `last_processed_sha` from TiDB if present; otherwise run `git rev-list --max-parents=0 HEAD` on the local clone to get the repository's initial commit); run `nx affected --base=<base-sha> --head=<push-sha> --plain`; parse output; filter to projects whose root is under `apps/`
- [x] 10.4 Implement parallel build dispatcher with configurable concurrency semaphore
- [x] 10.5 Implement two-phase claim idempotency (called first, per project, before build): attempt atomic INSERT of pending record; on duplicate key, read existing status — skip if success/failure; skip if pending with recent claimed_at; attempt conditional re-claim UPDATE if pending record is stale (claimed_at > threshold); skip if re-claim UPDATE affects 0 rows (lost race); proceed with build only when claim is confirmed owned
- [x] 10.6 Implement per-project build pipeline (called after successful claim in 10.5): if language detection returns unknown language, log a warning with project name and skip to next project without consuming a retry attempt; otherwise: calculate version → generate Dockerfile content → exec buildah bud (using local repo clone as context) → exec buildah push → update version in TiDB on success
- [x] 10.7 Implement application-level retry per project: up to 3 attempts with exponential backoff; retries run within the same worker using the existing local clone; on 3rd failure mark build_record as `failure` and continue to next project — do NOT nack NATS message
- [x] 10.8 Update last processed SHA in TiDB to the push's after-SHA upon completion of all project builds, regardless of individual build outcomes (success or permanent failure); ack the NATS message
- [x] 10.9 Wire up fx app in `cmd/worker/main.go`

## 11. Observability

- [x] 11.1 Instrument build pipeline with Datadog DogStatsD metrics: `build.duration` (histogram, tags: project, language, status=success|failure); `build.status` (count, tags: project, status=success|failure); `build.queue_wait_time` (histogram, calculated from `published_at` field in NATS payload to processing start time); `build.projects_affected` (gauge, value = count of affected projects under apps/); `build.retry_count` (count, tags: project, attempt=1|2|3)
- [x] 11.2 Add structured zap logging across all phases (job received, clone started, nx affected result, build started per project, build completed/failed per project, job completed, errors with context including project name and commit SHA)

## 12. Kubernetes Manifests

- [x] 12.1 Create Deployment manifest for webhook-server (with service, health checks)
- [x] 12.2 Create StatefulSet manifest for worker: image built from `deploy/worker.Dockerfile`; mount nx-cache PVC and buildah-storage PVC via volumeClaimTemplates; SecurityContext with `CAP_SETUID` and `CAP_SETFCAP` for buildah overlay; mount registry secret and GitHub App credentials secret
- [x] 12.3 Create PVC manifests: nx-cache (RWX, shared across workers — use StorageClass with ReadWriteMany); buildah-storage defined as volumeClaimTemplates in the worker StatefulSet (RWO, one PVC per pod, automatically provisioned per replica); add a comment in the manifest documenting the RWX fallback: if the cluster lacks RWX storage, configure node affinity to pin all worker pods to the same node and use a single RWO PVC for nx-cache
- [x] 12.4 Create ServiceAccount with minimal RBAC (no pod creation permissions required; only standard workload permissions)
- [x] 12.5 Create Secret templates for GitHub App credentials and container registry config
- [x] 12.6 Create ConfigMap for Viper configuration (NATS URL, TiDB DSN, registry, concurrency limit, buildah storage path, stale claim threshold, NATS AckWait, etc.)

## 13. Testing

- [x] 13.1 Unit tests for Conventional Commits parser and SemVer calculator
- [x] 13.2 Unit tests for language detection
- [x] 13.3 Unit tests for Dockerfile template rendering
- [x] 13.4 Unit tests for webhook signature validation
- [x] 13.5 Integration test for NATS publish/subscribe flow
- [x] 13.6 Integration test for TiDB version and SHA persistence
