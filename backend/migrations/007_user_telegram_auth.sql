-- Telegram Login Widget identity + roles on user.
-- Dev rows that only had alias cannot satisfy telegram_id NOT NULL; clear fantasy
-- team ownership first. Reset devdata/app.sqlite if this migration fails mid-way.

DELETE FROM fantasy_team_member;
DELETE FROM fantasy_team;
DELETE FROM user;

CREATE TABLE user_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alias TEXT NOT NULL UNIQUE COLLATE NOCASE,
    telegram_id INTEGER NOT NULL UNIQUE,
    telegram_username TEXT,
    first_name TEXT NOT NULL,
    last_name TEXT,
    photo_url TEXT,
    role TEXT NOT NULL DEFAULT 'USER' CHECK (role IN ('ADMIN', 'USER')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    last_login_at TEXT
);

DROP TABLE user;
ALTER TABLE user_new RENAME TO user;
