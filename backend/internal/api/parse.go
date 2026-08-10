package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia/parse"
	"github.com/c7d5a6/c7d5a6l/internal/model"
)

type parseRequest struct {
	URL string `json:"url"`
}

type parseResponse struct {
	Message    string               `json:"message"`
	Tournament model.TournamentPage `json:"tournament"`
}

type errorResponse struct {
	Error string `json:"error"`
}

var liquipediaClient = liquipedia.NewClient()

func ParseLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req parseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	canonical, err := liquipedia.ValidateURL(req.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	fetched, err := liquipediaClient.FetchPage(r.Context(), canonical)
	if err != nil {
		log.Printf("fetch %s: %v", canonical, err)
		writeError(w, http.StatusBadGateway, "failed to fetch liquipedia page")
		return
	}

	tournament, err := parse.Tournament(canonical, fetched.HTML)
	if err != nil {
		log.Printf("parse %s: %v", canonical, err)
		writeError(w, http.StatusInternalServerError, "failed to parse liquipedia page")
		return
	}

	_ = json.NewEncoder(w).Encode(parseResponse{
		Message:    "parsed",
		Tournament: tournament,
	})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: msg})
}
