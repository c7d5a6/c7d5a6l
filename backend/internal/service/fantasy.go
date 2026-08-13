package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
)

var (
	ErrFantasyNotFound      = errors.New("fantasy league not found")
	ErrFantasyConflict      = errors.New("fantasy league already exists")
	ErrFantasyInvalid       = errors.New("invalid fantasy request")
	ErrFantasyLeagueStarted = errors.New("fantasy league already started")
	ErrFantasyNotStarted    = errors.New("fantasy league not started")
	ErrFantasyFinished      = errors.New("fantasy league finished")
	ErrFantasyTeamLocked    = errors.New("fantasy team locked")
	ErrFantasyTeamExists    = errors.New("fantasy team already exists")
)

const (
	DefaultMaxPlayers = 6
	DefaultMaxCost    = 28
	DefaultCostMin    = 0
	DefaultCostMax    = 10
)

// Fantasy orchestrates fantasy league reads and mutations.
type Fantasy struct {
	db   *sql.DB
	repo *repository.Fantasy
}

func NewFantasy(db *sql.DB, repo *repository.Fantasy) *Fantasy {
	return &Fantasy{db: db, repo: repo}
}

// ListLeagues returns all fantasy leagues.
func (s *Fantasy) ListLeagues(ctx context.Context) ([]model.FantasyLeague, error) {
	return s.repo.ListLeagues(ctx, s.db)
}

// GetLeague returns a league by id.
func (s *Fantasy) GetLeague(ctx context.Context, id int64) (*model.FantasyLeague, error) {
	league, err := s.repo.GetLeagueByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if league == nil {
		return nil, ErrFantasyNotFound
	}
	return league, nil
}

// GetActiveLeague returns the preferred active fantasy league, or nil when none exist.
func (s *Fantasy) GetActiveLeague(ctx context.Context) (*model.FantasyLeague, error) {
	return s.repo.GetActiveLeague(ctx, s.db)
}

// ListUnusedTournaments returns tournaments without a fantasy league.
func (s *Fantasy) ListUnusedTournaments(ctx context.Context) ([]model.TournamentSummary, error) {
	return s.repo.ListUnusedTournaments(ctx, s.db)
}

// ListPlayers returns fantasy players for a league.
func (s *Fantasy) ListPlayers(ctx context.Context, leagueID int64, sort repository.PlayerSort) ([]model.FantasyPlayerRow, error) {
	if _, err := s.GetLeague(ctx, leagueID); err != nil {
		return nil, err
	}
	return s.repo.ListPlayers(ctx, s.db, leagueID, sort)
}

// ListTeams returns fantasy teams for a league with competition ranks.
func (s *Fantasy) ListTeams(ctx context.Context, leagueID int64) ([]model.FantasyTeamRow, error) {
	if _, err := s.GetLeague(ctx, leagueID); err != nil {
		return nil, err
	}
	teams, err := s.repo.ListTeams(ctx, s.db, leagueID)
	if err != nil {
		return nil, err
	}
	return AssignTeamRanks(teams), nil
}

// CreateParams configures fantasy league creation.
type CreateParams struct {
	TournamentID int64
	MaxPlayers   int
	MaxCost      int
	CostMin      int
	CostMax      int
	Costs        []model.FantasyPlayerCostOverride
}

// Preview returns roster with computed costs for a tournament.
func (s *Fantasy) Preview(ctx context.Context, tournamentID int64, costMin, costMax int) ([]model.FantasyPreviewPlayer, error) {
	if tournamentID <= 0 {
		return nil, fmt.Errorf("%w: tournamentId is required", ErrFantasyInvalid)
	}
	exists, err := s.repo.TournamentExists(ctx, s.db, tournamentID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("%w: tournament not found", ErrFantasyInvalid)
	}
	roster, err := s.repo.ListRosterWithElo(ctx, s.db, tournamentID)
	if err != nil {
		return nil, err
	}
	costs := ComputeEloCosts(roster, costMin, costMax)
	out := make([]model.FantasyPreviewPlayer, 0, len(roster))
	for i, row := range roster {
		p := model.FantasyPreviewPlayer{
			TournamentPlayerID: row.TournamentPlayerID,
			Elo:                row.Elo,
			Cost:               costs[i],
		}
		n := row.Name
		p.Name = &n
		if row.Link.Valid && row.Link.String != "" {
			l := row.Link.String
			p.Link = &l
		}
		if row.Race != "" {
			rc := row.Race
			p.Race = &rc
		}
		out = append(out, p)
	}
	return out, nil
}

