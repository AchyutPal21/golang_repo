// FILE: book/part5_building_backends/chapter66_postgresql_pgx/examples/01_pgx_basics/main.go
// CHAPTER: 66 — PostgreSQL with pgx
// TOPIC: pgx v5 fundamentals — direct connection vs pool, named arguments,
//        pgx type system (pgtype), batch queries, CopyFrom for bulk insert,
//        and graceful fallback when no Postgres is available.
//
// Run (from the chapter folder):
//   go run ./examples/01_pgx_basics
//
// To run against a real PostgreSQL instance:
//   DATABASE_URL=postgres://user:pass@localhost:5432/mydb go run ./examples/01_pgx_basics

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─────────────────────────────────────────────────────────────────────────────
// SCHEMA (run once before connecting)
// ─────────────────────────────────────────────────────────────────────────────

const ddl = `
CREATE TABLE IF NOT EXISTS products (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT,
    price       NUMERIC(10,2) NOT NULL,
    category    TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

// ─────────────────────────────────────────────────────────────────────────────
// TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Product struct {
	ID          int64
	Name        string
	Description *string // nullable
	Price       float64
	Category    string
	CreatedAt   time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// DIRECT CONNECTION — pgx.Connect
// Use for one-off scripts or tests; not suitable for HTTP servers.
// ─────────────────────────────────────────────────────────────────────────────

func demoDirectConnection(ctx context.Context, dsn string) {
	fmt.Println("--- Direct connection (pgx.Connect) ---")

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		fmt.Printf("  (skip: %v)\n", err)
		return
	}
	defer conn.Close(ctx)

	// pgx uses $1, $2 ... placeholders (not ?)
	var version string
	err = conn.QueryRow(ctx, `SELECT version()`).Scan(&version)
	if err != nil {
		fmt.Printf("  ✗ version query: %v\n", err)
		return
	}
	fmt.Printf("  ✓ connected: %s\n", version[:40])

	// DDL
	if _, err := conn.Exec(ctx, ddl); err != nil {
		fmt.Printf("  ✗ DDL: %v\n", err)
		return
	}
	fmt.Println("  ✓ schema ready")
}

// ─────────────────────────────────────────────────────────────────────────────
// POOL — pgxpool.New
// Use in HTTP servers — goroutine-safe, manages connections automatically.
// ─────────────────────────────────────────────────────────────────────────────

func buildPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Pool tuning — mirror database/sql settings.
	cfg.MaxConns = 25
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 5 * time.Minute
	cfg.MaxConnIdleTime = 2 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	// AfterConnect: run setup per-connection (e.g. SET search_path).
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// In production: register custom types, set session params.
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// CRUD USING POOL
// ─────────────────────────────────────────────────────────────────────────────

func createProduct(ctx context.Context, pool *pgxpool.Pool, name, desc, category string, price float64) (*Product, error) {
	var p Product
	// RETURNING clause — get the inserted row back in one round trip.
	err := pool.QueryRow(ctx, `
		INSERT INTO products (name, description, price, category)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, description, price, category, created_at`,
		name, nullableStr(desc), price, category,
	).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category, &p.CreatedAt)
	return &p, err
}

func getProduct(ctx context.Context, pool *pgxpool.Pool, id int64) (*Product, error) {
	var p Product
	err := pool.QueryRow(ctx, `
		SELECT id, name, description, price, category, created_at
		FROM products WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category, &p.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("product %d: not found", id)
	}
	return &p, err
}

