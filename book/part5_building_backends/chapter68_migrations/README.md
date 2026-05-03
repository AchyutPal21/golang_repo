# Chapter 68 — Migrations

## What you'll learn

How to manage database schema changes safely over time using `golang-migrate/migrate/v4`: embedding SQL migration files with Go's `embed.FS`, applying and rolling back migrations, inspecting version state, handling the dirty flag, and production patterns for zero-downtime schema changes.

## Key concepts

| Concept | Description |
|---|---|
| Migration file | A pair: `000001_name.up.sql` (apply) and `000001_name.down.sql` (rollback) |
| `schema_migrations` | Table managed by golang-migrate; tracks current version and dirty flag |
| `m.Up()` | Apply all pending migrations |
| `m.Steps(n)` | Apply n steps forward (positive) or backward (negative) |
| `m.Migrate(v)` | Jump to exact version v |
| `ErrNoChange` | Already at target version — not an error to handle |
| Dirty flag | Set when a migration fails mid-way; blocks further migrations until resolved |

## Files

| File | Topic |
|---|---|
| `examples/01_migration_basics/main.go` | Up/Down, Steps, Version inspection, ErrNoChange, schema_migrations table |
| `examples/02_migration_patterns/main.go` | Expand/contract rename, data backfill, idempotent DDL, dirty state recovery |
| `exercises/01_schema_versioning/main.go` | Blog schema: 3 migrations with apply, rollback, reapply |

## Setup

```go
import (
    "embed"
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/sqlite"  // or postgres
    "github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func newMigrator(db *sql.DB) (*migrate.Migrate, error) {
    src, _ := iofs.New(migrationsFS, "migrations")
    driver, _ := sqlite.WithInstance(db, &sqlite.Config{})
    return migrate.NewWithInstance("iofs", src, "sqlite", driver)
}
```

## File naming convention

```
migrations/
  000001_create_users.up.sql
  000001_create_users.down.sql
  000002_add_profile.up.sql
  000002_add_profile.down.sql
```

- Numbers are zero-padded to at least 6 digits
- Name is descriptive (lowercase, underscores)
- `.up.sql` is applied; `.down.sql` rolls it back

## Zero-downtime column rename (expand/contract)

```
Step 1: ALTER TABLE t ADD COLUMN new_col TEXT     ← deploy: write to both columns
Step 2: UPDATE t SET new_col = old_col            ← backfill
Step 3: (deploy: code reads new_col only)
Step 4: DROP COLUMN old_col                       ← cleanup
```

Never rename a column in a single migration — it breaks the running application.

## Dirty state recovery

```go
// If a migration fails: schema_migrations has dirty=true
// Fix the SQL, then clear the dirty flag:
m.Force(v)  // sets version v as clean, no migration is run
```

## Production tips

- Run `m.Up()` at application startup or as a separate deploy step (not both)
- Use advisory locks in PostgreSQL to prevent concurrent migration runs
- Always test rollback (`m.Down()` or `m.Steps(-1)`) in CI/staging
- Never edit a migration that has been applied to any environment
- For large tables: backfill in batches (`UPDATE ... WHERE id BETWEEN ? AND ? AND new_col IS NULL`)
- `CREATE INDEX CONCURRENTLY` in PostgreSQL avoids locking the table during index creation
