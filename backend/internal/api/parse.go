package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

type Server struct {
	Liquipedia  *liquipedia.Client
	Players     *service.Player
	Tournaments *service.Tournament
	Fantasy     *service.Fantasy
	Auth        *service.Auth
}

type parseRequest struct {
	URL string `json:"url"`
}

type parseResponse struct {
	Message        string                  `json:"message"`
	PageType       parse.PageType          `json:"pageType"`
	Tournament     *model.TournamentPage   `json:"tournament,omitempty"`
	TournamentSync *model.TournamentSync   `json:"tournamentSync,omitempty"`
	Player         *model.PlayerPage       `json:"player,omitempty"`
	PlayerSync     *model.PlayerSync       `json:"playerSync,omitempty"`
}

type savePlayerResponse struct {
	Message    string           `json:"message"`
	Player     model.PlayerPage `json:"player"`
	PlayerSync model.PlayerSync `json:"playerSync"`
}

type saveTournamentResponse struct {
	Message        string               `json:"message"`
	Tournament     model.TournamentPage `json:"tournament"`
	TournamentSync model.TournamentSync `json:"tournamentSync"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) ParseLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req parseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debuglog.Printf("ParseLink invalid json: %v", err)
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	debuglog.Printf("ParseLink request url=%q", req.URL)

	canonical, err := liquipedia.ValidateURL(req.URL)
	if err != nil {
		debuglog.Printf("ParseLink validate failed: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	debuglog.Printf("ParseLink canonical=%s", canonical)

	fetched, err := s.Liquipedia.FetchPage(r.Context(), canonical)
	if err != nil {
		log.Printf("fetch %s: %v", canonical, err)
		writeError(w, http.StatusBadGateway, "failed to fetch liquipedia page")
		return
	}
	debuglog.Printf("ParseLink fetched title=%q htmlBytes=%d", fetched.Title, len(fetched.HTML))

	pageType, err := parse.PageTypeFromHTML(fetched.HTML)
	if err != nil {
		log.Printf("page type %s: %v", canonical, err)
		writeError(w, http.StatusInternalServerError, "failed to parse liquipedia page")
		return
	}
	debuglog.Printf("ParseLink pageType=%s", pageType)

	switch pageType {
	case parse.PageTypeTournament:
		tournament, err := parse.Tournament(canonical, fetched.HTML)
		if err != nil {
			log.Printf("parse tournament %s: %v", canonical, err)
			writeError(w, http.StatusInternalServerError, "failed to parse liquipedia page")
			return
		}
		sync, err := s.Tournaments.SyncStatus(r.Context(), tournament)
		if err != nil {
			log.Printf("tournament sync %s: %v", canonical, err)
			writeError(w, http.StatusInternalServerError, "failed to check tournament in database")
			return
		}
		debuglog.Printf("ParseLink tournament name=%s participants=%d results=%d exists=%v same=%v action=%s",
			debuglog.Str(tournament.Name), len(tournament.Participants), len(tournament.Results),
			sync.Exists, sync.Same, sync.Action)
		_ = json.NewEncoder(w).Encode(parseResponse{
			Message:        "parsed",
			PageType:       pageType,
			Tournament:     &tournament,
			TournamentSync: &sync,
		})
	case parse.PageTypePlayer:
		player, err := parse.Player(canonical, fetched.HTML)
		if err != nil {
			log.Printf("parse player %s: %v", canonical, err)
			writeError(w, http.StatusInternalServerError, "failed to parse liquipedia page")
			return
		}
		sync, err := s.Players.SyncStatus(r.Context(), player)
		if err != nil {
			log.Printf("player sync %s: %v", canonical, err)
			writeError(w, http.StatusInternalServerError, "failed to check player in database")
			return
		}
		debuglog.Printf("ParseLink player name=%s race=%s ids=%d exists=%v same=%v action=%s",
			debuglog.Str(player.Name), debuglog.Str(player.PreferredRace), len(player.IDs),
			sync.Exists, sync.Same, sync.Action)
		_ = json.NewEncoder(w).Encode(parseResponse{
			Message:    "parsed",
			PageType:   pageType,
			Player:     &player,
			PlayerSync: &sync,
		})
	default:
		debuglog.Printf("ParseLink unsupported pageType=%s", pageType)
		writeError(w, http.StatusBadRequest, "not a tournament or player page; only those URLs are supported")
	}
}

type listPlayersResponse struct {
	Players []model.PlayerRaceEntry `json:"players"`
}

// ListPlayers returns player_race rows merged with player info, sorted by elo.
func (s *Server) ListPlayers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	players, err := s.Players.ListRaceEntries(r.Context())
	if err != nil {
		log.Printf("list players: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list players")
		return
	}
	if players == nil {
		players = []model.PlayerRaceEntry{}
	}
	debuglog.Printf("ListPlayers count=%d", len(players))
	_ = json.NewEncoder(w).Encode(listPlayersResponse{Players: players})
}

type patchPlayerRaceRequest struct {
	Elo *float64 `json:"elo"`
}

type patchPlayerRaceResponse struct {
	Player model.PlayerRaceEntry `json:"player"`
}

// PatchPlayerRace updates elo for one player_race row (admin). Does not touch fantasy costs.
func (s *Server) PatchPlayerRace(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req patchPlayerRaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.Elo == nil {
		writeError(w, http.StatusBadRequest, "elo is required")
		return
	}

	entry, err := s.Players.UpdateRaceElo(r.Context(), id, *req.Elo)
	if err != nil {
		if errors.Is(err, service.ErrPlayerNotFound) {
			writeError(w, http.StatusNotFound, "player race not found")
			return
		}
		if errors.Is(err, service.ErrInvalidPlayer) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("patch player race %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to update elo")
		return
	}
	debuglog.Printf("PatchPlayerRace id=%d elo=%.0f", entry.PlayerRaceID, entry.Elo)
	_ = json.NewEncoder(w).Encode(patchPlayerRaceResponse{Player: entry})
}

// SavePlayer persists the player JSON from the client (no Liquipedia page re-fetch).
// Portrait image bytes are downloaded from portraitUrl when missing/changed.
func (s *Server) SavePlayer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var page model.PlayerPage
	if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if page.Link == "" {
		writeError(w, http.StatusBadRequest, "player link is required")
		return
	}
	canonical, err := liquipedia.ValidateURL(page.Link)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page.Link = canonical
	if page.IDs == nil {
		page.IDs = []string{}
	}
	if page.PortraitURL != nil {
		trimmed := strings.TrimSpace(*page.PortraitURL)
		if trimmed == "" {
			page.PortraitURL = nil
		} else if src, err := liquipedia.ValidateAssetURL(trimmed); err != nil {
			writeError(w, http.StatusBadRequest, "invalid portraitUrl")
			return
		} else {
			page.PortraitURL = &src
		}
	}

	saved, sync, err := s.Players.Save(r.Context(), page)
	if err != nil {
		log.Printf("save player %s: %v", page.Link, err)
		writeError(w, http.StatusInternalServerError, "failed to save player")
		return
	}
	debuglog.Printf("SavePlayer ok link=%s exists=%v same=%v hasPortrait=%v", saved.Link, sync.Exists, sync.Same, saved.HasPortrait)

	_ = json.NewEncoder(w).Encode(savePlayerResponse{
		Message:    "saved",
		Player:     saved,
		PlayerSync: sync,
	})
}

// GetPlayerLookup returns a stored player by Liquipedia link (DB only, no re-fetch).
func (s *Server) GetPlayerLookup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	link := strings.TrimSpace(r.URL.Query().Get("link"))
	if link == "" {
		writeError(w, http.StatusBadRequest, "link query parameter is required")
		return
	}
	canonical, err := liquipedia.ValidateURL(link)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	page, err := s.Players.GetByLink(r.Context(), canonical)
	if err != nil {
		log.Printf("player lookup %s: %v", canonical, err)
		writeError(w, http.StatusInternalServerError, "failed to load player")
		return
	}
	if page == nil {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}
	if page.IDs == nil {
		page.IDs = []string{}
	}
	debuglog.Printf("GetPlayerLookup ok link=%s hasPortrait=%v", page.Link, page.HasPortrait)
	_ = json.NewEncoder(w).Encode(page)
}

// GetPlayerPortrait serves a cached portrait blob for a player Liquipedia link.
func (s *Server) GetPlayerPortrait(w http.ResponseWriter, r *http.Request) {
	link := strings.TrimSpace(r.URL.Query().Get("link"))
	if link == "" {
		writeError(w, http.StatusBadRequest, "link query parameter is required")
		return
	}
	canonical, err := liquipedia.ValidateURL(link)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	data, ctype, err := s.Players.Portrait(r.Context(), canonical)
	if err != nil {
		log.Printf("player portrait %s: %v", canonical, err)
		writeError(w, http.StatusInternalServerError, "failed to load portrait")
		return
	}
	if len(data) == 0 {
		http.NotFound(w, r)
		return
	}
	// Public cacheable image: * so CDNs/browsers can reuse the response from any origin
	// (hover used to fetch() this; img-only still benefits if anything reads the blob).
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}


// ListTournaments returns stored tournaments for pickers (e.g. create fantasy league).
func (s *Server) ListTournaments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	list, err := s.Tournaments.ListSummaries(r.Context())
	if err != nil {
		log.Printf("list tournaments: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list tournaments")
		return
	}
	if list == nil {
		list = []model.TournamentSummary{}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"tournaments": list})
}

// SaveTournament persists tournament JSON from the client; missing players are fetched by link.
func (s *Server) SaveTournament(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var page model.TournamentPage
	if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if page.Link == "" {
		writeError(w, http.StatusBadRequest, "tournament link is required")
		return
	}
	canonical, err := liquipedia.ValidateURL(page.Link)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page.Link = canonical
	if page.Participants == nil {
		page.Participants = []model.Participant{}
	}
	if page.Results == nil {
		page.Results = []model.Result{}
	}

	saved, sync, err := s.Tournaments.Save(r.Context(), page)
	if err != nil {
		log.Printf("save tournament %s: %v", page.Link, err)
		writeError(w, http.StatusInternalServerError, "failed to save tournament")
		return
	}
	debuglog.Printf("SaveTournament ok link=%s exists=%v same=%v", saved.Link, sync.Exists, sync.Same)

	_ = json.NewEncoder(w).Encode(saveTournamentResponse{
		Message:        "saved",
		Tournament:     saved,
		TournamentSync: sync,
	})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	debuglog.Printf("api error status=%d msg=%s", status, msg)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: msg})
}
