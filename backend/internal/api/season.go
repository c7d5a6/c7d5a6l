package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/middleware"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

type seasonResponse struct {
	Season model.Season `json:"season"`
}

type closeSeasonRequest struct {
	TournamentIDs []int64 `json:"tournamentIds"`
}

type closeSeasonResponse struct {
	Message        string       `json:"message"`
	Season         model.Season `json:"season"`
	PlayersUpdated int          `json:"playersUpdated"`
}

// GetCurrentSeason returns the active season.
func (s *Server) GetCurrentSeason(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, ok := middleware.PrincipalFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	season, err := s.Seasons.GetCurrent(r.Context())
	if err != nil {
		log.Printf("get current season: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load season")
		return
	}
	if season == nil {
		writeError(w, http.StatusNotFound, "no active season")
		return
	}
	_ = json.NewEncoder(w).Encode(seasonResponse{Season: *season})
}

// GetSeasonClosePreview returns tournaments for the season-close admin screen.
func (s *Server) GetSeasonClosePreview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	preview, err := s.Seasons.GetClosePreview(r.Context())
	if err != nil {
		if errors.Is(err, service.ErrSeasonNoActive) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		log.Printf("season close preview: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load season preview")
		return
	}
	_ = json.NewEncoder(w).Encode(preview)
}

// CloseSeason finishes the active season and opens the next one.
func (s *Server) CloseSeason(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req closeSeasonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.TournamentIDs == nil {
		req.TournamentIDs = []int64{}
	}

	season, updated, err := s.Seasons.CloseSeason(r.Context(), req.TournamentIDs)
	if err != nil {
		if errors.Is(err, service.ErrSeasonNoActive) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		log.Printf("close season: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to close season")
		return
	}
	debuglog.Printf("CloseSeason newSeasonId=%d playersUpdated=%d", season.ID, updated)
	_ = json.NewEncoder(w).Encode(closeSeasonResponse{
		Message:        "season closed",
		Season:         *season,
		PlayersUpdated: updated,
	})
}
