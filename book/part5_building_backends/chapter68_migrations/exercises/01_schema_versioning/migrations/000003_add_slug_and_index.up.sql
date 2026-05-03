ALTER TABLE posts ADD COLUMN slug TEXT;

-- Backfill slugs for existing rows (simplified: use title as slug placeholder).
UPDATE posts SET slug = lower(replace(title, ' ', '-')) WHERE slug IS NULL;

CREATE INDEX IF NOT EXISTS idx_posts_slug      ON posts(slug);
CREATE INDEX IF NOT EXISTS idx_posts_published ON posts(published);
CREATE INDEX IF NOT EXISTS idx_posts_author    ON posts(author_id);
