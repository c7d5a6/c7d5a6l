package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

type mergeCandidatesResponse struct {
	Candidates []model.MergeCandidate `json:"candidates"`
}

// ListMergeCandidates suggests players that may be aliases of the main account.
func (s *Server) ListMergeCandidates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	playerID, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid player id")
		return
	}
	query := r.URL.Query().Get("q")
	limit := 15
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}

	candidates, err := s.Players.MergeCandidates(r.Context(), playerID, query, limit)
	if err != nil {
		if errors.Is(err, service.ErrPlayerMergeMissing) || errors.Is(err, service.ErrPlayerNotFound) {
			writeError(w, http.StatusNotFound, "player not found")
			return
		}
		if errors.Is(err, service.ErrInvalidPlayer) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("merge candidates player %d: %v", playerID, err)
		writeError(w, http.StatusInternalServerError, "failed to list merge candidates")
		return
	}
	debuglog.Printf("ListMergeCandidates player=%d count=%d", playerID, len(candidates))
	_ = json.NewEncoder(w).Encode(mergeCandidatesResponse{Candidates: candidates})
}

type mergePlayersRequest struct {
	MainPlayerID  int64 `json:"mainPlayerId"`
	AliasPlayerID int64 `json:"aliasPlayerId"`
}

// MergePlayers folds aliasPlayerId into mainPlayerId (admin).
func (s *Server) MergePlayers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req mergePlayersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.MainPlayerID <= 0 || req.AliasPlayerID <= 0 {
		writeError(w, http.StatusBadRequest, "mainPlayerId and aliasPlayerId are required")
		return
	}

	if err := s.Players.MergePlayers(r.Context(), req.MainPlayerID, req.AliasPlayerID); err != nil {
		switch {
		case errors.Is(err, service.ErrPlayerMergeSelf):
			writeError(w, http.StatusBadRequest, "cannot merge player into itself")
		case errors.Is(err, service.ErrPlayerMergeMissing), errors.Is(err, service.ErrPlayerNotFound):
			writeError(w, http.StatusNotFound, "player not found")
		case errors.Is(err, service.ErrInvalidPlayer):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			log.Printf("merge players main=%d alias=%d: %v", req.MainPlayerID, req.AliasPlayerID, err)
			writeError(w, http.StatusInternalServerError, "merge failed")
		}
		return
	}
	debuglog.Printf("MergePlayers main=%d alias=%d", req.MainPlayerID, req.AliasPlayerID)
	w.WriteHeader(http.StatusNoContent)
}
