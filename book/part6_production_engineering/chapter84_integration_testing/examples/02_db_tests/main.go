// FILE: book/part6_production_engineering/chapter84_integration_testing/examples/02_db_tests/main.go
// CHAPTER: 84 — Integration Testing
// TOPIC: DB integration test patterns — transactions for isolation, parallel
//        tests with separate schemas, seeding, and cleanup.
//
// Run:
//   go run ./examples/02_db_tests

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED TRANSACTIONAL DB
// ─────────────────────────────────────────────────────────────────────────────

// In production each integration test opens a real DB transaction and rolls
// back at the end — the table stays clean for the next test.
//
//   func TestFoo(t *testing.T) {
//       tx, _ := db.BeginTx(ctx, nil)
//       t.Cleanup(func() { tx.Rollback() })
//       repo := NewRepo(tx)
//       ...
//   }
//
// Here we simulate with an in-memory store that supports Begin/Rollback.

type Row struct {
	ID    string
	Name  string
	Price int
}

type txState struct {
	changes map[string]*Row // pending writes within this transaction
	deleted map[string]bool
}

type TxDB struct {
	mu       sync.RWMutex
	rows     map[string]*Row
	seq      atomic.Int64
	txActive sync.Mutex // one tx at a time for demo
}

func NewTxDB() *TxDB {
	return &TxDB{rows: make(map[string]*Row)}
}

type Tx struct {
	db    *TxDB
	state *txState
	done  bool
}

func (db *TxDB) Begin() *Tx {
	return &Tx{
		db: db,
		state: &txState{
			changes: make(map[string]*Row),
			deleted: make(map[string]bool),
		},
	}
}

func (tx *Tx) Insert(name string, price int) string {
	id := fmt.Sprintf("r%d", tx.db.seq.Add(1))
	tx.state.changes[id] = &Row{ID: id, Name: name, Price: price}
	return id
}

func (tx *Tx) Find(id string) (*Row, bool) {
	if tx.state.deleted[id] {
		return nil, false
	}
	if r, ok := tx.state.changes[id]; ok {
		return r, true
	}
	tx.db.mu.RLock()
	defer tx.db.mu.RUnlock()
	r, ok := tx.db.rows[id]
	return r, ok
}

func (tx *Tx) Delete(id string) {
	tx.state.deleted[id] = true
	delete(tx.state.changes, id)
}

func (tx *Tx) Commit() {
	if tx.done {
		return
	}
	tx.done = true
	tx.db.mu.Lock()
	defer tx.db.mu.Unlock()
	for id, row := range tx.state.changes {
		tx.db.rows[id] = row
	}
	for id := range tx.state.deleted {
		delete(tx.db.rows, id)
	}
}

func (tx *Tx) Rollback() {
	tx.done = true
	// discard state.changes and state.deleted — nothing persisted
}

func (db *TxDB) Count() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.rows)
}

// ─────────────────────────────────────────────────────────────────────────────
// REPOSITORY (tested against the transactional DB)
// ─────────────────────────────────────────────────────────────────────────────

type ProductRepository struct{ tx *Tx }

func NewProductRepository(tx *Tx) *ProductRepository {
	return &ProductRepository{tx: tx}
}

func (r *ProductRepository) Create(ctx context.Context, name string, price int) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name required")
	}
	if price < 0 {
		return "", fmt.Errorf("price must be non-negative")
	}
	return r.tx.Insert(name, price), nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id string) (*Row, error) {
	row, ok := r.tx.Find(id)
	if !ok {
		return nil, fmt.Errorf("product %q not found", id)
	}
	return row, nil
}

