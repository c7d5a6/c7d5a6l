package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// TournamentQueue persists discovered Liquipedia tournaments.
type TournamentQueue struct{}

func NewTournamentQueue() *TournamentQueue { return &TournamentQueue{} }

// QueueRow is a tournament_queue record.
type QueueRow struct {
	ID           int64
	Link         string
	Name         *string
	Disabled     bool
	TournamentID *int64
}

// GetByID loads a queue row. Returns nil, nil when missing.
func (r *TournamentQueue) GetByID(ctx context.Context, q DBTX, id int64) (*QueueRow, error) {
	var (
		row          QueueRow
		name         sql.NullString
		disabled     int
		tournamentID sql.NullInt64
	)
	err := q.QueryRowContext(ctx, `
		SELECT id, link, name, disabled, tournament_id
		FROM tournament_queue
		WHERE id = ?
	`, id).Scan(&row.ID, &row.Link, &name, &disabled, &tournamentID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tournament queue: %w", err)
	}
	if name.Valid {
		v := name.String
		row.Name = &v
	}
	row.Disabled = disabled != 0
	if tournamentID.Valid {
		v := tournamentID.Int64
		row.TournamentID = &v
	}
	return &row, nil
}

// UpsertListings inserts or refreshes listing rows. Does not change disabled.
func (r *TournamentQueue) UpsertListings(ctx context.Context, q DBTX, listings []model.TournamentListing) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for _, item := range listings {
		link := strings.TrimSpace(item.Link)
		if link == "" {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if _, err := q.ExecContext(ctx, `
			INSERT INTO tournament_queue (
				link, name, start_date, end_date, tier, section, disabled,
				tournament_id, created_at, updated_at, last_seen_at
			) VALUES (
				?, ?, ?, ?, ?, ?, 0,
				(SELECT id FROM tournament WHERE link = ? COLLATE NOCASE),
				?, ?, ?
			)
			ON CONFLICT(link) DO UPDATE SET
				name = excluded.name,
				start_date = excluded.start_date,
				end_date = excluded.end_date,
				tier = excluded.tier,
				section = excluded.section,
				last_seen_at = excluded.last_seen_at,
				updated_at = excluded.updated_at,
				tournament_id = (
					SELECT id FROM tournament t WHERE t.link = tournament_queue.link COLLATE NOCASE
				)
		`, link, nullableText(&name), nullableText(item.StartDate), nullableText(item.EndDate),
			nullableText(item.Tier), nullableText(item.Section),
			link, now, now, now); err != nil {
			return fmt.Errorf("upsert tournament queue %s: %w", link, err)
		}
	}
	return nil
}

// AttachByLink sets tournament_id when a queue row exists for the link.
func (r *TournamentQueue) AttachByLink(ctx context.Context, q DBTX, link string, tournamentID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := q.ExecContext(ctx, `
		UPDATE tournament_queue
		SET tournament_id = ?, disabled = 0, updated_at = ?
		WHERE link = ? COLLATE NOCASE
	`, tournamentID, now, link); err != nil {
		return fmt.Errorf("attach tournament queue: %w", err)
	}
	return nil
}

