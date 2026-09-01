-- Discovered Liquipedia tournaments awaiting admin parse/ignore.
-- disabled=1 means the admin chose to ignore the listing (keep the row).
CREATE TABLE tournament_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    link TEXT NOT NULL COLLATE NOCASE,
    name TEXT,
    start_date TEXT,
    end_date TEXT,
    tier TEXT,
    section TEXT,
    disabled INTEGER NOT NULL DEFAULT 0 CHECK (disabled IN (0, 1)),
    tournament_id INTEGER REFERENCES tournament (id) ON DELETE SET NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    UNIQUE (link)
);

CREATE INDEX tournament_queue_disabled_idx ON tournament_queue (disabled, id);
CREATE INDEX tournament_queue_tournament_id_idx ON tournament_queue (tournament_id);
CREATE INDEX tournament_queue_seen_idx ON tournament_queue (last_seen_at);
