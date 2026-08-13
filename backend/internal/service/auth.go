package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrAuthConfig   = errors.New("auth not configured")
	ErrAliasTaken   = errors.New("alias taken")
	ErrInvalidAlias = errors.New("invalid alias")
)

const maxAliasLen = 64

// AuthConfig holds Telegram Login + JWT settings.
type AuthConfig struct {
	BotToken         string
	BotID            string
	BotUsername      string
	JWTSecret        string
	JWTTTL           time.Duration
	AdminTelegramIDs map[int64]struct{}
	AuthMaxAge       time.Duration
}

// Auth handles Telegram login, JWT issue/parse, and /me.
type Auth struct {
	db   *sql.DB
	repo *repository.User
	cfg  AuthConfig
}

func NewAuth(db *sql.DB, repo *repository.User, cfg AuthConfig) *Auth {
	if cfg.JWTTTL <= 0 {
		cfg.JWTTTL = 7 * 24 * time.Hour
	}
	if cfg.AuthMaxAge <= 0 {
		cfg.AuthMaxAge = DefaultTelegramAuthMaxAge
	}
	if cfg.AdminTelegramIDs == nil {
		cfg.AdminTelegramIDs = map[int64]struct{}{}
	}
	return &Auth{db: db, repo: repo, cfg: cfg}
}

// BotID returns the numeric bot id for the frontend (never the token).
func (s *Auth) BotID() string {
	return s.cfg.BotID
}

// BotUsername returns the bot username without @ (for the Login Widget).
func (s *Auth) BotUsername() string {
	return strings.TrimPrefix(strings.TrimSpace(s.cfg.BotUsername), "@")
}

// Configured reports whether login can run (widget needs bot username).
func (s *Auth) Configured() bool {
	return s.cfg.BotToken != "" && s.BotUsername() != "" && s.cfg.JWTSecret != ""
}

type authClaims struct {
	Role  string `json:"role"`
	Alias string `json:"alias"`
	jwt.RegisteredClaims
}

