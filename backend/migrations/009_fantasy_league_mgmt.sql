-- Fantasy league management: lifecycle, caps, stage points, status flags.

CREATE TABLE fantasy_league_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tournament_id INTEGER NOT NULL UNIQUE REFERENCES tournament (id) ON DELETE CASCADE,
    started INTEGER NOT NULL DEFAULT 0 CHECK (started IN (0, 1)),
    finished INTEGER NOT NULL DEFAULT 0 CHECK (finished IN (0, 1)),
    max_players INTEGER NOT NULL DEFAULT 6,
    max_cost INTEGER NOT NULL DEFAULT 28
);

INSERT INTO fantasy_league_new (id, tournament_id, started, finished, max_players, max_cost)
SELECT id, tournament_id, 0, finished, 6, 28
FROM fantasy_league;

CREATE TABLE fantasy_player_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fantasy_league_id INTEGER NOT NULL REFERENCES fantasy_league_new (id) ON DELETE CASCADE,
    tournament_player_id INTEGER NOT NULL REFERENCES tournament_player (id) ON DELETE CASCADE,
    cost INTEGER NOT NULL DEFAULT 0,
    points_ro24 INTEGER,
    points_ro16 INTEGER,
    points_ro8 INTEGER,
    points_ro4 INTEGER,
    points_ro2 INTEGER,
    defeated INTEGER NOT NULL DEFAULT 0 CHECK (defeated IN (0, 1)),
    is_winner INTEGER NOT NULL DEFAULT 0 CHECK (is_winner IN (0, 1)),
    UNIQUE (fantasy_league_id, tournament_player_id)
);

INSERT INTO fantasy_player_new (
    id, fantasy_league_id, tournament_player_id, cost,
    points_ro24, points_ro16, points_ro8, points_ro4, points_ro2,
    defeated, is_winner
)
SELECT
    id, fantasy_league_id, tournament_player_id, cost,
    NULL, NULL, NULL, NULL, NULL,
    0, 0
FROM fantasy_player;

-- Remap team members to survive player table rebuild (ids preserved).
-- Drop dependent tables that reference fantasy_player / fantasy_league, then rename.

CREATE TABLE fantasy_team_member_bak AS SELECT * FROM fantasy_team_member;
CREATE TABLE fantasy_team_bak AS SELECT * FROM fantasy_team;

DROP TABLE fantasy_team_member;
DROP TABLE fantasy_team;
DROP TABLE fantasy_player;
DROP TABLE fantasy_league;

ALTER TABLE fantasy_league_new RENAME TO fantasy_league;
ALTER TABLE fantasy_player_new RENAME TO fantasy_player;

CREATE TABLE fantasy_team (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fantasy_league_id INTEGER NOT NULL REFERENCES fantasy_league (id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES user (id) ON DELETE CASCADE,
    UNIQUE (fantasy_league_id, user_id)
);

INSERT INTO fantasy_team (id, fantasy_league_id, user_id)
SELECT id, fantasy_league_id, user_id FROM fantasy_team_bak;

CREATE TABLE fantasy_team_member (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fantasy_team_id INTEGER NOT NULL REFERENCES fantasy_team (id) ON DELETE CASCADE,
    fantasy_player_id INTEGER NOT NULL REFERENCES fantasy_player (id) ON DELETE CASCADE,
    UNIQUE (fantasy_team_id, fantasy_player_id)
);

INSERT INTO fantasy_team_member (id, fantasy_team_id, fantasy_player_id)
SELECT id, fantasy_team_id, fantasy_player_id FROM fantasy_team_member_bak;

DROP TABLE fantasy_team_member_bak;
DROP TABLE fantasy_team_bak;
