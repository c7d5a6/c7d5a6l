package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
)

// PlayerImporter fetches Liquipedia player pages from the import queue.
type PlayerImporter struct {
	db      *sql.DB
	queue   *repository.PlayerImport
	players *repository.Player
	fetcher PlayerPageFetcher
	assets  AssetFetcher
}

func NewPlayerImporter(
	db *sql.DB,
	queue *repository.PlayerImport,
	players *repository.Player,
	fetcher PlayerPageFetcher,
	assets AssetFetcher,
) *PlayerImporter {
	return &PlayerImporter{db: db, queue: queue, players: players, fetcher: fetcher, assets: assets}
}

// RecoverRunning re-queues rows left in running state after a crash.
func (s *PlayerImporter) RecoverRunning(ctx context.Context) (int64, error) {
	return s.queue.ResetRunningToPending(ctx, s.db)
}

// ProcessNext claims one pending import, fetches Liquipedia, upserts the player.
// Returns the link processed, or "" when idle.
func (s *PlayerImporter) ProcessNext(ctx context.Context) (string, error) {
	link, err := s.queue.ClaimNext(ctx, s.db)
	if err != nil {
		return "", err
	}
	if link == "" {
		return "", nil
	}
	if err := s.enrich(ctx, link); err != nil {
		log.Printf("player-import: enrich %s: %v", link, err)
		_ = s.queue.MarkError(ctx, s.db, link, err.Error())
		return "", err
	}
	if err := s.queue.MarkDone(ctx, s.db, link); err != nil {
		return link, err
	}
	return link, nil
}

func (s *PlayerImporter) enrich(ctx context.Context, link string) error {
	if s.fetcher == nil {
		return fmt.Errorf("player fetcher not configured")
	}
	canonical, err := liquipedia.ValidateURL(link)
	if err != nil {
		return err
	}
	debuglog.Printf("PlayerImporter.enrich fetch link=%s", canonical)
	pp, err := s.fetcher.FetchPlayerPage(ctx, canonical)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	pp.Link = canonical
	portrait, err := resolvePortrait(ctx, s.assets, pp, nil)
	if err != nil {
		return fmt.Errorf("portrait: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.players.Upsert(ctx, tx, pp, portrait); err != nil {
		return fmt.Errorf("upsert: %w", err)
	}
	return tx.Commit()
}

// ActiveCount returns pending+running imports.
func (s *PlayerImporter) ActiveCount(ctx context.Context) (int, error) {
	return s.queue.CountActive(ctx, s.db)
}
