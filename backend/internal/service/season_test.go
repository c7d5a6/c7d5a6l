package service_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/db"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

func setupSeasonFixture(t *testing.T) (context.Context, *service.Season, *service.Fantasy, *repository.Season) {
	t.Helper()
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	playerRepo := repository.NewPlayer(sqlDB)
	tourRepo := repository.NewTournament(sqlDB)
	seasonRepo := repository.NewSeason(sqlDB)
	seasonSvc := service.NewSeason(sqlDB, seasonRepo, playerRepo)
	fantasyRepo := repository.NewFantasy(sqlDB)
	fantasySvc := service.NewFantasy(sqlDB, fantasyRepo, tourRepo, seasonSvc)

	tourSvc := service.NewTournament(sqlDB, tourRepo, playerRepo, nil, stubPlayerFetcher{
		"https://liquipedia.net/starcraft/A": {
			Link: "https://liquipedia.net/starcraft/A", Name: str("A"), PreferredRace: str("terran"),
		},
		"https://liquipedia.net/starcraft/B": {
			Link: "https://liquipedia.net/starcraft/B", Name: str("B"), PreferredRace: str("zerg"),
		},
	}, nil)

	start := str("2026-01-01")
	end := str("2026-01-31")
	finished := true
	page := model.TournamentPage{
		Link:      "https://liquipedia.net/starcraft/TestCup",
		Name:      str("Test Cup"),
		StartDate: start,
		EndDate:   end,
		Finished:  &finished,
		Participants: []model.Participant{
			{Name: str("A"), Link: str("https://liquipedia.net/starcraft/A"), Race: str("terran")},
			{Name: str("B"), Link: str("https://liquipedia.net/starcraft/B"), Race: str("zerg")},
		},
		Results: []model.Result{
			{
				Played:       true,
				ScoreA:       intPtr(2),
				ScoreB:       intPtr(0),
				ParticipantA: &model.Participant{Name: str("A"), Link: str("https://liquipedia.net/starcraft/A"), Race: str("terran")},
				ParticipantB: &model.Participant{Name: str("B"), Link: str("https://liquipedia.net/starcraft/B"), Race: str("zerg")},
			},
		},
	}
	if _, _, _, err := tourSvc.Save(ctx, page); err != nil {
		t.Fatal(err)
	}

	if _, err := seasonRepo.DB().ExecContext(ctx, `UPDATE season SET started_at = '2026-01-01' WHERE status = 'active'`); err != nil {
		t.Fatal(err)
	}

	// B starts ranked above A so a win moves A up in the calculated standings.
	if _, err := seasonRepo.DB().ExecContext(ctx, `
		UPDATE season_player_race SET start_elo = 1800, start_rank = 1
		WHERE player_race_id = (
			SELECT pr.id FROM player_race pr
			JOIN player p ON p.id = pr.player_id
			WHERE p.link LIKE '%/B'
		)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := seasonRepo.DB().ExecContext(ctx, `
		UPDATE season_player_race SET start_elo = 1700, start_rank = 2
		WHERE player_race_id = (
			SELECT pr.id FROM player_race pr
			JOIN player p ON p.id = pr.player_id
			WHERE p.link LIKE '%/A'
		)
	`); err != nil {
		t.Fatal(err)
	}

	return ctx, seasonSvc, fantasySvc, seasonRepo
}

func TestSeasonCloseOpensNextSeason(t *testing.T) {
	ctx, seasonSvc, fantasySvc, seasonRepo := setupSeasonFixture(t)

	active, err := seasonSvc.GetCurrent(ctx)
	if err != nil || active == nil {
		t.Fatalf("active season: %v", err)
	}

	var tourID int64
	if err := seasonRepo.DB().QueryRowContext(ctx, `SELECT id FROM tournament LIMIT 1`).Scan(&tourID); err != nil {
		t.Fatal(err)
	}

	league, err := fantasySvc.CreateOrSeed(ctx, tourID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fantasySvc.StartLeague(ctx, league.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := fantasySvc.FinishLeague(ctx, league.ID); err != nil {
		t.Fatal(err)
	}

	active, err = seasonSvc.GetCurrent(ctx)
	if err != nil || !active.ReadyToClose {
		t.Fatalf("ready to close: %#v err=%v", active, err)
	}

	next, updated, err := seasonSvc.CloseSeason(ctx, []int64{tourID})
	if err != nil {
		t.Fatal(err)
	}
	if updated < 2 {
		t.Fatalf("playersUpdated=%d", updated)
	}
	if next == nil || next.Status != "active" || next.Name != "Season 2" {
		t.Fatalf("next season: %#v", next)
	}

	var closedCount int
	if err := seasonRepo.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM season WHERE status='closed'`).Scan(&closedCount); err != nil {
		t.Fatal(err)
	}
	if closedCount != 1 {
		t.Fatalf("closedCount=%d", closedCount)
	}
}

