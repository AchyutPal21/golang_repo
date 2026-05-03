# Chapter 68 Exercises — Migrations

## Exercise 1 — Schema Versioning (`exercises/01_schema_versioning`)

Build a versioned blog schema with three migrations and demonstrate apply, rollback, and reapply.

### Migrations to implement

| Version | File | Contents |
|---|---|---|
| 1 | `000001_create_blog.up.sql` | `authors` table (id, name, email, created_at), `posts` table (id, author_id, title, body, published, created_at) |
| 2 | `000002_add_tags.up.sql` | `tags` (id, name UNIQUE), `post_tags` join table with FK CASCADE and index |
| 3 | `000003_add_slug_and_index.up.sql` | `ALTER TABLE posts ADD COLUMN slug TEXT`, backfill slugs from title, 3 indexes on posts |

### Down migrations

- `000001_create_blog.down.sql` — DROP TABLE posts; DROP TABLE authors
- `000002_add_tags.down.sql` — DROP INDEX, DROP TABLE post_tags, DROP TABLE tags
- `000003_add_slug_and_index.down.sql` — DROP INDEXes, recreate posts without slug (SQLite table-recreation idiom)

### Demonstration flow in main()

1. Apply migration 1 → verify `authors` and `posts` tables exist, seed 2 posts
2. Apply migration 2 → verify `tags` and `post_tags` tables exist, seed tags + post_tags
3. Apply migration 3 → verify slug column and indexes, print `(title, slug)` pairs showing backfill
4. Call `m.Up()` again → should get `ErrNoChange`
5. Roll back step by step to version 1 → verify tables after each rollback
6. Verify posts seeded in step 1 are still present (data migrations don't delete data)
7. Reapply `m.Up()` → back to version 3

### Expected output

```
version: 3
[final state]
  tables:  [authors post_tags posts tags]
  posts cols: [id author_id title body published created_at slug]
  post indexes: [idx_posts_author idx_posts_published idx_posts_slug]
```

### Hints

- Use `iofs.New(migrationsFS, "migrations")` with `//go:embed migrations/*.sql`
- `m.Steps(-1)` rolls back exactly one migration
- `m.Migrate(1)` jumps to version 1 from any higher version — rolls back 2 and 3
- SQLite doesn't support `DROP COLUMN` in older versions — recreate the table in down migrations
- `m.Version()` returns `(uint, bool, error)` — check for `migrate.ErrNilVersion` when no migration has run
