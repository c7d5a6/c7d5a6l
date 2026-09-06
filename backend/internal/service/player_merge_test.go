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

func TestPlayerMergeCandidatesAndMerge(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	repo := repository.NewPlayer(sqlDB)
	svc := service.NewPlayer(sqlDB, repo, nil)

	mainLink := "https://liquipedia.net/starcraft/MainPlayer"
	aliasLink := "https://liquipedia.net/starcraft/MainPlayer_Alt"

	mainSaved, _, err := svc.Save(ctx, model.PlayerPage{
		Link:          mainLink,
		Name:          str("MainPlayer"),
		IDs:           []string{"MP"},
		PreferredRace: str("terran"),
	})
	if err != nil {
		t.Fatal(err)
	}
	aliasSaved, _, err := svc.Save(ctx, model.PlayerPage{
		Link:          aliasLink,
		Name:          str("MainPlayerAlt"),
		IDs:           []string{"MP_Alt"},
		PreferredRace: str("terran"),
	})
	if err != nil {
		t.Fatal(err)
	}

	mainID, err := repo.IDByLink(ctx, sqlDB, mainSaved.Link)
	if err != nil || mainID == 0 {
		t.Fatalf("main id: %v", err)
	}
	aliasID, err := repo.IDByLink(ctx, sqlDB, aliasSaved.Link)
	if err != nil || aliasID == 0 {
		t.Fatalf("alias id: %v", err)
	}

	candidates, err := svc.MergeCandidates(ctx, mainID, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) == 0 {
		t.Fatal("expected merge candidates")
	}
	found := false
	for _, c := range candidates {
		if c.PlayerID == aliasID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("alias candidate missing: %+v", candidates)
	}

	mainRaceID, err := repo.EnsureRaceID(ctx, sqlDB, mainID, "terran")
	if err != nil {
		t.Fatal(err)
	}
	aliasRaceID, err := repo.EnsureRaceID(ctx, sqlDB, aliasID, "terran")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `
		UPDATE player_race SET elo = 1900 WHERE id = ?
	`, mainRaceID); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `
		UPDATE player_race SET elo = 1600 WHERE id = ?
	`, aliasRaceID); err != nil {
		t.Fatal(err)
	}

	tourRepo := repository.NewTournament(sqlDB)
	tourID, err := tourRepo.Upsert(ctx, sqlDB, model.TournamentPage{
		Link: "https://liquipedia.net/starcraft/Test_Tournament",
	})
	if err != nil {
		t.Fatal(err)
	}
	mainAliasID, err := repo.EnsureAliasID(ctx, sqlDB, mainID, "MainPlayer")
	if err != nil {
		t.Fatal(err)
	}
	aliasAliasID, err := repo.EnsureAliasID(ctx, sqlDB, aliasID, "MainPlayerAlt")
	if err != nil {
		t.Fatal(err)
	}
	if err := tourRepo.ReplaceRoster(ctx, sqlDB, tourID, []repository.RosterEntry{
		{PlayerRaceID: mainRaceID, PlayerAliasID: mainAliasID, Excluded: false},
		{PlayerRaceID: aliasRaceID, PlayerAliasID: aliasAliasID, Excluded: false},
	}); err != nil {
		t.Fatal(err)
	}

	if err := svc.MergePlayers(ctx, mainID, aliasID); err != nil {
		t.Fatal(err)
	}

	if id, err := repo.IDByLink(ctx, sqlDB, aliasLink); err != nil || id != 0 {
		t.Fatalf("alias player should be deleted, id=%d err=%v", id, err)
	}
	mainPage, err := repo.GetByLink(ctx, sqlDB, mainLink)
	if err != nil || mainPage == nil {
		t.Fatalf("main player missing: %v", err)
	}
	names := map[string]bool{}
	for _, id := range mainPage.IDs {
		names[id] = true
	}
	if !names["MainPlayerAlt"] || !names["MP_Alt"] {
		t.Fatalf("expected absorbed aliases, got %v", mainPage.IDs)
	}

	entry, err := repo.GetRaceEntryByID(ctx, sqlDB, mainRaceID)
	if err != nil || entry == nil {
		t.Fatal(err)
	}
	if entry.Elo != 1900 {
		t.Fatalf("main elo=%v want 1900", entry.Elo)
	}
	aliasEntry, err := repo.GetRaceEntryByID(ctx, sqlDB, aliasRaceID)
	if err != nil {
		t.Fatal(err)
	}
	if aliasEntry != nil {
		t.Fatalf("alias race row should be deleted, got %+v", aliasEntry)
	}

	var rosterCount int
	if err := sqlDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tournament_player tp
		JOIN player_race pr ON pr.id = tp.player_race_id
		WHERE tp.tournament_id = ? AND pr.player_id = ?
	`, tourID, mainID).Scan(&rosterCount); err != nil {
		t.Fatal(err)
	}
	if rosterCount != 1 {
		t.Fatalf("roster rows after merge=%d want 1", rosterCount)
	}
}
