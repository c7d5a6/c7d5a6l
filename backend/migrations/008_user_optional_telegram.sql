-- Allow alias-only users (no Telegram yet). telegram_id becomes nullable UNIQUE.

CREATE TABLE user_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alias TEXT NOT NULL UNIQUE COLLATE NOCASE,
    telegram_id INTEGER UNIQUE,
    telegram_username TEXT,
    first_name TEXT NOT NULL DEFAULT '',
    last_name TEXT,
    photo_url TEXT,
    role TEXT NOT NULL DEFAULT 'USER' CHECK (role IN ('ADMIN', 'USER')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    last_login_at TEXT
);

INSERT INTO user_new (
    id, alias, telegram_id, telegram_username, first_name, last_name,
    photo_url, role, created_at, updated_at, last_login_at
)
SELECT
    id, alias, telegram_id, telegram_username, first_name, last_name,
    photo_url, role, created_at, updated_at, last_login_at
FROM user;

DROP TABLE user;
ALTER TABLE user_new RENAME TO user;
