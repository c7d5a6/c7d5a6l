package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

type getPlayerResponse struct {
	Player model.PlayerDetail `json:"player"`
}

// GetPlayer returns the player dossier (identity, race ratings, winrates, season matches).
func (s *Server) GetPlayer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid player id")
		return
	}

	detail, err := s.Seasons.GetPlayerDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPlayerNotFound) {
			writeError(w, http.StatusNotFound, "player not found")
			return
		}
		log.Printf("get player %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to load player")
		return
	}

	debuglog.Printf("GetPlayer id=%d races=%d seasonMatches=%d", id, len(detail.Races), len(detail.SeasonMatches))
	_ = json.NewEncoder(w).Encode(getPlayerResponse{Player: *detail})
}
