package db_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/db"
)

func TestMigrate_idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sqlite")

	sqlDB, err := db.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatalf("migrate again: %v", err)
	}

	var version int
	if err := sqlDB.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("version: %v", err)
	}
	if version != 13 {
		t.Fatalf("version=%d, want 13", version)
	}

	tables := []string{
		"tournament",
		"player",
		"player_alias",
		"player_race",
		"tournament_player",
		"tournament_group",
		"tournament_group_player",
		"user",
		"fantasy_league",
		"fantasy_player",
		"fantasy_team",
		"fantasy_team_member",
		"user_title",
	}
	for _, name := range tables {
		var n int
		err := sqlDB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name,
		).Scan(&n)
		if err != nil {
			t.Fatalf("check table %s: %v", name, err)
		}
		if n != 1 {
			t.Errorf("missing table %s", name)
		}
	}

	var aliasCol int
	err = sqlDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pragma_table_info('tournament_player') WHERE name='player_alias_id'`,
	).Scan(&aliasCol)
	if err != nil {
		t.Fatalf("pragma tournament_player: %v", err)
	}
	if aliasCol != 1 {
		t.Fatal("tournament_player.player_alias_id missing")
	}

	for _, col := range []string{"telegram_id", "first_name", "role", "last_login_at"} {
		var n int
		err = sqlDB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM pragma_table_info('user') WHERE name=?`, col,
		).Scan(&n)
		if err != nil {
			t.Fatalf("pragma user.%s: %v", col, err)
		}
		if n != 1 {
			t.Errorf("user.%s missing", col)
		}
	}
}
