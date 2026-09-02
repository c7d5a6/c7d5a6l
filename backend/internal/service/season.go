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
	ErrSeasonNotFound = errors.New("season not found")
	ErrSeasonNoActive = errors.New("no active season")
	ErrSeasonInvalid  = errors.New("invalid season request")
)

// Season orchestrates season lifecycle and rating recalculation.
type Season struct {
	db      *sql.DB
	repo    *repository.Season
	players *repository.Player
	calc    rating.Calculator
}

func NewSeason(db *sql.DB, repo *repository.Season, players *repository.Player) *Season {
	return &Season{db: db, repo: repo, players: players}
}

// GetCurrent returns the active season summary.
func (s *Season) GetCurrent(ctx context.Context) (*model.Season, error) {
	return s.repo.GetActiveSeason(ctx, s.db)
}

// SyncActiveSeasonStartElo updates the active season baseline for one player_race row.
// player_race.elo and season start rating stay equal after an admin edit.
func (s *Season) SyncActiveSeasonStartElo(ctx context.Context, playerRaceID int64, elo float64) error {
	return s.repo.SyncActiveSeasonStartElo(ctx, s.db, playerRaceID, elo)
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
	seasonStart := active.StartedAt
	if len(seasonStart) > 10 {
		seasonStart = seasonStart[:10]
	}

	tournaments, err := s.repo.ListTournamentsInSeasonWindow(ctx, s.db, seasonStart, nowISO)
	if err != nil {
		return nil, err
	}

	return &model.SeasonClosePreview{
		Season:                 *active,
		Tournaments:            tournaments,
		ClosingFantasyLeagueID: active.ClosingFantasyLeagueID,
	}, nil
}

// CloseSeason recalculates elos and closes the active season in one transaction.
// A nil tournamentIDs list includes every finished tournament in the season window.
func (s *Season) CloseSeason(ctx context.Context, tournamentIDs []int64) (*model.Season, int, error) {
	return s.closeSeason(ctx, tournamentIDs, nil)
}

// CloseSeasonForFantasyStart freezes live ratings as the season end and opens the next season.
// No-op when there is no active season.
func (s *Season) CloseSeasonForFantasyStart(ctx context.Context, fantasyLeagueID int64) (*model.Season, int, error) {
	season, n, err := s.closeSeason(ctx, nil, &fantasyLeagueID)
	if errors.Is(err, ErrSeasonNoActive) {
		return nil, 0, nil
	}
	return season, n, err
}

func (s *Season) closeSeason(ctx context.Context, tournamentIDs []int64, fantasyLeagueID *int64) (*model.Season, int, error) {
	active, err := s.repo.GetActiveSeason(ctx, s.db)
	if err != nil {
		return nil, 0, err
	}
	if active == nil {
		return nil, 0, ErrSeasonNoActive
	}

	ids := tournamentIDs
	if ids == nil {
		ids, err = s.finishedSeasonTournamentIDs(ctx, active)
		if err != nil {
			return nil, 0, err
		}
	}

	endElos, endRanks, err := s.computeSeasonRatings(ctx, active, ids)
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
		SeasonID:               active.ID,
		NewSeasonName:          newName,
		ClosedAt:               now,
		StartedAt:              now,
		TournamentIDs:          ids,
		ClosingFantasyLeagueID: fantasyLeagueID,
		EndElos:                endElos,
		EndRanks:               endRanks,
		StartElos:              endElos,
		StartRanks:             endRanks,
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
// Live ratings come from current-season matches. Rank/rating deltas compare against
// the previous season's end snapshot. player_race.elo is the current season start.
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

	snapshots, err := s.repo.ListActiveSeasonSnapshots(ctx, s.db, active.ID)
	if err != nil {
		return nil, nil, err
	}
	startEloByPR := make(map[int64]float64, len(snapshots))
	for _, snap := range snapshots {
		startEloByPR[snap.PlayerRaceID] = snap.StartElo
	}

	prevEndElo, prevEndRank, err := s.previousSeasonEnd(ctx, active.ID)
	if err != nil {
		return nil, nil, err
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
			v := projected
			entries[i].ProjectedElo = &v
		}
		if elo, ok := prevEndElo[id]; ok {
			v := elo
			entries[i].LastSeasonEndElo = &v
		}
		if rank, ok := prevEndRank[id]; ok {
			v := rank
			entries[i].LastSeasonEndRank = &v
			if projectedRank, hasProjected := projectedRanks[id]; hasProjected {
				delta := rank - projectedRank
				entries[i].RankDelta = &delta
			}
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

func (s *Season) previousSeasonEnd(ctx context.Context, activeSeasonID int64) (map[int64]float64, map[int64]int, error) {
	prev, err := s.repo.GetPreviousClosedSeason(ctx, s.db, activeSeasonID)
	if err != nil || prev == nil {
		return nil, nil, err
	}
	snaps, err := s.repo.ListActiveSeasonSnapshots(ctx, s.db, prev.ID)
	if err != nil {
		return nil, nil, err
	}
	endElo := make(map[int64]float64)
	endRank := make(map[int64]int)
	for _, snap := range snaps {
		if snap.EndElo != nil {
			endElo[snap.PlayerRaceID] = *snap.EndElo
		}
		if snap.EndRank != nil {
			endRank[snap.PlayerRaceID] = *snap.EndRank
		}
	}
	return endElo, endRank, nil
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

func (s *Season) finishedSeasonTournamentIDs(ctx context.Context, active *model.Season) ([]int64, error) {
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
		if !t.Finished {
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

	seasonIDs := seasonTournamentIDs
	if seasonIDs == nil {
		seasonIDs, err = s.finishedSeasonTournamentIDs(ctx, active)
		if err != nil {
			return nil, nil, err
		}
	}

	seasonMatches, err := s.repo.ListRatingMatches(ctx, s.db, seasonIDs)
	if err != nil {
		return nil, nil, err
	}

	elos := s.calc.Compute(startElos, seasonMatches, nil)
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
		ei := entries[i].Elo
		if entries[i].ProjectedElo != nil {
			ei = *entries[i].ProjectedElo
		}
		ej := entries[j].Elo
		if entries[j].ProjectedElo != nil {
			ej = *entries[j].ProjectedElo
		}
		if ei != ej {
			return ei > ej
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