// LoginTelegram verifies the widget payload, upserts the user, and returns a JWT.
func (s *Auth) LoginTelegram(ctx context.Context, payload model.TelegramAuthPayload) (token string, user model.User, err error) {
	if !s.Configured() {
		return "", model.User{}, ErrAuthConfig
	}
	if err := VerifyTelegramAuth(s.cfg.BotToken, payload, s.cfg.AuthMaxAge); err != nil {
		return "", model.User{}, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", model.User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	existing, err := s.repo.GetByTelegramID(ctx, tx, payload.ID)
	if err != nil {
		return "", model.User{}, err
	}

	role := model.RoleUser
	if _, ok := s.cfg.AdminTelegramIDs[payload.ID]; ok {
		role = model.RoleAdmin
	}

	username := strings.TrimSpace(payload.Username)
	firstName := strings.TrimSpace(payload.FirstName)
	lastName := strings.TrimSpace(payload.LastName)
	photo := strings.TrimSpace(payload.PhotoURL)

	if existing == nil {
		alias, err := s.pickAlias(ctx, tx, username, firstName, payload.ID, 0)
		if err != nil {
			return "", model.User{}, err
		}
		u := model.User{
			Alias:            alias,
			TelegramID:       int64Ptr(payload.ID),
			TelegramUsername: strPtrOrNil(username),
			FirstName:        firstName,
			LastName:         strPtrOrNil(lastName),
			PhotoURL:         strPtrOrNil(photo),
			Role:             role,
			LastLoginAt:      &now,
		}
		id, err := s.repo.Insert(ctx, tx, u)
		if err != nil {
			return "", model.User{}, err
		}
		created, err := s.repo.GetByID(ctx, tx, id)
		if err != nil || created == nil {
			return "", model.User{}, fmt.Errorf("reload user after insert: %w", err)
		}
		user = *created
	} else {
		alias := strings.TrimSpace(existing.Alias)
		// Only fill alias when missing; never overwrite a stored alias on login.
		if alias == "" {
			picked, err := s.pickAlias(ctx, tx, username, firstName, payload.ID, existing.ID)
			if err != nil {
				return "", model.User{}, err
			}
			alias = picked
		}
		keepRole := existing.Role
		if role == model.RoleAdmin {
			keepRole = model.RoleAdmin
		}
		u := *existing
		u.Alias = alias
		u.TelegramUsername = strPtrOrNil(username)
		u.FirstName = firstName
		u.LastName = strPtrOrNil(lastName)
		u.PhotoURL = strPtrOrNil(photo)
		u.Role = keepRole
		u.LastLoginAt = &now
		if err := s.repo.UpdateTelegramProfile(ctx, tx, u); err != nil {
			return "", model.User{}, err
		}
		updated, err := s.repo.GetByID(ctx, tx, existing.ID)
		if err != nil || updated == nil {
			return "", model.User{}, fmt.Errorf("reload user after update: %w", err)
		}
		user = *updated
	}

	if err := tx.Commit(); err != nil {
		return "", model.User{}, fmt.Errorf("commit: %w", err)
	}

	token, err = s.issueJWT(user)
	if err != nil {
		return "", model.User{}, err
	}
	debuglog.Printf("auth.LoginTelegram userId=%d telegramId=%v role=%s", user.ID, user.TelegramID, user.Role)
	return token, user, nil
}

// UserByID returns the user or ErrUnauthorized if missing.
func (s *Auth) UserByID(ctx context.Context, id int64) (model.User, error) {
	if id <= 0 {
		return model.User{}, ErrUnauthorized
	}
	u, err := s.repo.GetByID(ctx, s.db, id)
	if err != nil {
		return model.User{}, err
	}
	if u == nil {
		return model.User{}, ErrUnauthorized
	}
	return *u, nil
}

// UpdateAlias changes the caller's display alias and re-issues a JWT.
func (s *Auth) UpdateAlias(ctx context.Context, userID int64, alias string) (token string, user model.User, err error) {
	alias = strings.TrimSpace(alias)
	if alias == "" || len(alias) > maxAliasLen {
		return "", model.User{}, ErrInvalidAlias
	}
	if userID <= 0 {
		return "", model.User{}, ErrUnauthorized
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", model.User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	existing, err := s.repo.GetByID(ctx, tx, userID)
	if err != nil {
		return "", model.User{}, err
	}
	if existing == nil {
		return "", model.User{}, ErrUnauthorized
	}

	taken, err := s.repo.AliasTaken(ctx, tx, alias, userID)
	if err != nil {
		return "", model.User{}, err
	}
	if taken {
		return "", model.User{}, ErrAliasTaken
	}

	if err := s.repo.UpdateAlias(ctx, tx, userID, alias); err != nil {
		return "", model.User{}, err
	}
	updated, err := s.repo.GetByID(ctx, tx, userID)
	if err != nil || updated == nil {
		return "", model.User{}, fmt.Errorf("reload user after alias update: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", model.User{}, fmt.Errorf("commit: %w", err)
	}

	token, err = s.issueJWT(*updated)
	if err != nil {
		return "", model.User{}, err
	}
	debuglog.Printf("auth.UpdateAlias userId=%d alias=%s", updated.ID, updated.Alias)
	return token, *updated, nil
}

// ListUsers returns all users (admin roster).
func (s *Auth) ListUsers(ctx context.Context) ([]model.User, error) {
	return s.repo.ListAll(ctx, s.db)
}

// CreateUser creates an alias-only USER account (no Telegram).
func (s *Auth) CreateUser(ctx context.Context, alias string) (model.User, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" || len(alias) > maxAliasLen {
		return model.User{}, ErrInvalidAlias
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	taken, err := s.repo.AliasTaken(ctx, tx, alias, 0)
	if err != nil {
		return model.User{}, err
	}
	if taken {
		return model.User{}, ErrAliasTaken
	}

	id, err := s.repo.Insert(ctx, tx, model.User{
		Alias:     alias,
		FirstName: alias,
		Role:      model.RoleUser,
	})
	if err != nil {
		return model.User{}, err
	}
	created, err := s.repo.GetByID(ctx, tx, id)
	if err != nil || created == nil {
		return model.User{}, fmt.Errorf("reload user after create: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return model.User{}, fmt.Errorf("commit: %w", err)
	}
	debuglog.Printf("auth.CreateUser userId=%d alias=%s", created.ID, created.Alias)
	return *created, nil
}

// ParseAccessToken validates JWT and returns user id + role for middleware.
func (s *Auth) ParseAccessToken(accessToken string) (userID int64, role, alias string, err error) {
	claims, err := s.parseJWT(accessToken)
	if err != nil {
		return 0, "", "", ErrUnauthorized
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || id <= 0 {
		return 0, "", "", ErrUnauthorized
	}
	return id, claims.Role, claims.Alias, nil
}

func (s *Auth) issueJWT(user model.User) (string, error) {
	now := time.Now()
	claims := authClaims{
		Role:  user.Role,
		Alias: user.Alias,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWTTTL)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

func (s *Auth) parseJWT(accessToken string) (*authClaims, error) {
	if accessToken == "" || s.cfg.JWTSecret == "" {
		return nil, ErrUnauthorized
	}
	parsed, err := jwt.ParseWithClaims(accessToken, &authClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, ErrUnauthorized
	}
	claims, ok := parsed.Claims.(*authClaims)
	if !ok || claims.Subject == "" {
		return nil, ErrUnauthorized
	}
	return claims, nil
}

func (s *Auth) pickAlias(ctx context.Context, tx *sql.Tx, username, firstName string, telegramID, excludeID int64) (string, error) {
	candidates := make([]string, 0, 3)
	if username != "" {
		candidates = append(candidates, "@"+username)
	}
	if firstName != "" {
		candidates = append(candidates, firstName)
	}
	candidates = append(candidates, fmt.Sprintf("tg_%d", telegramID))

	for _, alias := range candidates {
		taken, err := s.repo.AliasTaken(ctx, tx, alias, excludeID)
		if err != nil {
			return "", err
		}
		if !taken {
			return alias, nil
		}
	}
	return fmt.Sprintf("tg_%d", telegramID), nil
}

func strPtrOrNil(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func int64Ptr(v int64) *int64 {
	return &v
}

// ParseAdminTelegramIDs parses comma-separated Telegram user IDs.
func ParseAdminTelegramIDs(raw string) map[int64]struct{} {
	out := map[int64]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}

// ParseDurationEnv parses a duration string; empty returns fallback.
func ParseDurationEnv(raw string, fallback time.Duration) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