func TestFinishLeagueMarksSeasonReady(t *testing.T) {
	ctx, seasonSvc, fantasySvc, seasonRepo := setupSeasonFixture(t)

	var tourID int64
	if err := seasonRepo.DB().QueryRowContext(ctx, `SELECT id FROM tournament LIMIT 1`).Scan(&tourID); err != nil {
		t.Fatal(err)
	}
	league, err := fantasySvc.CreateOrSeed(ctx, tourID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fantasySvc.StartLeague(ctx, league.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := fantasySvc.FinishLeague(ctx, league.ID); err != nil {
		t.Fatal(err)
	}

	active, err := seasonSvc.GetCurrent(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !active.ReadyToClose || active.ClosingFantasyLeagueID == nil || *active.ClosingFantasyLeagueID != league.ID {
		t.Fatalf("season not ready: %#v", active)
	}
}

func TestListRaceEntriesWithSeasonRankDelta(t *testing.T) {
	ctx, seasonSvc, _, seasonRepo := setupSeasonFixture(t)

	entries, season, err := seasonSvc.ListRaceEntriesWithSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if season == nil || season.Name != "Season 1" {
		t.Fatalf("season=%#v", season)
	}
	if len(entries) < 2 {
		t.Fatalf("entries=%d", len(entries))
	}
	// A won 2-0 — should rank above B with positive delta vs season start.
	top := entries[0]
	bottom := entries[len(entries)-1]
	if top.RankDelta == nil || *top.RankDelta <= 0 {
		t.Fatalf("winner should have positive rank delta, got %#v", top.RankDelta)
	}
	if bottom.RankDelta == nil || *bottom.RankDelta >= 0 {
		t.Fatalf("loser should have negative rank delta, got %#v", bottom.RankDelta)
	}
	if top.ProjectedElo == nil || bottom.ProjectedElo == nil || *top.ProjectedElo <= *bottom.ProjectedElo {
		t.Fatalf("expected winner above loser by calculated elo: %v vs %v", top.ProjectedElo, bottom.ProjectedElo)
	}
	for _, e := range entries {
		if e.ProjectedElo != nil && e.Elo == *e.ProjectedElo && e.SeasonStartElo != nil && e.Elo != *e.SeasonStartElo {
			// Stored elo should remain the season-open baseline, not the live projection.
			t.Fatalf("stored elo should not be overwritten by projection: %#v", e)
		}
	}

	var tourID int64
	if err := seasonRepo.DB().QueryRowContext(ctx, `SELECT id FROM tournament LIMIT 1`).Scan(&tourID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := seasonSvc.CloseSeason(ctx, []int64{tourID}); err != nil {
		t.Fatal(err)
	}

	entries2, season2, err := seasonSvc.ListRaceEntriesWithSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if season2 == nil || season2.Name != "Season 2" {
		t.Fatalf("season2=%#v", season2)
	}
	if len(entries2) < 2 {
		t.Fatalf("entries=%d", len(entries2))
	}
	for _, e := range entries2 {
		if e.SeasonStartElo == nil {
			t.Fatalf("missing seasonStartElo for %d", e.PlayerRaceID)
		}
		if e.RankDelta == nil || *e.RankDelta != 0 {
			t.Fatalf("expected zero rank delta at season open, got %v for %d", e.RankDelta, e.PlayerRaceID)
		}
	}
}
