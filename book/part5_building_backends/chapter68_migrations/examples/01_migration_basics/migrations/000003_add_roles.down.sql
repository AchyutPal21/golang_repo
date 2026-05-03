DROP INDEX IF EXISTS idx_users_role;
-- SQLite: removing columns requires table reconstruction.
-- Down migrations for ALTER TABLE ADD COLUMN are typically no-ops in SQLite demos.
