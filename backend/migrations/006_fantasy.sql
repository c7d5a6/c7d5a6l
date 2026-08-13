-- Fantasy league schema
CREATE TABLE user (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alias TEXT NOT NULL UNIQUE COLLATE NOCASE
);

CREATE TABLE fantasy_league (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tournament_id INTEGER NOT NULL UNIQUE REFERENCES tournament (id) ON DELETE CASCADE,
    finished INTEGER NOT NULL DEFAULT 0 CHECK (finished IN (0, 1))
);

CREATE TABLE fantasy_player (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fantasy_league_id INTEGER NOT NULL REFERENCES fantasy_league (id) ON DELETE CASCADE,
    tournament_player_id INTEGER NOT NULL REFERENCES tournament_player (id) ON DELETE CASCADE,
    cost INTEGER NOT NULL DEFAULT 0,
    points_earned INTEGER NOT NULL DEFAULT 0,
    UNIQUE (fantasy_league_id, tournament_player_id)
);

CREATE TABLE fantasy_team (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fantasy_league_id INTEGER NOT NULL REFERENCES fantasy_league (id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES user (id) ON DELETE CASCADE,
    UNIQUE (fantasy_league_id, user_id)
);

CREATE TABLE fantasy_team_member (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fantasy_team_id INTEGER NOT NULL REFERENCES fantasy_team (id) ON DELETE CASCADE,
    fantasy_player_id INTEGER NOT NULL REFERENCES fantasy_player (id) ON DELETE CASCADE,
    UNIQUE (fantasy_team_id, fantasy_player_id)
);
