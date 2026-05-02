// FILE: book/part5_building_backends/chapter58_routing_options/exercises/01_route_table/main.go
// CHAPTER: 58 — Routing Options
// EXERCISE: Build a route table API:
//   - Use Go 1.22 net/http.ServeMux with method-qualified patterns
//   - Expose GET /routes that returns a JSON description of all registered routes
//   - Implement an API for two resources (Orders and Items) with full CRUD
//   - Route groups via a helper that shares a path prefix and auth middleware
//   - All routes log method + path + status + latency (structured log line)
//   - Method mismatches return 405 with correct Allow header
//
// Run (from the chapter folder):
//   go run ./exercises/01_route_table

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// LOGGING MIDDLEWARE
// ─────────────────────────────────────────────────────────────────────────────

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func loggingMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		fmt.Printf(`{"method":%q,"path":%q,"status":%d,"ms":%d}`+"\n",
			r.Method, r.URL.Path, rec.status, time.Since(start).Milliseconds())
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// AUTH MIDDLEWARE
// ─────────────────────────────────────────────────────────────────────────────

func authMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// ─────────────────────────────────────────────────────────────────────────────
// ROUTE TABLE — tracks registered routes for GET /routes
// ─────────────────────────────────────────────────────────────────────────────

type RouteEntry struct {
	Method      string `json:"method"`
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
	Auth        bool   `json:"auth_required"`
}

var routeTable []RouteEntry

func register(mux *http.ServeMux, method, pattern, desc string, auth bool, h http.HandlerFunc) {
	routeTable = append(routeTable, RouteEntry{
		Method: method, Pattern: pattern, Description: desc, Auth: auth,
	})
	var handler http.Handler = h
	if auth {
		handler = authMW(handler)
	}
	mux.Handle(method+" "+pattern, loggingMW(handler))
}

// ─────────────────────────────────────────────────────────────────────────────
// ORDERS STORE
// ─────────────────────────────────────────────────────────────────────────────

type Order struct {
	ID        int       `json:"id"`
	CustomerID int      `json:"customer_id"`
	Total     int       `json:"total_cents"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type orderStore struct {
	mu     sync.RWMutex
	items  map[int]*Order
	nextID atomic.Int64
}

var orders = func() *orderStore {
	s := &orderStore{items: make(map[int]*Order)}
	s.nextID.Store(1)
	for i, status := range []string{"pending", "shipped", "delivered"} {
		id := i + 1
		s.items[id] = &Order{ID: id, CustomerID: 1, Total: (i + 1) * 1000, Status: status, CreatedAt: time.Now().UTC()}
	}
	s.nextID.Store(4)
	return s
}()

func (s *orderStore) List() []*Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Order, 0, len(s.items))
	for _, o := range s.items {
		out = append(out, o)
	}
	return out
}

func (s *orderStore) Get(id int) (*Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.items[id]
	return o, ok
}

func (s *orderStore) Create(o *Order) *Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	o.ID = int(s.nextID.Load())
	o.CreatedAt = time.Now().UTC()
	if o.Status == "" {
		o.Status = "pending"
	}
	s.items[o.ID] = o
	s.nextID.Add(1)
	return o
}

func (s *orderStore) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return false
	}
	delete(s.items, id)
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// ITEMS STORE (line items within an order)
// ─────────────────────────────────────────────────────────────────────────────

type Item struct {
	ID       int    `json:"id"`
	OrderID  int    `json:"order_id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int    `json:"price_cents"`
}

type itemStore struct {
	mu     sync.RWMutex
	items  map[int]*Item
	nextID atomic.Int64
}

var items = func() *itemStore {
	s := &itemStore{items: make(map[int]*Item)}
	s.nextID.Store(1)
	seed := []struct{ orderID int; name string; qty, price int }{
		{1, "Widget A", 2, 500},
		{1, "Widget B", 1, 750},
		{2, "Gadget X", 3, 1200},
	}
	for i, d := range seed {
		s.items[i+1] = &Item{ID: i + 1, OrderID: d.orderID, Name: d.name, Quantity: d.qty, Price: d.price}
	}
	s.nextID.Store(4)
	return s
}()

func (s *itemStore) ListByOrder(orderID int) []*Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Item
	for _, it := range s.items {
		if it.OrderID == orderID {
			out = append(out, it)
		}
	}
	return out
}

func (s *itemStore) Create(it *Item) *Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	it.ID = int(s.nextID.Load())
	s.items[it.ID] = it
	s.nextID.Add(1)
	return it
}

// ─────────────────────────────────────────────────────────────────────────────
// ROUTE HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func handleListOrders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, orders.List())
}

func handleGetOrder(w http.ResponseWriter, r *http.Request) {
	var id int
	fmt.Sscanf(r.PathValue("id"), "%d", &id)
	o, ok := orders.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var o Order
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if o.CustomerID == 0 {
		writeError(w, http.StatusUnprocessableEntity, "customer_id is required")
		return
	}
	created := orders.Create(&o)
	w.Header().Set("Location", fmt.Sprintf("/v1/orders/%d", created.ID))
	writeJSON(w, http.StatusCreated, created)
}

