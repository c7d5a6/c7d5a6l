CREATE TABLE tournament_result (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tournament_id INTEGER NOT NULL REFERENCES tournament (id) ON DELETE CASCADE,
    tournament_group_id INTEGER REFERENCES tournament_group (id) ON DELETE SET NULL,
    phase TEXT NOT NULL,
    round TEXT NOT NULL DEFAULT '',
    tournament_player_a_id INTEGER NOT NULL REFERENCES tournament_player (id) ON DELETE CASCADE,
    tournament_player_b_id INTEGER NOT NULL REFERENCES tournament_player (id) ON DELETE CASCADE,
    player_lo INTEGER NOT NULL,
    player_hi INTEGER NOT NULL,
    score_a INTEGER,
    score_b INTEGER,
    played INTEGER NOT NULL DEFAULT 0 CHECK (played IN (0, 1)),
    played_at TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE (tournament_id, player_lo, player_hi)
);

CREATE INDEX tournament_result_played_at_idx ON tournament_result (tournament_id, played_at);
CREATE INDEX tournament_result_group_idx ON tournament_result (tournament_group_id);
