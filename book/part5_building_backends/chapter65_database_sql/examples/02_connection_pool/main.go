// FILE: book/part5_building_backends/chapter65_database_sql/examples/02_connection_pool/main.go
// CHAPTER: 65 — database/sql
// TOPIC: Connection pool configuration, context-aware queries,
//        QueryContext / ExecContext, connection lifecycle,
//        and concurrent access patterns.
//
// Run (from the chapter folder):
//   go run ./examples/02_connection_pool

package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// ─────────────────────────────────────────────────────────────────────────────
// CONNECTION POOL SETTINGS
// ─────────────────────────────────────────────────────────────────────────────

// configurePool shows recommended settings for a production PostgreSQL/MySQL pool.
// Not called directly in main() — settings are set inline for SQLite compatibility.
func configurePool(db *sql.DB) {
	db.SetMaxOpenConns(25)        // max TCP connections to DB
	db.SetMaxIdleConns(25)        // idle connections kept in pool
	db.SetConnMaxLifetime(5 * time.Minute) // reuse limit
	db.SetConnMaxIdleTime(2 * time.Minute) // idle timeout
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT-AWARE QUERIES
//
// Always use *Context variants in production:
//   QueryContext, QueryRowContext, ExecContext, BeginTx
//
// When the context is cancelled (request timeout, client disconnect),
// the database driver cancels the in-flight query.
// ─────────────────────────────────────────────────────────────────────────────

type Product struct {
	ID    int
	Name  string
	Price float64
}

func getProductCtx(ctx context.Context, db *sql.DB, id int) (*Product, error) {
	var p Product
	err := db.QueryRowContext(ctx,
		`SELECT id, name, price FROM products WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.Price)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func listProductsCtx(ctx context.Context, db *sql.DB) ([]*Product, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, name, price FROM products ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// CONCURRENT ACCESS
// ─────────────────────────────────────────────────────────────────────────────

// concurrentReads fires N goroutines each querying a different product.
// sql.DB is safe for concurrent use — each goroutine borrows a connection
// from the pool and returns it when done.
func concurrentReads(db *sql.DB, n int) {
	var wg sync.WaitGroup
	start := time.Now()
	for i := 1; i <= n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			p, err := getProductCtx(ctx, db, id)
			if err != nil {
				fmt.Printf("    goroutine %d: %v\n", id, err)
			} else {
				_ = p // success — don't print to avoid interleaved output
			}
		}(i)
	}
	wg.Wait()
	fmt.Printf("  %d concurrent reads completed in %dms\n", n, time.Since(start).Milliseconds())
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT CANCELLATION DEMO
// ─────────────────────────────────────────────────────────────────────────────

func cancelledQuery(db *sql.DB) {
	// Cancel immediately — the query should not proceed.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel before querying

	err := db.QueryRowContext(ctx, `SELECT count(*) FROM products`).Scan(new(int))
	if err != nil {
		fmt.Printf("  cancelled query error: %v\n", err)
	}
}

func timeoutQuery(db *sql.DB) {
	// Very short timeout — expires before the query returns.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	err := db.QueryRowContext(ctx, `SELECT count(*) FROM products`).Scan(new(int))
	if err != nil {
		fmt.Printf("  timeout query error: %v\n", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// POOL STATS
// ─────────────────────────────────────────────────────────────────────────────

func printStats(db *sql.DB, label string) {
	s := db.Stats()
	fmt.Printf("  [%s] open=%d idle=%d inuse=%d waitCount=%d waitDuration=%s\n",
		label, s.OpenConnections, s.Idle, s.InUse, s.WaitCount, s.WaitDuration)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// file::memory:?cache=shared — all connections share the same in-memory database.
	// SQLite doesn't support true concurrent writes, so we cap at 1 open connection.
	// In a real PostgreSQL/MySQL deployment, MaxOpenConns(25) makes sense.
	db, _ := sql.Open("sqlite", "file::memory:?cache=shared")
	defer db.Close()

	// For SQLite: use 1 connection (SQLite serializes writes).
	// The settings below are shown for educational purposes — use these values
	// with PostgreSQL/MySQL in production.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	// Seed data.
	db.Exec(`CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL)`)
	for i := 1; i <= 10; i++ {
		db.Exec(`INSERT INTO products VALUES (?, ?, ?)`, i, fmt.Sprintf("Product %d", i), float64(i)*9.99)
	}

	fmt.Println("=== Connection Pool & Context ===")

	// ── POOL SETTINGS ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Pool configuration ---")
	fmt.Println("  MaxOpenConns(25)      — max TCP connections to DB")
	fmt.Println("  MaxIdleConns(25)      — max idle connections kept in pool")
	fmt.Println("  ConnMaxLifetime(5min) — reuse limit per connection")
	fmt.Println("  ConnMaxIdleTime(2min) — idle timeout per connection")
	printStats(db, "after seed")

	// ── CONTEXT QUERIES ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Context-aware queries ---")
	ctx := context.Background()
	p, err := getProductCtx(ctx, db, 3)
	if err == nil {
		fmt.Printf("  ✓ getProductCtx(3): id=%d name=%q price=%.2f\n", p.ID, p.Name, p.Price)
	}

	products, err := listProductsCtx(ctx, db)
	if err == nil {
		fmt.Printf("  ✓ listProductsCtx: returned %d products\n", len(products))
	}

	// ── CONCURRENT ACCESS ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Concurrent access (sql.DB is goroutine-safe) ---")
	concurrentReads(db, 10)
	printStats(db, "after concurrent reads")

	// ── CONTEXT CANCELLATION ─────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Context cancellation ---")
	cancelledQuery(db)
	timeoutQuery(db)

	// ── CONNECTION LIFECYCLE ─────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Connection lifecycle ---")
	fmt.Println("  sql.Open()   → validates driver name, does NOT open a connection")
	fmt.Println("  db.Ping()    → opens the first real connection, verifies reachability")
	fmt.Println("  QueryRow()   → borrows a connection from the pool, returns it on Rows.Close()")
	fmt.Println("  db.Close()   → closes all idle connections; blocks until active ones finish")
	fmt.Println()
	fmt.Println("  Always call:")
	fmt.Println("    defer rows.Close()  — returns connection to pool")
	fmt.Println("    rows.Err()          — checks for iteration errors")
	fmt.Println("    defer tx.Rollback() — safe no-op if already committed")

	// ── TRANSACTION WITH CONTEXT ─────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Transaction with context ---")
	txCtx, txCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer txCancel()

	tx, err := db.BeginTx(txCtx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	})
	if err != nil {
		fmt.Printf("  ✗ begin tx: %v\n", err)
		return
	}
	defer tx.Rollback()

	var count int
	tx.QueryRowContext(txCtx, `SELECT count(*) FROM products`).Scan(&count)
	tx.ExecContext(txCtx, `UPDATE products SET price = price * 1.1`)
	var newCount int
	tx.QueryRowContext(txCtx, `SELECT count(*) FROM products`).Scan(&newCount)

	if err := tx.Commit(); err != nil {
		fmt.Printf("  ✗ commit: %v\n", err)
		return
	}
	fmt.Printf("  ✓ transaction: updated %d products (count before=%d after=%d)\n", count, count, newCount)

	// Verify price update.
	p2, err2 := getProductCtx(context.Background(), db, 1)
	if err2 == nil {
		fmt.Printf("  product 1 price after 10%% increase: %.2f\n", p2.Price)
	}
}
