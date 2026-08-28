package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/rating"
)

// Season persists season lifecycle data.
type Season struct {
	db *sql.DB
}

func NewSeason(db *sql.DB) *Season {
	return &Season{db: db}
}

func (r *Season) DB() *sql.DB {
	return r.db
}

// SeasonPlayerSnapshot is one row from season_player_race.
type SeasonPlayerSnapshot struct {
	PlayerRaceID int64
	StartElo     float64
	EndElo       *float64
	StartRank    int
	EndRank      *int
}

// GetActiveSeason returns the active season or nil when missing.
func (r *Season) GetActiveSeason(ctx context.Context, q DBTX) (*model.Season, error) {
	var (
		s         model.Season
		closedAt  sql.NullString
		flID      sql.NullInt64
		flName    sql.NullString
		ready     int
	)
	err := q.QueryRowContext(ctx, `
		SELECT
			s.id, s.name, s.status, s.started_at, s.closed_at, s.ready_to_close, s.closing_fantasy_league_id,
			t.name
		FROM season s
		LEFT JOIN fantasy_league fl ON fl.id = s.closing_fantasy_league_id
		LEFT JOIN tournament t ON t.id = fl.tournament_id
		WHERE s.status = 'active'
		ORDER BY s.id DESC
		LIMIT 1
	`).Scan(&s.ID, &s.Name, &s.Status, &s.StartedAt, &closedAt, &ready, &flID, &flName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active season: %w", err)
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
	if flName.Valid {
		v := flName.String
		s.ClosingFantasyLeagueName = &v
	}
	return &s, nil
}

// GetSeasonByID loads one season by id.
func (r *Season) GetSeasonByID(ctx context.Context, q DBTX, id int64) (*model.Season, error) {
	var (
		s        model.Season
		closedAt sql.NullString
		flID     sql.NullInt64
		flName   sql.NullString
		ready    int
	)
	err := q.QueryRowContext(ctx, `
		SELECT
			s.id, s.name, s.status, s.started_at, s.closed_at, s.ready_to_close, s.closing_fantasy_league_id,
			t.name
		FROM season s
		LEFT JOIN fantasy_league fl ON fl.id = s.closing_fantasy_league_id
		LEFT JOIN tournament t ON t.id = fl.tournament_id
		WHERE s.id = ?
	`, id).Scan(&s.ID, &s.Name, &s.Status, &s.StartedAt, &closedAt, &ready, &flID, &flName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get season: %w", err)
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
	if flName.Valid {
		v := flName.String
		s.ClosingFantasyLeagueName = &v
	}
	return &s, nil
}

// GetPreviousClosedSeason returns the most recently closed season before the given id, or nil.
func (r *Season) GetPreviousClosedSeason(ctx context.Context, q DBTX, beforeSeasonID int64) (*model.Season, error) {
	var (
		s        model.Season
		closedAt sql.NullString
		flID     sql.NullInt64
		ready    int
	)
	err := q.QueryRowContext(ctx, `
		SELECT id, name, status, started_at, closed_at, ready_to_close, closing_fantasy_league_id
		FROM season
		WHERE status = 'closed' AND id < ?
		ORDER BY id DESC
		LIMIT 1
	`, beforeSeasonID).Scan(&s.ID, &s.Name, &s.Status, &s.StartedAt, &closedAt, &ready, &flID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get previous closed season: %w", err)
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

// ListActiveSeasonSnapshots returns start snapshots for the active season.
func (r *Season) ListActiveSeasonSnapshots(ctx context.Context, q DBTX, seasonID int64) ([]SeasonPlayerSnapshot, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT player_race_id, start_elo, end_elo, start_rank, end_rank
		FROM season_player_race
		WHERE season_id = ?
	`, seasonID)
	if err != nil {
		return nil, fmt.Errorf("list season snapshots: %w", err)
	}
	defer rows.Close()

	out := make([]SeasonPlayerSnapshot, 0)
	for rows.Next() {
		var snap SeasonPlayerSnapshot
		var endElo sql.NullFloat64
		var endRank sql.NullInt64
		if err := rows.Scan(&snap.PlayerRaceID, &snap.StartElo, &endElo, &snap.StartRank, &endRank); err != nil {
			return nil, err
		}
		if endElo.Valid {
			v := endElo.Float64
			snap.EndElo = &v
		}
		if endRank.Valid {
			v := int(endRank.Int64)
			snap.EndRank = &v
		}
		out = append(out, snap)
	}
	return out, rows.Err()
}

// ListSeasonStartRanks returns player_race_id → start_rank for a season.
func (r *Season) ListSeasonStartRanks(ctx context.Context, q DBTX, seasonID int64) (map[int64]int, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT player_race_id, start_rank FROM season_player_race WHERE season_id = ?
	`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64]int)
	for rows.Next() {
		var id int64
		var rank int
		if err := rows.Scan(&id, &rank); err != nil {
			return nil, err
		}
		out[id] = rank
	}
	return out, rows.Err()
}

// ListTournamentsInSeasonWindow returns tournaments fully contained in [seasonStart, now].
func (r *Season) ListTournamentsInSeasonWindow(ctx context.Context, q DBTX, seasonStartedAt string, nowISO string) ([]model.ClosePreviewTournament, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, link, name, start_date, end_date, finished
		FROM tournament
		WHERE start_date IS NOT NULL
			AND end_date IS NOT NULL
			AND start_date >= ?
			AND end_date <= ?
		ORDER BY start_date ASC, id ASC
	`, seasonStartedAt, nowISO)
	if err != nil {
		return nil, fmt.Errorf("list season window tournaments: %w", err)
	}
	defer rows.Close()

	out := make([]model.ClosePreviewTournament, 0)
	for rows.Next() {
		var t model.ClosePreviewTournament
		var name, start, end sql.NullString
		var finished int
		if err := rows.Scan(&t.ID, &t.Link, &name, &start, &end, &finished); err != nil {
			return nil, err
		}
		if name.Valid {
			v := name.String
			t.Name = &v
		}
		if start.Valid {
			v := start.String
			t.StartDate = &v
		}
		if end.Valid {
			v := end.String
			t.EndDate = &v
		}
		t.Finished = finished != 0
		t.Selected = t.Finished
		out = append(out, t)
	}
	return out, rows.Err()
}

// FantasyLeagueTournamentID returns tournament_id for a fantasy league.
func (r *Season) FantasyLeagueTournamentID(ctx context.Context, q DBTX, leagueID int64) (int64, error) {
	var tourID int64
	err := q.QueryRowContext(ctx, `SELECT tournament_id FROM fantasy_league WHERE id = ?`, leagueID).Scan(&tourID)
	if err == sql.ErrNoRows {
		return 0, sql.ErrNoRows
	}
	return tourID, err
}

// SetReadyToClose marks the active season ready to close after a fantasy league finishes.
func (r *Season) SetReadyToClose(ctx context.Context, q DBTX, fantasyLeagueID int64) error {
	res, err := q.ExecContext(ctx, `
		UPDATE season
		SET ready_to_close = 1, closing_fantasy_league_id = ?
		WHERE status = 'active'
	`, fantasyLeagueID)
	if err != nil {
		return fmt.Errorf("set ready to close: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("no active season")
	}
	return nil
}

// CloseSeasonParams is input for closing a season inside a transaction.
type CloseSeasonParams struct {
	SeasonID           int64
	NewSeasonName      string
	ClosedAt           string
	StartedAt          string
	TournamentIDs      []int64
	EndElos            map[int64]float64
	EndRanks           map[int64]int
	StartElos          map[int64]float64
	StartRanks         map[int64]int
}

// CloseAndOpenSeason closes the active season and opens the next one atomically.
func (r *Season) CloseAndOpenSeason(ctx context.Context, tx *sql.Tx, p CloseSeasonParams) (int64, error) {
	res, err := tx.ExecContext(ctx, `
		UPDATE season
		SET status = 'closed', closed_at = ?, ready_to_close = 0
		WHERE id = ? AND status = 'active'
	`, p.ClosedAt, p.SeasonID)
	if err != nil {
		return 0, fmt.Errorf("close season: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n != 1 {
		return 0, errors.New("active season not found")
	}

	for prID, endElo := range p.EndElos {
		endRank := p.EndRanks[prID]
		if _, err := tx.ExecContext(ctx, `
			UPDATE season_player_race
			SET end_elo = ?, end_rank = ?
			WHERE season_id = ? AND player_race_id = ?
		`, endElo, endRank, p.SeasonID, prID); err != nil {
			return 0, fmt.Errorf("update season end snapshot: %w", err)
		}
	}

	for _, tourID := range p.TournamentIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO season_tournament (season_id, tournament_id, included_in_rating)
			VALUES (?, ?, 1)
		`, p.SeasonID, tourID); err != nil {
			return 0, fmt.Errorf("insert season tournament: %w", err)
		}
	}

	res, err = tx.ExecContext(ctx, `
		INSERT INTO season (name, status, started_at, ready_to_close)
		VALUES (?, 'active', ?, 0)
	`, p.NewSeasonName, p.StartedAt)
	if err != nil {
		return 0, fmt.Errorf("insert new season: %w", err)
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for prID, startElo := range p.StartElos {
		startRank := p.StartRanks[prID]
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO season_player_race (season_id, player_race_id, start_elo, start_rank)
			VALUES (?, ?, ?, ?)
		`, newID, prID, startElo, startRank); err != nil {
			return 0, fmt.Errorf("insert new season snapshot: %w", err)
		}
	}

	return newID, nil
}

// UpdatePlayerRaceElos batch-updates player_race.elo inside a transaction.
func (r *Season) UpdatePlayerRaceElos(ctx context.Context, tx *sql.Tx, elos map[int64]float64) error {
	for id, elo := range elos {
		if _, err := tx.ExecContext(ctx, `UPDATE player_race SET elo = ? WHERE id = ?`, elo, id); err != nil {
			return fmt.Errorf("update player_race elo: %w", err)
		}
	}
	return nil
}

// EnsureActiveSeasonSnapshot adds a player_race to the active season if missing.
func (r *Season) EnsureActiveSeasonSnapshot(ctx context.Context, q DBTX, playerRaceID int64, elo float64) error {
	active, err := r.GetActiveSeason(ctx, q)
	if err != nil || active == nil {
		return err
	}
	_, err = q.ExecContext(ctx, `
		INSERT OR IGNORE INTO season_player_race (season_id, player_race_id, start_elo, start_rank)
		SELECT ?, ?, ?,
			COALESCE((SELECT MAX(start_rank) FROM season_player_race WHERE season_id = ?), 0) + 1
	`, active.ID, playerRaceID, elo, active.ID)
	if err != nil {
		return fmt.Errorf("ensure season snapshot: %w", err)
	}
	return nil
}

// ListAllPlayerRaceElos returns every player_race id and elo.
func (r *Season) ListAllPlayerRaceElos(ctx context.Context, q DBTX) (map[int64]float64, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, elo FROM player_race`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64]float64)
	for rows.Next() {
		var id int64
		var elo float64
		if err := rows.Scan(&id, &elo); err != nil {
			return nil, err
		}
		out[id] = elo
	}
	return out, rows.Err()
}

