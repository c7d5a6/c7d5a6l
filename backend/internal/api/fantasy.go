package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/middleware"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

type listFantasyLeaguesResponse struct {
	Leagues []model.FantasyLeague `json:"leagues"`
}

type listFantasyPlayersResponse struct {
	Players []model.FantasyPlayerRow `json:"players"`
}

type listFantasyTeamsResponse struct {
	Teams []model.FantasyTeamRow `json:"teams"`
}

type listFantasyGroupsResponse struct {
	Groups []model.FantasyGroup `json:"groups"`
}

type fantasyLeagueResponse struct {
	League model.FantasyLeague `json:"league"`
}

type createFantasyLeagueRequest struct {
	TournamentID int64                            `json:"tournamentId"`
	MaxPlayers   *int                             `json:"maxPlayers"`
	MaxCost      *int                             `json:"maxCost"`
	CostMin      *int                             `json:"costMin"`
	CostMax      *int                             `json:"costMax"`
	Costs        []model.FantasyPlayerCostOverride `json:"costs"`
}

type createFantasyLeagueResponse struct {
	Message string              `json:"message"`
	League  model.FantasyLeague `json:"league"`
}

type activeFantasyLeagueResponse struct {
	League model.FantasyLeague `json:"league"`
}

type previewFantasyResponse struct {
	Players []model.FantasyPreviewPlayer `json:"players"`
}

type unusedTournamentsResponse struct {
	Tournaments []model.TournamentSummary `json:"tournaments"`
}

type patchLeagueRequest struct {
	MaxPlayers *int `json:"maxPlayers"`
	MaxCost    *int `json:"maxCost"`
}

type teamBodyRequest struct {
	UserID           int64   `json:"userId"`
	FantasyPlayerIDs []int64 `json:"fantasyPlayerIds"`
}

type myTeamBodyRequest struct {
	FantasyPlayerIDs []int64 `json:"fantasyPlayerIds"`
}

type teamResponse struct {
	Team model.FantasyTeamRow `json:"team"`
}

type playerResponse struct {
	Player model.FantasyPlayerRow `json:"player"`
}

// ListFantasyLeagues returns all fantasy leagues.
func (s *Server) ListFantasyLeagues(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	leagues, err := s.Fantasy.ListLeagues(r.Context())
	if err != nil {
		log.Printf("list fantasy leagues: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list fantasy leagues")
		return
	}
	if leagues == nil {
		leagues = []model.FantasyLeague{}
	}
	_ = json.NewEncoder(w).Encode(listFantasyLeaguesResponse{Leagues: leagues})
}

// GetFantasyLeague returns one league.
func (s *Server) GetFantasyLeague(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	league, err := s.Fantasy.GetLeague(r.Context(), id)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(fantasyLeagueResponse{League: *league})
}

// GetActiveFantasyLeague returns the preferred active fantasy league.
func (s *Server) GetActiveFantasyLeague(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	league, err := s.Fantasy.GetActiveLeague(r.Context())
	if err != nil {
		log.Printf("get active fantasy league: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get active fantasy league")
		return
	}
	if league == nil {
		writeError(w, http.StatusNotFound, "fantasy league not found")
		return
	}
	_ = json.NewEncoder(w).Encode(activeFantasyLeagueResponse{League: *league})
}

// PreviewFantasyLeague returns roster with computed costs.
func (s *Server) PreviewFantasyLeague(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	tid, err := strconv.ParseInt(r.URL.Query().Get("tournamentId"), 10, 64)
	if err != nil || tid <= 0 {
		writeError(w, http.StatusBadRequest, "tournamentId is required")
		return
	}
	costMin := service.DefaultCostMin
	costMax := service.DefaultCostMax
	if v := r.URL.Query().Get("costMin"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid costMin")
			return
		}
		costMin = n
	}
	if v := r.URL.Query().Get("costMax"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid costMax")
			return
		}
		costMax = n
	}
	players, err := s.Fantasy.Preview(r.Context(), tid, costMin, costMax)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	if players == nil {
		players = []model.FantasyPreviewPlayer{}
	}
	_ = json.NewEncoder(w).Encode(previewFantasyResponse{Players: players})
}

// ListUnusedTournamentsForFantasy lists tournaments without a league.
func (s *Server) ListUnusedTournamentsForFantasy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	list, err := s.Fantasy.ListUnusedTournaments(r.Context())
	if err != nil {
		log.Printf("unused tournaments: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list tournaments")
		return
	}
	if list == nil {
		list = []model.TournamentSummary{}
	}
	_ = json.NewEncoder(w).Encode(unusedTournamentsResponse{Tournaments: list})
}

