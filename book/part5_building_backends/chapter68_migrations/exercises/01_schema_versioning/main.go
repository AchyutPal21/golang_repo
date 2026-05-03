// FILE: book/part5_building_backends/chapter68_migrations/exercises/01_schema_versioning/main.go
// CHAPTER: 68 — Migrations
// EXERCISE: Build a versioned blog schema with golang-migrate:
//   - 3 migrations: create (authors+posts), add tags, add slug+indexes
//   - Apply, inspect, roll back, re-apply
//   - Seed data at each migration step and verify schema state
//   - Demonstrate idempotent Up() and ErrNoChange
//
// Run (from the chapter folder):
//   go run ./exercises/01_schema_versioning

package main

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func newMigrator(db *sql.DB) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}
	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return nil, err
	}
	return migrate.NewWithInstance("iofs", src, "sqlite", driver)
}

func tables(db *sql.DB) []string {
	rows, _ := db.Query(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name`)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		if name != "schema_migrations" {
			out = append(out, name)
		}
	}
	return out
}

func columns(db *sql.DB, table string) []string {
	rows, _ := db.Query(`PRAGMA table_info(` + table + `)`)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var cid, notnull, pk int
		var name, colType string
		var dflt interface{}
		rows.Scan(&cid, &name, &colType, &notnull, &dflt, &pk)
		out = append(out, name)
	}
	return out
}

func indexes(db *sql.DB, table string) []string {
	rows, _ := db.Query(`PRAGMA index_list(` + table + `)`)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var seq int
		var name, origin string
		var unique, partial int
		rows.Scan(&seq, &name, &unique, &origin, &partial)
		out = append(out, name)
	}
	return out
}

func printState(db *sql.DB, label string) {
	fmt.Printf("  [%s]\n", label)
	fmt.Printf("    tables:  %v\n", tables(db))
	fmt.Printf("    posts cols: %v\n", columns(db, "posts"))
	fmt.Printf("    post indexes: %v\n", indexes(db, "posts"))
}

func main() {
	db, _ := sql.Open("sqlite", "file:/tmp/upskill_blog_migration.db?cache=shared&_journal_mode=WAL")
	db.SetMaxOpenConns(1)
	defer db.Close()

	// Clean slate.
	for _, t := range []string{"post_tags", "tags", "posts", "authors", "schema_migrations"} {
		db.Exec(`DROP TABLE IF EXISTS ` + t)
	}
	for _, idx := range []string{"idx_posts_slug", "idx_posts_published", "idx_posts_author", "idx_post_tags_tag"} {
		db.Exec(`DROP INDEX IF EXISTS ` + idx)
	}

	m, err := newMigrator(db)
	if err != nil {
		panic(err)
	}

	check := func(label string, err error) bool {
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fmt.Printf("  ✗ %s: %v\n", label, err)
			return false
		}
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Printf("  ✓ %s: ErrNoChange\n", label)
		} else {
			fmt.Printf("  ✓ %s\n", label)
		}
		return true
	}

	ver := func() uint {
		v, _, err := m.Version()
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0
		}
		return v
	}

	fmt.Println("=== Blog Schema Versioning ===")
	fmt.Println()

	// ── APPLY MIGRATION 1 ─────────────────────────────────────────────────────
	fmt.Println("--- Step 1: Apply migration 1 (authors + posts) ---")
	check("apply step 1", m.Steps(1))
	fmt.Printf("  version: %d\n", ver())
	printState(db, "after migration 1")

	// Seed data at version 1.
	db.Exec(`INSERT INTO authors (name, email) VALUES ('Alice', 'alice@example.com')`)
	db.Exec(`INSERT INTO posts (author_id, title, body, published) VALUES (1, 'Hello World', 'First post!', 1)`)
	db.Exec(`INSERT INTO posts (author_id, title, body, published) VALUES (1, 'Go Tips', 'Some tips...', 0)`)
	var postCount int
	db.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&postCount)
	fmt.Printf("  seeded %d posts\n", postCount)

	// ── APPLY MIGRATION 2 ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Step 2: Apply migration 2 (tags) ---")
	check("apply step 2", m.Steps(1))
	fmt.Printf("  version: %d\n", ver())
	printState(db, "after migration 2")

	db.Exec(`INSERT OR IGNORE INTO tags (name) VALUES ('go'), ('backend'), ('tutorial')`)
	db.Exec(`INSERT INTO post_tags VALUES (1, 1), (1, 2)`) // hello world → go, backend
	db.Exec(`INSERT INTO post_tags VALUES (2, 1), (2, 3)`) // go tips → go, tutorial
	var tagCount int
	db.QueryRow(`SELECT COUNT(*) FROM tags`).Scan(&tagCount)
	fmt.Printf("  seeded %d tags\n", tagCount)

	// ── APPLY MIGRATION 3 ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Step 3: Apply migration 3 (slug + indexes) ---")
	check("apply step 3", m.Steps(1))
	fmt.Printf("  version: %d\n", ver())
	printState(db, "after migration 3")

	// Verify slugs were backfilled.
	rows, _ := db.Query(`SELECT title, slug FROM posts ORDER BY id`)
	defer rows.Close()
	for rows.Next() {
		var title, slug sql.NullString
		rows.Scan(&title, &slug)
		fmt.Printf("  title=%-20q  slug=%q\n", title.String, slug.String)
	}

	// ── IDEMPOTENT UP ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Up again (idempotent) ---")
	check("up (no change)", m.Up())

	// ── ROLL BACK TO V2 ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Roll back to version 2 ---")
	check("step -1", m.Steps(-1))
	fmt.Printf("  version: %d\n", ver())
	printState(db, "after rollback to v2")

	// ── ROLL BACK TO V1 ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Roll back to version 1 ---")
	check("migrate(1)", m.Migrate(1))
	fmt.Printf("  version: %d\n", ver())
	printState(db, "after rollback to v1")

	// Posts from seed still present (rollback only removed tags schema).
	db.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&postCount)
	fmt.Printf("  posts still present: %d\n", postCount)

	// ── REAPPLY ALL ───────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Reapply all migrations (Up) ---")
	check("up", m.Up())
	fmt.Printf("  version: %d\n", ver())
	printState(db, "final state")
}
