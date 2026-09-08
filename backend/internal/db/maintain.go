package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
)

// Maintain runs lightweight SQLite housekeeping after migrations.
// Always checkpoints and truncates the WAL file. A full VACUUM (rewrites the
// whole database; needs free disk space and blocks writes) runs only when
// C7D5A6L_DB_VACUUM is 1/true/yes/on.
func Maintain(ctx context.Context, sqlDB *sql.DB) error {
	if _, err := sqlDB.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return fmt.Errorf("wal checkpoint: %w", err)
	}
	debuglog.Printf("db.Maintain wal_checkpoint ok")

	if !envTruthy(os.Getenv("C7D5A6L_DB_VACUUM")) {
		return nil
	}

	before := dbPageCount(ctx, sqlDB)
	if _, err := sqlDB.ExecContext(ctx, `VACUUM`); err != nil {
		return fmt.Errorf("vacuum: %w", err)
	}
	after := dbPageCount(ctx, sqlDB)
	debuglog.Printf("db.Maintain vacuum pages %d -> %d", before, after)
	return nil
}

func dbPageCount(ctx context.Context, sqlDB *sql.DB) int {
	var n int
	_ = sqlDB.QueryRowContext(ctx, `PRAGMA page_count`).Scan(&n)
	return n
}

func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
