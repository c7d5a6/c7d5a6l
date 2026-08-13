package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// Tournament stores tournaments and roster enrollment.
type Tournament struct {
	db *sql.DB
}

func NewTournament(db *sql.DB) *Tournament {
	return &Tournament{db: db}
}

func (r *Tournament) DB() *sql.DB {
	return r.db
}

// StoredTournament is the DB view used for sync compare (no results).
type StoredTournament struct {
	Page         model.TournamentPage
	Participants []model.Participant
}

// GetByLink loads a tournament and its roster. Returns nil, nil when missing.
func (r *Tournament) GetByLink(ctx context.Context, q DBTX, link string) (*StoredTournament, error) {
	var (
		id          int64
		name        sql.NullString
		startDate   sql.NullString
		endDate     sql.NullString
		tier        sql.NullString
		playerCount sql.NullInt64
		finished    int
	)
	err := q.QueryRowContext(ctx, `
		SELECT id, name, start_date, end_date, tier, player_count, finished
		FROM tournament
		WHERE link = ? COLLATE NOCASE
	`, link).Scan(&id, &name, &startDate, &endDate, &tier, &playerCount, &finished)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tournament by link: %w", err)
	}

	page := model.NewTournamentPage(link)
	if name.Valid {
		v := name.String
		page.Name = &v
	}
	if startDate.Valid {
		v := startDate.String
		page.StartDate = &v
	}
	if endDate.Valid {
		v := endDate.String
		page.EndDate = &v
	}
	if tier.Valid {
		v := tier.String
		page.LiquipediaTier = &v
	}
	if playerCount.Valid {
		n := int(playerCount.Int64)
		page.PlayerCounts = &model.PlayerCounts{Total: &n}
	}
	fin := finished != 0
	page.Finished = &fin

	participants, err := r.listRoster(ctx, q, id)
	if err != nil {
		return nil, err
	}
	page.Participants = participants

	return &StoredTournament{Page: page, Participants: participants}, nil
}

func (r *Tournament) listRoster(ctx context.Context, q DBTX, tournamentID int64) ([]model.Participant, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT
			p.link,
			pa.name,
			pr.race,
			tp.excluded
		FROM tournament_player tp
		JOIN player_race pr ON pr.id = tp.player_race_id
		JOIN player_alias pa ON pa.id = tp.player_alias_id
		JOIN player p ON p.id = pr.player_id
		WHERE tp.tournament_id = ?
		ORDER BY pa.name COLLATE NOCASE
	`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("list tournament roster: %w", err)
	}
	defer rows.Close()

	var out []model.Participant
	for rows.Next() {
		var (
			link     sql.NullString
			name     string
			race     string
			excluded int
		)
		if err := rows.Scan(&link, &name, &race, &excluded); err != nil {
			return nil, err
		}
		p := model.Participant{Excluded: excluded != 0}
		n := name
		p.Name = &n
		if link.Valid && link.String != "" {
			l := link.String
			p.Link = &l
		}
		if race != "" {
			r := race
			p.Race = &r
		}
		out = append(out, p)
	}
	if out == nil {
		out = []model.Participant{}
	}
	return out, rows.Err()
}

// TournamentSummary is a lightweight tournament row for pickers.
type TournamentSummary struct {
	ID   int64   `json:"id"`
	Link string  `json:"link"`
	Name *string `json:"name"`
}

// ListSummaries returns all tournaments ordered by name.
func (r *Tournament) ListSummaries(ctx context.Context, q DBTX) ([]model.TournamentSummary, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, link, name FROM tournament ORDER BY name COLLATE NOCASE ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list tournaments: %w", err)
	}
	defer rows.Close()

	out := make([]model.TournamentSummary, 0)
	for rows.Next() {
		var (
			s    model.TournamentSummary
			name sql.NullString
		)
		if err := rows.Scan(&s.ID, &s.Link, &name); err != nil {
			return nil, err
		}
		if name.Valid {
			v := name.String
			s.Name = &v
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Upsert inserts or updates a tournament by link and returns its id.
func (r *Tournament) Upsert(ctx context.Context, q DBTX, page model.TournamentPage) (int64, error) {
	if page.Link == "" {
		return 0, fmt.Errorf("tournament link is required")
	}
	var playerCount any
	if page.PlayerCounts != nil && page.PlayerCounts.Total != nil {
		playerCount = *page.PlayerCounts.Total
	}
	finished := 0
	if page.Finished != nil && *page.Finished {
		finished = 1
	}

	var id int64
	err := q.QueryRowContext(ctx, `SELECT id FROM tournament WHERE link = ? COLLATE NOCASE`, page.Link).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		res, err := q.ExecContext(ctx, `
			INSERT INTO tournament (link, name, start_date, end_date, tier, player_count, finished)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, page.Link, nullableText(page.Name), nullableText(page.StartDate), nullableText(page.EndDate),
			nullableText(page.LiquipediaTier), playerCount, finished)
		if err != nil {
			return 0, fmt.Errorf("insert tournament: %w", err)
		}
		id, err = res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("tournament last insert id: %w", err)
		}
	case err != nil:
		return 0, fmt.Errorf("lookup tournament: %w", err)
	default:
		if _, err := q.ExecContext(ctx, `
			UPDATE tournament
			SET name = ?, start_date = ?, end_date = ?, tier = ?, player_count = ?, finished = ?
			WHERE id = ?
		`, nullableText(page.Name), nullableText(page.StartDate), nullableText(page.EndDate),
			nullableText(page.LiquipediaTier), playerCount, finished, id); err != nil {
			return 0, fmt.Errorf("update tournament: %w", err)
		}
	}
	return id, nil
}

// RosterEntry is one enrollment row to write.
type RosterEntry struct {
	PlayerRaceID  int64
	PlayerAliasID int64
	Excluded      bool
}

// ReplaceRoster deletes existing enrollment and inserts entries.
func (r *Tournament) ReplaceRoster(ctx context.Context, q DBTX, tournamentID int64, entries []RosterEntry) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM tournament_player WHERE tournament_id = ?`, tournamentID); err != nil {
		return fmt.Errorf("clear tournament roster: %w", err)
	}
	for _, e := range entries {
		excluded := 0
		if e.Excluded {
			excluded = 1
		}
		if _, err := q.ExecContext(ctx, `
			INSERT INTO tournament_player (tournament_id, player_race_id, player_alias_id, excluded)
			VALUES (?, ?, ?, ?)
		`, tournamentID, e.PlayerRaceID, e.PlayerAliasID, excluded); err != nil {
			return fmt.Errorf("insert tournament_player: %w", err)
		}
	}
	return nil
}
