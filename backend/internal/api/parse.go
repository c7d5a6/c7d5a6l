package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

type parseRequest struct {
	URL string `json:"url"`
}

type parseResponse struct {
	Message    string                `json:"message"`
	PageType   parse.PageType        `json:"pageType"`
	Tournament *model.TournamentPage `json:"tournament,omitempty"`
	Player     *model.PlayerPage     `json:"player,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

var liquipediaClient = liquipedia.NewClient()

func ParseLink(w http.ResponseWriter, r *http.Request) {
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

	fetched, err := liquipediaClient.FetchPage(r.Context(), canonical)
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
		debuglog.Printf("ParseLink tournament name=%s participants=%d results=%d",
			debuglog.Str(tournament.Name), len(tournament.Participants), len(tournament.Results))
		_ = json.NewEncoder(w).Encode(parseResponse{
			Message:    "parsed",
			PageType:   pageType,
			Tournament: &tournament,
		})
	case parse.PageTypePlayer:
		player, err := parse.Player(canonical, fetched.HTML)
		if err != nil {
			log.Printf("parse player %s: %v", canonical, err)
			writeError(w, http.StatusInternalServerError, "failed to parse liquipedia page")
			return
		}
		debuglog.Printf("ParseLink player name=%s race=%s ids=%d",
			debuglog.Str(player.Name), debuglog.Str(player.PreferredRace), len(player.IDs))
		_ = json.NewEncoder(w).Encode(parseResponse{
			Message:  "parsed",
			PageType: pageType,
			Player:   &player,
		})
	default:
		debuglog.Printf("ParseLink unsupported pageType=%s", pageType)
		writeError(w, http.StatusBadRequest, "not a tournament or player page; only those URLs are supported")
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	debuglog.Printf("ParseLink error status=%d msg=%s", status, msg)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: msg})
}