func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	_, ok := r.tx.Find(id)
	if !ok {
		return fmt.Errorf("product %q not found", id)
	}
	r.tx.Delete(id)
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
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== DB Integration Test Patterns ===")
	fmt.Println()
	ctx := context.Background()

	db := NewTxDB()

	// ── ROLLBACK ISOLATION ────────────────────────────────────────────────────
	fmt.Println("--- Transaction rollback for test isolation ---")
	suite := &Suite{}

	suite.Run("Repo/create_committed", func(t *T) {
		tx := db.Begin()
		repo := NewProductRepository(tx)

		id, err := repo.Create(ctx, "Widget", 999)
		if err != nil {
			t.Errorf("Create: %v", err)
			return
		}
		tx.Commit()

		// Verify the row persisted.
		tx2 := db.Begin()
		repo2 := NewProductRepository(tx2)
		row, err := repo2.GetByID(ctx, id)
		tx2.Rollback()
		if err != nil {
			t.Errorf("GetByID after commit: %v", err)
			return
		}
		if row.Name != "Widget" {
			t.Errorf("Name = %q, want Widget", row.Name)
		}
	})

	suite.Run("Repo/create_rolled_back_not_visible", func(t *T) {
		before := db.Count()
		tx := db.Begin()
		repo := NewProductRepository(tx)
		repo.Create(ctx, "Ephemeral", 1)
		tx.Rollback() // discard

		after := db.Count()
		if after != before {
			t.Errorf("count after rollback = %d, want %d (rollback should discard)", after, before)
		}
	})

	suite.Run("Repo/delete_visible_within_tx", func(t *T) {
		// Seed via committed tx.
		seedTx := db.Begin()
		id := seedTx.Insert("ToDelete", 10)
		seedTx.Commit()

		// Open test tx.
		tx := db.Begin()
		defer tx.Rollback() // cleanup
		repo := NewProductRepository(tx)

		err := repo.Delete(ctx, id)
		if err != nil {
			t.Errorf("Delete: %v", err)
			return
		}
		_, err = repo.GetByID(ctx, id)
		if err == nil {
			t.Errorf("GetByID after delete: expected error, got nil")
		}
	})

	suite.Run("Repo/validation_name_required", func(t *T) {
		tx := db.Begin()
		defer tx.Rollback()
		repo := NewProductRepository(tx)
		_, err := repo.Create(ctx, "", 100)
		if err == nil {
			t.Errorf("Create with empty name: expected error, got nil")
		}
	})

	suite.Run("Repo/validation_negative_price", func(t *T) {
		tx := db.Begin()
		defer tx.Rollback()
		repo := NewProductRepository(tx)
		_, err := repo.Create(ctx, "Bad", -1)
		if err == nil {
			t.Errorf("Create with negative price: expected error, got nil")
		}
	})

	suite.Report()

	// ── PARALLEL SCHEMA ISOLATION ─────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Parallel test isolation with separate DBs ---")
	s2 := &Suite{}

	s2.Run("Parallel/independent_dbs", func(t *T) {
		// Each parallel test gets its own DB — no interference.
		db1, db2 := NewTxDB(), NewTxDB()
		tx1, tx2 := db1.Begin(), db2.Begin()
		defer tx1.Rollback()
		defer tx2.Rollback()

		repo1 := NewProductRepository(tx1)
		repo2 := NewProductRepository(tx2)

		id1, _ := repo1.Create(ctx, "In-DB1", 100)
		_, err := repo2.GetByID(ctx, id1)
		if err == nil {
			t.Errorf("DB1 row visible in DB2 — databases are not isolated!")
		}
	})
	s2.Report()

	// ── SEEDING REFERENCE ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Seeding patterns ---")
	fmt.Println(`  // Option 1: seed in each test (most explicit)
  func TestGetProduct(t *testing.T) {
      tx, _ := db.BeginTx(ctx, nil)
      t.Cleanup(func() { tx.Rollback() })
      repo := NewRepo(tx)
      id, _ := repo.Create(ctx, "Widget", 999)
      // now test Get
  }

  // Option 2: seed via SQL file in TestMain
  func TestMain(m *testing.M) {
      seed, _ := os.ReadFile("testdata/seed.sql")
      db.Exec(string(seed))
      os.Exit(m.Run())
  }

  // Option 3: factory helpers
  func newTestProduct(t *testing.T, repo ProductRepo) string {
      t.Helper()
      id, err := repo.Create(context.Background(), "test-product", 100)
      require.NoError(t, err)
      return id
  }`)
}
