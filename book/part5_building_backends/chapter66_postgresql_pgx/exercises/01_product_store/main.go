// FILE: book/part5_building_backends/chapter66_postgresql_pgx/exercises/01_product_store/main.go
// CHAPTER: 66 — PostgreSQL with pgx
// EXERCISE: Build a product inventory store using pgx v5:
//   - Schema: products (id, name, description, price, category, stock, created_at)
//   - Schema: price_history (id, product_id, old_price, new_price, changed_at)
//   - Full CRUD with pgtype nullable fields
//   - UpdatePrice: records old price in price_history atomically
//   - BulkUpdateStock: batch update using pgx.Batch
//   - LowStockReport: returns products where stock < threshold
//   - SearchProducts: full-text ILIKE search across name and description
//
// Run (from the chapter folder):
//   go run ./exercises/01_product_store
//
// With a live Postgres:
//   DATABASE_URL=postgres://user:pass@localhost:5432/mydb go run ./exercises/01_product_store

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─────────────────────────────────────────────────────────────────────────────
// SCHEMA
// ─────────────────────────────────────────────────────────────────────────────

const ddl = `
CREATE TABLE IF NOT EXISTS products (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT           NOT NULL,
    description TEXT,
    price       NUMERIC(10,2)  NOT NULL,
    category    TEXT           NOT NULL,
    stock       INT            NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS price_history (
    id          BIGSERIAL PRIMARY KEY,
    product_id  BIGINT         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    old_price   NUMERIC(10,2)  NOT NULL,
    new_price   NUMERIC(10,2)  NOT NULL,
    changed_at  TIMESTAMPTZ    NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
CREATE INDEX IF NOT EXISTS idx_products_stock    ON products(stock);
`

// ─────────────────────────────────────────────────────────────────────────────
// TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Product struct {
	ID          int64
	Name        string
	Description pgtype.Text
	Price       float64
	Category    string
	Stock       int
	CreatedAt   time.Time
}

type PriceChange struct {
	ProductID int64
	OldPrice  float64
	NewPrice  float64
	ChangedAt time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// STORE
// ─────────────────────────────────────────────────────────────────────────────

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// ── CREATE ───────────────────────────────────────────────────────────────────

func (s *Store) CreateProduct(ctx context.Context, name, desc, category string, price float64, stock int) (*Product, error) {
	var p Product
	err := s.pool.QueryRow(ctx, `
		INSERT INTO products (name, description, price, category, stock)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, price, category, stock, created_at`,
		name,
		pgtype.Text{String: desc, Valid: desc != ""},
		price, category, stock,
	).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category, &p.Stock, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	return &p, nil
}

// ── READ ─────────────────────────────────────────────────────────────────────

func (s *Store) GetProduct(ctx context.Context, id int64) (*Product, error) {
	var p Product
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, description, price, category, stock, created_at
		FROM products WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category, &p.Stock, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("product %d: not found", id)
	}
	return &p, err
}

func (s *Store) ListByCategory(ctx context.Context, category string) ([]*Product, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, price, category, stock, created_at
		FROM products WHERE category = $1 ORDER BY name`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProducts(rows)
}

func (s *Store) SearchProducts(ctx context.Context, query string) ([]*Product, error) {
	pattern := "%" + query + "%"
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, price, category, stock, created_at
		FROM products
		WHERE name ILIKE $1 OR description ILIKE $1
		ORDER BY name`, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProducts(rows)
}

