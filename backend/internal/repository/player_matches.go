package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// PlayerMatchRow is a raw played series involving a player (either side).
type PlayerMatchRow struct {
	ID               int64
	TournamentID     int64
	TournamentLink   string
	TournamentName   sql.NullString
	Played           bool
	ScoreA           int
	ScoreB           int
	PlayedAt         sql.NullString
	Phase            string
	Round            string
	PlayerAID        int64
	PlayerARace      string
	PlayerAName      sql.NullString
	PlayerALink      string
	PlayerBID        int64
	PlayerBRace      string
	PlayerBName      sql.NullString
	PlayerBLink      string
}

// ListPlayedMatchesForPlayer returns played series for a player, optionally
// restricted to tournamentIDs. Pass nil tournamentIDs for all tournaments.
// Pass an empty non-nil slice to return no rows.
func (r *Player) ListPlayedMatchesForPlayer(
	ctx context.Context,
	q DBTX,
	playerID int64,
	tournamentIDs []int64,
) ([]PlayerMatchRow, error) {
	if playerID <= 0 {
		return nil, nil
	}
	if tournamentIDs != nil && len(tournamentIDs) == 0 {
		return nil, nil
	}

	var b strings.Builder
	args := make([]any, 0, 2+len(tournamentIDs))
	b.WriteString(`
		SELECT
			tr.id,
			tr.tournament_id,
			t.link,
			t.name,
			tr.played,
			tr.score_a,
			tr.score_b,
			tr.played_at,
			tr.phase,
			tr.round,
			pra.player_id,
			pra.race,
			paa.name,
			COALESCE(pa.link_v2, pa.link) AS link_a,
			prb.player_id,
			prb.race,
			pab.name,
			COALESCE(pb.link_v2, pb.link) AS link_b
		FROM tournament_result tr
		JOIN tournament t ON t.id = tr.tournament_id
		JOIN tournament_player tpa ON tpa.id = tr.tournament_player_a_id
		JOIN tournament_player tpb ON tpb.id = tr.tournament_player_b_id
		JOIN player_race pra ON pra.id = tpa.player_race_id
		JOIN player_race prb ON prb.id = tpb.player_race_id
		JOIN player pa ON pa.id = pra.player_id
		JOIN player pb ON pb.id = prb.player_id
		JOIN player_alias paa ON paa.id = tpa.player_alias_id
		JOIN player_alias pab ON pab.id = tpb.player_alias_id
		WHERE tr.played = 1
			AND tr.tournament_player_a_id IS NOT NULL
			AND tr.tournament_player_b_id IS NOT NULL
			AND tr.score_a IS NOT NULL
			AND tr.score_b IS NOT NULL
			AND (pra.player_id = ? OR prb.player_id = ?)
	`)
	args = append(args, playerID, playerID)

	if tournamentIDs != nil {
		b.WriteString(" AND tr.tournament_id IN (")
		for i, id := range tournamentIDs {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString("?")
			args = append(args, id)
		}
		b.WriteString(")")
	}
	b.WriteString(" ORDER BY tr.played_at DESC, tr.sort_order DESC, tr.id DESC")

	rows, err := q.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list player matches: %w", err)
	}
	defer rows.Close()

	out := make([]PlayerMatchRow, 0)
	for rows.Next() {
		var (
			m              PlayerMatchRow
			played         int
			scoreA, scoreB sql.NullInt64
			linkA          string
		)
		if err := rows.Scan(
			&m.ID,
			&m.TournamentID,
			&m.TournamentLink,
			&m.TournamentName,
			&played,
			&scoreA,
			&scoreB,
			&m.PlayedAt,
			&m.Phase,
			&m.Round,
			&m.PlayerAID,
			&m.PlayerARace,
			&m.PlayerAName,
			&linkA,
			&m.PlayerBID,
			&m.PlayerBRace,
			&m.PlayerBName,
			&m.PlayerBLink,
		); err != nil {
			return nil, fmt.Errorf("scan player match: %w", err)
		}
		m.Played = played != 0
		m.PlayerALink = linkA
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

// ToPlayerMatch flips a raw row into the given player's perspective.
func (m PlayerMatchRow) ToPlayerMatch(playerID int64) (model.PlayerMatch, bool) {
	out := model.PlayerMatch{
		ID:             m.ID,
		TournamentID:   m.TournamentID,
		TournamentLink: m.TournamentLink,
		Played:         m.Played,
		Phase:          m.Phase,
		Round:          m.Round,
	}
	if m.TournamentName.Valid {
		v := m.TournamentName.String
		out.TournamentName = &v
	}
	if m.PlayedAt.Valid && strings.TrimSpace(m.PlayedAt.String) != "" {
		v := m.PlayedAt.String
		out.DateTime = &v
	}

	switch playerID {
	case m.PlayerAID:
		out.ScoreMine = m.ScoreA
		out.ScoreOpp = m.ScoreB
		out.MyRace = m.PlayerARace
		if m.PlayerBName.Valid {
			v := m.PlayerBName.String
			out.OpponentName = &v
		}
		link := m.PlayerBLink
		out.OpponentLink = &link
		race := m.PlayerBRace
		out.OpponentRace = &race
		id := m.PlayerBID
		out.OpponentPlayerID = &id
		return out, true
	case m.PlayerBID:
		out.ScoreMine = m.ScoreB
		out.ScoreOpp = m.ScoreA
		out.MyRace = m.PlayerBRace
		if m.PlayerAName.Valid {
			v := m.PlayerAName.String
			out.OpponentName = &v
		}
		link := m.PlayerALink
		out.OpponentLink = &link
		race := m.PlayerARace
		out.OpponentRace = &race
		id := m.PlayerAID
		out.OpponentPlayerID = &id
		return out, true
	default:
		return out, false
	}
}
