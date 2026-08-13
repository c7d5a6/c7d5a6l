package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// Fantasy stores fantasy leagues, players, and teams.
type Fantasy struct {
	db *sql.DB
}

func NewFantasy(db *sql.DB) *Fantasy {
	return &Fantasy{db: db}
}

func (r *Fantasy) DB() *sql.DB {
	return r.db
}

const leagueSelectCols = `
	fl.id, fl.tournament_id, t.link, t.name,
	fl.started, fl.finished, fl.max_players, fl.max_cost
`

const pointsEarnedExpr = `
	(COALESCE(fp.points_ro24,0)+COALESCE(fp.points_ro16,0)+COALESCE(fp.points_ro8,0)+COALESCE(fp.points_ro4,0)+COALESCE(fp.points_ro2,0))
`

// ListLeagues returns all fantasy leagues with tournament identity.
func (r *Fantasy) ListLeagues(ctx context.Context, q DBTX) ([]model.FantasyLeague, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT `+leagueSelectCols+`
		FROM fantasy_league fl
		JOIN tournament t ON t.id = fl.tournament_id
		ORDER BY t.name COLLATE NOCASE ASC, fl.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list fantasy leagues: %w", err)
	}
	defer rows.Close()

	out := make([]model.FantasyLeague, 0)
	for rows.Next() {
		league, err := scanLeague(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *league)
	}
	return out, rows.Err()
}

// GetLeagueByID returns a league or nil, nil when missing.
func (r *Fantasy) GetLeagueByID(ctx context.Context, q DBTX, id int64) (*model.FantasyLeague, error) {
	row := q.QueryRowContext(ctx, `
		SELECT `+leagueSelectCols+`
		FROM fantasy_league fl
		JOIN tournament t ON t.id = fl.tournament_id
		WHERE fl.id = ?
	`, id)
	league, err := scanLeague(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return league, err
}

// GetActiveLeague returns the newest unfinished league, else newest league overall.
func (r *Fantasy) GetActiveLeague(ctx context.Context, q DBTX) (*model.FantasyLeague, error) {
	league, err := scanLeague(q.QueryRowContext(ctx, `
		SELECT `+leagueSelectCols+`
		FROM fantasy_league fl
		JOIN tournament t ON t.id = fl.tournament_id
		WHERE fl.finished = 0
		ORDER BY fl.id DESC
		LIMIT 1
	`))
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil && league != nil {
		return league, nil
	}
	league, err = scanLeague(q.QueryRowContext(ctx, `
		SELECT `+leagueSelectCols+`
		FROM fantasy_league fl
		JOIN tournament t ON t.id = fl.tournament_id
		ORDER BY fl.id DESC
		LIMIT 1
	`))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return league, err
}

type leagueScanner interface {
	Scan(dest ...any) error
}

func scanLeague(row leagueScanner) (*model.FantasyLeague, error) {
	var (
		league   model.FantasyLeague
		name     sql.NullString
		started  int
		finished int
	)
	err := row.Scan(
		&league.ID,
		&league.TournamentID,
		&league.TournamentLink,
		&name,
		&started,
		&finished,
		&league.MaxPlayers,
		&league.MaxCost,
	)
	if err != nil {
		return nil, err
	}
	if name.Valid {
		v := name.String
		league.TournamentName = &v
	}
	league.Started = started != 0
	league.Finished = finished != 0
	return &league, nil
}

// GetLeagueIDByTournament returns fantasy league id for tournament, or 0 if none.
func (r *Fantasy) GetLeagueIDByTournament(ctx context.Context, q DBTX, tournamentID int64) (int64, error) {
	var id int64
	err := q.QueryRowContext(ctx, `
		SELECT id FROM fantasy_league WHERE tournament_id = ?
	`, tournamentID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("lookup fantasy league by tournament: %w", err)
	}
	return id, nil
}

// TournamentExists reports whether a tournament id exists.
func (r *Fantasy) TournamentExists(ctx context.Context, q DBTX, tournamentID int64) (bool, error) {
	var n int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM tournament WHERE id = ?`, tournamentID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("check tournament: %w", err)
	}
	return n > 0, nil
}

// ListUnusedTournaments returns tournaments without a fantasy league.
func (r *Fantasy) ListUnusedTournaments(ctx context.Context, q DBTX) ([]model.TournamentSummary, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT t.id, t.link, t.name
		FROM tournament t
		WHERE NOT EXISTS (
			SELECT 1 FROM fantasy_league fl WHERE fl.tournament_id = t.id
		)
		ORDER BY t.name COLLATE NOCASE ASC, t.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list unused tournaments: %w", err)
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

// CreateLeague inserts a fantasy league and returns its id.
func (r *Fantasy) CreateLeague(ctx context.Context, q DBTX, tournamentID int64, maxPlayers, maxCost int) (int64, error) {
	res, err := q.ExecContext(ctx, `
		INSERT INTO fantasy_league (tournament_id, started, finished, max_players, max_cost)
		VALUES (?, 0, 0, ?, ?)
	`, tournamentID, maxPlayers, maxCost)
	if err != nil {
		return 0, fmt.Errorf("insert fantasy league: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("fantasy league last insert id: %w", err)
	}
	return id, nil
}

// UpdateLeagueCaps updates max_players / max_cost.
func (r *Fantasy) UpdateLeagueCaps(ctx context.Context, q DBTX, id int64, maxPlayers, maxCost int) error {
	_, err := q.ExecContext(ctx, `
		UPDATE fantasy_league SET max_players = ?, max_cost = ? WHERE id = ?
	`, maxPlayers, maxCost, id)
	if err != nil {
		return fmt.Errorf("update fantasy league caps: %w", err)
	}
	return nil
}

// SetLeagueStarted marks a league as started.
func (r *Fantasy) SetLeagueStarted(ctx context.Context, q DBTX, id int64) error {
	_, err := q.ExecContext(ctx, `UPDATE fantasy_league SET started = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("start fantasy league: %w", err)
	}
	return nil
}

