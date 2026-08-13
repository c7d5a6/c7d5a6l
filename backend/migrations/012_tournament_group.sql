CREATE TABLE tournament_group (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tournament_id INTEGER NOT NULL REFERENCES tournament (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    phase TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE (tournament_id, phase, name)
);

CREATE TABLE tournament_group_player (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tournament_group_id INTEGER NOT NULL REFERENCES tournament_group (id) ON DELETE CASCADE,
    tournament_player_id INTEGER NOT NULL REFERENCES tournament_player (id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE (tournament_group_id, tournament_player_id)
);
