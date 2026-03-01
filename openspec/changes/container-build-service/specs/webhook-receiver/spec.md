## ADDED Requirements

### Requirement: Receive GitHub push webhooks
The system SHALL expose an HTTP endpoint that receives push event webhooks from GitHub.

#### Scenario: Valid push event to main branch
- **WHEN** GitHub sends a push event webhook for the main branch with a valid signature
- **THEN** the system SHALL respond with HTTP 202 Accepted and publish a build job to the NATS JetStream queue

#### Scenario: Push event to non-main branch
- **WHEN** GitHub sends a push event webhook for a branch other than main
- **THEN** the system SHALL respond with HTTP 200 OK and take no further action

#### Scenario: Non-push event
- **WHEN** GitHub sends a webhook event that is not a push event
- **THEN** the system SHALL respond with HTTP 200 OK and ignore the event

### Requirement: Validate webhook signature
The system SHALL validate the HMAC-SHA256 signature of every incoming webhook using the GitHub App webhook secret.

#### Scenario: Valid signature
- **WHEN** a webhook request arrives with a valid `X-Hub-Signature-256` header
- **THEN** the system SHALL accept and process the request

#### Scenario: Invalid signature
- **WHEN** a webhook request arrives with an invalid or missing `X-Hub-Signature-256` header
- **THEN** the system SHALL respond with HTTP 401 Unauthorized and discard the request

### Requirement: Extract installation ID from webhook payload
The system SHALL extract the GitHub App `installation_id` from the webhook payload and include it in the build job message. The webhook server SHALL NOT generate installation tokens — token generation is deferred to the worker at clone time.

#### Scenario: Installation ID extracted
- **WHEN** a valid push event is received
- **THEN** the system SHALL extract the `installation.id` field from the GitHub webhook payload and include it as `installation_id` in the NATS build job message

### Requirement: Publish build job to NATS
The system SHALL publish a build job message to NATS JetStream containing the repository URL, commit SHA, installation ID, and commit messages from the push.

#### Scenario: Job published successfully
- **WHEN** webhook signature is valid and event is a push to main
- **THEN** the system SHALL publish a JSON message to the configured NATS JetStream subject with repository, SHA (after), commit messages, installation ID, and `published_at` (RFC3339 UTC timestamp of publish time)

#### Scenario: NATS unavailable
- **WHEN** NATS JetStream is unreachable during publish
- **THEN** the system SHALL respond with HTTP 503 Service Unavailable and log the error