// SetDisabled marks a queue row ignored (or not). Returns false when missing.
func (r *TournamentQueue) SetDisabled(ctx context.Context, q DBTX, id int64, disabled bool) (bool, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	flag := 0
	if disabled {
		flag = 1
	}
	res, err := q.ExecContext(ctx, `
		UPDATE tournament_queue SET disabled = ?, updated_at = ? WHERE id = ?
	`, flag, now, id)
	if err != nil {
		return false, fmt.Errorf("set tournament queue disabled: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

const adminListCTE = `
WITH items AS (
	SELECT
		q.id AS queue_id,
		t.id AS tournament_id,
		COALESCE(t.link, q.link) AS link,
		COALESCE(t.name, q.name) AS name,
		COALESCE(t.start_date, q.start_date) AS start_date,
		COALESCE(t.end_date, q.end_date) AS end_date,
		COALESCE(t.tier, q.tier) AS tier,
		q.section AS section,
		COALESCE(q.disabled, 0) AS disabled,
		t.finished AS finished,
		fl.id AS fantasy_league_id
	FROM tournament_queue q
	LEFT JOIN tournament t ON t.link = q.link COLLATE NOCASE
	LEFT JOIN fantasy_league fl ON fl.tournament_id = t.id
	UNION ALL
	SELECT
		NULL,
		t.id,
		t.link,
		t.name,
		t.start_date,
		t.end_date,
		t.tier,
		NULL,
		0,
		t.finished,
		fl.id
	FROM tournament t
	LEFT JOIN tournament_queue q ON q.link = t.link COLLATE NOCASE
	LEFT JOIN fantasy_league fl ON fl.tournament_id = t.id
	WHERE q.id IS NULL
)
`

func adminFilterSQL(filter string) (string, error) {
	switch filter {
	case "", model.AdminFilterAll:
		return "1=1", nil
	case model.AdminFilterQueue:
		return "queue_id IS NOT NULL AND disabled = 0 AND tournament_id IS NULL", nil
	case model.AdminFilterOngoing:
		return "tournament_id IS NOT NULL AND finished = 0 AND disabled = 0", nil
	case model.AdminFilterParsed:
		return "tournament_id IS NOT NULL AND disabled = 0", nil
	case model.AdminFilterFinished:
		return "tournament_id IS NOT NULL AND finished = 1", nil
	case model.AdminFilterIgnored:
		return "disabled = 1", nil
	case model.AdminFilterFantasy:
		return "fantasy_league_id IS NOT NULL", nil
	default:
		return "", fmt.Errorf("unknown tournament filter %q", filter)
	}
}

// ListAdmin returns a paginated admin tournament list for the given filter.
func (r *TournamentQueue) ListAdmin(ctx context.Context, q DBTX, filter string, page, pageSize int) ([]model.AdminTournament, int, error) {
	where, err := adminFilterSQL(filter)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int
	if err := q.QueryRowContext(ctx, adminListCTE+`SELECT COUNT(*) FROM items WHERE `+where).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count admin tournaments: %w", err)
	}

	offset := (page - 1) * pageSize
	rows, err := q.QueryContext(ctx, adminListCTE+`
		SELECT queue_id, tournament_id, link, name, start_date, end_date, tier, section,
		       disabled, finished, fantasy_league_id
		FROM items
		WHERE `+where+`
		ORDER BY (start_date IS NULL) ASC, start_date DESC, name COLLATE NOCASE ASC,
		         COALESCE(queue_id, tournament_id) ASC
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin tournaments: %w", err)
	}
	defer rows.Close()

	out := make([]model.AdminTournament, 0)
	for rows.Next() {
		item, err := scanAdminTournament(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func scanAdminTournament(rows *sql.Rows) (model.AdminTournament, error) {
	var (
		queueID, tournamentID, fantasyID sql.NullInt64
		link                             string
		name, startDate, endDate, tier   sql.NullString
		section                          sql.NullString
		disabled                         int
		finished                         sql.NullInt64
	)
	if err := rows.Scan(
		&queueID, &tournamentID, &link, &name, &startDate, &endDate, &tier, &section,
		&disabled, &finished, &fantasyID,
	); err != nil {
		return model.AdminTournament{}, err
	}
	item := model.AdminTournament{
		Link:     link,
		Disabled: disabled != 0,
		Flags:    []string{},
	}
	if queueID.Valid {
		v := queueID.Int64
		item.QueueID = &v
	}
	if tournamentID.Valid {
		v := tournamentID.Int64
		item.TournamentID = &v
	}
	if name.Valid {
		v := name.String
		item.Name = &v
	}
	if startDate.Valid {
		v := startDate.String
		item.StartDate = &v
	}
	if endDate.Valid {
		v := endDate.String
		item.EndDate = &v
	}
	if tier.Valid {
		v := tier.String
		item.LiquipediaTier = &v
	}
	if section.Valid {
		v := section.String
		item.Section = &v
	}
	if finished.Valid {
		v := finished.Int64 != 0
		item.Finished = &v
	}
	if fantasyID.Valid {
		v := fantasyID.Int64
		item.FantasyLeagueID = &v
	}
	item.Flags = adminFlags(item)
	return item, nil
}

func adminFlags(item model.AdminTournament) []string {
	flags := make([]string, 0, 4)
	if item.Disabled {
		flags = append(flags, "ignored")
	}
	if item.TournamentID != nil {
		flags = append(flags, "parsed")
		if item.Finished != nil && *item.Finished {
			flags = append(flags, "finished")
		} else {
			flags = append(flags, "ongoing")
		}
	} else if !item.Disabled {
		flags = append(flags, "queue")
	}
	if item.FantasyLeagueID != nil {
		flags = append(flags, "fantasy")
	}
	return flags
}
