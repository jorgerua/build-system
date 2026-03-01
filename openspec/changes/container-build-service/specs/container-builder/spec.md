## ADDED Requirements

### Requirement: Build container image via buildah in worker pod
The system SHALL execute `buildah bud` as a subprocess within the worker pod to build each container image, using the local repository clone as build context. No additional Kubernetes pods SHALL be created for builds.

#### Scenario: Successful build
- **WHEN** a build is triggered for a project
- **THEN** the system SHALL write the generated Dockerfile to a temporary file, execute `buildah bud -f <dockerfile-path> -t <registry>/<project>:<version> /tmp/repo-<job-id>`, capture stdout/stderr for logging, and delete the temporary Dockerfile file after completion

#### Scenario: Build failure
- **WHEN** `buildah bud` exits with a non-zero status
- **THEN** the system SHALL log the captured stderr output, delete the temporary Dockerfile file, and report the failure to the orchestrator for retry

### Requirement: Push image to container registry
The system SHALL execute `buildah push` to push the built image to the configured container registry using the registry credentials secret mounted in the worker pod.

#### Scenario: Image pushed to registry
- **WHEN** `buildah bud` completes successfully
- **THEN** the system SHALL execute `buildah push <registry>/<project>:<version> --authfile <registry-secret-mount-path>` to push the image

#### Scenario: Push failure
- **WHEN** `buildah push` exits with a non-zero status
- **THEN** the system SHALL log the error and report the failure to the orchestrator for retry

### Requirement: Build context scoped to monorepo root
The system SHALL use the full monorepo root directory (the local clone used for `nx affected`) as the build context for `buildah bud`, making all monorepo files available during the build.

#### Scenario: Context set to monorepo root
- **WHEN** building project `apps/api`
- **THEN** `buildah bud` SHALL receive the monorepo root clone directory as build context, making shared libraries under `libs/` available during compilation

### Requirement: Buildah layer cache via PVC
The system SHALL configure buildah to use a PVC-mounted directory as its storage root (`--root`), persisting image layers across builds within the same worker pod.

#### Scenario: Layer cache reused
- **WHEN** a project is rebuilt and base image layers are already present in the buildah storage PVC
- **THEN** buildah SHALL reuse the cached layers, reducing build time

#### Scenario: Cache populated on first build
- **WHEN** a build runs with an empty buildah storage PVC
- **THEN** buildah SHALL pull and cache the base image layers into the PVC storage root for future reuse

### Requirement: Configure registry credentials
The system SHALL provide container registry credentials to buildah via a Kubernetes Secret mounted in the worker pod, referenced by `--authfile`.

#### Scenario: Registry authentication
- **WHEN** `buildah push` is executed
- **THEN** buildah SHALL authenticate to the registry using the credentials file mounted from the Kubernetes Secret

### Requirement: Buildah capabilities in worker pod SecurityContext
The worker pod SHALL have the necessary Linux capabilities configured in its SecurityContext to allow buildah to operate with overlay filesystem support.

#### Scenario: Overlay storage driver available
- **WHEN** the worker pod has `CAP_SETUID` and `CAP_SETFCAP` in its SecurityContext
- **THEN** buildah SHALL use the overlay storage driver for efficient layer caching

#### Scenario: Rootless fallback (no capabilities)
- **WHEN** the cluster security policy prohibits additional capabilities
- **THEN** buildah SHALL fall back to `--storage-driver vfs`, operating without kernel namespace requirements at the cost of layer deduplication performance
