# Chapter 68 Checkpoint — Migrations

## Self-assessment questions

1. What are the two files in a golang-migrate migration pair, and what does each do?
2. What does the `schema_migrations` table store?
3. What is the difference between `m.Up()`, `m.Steps(1)`, and `m.Migrate(3)`?
4. What does `migrate.ErrNoChange` mean, and how should you handle it?
5. What is the "dirty" flag, and how do you recover from it?
6. Why should you never rename a column in a single migration in production?
7. What is the expand/contract pattern for zero-downtime column renames?

## Checklist

- [ ] Can create numbered `.up.sql` and `.down.sql` migration files
- [ ] Can embed migration files with `//go:embed migrations/*.sql`
- [ ] Can create a `migrate.Migrate` instance using `iofs.New` and `sqlite.WithInstance`
- [ ] Can apply migrations with `m.Up()`, `m.Steps(n)`, `m.Migrate(v)`
- [ ] Can check the current version with `m.Version()`
- [ ] Can handle `migrate.ErrNoChange` (idempotent Up)
- [ ] Know how to recover from a dirty migration state with `m.Force(v)`
- [ ] Can write a down migration for SQLite that drops tables or recreates them (no DROP COLUMN)
- [ ] Know the expand/contract pattern for zero-downtime renames

## Answers

1. `000001_name.up.sql` applies the migration (CREATE TABLE, ALTER TABLE, etc). `000001_name.down.sql` rolls it back (DROP TABLE, recreate table without column). The pair allows forward and backward schema changes.

2. The current migration version (a uint) and a dirty flag (bool). One row, always present after the first migration is applied. golang-migrate uses it to know where the schema currently is.

3. `m.Up()` applies all pending migrations (from current version to latest). `m.Steps(1)` applies exactly one migration. `m.Migrate(3)` moves to exactly version 3 — applies or rolls back as needed.

4. `ErrNoChange` means the database is already at the target version — no migration was applied. It is not an error condition; check for it with `errors.Is(err, migrate.ErrNoChange)` and ignore it.

5. The dirty flag is set when a migration starts but fails before completing. It prevents subsequent migrations from running (to avoid applying on top of a broken state). Recovery: fix the SQL, then call `m.Force(v)` with the intended version — this clears the dirty flag without running any SQL.

6. A column rename (`ALTER TABLE RENAME COLUMN`) is not backward-compatible. If the application is already deployed and reading the old column name, the running code will break the moment the migration runs. You need the old and new column to coexist during the transition window.

7. Expand: add the new column, deploy code that writes to both. Backfill: copy data from old to new column. Switch: deploy code that reads only from new column. Contract: drop the old column. Each step is a separate migration, each deployed independently.
