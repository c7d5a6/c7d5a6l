package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
)

// Open creates the SQLite file (and parent dirs) if needed, then configures
// WAL, foreign keys, and a busy timeout.
func Open(path string) (*sql.DB, error) {
	debuglog.Printf("db.Open path=%s", path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("pragma foreign_keys: %w", err)
	}
	if _, err := sqlDB.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("pragma busy_timeout: %w", err)
	}
	if _, err := sqlDB.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("pragma journal_mode: %w", err)
	}

	debuglog.Printf("db.Open ok path=%s", path)
	return sqlDB, nil
}
