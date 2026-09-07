-- Outbound Telegram notice log. UNIQUE(kind, league_id, day) is the send-once key.
CREATE TABLE telegram_notice (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL CHECK (kind IN ('prematch', 'day_done')),
    league_id INTEGER NOT NULL REFERENCES fantasy_league (id) ON DELETE CASCADE,
    day TEXT NOT NULL,
    sent_at TEXT NOT NULL,
    UNIQUE (kind, league_id, day)
);