func handleDeleteOrder(w http.ResponseWriter, r *http.Request) {
	var id int
	fmt.Sscanf(r.PathValue("id"), "%d", &id)
	if !orders.Delete(id) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleListItems(w http.ResponseWriter, r *http.Request) {
	var orderID int
	fmt.Sscanf(r.PathValue("orderID"), "%d", &orderID)
	if _, ok := orders.Get(orderID); !ok {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	writeJSON(w, http.StatusOK, items.ListByOrder(orderID))
}

func handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var orderID int
	fmt.Sscanf(r.PathValue("orderID"), "%d", &orderID)
	if _, ok := orders.Get(orderID); !ok {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	var it Item
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if strings.TrimSpace(it.Name) == "" {
		writeError(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	it.OrderID = orderID
	created := items.Create(&it)
	writeJSON(w, http.StatusCreated, created)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()

	// Route introspection endpoint (no auth, no logging wrapper — register manually).
	mux.HandleFunc("GET /routes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, routeTable)
	})

	// Orders — public read, auth required for write/delete.
	register(mux, "GET", "/v1/orders", "List all orders", false, handleListOrders)
	register(mux, "GET", "/v1/orders/{id}", "Get order by ID", false, handleGetOrder)
	register(mux, "POST", "/v1/orders", "Create a new order", true, handleCreateOrder)
	register(mux, "DELETE", "/v1/orders/{id}", "Delete an order", true, handleDeleteOrder)

	// Items (nested under orders).
	register(mux, "GET", "/v1/orders/{orderID}/items", "List items in an order", false, handleListItems)
	register(mux, "POST", "/v1/orders/{orderID}/items", "Add item to an order", true, handleCreateItem)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}

	do := func(method, path string, body string, headers map[string]string) (int, string) {
		var br *strings.Reader
		if body != "" {
			br = strings.NewReader(body)
		} else {
			br = strings.NewReader("")
		}
		req, _ := http.NewRequest(method, base+path, br)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0, err.Error()
		}
		defer resp.Body.Close()
		buf := make([]byte, 8192)
		n, _ := resp.Body.Read(buf)
		return resp.StatusCode, strings.TrimSpace(string(buf[:n]))
	}

	auth := map[string]string{"Authorization": "Bearer tok"}

	check := func(label string, code, want int) {
		mark := "✓"
		if code != want {
			mark = "✗"
		}
		fmt.Printf("  %s %-52s %d\n", mark, label, code)
	}

	fmt.Printf("=== Route Table API — %s ===\n\n", base)

	fmt.Println("--- Route introspection ---")
	code, body := do("GET", "/routes", "", nil)
	check("GET /routes → 200", code, 200)
	var routes []RouteEntry
	json.Unmarshal([]byte(body), &routes)
	for _, rt := range routes {
		fmt.Printf("    %-8s %-35s auth=%-5v %s\n", rt.Method, rt.Pattern, rt.Auth, rt.Description)
	}

	fmt.Println()
	fmt.Println("--- Orders CRUD ---")
	code, _ = do("GET", "/v1/orders", "", nil)
	check("GET /v1/orders", code, 200)

	code, _ = do("GET", "/v1/orders/1", "", nil)
	check("GET /v1/orders/1", code, 200)

	code, _ = do("POST", "/v1/orders", `{"customer_id":2,"total_cents":3000}`, nil)
	check("POST /v1/orders (no auth) → 401", code, 401)

	code, _ = do("POST", "/v1/orders", `{"customer_id":2,"total_cents":3000}`, auth)
	check("POST /v1/orders (with auth) → 201", code, 201)

	code, _ = do("DELETE", "/v1/orders/2", "", auth)
	check("DELETE /v1/orders/2 → 204", code, 204)

	code, _ = do("DELETE", "/v1/orders/2", "", auth)
	check("DELETE /v1/orders/2 again → 404", code, 404)

	fmt.Println()
	fmt.Println("--- Nested items ---")
	code, _ = do("GET", "/v1/orders/1/items", "", nil)
	check("GET /v1/orders/1/items → 200", code, 200)

	code, _ = do("POST", "/v1/orders/1/items", `{"name":"Bolt","quantity":10,"price_cents":50}`, auth)
	check("POST /v1/orders/1/items → 201", code, 201)

	code, _ = do("GET", "/v1/orders/99/items", "", nil)
	check("GET /v1/orders/99/items → 404", code, 404)

	fmt.Println()
	fmt.Println("--- Error cases ---")
	code, _ = do("POST", "/v1/orders", `{"total_cents":999}`, auth)
	check("POST order missing customer_id → 422", code, 422)

	code, _ = do("POST", "/v1/orders", `bad`, auth)
	check("POST order bad JSON → 400", code, 400)

	code, _ = do("PATCH", "/v1/orders/1", "", nil)
	check("PATCH /v1/orders/1 → 405", code, 405)
}
