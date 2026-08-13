package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
)

var (
	ErrPlayerNotFound = errors.New("player not found")
	ErrInvalidPlayer  = errors.New("invalid player")
)

// AssetFetcher downloads Liquipedia-hosted binary assets (portraits).
type AssetFetcher interface {
	FetchBytes(ctx context.Context, assetURL string) ([]byte, string, error)
}

// Player orchestrates player DB lookups and writes.
type Player struct {
	db     *sql.DB
	repo   *repository.Player
	assets AssetFetcher
}

func NewPlayer(db *sql.DB, repo *repository.Player, assets AssetFetcher) *Player {
	return &Player{db: db, repo: repo, assets: assets}
}

// SyncStatus loads the DB row for page.Link (if any) and compares to page.
func (s *Player) SyncStatus(ctx context.Context, page model.PlayerPage) (model.PlayerSync, error) {
	stored, err := s.repo.GetByLink(ctx, s.db, page.Link)
	if err != nil {
		return model.PlayerSync{}, err
	}
	return ComparePlayer(page, stored), nil
}

// GetByLink returns the stored player page, or nil when missing.
func (s *Player) GetByLink(ctx context.Context, link string) (*model.PlayerPage, error) {
	return s.repo.GetByLink(ctx, s.db, link)
}

// Portrait returns cached portrait bytes for a player Liquipedia link.
func (s *Player) Portrait(ctx context.Context, link string) ([]byte, string, error) {
	return s.repo.GetPortraitByLink(ctx, s.db, link)
}

// ListRaceEntries returns player_race rows merged with player info, sorted by elo descending.
func (s *Player) ListRaceEntries(ctx context.Context) ([]model.PlayerRaceEntry, error) {
	return s.repo.ListRaceEntries(ctx, s.db)
}

// UpdateRaceElo sets elo for one player_race row (fantasy costs are untouched).
func (s *Player) UpdateRaceElo(ctx context.Context, playerRaceID int64, elo float64) (model.PlayerRaceEntry, error) {
	if playerRaceID <= 0 {
		return model.PlayerRaceEntry{}, fmt.Errorf("%w: playerRaceId is required", ErrInvalidPlayer)
	}
	if math.IsNaN(elo) || math.IsInf(elo, 0) || elo < 0 || elo > 9999 {
		return model.PlayerRaceEntry{}, fmt.Errorf("%w: elo must be between 0 and 9999", ErrInvalidPlayer)
	}
	elo = math.Round(elo)

	ok, err := s.repo.UpdateRaceElo(ctx, s.db, playerRaceID, elo)
	if err != nil {
		return model.PlayerRaceEntry{}, err
	}
	if !ok {
		return model.PlayerRaceEntry{}, ErrPlayerNotFound
	}
	entry, err := s.repo.GetRaceEntryByID(ctx, s.db, playerRaceID)
	if err != nil {
		return model.PlayerRaceEntry{}, err
	}
	if entry == nil {
		return model.PlayerRaceEntry{}, ErrPlayerNotFound
	}
	debuglog.Printf("service.Player.UpdateRaceElo playerRaceId=%d race=%s elo=%.0f", playerRaceID, entry.Race, entry.Elo)
	return *entry, nil
}

