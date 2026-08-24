package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

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

	results, err := r.ListResults(ctx, q, id)
	if err != nil {
		return nil, err
	}
	page.Results = results

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

// GroupPlayerEntry is one group member (roster link + winner flag).
type GroupPlayerEntry struct {
	Link     string
	IsWinner bool
}

// GroupEntry is one group row plus roster members.
type GroupEntry struct {
	Name      string
	Phase     string
	SortOrder int
	Players   []GroupPlayerEntry
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
		for i, gp := range e.Players {
			link := strings.TrimSpace(gp.Link)
			if link == "" {
				continue
			}
			tpID, err := r.TournamentPlayerIDByLink(ctx, q, tournamentID, link)
			if err != nil {
				return err
			}
			if tpID == 0 {
				debuglog.Printf("ReplaceGroups skip orphan link=%s tournamentID=%d", link, tournamentID)
				continue
			}
			winner := 0
			if gp.IsWinner {
				winner = 1
			}
			if _, err := q.ExecContext(ctx, `
				INSERT INTO tournament_group_player (tournament_group_id, tournament_player_id, sort_order, is_winner)
				VALUES (?, ?, ?, ?)
			`, groupID, tpID, i, winner); err != nil {
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
			ID:        g.id,
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
			tp.excluded,
			tgp.is_winner
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
			winner   int
		)
		if err := rows.Scan(&link, &name, &race, &excluded, &winner); err != nil {
			return nil, err
		}
		p := model.Participant{Excluded: excluded != 0, IsWinner: winner != 0}
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

// TournamentPlayerIDByLink resolves a roster row id by player profile URL.
func (r *Tournament) TournamentPlayerIDByLink(ctx context.Context, q DBTX, tournamentID int64, link string) (int64, error) {
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

// ResultEntry is one match to upsert or insert (TBD).
// Zero TournamentPlayerAID/BID means TBD (NULL in DB); player_lo/hi use 0 for that side.
type ResultEntry struct {
	Phase               string
	Round               string
	TournamentGroupID   *int64
	TournamentPlayerAID int64
	TournamentPlayerBID int64
	// PairIndex is the 0-based nth meeting of this unordered pair in parse order.
	PairIndex int
	ScoreA    *int
	ScoreB    *int
	Played    bool
	PlayedAt  *string
	SortOrder int
}

func resultPlayerIDs(a, b int64) (lo, hi int64, idA, idB any) {
	lo, hi = a, b
	if lo > hi {
		lo, hi = hi, lo
	}
	if a != 0 {
		idA = a
	}
	if b != 0 {
		idB = b
	}
	return lo, hi, idA, idB
}

func resultScoresPlayedAt(e ResultEntry) (scoreA, scoreB, groupID, playedAt any, played int) {
	if e.ScoreA != nil {
		scoreA = *e.ScoreA
	}
	if e.ScoreB != nil {
		scoreB = *e.ScoreB
	}
	if e.TournamentGroupID != nil {
		groupID = *e.TournamentGroupID
	}
	if e.PlayedAt != nil && strings.TrimSpace(*e.PlayedAt) != "" {
		playedAt = strings.TrimSpace(*e.PlayedAt)
	}
	if e.Played {
		played = 1
	}
	return scoreA, scoreB, groupID, playedAt, played
}

// UpsertResults inserts or updates two-player matches by unordered pair + pair_index. Never deletes.
func (r *Tournament) UpsertResults(ctx context.Context, q DBTX, tournamentID int64, entries []ResultEntry) error {
	for _, e := range entries {
		if e.TournamentPlayerAID == 0 || e.TournamentPlayerBID == 0 {
			continue
		}
		lo, hi, idA, idB := resultPlayerIDs(e.TournamentPlayerAID, e.TournamentPlayerBID)
		scoreA, scoreB, groupID, playedAt, played := resultScoresPlayedAt(e)
		if _, err := q.ExecContext(ctx, `
			INSERT INTO tournament_result (
				tournament_id, tournament_group_id, phase, round,
				tournament_player_a_id, tournament_player_b_id, player_lo, player_hi, pair_index,
				score_a, score_b, played, played_at, sort_order
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (tournament_id, player_lo, player_hi, pair_index) DO UPDATE SET
				tournament_group_id = excluded.tournament_group_id,
				phase = excluded.phase,
				round = excluded.round,
				tournament_player_a_id = excluded.tournament_player_a_id,
				tournament_player_b_id = excluded.tournament_player_b_id,
				score_a = excluded.score_a,
				score_b = excluded.score_b,
				played = excluded.played,
				played_at = excluded.played_at,
				sort_order = excluded.sort_order
		`, tournamentID, groupID, e.Phase, e.Round,
			idA, idB, lo, hi, e.PairIndex,
			scoreA, scoreB, played, playedAt, e.SortOrder); err != nil {
			return fmt.Errorf("upsert tournament_result: %w", err)
		}
	}
	return nil
}

// DeleteTBDResults removes matches that have a missing (TBD) player side.
func (r *Tournament) DeleteTBDResults(ctx context.Context, q DBTX, tournamentID int64) error {
	if _, err := q.ExecContext(ctx, `
		DELETE FROM tournament_result
		WHERE tournament_id = ?
		  AND (tournament_player_a_id IS NULL OR tournament_player_b_id IS NULL)
	`, tournamentID); err != nil {
		return fmt.Errorf("delete TBD tournament_result: %w", err)
	}
	return nil
}

// InsertTBDResults inserts one-sided (real + TBD) matches. Caller should DeleteTBDResults first.
func (r *Tournament) InsertTBDResults(ctx context.Context, q DBTX, tournamentID int64, entries []ResultEntry) error {
	for _, e := range entries {
		aZero := e.TournamentPlayerAID == 0
		bZero := e.TournamentPlayerBID == 0
		if aZero == bZero {
			// Need exactly one real player.
			continue
		}
		lo, hi, idA, idB := resultPlayerIDs(e.TournamentPlayerAID, e.TournamentPlayerBID)
		scoreA, scoreB, groupID, playedAt, played := resultScoresPlayedAt(e)
		if _, err := q.ExecContext(ctx, `
			INSERT INTO tournament_result (
				tournament_id, tournament_group_id, phase, round,
				tournament_player_a_id, tournament_player_b_id, player_lo, player_hi, pair_index,
				score_a, score_b, played, played_at, sort_order
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, tournamentID, groupID, e.Phase, e.Round,
			idA, idB, lo, hi, e.PairIndex,
			scoreA, scoreB, played, playedAt, e.SortOrder); err != nil {
			return fmt.Errorf("insert TBD tournament_result: %w", err)
		}
	}
	return nil
}

// ListResults returns matches for a tournament with participant display fields.
func (r *Tournament) ListResults(ctx context.Context, q DBTX, tournamentID int64) ([]model.Result, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT
			tr.played,
			tr.score_a,
			tr.score_b,
			tr.played_at,
			tr.phase,
			tr.round,
			tr.tournament_group_id,
			tr.sort_order,
			tr.tournament_player_a_id,
			tr.tournament_player_b_id,
			pa.link, paa.name, pra.race, tpa.excluded,
			pb.link, pab.name, prb.race, tpb.excluded
		FROM tournament_result tr
		LEFT JOIN tournament_player tpa ON tpa.id = tr.tournament_player_a_id
		LEFT JOIN player_race pra ON pra.id = tpa.player_race_id
		LEFT JOIN player_alias paa ON paa.id = tpa.player_alias_id
		LEFT JOIN player pa ON pa.id = pra.player_id
		LEFT JOIN tournament_player tpb ON tpb.id = tr.tournament_player_b_id
		LEFT JOIN player_race prb ON prb.id = tpb.player_race_id
		LEFT JOIN player_alias pab ON pab.id = tpb.player_alias_id
		LEFT JOIN player pb ON pb.id = prb.player_id
		WHERE tr.tournament_id = ?
		ORDER BY tr.sort_order ASC, tr.id ASC
	`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("list tournament results: %w", err)
	}
	defer rows.Close()

	out := make([]model.Result, 0)
	for rows.Next() {
		var (
			played              int
			scoreA, scoreB      sql.NullInt64
			playedAt            sql.NullString
			phase, round        string
			groupID             sql.NullInt64
			sortOrder           int
			playerAID           sql.NullInt64
			playerBID           sql.NullInt64
			linkA               sql.NullString
			nameA               sql.NullString
			raceA               sql.NullString
			exclA               sql.NullInt64
			linkB               sql.NullString
			nameB               sql.NullString
			raceB               sql.NullString
			exclB               sql.NullInt64
		)
		if err := rows.Scan(
			&played, &scoreA, &scoreB, &playedAt, &phase, &round, &groupID, &sortOrder,
			&playerAID, &playerBID,
			&linkA, &nameA, &raceA, &exclA,
			&linkB, &nameB, &raceB, &exclB,
		); err != nil {
			return nil, err
		}
		res := model.Result{
			Played: played != 0,
			Phase:  phase,
			Round:  round,
			Order:  sortOrder,
		}
		if playerAID.Valid {
			res.ParticipantA = participantFromScan(linkA, nameA.String, raceA, int(exclA.Int64))
		} else {
			res.ParticipantA = tbdParticipant()
		}
		if playerBID.Valid {
			res.ParticipantB = participantFromScan(linkB, nameB.String, raceB, int(exclB.Int64))
		} else {
			res.ParticipantB = tbdParticipant()
		}
		if scoreA.Valid {
			v := int(scoreA.Int64)
			res.ScoreA = &v
		}
		if scoreB.Valid {
			v := int(scoreB.Int64)
			res.ScoreB = &v
		}
		if playedAt.Valid && playedAt.String != "" {
			v := playedAt.String
			res.DateTime = &v
		}
		if groupID.Valid {
			v := groupID.Int64
			res.GroupID = &v
		}
		if phase != "" || round != "" {
			stage := phase
			if round != "" {
				if stage != "" {
					stage += " / "
				}
				stage += round
			}
			res.Stage = &stage
		}
		out = append(out, res)
	}
	return out, rows.Err()
}

func tbdParticipant() *model.Participant {
	n := "TBD"
	return &model.Participant{Name: &n}
}

func participantFromScan(link sql.NullString, name string, race sql.NullString, excluded int) *model.Participant {
	p := model.Participant{Excluded: excluded != 0}
	n := name
	p.Name = &n
	if link.Valid && link.String != "" {
		l := link.String
		p.Link = &l
	}
	if race.Valid && race.String != "" {
		r := race.String
		p.Race = &r
	}
	return &p
}

// GroupIDByPhaseName maps lower(phase)+"\x00"+lower(name) → group id.
func (r *Tournament) GroupIDByPhaseName(ctx context.Context, q DBTX, tournamentID int64) (map[string]int64, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, phase, name FROM tournament_group WHERE tournament_id = ?
	`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("list group ids: %w", err)
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var id int64
		var phase, name string
		if err := rows.Scan(&id, &phase, &name); err != nil {
			return nil, err
		}
		key := strings.ToLower(strings.TrimSpace(phase)) + "\x00" + strings.ToLower(strings.TrimSpace(name))
		out[key] = id
	}
	return out, rows.Err()
}

// TournamentDueRefresh is a tournament needing a Liquipedia re-parse.
type TournamentDueRefresh struct {
	ID   int64
	Link string
}

func scanTournamentIDs(rows *sql.Rows, wrap error) ([]TournamentDueRefresh, error) {
	if wrap != nil {
		return nil, wrap
	}
	defer rows.Close()
	out := make([]TournamentDueRefresh, 0)
	for rows.Next() {
		var row TournamentDueRefresh
		if err := rows.Scan(&row.ID, &row.Link); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ListTournamentsDueRefresh finds tournaments with unplayed matches whose played_at is today UTC and in the past.
func (r *Tournament) ListTournamentsDueRefresh(ctx context.Context, q DBTX, nowUTC time.Time) ([]TournamentDueRefresh, error) {
	today := nowUTC.UTC().Format("2006-01-02")
	now := nowUTC.UTC().Format(time.RFC3339)
	rows, err := q.QueryContext(ctx, `
		SELECT DISTINCT t.id, t.link
		FROM tournament t
		JOIN tournament_result tr ON tr.tournament_id = t.id
		WHERE tr.played = 0
		  AND tr.played_at IS NOT NULL
		  AND tr.played_at < ?
		  AND substr(tr.played_at, 1, 10) = ?
		ORDER BY t.id ASC
	`, now, today)
	return scanTournamentIDs(rows, errWrap(err, "list tournaments due refresh"))
}

// ListTournamentsInProgress finds tournaments with unplayed matches whose start is already past (any day).
func (r *Tournament) ListTournamentsInProgress(ctx context.Context, q DBTX, nowUTC time.Time) ([]TournamentDueRefresh, error) {
	now := nowUTC.UTC().Format(time.RFC3339)
	rows, err := q.QueryContext(ctx, `
		SELECT DISTINCT t.id, t.link
		FROM tournament t
		JOIN tournament_result tr ON tr.tournament_id = t.id
		WHERE tr.played = 0
		  AND tr.played_at IS NOT NULL
		  AND tr.played_at < ?
		ORDER BY t.id ASC
	`, now)
	return scanTournamentIDs(rows, errWrap(err, "list tournaments in progress"))
}

// ListUnfinishedTournaments returns stored tournaments that are not finished.
func (r *Tournament) ListUnfinishedTournaments(ctx context.Context, q DBTX) ([]TournamentDueRefresh, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, link
		FROM tournament
		WHERE finished = 0
		ORDER BY id ASC
	`)
	return scanTournamentIDs(rows, errWrap(err, "list unfinished tournaments"))
}

func errWrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}
