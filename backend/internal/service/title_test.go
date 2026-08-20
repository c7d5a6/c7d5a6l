package service_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/db"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

// 1x1 PNG.
var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

func setupTitleSvc(t *testing.T) (context.Context, *sql.DB, *service.Title, int64, int64) {
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
	userRepo := repository.NewUser(sqlDB)
	userID, err := userRepo.Insert(ctx, sqlDB, model.User{Alias: "Commander", FirstName: "Jim", Role: model.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	res, err := sqlDB.ExecContext(ctx, `
		INSERT INTO tournament (link, name, finished) VALUES (?, ?, 0)
	`, "https://liquipedia.net/starcraft/ASL/20", "ASL Season 20")
	if err != nil {
		t.Fatal(err)
	}
	tid, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	res, err = sqlDB.ExecContext(ctx, `
		INSERT INTO fantasy_league (tournament_id, started, finished, max_players, max_cost)
		VALUES (?, 0, 0, 6, 28)
	`, tid)
	if err != nil {
		t.Fatal(err)
	}
	leagueID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewTitle(sqlDB, repository.NewTitle(sqlDB), userRepo, repository.NewFantasy(sqlDB))
	return ctx, sqlDB, svc, userID, leagueID
}

func TestTitleCreateAndImageRoundTrip(t *testing.T) {
	ctx, _, svc, userID, leagueID := setupTitleSvc(t)
	leagueIDCopy := leagueID
	got, err := svc.Create(ctx, service.TitleParams{
		UserID:          userID,
		Kind:            model.TitleKindFantasy,
		Name:            "ASL 20 Champion",
		FantasyLeagueID: &leagueIDCopy,
		Image:           tinyPNG,
		ImageMime:       "image/png",
		ImageOp:         service.ImageSet,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasImage || got.UserAlias != "Commander" || got.Kind != model.TitleKindFantasy {
		t.Fatalf("got %#v", got)
	}
	data, mime, err := svc.GetImage(ctx, got.ID)
	if err != nil {
		t.Fatal(err)
	}
	if mime != "image/png" || !bytes.Equal(data, tinyPNG) {
		t.Fatalf("image mime=%s len=%d", mime, len(data))
	}
}

func TestTitleUniquePerLeague(t *testing.T) {
	ctx, sqlDB, svc, userID, leagueID := setupTitleSvc(t)
	leagueIDCopy := leagueID
	if _, err := svc.Create(ctx, service.TitleParams{
		UserID: userID, Kind: model.TitleKindFantasy, Name: "One", FantasyLeagueID: &leagueIDCopy,
	}); err != nil {
		t.Fatal(err)
	}
	user2, err := repository.NewUser(sqlDB).Insert(ctx, sqlDB, model.User{Alias: "Raynor", FirstName: "Jim", Role: model.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Create(ctx, service.TitleParams{
		UserID: user2, Kind: model.TitleKindFantasy, Name: "Two", FantasyLeagueID: &leagueIDCopy,
	})
	if !errors.Is(err, service.ErrTitleConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
}

func TestTitleLeagueDeleteNullsFK(t *testing.T) {
	ctx, sqlDB, svc, userID, leagueID := setupTitleSvc(t)
	leagueIDCopy := leagueID
	got, err := svc.Create(ctx, service.TitleParams{
		UserID: userID, Kind: model.TitleKindFantasy, Name: "Champ", FantasyLeagueID: &leagueIDCopy,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `DELETE FROM fantasy_league WHERE id = ?`, leagueID); err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListByUserID(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != got.ID || list[0].FantasyLeagueID != nil {
		t.Fatalf("after league delete got %#v", list)
	}
}

func TestTitleAttachToTeams(t *testing.T) {
	ctx, _, svc, userID, _ := setupTitleSvc(t)
	if _, err := svc.Create(ctx, service.TitleParams{
		UserID: userID, Kind: model.TitleKindTournament, Name: "Cup",
	}); err != nil {
		t.Fatal(err)
	}
	teams := []model.FantasyTeamRow{{UserID: userID, UserAlias: "Commander"}}
	if err := svc.AttachToTeams(ctx, teams); err != nil {
		t.Fatal(err)
	}
	if len(teams[0].Titles) != 1 || teams[0].Titles[0].Name != "Cup" {
		t.Fatalf("titles=%#v", teams[0].Titles)
	}
}

func TestTitleClearImage(t *testing.T) {
	ctx, _, svc, userID, _ := setupTitleSvc(t)
	got, err := svc.Create(ctx, service.TitleParams{
		UserID: userID, Kind: model.TitleKindTournament, Name: "Cup",
		Image: tinyPNG, ImageMime: "image/png", ImageOp: service.ImageSet,
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.Update(ctx, got.ID, service.TitleParams{
		UserID: userID, Kind: model.TitleKindTournament, Name: "Cup", ImageOp: service.ImageClear,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.HasImage {
		t.Fatal("expected image cleared")
	}
	data, _, err := svc.GetImage(ctx, got.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("blob still %d bytes", len(data))
	}
}

func TestTitleRejectsBadKindAndName(t *testing.T) {
	ctx, _, svc, userID, _ := setupTitleSvc(t)
	if _, err := svc.Create(ctx, service.TitleParams{UserID: userID, Kind: "gold", Name: "X"}); !errors.Is(err, service.ErrTitleInvalid) {
		t.Fatalf("kind err=%v", err)
	}
	if _, err := svc.Create(ctx, service.TitleParams{UserID: userID, Kind: model.TitleKindFantasy, Name: ""}); !errors.Is(err, service.ErrTitleInvalid) {
		t.Fatalf("name err=%v", err)
	}
}

func TestTitleDateRoundTripAndSort(t *testing.T) {
	ctx, _, svc, userID, _ := setupTitleSvc(t)
	bad := "13/08/2026"
	if _, err := svc.Create(ctx, service.TitleParams{
		UserID: userID, Kind: model.TitleKindTournament, Name: "Bad", Date: &bad,
	}); !errors.Is(err, service.ErrTitleInvalid) {
		t.Fatalf("invalid date err=%v", err)
	}

	d1 := "2024-06-01"
	d2 := "2025-01-15"
	older, err := svc.Create(ctx, service.TitleParams{
		UserID: userID, Kind: model.TitleKindTournament, Name: "Older", Date: &d1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if older.Date == nil || *older.Date != d1 {
		t.Fatalf("older date=%v", older.Date)
	}
	newer, err := svc.Create(ctx, service.TitleParams{
		UserID: userID, Kind: model.TitleKindTournament, Name: "Newer", Date: &d2,
	})
	if err != nil {
		t.Fatal(err)
	}
	undated, err := svc.Create(ctx, service.TitleParams{
		UserID: userID, Kind: model.TitleKindTournament, Name: "Undated",
	})
	if err != nil {
		t.Fatal(err)
	}
	if undated.Date != nil {
		t.Fatalf("undated want nil, got %v", *undated.Date)
	}

	empty := ""
	cleared, err := svc.Update(ctx, newer.ID, service.TitleParams{
		UserID: userID, Kind: model.TitleKindTournament, Name: "Newer", Date: &empty,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Date != nil {
		t.Fatalf("cleared date=%v", cleared.Date)
	}
	restored, err := svc.Update(ctx, newer.ID, service.TitleParams{
		UserID: userID, Kind: model.TitleKindTournament, Name: "Newer", Date: &d2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if restored.Date == nil || *restored.Date != d2 {
		t.Fatalf("restored date=%v", restored.Date)
	}

	list, err := svc.ListByUserID(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("len=%d", len(list))
	}
	if list[0].ID != newer.ID || list[1].ID != older.ID || list[2].ID != undated.ID {
		t.Fatalf("order=%s,%s,%s", list[0].Name, list[1].Name, list[2].Name)
	}
}
