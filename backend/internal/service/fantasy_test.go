package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/db"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

func setupFantasyFixture(t *testing.T) (context.Context, *service.Fantasy, *repository.Fantasy, int64) {
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
	tourSvc := service.NewTournament(sqlDB, tourRepo, playerRepo, stubPlayerFetcher{
		"https://liquipedia.net/starcraft/Jaedong": {
			Link: "https://liquipedia.net/starcraft/Jaedong", Name: str("Jaedong"), IDs: []string{}, PreferredRace: str("zerg"),
		},
		"https://liquipedia.net/starcraft/Flash": {
			Link: "https://liquipedia.net/starcraft/Flash", Name: str("Flash"), IDs: []string{}, PreferredRace: str("terran"),
		},
		"https://liquipedia.net/starcraft/Skip": {
			Link: "https://liquipedia.net/starcraft/Skip", Name: str("Skip"), IDs: []string{}, PreferredRace: str("protoss"),
		},
	}, nil)

	page := model.TournamentPage{
		Link: "https://liquipedia.net/starcraft/ASL/20",
		Name: str("ASL Season 20"),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str("https://liquipedia.net/starcraft/Jaedong"), Race: str("zerg")},
			{Name: str("Flash"), Link: str("https://liquipedia.net/starcraft/Flash"), Race: str("terran")},
			{Name: str("Skip"), Link: str("https://liquipedia.net/starcraft/Skip"), Race: str("protoss"), Excluded: true},
		},
		Results: []model.Result{},
	}
	if _, _, err := tourSvc.Save(ctx, page); err != nil {
		t.Fatal(err)
	}

	var tournamentID int64
	if err := sqlDB.QueryRowContext(ctx, `SELECT id FROM tournament WHERE link = ?`, page.Link).Scan(&tournamentID); err != nil {
		t.Fatal(err)
	}

	fantasyRepo := repository.NewFantasy(sqlDB)
	fantasySvc := service.NewFantasy(sqlDB, fantasyRepo)
	return ctx, fantasySvc, fantasyRepo, tournamentID
}

func TestFantasyCreateOrSeedIdempotent(t *testing.T) {
	ctx, svc, repo, tournamentID := setupFantasyFixture(t)

	league, err := svc.CreateOrSeed(ctx, tournamentID)
	if err != nil {
		t.Fatal(err)
	}
	if league.ID == 0 || league.TournamentID != tournamentID {
		t.Fatalf("league=%#v", league)
	}

	players, err := svc.ListPlayers(ctx, league.ID, repository.PlayerSortCost)
	if err != nil {
		t.Fatal(err)
	}
	// Excluded Skip must not be seeded.
	if len(players) != 2 {
		t.Fatalf("seeded players=%d want 2", len(players))
	}

	again, err := svc.CreateOrSeed(ctx, tournamentID)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != league.ID {
		t.Fatalf("idempotent league id=%d want %d", again.ID, league.ID)
	}
	players2, err := repo.ListPlayers(ctx, repo.DB(), league.ID, repository.PlayerSortCost)
	if err != nil {
		t.Fatal(err)
	}
	if len(players2) != 2 {
		t.Fatalf("after reseed players=%d want 2", len(players2))
	}
}

