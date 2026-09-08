package job_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/db"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

func str(s string) *string { return &s }
func intPtr(n int) *int    { return &n }

type stubFetcher map[string]model.PlayerPage

func (s stubFetcher) FetchPlayerPage(_ context.Context, link string) (model.PlayerPage, error) {
	p := s[link]
	p.Link = link
	return p, nil
}

func TestListTournamentsDueRefresh(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	jaedong := "https://liquipedia.net/starcraft/Jaedong"
	flash := "https://liquipedia.net/starcraft/Flash"
	tourRepo := repository.NewTournament(sqlDB)
	playerRepo := repository.NewPlayer(sqlDB)
	svc := service.NewTournament(sqlDB, tourRepo, playerRepo, nil, stubFetcher{
		jaedong: {Name: str("Jaedong"), PreferredRace: str("zerg"), IDs: []string{}},
		flash:   {Name: str("Flash"), PreferredRace: str("terran"), IDs: []string{}},
	}, nil)

	now := time.Date(2026, 8, 20, 15, 0, 0, 0, time.UTC)
	past := now.Add(-2 * time.Hour).Format(time.RFC3339)
	page := model.TournamentPage{
		Link: "https://liquipedia.net/starcraft/ASL/due",
		Name: str("Due"),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str(jaedong), Race: str("zerg")},
			{Name: str("Flash"), Link: str(flash), Race: str("terran")},
		},
		Results: []model.Result{{
			Played: false, Phase: "Round of 24", Round: "Group A", Order: 1,
			DateTime:     &past,
			ParticipantA: &model.Participant{Name: str("Jaedong"), Link: str(jaedong), Race: str("zerg")},
			ParticipantB: &model.Participant{Name: str("Flash"), Link: str(flash), Race: str("terran")},
		}},
	}
	if _, _, _, err := svc.Save(ctx, page); err != nil {
		t.Fatal(err)
	}

	due, err := svc.ListDueRefresh(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].Link != page.Link {
		t.Fatalf("due=%+v", due)
	}

	// Played match should not qualify.
	page.Results[0].Played = true
	page.Results[0].ScoreA = intPtr(2)
	page.Results[0].ScoreB = intPtr(0)
	if _, _, _, err := svc.Save(ctx, page); err != nil {
		t.Fatal(err)
	}
	due, err = svc.ListDueRefresh(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Fatalf("want none after played, got %+v", due)
	}
}

