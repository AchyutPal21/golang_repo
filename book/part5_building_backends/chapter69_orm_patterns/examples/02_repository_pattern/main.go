// FILE: book/part5_building_backends/chapter69_orm_patterns/examples/02_repository_pattern/main.go
// CHAPTER: 69 — ORM vs Builder vs Raw SQL
// TOPIC: Repository pattern in Go — interface-based abstraction over database
//        access, in-memory implementation for tests, SQL implementation for
//        production, and how to wire them together.
//
// Run (from the chapter folder):
//   go run ./examples/02_repository_pattern

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Product struct {
	ID        int64
	Name      string
	Category  string
	PriceCent int
	Stock     int
	CreatedAt time.Time
}

type ProductFilter struct {
	Category *string
	MinPrice *int
	MaxPrice *int
}

// ─────────────────────────────────────────────────────────────────────────────
// REPOSITORY INTERFACE — the abstraction boundary
// ─────────────────────────────────────────────────────────────────────────────

type ProductRepository interface {
	Create(ctx context.Context, p Product) (*Product, error)
	GetByID(ctx context.Context, id int64) (*Product, error)
	List(ctx context.Context, filter ProductFilter) ([]*Product, error)
	Update(ctx context.Context, p Product) error
	Delete(ctx context.Context, id int64) (bool, error)
}

// ─────────────────────────────────────────────────────────────────────────────
// IN-MEMORY IMPLEMENTATION — for tests and local dev
// ─────────────────────────────────────────────────────────────────────────────

type memProductRepo struct {
	mu      sync.RWMutex
	items   map[int64]*Product
	nextID  int64
}

func NewMemProductRepo() ProductRepository {
	return &memProductRepo{items: map[int64]*Product{}, nextID: 1}
}

func (r *memProductRepo) Create(ctx context.Context, p Product) (*Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.ID = r.nextID
	r.nextID++
	p.CreatedAt = time.Now()
	cp := p
	r.items[p.ID] = &cp
	return &cp, nil
}

func (r *memProductRepo) GetByID(ctx context.Context, id int64) (*Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.items[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, fmt.Errorf("product %d: not found", id)
}

func (r *memProductRepo) List(ctx context.Context, filter ProductFilter) ([]*Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*Product
	for _, p := range r.items {
		if filter.Category != nil && p.Category != *filter.Category {
			continue
		}
		if filter.MinPrice != nil && p.PriceCent < *filter.MinPrice {
			continue
		}
		if filter.MaxPrice != nil && p.PriceCent > *filter.MaxPrice {
			continue
		}
		cp := *p
		out = append(out, &cp)
	}
	return out, nil
}

func (r *memProductRepo) Update(ctx context.Context, p Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[p.ID]; !ok {
		return fmt.Errorf("product %d: not found", p.ID)
	}
	cp := p
	r.items[p.ID] = &cp
	return nil
}

func (r *memProductRepo) Delete(ctx context.Context, id int64) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return false, nil
	}
	delete(r.items, id)
	return true, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SQL IMPLEMENTATION — production
// ─────────────────────────────────────────────────────────────────────────────

const productSchema = `
CREATE TABLE IF NOT EXISTS products (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    category   TEXT    NOT NULL,
    price_cent INTEGER NOT NULL,
    stock      INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

type sqlProductRepo struct {
	db *sql.DB
}

func NewSQLProductRepo(db *sql.DB) ProductRepository {
	return &sqlProductRepo{db: db}
}

func (r *sqlProductRepo) Create(ctx context.Context, p Product) (*Product, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO products (name, category, price_cent, stock) VALUES (?, ?, ?, ?)`,
		p.Name, p.Category, p.PriceCent, p.Stock,
	)
	if err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *sqlProductRepo) GetByID(ctx context.Context, id int64) (*Product, error) {
	var p Product
	var createdAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, category, price_cent, stock, created_at FROM products WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.Category, &p.PriceCent, &p.Stock, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("product %d: not found", id)
	}
	if err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &p, nil
}

