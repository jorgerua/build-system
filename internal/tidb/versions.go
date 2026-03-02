package tidb

import (
	"context"
	"database/sql"
	"fmt"
)

// VersionRepository manages project SemVer versions in TiDB.
type VersionRepository struct {
	db *sql.DB
}

// NewVersionRepository creates a VersionRepository.
func NewVersionRepository(db *sql.DB) *VersionRepository {
	return &VersionRepository{db: db}
}

// Get returns the current version for a project.
// Returns "0.1.0" and inserts the initial record if the project is new.
func (r *VersionRepository) Get(ctx context.Context, project string) (string, error) {
	const q = `
		INSERT INTO project_versions (project, version)
		VALUES (?, '0.1.0')
		ON DUPLICATE KEY UPDATE project = project
	`
	if _, err := r.db.ExecContext(ctx, q, project); err != nil {
		return "", fmt.Errorf("version upsert: %w", err)
	}

	var version string
	if err := r.db.QueryRowContext(ctx,
		`SELECT version FROM project_versions WHERE project = ?`, project,
	).Scan(&version); err != nil {
		return "", fmt.Errorf("version get: %w", err)
	}
	return version, nil
}

// Update sets the version for a project.
func (r *VersionRepository) Update(ctx context.Context, project, version string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE project_versions SET version = ? WHERE project = ?`,
		version, project,
	)
	if err != nil {
		return fmt.Errorf("version update: %w", err)
	}
	return nil
}
