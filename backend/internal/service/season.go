package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/rating"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
)

var (
	ErrSeasonNotFound    = errors.New("season not found")
	ErrSeasonNotReady    = errors.New("season not ready to close")
	ErrSeasonNoActive    = errors.New("no active season")
	ErrSeasonInvalid     = errors.New("invalid season request")
)

// Season orchestrates season lifecycle and rating recalculation.
type Season struct {
	db       *sql.DB
	repo     *repository.Season
	players  *repository.Player
	calc     rating.Calculator
}

func NewSeason(db *sql.DB, repo *repository.Season, players *repository.Player) *Season {
	return &Season{db: db, repo: repo, players: players}
}

// GetCurrent returns the active season summary.
func (s *Season) GetCurrent(ctx context.Context) (*model.Season, error) {
	return s.repo.GetActiveSeason(ctx, s.db)
}

// GetClosePreview returns tournaments eligible for season close selection.
func (s *Season) GetClosePreview(ctx context.Context) (*model.SeasonClosePreview, error) {
	active, err := s.repo.GetActiveSeason(ctx, s.db)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return nil, ErrSeasonNoActive
	}

	nowISO := repository.NowISO()
	// Use date portion of started_at for window comparison when it includes time.
	seasonStart := active.StartedAt
	if len(seasonStart) > 10 {
		seasonStart = seasonStart[:10]
	}

	tournaments, err := s.repo.ListTournamentsInSeasonWindow(ctx, s.db, seasonStart, nowISO)
	if err != nil {
		return nil, err
	}

	var flTourID int64
	if active.ClosingFantasyLeagueID != nil {
		flTourID, err = s.repo.FantasyLeagueTournamentID(ctx, s.db, *active.ClosingFantasyLeagueID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	for i := range tournaments {
		if flTourID != 0 && tournaments[i].ID == flTourID {
			tournaments[i].IsFantasySource = true
		}
	}

	return &model.SeasonClosePreview{
		Season:                 *active,
		Tournaments:            tournaments,
		ClosingFantasyLeagueID: active.ClosingFantasyLeagueID,
	}, nil
}

// MarkReadyToClose flags the active season after a fantasy league finishes.
func (s *Season) MarkReadyToClose(ctx context.Context, fantasyLeagueID int64) error {
	return s.repo.SetReadyToClose(ctx, s.db, fantasyLeagueID)
}

// CloseSeason recalculates elos and closes the active season in one transaction.
func (s *Season) CloseSeason(ctx context.Context, tournamentIDs []int64) (*model.Season, int, error) {
	active, err := s.repo.GetActiveSeason(ctx, s.db)
	if err != nil {
		return nil, 0, err
	}
	if active == nil {
		return nil, 0, ErrSeasonNoActive
	}

	snapshots, err := s.repo.ListActiveSeasonSnapshots(ctx, s.db, active.ID)
	if err != nil {
		return nil, 0, err
	}
	startElos := make(map[int64]float64, len(snapshots))
	for _, snap := range snapshots {
		startElos[snap.PlayerRaceID] = snap.StartElo
	}
	// Include any player_race rows missing from snapshot (safety net).
	allElos, err := s.repo.ListAllPlayerRaceElos(ctx, s.db)
	if err != nil {
		return nil, 0, err
	}
	for id, elo := range allElos {
		if _, ok := startElos[id]; !ok {
			startElos[id] = elo
		}
	}

	seasonMatches, err := s.repo.ListRatingMatches(ctx, s.db, tournamentIDs)
	if err != nil {
		return nil, 0, err
	}

	var flMatches []rating.Match
	if active.ClosingFantasyLeagueID != nil {
		flTourID, err := s.repo.FantasyLeagueTournamentID(ctx, s.db, *active.ClosingFantasyLeagueID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, 0, err
		}
		if flTourID != 0 {
			flMatches, err = s.repo.ListRatingMatches(ctx, s.db, []int64{flTourID})
			if err != nil {
				return nil, 0, err
			}
		}
	}

	endElos := s.calc.Compute(startElos, seasonMatches, flMatches)
	endRanks := computeRanks(endElos)

	now := time.Now().UTC().Format(time.RFC3339)
	newName := fmt.Sprintf("Season %d", active.ID+1)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	if err := s.repo.UpdatePlayerRaceElos(ctx, tx, endElos); err != nil {
		return nil, 0, err
	}

	newSeasonID, err := s.repo.CloseAndOpenSeason(ctx, tx, repository.CloseSeasonParams{
		SeasonID:      active.ID,
		NewSeasonName: newName,
		ClosedAt:      now,
		StartedAt:     now,
		TournamentIDs: tournamentIDs,
		EndElos:       endElos,
		EndRanks:      endRanks,
		StartElos:     endElos,
		StartRanks:    endRanks,
	})
	if err != nil {
		return nil, 0, err
	}

	if err := tx.Commit(); err != nil {
		return nil, 0, err
	}

	updated, err := s.repo.GetSeasonByID(ctx, s.db, newSeasonID)
	if err != nil {
		return nil, 0, err
	}
	return updated, len(endElos), nil
}

// ListRaceEntriesWithSeason returns player rows enriched with season metadata.
func (s *Season) ListRaceEntriesWithSeason(ctx context.Context) ([]model.PlayerRaceEntry, *model.SeasonSummary, error) {
	entries, err := s.players.ListRaceEntries(ctx, s.db)
	if err != nil {
		return nil, nil, err
	}

	active, err := s.repo.GetActiveSeason(ctx, s.db)
	if err != nil {
		return nil, nil, err
	}
	if active == nil {
		return entries, nil, nil
	}

	currRanks, err := s.repo.ListSeasonStartRanks(ctx, s.db, active.ID)
	if err != nil {
		return nil, nil, err
	}

	snapshots, err := s.repo.ListActiveSeasonSnapshots(ctx, s.db, active.ID)
	if err != nil {
		return nil, nil, err
	}
	startEloByPR := make(map[int64]float64, len(snapshots))
	for _, snap := range snapshots {
		startEloByPR[snap.PlayerRaceID] = snap.StartElo
	}

	var prevRanks map[int64]int
	prev, err := s.repo.GetPreviousClosedSeason(ctx, s.db, active.ID)
	if err != nil {
		return nil, nil, err
	}
	if prev != nil {
		prevRanks, err = s.repo.ListSeasonStartRanks(ctx, s.db, prev.ID)
		if err != nil {
			return nil, nil, err
		}
	}

	for i := range entries {
		id := entries[i].PlayerRaceID
		if elo, ok := startEloByPR[id]; ok {
			v := elo
			entries[i].SeasonStartElo = &v
		}
		if prevRanks != nil {
			prevRank, hasPrev := prevRanks[id]
			currRank, hasCurr := currRanks[id]
			if hasPrev && hasCurr {
				delta := prevRank - currRank
				entries[i].RankDelta = &delta
			}
		}
	}

	summary := &model.SeasonSummary{
		ID:           active.ID,
		Name:         active.Name,
		Status:       active.Status,
		StartedAt:    active.StartedAt,
		ClosedAt:     active.ClosedAt,
		ReadyToClose: active.ReadyToClose,
	}
	return entries, summary, nil
}

type rankEntry struct {
	id  int64
	elo float64
}

func computeRanks(elos map[int64]float64) map[int64]int {
	items := make([]rankEntry, 0, len(elos))
	for id, elo := range elos {
		items = append(items, rankEntry{id: id, elo: elo})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].elo != items[j].elo {
			return items[i].elo > items[j].elo
		}
		return items[i].id < items[j].id
	})
	out := make(map[int64]int, len(items))
	for i, item := range items {
		out[item.id] = i + 1
	}
	return out
}
