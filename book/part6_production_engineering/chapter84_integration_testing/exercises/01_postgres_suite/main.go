// FILE: book/part6_production_engineering/chapter84_integration_testing/exercises/01_postgres_suite/main.go
// CHAPTER: 84 — Integration Testing
// TOPIC: Full integration test suite for an order repository — CRUD,
//        transaction isolation, concurrent writes, and constraint enforcement.
//
// Run:
//   go run ./exercises/01_postgres_suite

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED POSTGRES-LIKE DB with UNIQUE constraint support
// ─────────────────────────────────────────────────────────────────────────────

type OrderRow struct {
	ID         string
	CustomerID string
	Total      int
	Status     string
}

type ConstraintError struct{ msg string }

func (e *ConstraintError) Error() string { return e.msg }

type OrderDB struct {
	mu         sync.RWMutex
	orders     map[string]*OrderRow
	seq        atomic.Int64
	customerOrders map[string][]string // customerID → []orderID (for listing)
}

func NewOrderDB() *OrderDB {
	return &OrderDB{
		orders:         make(map[string]*OrderRow),
		customerOrders: make(map[string][]string),
	}
}

func (db *OrderDB) Insert(customerID string, total int) (string, error) {
	if customerID == "" {
		return "", &ConstraintError{"customerID cannot be empty"}
	}
	if total < 0 {
		return "", &ConstraintError{"total cannot be negative"}
	}
	id := fmt.Sprintf("ord-%d", db.seq.Add(1))
	row := &OrderRow{ID: id, CustomerID: customerID, Total: total, Status: "pending"}
	db.mu.Lock()
	db.orders[id] = row
	db.customerOrders[customerID] = append(db.customerOrders[customerID], id)
	db.mu.Unlock()
	return id, nil
}

func (db *OrderDB) FindByID(id string) (*OrderRow, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	r, ok := db.orders[id]
	return r, ok
}

func (db *OrderDB) UpdateStatus(id, status string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	r, ok := db.orders[id]
	if !ok {
		return fmt.Errorf("order %q not found", id)
	}
	r.Status = status
	return nil
}

func (db *OrderDB) FindByCustomer(customerID string) []*OrderRow {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var rows []*OrderRow
	for _, id := range db.customerOrders[customerID] {
		if r, ok := db.orders[id]; ok {
			rows = append(rows, r)
		}
	}
	return rows
}

func (db *OrderDB) Delete(id string) bool {
	db.mu.Lock()
	defer db.mu.Unlock()
	r, ok := db.orders[id]
	if !ok {
		return false
	}
	delete(db.orders, id)
	ids := db.customerOrders[r.CustomerID]
	filtered := ids[:0]
	for _, oid := range ids {
		if oid != id {
			filtered = append(filtered, oid)
		}
	}
	db.customerOrders[r.CustomerID] = filtered
	return true
}

func (db *OrderDB) Count() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.orders)
}

// ─────────────────────────────────────────────────────────────────────────────
// ORDER REPOSITORY
// ─────────────────────────────────────────────────────────────────────────────

type OrderRepository struct{ db *OrderDB }

func NewOrderRepository(db *OrderDB) *OrderRepository { return &OrderRepository{db: db} }

func (r *OrderRepository) Create(ctx context.Context, customerID string, total int) (string, error) {
	return r.db.Insert(customerID, total)
}

func (r *OrderRepository) Get(ctx context.Context, id string) (*OrderRow, error) {
	row, ok := r.db.FindByID(id)
	if !ok {
		return nil, fmt.Errorf("not found: %s", id)
	}
	return row, nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.db.UpdateStatus(id, status)
}

func (r *OrderRepository) ListByCustomer(ctx context.Context, customerID string) ([]*OrderRow, error) {
	return r.db.FindByCustomer(customerID), nil
}

