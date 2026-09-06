-- Case-sensitive player links (Liquipedia titles: TerrOr ≠ Terror).
-- Keeps legacy link column; Go uses link_v2 as the canonical URL.
ALTER TABLE player ADD COLUMN link_v2 TEXT;

UPDATE player SET link_v2 = link WHERE link_v2 IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS player_link_v2 ON player (link_v2);
