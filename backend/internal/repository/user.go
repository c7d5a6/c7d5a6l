package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// User stores Telegram-authenticated and alias-only accounts.
type User struct {
	db *sql.DB
}

func NewUser(db *sql.DB) *User {
	return &User{db: db}
}

func (r *User) DB() *sql.DB {
	return r.db
}

// GetByTelegramID returns the user or nil if missing.
func (r *User) GetByTelegramID(ctx context.Context, q DBTX, telegramID int64) (*model.User, error) {
	row := q.QueryRowContext(ctx, `
		SELECT id, alias, telegram_id, telegram_username, first_name, last_name,
		       photo_url, role, created_at, updated_at, last_login_at
		FROM user
		WHERE telegram_id = ?
	`, telegramID)
	return scanUser(row)
}

// GetByID returns the user or nil if missing.
func (r *User) GetByID(ctx context.Context, q DBTX, id int64) (*model.User, error) {
	row := q.QueryRowContext(ctx, `
		SELECT id, alias, telegram_id, telegram_username, first_name, last_name,
		       photo_url, role, created_at, updated_at, last_login_at
		FROM user
		WHERE id = ?
	`, id)
	return scanUser(row)
}

// AliasTaken reports whether alias is used by another user (excludeID 0 = none).
func (r *User) AliasTaken(ctx context.Context, q DBTX, alias string, excludeID int64) (bool, error) {
	var n int
	err := q.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user WHERE alias = ? COLLATE NOCASE AND id != ?
	`, alias, excludeID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("alias taken: %w", err)
	}
	return n > 0, nil
}

// Insert creates a new user.
func (r *User) Insert(ctx context.Context, q DBTX, u model.User) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if u.CreatedAt == "" {
		u.CreatedAt = now
	}
	if u.UpdatedAt == "" {
		u.UpdatedAt = now
	}
	res, err := q.ExecContext(ctx, `
		INSERT INTO user (
			alias, telegram_id, telegram_username, first_name, last_name,
			photo_url, role, created_at, updated_at, last_login_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		u.Alias,
		nullableInt64(u.TelegramID),
		nullableText(u.TelegramUsername),
		u.FirstName,
		nullableText(u.LastName),
		nullableText(u.PhotoURL),
		u.Role,
		u.CreatedAt,
		u.UpdatedAt,
		nullableText(u.LastLoginAt),
	)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("user last insert id: %w", err)
	}
	return id, nil
}

// UpdateTelegramProfile refreshes Telegram fields, optional role promote, and login stamp.
func (r *User) UpdateTelegramProfile(ctx context.Context, q DBTX, u model.User) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := q.ExecContext(ctx, `
		UPDATE user SET
			alias = ?,
			telegram_id = ?,
			telegram_username = ?,
			first_name = ?,
			last_name = ?,
			photo_url = ?,
			role = ?,
			updated_at = ?,
			last_login_at = ?
		WHERE id = ?
	`,
		u.Alias,
		nullableInt64(u.TelegramID),
		nullableText(u.TelegramUsername),
		u.FirstName,
		nullableText(u.LastName),
		nullableText(u.PhotoURL),
		u.Role,
		now,
		nullableText(u.LastLoginAt),
		u.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// UpdateAlias sets display alias for a user.
func (r *User) UpdateAlias(ctx context.Context, q DBTX, id int64, alias string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := q.ExecContext(ctx, `
		UPDATE user SET alias = ?, updated_at = ? WHERE id = ?
	`, alias, now, id)
	if err != nil {
		return fmt.Errorf("update alias: %w", err)
	}
	return nil
}

// ListAll returns users ordered by alias.
func (r *User) ListAll(ctx context.Context, q DBTX) ([]model.User, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, alias, telegram_id, telegram_username, first_name, last_name,
		       photo_url, role, created_at, updated_at, last_login_at
		FROM user
		ORDER BY alias COLLATE NOCASE ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	out := make([]model.User, 0)
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(row *sql.Row) (*model.User, error) {
	u, err := scanUserRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func scanUserRow(row userScanner) (*model.User, error) {
	var u model.User
	var telegramID sql.NullInt64
	var username, lastName, photo, lastLogin sql.NullString
	err := row.Scan(
		&u.ID,
		&u.Alias,
		&telegramID,
		&username,
		&u.FirstName,
		&lastName,
		&photo,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
		&lastLogin,
	)
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.TelegramID = nullInt64ToPtr(telegramID)
	u.TelegramUsername = nullToPtr(username)
	u.LastName = nullToPtr(lastName)
	u.PhotoURL = nullToPtr(photo)
	u.LastLoginAt = nullToPtr(lastLogin)
	return &u, nil
}

func nullToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func nullInt64ToPtr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func nullableInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}
