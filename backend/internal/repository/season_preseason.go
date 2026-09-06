package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// PreSeasonName is the synthetic baseline season before the first rated season.
const PreSeasonName = "Pre-season"

// EnsurePreSeason bootstraps Season 1 when the season table is empty, then creates
// a closed Pre-season when missing.
func (r *Season) EnsurePreSeason(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin pre-season tx: %w", err)
	}
	defer tx.Rollback()

	var seasonCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM season`).Scan(&seasonCount); err != nil {
		return fmt.Errorf("count seasons: %w", err)
	}
	if seasonCount == 0 {
		if err := r.insertBootstrapSeason1(ctx, tx); err != nil {
			return err
		}
	}

	var exists int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM season WHERE name = ? COLLATE NOCASE
	`, PreSeasonName).Scan(&exists); err != nil {
		return fmt.Errorf("check pre-season: %w", err)
	}
	if exists > 0 {
		return tx.Commit()
	}

	first, err := r.getEarliestSeasonExcept(ctx, tx, PreSeasonName)
	if err != nil {
		return err
	}
	if first == nil {
		return tx.Commit()
	}

	snapshots, err := r.ListActiveSeasonSnapshots(ctx, tx, first.ID)
	if err != nil {
		return err
	}
	if len(snapshots) == 0 {
		return tx.Commit()
	}

	startedAt, closedAt, err := preSeasonBounds(first.StartedAt)
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO season (name, status, started_at, closed_at, ready_to_close)
		VALUES (?, 'closed', ?, ?, 0)
	`, PreSeasonName, startedAt, closedAt)
	if err != nil {
		return fmt.Errorf("insert pre-season: %w", err)
	}
	preID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	for _, snap := range snapshots {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO season_player_race (
				season_id, player_race_id, start_elo, end_elo, start_rank, end_rank
			) VALUES (?, ?, ?, ?, ?, ?)
		`, preID, snap.PlayerRaceID, snap.StartElo, snap.StartElo, snap.StartRank, snap.StartRank); err != nil {
			return fmt.Errorf("insert pre-season snapshot: %w", err)
		}
	}

	return tx.Commit()
}

func (r *Season) insertBootstrapSeason1(ctx context.Context, tx *sql.Tx) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.ExecContext(ctx, `
		INSERT INTO season (name, status, started_at, ready_to_close)
		VALUES ('Season 1', 'active', ?, 0)
	`, now)
	if err != nil {
		return fmt.Errorf("insert bootstrap season 1: %w", err)
	}
	seasonID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO season_player_race (season_id, player_race_id, start_elo, start_rank)
		SELECT
			?,
			pr.id,
			pr.elo,
			ROW_NUMBER() OVER (ORDER BY pr.elo DESC, pr.id ASC)
		FROM player_race pr
	`, seasonID); err != nil {
		return fmt.Errorf("insert bootstrap season snapshots: %w", err)
	}
	return nil
}

func (r *Season) getEarliestSeasonExcept(ctx context.Context, q DBTX, excludeName string) (*model.Season, error) {
	var (
		s        model.Season
		closedAt sql.NullString
		flID     sql.NullInt64
		ready    int
	)
	err := q.QueryRowContext(ctx, `
		SELECT id, name, status, started_at, closed_at, ready_to_close, closing_fantasy_league_id
		FROM season
		WHERE name != ? COLLATE NOCASE
		ORDER BY started_at ASC, id ASC
		LIMIT 1
	`, excludeName).Scan(&s.ID, &s.Name, &s.Status, &s.StartedAt, &closedAt, &ready, &flID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get earliest season: %w", err)
	}
	s.ReadyToClose = ready != 0
	if closedAt.Valid {
		v := closedAt.String
		s.ClosedAt = &v
	}
	if flID.Valid {
		v := flID.Int64
		s.ClosingFantasyLeagueID = &v
	}
	return &s, nil
}

func preSeasonBounds(firstSeasonStartedAt string) (startedAt, closedAt string, err error) {
	startDay, err := seasonCalendarDay(firstSeasonStartedAt)
	if err != nil {
		return "", "", err
	}
	preDay := startDay.AddDate(0, 0, -1)
	return preDay.Format(time.RFC3339), startDay.Format(time.RFC3339), nil
}

func seasonCalendarDay(iso string) (time.Time, error) {
	s := strings.TrimSpace(iso)
	if s == "" {
		return time.Time{}, fmt.Errorf("season date is required")
	}
	if len(s) >= 10 {
		s = s[:10]
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse season date %q: %w", iso, err)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
}