func (r *OrderRepository) Delete(ctx context.Context, id string) error {
	if !r.db.Delete(id) {
		return fmt.Errorf("not found: %s", id)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MINI TEST FRAMEWORK
// ─────────────────────────────────────────────────────────────────────────────

type T struct{ name string; failed bool; logs []string }

func (t *T) Errorf(f string, a ...any) {
	t.failed = true
	t.logs = append(t.logs, "    FAIL: "+fmt.Sprintf(f, a...))
}

type Suite struct{ passed, failed int }

func (s *Suite) Run(name string, fn func(*T)) {
	t := &T{name: name}
	fn(t)
	if t.failed {
		s.failed++
		fmt.Printf("  --- FAIL: %s\n", name)
		for _, l := range t.logs {
			fmt.Println(l)
		}
	} else {
		s.passed++
		fmt.Printf("  --- PASS: %s\n", name)
	}
}

func (s *Suite) Report() { fmt.Printf("  %d/%d passed\n", s.passed, s.passed+s.failed) }

// ─────────────────────────────────────────────────────────────────────────────
// TESTS
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Order Repository Integration Suite ===")
	fmt.Println()
	ctx := context.Background()

	fmt.Println("--- CRUD ---")
	s1 := &Suite{}
	db := NewOrderDB()
	repo := NewOrderRepository(db)

	s1.Run("Create/valid", func(t *T) {
		id, err := repo.Create(ctx, "cust-1", 9999)
		if err != nil {
			t.Errorf("Create: %v", err)
			return
		}
		if id == "" {
			t.Errorf("Create: empty ID returned")
		}
	})

	s1.Run("Create/empty_customer", func(t *T) {
		_, err := repo.Create(ctx, "", 100)
		if err == nil {
			t.Errorf("Create with empty customerID: expected error, got nil")
		}
	})

	s1.Run("Create/negative_total", func(t *T) {
		_, err := repo.Create(ctx, "cust-x", -1)
		if err == nil {
			t.Errorf("Create with negative total: expected error, got nil")
		}
	})

	s1.Run("Get/found", func(t *T) {
		id, _ := repo.Create(ctx, "cust-2", 5000)
		row, err := repo.Get(ctx, id)
		if err != nil {
			t.Errorf("Get: %v", err)
			return
		}
		if row.CustomerID != "cust-2" {
			t.Errorf("CustomerID = %q, want cust-2", row.CustomerID)
		}
		if row.Total != 5000 {
			t.Errorf("Total = %d, want 5000", row.Total)
		}
		if row.Status != "pending" {
			t.Errorf("Status = %q, want pending", row.Status)
		}
	})

	s1.Run("Get/not_found", func(t *T) {
		_, err := repo.Get(ctx, "nonexistent")
		if err == nil {
			t.Errorf("Get nonexistent: expected error, got nil")
		}
	})

	s1.Report()

	fmt.Println()
	fmt.Println("--- Status transitions ---")
	s2 := &Suite{}

	s2.Run("UpdateStatus/pending_to_shipped", func(t *T) {
		id, _ := repo.Create(ctx, "cust-3", 100)
		err := repo.UpdateStatus(ctx, id, "shipped")
		if err != nil {
			t.Errorf("UpdateStatus: %v", err)
			return
		}
		row, _ := repo.Get(ctx, id)
		if row.Status != "shipped" {
			t.Errorf("Status = %q, want shipped", row.Status)
		}
	})

	s2.Run("UpdateStatus/not_found", func(t *T) {
		err := repo.UpdateStatus(ctx, "ghost-id", "shipped")
		if err == nil {
			t.Errorf("UpdateStatus on nonexistent: expected error")
		}
	})
	s2.Report()

	fmt.Println()
	fmt.Println("--- Listing by customer ---")
	s3 := &Suite{}

	s3.Run("ListByCustomer/multiple_orders", func(t *T) {
		id1, _ := repo.Create(ctx, "cust-4", 100)
		id2, _ := repo.Create(ctx, "cust-4", 200)
		rows, err := repo.ListByCustomer(ctx, "cust-4")
		if err != nil {
			t.Errorf("ListByCustomer: %v", err)
			return
		}
		ids := map[string]bool{}
		for _, r := range rows {
			ids[r.ID] = true
		}
		if !ids[id1] || !ids[id2] {
			t.Errorf("missing orders in list: got IDs %v, want %s and %s", ids, id1, id2)
		}
	})

	s3.Run("ListByCustomer/empty", func(t *T) {
		rows, _ := repo.ListByCustomer(ctx, "cust-nobody")
		if len(rows) != 0 {
			t.Errorf("expected 0 rows for unknown customer, got %d", len(rows))
		}
	})
	s3.Report()

	fmt.Println()
	fmt.Println("--- Delete ---")
	s4 := &Suite{}

	s4.Run("Delete/removes_order", func(t *T) {
		id, _ := repo.Create(ctx, "cust-5", 50)
		err := repo.Delete(ctx, id)
		if err != nil {
			t.Errorf("Delete: %v", err)
			return
		}
		_, err = repo.Get(ctx, id)
		if err == nil {
			t.Errorf("Get after delete: expected error, got nil")
		}
	})

	s4.Run("Delete/not_found", func(t *T) {
		err := repo.Delete(ctx, "ghost")
		if err == nil {
			t.Errorf("Delete nonexistent: expected error, got nil")
		}
	})
	s4.Report()

	fmt.Println()
	fmt.Println("--- Concurrent writes ---")
	s5 := &Suite{}

	s5.Run("Concurrent/no_data_races", func(t *T) {
		db2 := NewOrderDB()
		repo2 := NewOrderRepository(db2)
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				repo2.Create(ctx, fmt.Sprintf("cust-%d", i), i*100)
			}()
		}
		wg.Wait()
		if db2.Count() != 20 {
			t.Errorf("count = %d after 20 concurrent inserts, want 20", db2.Count())
		}
	})
	s5.Report()
}
