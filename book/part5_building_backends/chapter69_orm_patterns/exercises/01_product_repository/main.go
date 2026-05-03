// FILE: book/part5_building_backends/chapter69_orm_patterns/exercises/01_product_repository/main.go
// CHAPTER: 69 — ORM vs Builder vs Raw SQL
// EXERCISE: Build a complete product catalog using the repository pattern:
//   - ProductRepository interface with CRUD + search + stock management
//   - In-memory implementation (for tests)
//   - SQL implementation with dynamic filter builder
//   - CachingRepository wrapping any repo — adds read-through cache layer
//   - CatalogService using the interface
//
// Run (from the chapter folder):
//   go run ./exercises/01_product_repository

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type Product struct {
	ID          int64
	Name        string
	Description string
	Category    string
	PriceCent   int
	Stock       int
	Active      bool
	CreatedAt   time.Time
}

type SearchFilter struct {
	Category *string
	MinPrice *int
	MaxPrice *int
	Active   *bool
	Query    string // LIKE search on name/description
}

type StockAdjustment struct {
	ProductID int64
	Delta     int // positive=add, negative=remove
}

// ─────────────────────────────────────────────────────────────────────────────
// INTERFACE
// ─────────────────────────────────────────────────────────────────────────────

type ProductRepository interface {
	Create(ctx context.Context, p Product) (*Product, error)
	GetByID(ctx context.Context, id int64) (*Product, error)
	Search(ctx context.Context, f SearchFilter) ([]*Product, error)
	Update(ctx context.Context, p Product) error
	Delete(ctx context.Context, id int64) (bool, error)
	AdjustStock(ctx context.Context, adj StockAdjustment) (int, error) // returns new stock
	CountByCategory(ctx context.Context) (map[string]int, error)
}

var ErrNotFound = errors.New("not found")
var ErrInsufficientStock = errors.New("insufficient stock")

// ─────────────────────────────────────────────────────────────────────────────
// IN-MEMORY IMPLEMENTATION
// ─────────────────────────────────────────────────────────────────────────────

type memRepo struct {
	mu     sync.RWMutex
	store  map[int64]*Product
	nextID int64
}

func NewMemRepo() ProductRepository {
	return &memRepo{store: make(map[int64]*Product), nextID: 1}
}

func (r *memRepo) Create(ctx context.Context, p Product) (*Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.ID = r.nextID
	r.nextID++
	p.CreatedAt = time.Now()
	cp := p
	r.store[p.ID] = &cp
	return &cp, nil
}

func (r *memRepo) GetByID(ctx context.Context, id int64) (*Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.store[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, fmt.Errorf("product %d: %w", id, ErrNotFound)
}

func (r *memRepo) Search(ctx context.Context, f SearchFilter) ([]*Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*Product
	for _, p := range r.store {
		if f.Category != nil && p.Category != *f.Category {
			continue
		}
		if f.MinPrice != nil && p.PriceCent < *f.MinPrice {
			continue
		}
		if f.MaxPrice != nil && p.PriceCent > *f.MaxPrice {
			continue
		}
		if f.Active != nil && p.Active != *f.Active {
			continue
		}
		if f.Query != "" {
			q := strings.ToLower(f.Query)
			if !strings.Contains(strings.ToLower(p.Name), q) &&
				!strings.Contains(strings.ToLower(p.Description), q) {
				continue
			}
		}
		cp := *p
		out = append(out, &cp)
	}
	return out, nil
}

func (r *memRepo) Update(ctx context.Context, p Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.store[p.ID]; !ok {
		return fmt.Errorf("product %d: %w", p.ID, ErrNotFound)
	}
	cp := p
	r.store[p.ID] = &cp
	return nil
}

func (r *memRepo) Delete(ctx context.Context, id int64) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.store[id]; !ok {
		return false, nil
	}
	delete(r.store, id)
	return true, nil
}

func (r *memRepo) AdjustStock(ctx context.Context, adj StockAdjustment) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.store[adj.ProductID]
	if !ok {
		return 0, fmt.Errorf("product %d: %w", adj.ProductID, ErrNotFound)
	}
	newStock := p.Stock + adj.Delta
	if newStock < 0 {
		return p.Stock, fmt.Errorf("%w: current=%d requested=%d", ErrInsufficientStock, p.Stock, -adj.Delta)
	}
	p.Stock = newStock
	return p.Stock, nil
}

