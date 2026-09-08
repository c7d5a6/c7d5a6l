-- Opt out of Telegram @mentions in group notices. Existing users stay enabled.
ALTER TABLE user ADD COLUMN notifications_enabled INTEGER NOT NULL DEFAULT 1
    CHECK (notifications_enabled IN (0, 1));
