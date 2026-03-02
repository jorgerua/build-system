package tidb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// BuildStatus represents the status of a build record.
type BuildStatus string

const (
	BuildStatusPending BuildStatus = "pending"
	BuildStatusSuccess BuildStatus = "success"
	BuildStatusFailure BuildStatus = "failure"
)

// BuildRecord represents a row in build_records.
type BuildRecord struct {
	ID        int64
	Project   string
	CommitSHA string
	Status    BuildStatus
	ClaimedAt time.Time
}

// BuildRecordRepository implements the two-phase claim idempotency pattern.
type BuildRecordRepository struct {
	db *sql.DB
}

// NewBuildRecordRepository creates a BuildRecordRepository.
func NewBuildRecordRepository(db *sql.DB) *BuildRecordRepository {
	return &BuildRecordRepository{db: db}
}

// Claim attempts to atomically claim a (project, commitSHA) build slot.
//
// Returns (true, nil) when the claim succeeds (this worker owns the build).
// Returns (false, nil) when the build should be skipped (already claimed,
// completed, or another worker won a re-claim race).
func (r *BuildRecordRepository) Claim(ctx context.Context, project, commitSHA string, staleThreshold time.Duration) (bool, error) {
	// Phase 1: atomic INSERT. INSERT … ON DUPLICATE KEY UPDATE with a no-op
	// update returns affected=1 on insert, affected=0 on duplicate.
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO build_records (project, commit_sha, status)
		VALUES (?, ?, 'pending')
		ON DUPLICATE KEY UPDATE id = id
	`, project, commitSHA)
	if err != nil {
		return false, fmt.Errorf("build record insert: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	if affected == 1 {
		// Fresh insert — this worker owns the build.
		return true, nil
	}

	// Phase 2: duplicate key — read the existing record.
	var rec BuildRecord
	err = r.db.QueryRowContext(ctx,
		`SELECT id, status, claimed_at FROM build_records WHERE project = ? AND commit_sha = ?`,
		project, commitSHA,
	).Scan(&rec.ID, &rec.Status, &rec.ClaimedAt)
	if err != nil {
		return false, fmt.Errorf("build record read: %w", err)
	}

	switch rec.Status {
	case BuildStatusSuccess, BuildStatusFailure:
		// Already completed — skip.
		return false, nil
	case BuildStatusPending:
		if time.Since(rec.ClaimedAt) < staleThreshold {
			// Recent pending — another worker is actively processing.
			return false, nil
		}
		// Stale pending — attempt conditional re-claim.
		upd, err := r.db.ExecContext(ctx, `
			UPDATE build_records
			SET claimed_at = NOW()
			WHERE project = ? AND commit_sha = ? AND status = 'pending' AND claimed_at = ?
		`, project, commitSHA, rec.ClaimedAt)
		if err != nil {
			return false, fmt.Errorf("re-claim update: %w", err)
		}
		rows, _ := upd.RowsAffected()
		return rows == 1, nil
	}

	return false, nil
}

// SetStatus updates the final status (success or failure) of a build record.
func (r *BuildRecordRepository) SetStatus(ctx context.Context, project, commitSHA string, status BuildStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE build_records SET status = ? WHERE project = ? AND commit_sha = ?`,
		string(status), project, commitSHA,
	)
	if err != nil {
		return fmt.Errorf("set build status: %w", err)
	}
	return nil
}

// GetStatus returns the current status of a build record, or ErrNoRows if not found.
func (r *BuildRecordRepository) GetStatus(ctx context.Context, project, commitSHA string) (BuildStatus, error) {
	var status BuildStatus
	err := r.db.QueryRowContext(ctx,
		`SELECT status FROM build_records WHERE project = ? AND commit_sha = ?`,
		project, commitSHA,
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", sql.ErrNoRows
	}
	if err != nil {
		return "", fmt.Errorf("get build status: %w", err)
	}
	return status, nil
}