func (r *memRepo) CountByCategory(ctx context.Context) (map[string]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	counts := make(map[string]int)
	for _, p := range r.store {
		counts[p.Category]++
	}
	return counts, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SQL IMPLEMENTATION
// ─────────────────────────────────────────────────────────────────────────────

const ddl = `
CREATE TABLE IF NOT EXISTS products (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    category    TEXT    NOT NULL,
    price_cent  INTEGER NOT NULL,
    stock       INTEGER NOT NULL DEFAULT 0,
    active      INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

type sqlRepo struct {
	db *sql.DB
}

func NewSQLRepo(db *sql.DB) ProductRepository {
	return &sqlRepo{db: db}
}

func (r *sqlRepo) Create(ctx context.Context, p Product) (*Product, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO products (name, description, category, price_cent, stock, active) VALUES (?, ?, ?, ?, ?, ?)`,
		p.Name, p.Description, p.Category, p.PriceCent, p.Stock, boolInt(p.Active),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *sqlRepo) GetByID(ctx context.Context, id int64) (*Product, error) {
	return r.scanOne(r.db.QueryRowContext(ctx,
		`SELECT id, name, description, category, price_cent, stock, active, created_at FROM products WHERE id = ?`, id))
}

func (r *sqlRepo) Search(ctx context.Context, f SearchFilter) ([]*Product, error) {
	q := `SELECT id, name, description, category, price_cent, stock, active, created_at FROM products WHERE 1=1`
	var args []any
	if f.Category != nil {
		q += " AND category = ?"
		args = append(args, *f.Category)
	}
	if f.MinPrice != nil {
		q += " AND price_cent >= ?"
		args = append(args, *f.MinPrice)
	}
	if f.MaxPrice != nil {
		q += " AND price_cent <= ?"
		args = append(args, *f.MaxPrice)
	}
	if f.Active != nil {
		q += " AND active = ?"
		args = append(args, boolInt(*f.Active))
	}
	if f.Query != "" {
		q += " AND (name LIKE ? OR description LIKE ?)"
		pattern := "%" + f.Query + "%"
		args = append(args, pattern, pattern)
	}
	q += " ORDER BY id"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Product
	for rows.Next() {
		p, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *sqlRepo) Update(ctx context.Context, p Product) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE products SET name=?, description=?, category=?, price_cent=?, stock=?, active=? WHERE id=?`,
		p.Name, p.Description, p.Category, p.PriceCent, p.Stock, boolInt(p.Active), p.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("product %d: %w", p.ID, ErrNotFound)
	}
	return nil
}

func (r *sqlRepo) Delete(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM products WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r *sqlRepo) AdjustStock(ctx context.Context, adj StockAdjustment) (int, error) {
	// Use CHECK constraint or manual check.
	var current int
	err := r.db.QueryRowContext(ctx, `SELECT stock FROM products WHERE id = ?`, adj.ProductID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("product %d: %w", adj.ProductID, ErrNotFound)
	}
	newStock := current + adj.Delta
	if newStock < 0 {
		return current, fmt.Errorf("%w: current=%d requested=%d", ErrInsufficientStock, current, -adj.Delta)
	}
	_, err = r.db.ExecContext(ctx, `UPDATE products SET stock = ? WHERE id = ?`, newStock, adj.ProductID)
	return newStock, err
}

func (r *sqlRepo) CountByCategory(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT category, COUNT(*) FROM products GROUP BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var cat string
		var n int
		rows.Scan(&cat, &n)
		counts[cat] = n
	}
	return counts, rows.Err()
}

func (r *sqlRepo) scanOne(row *sql.Row) (*Product, error) {
	var p Product
	var active int
	var createdAt string
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Category, &p.PriceCent, &p.Stock, &active, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w", ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	p.Active = active == 1
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &p, nil
}

func (r *sqlRepo) scanRow(rows *sql.Rows) (*Product, error) {
	var p Product
	var active int
	var createdAt string
	if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Category, &p.PriceCent, &p.Stock, &active, &createdAt); err != nil {
		return nil, err
	}
	p.Active = active == 1
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &p, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ─────────────────────────────────────────────────────────────────────────────
// CACHING REPOSITORY — wraps any repo with in-memory read-through cache
// ─────────────────────────────────────────────────────────────────────────────

type cachingRepo struct {
	inner  ProductRepository
	mu     sync.RWMutex
	byID   map[int64]*Product
	hits   int
	misses int
}

func NewCachingRepo(inner ProductRepository) ProductRepository {
	return &cachingRepo{inner: inner, byID: make(map[int64]*Product)}
}

func (c *cachingRepo) Create(ctx context.Context, p Product) (*Product, error) {
	created, err := c.inner.Create(ctx, p)
	if err == nil {
		c.mu.Lock()
		cp := *created
		c.byID[created.ID] = &cp
		c.mu.Unlock()
	}
	return created, err
}

func (c *cachingRepo) GetByID(ctx context.Context, id int64) (*Product, error) {
	c.mu.RLock()
	if p, ok := c.byID[id]; ok {
		cp := *p
		c.hits++
		c.mu.RUnlock()
		return &cp, nil
	}
	c.misses++
	c.mu.RUnlock()

	p, err := c.inner.GetByID(ctx, id)
	if err == nil {
		c.mu.Lock()
		cp := *p
		c.byID[id] = &cp
		c.mu.Unlock()
	}
	return p, err
}

func (c *cachingRepo) Update(ctx context.Context, p Product) error {
	err := c.inner.Update(ctx, p)
	if err == nil {
		c.mu.Lock()
		cp := p
		c.byID[p.ID] = &cp
		c.mu.Unlock()
	}
	return err
}

func (c *cachingRepo) Delete(ctx context.Context, id int64) (bool, error) {
	deleted, err := c.inner.Delete(ctx, id)
	if err == nil && deleted {
		c.mu.Lock()
		delete(c.byID, id)
		c.mu.Unlock()
	}
	return deleted, err
}

// Search, AdjustStock, CountByCategory pass through — not cached for simplicity.
func (c *cachingRepo) Search(ctx context.Context, f SearchFilter) ([]*Product, error) {
	return c.inner.Search(ctx, f)
}
func (c *cachingRepo) AdjustStock(ctx context.Context, adj StockAdjustment) (int, error) {
	// Invalidate after stock change.
	stock, err := c.inner.AdjustStock(ctx, adj)
	if err == nil {
		c.mu.Lock()
		delete(c.byID, adj.ProductID)
		c.mu.Unlock()
	}
	return stock, err
}
func (c *cachingRepo) CountByCategory(ctx context.Context) (map[string]int, error) {
	return c.inner.CountByCategory(ctx)
}

func (c *cachingRepo) CacheStats() (hits, misses int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func runSuite(ctx context.Context, repo ProductRepository, label string) {
	fmt.Printf("\n=== %s ===\n", label)

	check := func(label string, err error) bool {
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", label, err)
			return false
		}
		fmt.Printf("  ✓ %s\n", label)
		return true
	}

	active := true
	p1, err := repo.Create(ctx, Product{Name: "Widget", Category: "hardware", PriceCent: 999, Stock: 100, Active: true})
	check("create Widget", err)
	p2, err := repo.Create(ctx, Product{Name: "Gadget", Category: "electronics", PriceCent: 4999, Stock: 30, Active: true})
	check("create Gadget", err)
	p3, err := repo.Create(ctx, Product{Name: "Sprocket", Category: "hardware", PriceCent: 199, Stock: 500, Active: true, Description: "metal sprocket"})
	check("create Sprocket", err)
	repo.Create(ctx, Product{Name: "Legacy", Category: "hardware", PriceCent: 50, Stock: 5, Active: false})

	// Search.
	cat := "hardware"
	results, _ := repo.Search(ctx, SearchFilter{Category: &cat, Active: &active})
	fmt.Printf("  search hardware active: %d\n", len(results))

	results2, _ := repo.Search(ctx, SearchFilter{Query: "metal"})
	fmt.Printf("  search 'metal': %d (found: %v)\n", len(results2), len(results2) > 0 && results2[0].Name == "Sprocket")

	// Stock adjust.
	newStock, err := repo.AdjustStock(ctx, StockAdjustment{ProductID: p1.ID, Delta: -30})
	check("adjust stock -30", err)
	fmt.Printf("  Widget new stock: %d\n", newStock)

	_, err = repo.AdjustStock(ctx, StockAdjustment{ProductID: p2.ID, Delta: -999})
	if errors.Is(err, ErrInsufficientStock) {
		fmt.Println("  ✓ insufficient stock rejected")
	} else {
		check("should fail insufficient", err)
	}

	// Count by category.
	counts, _ := repo.CountByCategory(ctx)
	fmt.Printf("  category counts: hardware=%d electronics=%d\n", counts["hardware"], counts["electronics"])

	// Update.
	p1.PriceCent = 1299
	check("update Widget price", repo.Update(ctx, *p1))

	// Delete.
	deleted, _ := repo.Delete(ctx, p3.ID)
	fmt.Printf("  deleted Sprocket: %v\n", deleted)

	_ = p2
}

func main() {
	ctx := context.Background()

	// In-memory.
	runSuite(ctx, NewMemRepo(), "In-Memory Repository")

	// SQL.
	db, _ := sql.Open("sqlite", "file::memory:?cache=shared")
	db.SetMaxOpenConns(1)
	defer db.Close()
	db.Exec(ddl)
	runSuite(ctx, NewSQLRepo(db), "SQL Repository")

	// Caching wrapper around in-memory.
	fmt.Println("\n=== Caching Repository ===")
	ctx2 := context.Background()
	inner := NewMemRepo()
	cached := NewCachingRepo(inner).(*cachingRepo)

	p, _ := cached.Create(ctx2, Product{Name: "Cached Product", Category: "test", PriceCent: 100, Stock: 10, Active: true})
	cached.GetByID(ctx2, p.ID) // hit (Create populated cache)
	cached.GetByID(ctx2, p.ID) // hit
	cached.GetByID(ctx2, p.ID) // hit

	// Force a cache miss by reading an ID that was never cached.
	inner.Create(ctx2, Product{Name: "Direct", Category: "test", PriceCent: 50, Stock: 1, Active: true})
	cached.GetByID(ctx2, p.ID+1) // miss → fetches from inner and caches

	hits, misses := cached.CacheStats()
	fmt.Printf("  cache stats: hits=%d misses=%d\n", hits, misses)
}
