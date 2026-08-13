package repository

import (
	"context"
	"database/sql"
)

// DBTX is *sql.DB or *sql.Tx.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Player stores players, aliases, and race rows.
type Player struct {
	db *sql.DB
}

func NewPlayer(db *sql.DB) *Player {
	return &Player{db: db}
}

func (r *Player) DB() *sql.DB {
	return r.db
}
