package tidb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// BuildStateRepository manages the last processed SHA per repository.
type BuildStateRepository struct {
	db *sql.DB
}

// NewBuildStateRepository creates a BuildStateRepository.
func NewBuildStateRepository(db *sql.DB) *BuildStateRepository {
	return &BuildStateRepository{db: db}
}

// GetLastSHA returns the last processed commit SHA for a repo.
// Returns ("", nil) if no record exists (first run).
func (r *BuildStateRepository) GetLastSHA(ctx context.Context, repo string) (string, error) {
	var sha string
	err := r.db.QueryRowContext(ctx,
		`SELECT last_processed_sha FROM build_state WHERE repo = ?`, repo,
	).Scan(&sha)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get last sha: %w", err)
	}
	return sha, nil
}

// UpdateLastSHA upserts the last processed SHA for a repo.
func (r *BuildStateRepository) UpdateLastSHA(ctx context.Context, repo, sha string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO build_state (repo, last_processed_sha)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE last_processed_sha = VALUES(last_processed_sha)
	`, repo, sha)
	if err != nil {
		return fmt.Errorf("update last sha: %w", err)
	}
	return nil
}