func listProducts(ctx context.Context, pool *pgxpool.Pool, category string) ([]*Product, error) {
	var rows pgx.Rows
	var err error
	if category != "" {
		rows, err = pool.Query(ctx,
			`SELECT id, name, description, price, category, created_at FROM products WHERE category = $1 ORDER BY id`,
			category)
	} else {
		rows, err = pool.Query(ctx,
			`SELECT id, name, description, price, category, created_at FROM products ORDER BY id`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Product])
	if err != nil {
		return nil, err
	}
	return products, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// NAMED ARGUMENTS — pgx.NamedArgs
// ─────────────────────────────────────────────────────────────────────────────

func createProductNamed(ctx context.Context, pool *pgxpool.Pool, p Product) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO products (name, price, category)
		VALUES (@name, @price, @category)`,
		pgx.NamedArgs{
			"name":     p.Name,
			"price":    p.Price,
			"category": p.Category,
		},
	)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// BATCH QUERIES — send multiple statements in one round trip
// ─────────────────────────────────────────────────────────────────────────────

func batchInsert(ctx context.Context, pool *pgxpool.Pool, items []Product) error {
	batch := &pgx.Batch{}
	for _, item := range items {
		batch.Queue(
			`INSERT INTO products (name, price, category) VALUES ($1, $2, $3)`,
			item.Name, item.Price, item.Category,
		)
	}

	br := pool.SendBatch(ctx, batch)
	defer br.Close()

	for range items {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch exec: %w", err)
		}
	}
	return br.Close()
}

// ─────────────────────────────────────────────────────────────────────────────
// COPY FROM — fastest bulk insert via Postgres COPY protocol
// ─────────────────────────────────────────────────────────────────────────────

func bulkCopy(ctx context.Context, conn *pgx.Conn, items []Product) (int64, error) {
	rows := make([][]any, len(items))
	for i, item := range items {
		rows[i] = []any{item.Name, item.Price, item.Category}
	}

	n, err := conn.CopyFrom(
		ctx,
		pgx.Identifier{"products"},
		[]string{"name", "price", "category"},
		pgx.CopyFromRows(rows),
	)
	return n, err
}

// ─────────────────────────────────────────────────────────────────────────────
// ERROR HANDLING — pgconn.PgError for Postgres error codes
// ─────────────────────────────────────────────────────────────────────────────

func isDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if ok := errorAs(err, &pgErr); ok {
		return pgErr.Code == "23505" // unique_violation
	}
	return false
}

// errorAs is a thin wrapper so we can call errors.As without importing errors.
func errorAs(err error, target any) bool {
	type asInterface interface{ As(any) bool }
	if a, ok := err.(asInterface); ok {
		return a.As(target)
	}
	// fallback: walk the chain manually
	for err != nil {
		if pgErr, ok := target.(**pgconn.PgError); ok {
			if e, ok := err.(*pgconn.PgError); ok {
				*pgErr = e
				return true
			}
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
		} else {
			break
		}
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/upskill_go?sslmode=disable"
	}

	ctx := context.Background()

	fmt.Println("=== pgx v5 Basics ===")
	fmt.Println()

	// ── DIRECT CONNECTION ─────────────────────────────────────────────────────
	demoDirectConnection(ctx, dsn)

	// ── POOL ─────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Connection pool (pgxpool) ---")
	pool, err := buildPool(ctx, dsn)
	if err != nil {
		fmt.Printf("  (skip live queries: %v)\n\n", err)
		demoAPIShapes()
		return
	}
	defer pool.Close()

	stats := pool.Stat()
	fmt.Printf("  ✓ pool ready: total=%d idle=%d\n", stats.TotalConns(), stats.IdleConns())

	// ── CRUD ─────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- CRUD with RETURNING ---")

	p1, err := createProduct(ctx, pool, "Mechanical Keyboard", "Cherry MX Blue", "electronics", 129.99)
	if err != nil {
		fmt.Printf("  ✗ create: %v\n", err)
		return
	}
	fmt.Printf("  ✓ created: id=%d name=%q price=%.2f\n", p1.ID, p1.Name, p1.Price)

	p2, err := createProduct(ctx, pool, "Ergonomic Chair", "", "furniture", 299.00)
	if err != nil {
		fmt.Printf("  ✗ create: %v\n", err)
		return
	}
	fmt.Printf("  ✓ created: id=%d name=%q\n", p2.ID, p2.Name)

	fetched, err := getProduct(ctx, pool, p1.ID)
	if err != nil {
		fmt.Printf("  ✗ get: %v\n", err)
	} else {
		fmt.Printf("  ✓ fetched: id=%d name=%q\n", fetched.ID, fetched.Name)
	}

	// ── NAMED ARGS ────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Named arguments ---")
	err = createProductNamed(ctx, pool, Product{Name: "Standing Desk", Price: 499.00, Category: "furniture"})
	if err != nil {
		fmt.Printf("  ✗ named insert: %v\n", err)
	} else {
		fmt.Println("  ✓ named insert: ok")
	}

	// ── LIST ─────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- CollectRows (struct scan) ---")
	all, err := listProducts(ctx, pool, "")
	if err != nil {
		fmt.Printf("  ✗ list: %v\n", err)
	} else {
		fmt.Printf("  ✓ total products: %d\n", len(all))
		for _, pp := range all {
			fmt.Printf("  id=%-3d %-25s $%.2f  cat=%s\n", pp.ID, pp.Name, pp.Price, pp.Category)
		}
	}

	// ── BATCH ────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Batch insert ---")
	err = batchInsert(ctx, pool, []Product{
		{Name: "Monitor 27\"", Price: 349.00, Category: "electronics"},
		{Name: "USB Hub", Price: 39.99, Category: "electronics"},
		{Name: "Desk Lamp", Price: 49.99, Category: "home"},
	})
	if err != nil {
		fmt.Printf("  ✗ batch: %v\n", err)
	} else {
		fmt.Println("  ✓ batch insert: 3 rows")
	}

	// ── COPY FROM ────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- CopyFrom (bulk insert) ---")
	conn, err := pool.Acquire(ctx)
	if err != nil {
		fmt.Printf("  ✗ acquire: %v\n", err)
	} else {
		n, err := bulkCopy(ctx, conn.Conn(), []Product{
			{Name: "Mousepad XL", Price: 24.99, Category: "electronics"},
			{Name: "Cable Organiser", Price: 14.99, Category: "home"},
		})
		conn.Release()
		if err != nil {
			fmt.Printf("  ✗ CopyFrom: %v\n", err)
		} else {
			fmt.Printf("  ✓ CopyFrom: %d rows inserted\n", n)
		}
	}

	// Final count
	all2, _ := listProducts(ctx, pool, "")
	fmt.Printf("\n  Total products after all inserts: %d\n", len(all2))
}

// ─────────────────────────────────────────────────────────────────────────────
// API SHAPES — shown when no Postgres is available
// Demonstrates what the code structure looks like without a live DB.
// ─────────────────────────────────────────────────────────────────────────────

func demoAPIShapes() {
	fmt.Println("=== pgx API shapes (no live DB) ===")
	fmt.Println()

	fmt.Println("--- pgx.Connect DSN format ---")
	fmt.Println(`  postgres://user:pass@host:5432/dbname?sslmode=disable`)
	fmt.Println(`  host=localhost port=5432 user=postgres password=secret dbname=mydb sslmode=disable`)

	fmt.Println()
	fmt.Println("--- Pool config keys ---")
	fmt.Println("  MaxConns           = 25")
	fmt.Println("  MinConns           = 2")
	fmt.Println("  MaxConnLifetime    = 5m")
	fmt.Println("  MaxConnIdleTime    = 2m")
	fmt.Println("  HealthCheckPeriod  = 30s")

	fmt.Println()
	fmt.Println("--- pgx vs database/sql differences ---")
	diffs := [][2]string{
		{"Placeholders", "pgx: $1 $2   |  database/sql: ? (driver-specific)"},
		{"Scan struct", "pgx: pgx.RowToAddrOfStructByName[T]  |  sql: manual Scan"},
		{"Batch", "pgx: pgx.Batch{} in one round trip  |  sql: N separate queries"},
		{"Bulk insert", "pgx: CopyFrom (Postgres COPY protocol)  |  sql: no equivalent"},
		{"Named args", `pgx: @name syntax  |  sql: driver-dependent`},
		{"NULL", "pgx: *T (pointer) or pgtype.Text  |  sql: sql.NullString etc."},
		{"Error codes", "pgx: pgconn.PgError.Code  |  sql: driver-specific"},
	}
	for _, d := range diffs {
		fmt.Printf("  %-15s  %s\n", d[0], d[1])
	}

	fmt.Println()
	fmt.Println("--- Error code reference ---")
	codes := [][2]string{
		{"23505", "unique_violation  — duplicate key"},
		{"23503", "foreign_key_violation"},
		{"23502", "not_null_violation"},
		{"40001", "serialization_failure — retry transaction"},
		{"42P01", "undefined_table"},
		{"08006", "connection_failure"},
	}
	for _, c := range codes {
		fmt.Printf("  %s  %s\n", c[0], c[1])
	}
}
