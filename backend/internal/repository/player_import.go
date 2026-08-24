package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// PlayerImport statuses for Liquipedia enrichment jobs.
const (
	ImportPending = "pending"
	ImportRunning = "running"
	ImportDone    = "done"
	ImportError   = "error"
)

// PlayerImport persists async player page import queue rows.
type PlayerImport struct{}

func NewPlayerImport() *PlayerImport { return &PlayerImport{} }

// Enqueue inserts or re-queues links as pending (idempotent per link).
func (r *PlayerImport) Enqueue(ctx context.Context, q DBTX, links []string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for _, link := range links {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}
		if _, err := q.ExecContext(ctx, `
			INSERT INTO player_import_queue (link, status, error, created_at, updated_at)
			VALUES (?, ?, NULL, ?, ?)
			ON CONFLICT(link) DO UPDATE SET
				status = CASE
					WHEN player_import_queue.status IN ('done', 'error') THEN 'pending'
					ELSE player_import_queue.status
				END,
				error = CASE
					WHEN player_import_queue.status IN ('done', 'error') THEN NULL
					ELSE player_import_queue.error
				END,
				updated_at = excluded.updated_at
		`, link, ImportPending, now, now); err != nil {
			return fmt.Errorf("enqueue player import %s: %w", link, err)
		}
	}
	return nil
}

// ResetRunningToPending recovers jobs left mid-flight after a crash.
func (r *PlayerImport) ResetRunningToPending(ctx context.Context, q DBTX) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := q.ExecContext(ctx, `
		UPDATE player_import_queue
		SET status = ?, error = NULL, updated_at = ?
		WHERE status = ?
	`, ImportPending, now, ImportRunning)
	if err != nil {
		return 0, fmt.Errorf("reset running imports: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return n, nil
}

// ClaimNext marks the oldest pending row running and returns its link.
// Returns "", nil when the queue is empty.
func (r *PlayerImport) ClaimNext(ctx context.Context, q DBTX) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, ok := q.(*sql.Tx)
	if !ok {
		db, isDB := q.(*sql.DB)
		if !isDB {
			return "", fmt.Errorf("claim next import: need *sql.DB or *sql.Tx")
		}
		var err error
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return "", err
		}
		defer tx.Rollback()
		link, err := r.claimNextTx(ctx, tx, now)
		if err != nil {
			return "", err
		}
		if link == "" {
			return "", nil
		}
		if err := tx.Commit(); err != nil {
			return "", err
		}
		return link, nil
	}
	return r.claimNextTx(ctx, tx, now)
}

func (r *PlayerImport) claimNextTx(ctx context.Context, tx *sql.Tx, now string) (string, error) {
	var id int64
	var link string
	err := tx.QueryRowContext(ctx, `
		SELECT id, link FROM player_import_queue
		WHERE status = ?
		ORDER BY id ASC
		LIMIT 1
	`, ImportPending).Scan(&id, &link)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("select pending import: %w", err)
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE player_import_queue SET status = ?, updated_at = ? WHERE id = ? AND status = ?
	`, ImportRunning, now, id, ImportPending)
	if err != nil {
		return "", fmt.Errorf("claim import: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", nil
	}
	return link, nil
}

// MarkDone sets a queue row to done by link.
func (r *PlayerImport) MarkDone(ctx context.Context, q DBTX, link string) error {
	return r.setStatus(ctx, q, link, ImportDone, "")
}

// MarkError sets a queue row to error by link.
func (r *PlayerImport) MarkError(ctx context.Context, q DBTX, link, errMsg string) error {
	return r.setStatus(ctx, q, link, ImportError, errMsg)
}

func (r *PlayerImport) setStatus(ctx context.Context, q DBTX, link, status, errMsg string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var errAny any
	if strings.TrimSpace(errMsg) != "" {
		errAny = errMsg
	}
	if _, err := q.ExecContext(ctx, `
		UPDATE player_import_queue
		SET status = ?, error = ?, updated_at = ?
		WHERE link = ? COLLATE NOCASE
	`, status, errAny, now, link); err != nil {
		return fmt.Errorf("set import status %s: %w", status, err)
	}
	return nil
}

// PendingLinks returns the set of links currently pending or running (lowercased keys).
func (r *PlayerImport) PendingLinks(ctx context.Context, q DBTX) (map[string]struct{}, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT link FROM player_import_queue WHERE status IN (?, ?)
	`, ImportPending, ImportRunning)
	if err != nil {
		return nil, fmt.Errorf("list pending imports: %w", err)
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var link string
		if err := rows.Scan(&link); err != nil {
			return nil, err
		}
		out[strings.ToLower(strings.TrimSpace(link))] = struct{}{}
	}
	return out, rows.Err()
}

// CountActive returns pending + running rows.
func (r *PlayerImport) CountActive(ctx context.Context, q DBTX) (int, error) {
	var n int
	err := q.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM player_import_queue WHERE status IN (?, ?)
	`, ImportPending, ImportRunning).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count active imports: %w", err)
	}
	return n, nil
}
