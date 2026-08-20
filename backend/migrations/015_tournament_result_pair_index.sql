-- Allow rematches: identity is unordered pair + occurrence index, not pair alone.
CREATE TABLE tournament_result_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tournament_id INTEGER NOT NULL REFERENCES tournament (id) ON DELETE CASCADE,
    tournament_group_id INTEGER REFERENCES tournament_group (id) ON DELETE SET NULL,
    phase TEXT NOT NULL,
    round TEXT NOT NULL DEFAULT '',
    tournament_player_a_id INTEGER NOT NULL REFERENCES tournament_player (id) ON DELETE CASCADE,
    tournament_player_b_id INTEGER NOT NULL REFERENCES tournament_player (id) ON DELETE CASCADE,
    player_lo INTEGER NOT NULL,
    player_hi INTEGER NOT NULL,
    pair_index INTEGER NOT NULL DEFAULT 0,
    score_a INTEGER,
    score_b INTEGER,
    played INTEGER NOT NULL DEFAULT 0 CHECK (played IN (0, 1)),
    played_at TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE (tournament_id, player_lo, player_hi, pair_index)
);

INSERT INTO tournament_result_new (
    id, tournament_id, tournament_group_id, phase, round,
    tournament_player_a_id, tournament_player_b_id, player_lo, player_hi, pair_index,
    score_a, score_b, played, played_at, sort_order
)
SELECT
    id, tournament_id, tournament_group_id, phase, round,
    tournament_player_a_id, tournament_player_b_id, player_lo, player_hi, 0,
    score_a, score_b, played, played_at, sort_order
FROM tournament_result;

DROP TABLE tournament_result;
ALTER TABLE tournament_result_new RENAME TO tournament_result;

CREATE INDEX tournament_result_played_at_idx ON tournament_result (tournament_id, played_at);
CREATE INDEX tournament_result_group_idx ON tournament_result (tournament_group_id);