func TestFantasySurvivesTournamentResave(t *testing.T) {
	ctx, fantasySvc, fantasyRepo, tournamentID := setupFantasyFixture(t)
	sqlDB := fantasyRepo.DB()

	league, err := fantasySvc.CreateOrSeed(ctx, tournamentID)
	if err != nil {
		t.Fatal(err)
	}
	before, err := fantasySvc.ListPlayers(ctx, league.ID, repository.PlayerSortCost)
	if err != nil || len(before) != 2 {
		t.Fatalf("before=%d err=%v", len(before), err)
	}
	if _, err := sqlDB.ExecContext(ctx, `
		UPDATE fantasy_player SET cost = 42 WHERE id = ?
	`, before[0].ID); err != nil {
		t.Fatal(err)
	}

	playerRepo := repository.NewPlayer(sqlDB)
	tourRepo := repository.NewTournament(sqlDB)
	tourSvc := service.NewTournament(sqlDB, tourRepo, playerRepo, stubPlayerFetcher{
		"https://liquipedia.net/starcraft/Jaedong": {
			Link: "https://liquipedia.net/starcraft/Jaedong", Name: str("Jaedong"), IDs: []string{}, PreferredRace: str("zerg"),
		},
		"https://liquipedia.net/starcraft/Flash": {
			Link: "https://liquipedia.net/starcraft/Flash", Name: str("Flash"), IDs: []string{}, PreferredRace: str("terran"),
		},
		"https://liquipedia.net/starcraft/Skip": {
			Link: "https://liquipedia.net/starcraft/Skip", Name: str("Skip"), IDs: []string{}, PreferredRace: str("protoss"),
		},
	}, nil)

	page := model.TournamentPage{
		Link: "https://liquipedia.net/starcraft/ASL/20",
		Name: str("ASL Season 20"),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str("https://liquipedia.net/starcraft/Jaedong"), Race: str("zerg")},
			{Name: str("Flash"), Link: str("https://liquipedia.net/starcraft/Flash"), Race: str("terran")},
			{Name: str("Skip"), Link: str("https://liquipedia.net/starcraft/Skip"), Race: str("protoss"), Excluded: true},
		},
		Groups: []model.TournamentGroup{
			{
				Name: "Group A", Phase: "Round of 24", SortOrder: 0,
				Players: []model.Participant{
					{Name: str("Jaedong"), Link: str("https://liquipedia.net/starcraft/Jaedong"), Race: str("zerg")},
					{Name: str("Flash"), Link: str("https://liquipedia.net/starcraft/Flash"), Race: str("terran")},
				},
			},
		},
		Results: []model.Result{},
	}
	if _, _, err := tourSvc.Save(ctx, page); err != nil {
		t.Fatal(err)
	}

	after, err := fantasySvc.ListPlayers(ctx, league.ID, repository.PlayerSortCost)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 2 {
		t.Fatalf("fantasy players after tournament resave=%d want 2", len(after))
	}
	byID := map[int64]model.FantasyPlayerRow{}
	for _, p := range after {
		byID[p.ID] = p
	}
	for _, p := range before {
		if _, ok := byID[p.ID]; !ok {
			t.Fatalf("fantasy player id=%d missing after resave", p.ID)
		}
	}
	if byID[before[0].ID].Cost != 42 {
		t.Fatalf("preserved cost=%d want 42", byID[before[0].ID].Cost)
	}
}

func TestFantasyListPlayersSortCostAndPoints(t *testing.T) {
	ctx, svc, repo, tournamentID := setupFantasyFixture(t)
	league, err := svc.CreateOrSeed(ctx, tournamentID)
	if err != nil {
		t.Fatal(err)
	}

	sqlDB := repo.DB()
	if _, err := sqlDB.ExecContext(ctx, `
		UPDATE fantasy_player SET cost = 100, points_ro24 = 1
		WHERE fantasy_league_id = ? AND tournament_player_id IN (
			SELECT tp.id FROM tournament_player tp
			JOIN player_alias pa ON pa.id = tp.player_alias_id
			WHERE tp.tournament_id = ? AND pa.name = 'Flash'
		)
	`, league.ID, tournamentID); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `
		UPDATE fantasy_player SET cost = 50, points_ro24 = 10
		WHERE fantasy_league_id = ? AND tournament_player_id IN (
			SELECT tp.id FROM tournament_player tp
			JOIN player_alias pa ON pa.id = tp.player_alias_id
			WHERE tp.tournament_id = ? AND pa.name = 'Jaedong'
		)
	`, league.ID, tournamentID); err != nil {
		t.Fatal(err)
	}

	byCost, err := svc.ListPlayers(ctx, league.ID, repository.PlayerSortCost)
	if err != nil {
		t.Fatal(err)
	}
	if len(byCost) < 2 || byCost[0].Name == nil || *byCost[0].Name != "Flash" {
		t.Fatalf("cost sort want Flash first, got %#v", byCost)
	}

	byPoints, err := svc.ListPlayers(ctx, league.ID, repository.PlayerSortPoints)
	if err != nil {
		t.Fatal(err)
	}
	if len(byPoints) < 2 || byPoints[0].Name == nil || *byPoints[0].Name != "Jaedong" {
		t.Fatalf("points sort want Jaedong first, got %#v", byPoints)
	}
}

