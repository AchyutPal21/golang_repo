-- SQLite requires table recreation to remove columns (no DROP COLUMN before 3.35).
-- Recreate users without bio and avatar_url columns.
CREATE TABLE users_temp (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    username   TEXT    NOT NULL UNIQUE,
    email      TEXT    NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO users_temp SELECT id, username, email, created_at FROM users;
DROP TABLE users;
ALTER TABLE users_temp RENAME TO users;

DROP TABLE IF EXISTS user_settings;