// ListRatingMatches returns played matches for rating calculation across tournaments.
func (r *Season) ListRatingMatches(ctx context.Context, q DBTX, tournamentIDs []int64) ([]rating.Match, error) {
	if len(tournamentIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(tournamentIDs))
	args := make([]any, len(tournamentIDs))
	for i, id := range tournamentIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(`
		SELECT
			pra.id,
			prb.id,
			tr.score_a,
			tr.score_b,
			tr.played
		FROM tournament_result tr
		JOIN tournament_player tpa ON tpa.id = tr.tournament_player_a_id
		JOIN tournament_player tpb ON tpb.id = tr.tournament_player_b_id
		JOIN player_race pra ON pra.id = tpa.player_race_id
		JOIN player_race prb ON prb.id = tpb.player_race_id
		WHERE tr.tournament_id IN (%s)
			AND tr.played = 1
			AND tr.tournament_player_a_id IS NOT NULL
			AND tr.tournament_player_b_id IS NOT NULL
		ORDER BY tr.played_at ASC, tr.sort_order ASC, tr.id ASC
	`, joinPlaceholders(placeholders))

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list rating matches: %w", err)
	}
	defer rows.Close()

	out := make([]rating.Match, 0)
	for rows.Next() {
		var m rating.Match
		var scoreA, scoreB sql.NullInt64
		var played int
		if err := rows.Scan(&m.PlayerRaceA, &m.PlayerRaceB, &scoreA, &scoreB, &played); err != nil {
			return nil, err
		}
		m.Played = played != 0
		if scoreA.Valid {
			m.ScoreA = int(scoreA.Int64)
		}
		if scoreB.Valid {
			m.ScoreB = int(scoreB.Int64)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func joinPlaceholders(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

// NowISO returns current UTC time as ISO date (YYYY-MM-DD) for tournament window filtering.
func NowISO() string {
	return time.Now().UTC().Format("2006-01-02")
}
