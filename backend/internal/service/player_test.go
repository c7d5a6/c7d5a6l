package service_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/db"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

func str(s string) *string { return &s }

func TestPlayerSaveAndSync(t *testing.T) {
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

	page := model.PlayerPage{
		Link:          "https://liquipedia.net/starcraft/Jaedong",
		Name:          str("Jaedong"),
		RealName:      str("Lee Jae Dong"),
		IDs:           []string{"JD", "n.Die_Jaedong"},
		PreferredRace: str("zerg"),
	}

	sync, err := svc.SyncStatus(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if sync.Exists || sync.Action != model.PlayerActionAdd {
		t.Fatalf("want add, got exists=%v action=%s", sync.Exists, sync.Action)
	}

	saved, sync, err := svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if !sync.Exists || !sync.Same || sync.Action != model.PlayerActionNone {
		t.Fatalf("after save want same, got %+v", sync)
	}
	if saved.Name == nil || *saved.Name != "Jaedong" {
		t.Fatalf("saved name=%v", saved.Name)
	}

	changed := page
	changed.IDs = append(append([]string{}, page.IDs...), "n.Die_yOngKIN")
	sync, err = svc.SyncStatus(ctx, changed)
	if err != nil {
		t.Fatal(err)
	}
	if !sync.Exists || sync.Same || sync.Action != model.PlayerActionUpdate {
		t.Fatalf("want update, got %+v", sync)
	}
	if len(sync.Changes) == 0 {
		t.Fatal("expected changes")
	}

	_, sync, err = svc.Save(ctx, changed)
	if err != nil {
		t.Fatal(err)
	}
	if !sync.Same {
		t.Fatalf("after update want same, got %+v", sync)
	}

	got, err := repo.GetByLink(ctx, sqlDB, page.Link)
	if err != nil || got == nil {
		t.Fatalf("get: %v %#v", err, got)
	}
	if len(got.IDs) != 3 {
		t.Fatalf("ids=%v", got.IDs)
	}
}

func TestPlayerSyncStatusMissingAlternateID(t *testing.T) {
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
	link := "https://liquipedia.net/starcraft/Jaedong"

	savedPartial := model.PlayerPage{
		Link:          link,
		Name:          str("Jaedong"),
		RealName:      str("Lee Jae Dong"),
		IDs:           []string{"JD"},
		PreferredRace: str("zerg"),
	}
	if _, _, err := svc.Save(ctx, savedPartial); err != nil {
		t.Fatal(err)
	}

	parsedFull := model.PlayerPage{
		Link:          link,
		Name:          str("Jaedong"),
		RealName:      str("Lee Jae Dong"),
		IDs:           []string{"JD", "n.Die_Jaedong"},
		PreferredRace: str("zerg"),
	}
	sync, err := svc.SyncStatus(ctx, parsedFull)
	if err != nil {
		t.Fatal(err)
	}
	if !sync.Exists || sync.Same || sync.Action != model.PlayerActionUpdate {
		t.Fatalf("missing alternate ID: want exists=true same=false action=update, got exists=%v same=%v action=%s changes=%v storedIDs=%v",
			sync.Exists, sync.Same, sync.Action, sync.Changes, sync.Stored.IDs)
	}
}

func TestPlayerSyncStatusSameIDsNone(t *testing.T) {
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
	link := "https://liquipedia.net/starcraft/Jaedong"

	page := model.PlayerPage{
		Link:          link,
		Name:          str("Jaedong"),
		RealName:      str("Lee Jae Dong"),
		IDs:           []string{"JD", "n.Die_Jaedong"},
		PreferredRace: str("zerg"),
	}
	if _, _, err := svc.Save(ctx, page); err != nil {
		t.Fatal(err)
	}

	sync, err := svc.SyncStatus(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if !sync.Exists || !sync.Same || sync.Action != model.PlayerActionNone {
		t.Fatalf("identical IDs: want exists=true same=true action=none, got exists=%v same=%v action=%s changes=%v",
			sync.Exists, sync.Same, sync.Action, sync.Changes)
	}
}

// Primary name is stored in player_alias by syncAliases, but GetByLink excludes it from IDs.
// SyncStatus must not treat that alias row as making alternate-ID sets equal when an ID is missing,
// and must not report a spurious ids change solely because the primary is also an alias.
func TestPlayerSyncStatusPrimaryNameAliasDoesNotMaskMissingID(t *testing.T) {
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
	link := "https://liquipedia.net/starcraft/Jaedong"

	if _, _, err := svc.Save(ctx, model.PlayerPage{
		Link:          link,
		Name:          str("Jaedong"),
		RealName:      str("Lee Jae Dong"),
		IDs:           []string{"JD"},
		PreferredRace: str("zerg"),
	}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByLink(ctx, sqlDB, link)
	if err != nil || got == nil {
		t.Fatalf("get: %v %#v", err, got)
	}
	for _, id := range got.IDs {
		if strings.EqualFold(id, "Jaedong") {
			t.Fatalf("GetByLink must exclude primary name from IDs, got %v", got.IDs)
		}
	}

	// Parsed alternate IDs do not include the main nickname (parse.playerIDs).
	sync, err := svc.SyncStatus(ctx, model.PlayerPage{
		Link:          link,
		Name:          str("Jaedong"),
		RealName:      str("Lee Jae Dong"),
		IDs:           []string{"JD", "n.Die_Jaedong"},
		PreferredRace: str("zerg"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if sync.Same || sync.Action != model.PlayerActionUpdate {
		t.Fatalf("primary alias must not mask missing alternate ID: got same=%v action=%s changes=%v storedIDs=%v",
			sync.Same, sync.Action, sync.Changes, sync.Stored.IDs)
	}

	// After saving full alternates, primary still stored as alias but SyncStatus is none.
	full := model.PlayerPage{
		Link:          link,
		Name:          str("Jaedong"),
		RealName:      str("Lee Jae Dong"),
		IDs:           []string{"JD", "n.Die_Jaedong"},
		PreferredRace: str("zerg"),
	}
	if _, _, err := svc.Save(ctx, full); err != nil {
		t.Fatal(err)
	}
	sync, err = svc.SyncStatus(ctx, full)
	if err != nil {
		t.Fatal(err)
	}
	if !sync.Same || sync.Action != model.PlayerActionNone {
		t.Fatalf("after full save want none, got same=%v action=%s changes=%v", sync.Same, sync.Action, sync.Changes)
	}

	// If parse mistakenly included primary name in IDs, compare currently flags update
	// (stored IDs exclude primary). Document that behavior.
	withPrimaryInIDs := full
	withPrimaryInIDs.IDs = []string{"Jaedong", "JD", "n.Die_Jaedong"}
	sync, err = svc.SyncStatus(ctx, withPrimaryInIDs)
	if err != nil {
		t.Fatal(err)
	}
	if sync.Same {
		t.Fatalf("primary in parsed IDs currently differs from stored alternates-only; got same=true storedIDs=%v", sync.Stored.IDs)
	}
}

type stubAssets map[string]struct {
	data []byte
	mime string
}

func (s stubAssets) FetchBytes(_ context.Context, assetURL string) ([]byte, string, error) {
	got, ok := s[assetURL]
	if !ok {
		return nil, "", fmt.Errorf("stub asset missing %s", assetURL)
	}
	return got.data, got.mime, nil
}

func TestPlayerListRaceEntriesSortedByElo(t *testing.T) {
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

	if _, _, err := svc.Save(ctx, model.PlayerPage{
		Link: "https://liquipedia.net/starcraft/Flash", Name: str("Flash"), IDs: []string{}, PreferredRace: str("terran"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Save(ctx, model.PlayerPage{
		Link: "https://liquipedia.net/starcraft/Jaedong", Name: str("Jaedong"), IDs: []string{}, PreferredRace: str("zerg"),
	}); err != nil {
		t.Fatal(err)
	}

	flashID, err := repo.IDByLink(ctx, sqlDB, "https://liquipedia.net/starcraft/Flash")
	if err != nil || flashID == 0 {
		t.Fatalf("flash id: %v %d", err, flashID)
	}
	if _, err := sqlDB.ExecContext(ctx, `UPDATE player_race SET elo = 1800 WHERE player_id = ? AND race = 'terran'`, flashID); err != nil {
		t.Fatal(err)
	}

	list, err := svc.ListRaceEntries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 2 {
		t.Fatalf("want >=2 entries, got %d", len(list))
	}
	if list[0].Elo < list[1].Elo {
		t.Fatalf("expected elo descending, got %#v then %#v", list[0], list[1])
	}
	if list[0].Name == nil || *list[0].Name != "Flash" {
		t.Fatalf("top by elo want Flash, got %#v", list[0])
	}
}

func TestPlayerUpdateRaceEloSyncsSeasonBaseline(t *testing.T) {
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
	seasonRepo := repository.NewSeason(sqlDB)
	seasonSvc := service.NewSeason(sqlDB, seasonRepo, playerRepo)
	playerSvc := service.NewPlayer(sqlDB, playerRepo, nil)

	if _, _, err := playerSvc.Save(ctx, model.PlayerPage{
		Link: "https://liquipedia.net/starcraft/Flash", Name: str("Flash"), IDs: []string{}, PreferredRace: str("terran"),
	}); err != nil {
		t.Fatal(err)
	}
	list, err := playerSvc.ListRaceEntries(ctx)
	if err != nil || len(list) == 0 {
		t.Fatalf("list: %v %#v", err, list)
	}
	id := list[0].PlayerRaceID
	if _, err := playerSvc.UpdateRaceElo(ctx, id, 2050); err != nil {
		t.Fatal(err)
	}
	if err := seasonSvc.SyncActiveSeasonStartElo(ctx, id, 2050); err != nil {
		t.Fatal(err)
	}

	entries, _, err := seasonSvc.ListRaceEntriesWithSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var row *model.PlayerRaceEntry
	for i := range entries {
		if entries[i].PlayerRaceID == id {
			row = &entries[i]
			break
		}
	}
	if row == nil {
		t.Fatal("missing entry after update")
	}
	if row.Elo != 2050 {
		t.Fatalf("stored elo=%v want 2050", row.Elo)
	}
	if row.ProjectedElo == nil || *row.ProjectedElo != 2050 {
		t.Fatalf("projected elo=%v want 2050 with no matches", row.ProjectedElo)
	}
}

func TestPlayerUpdateRaceElo(t *testing.T) {
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

	if _, _, err := svc.Save(ctx, model.PlayerPage{
		Link: "https://liquipedia.net/starcraft/Flash", Name: str("Flash"), IDs: []string{}, PreferredRace: str("terran"),
	}); err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListRaceEntries(ctx)
	if err != nil || len(list) == 0 {
		t.Fatalf("list: %v %#v", err, list)
	}
	id := list[0].PlayerRaceID
	updated, err := svc.UpdateRaceElo(ctx, id, 2010.4)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Elo != 2010 {
		t.Fatalf("elo=%v, want 2010", updated.Elo)
	}

	if _, err := svc.UpdateRaceElo(ctx, id, -1); err == nil {
		t.Fatal("expected invalid elo")
	}
	if _, err := svc.UpdateRaceElo(ctx, 999999, 1000); !errors.Is(err, service.ErrPlayerNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestPlayerSaveCachesPortraitBlob(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	portraitURL := "https://liquipedia.net/commons/images/test/portrait.png"
	assets := stubAssets{
		portraitURL: {data: []byte("fake-png-bytes"), mime: "image/png"},
	}
	repo := repository.NewPlayer(sqlDB)
	svc := service.NewPlayer(sqlDB, repo, assets)

	page := model.PlayerPage{
		Link:          "https://liquipedia.net/starcraft/Jaedong",
		Name:          str("Jaedong"),
		IDs:           []string{},
		PreferredRace: str("zerg"),
		PortraitURL:   str(portraitURL),
	}
	saved, _, err := svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if !saved.HasPortrait {
		t.Fatal("expected hasPortrait after save")
	}

	data, mime, err := svc.Portrait(ctx, page.Link)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake-png-bytes" || mime != "image/png" {
		t.Fatalf("portrait data=%q mime=%q", data, mime)
	}

	// Second save with same URL must not re-fetch (delete stub entry to prove).
	delete(assets, portraitURL)
	saved, _, err = svc.Save(ctx, page)
	if err != nil {
		t.Fatal(err)
	}
	if !saved.HasPortrait {
		t.Fatal("expected portrait retained")
	}
}
