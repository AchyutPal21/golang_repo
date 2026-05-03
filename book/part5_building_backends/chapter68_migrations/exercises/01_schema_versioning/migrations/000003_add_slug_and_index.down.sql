DROP INDEX IF EXISTS idx_posts_author;
DROP INDEX IF EXISTS idx_posts_published;
DROP INDEX IF EXISTS idx_posts_slug;

-- Recreate posts table without slug column (SQLite column removal pattern).
CREATE TABLE posts_temp (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    author_id  INTEGER NOT NULL REFERENCES authors(id),
    title      TEXT    NOT NULL,
    body       TEXT    NOT NULL DEFAULT '',
    published  INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO posts_temp SELECT id, author_id, title, body, published, created_at FROM posts;
DROP TABLE posts;
ALTER TABLE posts_temp RENAME TO posts;
