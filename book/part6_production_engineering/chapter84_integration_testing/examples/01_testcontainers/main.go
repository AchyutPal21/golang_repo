// FILE: book/part6_production_engineering/chapter84_integration_testing/examples/01_testcontainers/main.go
// CHAPTER: 84 — Integration Testing
// TOPIC: Integration testing patterns — in-process fakes, test databases,
//        HTTP handler tests, and testcontainers concepts.
//        (Pure Go simulation — no Docker required.)
//
// Run:
//   go run ./examples/01_testcontainers

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// IN-PROCESS "DATABASE" — simulates testcontainers-go pattern
// ─────────────────────────────────────────────────────────────────────────────

// In production you'd use testcontainers-go:
//   container, _ := postgres.Run(ctx, "postgres:16-alpine", ...)
//   connStr, _ := container.ConnectionString(ctx)
//   defer container.Terminate(ctx)
//
// Here we use an in-memory store that behaves like a real DB.

type DB struct {
	mu      sync.RWMutex
	records map[string]map[string]any // table → id → row
	seq     int
}

func NewTestDB() *DB {
	return &DB{records: make(map[string]map[string]any)}
}

func (db *DB) Insert(table string, row map[string]any) string {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.seq++
	id := fmt.Sprintf("%d", db.seq)
	row["id"] = id
	if db.records[table] == nil {
		db.records[table] = make(map[string]any)
	}
	db.records[table][id] = row
	return id
}

func (db *DB) FindByID(table, id string) (map[string]any, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.records[table] == nil {
		return nil, false
	}
	row, ok := db.records[table][id]
	if !ok {
		return nil, false
	}
	return row.(map[string]any), true
}

func (db *DB) All(table string) []map[string]any {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var rows []map[string]any
	for _, v := range db.records[table] {
		rows = append(rows, v.(map[string]any))
	}
	return rows
}

func (db *DB) Delete(table, id string) bool {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.records[table] == nil {
		return false
	}
	_, ok := db.records[table][id]
	delete(db.records[table], id)
	return ok
}

