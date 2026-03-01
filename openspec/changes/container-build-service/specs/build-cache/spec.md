## ADDED Requirements

### Requirement: Nx computation cache via PVC
The system SHALL mount a shared PVC (ReadWriteMany) to persist the Nx computation cache (`.nx-cache`) across worker executions.

#### Scenario: Cache reused between builds
- **WHEN** a worker runs `nx affected` and previous computation results exist in the PVC cache
- **THEN** Nx SHALL reuse cached computation results, reducing analysis time

#### Scenario: Cache populated on first run
- **WHEN** a worker runs `nx affected` for the first time with an empty cache PVC
- **THEN** Nx SHALL compute all results and store them in the PVC for future use

### Requirement: Buildah layer cache via per-worker PVC
The system SHALL mount a dedicated PVC (ReadWriteOnce) per worker pod as the buildah storage root (`--root`), persisting image layers across builds executed by that worker.

#### Scenario: Layer cache reused within same worker
- **WHEN** the same worker pod builds projects `apps/api` and `apps/worker`, both using the same Go base image
- **THEN** buildah SHALL reuse the base image layers already present in the worker's storage PVC, avoiding a redundant pull

#### Scenario: Cache invalidation on base image update
- **WHEN** a Dockerfile template is updated to use a newer base image version
- **THEN** buildah SHALL detect the layer mismatch, pull the new base image, and cache the new layers in the PVC storage root

### Requirement: PVC storage class compatibility
The system SHALL use ReadWriteMany for the nx-cache PVC (shared across workers) and ReadWriteOnce for each buildah-storage PVC (one per worker pod). If RWX is unavailable for nx-cache, the system SHALL document fallback to node affinity with ReadWriteOnce.

#### Scenario: nx-cache with RWX storage
- **WHEN** the Kubernetes cluster has a ReadWriteMany-capable storage class
- **THEN** the system SHALL create the nx-cache PVC with RWX access mode, allowing concurrent access from multiple worker pods

#### Scenario: buildah-storage with RWO storage
- **WHEN** a worker pod starts
- **THEN** the system SHALL mount a dedicated RWO PVC as the buildah storage root, providing layer caching scoped to that worker instance

#### Scenario: nx-cache RWX unavailable (fallback)
- **WHEN** only ReadWriteOnce storage is available
- **THEN** the system deployment documentation SHALL describe configuring node affinity to pin all worker pods to the same node, sharing a single RWO nx-cache PVC
