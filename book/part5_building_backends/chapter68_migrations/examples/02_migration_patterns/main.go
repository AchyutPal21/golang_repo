// FILE: book/part5_building_backends/chapter68_migrations/examples/02_migration_patterns/main.go
// CHAPTER: 68 — Migrations
// TOPIC: Migration patterns for production:
//        zero-downtime migrations, backfill patterns, rename columns
//        via expand/contract, data migrations, and dirty state recovery.
//
// Run (from the chapter folder):
//   go run ./examples/02_migration_patterns

package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// ─────────────────────────────────────────────────────────────────────────────
// EXPAND / CONTRACT PATTERN
//
// Safe zero-downtime rename of a column:
//   Step 1 (Expand):   Add new column. Write to both old and new.
//   Step 2 (Backfill): Copy old data to new column.
//   Step 3 (Switch):   Deploy code that only reads from new column.
//   Step 4 (Contract): Drop old column.
// ─────────────────────────────────────────────────────────────────────────────

func demoExpandContract(db *sql.DB) {
	fmt.Println("--- Expand / Contract: rename 'fullname' → 'display_name' ---")

	// Initial schema with old column name.
	db.Exec(`DROP TABLE IF EXISTS profiles`)
	db.Exec(`CREATE TABLE profiles (id INTEGER PRIMARY KEY, fullname TEXT, email TEXT)`)
	db.Exec(`INSERT INTO profiles VALUES (1, 'Alice Smith', 'alice@example.com')`)
	db.Exec(`INSERT INTO profiles VALUES (2, 'Bob Jones', 'bob@example.com')`)

	showProfileCols := func(label string) {
		rows, _ := db.Query(`PRAGMA table_info(profiles)`)
		defer rows.Close()
		var cols []string
		for rows.Next() {
			var cid int
			var name, colType string
			var notnull, pk int
			var dflt interface{}
			rows.Scan(&cid, &name, &colType, &notnull, &dflt, &pk)
			cols = append(cols, name)
		}
		fmt.Printf("  [%s] columns: %v\n", label, cols)
	}

	// STEP 1: Expand — add new column alongside old.
	db.Exec(`ALTER TABLE profiles ADD COLUMN display_name TEXT`)
	showProfileCols("after expand")
	fmt.Println("  → deploy code that writes to BOTH fullname and display_name")

	// STEP 2: Backfill — copy existing data.
	res, _ := db.Exec(`UPDATE profiles SET display_name = fullname WHERE display_name IS NULL`)
	n, _ := res.RowsAffected()
	fmt.Printf("  backfilled %d rows\n", n)

	// Verify.
	rows, _ := db.Query(`SELECT id, fullname, display_name FROM profiles`)
	defer rows.Close()
	for rows.Next() {
		var id int
		var old, new_ sql.NullString
		rows.Scan(&id, &old, &new_)
		fmt.Printf("  id=%d  fullname=%q  display_name=%q\n", id, old.String, new_.String)
	}

	// STEP 3: Switch — code now reads only display_name. Old column still present.
	fmt.Println("  → deploy code that reads ONLY display_name")

	// STEP 4: Contract — drop old column (SQLite: recreate table).
	db.Exec(`
		CREATE TABLE profiles_new (id INTEGER PRIMARY KEY, display_name TEXT, email TEXT);
		INSERT INTO profiles_new SELECT id, display_name, email FROM profiles;
		DROP TABLE profiles;
		ALTER TABLE profiles_new RENAME TO profiles;
	`)
	showProfileCols("after contract (dropped fullname)")
}

// ─────────────────────────────────────────────────────────────────────────────
// DATA MIGRATION — compute and store derived values
// ─────────────────────────────────────────────────────────────────────────────

