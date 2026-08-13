package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
)

var (
	ErrTitleNotFound = errors.New("title not found")
	ErrTitleInvalid  = errors.New("invalid title")
	ErrTitleConflict = errors.New("title already exists for that league")
)

const (
	MaxTitleNameLen  = 80
	MaxTitleImageLen = 512 * 1024
)

// ImageUnchanged / ImageSet / ImageClear control blob updates.
const (
	ImageUnchanged = 0
	ImageSet       = 1
	ImageClear     = 2
)

// TitleParams is create/update input.
type TitleParams struct {
	UserID          int64
	Kind            string
	Name            string
	FantasyLeagueID *int64
	Image           []byte
	ImageMime       string
	ImageOp         int
}

// Title orchestrates user title awards.
type Title struct {
	db      *sql.DB
	repo    *repository.Title
	users   *repository.User
	fantasy *repository.Fantasy
}

func NewTitle(db *sql.DB, repo *repository.Title, users *repository.User, fantasy *repository.Fantasy) *Title {
	return &Title{db: db, repo: repo, users: users, fantasy: fantasy}
}

// ListAll returns every title.
func (s *Title) ListAll(ctx context.Context) ([]model.UserTitle, error) {
	return s.repo.ListAll(ctx, s.db)
}

// ListByUserID returns titles for a user.
func (s *Title) ListByUserID(ctx context.Context, userID int64) ([]model.UserTitle, error) {
	list, err := s.repo.ListByUserID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []model.UserTitle{}
	}
	return list, nil
}

// AttachToTeams fills Titles on each team from a batched lookup.
func (s *Title) AttachToTeams(ctx context.Context, teams []model.FantasyTeamRow) error {
	ids := make([]int64, 0, len(teams))
	seen := make(map[int64]struct{}, len(teams))
	for i := range teams {
		if teams[i].Titles == nil {
			teams[i].Titles = []model.UserTitle{}
		}
		uid := teams[i].UserID
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		ids = append(ids, uid)
	}
	byUser, err := s.repo.ListByUserIDs(ctx, s.db, ids)
	if err != nil {
		return err
	}
	for i := range teams {
		if t, ok := byUser[teams[i].UserID]; ok {
			teams[i].Titles = t
		} else {
			teams[i].Titles = []model.UserTitle{}
		}
	}
	return nil
}

// GetImage returns cached title art, or nil when missing.
func (s *Title) GetImage(ctx context.Context, id int64) ([]byte, string, error) {
	if id <= 0 {
		return nil, "", ErrTitleNotFound
	}
	return s.repo.GetImage(ctx, s.db, id)
}

// Create inserts a title.
func (s *Title) Create(ctx context.Context, p TitleParams) (model.UserTitle, error) {
	norm, err := s.normalize(ctx, p)
	if err != nil {
		return model.UserTitle{}, err
	}
	id, err := s.repo.Insert(ctx, s.db, model.UserTitle{
		UserID:          norm.UserID,
		Kind:            norm.Kind,
		Name:            norm.Name,
		FantasyLeagueID: norm.FantasyLeagueID,
	}, norm.Image, norm.ImageMime)
	if err != nil {
		return model.UserTitle{}, mapTitleErr(err)
	}
	got, err := s.repo.GetByID(ctx, s.db, id)
	if err != nil || got == nil {
		return model.UserTitle{}, fmt.Errorf("reload title: %w", err)
	}
	debuglog.Printf("service.Title.Create id=%d userId=%d kind=%s", got.ID, got.UserID, got.Kind)
	return *got, nil
}