func (r *sqlProductRepo) List(ctx context.Context, filter ProductFilter) ([]*Product, error) {
	q := "SELECT id, name, category, price_cent, stock, created_at FROM products WHERE 1=1"
	var args []any
	if filter.Category != nil {
		q += " AND category = ?"
		args = append(args, *filter.Category)
	}
	if filter.MinPrice != nil {
		q += " AND price_cent >= ?"
		args = append(args, *filter.MinPrice)
	}
	if filter.MaxPrice != nil {
		q += " AND price_cent <= ?"
		args = append(args, *filter.MaxPrice)
	}
	q += " ORDER BY id"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Product
	for rows.Next() {
		var p Product
		var createdAt string
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.PriceCent, &p.Stock, &createdAt); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *sqlProductRepo) Update(ctx context.Context, p Product) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE products SET name = ?, category = ?, price_cent = ?, stock = ? WHERE id = ?`,
		p.Name, p.Category, p.PriceCent, p.Stock, p.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("product %d: not found", p.ID)
	}
	return nil
}

func (r *sqlProductRepo) Delete(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM products WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SERVICE LAYER — uses the interface, unaware of implementation
// ─────────────────────────────────────────────────────────────────────────────

type CatalogService struct {
	repo ProductRepository
}

func NewCatalogService(repo ProductRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

func (s *CatalogService) AddProduct(ctx context.Context, name, category string, priceCent, stock int) (*Product, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if priceCent <= 0 {
		return nil, fmt.Errorf("price must be positive")
	}
	return s.repo.Create(ctx, Product{Name: name, Category: category, PriceCent: priceCent, Stock: stock})
}

func (s *CatalogService) GetAffordable(ctx context.Context, maxCent int) ([]*Product, error) {
	return s.repo.List(ctx, ProductFilter{MaxPrice: &maxCent})
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO HELPER
// ─────────────────────────────────────────────────────────────────────────────

func runDemo(ctx context.Context, repo ProductRepository, label string) {
	fmt.Printf("\n=== %s ===\n", label)

	svc := NewCatalogService(repo)

	p1, _ := svc.AddProduct(ctx, "Widget", "hardware", 999, 100)
	p2, _ := svc.AddProduct(ctx, "Gadget", "electronics", 4999, 50)
	p3, _ := svc.AddProduct(ctx, "Doohickey", "hardware", 299, 500)
	fmt.Printf("  created: %s(%d) %s(%d) %s(%d)\n", p1.Name, p1.ID, p2.Name, p2.ID, p3.Name, p3.ID)

	// List by category.
	cat := "hardware"
	hardware, _ := repo.List(ctx, ProductFilter{Category: &cat})
	fmt.Printf("  hardware products: %d\n", len(hardware))

	// Affordable.
	affordable, _ := svc.GetAffordable(ctx, 1000)
	fmt.Printf("  under $10.00: %d products\n", len(affordable))
	for _, p := range affordable {
		fmt.Printf("    %-12s $%.2f\n", p.Name, float64(p.PriceCent)/100)
	}

	// Update.
	p1.PriceCent = 1299
	repo.Update(ctx, *p1)
	updated, _ := repo.GetByID(ctx, p1.ID)
	fmt.Printf("  updated %s price: $%.2f\n", updated.Name, float64(updated.PriceCent)/100)

	// Delete.
	deleted, _ := repo.Delete(ctx, p3.ID)
	fmt.Printf("  deleted Doohickey: %v\n", deleted)
	_, err := repo.GetByID(ctx, p3.ID)
	fmt.Printf("  get deleted: %v\n", err)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	ctx := context.Background()

	// ── IN-MEMORY REPO ────────────────────────────────────────────────────────
	memRepo := NewMemProductRepo()
	runDemo(ctx, memRepo, "In-Memory Repository (test/dev)")

	// ── SQL REPO ──────────────────────────────────────────────────────────────
	db, _ := sql.Open("sqlite", "file::memory:?cache=shared")
	db.SetMaxOpenConns(1)
	defer db.Close()
	db.Exec(productSchema)

	sqlRepo := NewSQLProductRepo(db)
	runDemo(ctx, sqlRepo, "SQL Repository (production)")

	fmt.Println()
	fmt.Println("=== Repository Pattern Benefits ===")
	fmt.Println("  1. Swap implementations without changing service code")
	fmt.Println("  2. Test service logic with fast in-memory repo — no DB required")
	fmt.Println("  3. Clean separation: business logic vs data access")
	fmt.Println("  4. Easy to add caching layer by wrapping the interface")
	fmt.Println("  5. Single place to change query logic when schema evolves")
}
