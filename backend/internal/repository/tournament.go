package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
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

	groups, err := r.ListGroups(ctx, q, id)
	if err != nil {
		return nil, err
	}
	page.Groups = groups

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

// ReplaceRoster upserts enrollment by (tournament_id, player_race_id) so existing
// tournament_player ids stay stable (fantasy_player FKs). Rows not in entries are deleted.
func (r *Tournament) ReplaceRoster(ctx context.Context, q DBTX, tournamentID int64, entries []RosterEntry) error {
	rows, err := q.QueryContext(ctx, `
		SELECT id, player_race_id FROM tournament_player WHERE tournament_id = ?
	`, tournamentID)
	if err != nil {
		return fmt.Errorf("list tournament roster ids: %w", err)
	}
	existing := map[int64]int64{} // player_race_id -> tournament_player.id
	for rows.Next() {
		var id, raceID int64
		if err := rows.Scan(&id, &raceID); err != nil {
			rows.Close()
			return err
		}
		existing[raceID] = id
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	keep := make(map[int64]struct{}, len(entries))
	for _, e := range entries {
		keep[e.PlayerRaceID] = struct{}{}
		excluded := 0
		if e.Excluded {
			excluded = 1
		}
		if id, ok := existing[e.PlayerRaceID]; ok {
			if _, err := q.ExecContext(ctx, `
				UPDATE tournament_player
				SET player_alias_id = ?, excluded = ?
				WHERE id = ?
			`, e.PlayerAliasID, excluded, id); err != nil {
				return fmt.Errorf("update tournament_player: %w", err)
			}
			continue
		}
		if _, err := q.ExecContext(ctx, `
			INSERT INTO tournament_player (tournament_id, player_race_id, player_alias_id, excluded)
			VALUES (?, ?, ?, ?)
		`, tournamentID, e.PlayerRaceID, e.PlayerAliasID, excluded); err != nil {
			return fmt.Errorf("insert tournament_player: %w", err)
		}
	}

	for raceID, id := range existing {
		if _, ok := keep[raceID]; ok {
			continue
		}
		if _, err := q.ExecContext(ctx, `DELETE FROM tournament_player WHERE id = ?`, id); err != nil {
			return fmt.Errorf("delete removed tournament_player: %w", err)
		}
	}
	return nil
}

// GroupEntry is one group row plus roster member links (player profile URLs).
type GroupEntry struct {
	Name        string
	Phase       string
	SortOrder   int
	PlayerLinks []string
}

// ReplaceGroups deletes existing groups for a tournament and inserts entries.
// Player links that are not on the roster are skipped (not an error).
func (r *Tournament) ReplaceGroups(ctx context.Context, q DBTX, tournamentID int64, entries []GroupEntry) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM tournament_group WHERE tournament_id = ?`, tournamentID); err != nil {
		return fmt.Errorf("clear tournament groups: %w", err)
	}
	for _, e := range entries {
		res, err := q.ExecContext(ctx, `
			INSERT INTO tournament_group (tournament_id, name, phase, sort_order)
			VALUES (?, ?, ?, ?)
		`, tournamentID, e.Name, e.Phase, e.SortOrder)
		if err != nil {
			return fmt.Errorf("insert tournament_group: %w", err)
		}
		groupID, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("tournament_group last insert id: %w", err)
		}
		for i, link := range e.PlayerLinks {
			link = strings.TrimSpace(link)
			if link == "" {
				continue
			}
			tpID, err := r.tournamentPlayerIDByLink(ctx, q, tournamentID, link)
			if err != nil {
				return err
			}
			if tpID == 0 {
				debuglog.Printf("ReplaceGroups skip orphan link=%s tournamentID=%d", link, tournamentID)
				continue
			}
			if _, err := q.ExecContext(ctx, `
				INSERT INTO tournament_group_player (tournament_group_id, tournament_player_id, sort_order)
				VALUES (?, ?, ?)
			`, groupID, tpID, i); err != nil {
				return fmt.Errorf("insert tournament_group_player: %w", err)
			}
		}
	}
	return nil
}

// ListGroups returns groups with members ordered by sort_order.
func (r *Tournament) ListGroups(ctx context.Context, q DBTX, tournamentID int64) ([]model.TournamentGroup, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, name, phase, sort_order
		FROM tournament_group
		WHERE tournament_id = ?
		ORDER BY sort_order ASC, id ASC
	`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("list tournament groups: %w", err)
	}
	defer rows.Close()

	type groupRow struct {
		id        int64
		name      string
		phase     string
		sortOrder int
	}
	var groups []groupRow
	for rows.Next() {
		var g groupRow
		if err := rows.Scan(&g.id, &g.name, &g.phase, &g.sortOrder); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]model.TournamentGroup, 0, len(groups))
	for _, g := range groups {
		players, err := r.listGroupPlayers(ctx, q, g.id)
		if err != nil {
			return nil, err
		}
		out = append(out, model.TournamentGroup{
			Name:      g.name,
			Phase:     g.phase,
			SortOrder: g.sortOrder,
			Players:   players,
		})
	}
	return out, nil
}

func (r *Tournament) listGroupPlayers(ctx context.Context, q DBTX, groupID int64) ([]model.Participant, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT
			p.link,
			pa.name,
			pr.race,
			tp.excluded
		FROM tournament_group_player tgp
		JOIN tournament_player tp ON tp.id = tgp.tournament_player_id
		JOIN player_race pr ON pr.id = tp.player_race_id
		JOIN player_alias pa ON pa.id = tp.player_alias_id
		JOIN player p ON p.id = pr.player_id
		WHERE tgp.tournament_group_id = ?
		ORDER BY tgp.sort_order ASC, tgp.id ASC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list tournament group players: %w", err)
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

func (r *Tournament) tournamentPlayerIDByLink(ctx context.Context, q DBTX, tournamentID int64, link string) (int64, error) {
	var id int64
	err := q.QueryRowContext(ctx, `
		SELECT tp.id
		FROM tournament_player tp
		JOIN player_race pr ON pr.id = tp.player_race_id
		JOIN player p ON p.id = pr.player_id
		WHERE tp.tournament_id = ? AND p.link = ? COLLATE NOCASE
		LIMIT 1
	`, tournamentID, link).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("tournament player by link: %w", err)
	}
	return id, nil
}
