package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// Title stores user titles (awards).
type Title struct {
	db *sql.DB
}

func NewTitle(db *sql.DB) *Title {
	return &Title{db: db}
}

func (r *Title) DB() *sql.DB {
	return r.db
}

const titleSelectCols = `
	ut.id,
	ut.user_id,
	u.alias,
	ut.kind,
	ut.name,
	ut.fantasy_league_id,
	t.name,
	CASE WHEN ut.image IS NOT NULL AND length(ut.image) > 0 THEN 1 ELSE 0 END,
	ut.created_at
`

const titleFrom = `
	FROM user_title ut
	JOIN user u ON u.id = ut.user_id
	LEFT JOIN fantasy_league fl ON fl.id = ut.fantasy_league_id
	LEFT JOIN tournament t ON t.id = fl.tournament_id
`

// ListAll returns every title, newest first.
func (r *Title) ListAll(ctx context.Context, q DBTX) ([]model.UserTitle, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT `+titleSelectCols+titleFrom+`
		ORDER BY ut.created_at DESC, ut.id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list titles: %w", err)
	}
	defer rows.Close()
	return scanTitles(rows)
}

// ListByUserID returns titles for one user, newest first.
func (r *Title) ListByUserID(ctx context.Context, q DBTX, userID int64) ([]model.UserTitle, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT `+titleSelectCols+titleFrom+`
		WHERE ut.user_id = ?
		ORDER BY ut.created_at DESC, ut.id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list titles by user: %w", err)
	}
	defer rows.Close()
	return scanTitles(rows)
}

// ListByUserIDs returns titles grouped by user id.
func (r *Title) ListByUserIDs(ctx context.Context, q DBTX, userIDs []int64) (map[int64][]model.UserTitle, error) {
	out := make(map[int64][]model.UserTitle, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(userIDs))
	args := make([]any, len(userIDs))
	for i, id := range userIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := q.QueryContext(ctx, `
		SELECT `+titleSelectCols+titleFrom+`
		WHERE ut.user_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY ut.created_at DESC, ut.id DESC
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("list titles by users: %w", err)
	}
	defer rows.Close()
	list, err := scanTitles(rows)
	if err != nil {
		return nil, err
	}
	for _, t := range list {
		out[t.UserID] = append(out[t.UserID], t)
	}
	return out, nil
}

// GetByID returns a title or nil.
func (r *Title) GetByID(ctx context.Context, q DBTX, id int64) (*model.UserTitle, error) {
	row := q.QueryRowContext(ctx, `
		SELECT `+titleSelectCols+titleFrom+`
		WHERE ut.id = ?
	`, id)
	t, err := scanTitleRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

// Insert creates a title. image may be nil.
func (r *Title) Insert(ctx context.Context, q DBTX, t model.UserTitle, image []byte, mime string) (int64, error) {
	res, err := q.ExecContext(ctx, `
		INSERT INTO user_title (user_id, kind, name, fantasy_league_id, image, image_mime)
		VALUES (?, ?, ?, ?, ?, ?)
	`, t.UserID, t.Kind, t.Name, nullableInt64(t.FantasyLeagueID), nullableBlob(image), nullableMime(mime))
	if err != nil {
		return 0, fmt.Errorf("insert title: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("title last insert id: %w", err)
	}
	return id, nil
}

// Update writes metadata. imageOp: 0 keep, 1 set, 2 clear.
func (r *Title) Update(ctx context.Context, q DBTX, t model.UserTitle, image []byte, mime string, imageOp int) error {
	var err error
	switch imageOp {
	case 1:
		_, err = q.ExecContext(ctx, `
			UPDATE user_title
			SET user_id = ?, kind = ?, name = ?, fantasy_league_id = ?, image = ?, image_mime = ?
			WHERE id = ?
		`, t.UserID, t.Kind, t.Name, nullableInt64(t.FantasyLeagueID), image, nullableMime(mime), t.ID)
	case 2:
		_, err = q.ExecContext(ctx, `
			UPDATE user_title
			SET user_id = ?, kind = ?, name = ?, fantasy_league_id = ?, image = NULL, image_mime = NULL
			WHERE id = ?
		`, t.UserID, t.Kind, t.Name, nullableInt64(t.FantasyLeagueID), t.ID)
	default:
		_, err = q.ExecContext(ctx, `
			UPDATE user_title
			SET user_id = ?, kind = ?, name = ?, fantasy_league_id = ?
			WHERE id = ?
		`, t.UserID, t.Kind, t.Name, nullableInt64(t.FantasyLeagueID), t.ID)
	}
	if err != nil {
		return fmt.Errorf("update title: %w", err)
	}
	return nil
}

// Delete removes a title. Returns false when missing.
func (r *Title) Delete(ctx context.Context, q DBTX, id int64) (bool, error) {
	res, err := q.ExecContext(ctx, `DELETE FROM user_title WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete title: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetImage returns blob + mime, or nil when missing / no image.
func (r *Title) GetImage(ctx context.Context, q DBTX, id int64) ([]byte, string, error) {
	var blob []byte
	var mime sql.NullString
	err := q.QueryRowContext(ctx, `
		SELECT image, image_mime FROM user_title WHERE id = ?
	`, id).Scan(&blob, &mime)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("get title image: %w", err)
	}
	if len(blob) == 0 {
		return nil, "", nil
	}
	ct := "application/octet-stream"
	if mime.Valid && strings.TrimSpace(mime.String) != "" {
		ct = strings.TrimSpace(mime.String)
	}
	return blob, ct, nil
}

func scanTitles(rows *sql.Rows) ([]model.UserTitle, error) {
	out := make([]model.UserTitle, 0)
	for rows.Next() {
		t, err := scanTitleRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

type titleScanner interface {
	Scan(dest ...any) error
}

func scanTitleRow(row titleScanner) (*model.UserTitle, error) {
	var t model.UserTitle
	var leagueID sql.NullInt64
	var leagueName sql.NullString
	var hasImage int
	err := row.Scan(
		&t.ID,
		&t.UserID,
		&t.UserAlias,
		&t.Kind,
		&t.Name,
		&leagueID,
		&leagueName,
		&hasImage,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan title: %w", err)
	}
	t.FantasyLeagueID = nullInt64ToPtr(leagueID)
	t.FantasyLeagueName = nullToPtr(leagueName)
	t.HasImage = hasImage == 1
	return &t, nil
}

func nullableBlob(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}
