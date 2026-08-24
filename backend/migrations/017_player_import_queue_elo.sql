-- Queue Liquipedia player page fetches so tournament save does not block on the 30s parse rate limit.
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

-- New player_race rows use 1750 elo by default. Existing rows keep their elo.
PRAGMA foreign_keys = OFF;

CREATE TABLE player_race_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL REFERENCES player (id) ON DELETE CASCADE,
    race TEXT NOT NULL CHECK (race IN ('protoss', 'terran', 'zerg', 'random')),
    elo REAL NOT NULL DEFAULT 1750,
    UNIQUE (player_id, race)
);

INSERT INTO player_race_new (id, player_id, race, elo)
SELECT id, player_id, race, elo FROM player_race;

DROP TABLE player_race;
ALTER TABLE player_race_new RENAME TO player_race;

PRAGMA foreign_keys = ON;
