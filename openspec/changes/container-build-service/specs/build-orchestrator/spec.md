## ADDED Requirements

### Requirement: Consume build jobs from NATS
The system SHALL consume build job messages from a NATS JetStream durable consumer.

#### Scenario: Job received and acknowledged
- **WHEN** a build job message is available on the queue
- **THEN** the worker SHALL consume the message, process it, and acknowledge it upon completion

#### Scenario: Worker restarts mid-processing
- **WHEN** a worker crashes while processing a job that has not been acknowledged
- **THEN** NATS JetStream SHALL redeliver the message to another available worker

### Requirement: Clone repository at specific commit
The system SHALL generate a fresh GitHub App installation token immediately before cloning, using the `installation_id` from the build job, then clone the monorepo to a temporary local directory at the exact commit SHA for analysis purposes (nx affected and language detection).

#### Scenario: Successful clone
- **WHEN** the worker receives a build job with a valid SHA and installation ID
- **THEN** the system SHALL generate a fresh installation token from the GitHub App credentials, clone the repository to a temporary directory, and checkout the specified commit SHA

#### Scenario: Clone failure
- **WHEN** the clone operation fails (network error, invalid token, etc.)
- **THEN** the system SHALL nack the message for retry

### Requirement: Detect affected projects via nx affected
The system SHALL execute `nx affected` comparing the last processed SHA (from TiDB) with the current push SHA to determine which projects changed.

#### Scenario: Projects affected
- **WHEN** `nx affected` returns a list of affected projects
- **THEN** the system SHALL filter the list to include only projects under the `apps/` directory

#### Scenario: No projects affected
- **WHEN** `nx affected` returns an empty list or no projects under `apps/` are affected
- **THEN** the system SHALL acknowledge the job, update the last processed SHA, and take no further build action

#### Scenario: First run (no previous SHA in DB)
- **WHEN** no last processed SHA exists in TiDB for this repository
- **THEN** the system SHALL resolve the repository's initial commit via `git rev-list --max-parents=0 HEAD` on the local clone and use it as `--base` for `nx affected`, causing all currently-existing projects under `apps/` to be treated as affected

### Requirement: Orchestrate parallel builds
The system SHALL dispatch one buildah build per affected project, executing builds in parallel up to a configurable concurrency limit.

#### Scenario: Multiple projects affected
- **WHEN** 5 projects are affected and max concurrency is 3
- **THEN** the system SHALL run 3 builds in parallel and dispatch the remaining 2 as slots become available

#### Scenario: All builds succeed
- **WHEN** all parallel builds for a job complete successfully
- **THEN** the system SHALL update the last processed SHA in TiDB to the push's after-SHA and acknowledge the NATS message

#### Scenario: Some builds fail permanently
- **WHEN** one or more projects reach max retries and are marked as permanent failure
- **THEN** the system SHALL still update the last processed SHA in TiDB to the push's after-SHA, acknowledge the NATS message, and continue — the failed projects will be rebuilt automatically by the next push that includes changes to those projects

### Requirement: Application-level retry per project with NATS heartbeat
The system SHALL retry failed builds at the application level within the same worker, up to 3 attempts per project with exponential backoff. The NATS message SHALL remain in-progress throughout via periodic heartbeats, and SHALL only be acknowledged once all projects are fully processed.

#### Scenario: Transient build failure — first or second attempt
- **WHEN** a buildah build fails for the first or second time
- **THEN** the system SHALL retry the build with exponential backoff without nacking the NATS message

#### Scenario: Persistent build failure — third attempt
- **WHEN** a build fails 3 times consecutively
- **THEN** the system SHALL update the build_record status to `failure`, log the error with full context, and continue processing remaining projects — the NATS message is NOT nacked

#### Scenario: NATS heartbeat during long-running job
- **WHEN** a worker is processing a job that takes longer than the NATS AckWait timeout
- **THEN** the worker SHALL send periodic `msg.InProgress()` calls (every 2 minutes) to prevent NATS from treating the message as timed out and redelivering it to another worker

### Requirement: Idempotent build processing
The system SHALL use an atomic two-phase claim to ensure that only one worker processes a given project+SHA combination, even under concurrent NATS redelivery.

#### Scenario: First worker claims the build
- **WHEN** a build job arrives for a project+SHA with no existing record
- **THEN** the system SHALL insert a `pending` record (enforced by UNIQUE constraint on project+commit_sha) and proceed with the build

#### Scenario: Concurrent duplicate job received
- **WHEN** a second worker attempts to claim a project+SHA that is already `pending`
- **THEN** the INSERT SHALL fail with a duplicate key error and the worker SHALL skip the build

#### Scenario: Build already completed
- **WHEN** a build job arrives for a project+SHA that already has a `success` or `failure` record
- **THEN** the system SHALL skip the build and acknowledge the job

#### Scenario: Stale pending record (worker crash recovery)
- **WHEN** a build job arrives for a project+SHA with a `pending` record older than the configured stale threshold (default: 30 minutes)
- **THEN** the system SHALL attempt to re-claim via a conditional UPDATE on `claimed_at`; if the UPDATE affects 1 row the worker proceeds with the build, otherwise another worker won the re-claim race and this worker skips
