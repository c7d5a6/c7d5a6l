package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// TournamentEndDate returns the end_date for a tournament (YYYY-MM-DD), or "" when unset.
func (r *Season) TournamentEndDate(ctx context.Context, q DBTX, tournamentID int64) (string, error) {
	var end sql.NullString
	err := q.QueryRowContext(ctx, `SELECT end_date FROM tournament WHERE id = ?`, tournamentID).Scan(&end)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("tournament end date: %w", err)
	}
	if !end.Valid {
		return "", nil
	}
	return strings.TrimSpace(end.String), nil
}

// SplitTournamentsAroundFantasy partitions tournament ids by the fantasy tournament end date.
// The fantasy tournament itself is included in before.
func (r *Season) SplitTournamentsAroundFantasy(ctx context.Context, q DBTX, tournamentIDs []int64, fantasyTournamentID int64) (before []int64, after []int64, err error) {
	if fantasyTournamentID <= 0 || len(tournamentIDs) == 0 {
		return tournamentIDs, nil, nil
	}
	pivot, err := r.TournamentEndDate(ctx, q, fantasyTournamentID)
	if err != nil {
		return nil, nil, err
	}
	if pivot == "" {
		return tournamentIDs, nil, nil
	}

	before = make([]int64, 0, len(tournamentIDs))
	after = make([]int64, 0)
	for _, id := range tournamentIDs {
		if id == fantasyTournamentID {
			before = append(before, id)
			continue
		}
		end, err := r.TournamentEndDate(ctx, q, id)
		if err != nil {
			return nil, nil, err
		}
		if end == "" || end <= pivot {
			before = append(before, id)
			continue
		}
		after = append(after, id)
	}
	return before, after, nil
}

// FantasyTournamentFromLeague returns the tournament id for a fantasy league.
func (r *Season) FantasyTournamentFromLeague(ctx context.Context, q DBTX, fantasyLeagueID int64) (int64, error) {
	return r.FantasyLeagueTournamentID(ctx, q, fantasyLeagueID)
}
