package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

type adminTournamentListResponse struct {
	Items    []model.AdminTournament `json:"items"`
	Total    int                     `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"pageSize"`
}

type syncQueueResponse struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type getTournamentResponse struct {
	Message        string                 `json:"message"`
	Tournament     model.TournamentPage   `json:"tournament"`
	TournamentSync model.TournamentSync   `json:"tournamentSync"`
}

// ListAdminTournaments returns a paginated admin tournament list (queue + stored).
func (s *Server) ListAdminTournaments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	filter := strings.TrimSpace(r.URL.Query().Get("tab"))
	if filter == "" {
		filter = model.AdminFilterQueue
	}
	page := queryPositiveInt(r, "page", 1)
	pageSize := queryPositiveInt(r, "pageSize", 20)

	list, err := s.Tournaments.ListAdmin(r.Context(), filter, page, pageSize)
	if err != nil {
		if strings.Contains(err.Error(), "unknown tournament filter") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("list admin tournaments: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list tournaments")
		return
	}
	debuglog.Printf("ListAdminTournaments tab=%s page=%d count=%d total=%d", filter, list.Page, len(list.Items), list.Total)
	_ = json.NewEncoder(w).Encode(adminTournamentListResponse{
		Items:    list.Items,
		Total:    list.Total,
		Page:     list.Page,
		PageSize: list.PageSize,
	})
}

// SyncTournamentQueue fetches Liquipedia Recent Tournaments into the queue.
func (s *Server) SyncTournamentQueue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Liquipedia == nil {
		writeError(w, http.StatusInternalServerError, "liquipedia client not configured")
		return
	}
	fetched, err := s.Liquipedia.FetchPage(r.Context(), liquipedia.RecentTournamentsURL)
	if err != nil {
		log.Printf("sync tournament queue fetch: %v", err)
		writeError(w, http.StatusBadGateway, "failed to fetch liquipedia listing")
		return
	}
	n, err := s.Tournaments.SyncRecentFromHTML(r.Context(), fetched.HTML)
	if err != nil {
		log.Printf("sync tournament queue save: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to save tournament queue")
		return
	}
	_ = json.NewEncoder(w).Encode(syncQueueResponse{
		Message: fmt.Sprintf("queued %d listing(s)", n),
		Count:   n,
	})
}

// IgnoreTournamentQueue marks a queue row disabled.
func (s *Server) IgnoreTournamentQueue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.Tournaments.IgnoreQueueItem(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrQueueNotFound) {
			writeError(w, http.StatusNotFound, "queue item not found")
			return
		}
		log.Printf("ignore tournament queue %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to ignore tournament")
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ignored"})
}

// ParseTournamentQueue fetches, parses, and saves a queue item.
func (s *Server) ParseTournamentQueue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if s.Liquipedia == nil {
		writeError(w, http.StatusInternalServerError, "liquipedia client not configured")
		return
	}

	page, sync, queued, tournamentID, err := s.parseQueueItem(r, id)
	if err != nil {
		if errors.Is(err, service.ErrQueueNotFound) {
			writeError(w, http.StatusNotFound, "queue item not found")
			return
		}
		if err.Error() == "failed to fetch liquipedia page" {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		log.Printf("parse tournament queue %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to parse tournament")
		return
	}
	msg := "parsed"
	if queued > 0 {
		msg = fmt.Sprintf("parsed — queued %d player import(s)", queued)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message":        msg,
		"tournamentId":   tournamentID,
		"tournament":     page,
		"tournamentSync": sync,
		"importQueued":   queued,
	})
}

func (s *Server) parseQueueItem(r *http.Request, id int64) (model.TournamentPage, model.TournamentSync, int, int64, error) {
	link, err := s.Tournaments.QueueLinkByID(r.Context(), id)
	if err != nil {
		return model.TournamentPage{}, model.TournamentSync{}, 0, 0, err
	}
	fetched, err := s.Liquipedia.FetchPage(r.Context(), link)
	if err != nil {
		log.Printf("parse tournament queue fetch %s: %v", link, err)
		return model.TournamentPage{}, model.TournamentSync{}, 0, 0, fmt.Errorf("failed to fetch liquipedia page")
	}
	return s.Tournaments.ParseQueueFromHTML(r.Context(), id, fetched.HTML)
}

// GetTournament returns a stored tournament for the admin detail view.
func (s *Server) GetTournament(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	page, sync, err := s.Tournaments.GetPageByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		log.Printf("get tournament %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to load tournament")
		return
	}
	_ = json.NewEncoder(w).Encode(getTournamentResponse{
		Message:        "stored",
		Tournament:     page,
		TournamentSync: sync,
	})
}

func queryPositiveInt(r *http.Request, name string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
