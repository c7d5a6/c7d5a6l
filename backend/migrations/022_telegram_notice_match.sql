-- Per-match notices need result_id; kind check gains match_finished.
-- telegram_notice is a send log (no dependents). Rebuild preserves existing rows.
CREATE TABLE telegram_notice_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL CHECK (kind IN ('prematch', 'day_done', 'match_finished')),
    league_id INTEGER NOT NULL REFERENCES fantasy_league (id) ON DELETE CASCADE,
    day TEXT NOT NULL,
    result_id INTEGER NOT NULL DEFAULT 0,
    sent_at TEXT NOT NULL,
    UNIQUE (kind, league_id, day, result_id)
);

INSERT INTO telegram_notice_new (id, kind, league_id, day, result_id, sent_at)
SELECT id, kind, league_id, day, 0, sent_at FROM telegram_notice;

DROP TABLE telegram_notice;
ALTER TABLE telegram_notice_new RENAME TO telegram_notice;
