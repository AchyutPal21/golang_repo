// FILE: 10_advanced_patterns/07_testing_advanced.go
// TOPIC: Advanced Testing — table tests, subtests, mocks, httptest, benchmarks
//
// Run: go run 10_advanced_patterns/07_testing_advanced.go

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

// ── CODE UNDER TEST ───────────────────────────────────────────────────────────

type UserStore interface {
	FindByID(id int) (string, error)
}

type APIHandler struct {
	store UserStore
}

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"user":"alice","status":"ok"}`)
}

// ── MOCK IMPLEMENTATION ───────────────────────────────────────────────────────

type MockUserStore struct {
	users  map[int]string
	errors map[int]error
	calls  []int  // track which IDs were looked up
}

func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		users:  make(map[int]string),
		errors: make(map[int]error),
	}
}

func (m *MockUserStore) WillReturn(id int, user string)  { m.users[id] = user }
func (m *MockUserStore) WillError(id int, err error)     { m.errors[id] = err }

func (m *MockUserStore) FindByID(id int) (string, error) {
	m.calls = append(m.calls, id)
	if err, ok := m.errors[id]; ok { return "", err }
	if u, ok := m.users[id]; ok { return u, nil }
	return "", fmt.Errorf("user %d not found", id)
}

func (m *MockUserStore) WasCalled(id int) bool {
	for _, c := range m.calls { if c == id { return true } }
	return false
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Advanced Testing Patterns")
	fmt.Println("════════════════════════════════════════")

	// ── TABLE-DRIVEN TEST PATTERN ─────────────────────────────────────────
	fmt.Println("\n── Table-driven tests (pattern) ──")
	fmt.Println(`
  func TestAdd(t *testing.T) {
      cases := []struct {
          name     string
          a, b     int
          expected int
      }{
          {"positive", 2, 3, 5},
          {"negative", -1, -2, -3},
          {"zero", 0, 0, 0},
          {"mixed", 10, -3, 7},
      }
      for _, tc := range cases {
          t.Run(tc.name, func(t *testing.T) {
              got := Add(tc.a, tc.b)
              if got != tc.expected {
                  t.Errorf("Add(%d,%d) = %d, want %d", tc.a, tc.b, got, tc.expected)
              }
          })
      }
  }

  Run specific: go test -run TestAdd/positive`)

	// ── MOCK USAGE ────────────────────────────────────────────────────────
	fmt.Println("\n── Using mock in tests ──")
	mock := NewMockUserStore()
	mock.WillReturn(1, "alice")
	mock.WillReturn(2, "bob")
	mock.WillError(99, fmt.Errorf("not found"))

	// Simulate what a test would do:
	user, err := mock.FindByID(1)
	fmt.Printf("  FindByID(1): %q, err=%v\n", user, err)
	_, err2 := mock.FindByID(99)
	fmt.Printf("  FindByID(99): err=%v\n", err2)
	fmt.Printf("  WasCalled(1): %v\n", mock.WasCalled(1))
	fmt.Printf("  WasCalled(5): %v\n", mock.WasCalled(5))

	// ── HTTPTEST ──────────────────────────────────────────────────────────
	fmt.Println("\n── httptest.NewRecorder ──")
	handler := &APIHandler{store: mock}

	// Simulate GET request:
	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	fmt.Printf("  Status: %d\n", rec.Code)
	fmt.Printf("  Body:   %q\n", rec.Body.String())
	fmt.Printf("  Content-Type: %q\n", rec.Header().Get("Content-Type"))

	// Simulate wrong method:
	req2 := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader("{}"))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	fmt.Printf("  POST → Status: %d\n", rec2.Code)

	// httptest.NewServer — full server for integration tests:
	fmt.Println("\n── httptest.NewServer ──")
	ts := httptest.NewServer(handler)
	defer ts.Close()
	resp, _ := http.Get(ts.URL + "/user")
	resp.Body.Close()
	fmt.Printf("  Real HTTP request to test server: status=%d\n", resp.StatusCode)

	// ── TESTING PATTERNS ──────────────────────────────────────────────────
	fmt.Println("\n── Key patterns ──")
	fmt.Println(`
  1. t.Helper() — mark helper funcs so errors point to call site
     func assertEqual(t *testing.T, got, want string) {
         t.Helper()
         if got != want { t.Errorf("got %q, want %q", got, want) }
     }

  2. t.Cleanup() — run cleanup after test (like defer but scoped to test)
     t.Cleanup(func() { db.Close() })

  3. t.Parallel() — run test in parallel with other parallel tests
     func TestSomething(t *testing.T) {
         t.Parallel()
         ...
     }

  4. t.Skip() — conditionally skip
     if runtime.GOOS == "windows" { t.Skip("not supported on Windows") }

  5. Benchmark pattern:
     func BenchmarkSort(b *testing.B) {
         data := generateData(10000)
         b.ResetTimer()           // don't count setup
         for i := 0; i < b.N; i++ {
             sort.Ints(data)
             data = generateData(10000) // reset between runs
         }
     }
     go test -bench=. -benchmem -count=3

  6. Golden files — store expected output in .golden files:
     // Update goldens: go test ./... -update
     // Compare: diff(got, golden file)
`)

	fmt.Println("─── SUMMARY ────────────────────────────────")
	fmt.Println("  Table tests: name+inputs+expected in []struct, t.Run per case")
	fmt.Println("  Mock via interfaces: swap implementation without changing code")
	fmt.Println("  httptest.NewRecorder: test HTTP handlers without a server")
	fmt.Println("  httptest.NewServer: full server for integration tests")
	fmt.Println("  t.Helper(): make error lines point to test case, not helper")
	fmt.Println("  go test -race ./...  — always run with race detector")
}