func demoDataMigration(db *sql.DB) {
	fmt.Println()
	fmt.Println("--- Data migration: compute 'slug' from 'title' ---")

	db.Exec(`DROP TABLE IF EXISTS articles`)
	db.Exec(`CREATE TABLE articles (id INTEGER PRIMARY KEY, title TEXT NOT NULL, slug TEXT)`)
	titles := []string{"Hello World", "Go Is Awesome", "Building REST APIs"}
	for i, t := range titles {
		db.Exec(`INSERT INTO articles VALUES (?, ?, NULL)`, i+1, t)
	}

	// Migration: add slug column and populate it.
	slugify := func(s string) string {
		out := []byte{}
		for _, c := range []byte(s) {
			switch {
			case c >= 'A' && c <= 'Z':
				out = append(out, c+32)
			case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
				out = append(out, c)
			case c == ' ':
				out = append(out, '-')
			}
		}
		return string(out)
	}

	// In a real migration this would be done in SQL or a Go program run once.
	rows, _ := db.Query(`SELECT id, title FROM articles WHERE slug IS NULL`)
	type row struct{ id int; title string }
	var pending []row
	for rows.Next() {
		var r row
		rows.Scan(&r.id, &r.title)
		pending = append(pending, r)
	}
	rows.Close()

	for _, r := range pending {
		db.Exec(`UPDATE articles SET slug = ? WHERE id = ?`, slugify(r.title), r.id)
	}

	rows2, _ := db.Query(`SELECT id, title, slug FROM articles`)
	defer rows2.Close()
	for rows2.Next() {
		var id int
		var title, slug string
		rows2.Scan(&id, &title, &slug)
		fmt.Printf("  id=%d  title=%-25q  slug=%q\n", id, title, slug)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// IDEMPOTENT MIGRATIONS
// ─────────────────────────────────────────────────────────────────────────────

func demoIdempotent(db *sql.DB) {
	fmt.Println()
	fmt.Println("--- Idempotent migration patterns ---")

	// CREATE TABLE IF NOT EXISTS — safe to run multiple times.
	db.Exec(`CREATE TABLE IF NOT EXISTS tags (id INTEGER PRIMARY KEY, name TEXT UNIQUE)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS tags (id INTEGER PRIMARY KEY, name TEXT UNIQUE)`) // no error
	fmt.Println("  ✓ CREATE TABLE IF NOT EXISTS: safe to run multiple times")

	// CREATE INDEX IF NOT EXISTS.
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name)`) // no error
	fmt.Println("  ✓ CREATE INDEX IF NOT EXISTS: safe to run multiple times")

	// INSERT OR IGNORE — idempotent seed data.
	db.Exec(`INSERT OR IGNORE INTO tags (name) VALUES ('go'), ('backend'), ('database')`)
	db.Exec(`INSERT OR IGNORE INTO tags (name) VALUES ('go'), ('backend'), ('database')`) // no duplicates
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM tags`).Scan(&count)
	fmt.Printf("  ✓ INSERT OR IGNORE: %d tags (no duplicates)\n", count)
}

// ─────────────────────────────────────────────────────────────────────────────
// DIRTY STATE RECOVERY
// ─────────────────────────────────────────────────────────────────────────────

func demoDirtyState() {
	fmt.Println()
	fmt.Println("--- Dirty migration state ---")
	fmt.Println(`
  When a migration fails mid-way, golang-migrate marks the version as dirty:
    schema_migrations: version=5, dirty=true

  This prevents further migrations until resolved. Options:

  1. Fix the SQL error, then force-reset the version:
     m.Force(4)  // go back to last clean version

  2. Manually fix the partial migration in the database, then:
     m.Force(5)  // mark current version as clean

  3. In golang-migrate:
     m.Force(int(version))  // clears the dirty flag

  Prevention:
  - Wrap migrations in transactions (most DDL in PostgreSQL is transactional)
  - SQLite: DDL is auto-committed; partial failures need manual recovery
  - PostgreSQL: transactions allow atomic DDL — migrations either fully apply or roll back`)
}

// ─────────────────────────────────────────────────────────────────────────────
// MIGRATION BEST PRACTICES
// ─────────────────────────────────────────────────────────────────────────────

func showBestPractices() {
	fmt.Println()
	fmt.Println("--- Migration best practices ---")
	practices := []string{
		"Never edit a migration that has been applied — add a new one",
		"Always write a down migration, even if it's 'DROP TABLE' (for rollback in CI)",
		"Use IF NOT EXISTS / IF EXISTS for idempotent DDL",
		"Large backfills: batch in chunks (avoid long transactions, table locks)",
		"Add indexes CONCURRENTLY in PostgreSQL to avoid table locks",
		"For zero-downtime: expand → backfill → switch reads → contract (drop old)",
		"Store migration files in version control alongside application code",
		"Run migrations in CI/CD before deploying the new application version",
		"Use advisory locks (pg_advisory_lock) to prevent concurrent migration runs",
		"Test rollback in staging before production deploys",
	}
	for i, p := range practices {
		fmt.Printf("  %2d. %s\n", i+1, p)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	db, _ := sql.Open("sqlite", "file::memory:?cache=shared")
	db.SetMaxOpenConns(1)
	defer db.Close()

	fmt.Println("=== Migration Patterns ===")
	fmt.Println()

	demoExpandContract(db)
	demoDataMigration(db)
	demoIdempotent(db)
	demoDirtyState()
	showBestPractices()
}