func (s *Store) LowStockReport(ctx context.Context, threshold int) ([]*Product, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, price, category, stock, created_at
		FROM products WHERE stock < $1 ORDER BY stock ASC, name`, threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProducts(rows)
}

// ── UPDATE ───────────────────────────────────────────────────────────────────

// UpdatePrice atomically records old price in price_history and updates the product.
func (s *Store) UpdatePrice(ctx context.Context, productID int64, newPrice float64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldPrice float64
	err = tx.QueryRow(ctx,
		`SELECT price FROM products WHERE id = $1 FOR UPDATE`, productID,
	).Scan(&oldPrice)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("product %d: not found", productID)
	}
	if err != nil {
		return fmt.Errorf("read price: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE products SET price = $1 WHERE id = $2`, newPrice, productID)
	if err != nil {
		return fmt.Errorf("update price: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO price_history (product_id, old_price, new_price) VALUES ($1, $2, $3)`,
		productID, oldPrice, newPrice,
	)
	if err != nil {
		return fmt.Errorf("record history: %w", err)
	}

	return tx.Commit(ctx)
}

// BulkUpdateStock sends all updates in a single batch round-trip.
func (s *Store) BulkUpdateStock(ctx context.Context, updates map[int64]int) (int, error) {
	batch := &pgx.Batch{}
	for id, stock := range updates {
		batch.Queue(`UPDATE products SET stock = $1 WHERE id = $2`, stock, id)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	updated := 0
	for range updates {
		ct, err := br.Exec()
		if err != nil {
			return updated, fmt.Errorf("batch update: %w", err)
		}
		updated += int(ct.RowsAffected())
	}
	return updated, br.Close()
}

// ── DELETE ───────────────────────────────────────────────────────────────────

func (s *Store) DeleteProduct(ctx context.Context, id int64) (bool, error) {
	ct, err := s.pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

// ── PRICE HISTORY ─────────────────────────────────────────────────────────────

func (s *Store) PriceHistory(ctx context.Context, productID int64) ([]*PriceChange, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT product_id, old_price, new_price, changed_at
		FROM price_history WHERE product_id = $1 ORDER BY changed_at`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*PriceChange
	for rows.Next() {
		var c PriceChange
		if err := rows.Scan(&c.ProductID, &c.OldPrice, &c.NewPrice, &c.ChangedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func scanProducts(rows pgx.Rows) ([]*Product, error) {
	var out []*Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category, &p.Stock, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func isDuplicateKey(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
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
	fmt.Println("=== Product Store (pgx v5) ===")
	fmt.Println()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil || pool.Ping(ctx) != nil {
		fmt.Printf("  (skip live queries: no Postgres available)\n\n")
		fmt.Println("To run with a real database:")
		fmt.Println("  DATABASE_URL=postgres://user:pass@localhost:5432/mydb go run ./exercises/01_product_store")
		fmt.Println()
		fmt.Println("--- Store API (code review) ---")
		fmt.Println("  CreateProduct  → INSERT ... RETURNING (one round trip)")
		fmt.Println("  GetProduct     → SELECT by primary key")
		fmt.Println("  ListByCategory → SELECT WHERE category = $1 ORDER BY name")
		fmt.Println("  SearchProducts → ILIKE on name OR description")
		fmt.Println("  LowStockReport → SELECT WHERE stock < $1 ORDER BY stock ASC")
		fmt.Println("  UpdatePrice    → serialized transaction: SELECT FOR UPDATE + UPDATE + INSERT history")
		fmt.Println("  BulkUpdateStock → pgx.Batch: N updates in one round trip")
		fmt.Println("  DeleteProduct  → DELETE, returns bool (was deleted?)")
		fmt.Println("  PriceHistory   → SELECT from price_history WHERE product_id = $1")
		return
	}
	defer pool.Close()

	store := NewStore(pool)

	// Schema.
	if _, err := pool.Exec(ctx, ddl); err != nil {
		fmt.Printf("  ✗ DDL: %v\n", err)
		return
	}
	fmt.Println("--- Schema ready ---")

	check := func(label string, err error) bool {
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", label, err)
			return false
		}
		fmt.Printf("  ✓ %s\n", label)
		return true
	}

	// ── CREATE ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Create products ---")
	kb, err := store.CreateProduct(ctx, "Mechanical Keyboard", "Cherry MX switches", "electronics", 129.99, 50)
	check("create keyboard", err)
	mouse, err := store.CreateProduct(ctx, "Wireless Mouse", "", "electronics", 49.99, 120)
	check("create mouse", err)
	chair, err := store.CreateProduct(ctx, "Ergonomic Chair", "Lumbar support", "furniture", 299.00, 10)
	check("create chair", err)
	desk, err := store.CreateProduct(ctx, "Standing Desk", "", "furniture", 499.00, 3)
	check("create desk (low stock)", err)
	lamp, err := store.CreateProduct(ctx, "Desk Lamp", "LED, 3 brightness levels", "home", 39.99, 0)
	check("create lamp (out of stock)", err)
	fmt.Printf("  ids: keyboard=%d mouse=%d chair=%d desk=%d lamp=%d\n",
		kb.ID, mouse.ID, chair.ID, desk.ID, lamp.ID)

	// ── READ ────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Read ---")
	p, err := store.GetProduct(ctx, kb.ID)
	if check("get keyboard", err) {
		desc := "NULL"
		if p.Description.Valid {
			desc = p.Description.String
		}
		fmt.Printf("  name=%q price=%.2f stock=%d desc=%s\n", p.Name, p.Price, p.Stock, desc)
	}
	_, err = store.GetProduct(ctx, 99999)
	fmt.Printf("  ✓ get missing: %v\n", err)

	// ── LIST BY CATEGORY ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- List by category ---")
	electronics, err := store.ListByCategory(ctx, "electronics")
	if check("list electronics", err) {
		for _, e := range electronics {
			fmt.Printf("  id=%-4d %-25s $%.2f  stock=%d\n", e.ID, e.Name, e.Price, e.Stock)
		}
	}

	// ── SEARCH ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Search products ---")
	results, err := store.SearchProducts(ctx, "desk")
	if check("search 'desk'", err) {
		for _, r := range results {
			fmt.Printf("  %s (category=%s)\n", r.Name, r.Category)
		}
	}

	// ── LOW STOCK REPORT ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Low stock report (threshold=10) ---")
	low, err := store.LowStockReport(ctx, 10)
	if check("low stock", err) {
		for _, l := range low {
			fmt.Printf("  %-20s stock=%d\n", l.Name, l.Stock)
		}
	}

	// ── UPDATE PRICE ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- UpdatePrice (with history) ---")
	check("update keyboard price to 109.99", store.UpdatePrice(ctx, kb.ID, 109.99))
	check("update keyboard price to 119.99", store.UpdatePrice(ctx, kb.ID, 119.99))

	history, err := store.PriceHistory(ctx, kb.ID)
	if check("price history", err) {
		for _, h := range history {
			fmt.Printf("  %.2f → %.2f  at %s\n", h.OldPrice, h.NewPrice, h.ChangedAt.Format("15:04:05"))
		}
	}

	// ── BULK STOCK UPDATE ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- BulkUpdateStock (batch) ---")
	n, err := store.BulkUpdateStock(ctx, map[int64]int{
		lamp.ID:  25,
		desk.ID:  15,
		chair.ID: 8,
	})
	if check(fmt.Sprintf("bulk update (%d rows)", n), err) {
		low2, _ := store.LowStockReport(ctx, 10)
		fmt.Printf("  low stock items after update: %d\n", len(low2))
		for _, l := range low2 {
			fmt.Printf("  %-20s stock=%d\n", l.Name, l.Stock)
		}
	}

	// ── DELETE ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Delete ---")
	deleted, err := store.DeleteProduct(ctx, mouse.ID)
	check("delete mouse", err)
	fmt.Printf("  deleted=%v\n", deleted)
	deleted, _ = store.DeleteProduct(ctx, 99999)
	fmt.Printf("  ✓ delete non-existent: deleted=%v\n", deleted)

	// ── DUPLICATE KEY ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Error handling ---")
	fmt.Printf("  isDuplicateKey check: %v (not triggered here, products have no unique constraint beyond PK)\n",
		isDuplicateKey(nil))
}