// ListFantasyPlayers returns fantasy players for a league (sort=cost|points).
func (s *Server) ListFantasyPlayers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	sort := repository.ParsePlayerSort(r.URL.Query().Get("sort"))
	players, err := s.Fantasy.ListPlayers(r.Context(), id, sort)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	if players == nil {
		players = []model.FantasyPlayerRow{}
	}
	_ = json.NewEncoder(w).Encode(listFantasyPlayersResponse{Players: players})
}

// ListFantasyGroups returns tournament groups with fantasy costs for a league.
func (s *Server) ListFantasyGroups(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	groups, err := s.Fantasy.ListGroups(r.Context(), id)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	if groups == nil {
		groups = []model.FantasyGroup{}
	}
	_ = json.NewEncoder(w).Encode(listFantasyGroupsResponse{Groups: groups})
}

// GetFantasyMatchBoard returns groups + results for the Results tab.
func (s *Server) GetFantasyMatchBoard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	board, err := s.Fantasy.MatchBoard(r.Context(), id)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(board)
}

// ListFantasyTeams returns fantasy teams for a league.
func (s *Server) ListFantasyTeams(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	teams, err := s.Fantasy.ListTeams(r.Context(), id)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	if teams == nil {
		teams = []model.FantasyTeamRow{}
	}
	teams = s.attachTeamTitles(r, teams)
	_ = json.NewEncoder(w).Encode(listFantasyTeamsResponse{Teams: teams})
}

// CreateFantasyLeague creates a fantasy league for an unused tournament.
func (s *Server) CreateFantasyLeague(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	var req createFantasyLeagueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	params := service.CreateParams{
		TournamentID: req.TournamentID,
		MaxPlayers:   service.DefaultMaxPlayers,
		MaxCost:      service.DefaultMaxCost,
		CostMin:      service.DefaultCostMin,
		CostMax:      service.DefaultCostMax,
		Costs:        req.Costs,
	}
	if req.MaxPlayers != nil {
		params.MaxPlayers = *req.MaxPlayers
	}
	if req.MaxCost != nil {
		params.MaxCost = *req.MaxCost
	}
	if req.CostMin != nil {
		params.CostMin = *req.CostMin
	}
	if req.CostMax != nil {
		params.CostMax = *req.CostMax
	}
	league, err := s.Fantasy.Create(r.Context(), params)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	debuglog.Printf("CreateFantasyLeague ok id=%d tournamentId=%d", league.ID, league.TournamentID)
	_ = json.NewEncoder(w).Encode(createFantasyLeagueResponse{Message: "ok", League: league})
}

// PatchFantasyLeague updates caps when not started.
func (s *Server) PatchFantasyLeague(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	var req patchLeagueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	league, err := s.Fantasy.GetLeague(r.Context(), id)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	maxPlayers, maxCost := league.MaxPlayers, league.MaxCost
	if req.MaxPlayers != nil {
		maxPlayers = *req.MaxPlayers
	}
	if req.MaxCost != nil {
		maxCost = *req.MaxCost
	}
	updated, err := s.Fantasy.UpdateLeagueCaps(r.Context(), id, maxPlayers, maxCost)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(fantasyLeagueResponse{League: updated})
}

// StartFantasyLeague sets started.
func (s *Server) StartFantasyLeague(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	league, err := s.Fantasy.StartLeague(r.Context(), id)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(fantasyLeagueResponse{League: league})
}

// FinishFantasyLeague sets finished.
func (s *Server) FinishFantasyLeague(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	league, err := s.Fantasy.FinishLeague(r.Context(), id)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(fantasyLeagueResponse{League: league})
}

