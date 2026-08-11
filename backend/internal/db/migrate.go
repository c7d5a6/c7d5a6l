package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/migrations"
)

var migrationNameRE = regexp.MustCompile(`^(\d+)_.*\.sql$`)

// DefaultPath is the default SQLite file under backend/devdata (cwd-relative).
const DefaultPath = "devdata/app.sqlite"

// Migrate applies all embedded SQL migrations that are not yet recorded in
// schema_migrations. Refuses to start if the DB has a version newer than the app.
func Migrate(ctx context.Context, sqlDB *sql.DB) error {
	debuglog.Printf("db.Migrate start")
	if _, err := sqlDB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	files, err := listMigrations()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no migration files found")
	}

	appMax := files[len(files)-1].version
	dbMax, err := maxAppliedVersion(ctx, sqlDB)
	if err != nil {
		return err
	}
	debuglog.Printf("db.Migrate dbVersion=%d appVersion=%d files=%d", dbMax, appMax, len(files))
	if dbMax > appMax {
		return fmt.Errorf("database schema version %d is newer than app %d", dbMax, appMax)
	}

	applied, err := appliedVersions(ctx, sqlDB)
	if err != nil {
		return err
	}

	for _, m := range files {
		if applied[m.version] {
			debuglog.Printf("db.Migrate skip already-applied %s", m.name)
			continue
		}
		debuglog.Printf("db.Migrate applying %s", m.name)
		body, err := fs.ReadFile(migrations.FS, m.name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m.name, err)
		}
		tx, err := sqlDB.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", m.version, err)
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", m.name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, m.version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.version, err)
		}
		debuglog.Printf("db.Migrate applied %s", m.name)
	}
	debuglog.Printf("db.Migrate done")
	return nil
}

type migrationFile struct {
	version int
	name    string
}

func listMigrations() ([]migrationFile, error) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	var out []migrationFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		m := migrationNameRE.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		v, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("migration version %s: %w", name, err)
		}
		out = append(out, migrationFile{version: v, name: path.Clean(name)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	for i := 1; i < len(out); i++ {
		if out[i].version == out[i-1].version {
			return nil, fmt.Errorf("duplicate migration version %d", out[i].version)
		}
	}
	return out, nil
}

func appliedVersions(ctx context.Context, sqlDB *sql.DB) (map[int]bool, error) {
	rows, err := sqlDB.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("select schema_migrations: %w", err)
	}
	defer rows.Close()

	out := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func maxAppliedVersion(ctx context.Context, sqlDB *sql.DB) (int, error) {
	var v sql.NullInt64
	err := sqlDB.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("max schema version: %w", err)
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}

// ResolvePath returns path, or DefaultPath when path is empty.
func ResolvePath(path string) string {
	if path == "" {
		return filepath.Clean(DefaultPath)
	}
	return filepath.Clean(path)
}