func (db *DB) Reset() {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.records = make(map[string]map[string]any)
	db.seq = 0
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP HANDLER (system under test)
// ─────────────────────────────────────────────────────────────────────────────

type ProductHandler struct {
	db *DB
}

func (h *ProductHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/products":
		h.create(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/products/"):
		h.get(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/products/"):
		h.delete(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/products":
		h.list(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *ProductHandler) create(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	price := r.FormValue("price")
	if name == "" || price == "" {
		http.Error(w, `{"error":"name and price required"}`, http.StatusBadRequest)
		return
	}
	id := h.db.Insert("products", map[string]any{"name": name, "price": price})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id":%q,"name":%q,"price":%q}`, id, name, price)
}

func (h *ProductHandler) get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/products/")
	row, ok := h.db.FindByID("products", id)
	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id":%q,"name":%q,"price":%q}`, row["id"], row["name"], row["price"])
}

func (h *ProductHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/products/")
	if !h.db.Delete("products", id) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) list(w http.ResponseWriter, r *http.Request) {
	rows := h.db.All("products")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"count":%d}`, len(rows))
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
// INTEGRATION TESTS via httptest.NewRecorder
// ─────────────────────────────────────────────────────────────────────────────

func doRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func main() {
	fmt.Println("=== Integration Testing Patterns ===")
	fmt.Println()

	db := NewTestDB()
	handler := &ProductHandler{db: db}

	// ── FULL CRUD FLOW ────────────────────────────────────────────────────────
	fmt.Println("--- HTTP handler integration test (full CRUD) ---")
	suite := &Suite{}

	suite.Run("POST /products creates product", func(t *T) {
		db.Reset()
		rr := doRequest(handler, "POST", "/products", "name=Widget&price=999")
		if rr.Code != http.StatusCreated {
			t.Errorf("status = %d, want 201", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "Widget") {
			t.Errorf("body missing name: %q", rr.Body.String())
		}
	})

	suite.Run("GET /products/:id returns product", func(t *T) {
		db.Reset()
		id := db.Insert("products", map[string]any{"name": "Gadget", "price": "499"})
		rr := doRequest(handler, "GET", "/products/"+id, "")
		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "Gadget") {
			t.Errorf("body = %q, missing Gadget", rr.Body.String())
		}
	})

	suite.Run("GET /products/:id returns 404 for unknown", func(t *T) {
		db.Reset()
		rr := doRequest(handler, "GET", "/products/999", "")
		if rr.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rr.Code)
		}
	})

	suite.Run("DELETE /products/:id removes product", func(t *T) {
		db.Reset()
		id := db.Insert("products", map[string]any{"name": "ToDelete", "price": "1"})
		rr := doRequest(handler, "DELETE", "/products/"+id, "")
		if rr.Code != http.StatusNoContent {
			t.Errorf("delete status = %d, want 204", rr.Code)
		}
		// Verify gone.
		rr2 := doRequest(handler, "GET", "/products/"+id, "")
		if rr2.Code != http.StatusNotFound {
			t.Errorf("after delete: status = %d, want 404", rr2.Code)
		}
	})

	suite.Run("POST /products missing fields returns 400", func(t *T) {
		db.Reset()
		rr := doRequest(handler, "POST", "/products", "name=OnlyName")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})

	suite.Run("GET /products lists all", func(t *T) {
		db.Reset()
		db.Insert("products", map[string]any{"name": "A", "price": "1"})
		db.Insert("products", map[string]any{"name": "B", "price": "2"})
		rr := doRequest(handler, "GET", "/products", "")
		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "2") {
			t.Errorf("count missing in body: %q", rr.Body.String())
		}
	})

	suite.Report()

	// ── TESTCONTAINERS REFERENCE ──────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- testcontainers-go reference (production pattern) ---")
	fmt.Println(`  import "github.com/testcontainers/testcontainers-go/modules/postgres"

  func TestWithRealDB(t *testing.T) {
      ctx := context.Background()

      // Start a real Postgres container.
      container, err := postgres.Run(ctx, "postgres:16-alpine",
          postgres.WithDatabase("testdb"),
          postgres.WithUsername("test"),
          postgres.WithPassword("test"),
          testcontainers.WithWaitStrategy(
              wait.ForLog("database system is ready to accept connections"),
          ),
      )
      require.NoError(t, err)
      t.Cleanup(func() { container.Terminate(ctx) })

      connStr, _ := container.ConnectionString(ctx, "sslmode=disable")
      db, _ := sql.Open("pgx", connStr)

      // Run migrations.
      runMigrations(db)

      // Test against real Postgres.
      repo := NewProductRepository(db)
      id, err := repo.Create(ctx, "Widget", 999)
      require.NoError(t, err)
      require.NotEmpty(t, id)
  }

  // Tip: use TestMain to start one container per package:
  func TestMain(m *testing.M) {
      // start container
      code := m.Run()
      // stop container
      os.Exit(code)
  }`)

	// ── HTTPTEST.SERVER ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- httptest.NewServer (full HTTP stack) ---")
	_ = context.Background()

	db2 := NewTestDB()
	srv := httptest.NewServer(&ProductHandler{db: db2})
	defer srv.Close()

	s2 := &Suite{}
	s2.Run("httptest.NewServer/create_and_fetch", func(t *T) {
		// POST
		resp, err := http.PostForm(srv.URL+"/products",
			map[string][]string{"name": {"Sprocket"}, "price": {"199"}})
		if err != nil {
			t.Errorf("POST: %v", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("POST status = %d, want 201", resp.StatusCode)
		}
	})
	s2.Report()

	fmt.Println()
	fmt.Println("--- Integration test best practices ---")
	fmt.Println(`  1. Use httptest.NewRecorder for handler unit tests (fast, no network).
  2. Use httptest.NewServer when you need the full HTTP stack (middleware, routing).
  3. Use testcontainers-go for tests that must hit real Postgres/Redis/Kafka.
  4. Use TestMain to start one container per package — don't start/stop per test.
  5. Always call t.Cleanup (or defer container.Terminate) to avoid container leaks.
  6. Seed test data in the test, not a shared fixture file — tests must be independent.`)
}
