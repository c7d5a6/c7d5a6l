package service_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/db"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

func TestEnsurePreSeasonBootstrapsFromSeason1(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	seasonRepo := repository.NewSeason(sqlDB)
	playerRepo := repository.NewPlayer(sqlDB)
	seasonSvc := service.NewSeason(sqlDB, seasonRepo, playerRepo)

	if _, err := sqlDB.ExecContext(ctx, `UPDATE season SET started_at = '2026-03-15T12:00:00Z' WHERE status = 'active'`); err != nil {
		t.Fatal(err)
	}

	res, err := sqlDB.ExecContext(ctx, `
		INSERT INTO player (link, name, preferred_race) VALUES ('https://liquipedia.net/starcraft/A', 'A', 'terran')
	`)
	if err != nil {
		t.Fatal(err)
	}
	playerID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	res, err = sqlDB.ExecContext(ctx, `
		INSERT INTO player_race (player_id, race, elo) VALUES (?, 'terran', 1800)
	`, playerID)
	if err != nil {
		t.Fatal(err)
	}
	raceID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO season_player_race (season_id, player_race_id, start_elo, start_rank)
		VALUES (1, ?, 1800, 1)
	`, raceID); err != nil {
		t.Fatal(err)
	}

	if err := seasonSvc.EnsurePreSeason(ctx); err != nil {
		t.Fatal(err)
	}
	if err := seasonSvc.EnsurePreSeason(ctx); err != nil {
		t.Fatal("second ensure should be idempotent")
	}

	var count int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM season WHERE name = 'Pre-season'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("pre-season count=%d", count)
	}

	var startedAt, closedAt string
	if err := sqlDB.QueryRowContext(ctx, `
		SELECT started_at, closed_at FROM season WHERE name = 'Pre-season'
	`).Scan(&startedAt, &closedAt); err != nil {
		t.Fatal(err)
	}
	if startedAt[:10] != "2026-03-14" {
		t.Fatalf("pre-season start=%s", startedAt)
	}
	if closedAt[:10] != "2026-03-15" {
		t.Fatalf("pre-season end=%s", closedAt)
	}

	var snapCount int
	if err := sqlDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM season_player_race spr
		JOIN season s ON s.id = spr.season_id
		WHERE s.name = 'Pre-season'
			AND spr.start_elo = spr.end_elo
			AND spr.start_rank = spr.end_rank
	`).Scan(&snapCount); err != nil {
		t.Fatal(err)
	}
	if snapCount == 0 {
		t.Fatal("expected pre-season snapshots")
	}

	players, season, err := seasonSvc.ListRaceEntriesWithSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if season == nil || season.Name != "Season 1" {
		t.Fatalf("active season=%#v", season)
	}
	foundDelta := false
	for _, row := range players {
		if row.RankDelta != nil {
			foundDelta = true
			break
		}
	}
	if !foundDelta {
		t.Fatal("expected rank deltas against pre-season baseline")
	}
}

func TestEnsurePreSeasonBootstrapsEmptySeasonTable(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `DELETE FROM season_player_race`); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `DELETE FROM season`); err != nil {
		t.Fatal(err)
	}

	res, err := sqlDB.ExecContext(ctx, `
		INSERT INTO player (link, name, preferred_race) VALUES ('https://liquipedia.net/starcraft/B', 'B', 'zerg')
	`)
	if err != nil {
		t.Fatal(err)
	}
	playerID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	res, err = sqlDB.ExecContext(ctx, `
		INSERT INTO player_race (player_id, race, elo) VALUES (?, 'zerg', 1750)
	`, playerID)
	if err != nil {
		t.Fatal(err)
	}

	seasonRepo := repository.NewSeason(sqlDB)
	seasonSvc := service.NewSeason(sqlDB, seasonRepo, repository.NewPlayer(sqlDB))
	if err := seasonSvc.EnsurePreSeason(ctx); err != nil {
		t.Fatal(err)
	}

	var seasonCount int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM season`).Scan(&seasonCount); err != nil {
		t.Fatal(err)
	}
	if seasonCount != 2 {
		t.Fatalf("season count=%d want 2 (Pre-season + Season 1)", seasonCount)
	}

	active, err := seasonSvc.GetCurrent(ctx)
	if err != nil || active == nil || active.Name != "Season 1" {
		t.Fatalf("active season=%#v err=%v", active, err)
	}
}

func TestEnsurePreSeasonBootstrapsSeason1WithoutPlayers(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `DELETE FROM season_player_race`); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `DELETE FROM season`); err != nil {
		t.Fatal(err)
	}

	seasonRepo := repository.NewSeason(sqlDB)
	seasonSvc := service.NewSeason(sqlDB, seasonRepo, repository.NewPlayer(sqlDB))
	if err := seasonSvc.EnsurePreSeason(ctx); err != nil {
		t.Fatal(err)
	}

	var seasonCount int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM season`).Scan(&seasonCount); err != nil {
		t.Fatal(err)
	}
	if seasonCount != 1 {
		t.Fatalf("season count=%d want 1 (Season 1 only)", seasonCount)
	}

	var preCount int
	if err := sqlDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM season WHERE name = 'Pre-season'
	`).Scan(&preCount); err != nil {
		t.Fatal(err)
	}
	if preCount != 0 {
		t.Fatalf("pre-season count=%d", preCount)
	}
}
