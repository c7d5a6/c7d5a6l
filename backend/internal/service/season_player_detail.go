package service

import (
	"context"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
)

// GetPlayerDetail builds the player dossier: identity, season race ratings,
// map-based winrates, and current-season match list.
func (s *Season) GetPlayerDetail(ctx context.Context, playerID int64) (*model.PlayerDetail, error) {
	if playerID <= 0 {
		return nil, ErrPlayerNotFound
	}

	page, err := s.players.GetByID(ctx, s.db, playerID)
	if err != nil {
		return nil, err
	}
	if page == nil {
		return nil, ErrPlayerNotFound
	}

	entries, seasonSummary, err := s.ListRaceEntriesWithSeason(ctx)
	if err != nil {
		return nil, err
	}

	races := make([]model.PlayerRaceEntry, 0)
	for i, e := range entries {
		if e.PlayerID != playerID {
			continue
		}
		rank := i + 1
		e.Rank = &rank
		races = append(races, e)
	}

	seasonTourIDs, err := s.seasonWindowTournamentIDs(ctx)
	if err != nil {
		return nil, err
	}

	var fantasyTourIDs []int64
	if s.fantasy != nil {
		active, err := s.fantasy.GetActiveLeague(ctx, s.db)
		if err != nil {
			return nil, err
		}
		if active != nil {
			fantasyTourIDs = []int64{active.TournamentID}
		} else {
			fantasyTourIDs = []int64{}
		}
	} else {
		fantasyTourIDs = []int64{}
	}

	var allFantasyTourIDs []int64
	if s.fantasy != nil {
		allFantasyTourIDs, err = s.fantasy.ListFantasyTournamentIDs(ctx, s.db)
		if err != nil {
			return nil, err
		}
	} else {
		allFantasyTourIDs = []int64{}
	}

	seasonRows, err := s.players.ListPlayedMatchesForPlayer(ctx, s.db, playerID, seasonTourIDs)
	if err != nil {
		return nil, err
	}
	fantasyRows, err := s.players.ListPlayedMatchesForPlayer(ctx, s.db, playerID, fantasyTourIDs)
	if err != nil {
		return nil, err
	}
	overallRows, err := s.players.ListPlayedMatchesForPlayer(ctx, s.db, playerID, nil)
	if err != nil {
		return nil, err
	}
	allFantasyRows, err := s.players.ListPlayedMatchesForPlayer(ctx, s.db, playerID, allFantasyTourIDs)
	if err != nil {
		return nil, err
	}

	seasonMatches := make([]model.PlayerMatch, 0, len(seasonRows))
	for _, row := range seasonRows {
		m, ok := row.ToPlayerMatch(playerID)
		if !ok {
			continue
		}
		seasonMatches = append(seasonMatches, m)
	}

	ids := page.IDs
	if ids == nil {
		ids = []string{}
	}

	return &model.PlayerDetail{
		ID:            page.ID,
		Link:          page.Link,
		Name:          page.Name,
		RealName:      page.RealName,
		IDs:           ids,
		PreferredRace: page.PreferredRace,
		HasPortrait:   page.HasPortrait,
		Races:         races,
		Winrates: model.PlayerWinrates{
			Season:     winrateBlockFromRows(playerID, seasonRows),
			Fantasy:    winrateBlockFromRows(playerID, fantasyRows),
			Overall:    winrateBlockFromRows(playerID, overallRows),
			AllFantasy: winrateBlockFromRows(playerID, allFantasyRows),
		},
		SeasonMatches: seasonMatches,
		Season:        seasonSummary,
	}, nil
}

func (s *Season) seasonWindowTournamentIDs(ctx context.Context) ([]int64, error) {
	active, err := s.repo.GetActiveSeason(ctx, s.db)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return []int64{}, nil
	}
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
		out = append(out, t.ID)
	}
	return out, nil
}

func winrateBlockFromRows(playerID int64, rows []repository.PlayerMatchRow) model.WinrateBlock {
	var block model.WinrateBlock
	for _, row := range rows {
		m, ok := row.ToPlayerMatch(playerID)
		if !ok {
			continue
		}
		if m.ScoreMine+m.ScoreOpp == 0 {
			continue
		}
		opp := ""
		if m.OpponentRace != nil {
			opp = strings.ToLower(strings.TrimSpace(*m.OpponentRace))
		}
		addMaps(&block.VsAll, m.ScoreMine, m.ScoreOpp)
		switch opp {
		case "terran":
			addMaps(&block.VsTerran, m.ScoreMine, m.ScoreOpp)
		case "zerg":
			addMaps(&block.VsZerg, m.ScoreMine, m.ScoreOpp)
		case "protoss":
			addMaps(&block.VsProtoss, m.ScoreMine, m.ScoreOpp)
		}
	}
	finalizeWinrate(&block.VsAll)
	finalizeWinrate(&block.VsTerran)
	finalizeWinrate(&block.VsZerg)
	finalizeWinrate(&block.VsProtoss)
	return block
}

func addMaps(stat *model.WinrateStat, wins, losses int) {
	stat.Wins += wins
	stat.Losses += losses
}

func finalizeWinrate(stat *model.WinrateStat) {
	total := stat.Wins + stat.Losses
	if total == 0 {
		stat.Rate = nil
		return
	}
	rate := 100.0 * float64(stat.Wins) / float64(total)
	stat.Rate = &rate
}