// Create creates a new fantasy league (rejects if tournament already linked).
func (s *Fantasy) Create(ctx context.Context, p CreateParams) (model.FantasyLeague, error) {
	if p.TournamentID <= 0 {
		return model.FantasyLeague{}, fmt.Errorf("%w: tournamentId is required", ErrFantasyInvalid)
	}
	if p.MaxPlayers <= 0 {
		p.MaxPlayers = DefaultMaxPlayers
	}
	if p.MaxCost <= 0 {
		p.MaxCost = DefaultMaxCost
	}

	exists, err := s.repo.TournamentExists(ctx, s.db, p.TournamentID)
	if err != nil {
		return model.FantasyLeague{}, err
	}
	if !exists {
		return model.FantasyLeague{}, fmt.Errorf("%w: tournament not found", ErrFantasyInvalid)
	}

	existing, err := s.repo.GetLeagueIDByTournament(ctx, s.db, p.TournamentID)
	if err != nil {
		return model.FantasyLeague{}, err
	}
	if existing != 0 {
		return model.FantasyLeague{}, ErrFantasyConflict
	}

	roster, err := s.repo.ListRosterWithElo(ctx, s.db, p.TournamentID)
	if err != nil {
		return model.FantasyLeague{}, err
	}
	computed := ComputeEloCosts(roster, p.CostMin, p.CostMax)
	costByTP := make(map[int64]int, len(roster))
	for i, row := range roster {
		costByTP[row.TournamentPlayerID] = computed[i]
	}
	for _, ov := range p.Costs {
		if _, ok := costByTP[ov.TournamentPlayerID]; !ok {
			return model.FantasyLeague{}, fmt.Errorf("%w: unknown tournamentPlayerId %d", ErrFantasyInvalid, ov.TournamentPlayerID)
		}
		if ov.Cost < 0 {
			return model.FantasyLeague{}, fmt.Errorf("%w: cost must be >= 0", ErrFantasyInvalid)
		}
		costByTP[ov.TournamentPlayerID] = ov.Cost
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.FantasyLeague{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	leagueID, err := s.repo.CreateLeague(ctx, tx, p.TournamentID, p.MaxPlayers, p.MaxCost)
	if err != nil {
		return model.FantasyLeague{}, err
	}
	for _, row := range roster {
		if err := s.repo.InsertFantasyPlayer(ctx, tx, leagueID, row.TournamentPlayerID, costByTP[row.TournamentPlayerID]); err != nil {
			return model.FantasyLeague{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return model.FantasyLeague{}, fmt.Errorf("commit: %w", err)
	}

	league, err := s.repo.GetLeagueByID(ctx, s.db, leagueID)
	if err != nil || league == nil {
		return model.FantasyLeague{}, fmt.Errorf("reload league: %w", err)
	}
	debuglog.Printf("service.Fantasy.Create leagueId=%d tournamentId=%d players=%d", leagueID, p.TournamentID, len(roster))
	return *league, nil
}

// CreateOrSeed is a convenience for tests: creates with defaults or seeds missing players.
func (s *Fantasy) CreateOrSeed(ctx context.Context, tournamentID int64) (model.FantasyLeague, error) {
	existing, err := s.repo.GetLeagueIDByTournament(ctx, s.db, tournamentID)
	if err != nil {
		return model.FantasyLeague{}, err
	}
	if existing != 0 {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return model.FantasyLeague{}, err
		}
		defer tx.Rollback()
		if _, err := s.repo.SeedPlayersFromRoster(ctx, tx, existing, tournamentID); err != nil {
			return model.FantasyLeague{}, err
		}
		if err := tx.Commit(); err != nil {
			return model.FantasyLeague{}, err
		}
		league, err := s.repo.GetLeagueByID(ctx, s.db, existing)
		if err != nil || league == nil {
			return model.FantasyLeague{}, fmt.Errorf("reload: %w", err)
		}
		return *league, nil
	}
	return s.Create(ctx, CreateParams{
		TournamentID: tournamentID,
		MaxPlayers:   DefaultMaxPlayers,
		MaxCost:      DefaultMaxCost,
		CostMin:      DefaultCostMin,
		CostMax:      DefaultCostMax,
	})
}

// UpdateLeagueCaps updates max players/cost when not started.
func (s *Fantasy) UpdateLeagueCaps(ctx context.Context, id int64, maxPlayers, maxCost int) (model.FantasyLeague, error) {
	league, err := s.GetLeague(ctx, id)
	if err != nil {
		return model.FantasyLeague{}, err
	}
	if league.Started {
		return model.FantasyLeague{}, ErrFantasyLeagueStarted
	}
	if maxPlayers <= 0 || maxCost <= 0 {
		return model.FantasyLeague{}, fmt.Errorf("%w: maxPlayers and maxCost must be positive", ErrFantasyInvalid)
	}
	if err := s.repo.UpdateLeagueCaps(ctx, s.db, id, maxPlayers, maxCost); err != nil {
		return model.FantasyLeague{}, err
	}
	updated, err := s.repo.GetLeagueByID(ctx, s.db, id)
	if err != nil || updated == nil {
		return model.FantasyLeague{}, fmt.Errorf("reload: %w", err)
	}
	return *updated, nil
}

// StartLeague sets started=1.
func (s *Fantasy) StartLeague(ctx context.Context, id int64) (model.FantasyLeague, error) {
	league, err := s.GetLeague(ctx, id)
	if err != nil {
		return model.FantasyLeague{}, err
	}
	if league.Started {
		return model.FantasyLeague{}, ErrFantasyLeagueStarted
	}
	if league.Finished {
		return model.FantasyLeague{}, ErrFantasyFinished
	}
	if err := s.repo.SetLeagueStarted(ctx, s.db, id); err != nil {
		return model.FantasyLeague{}, err
	}
	updated, err := s.repo.GetLeagueByID(ctx, s.db, id)
	if err != nil || updated == nil {
		return model.FantasyLeague{}, fmt.Errorf("reload: %w", err)
	}
	return *updated, nil
}

// FinishLeague sets finished=1.
func (s *Fantasy) FinishLeague(ctx context.Context, id int64) (model.FantasyLeague, error) {
	league, err := s.GetLeague(ctx, id)
	if err != nil {
		return model.FantasyLeague{}, err
	}
	if !league.Started {
		return model.FantasyLeague{}, ErrFantasyNotStarted
	}
	if league.Finished {
		return model.FantasyLeague{}, ErrFantasyFinished
	}
	if err := s.repo.SetLeagueFinished(ctx, s.db, id); err != nil {
		return model.FantasyLeague{}, err
	}
	updated, err := s.repo.GetLeagueByID(ctx, s.db, id)
	if err != nil || updated == nil {
		return model.FantasyLeague{}, fmt.Errorf("reload: %w", err)
	}
	return *updated, nil
}

// PatchPlayerParams updates fantasy player fields; nil pointers mean leave unchanged.
type PlayerPatch struct {
	Cost       *int
	PointsRo24 *int // if setStageRo24
	SetRo24    bool
	PointsRo16 *int
	SetRo16    bool
	PointsRo8  *int
	SetRo8     bool
	PointsRo4  *int
	SetRo4     bool
	PointsRo2  *int
	SetRo2     bool
	Defeated   *bool
	IsWinner   *bool
}

// PatchPlayer updates a fantasy player.
func (s *Fantasy) PatchPlayer(ctx context.Context, leagueID, playerID int64, patch PlayerPatch) (model.FantasyPlayerRow, error) {
	if _, err := s.GetLeague(ctx, leagueID); err != nil {
		return model.FantasyPlayerRow{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.FantasyPlayerRow{}, err
	}
	defer tx.Rollback()

	cur, err := s.repo.GetPlayerByID(ctx, tx, leagueID, playerID)
	if err != nil {
		return model.FantasyPlayerRow{}, err
	}
	if cur == nil {
		return model.FantasyPlayerRow{}, ErrFantasyNotFound
	}

	if patch.Cost != nil {
		if *patch.Cost < 0 {
			return model.FantasyPlayerRow{}, fmt.Errorf("%w: cost must be >= 0", ErrFantasyInvalid)
		}
		cur.Cost = *patch.Cost
	}
	if patch.SetRo24 {
		cur.PointsRo24 = patch.PointsRo24
	}
	if patch.SetRo16 {
		cur.PointsRo16 = patch.PointsRo16
	}
	if patch.SetRo8 {
		cur.PointsRo8 = patch.PointsRo8
	}
	if patch.SetRo4 {
		cur.PointsRo4 = patch.PointsRo4
	}
	if patch.SetRo2 {
		cur.PointsRo2 = patch.PointsRo2
	}
	if patch.Defeated != nil {
		cur.Defeated = *patch.Defeated
	}
	if patch.IsWinner != nil {
		if *patch.IsWinner {
			if err := s.repo.ClearWinnersInLeague(ctx, tx, leagueID); err != nil {
				return model.FantasyPlayerRow{}, err
			}
			cur.IsWinner = true
		} else {
			cur.IsWinner = false
		}
	}

	if err := s.repo.UpdatePlayer(ctx, tx, *cur); err != nil {
		return model.FantasyPlayerRow{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.FantasyPlayerRow{}, err
	}
	updated, err := s.repo.GetPlayerByID(ctx, s.db, leagueID, playerID)
	if err != nil || updated == nil {
		return model.FantasyPlayerRow{}, fmt.Errorf("reload player: %w", err)
	}
	return *updated, nil
}

// UpsertTeamParams creates or replaces a team roster.
type UpsertTeamParams struct {
	LeagueID         int64
	UserID           int64
	TeamID           int64 // 0 = create / resolve by user
	FantasyPlayerIDs []int64
	RequireNotStarted bool
}

// UpsertTeam validates and saves a team roster.
func (s *Fantasy) UpsertTeam(ctx context.Context, p UpsertTeamParams) (model.FantasyTeamRow, error) {
	league, err := s.GetLeague(ctx, p.LeagueID)
	if err != nil {
		return model.FantasyTeamRow{}, err
	}
	if p.RequireNotStarted && league.Started {
		return model.FantasyTeamRow{}, ErrFantasyTeamLocked
	}
	if p.UserID <= 0 {
		return model.FantasyTeamRow{}, fmt.Errorf("%w: userId required", ErrFantasyInvalid)
	}
	ok, err := s.repo.UserExists(ctx, s.db, p.UserID)
	if err != nil {
		return model.FantasyTeamRow{}, err
	}
	if !ok {
		return model.FantasyTeamRow{}, fmt.Errorf("%w: user not found", ErrFantasyInvalid)
	}

	seen := map[int64]struct{}{}
	ids := make([]int64, 0, len(p.FantasyPlayerIDs))
	for _, id := range p.FantasyPlayerIDs {
		if id <= 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			return model.FantasyTeamRow{}, fmt.Errorf("%w: duplicate player", ErrFantasyInvalid)
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) > league.MaxPlayers {
		return model.FantasyTeamRow{}, fmt.Errorf("%w: too many players (max %d)", ErrFantasyInvalid, league.MaxPlayers)
	}

	players, err := s.repo.PlayersByIDs(ctx, s.db, p.LeagueID, ids)
	if err != nil {
		return model.FantasyTeamRow{}, err
	}
	if len(players) != len(ids) {
		return model.FantasyTeamRow{}, fmt.Errorf("%w: player not in league", ErrFantasyInvalid)
	}
	sum := 0
	for _, pl := range players {
		sum += pl.Cost
	}
	if sum > league.MaxCost {
		return model.FantasyTeamRow{}, fmt.Errorf("%w: cost sum %d exceeds max %d", ErrFantasyInvalid, sum, league.MaxCost)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.FantasyTeamRow{}, err
	}
	defer tx.Rollback()

	var teamID int64
	if p.TeamID > 0 {
		team, err := s.repo.GetTeamByID(ctx, tx, p.LeagueID, p.TeamID)
		if err != nil {
			return model.FantasyTeamRow{}, err
		}
		if team == nil {
			return model.FantasyTeamRow{}, ErrFantasyNotFound
		}
		teamID = team.ID
	} else {
		existing, err := s.repo.GetTeamByUser(ctx, tx, p.LeagueID, p.UserID)
		if err != nil {
			return model.FantasyTeamRow{}, err
		}
		if existing != nil {
			teamID = existing.ID
		} else {
			teamID, err = s.repo.CreateTeam(ctx, tx, p.LeagueID, p.UserID)
			if err != nil {
				return model.FantasyTeamRow{}, err
			}
		}
	}

	if err := s.repo.ReplaceTeamMembers(ctx, tx, teamID, ids); err != nil {
		return model.FantasyTeamRow{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.FantasyTeamRow{}, err
	}

	team, err := s.repo.GetTeamByID(ctx, s.db, p.LeagueID, teamID)
	if err != nil || team == nil {
		return model.FantasyTeamRow{}, fmt.Errorf("reload team: %w", err)
	}
	return *team, nil
}

// GetMyTeam returns the caller's team.
func (s *Fantasy) GetMyTeam(ctx context.Context, leagueID, userID int64) (*model.FantasyTeamRow, error) {
	if _, err := s.GetLeague(ctx, leagueID); err != nil {
		return nil, err
	}
	return s.repo.GetTeamByUser(ctx, s.db, leagueID, userID)
}

// AdminCreateTeam creates a team for a user who does not have one yet.
func (s *Fantasy) AdminCreateTeam(ctx context.Context, leagueID, userID int64, playerIDs []int64) (model.FantasyTeamRow, error) {
	existing, err := s.repo.GetTeamByUser(ctx, s.db, leagueID, userID)
	if err != nil {
		return model.FantasyTeamRow{}, err
	}
	if existing != nil {
		return model.FantasyTeamRow{}, ErrFantasyTeamExists
	}
	return s.UpsertTeam(ctx, UpsertTeamParams{
		LeagueID:          leagueID,
		UserID:            userID,
		FantasyPlayerIDs:  playerIDs,
		RequireNotStarted: false,
	})
}

// AdminUpdateTeam replaces an existing team's roster.
func (s *Fantasy) AdminUpdateTeam(ctx context.Context, leagueID, teamID int64, playerIDs []int64) (model.FantasyTeamRow, error) {
	team, err := s.repo.GetTeamByID(ctx, s.db, leagueID, teamID)
	if err != nil {
		return model.FantasyTeamRow{}, err
	}
	if team == nil {
		return model.FantasyTeamRow{}, ErrFantasyNotFound
	}
	return s.UpsertTeam(ctx, UpsertTeamParams{
		LeagueID:          leagueID,
		UserID:            team.UserID,
		TeamID:            teamID,
		FantasyPlayerIDs:  playerIDs,
		RequireNotStarted: false,
	})
}

// AdminDeleteTeam removes a team.
func (s *Fantasy) AdminDeleteTeam(ctx context.Context, leagueID, teamID int64) error {
	if _, err := s.GetLeague(ctx, leagueID); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.repo.DeleteTeam(ctx, tx, leagueID, teamID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFantasyNotFound
		}
		return err
	}
	return tx.Commit()
}

// AssignTeamRanks sets competition ranks and sorts by rank ASC, then cost DESC.
func AssignTeamRanks(teams []model.FantasyTeamRow) []model.FantasyTeamRow {
	for i := range teams {
		pts, cost := 0, 0
		for _, m := range teams[i].Members {
			pts += m.PointsEarned
			cost += m.Cost
		}
		teams[i].Points = pts
		teams[i].Cost = cost
	}
	// Order by points DESC for rank assignment.
	sort.SliceStable(teams, func(i, j int) bool {
		if teams[i].Points != teams[j].Points {
			return teams[i].Points > teams[j].Points
		}
		return teams[i].Cost > teams[j].Cost
	})
	for i := range teams {
		if i > 0 && teams[i].Points == teams[i-1].Points {
			teams[i].Rank = teams[i-1].Rank
		} else {
			teams[i].Rank = i + 1
		}
	}
	// Final display order: rank ASC, cost DESC.
	sort.SliceStable(teams, func(i, j int) bool {
		if teams[i].Rank != teams[j].Rank {
			return teams[i].Rank < teams[j].Rank
		}
		return teams[i].Cost > teams[j].Cost
	})
	return teams
}

// ComputeEloCosts maps roster elo to integer costs in [costMin, costMax].
func ComputeEloCosts(roster []repository.RosterEloRow, costMin, costMax int) []int {
	n := len(roster)
	out := make([]int, n)
	if n == 0 {
		return out
	}
	if costMax < costMin {
		costMin, costMax = costMax, costMin
	}
	minElo, maxElo := roster[0].Elo, roster[0].Elo
	for _, row := range roster[1:] {
		if row.Elo < minElo {
			minElo = row.Elo
		}
		if row.Elo > maxElo {
			maxElo = row.Elo
		}
	}
	if maxElo == minElo {
		for i := range out {
			out[i] = costMin
		}
		return out
	}
	span := float64(costMax - costMin)
	for i, row := range roster {
		raw := float64(costMin) + (row.Elo-minElo)/(maxElo-minElo)*span
		out[i] = int(math.Round(raw))
	}
	return out
}
