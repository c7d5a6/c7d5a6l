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

	endElos, endRanks, err := s.computeSeasonRatings(ctx, active, tournamentIDs)
	if err != nil {
		return nil, 0, err
	}

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
// Ratings, ranks, and rank deltas are computed in memory from current-season matches
// (all finished tournaments in the season window plus the closing fantasy league).
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

	startRanks, err := s.repo.ListSeasonStartRanks(ctx, s.db, active.ID)
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

	projectedElos, projectedRanks, err := s.computeLiveSeasonRatings(ctx, active)
	if err != nil {
		return nil, nil, err
	}

	for i := range entries {
		id := entries[i].PlayerRaceID
		if elo, ok := startEloByPR[id]; ok {
			v := elo
			entries[i].SeasonStartElo = &v
		}
		if projected, ok := projectedElos[id]; ok {
			entries[i].Elo = projected
		}
		startRank, hasStart := startRanks[id]
		projectedRank, hasProjected := projectedRanks[id]
		if hasStart && hasProjected {
			delta := startRank - projectedRank
			entries[i].RankDelta = &delta
		}
	}

	sortEntriesByRank(entries, projectedRanks)

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

func (s *Season) loadStartElos(ctx context.Context, seasonID int64) (map[int64]float64, error) {
	snapshots, err := s.repo.ListActiveSeasonSnapshots(ctx, s.db, seasonID)
	if err != nil {
		return nil, err
	}
	startElos := make(map[int64]float64, len(snapshots))
	for _, snap := range snapshots {
		startElos[snap.PlayerRaceID] = snap.StartElo
	}
	allElos, err := s.repo.ListAllPlayerRaceElos(ctx, s.db)
	if err != nil {
		return nil, err
	}
	for id, elo := range allElos {
		if _, ok := startElos[id]; !ok {
			startElos[id] = elo
		}
	}
	return startElos, nil
}

func (s *Season) fantasyLeagueTourID(ctx context.Context, active *model.Season) (int64, error) {
	if active.ClosingFantasyLeagueID == nil {
		return 0, nil
	}
	flTourID, err := s.repo.FantasyLeagueTournamentID(ctx, s.db, *active.ClosingFantasyLeagueID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	return flTourID, nil
}

func (s *Season) loadFantasyLeagueMatches(ctx context.Context, active *model.Season) ([]rating.Match, error) {
	flTourID, err := s.fantasyLeagueTourID(ctx, active)
	if err != nil || flTourID == 0 {
		return nil, err
	}
	return s.repo.ListRatingMatches(ctx, s.db, []int64{flTourID})
}

func (s *Season) finishedSeasonTournamentIDs(ctx context.Context, active *model.Season, flTourID int64) ([]int64, error) {
	seasonStart := active.StartedAt
	if len(seasonStart) > 10 {
		seasonStart = seasonStart[:10]
	}
	tournaments, err := s.repo.ListTournamentsInSeasonWindow(ctx, s.db, seasonStart, repository.NowISO())
	if err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(tournaments))
	for _, t := range tournaments {
		if !t.Finished || t.ID == flTourID {
			continue
		}
		out = append(out, t.ID)
	}
	return out, nil
}

func (s *Season) computeSeasonRatings(ctx context.Context, active *model.Season, seasonTournamentIDs []int64) (map[int64]float64, map[int64]int, error) {
	startElos, err := s.loadStartElos(ctx, active.ID)
	if err != nil {
		return nil, nil, err
	}

	flTourID, err := s.fantasyLeagueTourID(ctx, active)
	if err != nil {
		return nil, nil, err
	}

	seasonIDs := seasonTournamentIDs
	if seasonIDs == nil {
		seasonIDs, err = s.finishedSeasonTournamentIDs(ctx, active, flTourID)
		if err != nil {
			return nil, nil, err
		}
	}

	seasonMatches, err := s.repo.ListRatingMatches(ctx, s.db, seasonIDs)
	if err != nil {
		return nil, nil, err
	}

	flMatches, err := s.loadFantasyLeagueMatches(ctx, active)
	if err != nil {
		return nil, nil, err
	}

	elos := s.calc.Compute(startElos, seasonMatches, flMatches)
	return elos, computeRanks(elos), nil
}

func (s *Season) computeLiveSeasonRatings(ctx context.Context, active *model.Season) (map[int64]float64, map[int64]int, error) {
	return s.computeSeasonRatings(ctx, active, nil)
}

func sortEntriesByRank(entries []model.PlayerRaceEntry, ranks map[int64]int) {
	sort.Slice(entries, func(i, j int) bool {
		ri, okI := ranks[entries[i].PlayerRaceID]
		rj, okJ := ranks[entries[j].PlayerRaceID]
		if okI && okJ {
			if ri != rj {
				return ri < rj
			}
		} else if okI != okJ {
			return okI
		}
		if entries[i].Elo != entries[j].Elo {
			return entries[i].Elo > entries[j].Elo
		}
		return entries[i].PlayerRaceID < entries[j].PlayerRaceID
	})
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
