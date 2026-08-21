package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/middleware"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

type authConfigResponse struct {
	BotID       string `json:"botId,omitempty"`
	BotUsername string `json:"botUsername"`
}

type telegramLoginResponse struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}

type meResponse struct {
	User model.User `json:"user"`
}

type patchMeRequest struct {
	Alias string `json:"alias"`
}

type listUsersResponse struct {
	Users []model.User `json:"users"`
}

type userWriteRequest struct {
	Alias            string  `json:"alias"`
	FirstName        string  `json:"firstName"`
	LastName         *string `json:"lastName"`
	PhotoURL         *string `json:"photoUrl"`
	TelegramUsername *string `json:"telegramUsername"`
	TelegramID       *int64  `json:"telegramId"`
	Role             string  `json:"role"`
}

type userResponse struct {
	User model.User `json:"user"`
}

func (req userWriteRequest) toService() service.UserWrite {
	return service.UserWrite{
		Alias:            req.Alias,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		PhotoURL:         req.PhotoURL,
		TelegramUsername: req.TelegramUsername,
		TelegramID:       req.TelegramID,
		Role:             req.Role,
	}
}

// AuthConfig returns public Telegram Login settings (never the token).
func (s *Server) AuthConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Auth == nil || !s.Auth.Configured() {
		writeError(w, http.StatusServiceUnavailable, "auth not configured")
		return
	}
	_ = json.NewEncoder(w).Encode(authConfigResponse{
		BotID:       s.Auth.BotID(),
		BotUsername: s.Auth.BotUsername(),
	})
}

// AuthTelegram verifies Telegram Login Widget data and issues a JWT.
func (s *Server) AuthTelegram(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Auth == nil {
		writeError(w, http.StatusInternalServerError, "auth service not configured")
		return
	}
	var payload model.TelegramAuthPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	token, user, err := s.Auth.LoginTelegram(r.Context(), payload)
	if err != nil {
		if errors.Is(err, service.ErrAuthConfig) {
			writeError(w, http.StatusServiceUnavailable, "auth not configured")
			return
		}
		if errors.Is(err, service.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "telegram auth failed")
			return
		}
		log.Printf("auth telegram: %v", err)
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	debuglog.Printf("AuthTelegram userId=%d", user.ID)
	_ = json.NewEncoder(w).Encode(telegramLoginResponse{Token: token, User: user})
}

// Me returns the current user (auth required).
func (s *Server) Me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Auth == nil {
		writeError(w, http.StatusInternalServerError, "auth service not configured")
		return
	}
	p, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := s.Auth.UserByID(r.Context(), p.UserID)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		log.Printf("me userId=%d: %v", p.UserID, err)
		writeError(w, http.StatusInternalServerError, "failed to load profile")
		return
	}
	user, err = s.userWithTitles(r.Context(), user)
	if err != nil {
		log.Printf("me titles userId=%d: %v", user.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to load profile")
		return
	}
	_ = json.NewEncoder(w).Encode(meResponse{User: user})
}

// AuthLogout acknowledges logout (client discards JWT).
func (s *Server) AuthLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

// PatchMe updates the caller's alias and returns a fresh JWT.
func (s *Server) PatchMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Auth == nil {
		writeError(w, http.StatusInternalServerError, "auth service not configured")
		return
	}
	p, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req patchMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	token, user, err := s.Auth.UpdateAlias(r.Context(), p.UserID, req.Alias)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if errors.Is(err, service.ErrInvalidAlias) {
			writeError(w, http.StatusBadRequest, "invalid alias")
			return
		}
		if errors.Is(err, service.ErrAliasTaken) {
			writeError(w, http.StatusConflict, "alias taken")
			return
		}
		log.Printf("patch me userId=%d: %v", p.UserID, err)
		writeError(w, http.StatusInternalServerError, "failed to update alias")
		return
	}
	debuglog.Printf("PatchMe userId=%d alias=%s", user.ID, user.Alias)
	user, err = s.userWithTitles(r.Context(), user)
	if err != nil {
		log.Printf("patch me titles userId=%d: %v", user.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to update alias")
		return
	}
	_ = json.NewEncoder(w).Encode(telegramLoginResponse{Token: token, User: user})
}

// ListUsers returns all users (admin only).
func (s *Server) ListUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Auth == nil {
		writeError(w, http.StatusInternalServerError, "auth service not configured")
		return
	}
	users, err := s.Auth.ListUsers(r.Context())
	if err != nil {
		log.Printf("list users: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	if users == nil {
		users = []model.User{}
	}
	debuglog.Printf("ListUsers count=%d", len(users))
	_ = json.NewEncoder(w).Encode(listUsersResponse{Users: users})
}

// CreateUser creates a user (admin only).
func (s *Server) CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Auth == nil {
		writeError(w, http.StatusInternalServerError, "auth service not configured")
		return
	}
	var req userWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	user, err := s.Auth.CreateUser(r.Context(), req.toService())
	if err != nil {
		writeUserWriteError(w, err, "failed to create user")
		return
	}
	debuglog.Printf("CreateUser userId=%d alias=%s", user.ID, user.Alias)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(userResponse{User: user})
}

// UpdateUser updates an existing user (admin only).
func (s *Server) UpdateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Auth == nil {
		writeError(w, http.StatusInternalServerError, "auth service not configured")
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req userWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	user, err := s.Auth.UpdateUser(r.Context(), id, req.toService())
	if err != nil {
		writeUserWriteError(w, err, "failed to update user")
		return
	}
	debuglog.Printf("UpdateUser userId=%d alias=%s", user.ID, user.Alias)
	_ = json.NewEncoder(w).Encode(userResponse{User: user})
}

func writeUserWriteError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrInvalidAlias):
		writeError(w, http.StatusBadRequest, "invalid alias")
	case errors.Is(err, service.ErrInvalidRole):
		writeError(w, http.StatusBadRequest, "invalid role")
	case errors.Is(err, service.ErrInvalidUser):
		writeError(w, http.StatusBadRequest, "invalid user")
	case errors.Is(err, service.ErrInvalidTelegramID):
		writeError(w, http.StatusBadRequest, "invalid telegram id")
	case errors.Is(err, service.ErrAliasTaken):
		writeError(w, http.StatusConflict, "alias taken")
	case errors.Is(err, service.ErrTelegramIDTaken):
		writeError(w, http.StatusConflict, "telegram id taken")
	case errors.Is(err, service.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	default:
		log.Printf("user write: %v", err)
		writeError(w, http.StatusInternalServerError, fallback)
	}
}

func (s *Server) userWithTitles(ctx context.Context, user model.User) (model.User, error) {
	if s.Titles == nil {
		user.Titles = []model.UserTitle{}
		return user, nil
	}
	titles, err := s.Titles.ListByUserID(ctx, user.ID)
	if err != nil {
		return user, err
	}
	user.Titles = titles
	return user, nil
}
