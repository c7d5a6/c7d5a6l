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
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

const maxTitleFormMemory = 1 << 20

type listTitlesResponse struct {
	Titles []model.UserTitle `json:"titles"`
}

type titleResponse struct {
	Title model.UserTitle `json:"title"`
}

// ListUserTitles returns all titles (admin).
func (s *Server) ListUserTitles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Titles == nil {
		writeError(w, http.StatusInternalServerError, "title service not configured")
		return
	}
	list, err := s.Titles.ListAll(r.Context())
	if err != nil {
		log.Printf("list titles: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list titles")
		return
	}
	if list == nil {
		list = []model.UserTitle{}
	}
	debuglog.Printf("ListUserTitles count=%d", len(list))
	_ = json.NewEncoder(w).Encode(listTitlesResponse{Titles: list})
}

// CreateUserTitle creates a title (admin, multipart).
func (s *Server) CreateUserTitle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Titles == nil {
		writeError(w, http.StatusInternalServerError, "title service not configured")
		return
	}
	params, err := parseTitleForm(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	title, err := s.Titles.Create(r.Context(), params)
	if err != nil {
		writeTitleErr(w, err)
		return
	}
	debuglog.Printf("CreateUserTitle id=%d", title.ID)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(titleResponse{Title: title})
}

// UpdateUserTitle patches a title (admin, multipart).
func (s *Server) UpdateUserTitle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Titles == nil {
		writeError(w, http.StatusInternalServerError, "title service not configured")
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	params, err := parseTitleForm(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	title, err := s.Titles.Update(r.Context(), id, params)
	if err != nil {
		writeTitleErr(w, err)
		return
	}
	debuglog.Printf("UpdateUserTitle id=%d", title.ID)
	_ = json.NewEncoder(w).Encode(titleResponse{Title: title})
}

// DeleteUserTitle removes a title (admin).
func (s *Server) DeleteUserTitle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Titles == nil {
		writeError(w, http.StatusInternalServerError, "title service not configured")
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.Titles.Delete(r.Context(), id); err != nil {
		writeTitleErr(w, err)
		return
	}
	debuglog.Printf("DeleteUserTitle id=%d", id)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

// GetUserTitleImage serves title art (public).
func (s *Server) GetUserTitleImage(w http.ResponseWriter, r *http.Request) {
	if s.Titles == nil {
		writeError(w, http.StatusInternalServerError, "title service not configured")
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	data, ctype, err := s.Titles.GetImage(r.Context(), id)
	if err != nil {
		log.Printf("title image %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to load title image")
		return
	}
	if len(data) == 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func parseTitleForm(r *http.Request) (service.TitleParams, error) {
	if err := r.ParseMultipartForm(maxTitleFormMemory); err != nil {
		return service.TitleParams{}, errors.New("invalid multipart form")
	}
	userID, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("userId")), 10, 64)
	if err != nil {
		return service.TitleParams{}, errors.New("invalid userId")
	}
	params := service.TitleParams{
		UserID:  userID,
		Kind:    r.FormValue("kind"),
		Name:    r.FormValue("name"),
		ImageOp: service.ImageUnchanged,
	}
	if raw := strings.TrimSpace(r.FormValue("fantasyLeagueId")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 {
			return service.TitleParams{}, errors.New("invalid fantasyLeagueId")
		}
		if n > 0 {
			params.FantasyLeagueID = &n
		}
	}
	file, hdr, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		data, readErr := io.ReadAll(io.LimitReader(file, service.MaxTitleImageLen+1))
		if readErr != nil {
			return service.TitleParams{}, errors.New("failed to read image")
		}
		if len(data) == 0 || hdr.Size == 0 {
			params.ImageOp = service.ImageClear
		} else {
			params.ImageOp = service.ImageSet
			params.Image = data
			params.ImageMime = hdr.Header.Get("Content-Type")
		}
	} else if !errors.Is(err, http.ErrMissingFile) {
		return service.TitleParams{}, errors.New("invalid image")
	}
	return params, nil
}

func writeTitleErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrTitleNotFound):
		writeError(w, http.StatusNotFound, "title not found")
	case errors.Is(err, service.ErrTitleConflict):
		writeError(w, http.StatusConflict, "a title is already linked to that league")
	case errors.Is(err, service.ErrTitleInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("title error: %v", err)
		writeError(w, http.StatusInternalServerError, "title request failed")
	}
}

func (s *Server) attachTeamTitles(r *http.Request, teams []model.FantasyTeamRow) []model.FantasyTeamRow {
	if s.Titles == nil {
		for i := range teams {
			if teams[i].Titles == nil {
				teams[i].Titles = []model.UserTitle{}
			}
		}
		return teams
	}
	if err := s.Titles.AttachToTeams(r.Context(), teams); err != nil {
		log.Printf("attach team titles: %v", err)
		for i := range teams {
			if teams[i].Titles == nil {
				teams[i].Titles = []model.UserTitle{}
			}
		}
	}
	return teams
}

func (s *Server) attachOneTeamTitles(r *http.Request, team model.FantasyTeamRow) model.FantasyTeamRow {
	out := s.attachTeamTitles(r, []model.FantasyTeamRow{team})
	return out[0]
}
