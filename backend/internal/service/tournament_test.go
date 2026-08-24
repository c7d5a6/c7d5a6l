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
	svc := service.NewTournament(sqlDB, tourRepo, playerRepo, nil, fetcher, nil)

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

	saved, sync, queued, err := svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if queued != 2 {
		t.Fatalf("importQueued=%d, want 2", queued)
	}
	if !sync.Exists || !sync.Same || sync.Action != model.TournamentActionNone {
		t.Fatalf("after save want none, got %+v", sync)
	}
	pending := 0
	for _, p := range sync.Players {
		if p.WillImport {
			t.Fatalf("unexpected willImport after stub save for %v", p.Link)
		}
		if p.ImportPending {
			pending++
		}
	}
	if pending != 2 {
		t.Fatalf("importPending=%d, want 2", pending)
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

	_, sync, _, err = svc.Save(ctx, changed)
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
	svc := service.NewTournament(sqlDB, tourRepo, playerRepo, nil, stubPlayerFetcher{
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
	_, _, _, err = svc.Save(ctx, page)
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

func TestTournamentSaveGroupsRoundTrip(t *testing.T) {
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
	orphanLink := "https://liquipedia.net/starcraft/NotOnRoster"
	svc := service.NewTournament(sqlDB, tourRepo, playerRepo, nil, stubPlayerFetcher{
		jaedongLink: {Name: str("Jaedong"), PreferredRace: str("zerg"), IDs: []string{}},
		flashLink:   {Name: str("Flash"), PreferredRace: str("terran"), IDs: []string{}},
	}, nil)

	page := model.TournamentPage{
		Link: "https://liquipedia.net/starcraft/ASL/groups-test",
		Name: str("Groups Test"),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
			{Name: str("Flash"), Link: str(flashLink), Race: str("terran")},
		},
		Groups: []model.TournamentGroup{
			{
				Name:      "Group A",
				Phase:     "Round of 24",
				SortOrder: 0,
				Players: []model.Participant{
					{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
					{Name: str("Orphan"), Link: str(orphanLink), Race: str("zerg")},
				},
			},
			{
				Name:      "Group B",
				Phase:     "Round of 24",
				SortOrder: 1,
				Players: []model.Participant{
					{Name: str("Flash"), Link: str(flashLink), Race: str("terran")},
				},
			},
		},
	}

	saved, _, _, err := svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Groups) != 2 {
		t.Fatalf("groups=%d, want 2", len(saved.Groups))
	}
	if saved.Groups[0].Phase != "Round of 24" || saved.Groups[0].Name != "Group A" {
		t.Fatalf("group0=%+v", saved.Groups[0])
	}
	if len(saved.Groups[0].Players) != 1 {
		t.Fatalf("orphan should be skipped, players=%d", len(saved.Groups[0].Players))
	}
	if saved.Groups[0].Players[0].Link == nil || *saved.Groups[0].Players[0].Link != jaedongLink {
		t.Fatalf("group A player=%v", saved.Groups[0].Players[0].Link)
	}
	if saved.Groups[1].Name != "Group B" || len(saved.Groups[1].Players) != 1 {
		t.Fatalf("group1=%+v", saved.Groups[1])
	}

	page.Groups = []model.TournamentGroup{
		{
			Name:      "Group C",
			Phase:     "Round of 16",
			SortOrder: 0,
			Players: []model.Participant{
				{Name: str("Flash"), Link: str(flashLink), Race: str("terran")},
				{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
			},
		},
	}
	saved, _, _, err = svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Groups) != 1 || saved.Groups[0].Name != "Group C" {
		t.Fatalf("after replace groups=%+v", saved.Groups)
	}
	if len(saved.Groups[0].Players) != 2 {
		t.Fatalf("group C players=%d", len(saved.Groups[0].Players))
	}
	if saved.Groups[0].Players[0].Link == nil || *saved.Groups[0].Players[0].Link != flashLink {
		t.Fatalf("sort order: first=%v", saved.Groups[0].Players[0].Link)
	}

	stored, err := tourRepo.GetByLink(ctx, sqlDB, page.Link)
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || len(stored.Page.Groups) != 1 {
		t.Fatalf("GetByLink groups=%v", stored)
	}
}

func TestTournamentSaveResultsUpsert(t *testing.T) {
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
	svc := service.NewTournament(sqlDB, tourRepo, playerRepo, nil, stubPlayerFetcher{
		jaedongLink: {Name: str("Jaedong"), PreferredRace: str("zerg"), IDs: []string{}},
		flashLink:   {Name: str("Flash"), PreferredRace: str("terran"), IDs: []string{}},
	}, nil)

	page := model.TournamentPage{
		Link: "https://liquipedia.net/starcraft/ASL/results-test",
		Name: str("Results Test"),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
			{Name: str("Flash"), Link: str(flashLink), Race: str("terran")},
		},
		Groups: []model.TournamentGroup{
			{Name: "Group A", Phase: "Round of 24", SortOrder: 0, Players: []model.Participant{
				{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
				{Name: str("Flash"), Link: str(flashLink), Race: str("terran")},
			}},
		},
		Results: []model.Result{
			{
				Played: false, Phase: "Round of 24", Round: "Group A", Order: 1,
				DateTime:     str("2026-08-20T12:00:00Z"),
				ParticipantA: &model.Participant{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
				ParticipantB: &model.Participant{Name: str("Flash"), Link: str(flashLink), Race: str("terran")},
			},
		},
	}
	saved, _, _, err := svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Results) != 1 || saved.Results[0].Played {
		t.Fatalf("saved results=%+v", saved.Results)
	}
	if saved.Results[0].GroupID == nil {
		t.Fatal("expected groupId on result")
	}

	page.Results[0].Played = true
	page.Results[0].ScoreA = intPtr(2)
	page.Results[0].ScoreB = intPtr(1)
	saved, _, _, err = svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Results) != 1 {
		t.Fatalf("upsert should not duplicate, got %d", len(saved.Results))
	}
	if !saved.Results[0].Played || saved.Results[0].ScoreA == nil || *saved.Results[0].ScoreA != 2 {
		t.Fatalf("updated result=%+v", saved.Results[0])
	}

	// Same pair meets again (e.g. double round-robin) — both rows kept.
	page.Results = append(page.Results, model.Result{
		Played: true, Phase: "Round of 24", Round: "Group A", Order: 2,
		DateTime:     str("2026-08-21T12:00:00Z"),
		ScoreA:       intPtr(1),
		ScoreB:       intPtr(2),
		ParticipantA: &model.Participant{Name: str("Flash"), Link: str(flashLink), Race: str("terran")},
		ParticipantB: &model.Participant{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
	})
	saved, _, _, err = svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Results) != 2 {
		t.Fatalf("rematch should keep both meetings, got %d: %+v", len(saved.Results), saved.Results)
	}

	var n int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM tournament_result`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("db rows=%d want 2", n)
	}
}

func TestTournamentSaveTBDResultsReplace(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	jaedongLink := "https://liquipedia.net/starcraft/Jaedong"
	flashLink := "https://liquipedia.net/starcraft/Flash"
	svc := service.NewTournament(sqlDB, repository.NewTournament(sqlDB), repository.NewPlayer(sqlDB), nil, stubPlayerFetcher{
		jaedongLink: {Name: str("Jaedong"), PreferredRace: str("zerg"), IDs: []string{}},
		flashLink:   {Name: str("Flash"), PreferredRace: str("terran"), IDs: []string{}},
	}, nil)

	page := model.TournamentPage{
		Link: "https://liquipedia.net/starcraft/ASL/tbd-test",
		Name: str("TBD Test"),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
			{Name: str("Flash"), Link: str(flashLink), Race: str("terran")},
		},
		Results: []model.Result{{
			Played: false, Phase: "Playoffs", Round: "Semifinals", Order: 1,
			DateTime:     str("2026-08-20T12:00:00Z"),
			ParticipantA: &model.Participant{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
			ParticipantB: &model.Participant{Name: str("TBD")},
		}},
	}
	saved, _, _, err := svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Results) != 1 {
		t.Fatalf("saved TBD results=%+v", saved.Results)
	}
	b := saved.Results[0].ParticipantB
	if b == nil || b.Name == nil || *b.Name != "TBD" || b.Link != nil {
		t.Fatalf("want TBD side, got %+v", b)
	}

	page.Results[0].ParticipantB = &model.Participant{Name: str("Flash"), Link: str(flashLink), Race: str("terran")}
	page.Results[0].Played = true
	page.Results[0].ScoreA = intPtr(2)
	page.Results[0].ScoreB = intPtr(1)
	saved, _, _, err = svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Results) != 1 {
		t.Fatalf("TBD should be replaced by real match, got %d: %+v", len(saved.Results), saved.Results)
	}
	if !saved.Results[0].Played || saved.Results[0].ParticipantB == nil || saved.Results[0].ParticipantB.Link == nil {
		t.Fatalf("want resolved match, got %+v", saved.Results[0])
	}

	page.Results = append(page.Results, model.Result{
		Played: false, Phase: "Playoffs", Round: "Grand Final", Order: 2,
		ParticipantA: &model.Participant{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
		ParticipantB: &model.Participant{Name: str("TBD")},
	})
	saved, _, _, err = svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Results) != 2 {
		t.Fatalf("want real + TBD, got %d: %+v", len(saved.Results), saved.Results)
	}

	page.Results = page.Results[:1]
	saved, _, _, err = svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Results) != 1 {
		t.Fatalf("vanished TBD should be deleted, got %d: %+v", len(saved.Results), saved.Results)
	}
}

func TestTournamentSaveGroupWinners(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	jaedongLink := "https://liquipedia.net/starcraft/Jaedong"
	flashLink := "https://liquipedia.net/starcraft/Flash"
	svc := service.NewTournament(sqlDB, repository.NewTournament(sqlDB), repository.NewPlayer(sqlDB), nil, stubPlayerFetcher{
		jaedongLink: {Name: str("Jaedong"), PreferredRace: str("zerg"), IDs: []string{}},
		flashLink:   {Name: str("Flash"), PreferredRace: str("terran"), IDs: []string{}},
	}, nil)

	page := model.TournamentPage{
		Link: "https://liquipedia.net/starcraft/ASL/winners-test",
		Name: str("Winners"),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
			{Name: str("Flash"), Link: str(flashLink), Race: str("terran")},
		},
		Groups: []model.TournamentGroup{{
			Name: "Group A", Phase: "Round of 24",
			Players: []model.Participant{
				{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg"), IsWinner: true},
				{Name: str("Flash"), Link: str(flashLink), Race: str("terran")},
			},
		}},
	}
	saved, _, _, err := svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Groups) != 1 || len(saved.Groups[0].Players) != 2 {
		t.Fatalf("groups=%+v", saved.Groups)
	}
	if !saved.Groups[0].Players[0].IsWinner || saved.Groups[0].Players[1].IsWinner {
		t.Fatalf("winners=%+v", saved.Groups[0].Players)
	}

	noWinners := page
	noWinners.Groups = []model.TournamentGroup{{
		Name: "Group A", Phase: "Round of 24",
		Players: []model.Participant{
			{Name: str("Jaedong"), Link: str(jaedongLink), Race: str("zerg")},
			{Name: str("Flash"), Link: str(flashLink), Race: str("terran")},
		},
	}}
	if _, _, _, err := svc.Save(ctx, noWinners); err != nil {
		t.Fatal(err)
	}
	sync, err := svc.SyncStatus(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if sync.Same || sync.Action != model.TournamentActionUpdate {
		t.Fatalf("winner-only parse should update, got same=%v action=%s changes=%v", sync.Same, sync.Action, sync.Changes)
	}
}

func intPtr(n int) *int       { return &n }
func boolPtr(b bool) *bool    { return &b }