// Update patches a title.
func (s *Title) Update(ctx context.Context, id int64, p TitleParams) (model.UserTitle, error) {
	if id <= 0 {
		return model.UserTitle{}, ErrTitleNotFound
	}
	existing, err := s.repo.GetByID(ctx, s.db, id)
	if err != nil {
		return model.UserTitle{}, err
	}
	if existing == nil {
		return model.UserTitle{}, ErrTitleNotFound
	}
	norm, err := s.normalize(ctx, p)
	if err != nil {
		return model.UserTitle{}, err
	}
	if err := s.repo.Update(ctx, s.db, model.UserTitle{
		ID:              id,
		UserID:          norm.UserID,
		Kind:            norm.Kind,
		Name:            norm.Name,
		FantasyLeagueID: norm.FantasyLeagueID,
	}, norm.Image, norm.ImageMime, p.ImageOp); err != nil {
		return model.UserTitle{}, mapTitleErr(err)
	}
	got, err := s.repo.GetByID(ctx, s.db, id)
	if err != nil || got == nil {
		return model.UserTitle{}, fmt.Errorf("reload title: %w", err)
	}
	debuglog.Printf("service.Title.Update id=%d", id)
	return *got, nil
}

// Delete removes a title.
func (s *Title) Delete(ctx context.Context, id int64) error {
	ok, err := s.repo.Delete(ctx, s.db, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrTitleNotFound
	}
	debuglog.Printf("service.Title.Delete id=%d", id)
	return nil
}

func (s *Title) normalize(ctx context.Context, p TitleParams) (TitleParams, error) {
	p.Kind = strings.TrimSpace(p.Kind)
	p.Name = strings.TrimSpace(p.Name)
	if p.UserID <= 0 {
		return TitleParams{}, fmt.Errorf("%w: userId is required", ErrTitleInvalid)
	}
	if p.Kind != model.TitleKindFantasy && p.Kind != model.TitleKindTournament {
		return TitleParams{}, fmt.Errorf("%w: kind must be fantasy or tournament", ErrTitleInvalid)
	}
	if p.Name == "" || utf8.RuneCountInString(p.Name) > MaxTitleNameLen {
		return TitleParams{}, fmt.Errorf("%w: name is required (max %d)", ErrTitleInvalid, MaxTitleNameLen)
	}
	user, err := s.users.GetByID(ctx, s.db, p.UserID)
	if err != nil {
		return TitleParams{}, err
	}
	if user == nil {
		return TitleParams{}, fmt.Errorf("%w: user not found", ErrTitleInvalid)
	}
	if p.Kind == model.TitleKindTournament {
		p.FantasyLeagueID = nil
	} else if p.FantasyLeagueID != nil {
		if *p.FantasyLeagueID <= 0 {
			p.FantasyLeagueID = nil
		} else {
			league, err := s.fantasy.GetLeagueByID(ctx, s.db, *p.FantasyLeagueID)
			if err != nil {
				return TitleParams{}, err
			}
			if league == nil {
				return TitleParams{}, fmt.Errorf("%w: fantasy league not found", ErrTitleInvalid)
			}
		}
	}
	if p.ImageOp == ImageSet {
		mime, err := validateTitleImage(p.Image, p.ImageMime)
		if err != nil {
			return TitleParams{}, err
		}
		p.ImageMime = mime
	} else {
		p.Image = nil
		p.ImageMime = ""
	}
	return p, nil
}

func validateTitleImage(data []byte, headerType string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("%w: image is empty", ErrTitleInvalid)
	}
	if len(data) > MaxTitleImageLen {
		return "", fmt.Errorf("%w: image must be at most 512KB", ErrTitleInvalid)
	}
	ct := strings.ToLower(strings.TrimSpace(strings.Split(headerType, ";")[0]))
	if !allowedTitleImageType(ct) {
		ct = strings.ToLower(http.DetectContentType(data))
	}
	if !allowedTitleImageType(ct) && isWebP(data) {
		ct = "image/webp"
	}
	if !allowedTitleImageType(ct) {
		return "", fmt.Errorf("%w: image must be jpeg, png, or webp", ErrTitleInvalid)
	}
	return ct, nil
}

func allowedTitleImageType(ct string) bool {
	return ct == "image/jpeg" || ct == "image/png" || ct == "image/webp"
}

func isWebP(data []byte) bool {
	return len(data) >= 12 && bytes.Equal(data[0:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP"))
}

func mapTitleErr(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unique") {
		return ErrTitleConflict
	}
	return err
}