func TestListUnfinishedAndInProgress(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	jaedong := "https://liquipedia.net/starcraft/Jaedong"
	flash := "https://liquipedia.net/starcraft/Flash"
	svc := service.NewTournament(sqlDB, repository.NewTournament(sqlDB), repository.NewPlayer(sqlDB), nil, stubFetcher{
		jaedong: {Name: str("Jaedong"), PreferredRace: str("zerg"), IDs: []string{}},
		flash:   {Name: str("Flash"), PreferredRace: str("terran"), IDs: []string{}},
	}, nil)

	now := time.Date(2026, 8, 20, 15, 0, 0, 0, time.UTC)
	yesterday := now.Add(-26 * time.Hour).Format(time.RFC3339)
	open := model.TournamentPage{
		Link:     "https://liquipedia.net/starcraft/ASL/open",
		Name:     str("Open"),
		Finished: boolPtr(false),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str(jaedong), Race: str("zerg")},
			{Name: str("Flash"), Link: str(flash), Race: str("terran")},
		},
		Results: []model.Result{{
			Played: false, Phase: "Round of 24", Round: "Group A", Order: 1,
			DateTime:     &yesterday,
			ParticipantA: &model.Participant{Name: str("Jaedong"), Link: str(jaedong), Race: str("zerg")},
			ParticipantB: &model.Participant{Name: str("Flash"), Link: str(flash), Race: str("terran")},
		}},
	}
	if _, _, _, err := svc.Save(ctx, open); err != nil {
		t.Fatal(err)
	}
	done := model.TournamentPage{
		Link:     "https://liquipedia.net/starcraft/ASL/done",
		Name:     str("Done"),
		Finished: boolPtr(true),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str(jaedong), Race: str("zerg")},
			{Name: str("Flash"), Link: str(flash), Race: str("terran")},
		},
	}
	if _, _, _, err := svc.Save(ctx, done); err != nil {
		t.Fatal(err)
	}

	unfinished, err := svc.ListUnfinished(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(unfinished) != 1 || unfinished[0].Link != open.Link {
		t.Fatalf("unfinished=%+v", unfinished)
	}

	due, err := svc.ListDueRefresh(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Fatalf("yesterday match should not be due-today, got %+v", due)
	}

	inProg, err := svc.ListInProgressRefresh(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(inProg) != 1 || inProg[0].Link != open.Link {
		t.Fatalf("in-progress=%+v", inProg)
	}
}

func TestFinishIfEndedMonthAgo(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	jaedong := "https://liquipedia.net/starcraft/Jaedong"
	flash := "https://liquipedia.net/starcraft/Flash"
	tourRepo := repository.NewTournament(sqlDB)
	svc := service.NewTournament(sqlDB, tourRepo, repository.NewPlayer(sqlDB), nil, stubFetcher{
		jaedong: {Name: str("Jaedong"), PreferredRace: str("zerg"), IDs: []string{}},
		flash:   {Name: str("Flash"), PreferredRace: str("terran"), IDs: []string{}},
	}, nil)

	now := time.Date(2026, 9, 8, 15, 0, 0, 0, time.UTC)
	past := now.Add(-2 * time.Hour).Format(time.RFC3339)
	participants := []model.Participant{
		{Name: str("Jaedong"), Link: str(jaedong), Race: str("zerg")},
		{Name: str("Flash"), Link: str(flash), Race: str("terran")},
	}
	save := func(link, end string) int64 {
		t.Helper()
		page := model.TournamentPage{
			Link:         link,
			Name:         str("Open"),
			Finished:     boolPtr(false),
			Participants: participants,
			Results: []model.Result{{
				Played: false, Phase: "Round of 24", Round: "Group A", Order: 1,
				DateTime:     &past,
				ParticipantA: &participants[0],
				ParticipantB: &participants[1],
			}},
		}
		if end != "" {
			page.EndDate = str(end)
		}
		if _, _, _, err := svc.Save(ctx, page); err != nil {
			t.Fatal(err)
		}
		stored, err := tourRepo.GetByLink(ctx, sqlDB, link)
		if err != nil || stored == nil {
			t.Fatalf("get %s: %v", link, err)
		}
		return stored.ID
	}

	staleID := save("https://liquipedia.net/starcraft/ASL/stale", "2026-08-08")
	freshID := save("https://liquipedia.net/starcraft/ASL/fresh", "2026-08-09")
	noEndID := save("https://liquipedia.net/starcraft/ASL/noend", "")

	skip, err := svc.FinishIfEndedMonthAgo(ctx, staleID, now)
	if err != nil {
		t.Fatal(err)
	}
	if !skip {
		t.Fatal("want skip after a month past end date")
	}
	stale, err := tourRepo.GetByID(ctx, sqlDB, staleID)
	if err != nil || stale == nil || stale.Page.Finished == nil || !*stale.Page.Finished {
		t.Fatalf("stale should be finished, got %+v", stale)
	}

	skip, err = svc.FinishIfEndedMonthAgo(ctx, freshID, now)
	if err != nil {
		t.Fatal(err)
	}
	if skip {
		t.Fatal("want liquipedia sync when end date is less than a month ago")
	}
	fresh, err := tourRepo.GetByID(ctx, sqlDB, freshID)
	if err != nil || fresh == nil || (fresh.Page.Finished != nil && *fresh.Page.Finished) {
		t.Fatalf("fresh should stay unfinished, got %+v", fresh)
	}

	skip, err = svc.FinishIfEndedMonthAgo(ctx, noEndID, now)
	if err != nil {
		t.Fatal(err)
	}
	if skip {
		t.Fatal("want liquipedia sync when end date is missing")
	}

	unfinished, err := svc.ListUnfinished(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(unfinished) != 2 {
		t.Fatalf("unfinished=%+v", unfinished)
	}

	inProg, err := svc.ListInProgressRefresh(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(inProg) != 2 {
		t.Fatalf("in-progress after stale finish=%+v", inProg)
	}

	skip, err = svc.FinishIfEndedMonthAgo(ctx, staleID, now)
	if err != nil || !skip {
		t.Fatalf("already finished stale want skip, skip=%v err=%v", skip, err)
	}
}

func boolPtr(b bool) *bool { return &b }
