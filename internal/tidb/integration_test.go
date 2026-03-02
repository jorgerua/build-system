package tidb_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jorgerua/build-system/container-build-service/internal/tidb"
)

// TestTiDBVersionAndSHA tests version and SHA persistence.
// Requires a running TiDB/MySQL instance.
// Set TIDB_DSN env var to enable (e.g., TIDB_DSN=root@tcp(localhost:4000)/testdb).
func TestTiDBVersionAndSHA(t *testing.T) {
	dsn := os.Getenv("TIDB_DSN")
	if dsn == "" {
		t.Skip("TIDB_DSN not set — skipping integration test")
	}

	db, err := sql.Open("mysql", dsn+"?parseTime=true&multiStatements=true")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	// Apply schema.
	if _, err := db.Exec(tidb.Schema); err != nil {
		t.Fatalf("schema: %v", err)
	}

	ctx := context.Background()
	project := "test-project-" + time.Now().Format("20060102150405")
	repo := "https://github.com/test/repo"

	// Version repository.
	vr := tidb.NewVersionRepository(db)

	version, err := vr.Get(ctx, project)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	if version != "0.1.0" {
		t.Errorf("initial version: got %q, want 0.1.0", version)
	}

	if err := vr.Update(ctx, project, "1.2.3"); err != nil {
		t.Fatalf("update version: %v", err)
	}
	version, err = vr.Get(ctx, project)
	if err != nil {
		t.Fatalf("get updated version: %v", err)
	}
	if version != "1.2.3" {
		t.Errorf("updated version: got %q, want 1.2.3", version)
	}

	// Build state repository.
	bsr := tidb.NewBuildStateRepository(db)

	sha, err := bsr.GetLastSHA(ctx, repo)
	if err != nil {
		t.Fatalf("get last sha: %v", err)
	}
	if sha != "" {
		t.Errorf("initial sha should be empty, got %q", sha)
	}

	if err := bsr.UpdateLastSHA(ctx, repo, "abc123"); err != nil {
		t.Fatalf("update last sha: %v", err)
	}
	sha, err = bsr.GetLastSHA(ctx, repo)
	if err != nil {
		t.Fatalf("get sha after update: %v", err)
	}
	if sha != "abc123" {
		t.Errorf("sha: got %q, want abc123", sha)
	}

	// Build record two-phase claim.
	brr := tidb.NewBuildRecordRepository(db)
	commitSHA := "def456" + time.Now().Format("150405")

	claimed, err := brr.Claim(ctx, project, commitSHA, 30*time.Minute)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if !claimed {
		t.Error("first claim should succeed")
	}

	// Second claim attempt should be skipped (not stale).
	claimed, err = brr.Claim(ctx, project, commitSHA, 30*time.Minute)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if claimed {
		t.Error("second claim should be skipped")
	}

	// Update to success.
	if err := brr.SetStatus(ctx, project, commitSHA, tidb.BuildStatusSuccess); err != nil {
		t.Fatalf("set status: %v", err)
	}

	status, err := brr.GetStatus(ctx, project, commitSHA)
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if status != tidb.BuildStatusSuccess {
		t.Errorf("status: got %q, want success", status)
	}
}