// Save upserts the given parsed player payload inside a transaction, then returns fresh sync status.
// Downloads portrait bytes outside the TX when needed.
func (s *Player) Save(ctx context.Context, page model.PlayerPage) (model.PlayerPage, model.PlayerSync, error) {
	if page.Link == "" {
		return model.PlayerPage{}, model.PlayerSync{}, fmt.Errorf("player link is required")
	}
	debuglog.Printf("service.Player.Save link=%s", page.Link)

	stored, err := s.repo.GetByLink(ctx, s.db, page.Link)
	if err != nil {
		return model.PlayerPage{}, model.PlayerSync{}, err
	}
	portrait, err := resolvePortrait(ctx, s.assets, page, stored)
	if err != nil {
		return model.PlayerPage{}, model.PlayerSync{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.PlayerPage{}, model.PlayerSync{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.repo.Upsert(ctx, tx, page, portrait); err != nil {
		return model.PlayerPage{}, model.PlayerSync{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.PlayerPage{}, model.PlayerSync{}, fmt.Errorf("commit: %w", err)
	}

	saved, err := s.repo.GetByLink(ctx, s.db, page.Link)
	if err != nil {
		return model.PlayerPage{}, model.PlayerSync{}, err
	}
	if saved == nil {
		return model.PlayerPage{}, model.PlayerSync{}, fmt.Errorf("player missing after save")
	}
	sync := ComparePlayer(*saved, saved)
	return *saved, sync, nil
}

// resolvePortrait downloads a portrait when the source URL is new or the blob is missing.
// Returns nil when the existing blob should be left unchanged.
func resolvePortrait(ctx context.Context, assets AssetFetcher, page model.PlayerPage, stored *model.PlayerPage) (*repository.PortraitBlob, error) {
	src := strings.TrimSpace(nullStr(page.PortraitURL))
	if src == "" || assets == nil {
		return nil, nil
	}
	if stored != nil && stored.HasPortrait && strPtrEqual(page.PortraitURL, stored.PortraitURL) {
		return nil, nil
	}
	debuglog.Printf("resolvePortrait fetch url=%s", src)
	data, mime, err := assets.FetchBytes(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("fetch portrait: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	return &repository.PortraitBlob{Data: data, Mime: mime}, nil
}

// ComparePlayer builds sync metadata for a parsed page vs an optional stored row.
func ComparePlayer(parsed model.PlayerPage, stored *model.PlayerPage) model.PlayerSync {
	if stored == nil {
		return model.PlayerSync{
			Exists: false,
			Same:   false,
			Action: model.PlayerActionAdd,
		}
	}

	var changes []model.FieldChange
	if !strPtrEqual(parsed.Name, stored.Name) {
		changes = append(changes, model.FieldChange{
			Field:  "name",
			Before: strOrNil(stored.Name),
			After:  strOrNil(parsed.Name),
		})
	}
	if !strPtrEqual(parsed.RealName, stored.RealName) {
		changes = append(changes, model.FieldChange{
			Field:  "realName",
			Before: strOrNil(stored.RealName),
			After:  strOrNil(parsed.RealName),
		})
	}
	if !strPtrEqual(parsed.PreferredRace, stored.PreferredRace) {
		changes = append(changes, model.FieldChange{
			Field:  "preferredRace",
			Before: strOrNil(stored.PreferredRace),
			After:  strOrNil(parsed.PreferredRace),
		})
	}
	if !strPtrEqual(parsed.PortraitURL, stored.PortraitURL) {
		changes = append(changes, model.FieldChange{
			Field:  "portraitUrl",
			Before: strOrNil(stored.PortraitURL),
			After:  strOrNil(parsed.PortraitURL),
		})
	}
	if !stringSetEqual(parsed.IDs, stored.IDs) {
		changes = append(changes, model.FieldChange{
			Field:  "ids",
			Before: normalizeIDList(stored.IDs),
			After:  normalizeIDList(parsed.IDs),
		})
	}

	same := len(changes) == 0
	action := model.PlayerActionNone
	if !same {
		action = model.PlayerActionUpdate
	}
	storedCopy := *stored
	return model.PlayerSync{
		Exists:  true,
		Same:    same,
		Action:  action,
		Stored:  &storedCopy,
		Changes: changes,
	}
}

func strPtrEqual(a, b *string) bool {
	return strings.TrimSpace(nullStr(a)) == strings.TrimSpace(nullStr(b))
}

func strOrNil(p *string) any {
	if p == nil || *p == "" {
		return nil
	}
	return *p
}

func nullStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func normalizeIDList(ids []string) []string {
	seen := make(map[string]string)
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = id
	}
	out := make([]string, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func stringSetEqual(a, b []string) bool {
	na := normalizeIDList(a)
	nb := normalizeIDList(b)
	if len(na) != len(nb) {
		return false
	}
	for i := range na {
		if strings.ToLower(na[i]) != strings.ToLower(nb[i]) {
			return false
		}
	}
	return true
}
