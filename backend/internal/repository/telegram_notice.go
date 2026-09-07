package repository

import (
	"context"
	"fmt"
)

const (
	TelegramNoticePrematch      = "prematch"
	TelegramNoticeDayDone       = "day_done"
	TelegramNoticeMatchFinished = "match_finished"
)

// TelegramNotice stores send-once keys for group messages.
type TelegramNotice struct{}

func NewTelegramNotice() *TelegramNotice {
	return &TelegramNotice{}
}

// Claim inserts the notice key. Returns true if this caller owns the send.
// resultID and scoreText are 0/"" for day-scoped kinds (prematch, day_done).
func (r *TelegramNotice) Claim(ctx context.Context, q DBTX, kind string, leagueID int64, day string, resultID int64, scoreText, sentAt string) (bool, error) {
	res, err := q.ExecContext(ctx, `
		INSERT OR IGNORE INTO telegram_notice (kind, league_id, day, result_id, score_text, sent_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, kind, leagueID, day, resultID, scoreText, sentAt)
	if err != nil {
		return false, fmt.Errorf("claim telegram notice: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("claim telegram notice rows: %w", err)
	}
	return n == 1, nil
}

// Release deletes a claimed key so a failed send can retry.
func (r *TelegramNotice) Release(ctx context.Context, q DBTX, kind string, leagueID int64, day string, resultID int64, scoreText string) error {
	_, err := q.ExecContext(ctx, `
		DELETE FROM telegram_notice
		WHERE kind = ? AND league_id = ? AND day = ? AND result_id = ? AND score_text = ?
	`, kind, leagueID, day, resultID, scoreText)
	if err != nil {
		return fmt.Errorf("release telegram notice: %w", err)
	}
	return nil
}