// PatchFantasyPlayer updates cost / stage points / flags.
func (s *Server) PatchFantasyPlayer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	leagueID, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	playerID, ok := pathInt64(r, "playerId")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid player id")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	patch := service.PlayerPatch{}
	if v, ok := raw["cost"]; ok {
		var n int
		if err := json.Unmarshal(v, &n); err != nil {
			writeError(w, http.StatusBadRequest, "invalid cost")
			return
		}
		patch.Cost = &n
	}
	applyStage := func(key string, set *bool, dest **int) bool {
		v, ok := raw[key]
		if !ok {
			return true
		}
		*set = true
		if string(v) == "null" {
			*dest = nil
			return true
		}
		var n int
		if err := json.Unmarshal(v, &n); err != nil {
			writeError(w, http.StatusBadRequest, "invalid "+key)
			return false
		}
		*dest = &n
		return true
	}
	if !applyStage("pointsRo24", &patch.SetRo24, &patch.PointsRo24) ||
		!applyStage("pointsRo16", &patch.SetRo16, &patch.PointsRo16) ||
		!applyStage("pointsRo8", &patch.SetRo8, &patch.PointsRo8) ||
		!applyStage("pointsRo4", &patch.SetRo4, &patch.PointsRo4) ||
		!applyStage("pointsRo2", &patch.SetRo2, &patch.PointsRo2) {
		return
	}
	if v, ok := raw["defeated"]; ok {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			writeError(w, http.StatusBadRequest, "invalid defeated")
			return
		}
		patch.Defeated = &b
	}
	if v, ok := raw["isWinner"]; ok {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			writeError(w, http.StatusBadRequest, "invalid isWinner")
			return
		}
		patch.IsWinner = &b
	}
	player, err := s.Fantasy.PatchPlayer(r.Context(), leagueID, playerID, patch)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(playerResponse{Player: player})
}

// CreateFantasyTeam admin-creates a team for a user.
func (s *Server) CreateFantasyTeam(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	leagueID, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	var req teamBodyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	team, err := s.Fantasy.AdminCreateTeam(r.Context(), leagueID, req.UserID, req.FantasyPlayerIDs)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(teamResponse{Team: s.attachOneTeamTitles(r, team)})
}

// UpdateFantasyTeam admin-updates a team roster.
func (s *Server) UpdateFantasyTeam(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	leagueID, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	teamID, ok := pathInt64(r, "teamId")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
	}
	var req teamBodyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	team, err := s.Fantasy.AdminUpdateTeam(r.Context(), leagueID, teamID, req.FantasyPlayerIDs)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(teamResponse{Team: s.attachOneTeamTitles(r, team)})
}

// DeleteFantasyTeam admin-deletes a team.
func (s *Server) DeleteFantasyTeam(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	leagueID, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	teamID, ok := pathInt64(r, "teamId")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
	}
	if err := s.Fantasy.AdminDeleteTeam(r.Context(), leagueID, teamID); err != nil {
		writeFantasyErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

// GetMyFantasyTeam returns the caller's team.
func (s *Server) GetMyFantasyTeam(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	leagueID, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	p, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	team, err := s.Fantasy.GetMyTeam(r.Context(), leagueID, p.UserID)
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	if team == nil {
		writeError(w, http.StatusNotFound, "team not found")
		return
	}
	_ = json.NewEncoder(w).Encode(teamResponse{Team: s.attachOneTeamTitles(r, *team)})
}

// PutMyFantasyTeam upserts the caller's team (only before start).
func (s *Server) PutMyFantasyTeam(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Fantasy == nil {
		writeError(w, http.StatusInternalServerError, "fantasy service not configured")
		return
	}
	leagueID, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid fantasy league id")
		return
	}
	p, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req myTeamBodyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	team, err := s.Fantasy.UpsertTeam(r.Context(), service.UpsertTeamParams{
		LeagueID:          leagueID,
		UserID:            p.UserID,
		FantasyPlayerIDs:  req.FantasyPlayerIDs,
		RequireNotStarted: true,
	})
	if err != nil {
		writeFantasyErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(teamResponse{Team: s.attachOneTeamTitles(r, team)})
}

func writeFantasyErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrFantasyNotFound):
		writeError(w, http.StatusNotFound, "fantasy league not found")
	case errors.Is(err, service.ErrFantasyConflict):
		writeError(w, http.StatusConflict, "fantasy league already exists")
	case errors.Is(err, service.ErrFantasyTeamExists):
		writeError(w, http.StatusConflict, "fantasy team already exists")
	case errors.Is(err, service.ErrFantasyLeagueStarted):
		writeError(w, http.StatusConflict, "fantasy league already started")
	case errors.Is(err, service.ErrFantasyNotStarted):
		writeError(w, http.StatusConflict, "fantasy league not started")
	case errors.Is(err, service.ErrFantasyFinished):
		writeError(w, http.StatusConflict, "fantasy league finished")
	case errors.Is(err, service.ErrFantasyTeamLocked):
		writeError(w, http.StatusConflict, "fantasy team locked")
	case errors.Is(err, service.ErrFantasyInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	case strings.Contains(err.Error(), "not found"):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		log.Printf("fantasy error: %v", err)
		writeError(w, http.StatusInternalServerError, "fantasy request failed")
	}
}

func pathInt64(r *http.Request, name string) (int64, bool) {
	raw := r.PathValue(name)
	if raw == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
