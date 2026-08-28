-- Season lifecycle for ELO rating recalculation. Additive only — no changes to player_race schema.

CREATE TABLE season (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'closed')) DEFAULT 'active',
    started_at TEXT NOT NULL,
    closed_at TEXT,
    ready_to_close INTEGER NOT NULL DEFAULT 0 CHECK (ready_to_close IN (0, 1)),
    closing_fantasy_league_id INTEGER REFERENCES fantasy_league (id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX season_one_active ON season (status) WHERE status = 'active';

CREATE TABLE season_tournament (
    season_id INTEGER NOT NULL REFERENCES season (id) ON DELETE CASCADE,
    tournament_id INTEGER NOT NULL REFERENCES tournament (id) ON DELETE CASCADE,
    included_in_rating INTEGER NOT NULL DEFAULT 1 CHECK (included_in_rating IN (0, 1)),
    PRIMARY KEY (season_id, tournament_id)
);

CREATE TABLE season_player_race (
    season_id INTEGER NOT NULL REFERENCES season (id) ON DELETE CASCADE,
    player_race_id INTEGER NOT NULL REFERENCES player_race (id) ON DELETE CASCADE,
    start_elo REAL NOT NULL,
    end_elo REAL,
    start_rank INTEGER NOT NULL,
    end_rank INTEGER,
    PRIMARY KEY (season_id, player_race_id)
);

CREATE INDEX season_player_race_player_race_idx ON season_player_race (player_race_id);

-- Bootstrap Season 1 from current player_race rows without changing elo values.
INSERT INTO season (name, status, started_at)
VALUES ('Season 1', 'active', datetime('now'));

INSERT INTO season_player_race (season_id, player_race_id, start_elo, start_rank)
SELECT
    1,
    pr.id,
    pr.elo,
    ROW_NUMBER() OVER (ORDER BY pr.elo DESC, pr.id ASC)
FROM player_race pr;