func TestFantasyListTeamsIncludesAliasAndMembers(t *testing.T) {
	ctx, svc, repo, tournamentID := setupFantasyFixture(t)
	league, err := svc.CreateOrSeed(ctx, tournamentID)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB := repo.DB()

	res, err := sqlDB.ExecContext(ctx, `
		INSERT INTO user (alias, telegram_id, first_name, role)
		VALUES ('Commander', 9001, 'Commander', 'USER')
	`)
	if err != nil {
		t.Fatal(err)
	}
	userID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	tres, err := sqlDB.ExecContext(ctx, `
		INSERT INTO fantasy_team (fantasy_league_id, user_id) VALUES (?, ?)
	`, league.ID, userID)
	if err != nil {
		t.Fatal(err)
	}
	teamID, err := tres.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	var fpID int64
	if err := sqlDB.QueryRowContext(ctx, `
		SELECT id FROM fantasy_player WHERE fantasy_league_id = ? LIMIT 1
	`, league.ID).Scan(&fpID); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO fantasy_team_member (fantasy_team_id, fantasy_player_id) VALUES (?, ?)
	`, teamID, fpID); err != nil {
		t.Fatal(err)
	}

	teams, err := svc.ListTeams(ctx, league.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 1 || teams[0].UserAlias != "Commander" {
		t.Fatalf("teams=%#v", teams)
	}
	if len(teams[0].Members) != 1 {
		t.Fatalf("members=%#v", teams[0].Members)
	}
}

func TestFantasyGetActiveLeaguePrefersUnfinishedNewest(t *testing.T) {
	ctx, svc, repo, tournamentID := setupFantasyFixture(t)
	sqlDB := repo.DB()

	// Second tournament + leagues so we can mark finished independently.
	playerRepo := repository.NewPlayer(sqlDB)
	tourRepo := repository.NewTournament(sqlDB)
	tourSvc := service.NewTournament(sqlDB, tourRepo, playerRepo, stubPlayerFetcher{
		"https://liquipedia.net/starcraft/Jaedong": {
			Link: "https://liquipedia.net/starcraft/Jaedong", Name: str("Jaedong"), IDs: []string{}, PreferredRace: str("zerg"),
		},
	}, nil)
	page2 := model.TournamentPage{
		Link: "https://liquipedia.net/starcraft/ASL/21",
		Name: str("ASL Season 21"),
		Participants: []model.Participant{
			{Name: str("Jaedong"), Link: str("https://liquipedia.net/starcraft/Jaedong"), Race: str("zerg")},
		},
		Results: []model.Result{},
	}
	if _, _, err := tourSvc.Save(ctx, page2); err != nil {
		t.Fatal(err)
	}
	var tournamentID2 int64
	if err := sqlDB.QueryRowContext(ctx, `SELECT id FROM tournament WHERE link = ?`, page2.Link).Scan(&tournamentID2); err != nil {
		t.Fatal(err)
	}

	league1, err := svc.CreateOrSeed(ctx, tournamentID)
	if err != nil {
		t.Fatal(err)
	}
	league2, err := svc.CreateOrSeed(ctx, tournamentID2)
	if err != nil {
		t.Fatal(err)
	}
	if league2.ID <= league1.ID {
		t.Fatalf("expected league2 newer id=%d league1=%d", league2.ID, league1.ID)
	}

	active, err := svc.GetActiveLeague(ctx)
	if err != nil || active == nil {
		t.Fatalf("active=%v err=%v", active, err)
	}
	if active.ID != league2.ID {
		t.Fatalf("want unfinished newest id=%d got %d", league2.ID, active.ID)
	}

	if _, err := sqlDB.ExecContext(ctx, `UPDATE fantasy_league SET finished = 1 WHERE id = ?`, league2.ID); err != nil {
		t.Fatal(err)
	}
	active2, err := svc.GetActiveLeague(ctx)
	if err != nil || active2 == nil {
		t.Fatalf("active2=%v err=%v", active2, err)
	}
	if active2.ID != league1.ID {
		t.Fatalf("want unfinished league1 id=%d got %d", league1.ID, active2.ID)
	}

	if _, err := sqlDB.ExecContext(ctx, `UPDATE fantasy_league SET finished = 1 WHERE id = ?`, league1.ID); err != nil {
		t.Fatal(err)
	}
	active3, err := svc.GetActiveLeague(ctx)
	if err != nil || active3 == nil {
		t.Fatalf("active3=%v err=%v", active3, err)
	}
	if active3.ID != league2.ID {
		t.Fatalf("all finished want newest id=%d got %d", league2.ID, active3.ID)
	}
}

func TestComputeEloCostsDefaultsAndCustom(t *testing.T) {
	roster := []repository.RosterEloRow{
		{TournamentPlayerID: 1, Elo: 1000},
		{TournamentPlayerID: 2, Elo: 1500},
		{TournamentPlayerID: 3, Elo: 2000},
	}
	costs := service.ComputeEloCosts(roster, 0, 10)
	if costs[0] != 0 || costs[2] != 10 || costs[1] != 5 {
		t.Fatalf("default scale costs=%v", costs)
	}
	custom := service.ComputeEloCosts(roster, 2, 8)
	if custom[0] != 2 || custom[2] != 8 {
		t.Fatalf("custom scale costs=%v", custom)
	}
	flat := service.ComputeEloCosts([]repository.RosterEloRow{
		{Elo: 1200}, {Elo: 1200},
	}, 3, 9)
	if flat[0] != 3 || flat[1] != 3 {
		t.Fatalf("flat elo costs=%v", flat)
	}
}

func TestFantasyCreateRejectsUsedTournament(t *testing.T) {
	ctx, svc, _, tournamentID := setupFantasyFixture(t)
	if _, err := svc.Create(ctx, service.CreateParams{TournamentID: tournamentID}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Create(ctx, service.CreateParams{TournamentID: tournamentID})
	if err == nil || !errors.Is(err, service.ErrFantasyConflict) {
		t.Fatalf("err=%v", err)
	}
}

func TestFantasyLifecycleAndCaps(t *testing.T) {
	ctx, svc, _, tournamentID := setupFantasyFixture(t)
	league, err := svc.Create(ctx, service.CreateParams{TournamentID: tournamentID, MaxPlayers: 6, MaxCost: 28})
	if err != nil {
		t.Fatal(err)
	}
	if league.Started || league.Finished {
		t.Fatalf("league=%#v", league)
	}
	updated, err := svc.UpdateLeagueCaps(ctx, league.ID, 5, 20)
	if err != nil {
		t.Fatal(err)
	}
	if updated.MaxPlayers != 5 || updated.MaxCost != 20 {
		t.Fatalf("caps=%#v", updated)
	}
	started, err := svc.StartLeague(ctx, league.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !started.Started {
		t.Fatal("expected started")
	}
	if _, err := svc.UpdateLeagueCaps(ctx, league.ID, 4, 10); !errors.Is(err, service.ErrFantasyLeagueStarted) {
		t.Fatalf("err=%v", err)
	}
	finished, err := svc.FinishLeague(ctx, league.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !finished.Finished {
		t.Fatal("expected finished")
	}
}

func TestFantasyTeamCapsAndLock(t *testing.T) {
	ctx, svc, repo, tournamentID := setupFantasyFixture(t)
	league, err := svc.Create(ctx, service.CreateParams{
		TournamentID: tournamentID,
		MaxPlayers:   1,
		MaxCost:      100,
		CostMin:      0,
		CostMax:      10,
	})
	if err != nil {
		t.Fatal(err)
	}
	players, err := svc.ListPlayers(ctx, league.ID, repository.PlayerSortCost)
	if err != nil || len(players) < 2 {
		t.Fatalf("players=%v err=%v", players, err)
	}
	res, err := repo.DB().ExecContext(ctx, `
		INSERT INTO user (alias, first_name, role) VALUES ('Pilot', 'Pilot', 'USER')
	`)
	if err != nil {
		t.Fatal(err)
	}
	userID, _ := res.LastInsertId()

	ids := []int64{players[0].ID, players[1].ID}
	_, err = svc.UpsertTeam(ctx, service.UpsertTeamParams{
		LeagueID: league.ID, UserID: userID, FantasyPlayerIDs: ids, RequireNotStarted: true,
	})
	if err == nil || !errors.Is(err, service.ErrFantasyInvalid) {
		t.Fatalf("expected too many players err=%v", err)
	}

	team, err := svc.UpsertTeam(ctx, service.UpsertTeamParams{
		LeagueID: league.ID, UserID: userID, FantasyPlayerIDs: []int64{players[0].ID}, RequireNotStarted: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(team.Members) != 1 {
		t.Fatalf("members=%#v", team.Members)
	}

	if _, err := svc.StartLeague(ctx, league.ID); err != nil {
		t.Fatal(err)
	}
	_, err = svc.UpsertTeam(ctx, service.UpsertTeamParams{
		LeagueID: league.ID, UserID: userID, FantasyPlayerIDs: []int64{players[1].ID}, RequireNotStarted: true,
	})
	if !errors.Is(err, service.ErrFantasyTeamLocked) {
		t.Fatalf("err=%v", err)
	}
}

func TestFantasyPatchPlayerPointsAndWinner(t *testing.T) {
	ctx, svc, _, tournamentID := setupFantasyFixture(t)
	league, err := svc.Create(ctx, service.CreateParams{TournamentID: tournamentID})
	if err != nil {
		t.Fatal(err)
	}
	players, err := svc.ListPlayers(ctx, league.ID, repository.PlayerSortCost)
	if err != nil || len(players) < 2 {
		t.Fatal(err)
	}
	ten := 10
	five := 5
	p1, err := svc.PatchPlayer(ctx, league.ID, players[0].ID, service.PlayerPatch{
		SetRo24: true, PointsRo24: &ten,
		SetRo16: true, PointsRo16: &five,
		IsWinner: boolPtr(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if p1.PointsEarned != 15 || !p1.IsWinner {
		t.Fatalf("p1=%#v", p1)
	}
	p2, err := svc.PatchPlayer(ctx, league.ID, players[1].ID, service.PlayerPatch{
		IsWinner: boolPtr(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !p2.IsWinner {
		t.Fatal("expected p2 winner")
	}
	reload, err := svc.ListPlayers(ctx, league.ID, repository.PlayerSortCost)
	if err != nil {
		t.Fatal(err)
	}
	winners := 0
	for _, p := range reload {
		if p.IsWinner {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("winners=%d", winners)
	}
}

func TestAssignTeamRanksCompetition(t *testing.T) {
	teams := []model.FantasyTeamRow{
		{ID: 1, UserAlias: "A", Members: []model.FantasyTeamMemberRow{{PointsEarned: 10, Cost: 5}}},
		{ID: 2, UserAlias: "B", Members: []model.FantasyTeamMemberRow{{PointsEarned: 10, Cost: 8}}},
		{ID: 3, UserAlias: "C", Members: []model.FantasyTeamMemberRow{{PointsEarned: 4, Cost: 3}}},
	}
	out := service.AssignTeamRanks(teams)
	if len(out) != 3 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Rank != 1 || out[1].Rank != 1 || out[2].Rank != 3 {
		t.Fatalf("ranks=%d,%d,%d", out[0].Rank, out[1].Rank, out[2].Rank)
	}
	// Same rank: higher cost first.
	if out[0].UserAlias != "B" || out[1].UserAlias != "A" {
		t.Fatalf("order=%s,%s", out[0].UserAlias, out[1].UserAlias)
	}
}


