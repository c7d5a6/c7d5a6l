package service_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/db"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

type stubPlayerFetcher map[string]model.PlayerPage

func (s stubPlayerFetcher) FetchPlayerPage(_ context.Context, link string) (model.PlayerPage, error) {
	p, ok := s[link]
	if !ok {
		return model.PlayerPage{}, fmt.Errorf("stub missing %s", link)
	}
	p.Link = link
	return p, nil
}

func TestTournamentSyncAndSave(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	playerRepo := repository.NewPlayer(sqlDB)
	tourRepo := repository.NewTournament(sqlDB)

	jaedongLink := "https://liquipedia.net/starcraft/Jaedong"
	flashLink := "https://liquipedia.net/starcraft/Flash"
	fetcher := stubPlayerFetcher{
		jaedongLink: {
			Name:          str("Jaedong"),
			RealName:      str("Lee Jae Dong"),
			IDs:           []string{"JD"},
			PreferredRace: str("zerg"),
		},
		flashLink: {
			Name:          str("Flash"),
			IDs:           []string{},
			PreferredRace: str("terran"),
		},
	}
	svc := service.NewTournament(sqlDB, tourRepo, playerRepo, fetcher, nil)

	page := model.TournamentPage{
		Link:           "https://liquipedia.net/starcraft/ASL/20",
		Name:           str("ASL Season 20"),
		StartDate:      str("2025-01-01"),
		EndDate:        str("2025-02-01"),
		LiquipediaTier: str("Premier"),
		PlayerCounts:   &model.PlayerCounts{Total: intPtr(2)},
		Finished:       boolPtr(true),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
			{Name: str("Flash"), Link: str(flashLink), Race: str("terran")},
		},
		Results: []model.Result{},
	}

	sync, err := svc.SyncStatus(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if sync.Exists || sync.Action != model.TournamentActionAdd {
		t.Fatalf("want add, got %+v", sync)
	}
	importCount := 0
	for _, p := range sync.Players {
		if p.WillImport {
			importCount++
		}
	}
	if importCount != 2 {
		t.Fatalf("willImport count=%d, want 2", importCount)
	}

	saved, sync, err := svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if !sync.Exists || !sync.Same || sync.Action != model.TournamentActionNone {
		t.Fatalf("after save want none, got %+v", sync)
	}
	if saved.Name == nil || *saved.Name != "ASL Season 20" {
		t.Fatalf("saved name=%v", saved.Name)
	}
	if len(saved.Participants) != 2 {
		t.Fatalf("participants=%d", len(saved.Participants))
	}

	// Existing players: change name → update; willImport should be false.
	changed := page
	changed.Name = str("ASL S20")
	sync, err = svc.SyncStatus(ctx, changed)
	if err != nil {
		t.Fatal(err)
	}
	if !sync.Exists || sync.Same || sync.Action != model.TournamentActionUpdate {
		t.Fatalf("want update, got %+v", sync)
	}
	for _, p := range sync.Players {
		if p.WillImport {
			t.Fatalf("unexpected willImport for %v", p.Link)
		}
	}

	_, sync, err = svc.Save(ctx, changed)
	if err != nil {
		t.Fatal(err)
	}
	if !sync.Same {
		t.Fatalf("after update want same, got %+v", sync)
	}
}

func TestTournamentSaveRollsBackOnRosterError(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	playerRepo := repository.NewPlayer(sqlDB)
	tourRepo := repository.NewTournament(sqlDB)
	link := "https://liquipedia.net/starcraft/Jaedong"
	svc := service.NewTournament(sqlDB, tourRepo, playerRepo, stubPlayerFetcher{
		link: {Name: str("Jaedong"), PreferredRace: str("zerg"), IDs: []string{}},
	}, nil)

	// Participant missing race → buildRosterEntries fails inside TX after player upsert.
	page := model.TournamentPage{
		Link: "https://liquipedia.net/starcraft/ASL/99",
		Name: str("Bad Roster"),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str(link)}, // no race
		},
	}
	_, _, err = svc.Save(ctx, page)
	if err == nil {
		t.Fatal("expected error")
	}

	stored, err := tourRepo.GetByLink(ctx, sqlDB, page.Link)
	if err != nil {
		t.Fatal(err)
	}
	if stored != nil {
		t.Fatal("tournament should not exist after rollback")
	}
	exists, err := playerRepo.ExistsByLink(ctx, sqlDB, link)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("imported player should roll back with tournament")
	}
}

func intPtr(n int) *int       { return &n }
func boolPtr(b bool) *bool    { return &b }