// SetLeagueFinished marks a league as finished.
func (r *Fantasy) SetLeagueFinished(ctx context.Context, q DBTX, id int64) error {
	_, err := q.ExecContext(ctx, `UPDATE fantasy_league SET finished = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("finish fantasy league: %w", err)
	}
	return nil
}

// RosterEloRow is a non-excluded tournament player with elo.
type RosterEloRow struct {
	TournamentPlayerID int64
	Name               string
	Link               sql.NullString
	Race               string
	Elo                float64
}

// ListRosterWithElo returns non-excluded tournament players joined to race elo.
func (r *Fantasy) ListRosterWithElo(ctx context.Context, q DBTX, tournamentID int64) ([]RosterEloRow, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT tp.id, pa.name, p.link, pr.race, pr.elo
		FROM tournament_player tp
		JOIN player_race pr ON pr.id = tp.player_race_id
		JOIN player_alias pa ON pa.id = tp.player_alias_id
		JOIN player p ON p.id = pr.player_id
		WHERE tp.tournament_id = ? AND tp.excluded = 0
		ORDER BY pr.elo DESC, pa.name COLLATE NOCASE ASC
	`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("list roster elo: %w", err)
	}
	defer rows.Close()

	out := make([]RosterEloRow, 0)
	for rows.Next() {
		var row RosterEloRow
		if err := rows.Scan(&row.TournamentPlayerID, &row.Name, &row.Link, &row.Race, &row.Elo); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// InsertFantasyPlayer inserts one fantasy player with cost.
func (r *Fantasy) InsertFantasyPlayer(ctx context.Context, q DBTX, leagueID, tournamentPlayerID int64, cost int) error {
	_, err := q.ExecContext(ctx, `
		INSERT INTO fantasy_player (fantasy_league_id, tournament_player_id, cost)
		VALUES (?, ?, ?)
	`, leagueID, tournamentPlayerID, cost)
	if err != nil {
		return fmt.Errorf("insert fantasy player: %w", err)
	}
	return nil
}

// SeedPlayersFromRoster inserts missing players at cost 0 (legacy/idempotent helper).
func (r *Fantasy) SeedPlayersFromRoster(ctx context.Context, q DBTX, leagueID, tournamentID int64) (int, error) {
	res, err := q.ExecContext(ctx, `
		INSERT OR IGNORE INTO fantasy_player (fantasy_league_id, tournament_player_id, cost)
		SELECT ?, tp.id, 0
		FROM tournament_player tp
		WHERE tp.tournament_id = ? AND tp.excluded = 0
	`, leagueID, tournamentID)
	if err != nil {
		return 0, fmt.Errorf("seed fantasy players: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// PlayerSort is the order for ListPlayers.
type PlayerSort string

const (
	PlayerSortCost   PlayerSort = "cost"
	PlayerSortPoints PlayerSort = "points"
	PlayerSortElo    PlayerSort = "elo"
)

// ListPlayers returns fantasy players for a league, sorted by cost, points, or elo (desc).
func (r *Fantasy) ListPlayers(ctx context.Context, q DBTX, leagueID int64, sort PlayerSort) ([]model.FantasyPlayerRow, error) {
	order := "pr.elo DESC, pa.name COLLATE NOCASE ASC"
	switch sort {
	case PlayerSortPoints:
		order = pointsEarnedExpr + ` DESC, pr.elo DESC, pa.name COLLATE NOCASE ASC`
	case PlayerSortCost:
		order = "fp.cost DESC, pr.elo DESC, pa.name COLLATE NOCASE ASC"
	}
	rows, err := q.QueryContext(ctx, `
		SELECT
			fp.id,
			fp.fantasy_league_id,
			fp.tournament_player_id,
			pa.name,
			p.link,
			pr.race,
			fp.cost,
			fp.points_ro24,
			fp.points_ro16,
			fp.points_ro8,
			fp.points_ro4,
			fp.points_ro2,
			`+pointsEarnedExpr+`,
			fp.defeated,
			fp.is_winner,
			pr.elo
		FROM fantasy_player fp
		JOIN tournament_player tp ON tp.id = fp.tournament_player_id
		JOIN player_race pr ON pr.id = tp.player_race_id
		JOIN player_alias pa ON pa.id = tp.player_alias_id
		JOIN player p ON p.id = pr.player_id
		WHERE fp.fantasy_league_id = ?
		ORDER BY `+order+`
	`, leagueID)
	if err != nil {
		return nil, fmt.Errorf("list fantasy players: %w", err)
	}
	defer rows.Close()

	out := make([]model.FantasyPlayerRow, 0)
	for rows.Next() {
		row, err := scanFantasyPlayer(rows, true)
		if err != nil {
			return nil, err
		}
		out = append(out, *row)
	}
	return out, rows.Err()
}

// GetPlayerByID returns a fantasy player in a league, or nil.
func (r *Fantasy) GetPlayerByID(ctx context.Context, q DBTX, leagueID, playerID int64) (*model.FantasyPlayerRow, error) {
	row := q.QueryRowContext(ctx, `
		SELECT
			fp.id,
			fp.fantasy_league_id,
			fp.tournament_player_id,
			pa.name,
			p.link,
			pr.race,
			fp.cost,
			fp.points_ro24,
			fp.points_ro16,
			fp.points_ro8,
			fp.points_ro4,
			fp.points_ro2,
			`+pointsEarnedExpr+`,
			fp.defeated,
			fp.is_winner,
			pr.elo
		FROM fantasy_player fp
		JOIN tournament_player tp ON tp.id = fp.tournament_player_id
		JOIN player_race pr ON pr.id = tp.player_race_id
		JOIN player_alias pa ON pa.id = tp.player_alias_id
		JOIN player p ON p.id = pr.player_id
		WHERE fp.fantasy_league_id = ? AND fp.id = ?
	`, leagueID, playerID)
	p, err := scanFantasyPlayer(row, true)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

type playerScanner interface {
	Scan(dest ...any) error
}

func scanFantasyPlayer(row playerScanner, withElo bool) (*model.FantasyPlayerRow, error) {
	var (
		p        model.FantasyPlayerRow
		name     string
		link     sql.NullString
		race     string
		ro24     sql.NullInt64
		ro16     sql.NullInt64
		ro8      sql.NullInt64
		ro4      sql.NullInt64
		ro2      sql.NullInt64
		defeated int
		winner   int
		elo      float64
	)
	err := row.Scan(
		&p.ID,
		&p.FantasyLeagueID,
		&p.TournamentPlayerID,
		&name,
		&link,
		&race,
		&p.Cost,
		&ro24, &ro16, &ro8, &ro4, &ro2,
		&p.PointsEarned,
		&defeated,
		&winner,
		&elo,
	)
	if err != nil {
		return nil, err
	}
	n := name
	p.Name = &n
	if link.Valid && link.String != "" {
		l := link.String
		p.Link = &l
	}
	if race != "" {
		rc := race
		p.Race = &rc
	}
	p.PointsRo24 = nullIntPtr(ro24)
	p.PointsRo16 = nullIntPtr(ro16)
	p.PointsRo8 = nullIntPtr(ro8)
	p.PointsRo4 = nullIntPtr(ro4)
	p.PointsRo2 = nullIntPtr(ro2)
	p.Defeated = defeated != 0
	p.IsWinner = winner != 0
	if withElo {
		e := elo
		p.Elo = &e
	}
	return &p, nil
}

func nullIntPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

// UpdatePlayer patches cost, stage points, and flags.
func (r *Fantasy) UpdatePlayer(ctx context.Context, q DBTX, p model.FantasyPlayerRow) error {
	_, err := q.ExecContext(ctx, `
		UPDATE fantasy_player SET
			cost = ?,
			points_ro24 = ?,
			points_ro16 = ?,
			points_ro8 = ?,
			points_ro4 = ?,
			points_ro2 = ?,
			defeated = ?,
			is_winner = ?
		WHERE id = ? AND fantasy_league_id = ?
	`,
		p.Cost,
		nullableInt(p.PointsRo24),
		nullableInt(p.PointsRo16),
		nullableInt(p.PointsRo8),
		nullableInt(p.PointsRo4),
		nullableInt(p.PointsRo2),
		boolToInt(p.Defeated),
		boolToInt(p.IsWinner),
		p.ID,
		p.FantasyLeagueID,
	)
	if err != nil {
		return fmt.Errorf("update fantasy player: %w", err)
	}
	return nil
}

// ClearWinnersInLeague clears is_winner for all players in a league.
func (r *Fantasy) ClearWinnersInLeague(ctx context.Context, q DBTX, leagueID int64) error {
	_, err := q.ExecContext(ctx, `
		UPDATE fantasy_player SET is_winner = 0 WHERE fantasy_league_id = ?
	`, leagueID)
	if err != nil {
		return fmt.Errorf("clear winners: %w", err)
	}
	return nil
}

func nullableInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// ListTeams returns fantasy teams with members for a league.
func (r *Fantasy) ListTeams(ctx context.Context, q DBTX, leagueID int64) ([]model.FantasyTeamRow, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT ft.id, ft.fantasy_league_id, ft.user_id, u.alias
		FROM fantasy_team ft
		JOIN user u ON u.id = ft.user_id
		WHERE ft.fantasy_league_id = ?
		ORDER BY u.alias COLLATE NOCASE ASC
	`, leagueID)
	if err != nil {
		return nil, fmt.Errorf("list fantasy teams: %w", err)
	}
	defer rows.Close()

	out := make([]model.FantasyTeamRow, 0)
	for rows.Next() {
		var team model.FantasyTeamRow
		if err := rows.Scan(&team.ID, &team.FantasyLeagueID, &team.UserID, &team.UserAlias); err != nil {
			return nil, err
		}
		team.Members = []model.FantasyTeamMemberRow{}
		out = append(out, team)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		members, err := r.listTeamMembers(ctx, q, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Members = members
	}
	return out, nil
}

// GetTeamByUser returns the user's team in a league, or nil.
func (r *Fantasy) GetTeamByUser(ctx context.Context, q DBTX, leagueID, userID int64) (*model.FantasyTeamRow, error) {
	var team model.FantasyTeamRow
	err := q.QueryRowContext(ctx, `
		SELECT ft.id, ft.fantasy_league_id, ft.user_id, u.alias
		FROM fantasy_team ft
		JOIN user u ON u.id = ft.user_id
		WHERE ft.fantasy_league_id = ? AND ft.user_id = ?
	`, leagueID, userID).Scan(&team.ID, &team.FantasyLeagueID, &team.UserID, &team.UserAlias)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get fantasy team: %w", err)
	}
	members, err := r.listTeamMembers(ctx, q, team.ID)
	if err != nil {
		return nil, err
	}
	team.Members = members
	return &team, nil
}

// GetTeamByID returns a team by id in a league, or nil.
func (r *Fantasy) GetTeamByID(ctx context.Context, q DBTX, leagueID, teamID int64) (*model.FantasyTeamRow, error) {
	var team model.FantasyTeamRow
	err := q.QueryRowContext(ctx, `
		SELECT ft.id, ft.fantasy_league_id, ft.user_id, u.alias
		FROM fantasy_team ft
		JOIN user u ON u.id = ft.user_id
		WHERE ft.fantasy_league_id = ? AND ft.id = ?
	`, leagueID, teamID).Scan(&team.ID, &team.FantasyLeagueID, &team.UserID, &team.UserAlias)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get fantasy team by id: %w", err)
	}
	members, err := r.listTeamMembers(ctx, q, team.ID)
	if err != nil {
		return nil, err
	}
	team.Members = members
	return &team, nil
}

func (r *Fantasy) listTeamMembers(ctx context.Context, q DBTX, teamID int64) ([]model.FantasyTeamMemberRow, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT
			fp.id,
			pa.name,
			p.link,
			pr.race,
			fp.cost,
			`+pointsEarnedExpr+`,
			fp.defeated,
			fp.is_winner,
			pr.elo
		FROM fantasy_team_member ftm
		JOIN fantasy_player fp ON fp.id = ftm.fantasy_player_id
		JOIN tournament_player tp ON tp.id = fp.tournament_player_id
		JOIN player_race pr ON pr.id = tp.player_race_id
		JOIN player_alias pa ON pa.id = tp.player_alias_id
		JOIN player p ON p.id = pr.player_id
		WHERE ftm.fantasy_team_id = ?
		ORDER BY pr.elo DESC, pa.name COLLATE NOCASE ASC
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("list fantasy team members: %w", err)
	}
	defer rows.Close()

	out := make([]model.FantasyTeamMemberRow, 0)
	for rows.Next() {
		var (
			m        model.FantasyTeamMemberRow
			name     string
			link     sql.NullString
			race     string
			defeated int
			winner   int
		)
		if err := rows.Scan(
			&m.FantasyPlayerID, &name, &link, &race, &m.Cost, &m.PointsEarned,
			&defeated, &winner, &m.Elo,
		); err != nil {
			return nil, err
		}
		n := name
		m.Name = &n
		if link.Valid && link.String != "" {
			l := link.String
			m.Link = &l
		}
		if race != "" {
			rc := race
			m.Race = &rc
		}
		m.Defeated = defeated != 0
		m.IsWinner = winner != 0
		out = append(out, m)
	}
	return out, rows.Err()
}

// CreateTeam inserts a fantasy team and returns its id.
func (r *Fantasy) CreateTeam(ctx context.Context, q DBTX, leagueID, userID int64) (int64, error) {
	res, err := q.ExecContext(ctx, `
		INSERT INTO fantasy_team (fantasy_league_id, user_id) VALUES (?, ?)
	`, leagueID, userID)
	if err != nil {
		return 0, fmt.Errorf("insert fantasy team: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("fantasy team last insert id: %w", err)
	}
	return id, nil
}

// ReplaceTeamMembers deletes existing members and inserts fantasyPlayerIDs.
func (r *Fantasy) ReplaceTeamMembers(ctx context.Context, q DBTX, teamID int64, fantasyPlayerIDs []int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM fantasy_team_member WHERE fantasy_team_id = ?`, teamID); err != nil {
		return fmt.Errorf("clear team members: %w", err)
	}
	for _, pid := range fantasyPlayerIDs {
		if _, err := q.ExecContext(ctx, `
			INSERT INTO fantasy_team_member (fantasy_team_id, fantasy_player_id) VALUES (?, ?)
		`, teamID, pid); err != nil {
			return fmt.Errorf("insert team member: %w", err)
		}
	}
	return nil
}

// DeleteTeam removes a team and its members.
func (r *Fantasy) DeleteTeam(ctx context.Context, q DBTX, leagueID, teamID int64) error {
	if _, err := q.ExecContext(ctx, `
		DELETE FROM fantasy_team_member
		WHERE fantasy_team_id IN (
			SELECT id FROM fantasy_team WHERE id = ? AND fantasy_league_id = ?
		)
	`, teamID, leagueID); err != nil {
		return fmt.Errorf("delete team members: %w", err)
	}
	res, err := q.ExecContext(ctx, `
		DELETE FROM fantasy_team WHERE id = ? AND fantasy_league_id = ?
	`, teamID, leagueID)
	if err != nil {
		return fmt.Errorf("delete fantasy team: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// PlayersByIDs returns fantasy players in a league matching the given ids.
func (r *Fantasy) PlayersByIDs(ctx context.Context, q DBTX, leagueID int64, ids []int64) ([]model.FantasyPlayerRow, error) {
	if len(ids) == 0 {
		return []model.FantasyPlayerRow{}, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, leagueID)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	rows, err := q.QueryContext(ctx, `
		SELECT
			fp.id,
			fp.fantasy_league_id,
			fp.tournament_player_id,
			pa.name,
			p.link,
			pr.race,
			fp.cost,
			fp.points_ro24,
			fp.points_ro16,
			fp.points_ro8,
			fp.points_ro4,
			fp.points_ro2,
			`+pointsEarnedExpr+`,
			fp.defeated,
			fp.is_winner,
			pr.elo
		FROM fantasy_player fp
		JOIN tournament_player tp ON tp.id = fp.tournament_player_id
		JOIN player_race pr ON pr.id = tp.player_race_id
		JOIN player_alias pa ON pa.id = tp.player_alias_id
		JOIN player p ON p.id = pr.player_id
		WHERE fp.fantasy_league_id = ? AND fp.id IN (`+strings.Join(placeholders, ",")+`)
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("players by ids: %w", err)
	}
	defer rows.Close()

	out := make([]model.FantasyPlayerRow, 0, len(ids))
	for rows.Next() {
		row, err := scanFantasyPlayer(rows, true)
		if err != nil {
			return nil, err
		}
		out = append(out, *row)
	}
	return out, rows.Err()
}

// UserExists reports whether a user id exists.
func (r *Fantasy) UserExists(ctx context.Context, q DBTX, userID int64) (bool, error) {
	var n int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM user WHERE id = ?`, userID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("check user: %w", err)
	}
	return n > 0, nil
}

// ParsePlayerSort maps query values to PlayerSort; unknown defaults to cost.
func ParsePlayerSort(raw string) PlayerSort {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "points", "points_earned", "point":
		return PlayerSortPoints
	case "cost":
		return PlayerSortCost
	default:
		return PlayerSortElo
	}
}
