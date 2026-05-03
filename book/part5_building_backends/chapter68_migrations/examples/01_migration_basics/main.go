// FILE: book/part5_building_backends/chapter68_migrations/examples/01_migration_basics/main.go
// CHAPTER: 68 — Migrations
// TOPIC: Database schema migrations with golang-migrate:
//        embed SQL files, apply Up/Down, version inspection,
//        idempotent migrations, and the schema_migrations table.
//
// Run (from the chapter folder):
//   go run ./examples/01_migration_basics

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

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func newMigrator(db *sql.DB) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("create iofs source: %w", err)
	}

	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return nil, fmt.Errorf("create sqlite driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}
	return m, nil
}

func currentVersion(m *migrate.Migrate) (uint, bool, error) {
	return m.Version()
}

func showTables(db *sql.DB) {
	rows, _ := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
	}
	fmt.Printf("  tables: %v\n", tables)
}

func showColumns(db *sql.DB, table string) {
	rows, _ := db.Query(`PRAGMA table_info(` + table + `)`)
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
	fmt.Printf("  %s columns: %v\n", table, cols)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// Use a file-based SQLite so migrations persist and can be verified.
	// We use a temp file instead of :memory: because golang-migrate needs a real path.
	db, err := sql.Open("sqlite", "file:/tmp/upskill_migrations_demo.db?cache=shared&_journal_mode=WAL")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	// Clean slate for demo.
	db.Exec(`DROP TABLE IF EXISTS user_settings`)
	db.Exec(`DROP TABLE IF EXISTS users`)
	db.Exec(`DROP TABLE IF EXISTS schema_migrations`)

	fmt.Println("=== Database Migrations (golang-migrate) ===")
	fmt.Println()

	m, err := newMigrator(db)
	if err != nil {
		panic(err)
	}

	// ── VERSION BEFORE ──────────────────────────────────────────────────────
	fmt.Println("--- Initial state ---")
	ver, dirty, err := currentVersion(m)
	if errors.Is(err, migrate.ErrNilVersion) {
		fmt.Println("  version: none (no migrations applied)")
	} else {
		fmt.Printf("  version: %d  dirty: %v\n", ver, dirty)
	}
	showTables(db)

	// ── APPLY ONE STEP ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Apply migration 1 (create users table) ---")
	if err := m.Steps(1); err != nil {
		fmt.Printf("  ✗ step 1: %v\n", err)
	} else {
		ver, _, _ = currentVersion(m)
		fmt.Printf("  ✓ version: %d\n", ver)
		showTables(db)
		showColumns(db, "users")
	}

	// ── APPLY NEXT STEP ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Apply migration 2 (add profile columns + settings table) ---")
	if err := m.Steps(1); err != nil {
		fmt.Printf("  ✗ step 2: %v\n", err)
	} else {
		ver, _, _ = currentVersion(m)
		fmt.Printf("  ✓ version: %d\n", ver)
		showTables(db)
		showColumns(db, "users")
	}

	// ── APPLY ALL REMAINING ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Apply all remaining migrations (Up) ---")
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		fmt.Printf("  ✗ up: %v\n", err)
	} else {
		ver, _, _ = currentVersion(m)
		fmt.Printf("  ✓ version: %d\n", ver)
		showTables(db)
		showColumns(db, "users")
	}

	// ── IDEMPOTENT — run Up again ────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Run Up again (should be no-change) ---")
	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("  ✓ ErrNoChange — already at latest version")
	} else if err != nil {
		fmt.Printf("  ✗ unexpected error: %v\n", err)
	}

	// ── SCHEMA MIGRATIONS TABLE ──────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- schema_migrations table ---")
	rows, _ := db.Query(`SELECT version, dirty FROM schema_migrations`)
	defer rows.Close()
	for rows.Next() {
		var v uint
		var dirty bool
		rows.Scan(&v, &dirty)
		fmt.Printf("  version=%d  dirty=%v\n", v, dirty)
	}

	// ── ROLL BACK ONE STEP ───────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Roll back one step (Down 1) ---")
	if err := m.Steps(-1); err != nil {
		fmt.Printf("  ✗ step -1: %v\n", err)
	} else {
		ver, _, _ = currentVersion(m)
		fmt.Printf("  ✓ version: %d\n", ver)
		showTables(db)
	}

	// ── MIGRATE TO SPECIFIC VERSION ──────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Migrate to specific version (1) ---")
	if err := m.Migrate(1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		fmt.Printf("  ✗ migrate(1): %v\n", err)
	} else {
		ver, _, _ = currentVersion(m)
		fmt.Printf("  ✓ version: %d\n", ver)
		showTables(db)
	}

	// ── APPLY ALL AGAIN ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Reapply all (Up from version 1) ---")
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		fmt.Printf("  ✗ up: %v\n", err)
	} else {
		ver, _, _ = currentVersion(m)
		fmt.Printf("  ✓ version: %d\n", ver)
		showColumns(db, "users")
	}

	fmt.Println()
	fmt.Println("=== Summary ===")
	fmt.Println("  m.Up()       — apply all pending migrations")
	fmt.Println("  m.Steps(n)   — apply n steps (negative = rollback n steps)")
	fmt.Println("  m.Migrate(v) — go to exact version v")
	fmt.Println("  m.Down()     — roll back all migrations")
	fmt.Println("  m.Version()  — current version, dirty flag")
	fmt.Println("  ErrNoChange  — returned when already at target version")
}
