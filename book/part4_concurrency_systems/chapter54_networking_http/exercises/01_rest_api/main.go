// FILE: book/part4_concurrency_systems/chapter54_networking_http/exercises/01_rest_api/main.go
// CHAPTER: 54 — Networking II: HTTP/1.1
// EXERCISE: CRUD REST API for a Todo resource — GET, POST, PUT, DELETE —
//           with middleware (logging, request ID), JSON validation,
//           path parsing (no external router), and a full test harness.
//
// Run (from the chapter folder):
//   go run ./exercises/01_rest_api

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTodoRequest struct {
	Title string `json:"title"`
}

type UpdateTodoRequest struct {
	Title *string `json:"title,omitempty"`
	Done  *bool   `json:"done,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// STORE
// ─────────────────────────────────────────────────────────────────────────────

type Store struct {
	mu      sync.RWMutex
	items   map[int]*Todo
	nextID  int
}

func NewStore() *Store {
	return &Store{items: make(map[int]*Todo), nextID: 1}
}

func (s *Store) Create(title string) *Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := &Todo{ID: s.nextID, Title: title, CreatedAt: time.Now().UTC()}
	s.items[s.nextID] = t
	s.nextID++
	return t
}

func (s *Store) Get(id int) (*Todo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.items[id]
	return t, ok
}

func (s *Store) List() []*Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Todo, 0, len(s.items))
	for _, t := range s.items {
		out = append(out, t)
	}
	return out
}

func (s *Store) Update(id int, req UpdateTodoRequest) (*Todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[id]
	if !ok {
		return nil, false
	}
	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.Done != nil {
		t.Done = *req.Done
	}
	return t, true
}

func (s *Store) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return false
	}
	delete(s.items, id)
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLER
// ─────────────────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

type Handler struct{ store *Store }

// Route: /todos          GET (list), POST (create)
//        /todos/{id}     GET, PUT, DELETE
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(strings.TrimPrefix(path, "/todos"), "/")

	// /todos
	if len(parts) == 1 && parts[0] == "" {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, h.store.List())
		case http.MethodPost:
			h.create(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	// /todos/{id}
	if len(parts) == 2 && parts[0] == "" {
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.getOne(w, id)
		case http.MethodPut:
			h.update(w, r, id)
		case http.MethodDelete:
			h.delete(w, id)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	http.NotFound(w, r)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}
	t := h.store.Create(req.Title)
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) getOne(w http.ResponseWriter, id int) {
	t, ok := h.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id int) {
	var req UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	t, ok := h.store.Update(id, req)
	if !ok {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) delete(w http.ResponseWriter, id int) {
	if !h.store.Delete(id) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE
// ─────────────────────────────────────────────────────────────────────────────

type reqIDKey struct{}

var reqCounter atomic.Int64

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := reqCounter.Add(1)
		reqID := fmt.Sprintf("req-%05d", id)
		ctx := context.WithValue(r.Context(), reqIDKey{}, reqID)
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		reqID, _ := r.Context().Value(reqIDKey{}).(string)
		log.Printf("[%s] %s %s %d %s", reqID, r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Microsecond))
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// TEST HARNESS
// ─────────────────────────────────────────────────────────────────────────────

func do(client *http.Client, baseURL, method, path, body string, headers map[string]string) (int, string) {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req, _ := http.NewRequest(method, baseURL+path, bodyReader)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	var sb strings.Builder
	b := make([]byte, 4096)
	n, _ := resp.Body.Read(b)
	sb.Write(b[:n])
	return resp.StatusCode, strings.TrimSpace(sb.String())
}

func main() {
	store := NewStore()
	mux := http.NewServeMux()
	mux.Handle("/todos", &Handler{store: store})
	mux.Handle("/todos/", &Handler{store: store})

	srv := &http.Server{
		Handler:      requestIDMiddleware(loggingMiddleware(mux)),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	go srv.Serve(ln)
	base := "http://" + ln.Addr().String()

	fmt.Printf("=== REST API exercise on %s ===\n\n", base)

	ct := map[string]string{"Content-Type": "application/json"}
	client := &http.Client{Timeout: 5 * time.Second}

	check := func(label string, code int, body string, wantCode int) {
		mark := "✓"
		if code != wantCode {
			mark = "✗"
		}
		fmt.Printf("  %s %-35s %d  %s\n", mark, label, code, body)
	}

	// Create.
	code, body := do(client, base, "POST", "/todos", `{"title":"Buy groceries"}`, ct)
	check("POST /todos (create)", code, body, 201)

	code, body = do(client, base, "POST", "/todos", `{"title":"Write tests"}`, ct)
	check("POST /todos (create 2)", code, body, 201)

	code, body = do(client, base, "POST", "/todos", `{"title":""}`, ct)
	check("POST /todos (empty title)", code, body, 422)

	// List.
	code, body = do(client, base, "GET", "/todos", "", nil)
	check("GET /todos (list)", code, body, 200)

	// Get one.
	code, body = do(client, base, "GET", "/todos/1", "", nil)
	check("GET /todos/1", code, body, 200)

	code, body = do(client, base, "GET", "/todos/99", "", nil)
	check("GET /todos/99 (not found)", code, body, 404)

	// Update.
	code, body = do(client, base, "PUT", "/todos/1", `{"done":true}`, ct)
	check("PUT /todos/1 (mark done)", code, body, 200)

	code, body = do(client, base, "PUT", "/todos/99", `{"done":true}`, ct)
	check("PUT /todos/99 (not found)", code, body, 404)

	// Delete.
	code, body = do(client, base, "DELETE", "/todos/2", "", nil)
	check("DELETE /todos/2", code, body, 204)

	code, body = do(client, base, "DELETE", "/todos/2", "", nil)
	check("DELETE /todos/2 (already gone)", code, body, 404)

	// Method not allowed.
	code, body = do(client, base, "PATCH", "/todos", "", nil)
	check("PATCH /todos (405)", code, body, 405)

	fmt.Println()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
