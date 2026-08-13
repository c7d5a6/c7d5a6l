-- Case-insensitive uniqueness for links on DBs created before COLLATE NOCASE on columns.
CREATE UNIQUE INDEX IF NOT EXISTS tournament_link_nocase ON tournament (link COLLATE NOCASE);
CREATE UNIQUE INDEX IF NOT EXISTS player_link_nocase ON player (link COLLATE NOCASE);
