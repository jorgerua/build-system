## ADDED Requirements

### Requirement: Structured JSON logging
The system SHALL emit all logs as structured JSON to stdout using zap, with consistent fields across all components.

#### Scenario: Build lifecycle logged
- **WHEN** a build job is processed
- **THEN** the system SHALL emit structured log entries for: job received, clone started, nx affected result, build started (per project), build completed/failed (per project), and job completed

#### Scenario: Error logging with context
- **WHEN** an error occurs during any phase
- **THEN** the log entry SHALL include the error message, stack trace, project name (if applicable), commit SHA, and build attempt number

### Requirement: Build duration metric
The system SHALL emit a `build.duration` histogram metric to Datadog via DogStatsD, measuring the time from build start to completion for each project.

#### Scenario: Successful build duration
- **WHEN** a buildah build completes successfully
- **THEN** the system SHALL emit `build.duration` with tags `project:<name>`, `language:<lang>`, and `status:success`

#### Scenario: Failed build duration
- **WHEN** a buildah build fails
- **THEN** the system SHALL emit `build.duration` with tags `project:<name>`, `language:<lang>`, and `status:failure`

### Requirement: Build status metric
The system SHALL emit a `build.status` count metric to Datadog for each build attempt, tagged with success or failure.

#### Scenario: Build success counted
- **WHEN** a build completes successfully
- **THEN** the system SHALL increment `build.status` with tag `status:success` and `project:<name>`

#### Scenario: Build failure counted
- **WHEN** a build fails (each attempt)
- **THEN** the system SHALL increment `build.status` with tag `status:failure` and `project:<name>`

### Requirement: Queue wait time metric
The system SHALL emit a `build.queue_wait_time` histogram metric measuring the time between job publish and job processing start.

#### Scenario: Queue latency measured
- **WHEN** a worker picks up a job from NATS
- **THEN** the system SHALL calculate the duration between the job's publish timestamp and the current time, emitting it as `build.queue_wait_time`

### Requirement: Projects affected metric
The system SHALL emit a `build.projects_affected` gauge metric indicating how many projects were detected as affected per push event.

#### Scenario: Affected count reported
- **WHEN** `nx affected` completes and returns the filtered list of buildable projects
- **THEN** the system SHALL emit `build.projects_affected` with the count as the gauge value

### Requirement: Retry count metric
The system SHALL emit a `build.retry_count` count metric for each retry attempt.

#### Scenario: Retry counted
- **WHEN** a build is retried after failure
- **THEN** the system SHALL increment `build.retry_count` with tag `project:<name>` and `attempt:<number>`
