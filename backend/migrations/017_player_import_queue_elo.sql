-- Queue Liquipedia player page fetches so tournament save does not block on the 30s parse rate limit.
--
-- NOTE: Do not rebuild player_race here. Migrations run inside a transaction, so
-- PRAGMA foreign_keys=OFF is a no-op and DROP player_race would CASCADE-delete
-- tournament_player / fantasy roster rows. New-race elo 1750 is set in app code
-- (repository.DefaultElo) and in 001_init for fresh databases.
CREATE TABLE player_import_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    link TEXT NOT NULL COLLATE NOCASE,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'done', 'error')),
    error TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (link)
);

CREATE INDEX player_import_queue_status_idx ON player_import_queue (status, id);
