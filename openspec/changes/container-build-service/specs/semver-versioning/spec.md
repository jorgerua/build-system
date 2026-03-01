## ADDED Requirements

### Requirement: Parse Conventional Commits for version bump type
The system SHALL parse commit messages following the Conventional Commits specification to determine the SemVer bump type.

#### Scenario: feat commit (minor bump)
- **WHEN** the commit message starts with `feat:` or `feat(scope):`
- **THEN** the system SHALL classify the bump as MINOR

#### Scenario: fix commit (patch bump)
- **WHEN** the commit message starts with `fix:` or `fix(scope):`
- **THEN** the system SHALL classify the bump as PATCH

#### Scenario: Breaking change via bang (major bump)
- **WHEN** the commit message starts with `feat!:` or `fix!:` or any `type!:`
- **THEN** the system SHALL classify the bump as MAJOR

#### Scenario: Breaking change via footer (major bump)
- **WHEN** the commit message body contains `BREAKING CHANGE:` in the footer
- **THEN** the system SHALL classify the bump as MAJOR

#### Scenario: Non-conventional commit (default patch)
- **WHEN** the commit message does not match any Conventional Commits pattern
- **THEN** the system SHALL default the bump to PATCH

### Requirement: Calculate highest bump from multiple commits
The system SHALL determine the highest-priority bump type when a push contains multiple commits (MAJOR > MINOR > PATCH).

#### Scenario: Push with mixed commit types
- **WHEN** a push contains commits with `fix:`, `feat:`, and `chore:`
- **THEN** the system SHALL use MINOR as the bump type (highest among the set)

#### Scenario: Push with breaking change among others
- **WHEN** a push contains a `feat!:` commit and several `fix:` commits
- **THEN** the system SHALL use MAJOR as the bump type

### Requirement: Persist version per project in TiDB
The system SHALL store the current SemVer version for each project in TiDB and update it after a successful build.

#### Scenario: Version incremented after successful build
- **WHEN** a build for project `api` succeeds and current version is `1.2.3` with a MINOR bump
- **THEN** the system SHALL update the version to `1.3.0` in TiDB

#### Scenario: First build of a new project
- **WHEN** no version record exists for a project in TiDB
- **THEN** the system SHALL initialize the version as `0.1.0`

### Requirement: Persist last processed SHA in TiDB
The system SHALL store the last successfully processed commit SHA in TiDB and update it after all builds for a push complete.

#### Scenario: SHA updated after successful processing
- **WHEN** all builds for a push event complete (success or marked as failed after retries)
- **THEN** the system SHALL update the last processed SHA in TiDB to the push's after SHA

### Requirement: Tag container images with SemVer
The system SHALL tag built container images with the calculated SemVer version.

#### Scenario: Image tagged with new version
- **WHEN** a container image is built successfully for project `api` with calculated version `2.0.0`
- **THEN** the image SHALL be pushed to the registry as `<registry>/api:2.0.0`
