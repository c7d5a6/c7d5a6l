-- tournament / player core schema
CREATE TABLE tournament (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    link TEXT UNIQUE COLLATE NOCASE,
    name TEXT,
    start_date TEXT,
    end_date TEXT,
    tier TEXT,
    player_count INTEGER,
    finished INTEGER NOT NULL DEFAULT 0 CHECK (finished IN (0, 1))
);

CREATE TABLE player (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    link TEXT UNIQUE COLLATE NOCASE,
    name TEXT,
    real_name TEXT,
    preferred_race TEXT CHECK (
        preferred_race IS NULL
        OR preferred_race IN ('protoss', 'terran', 'zerg', 'random')
    )
);

CREATE TABLE player_alias (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL REFERENCES player (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    UNIQUE (player_id, name)
);

CREATE TABLE player_race (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL REFERENCES player (id) ON DELETE CASCADE,
    race TEXT NOT NULL CHECK (race IN ('protoss', 'terran', 'zerg', 'random')),
    elo REAL NOT NULL DEFAULT 1750,
    UNIQUE (player_id, race)
);

CREATE TABLE tournament_player (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tournament_id INTEGER NOT NULL REFERENCES tournament (id) ON DELETE CASCADE,
    player_race_id INTEGER NOT NULL REFERENCES player_race (id) ON DELETE CASCADE,
    player_alias_id INTEGER NOT NULL REFERENCES player_alias (id) ON DELETE CASCADE,
    excluded INTEGER NOT NULL DEFAULT 0 CHECK (excluded IN (0, 1)),
    UNIQUE (tournament_id, player_race_id)
);
