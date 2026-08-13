CREATE TABLE user_title (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES user (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('fantasy', 'tournament')),
    name TEXT NOT NULL,
    fantasy_league_id INTEGER REFERENCES fantasy_league (id) ON DELETE SET NULL,
    image BLOB,
    image_mime TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE UNIQUE INDEX user_title_league_unique
ON user_title (fantasy_league_id)
WHERE fantasy_league_id IS NOT NULL;
