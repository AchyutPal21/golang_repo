// FILE: book/part5_building_backends/chapter57_rest_api_design/examples/01_rest_principles/main.go
// CHAPTER: 57 — REST API Design
// TOPIC: REST constraints in practice — resource naming, HTTP method semantics,
//        status code selection, idempotency, HATEOAS links, content negotiation.
//
// Run (from the chapter folder):
//   go run ./examples/01_rest_principles

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type Article struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	AuthorID  int       `json:"author_id"`
	Published bool      `json:"published"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// HATEOAS ENVELOPE
//
// REST Level 3: responses include links to related resources and transitions.
// ─────────────────────────────────────────────────────────────────────────────

type Link struct {
	Href   string `json:"href"`
	Method string `json:"method"`
	Rel    string `json:"rel"`
}

type ArticleResponse struct {
	*Article
	Links []Link `json:"_links"`
}

func articleLinks(baseURL string, a *Article) []Link {
	links := []Link{
		{Href: fmt.Sprintf("%s/articles/%d", baseURL, a.ID), Method: "GET", Rel: "self"},
		{Href: fmt.Sprintf("%s/articles/%d", baseURL, a.ID), Method: "PUT", Rel: "update"},
		{Href: fmt.Sprintf("%s/articles/%d", baseURL, a.ID), Method: "DELETE", Rel: "delete"},
		{Href: fmt.Sprintf("%s/authors/%d", baseURL, a.AuthorID), Method: "GET", Rel: "author"},
	}
	if !a.Published {
		links = append(links, Link{
			Href:   fmt.Sprintf("%s/articles/%d/publish", baseURL, a.ID),
			Method: "POST",
			Rel:    "publish",
		})
	}
	return links
}

// ─────────────────────────────────────────────────────────────────────────────
// STORE
// ─────────────────────────────────────────────────────────────────────────────

type Store struct {
	mu     sync.RWMutex
	items  map[int]*Article
	nextID int
}

func NewStore() *Store {
	s := &Store{items: make(map[int]*Article), nextID: 1}
	// Seed data.
	for i, title := range []string{"Go Concurrency", "REST in Go", "TLS Explained"} {
		s.items[i+1] = &Article{
			ID: i + 1, Title: title, Body: "...", AuthorID: 1,
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
			UpdatedAt: time.Now(),
		}
	}
	s.nextID = 4
	return s
}

func (s *Store) List() []*Article {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Article, 0, len(s.items))
	for _, a := range s.items {
		out = append(out, a)
	}
	return out
}

func (s *Store) Get(id int) (*Article, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.items[id]
	return a, ok
}

func (s *Store) Create(a *Article) *Article {
	s.mu.Lock()
	defer s.mu.Unlock()
	a.ID = s.nextID
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	s.items[s.nextID] = a
	s.nextID++
	return a
}

func (s *Store) Update(id int, fn func(*Article)) (*Article, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.items[id]
	if !ok {
		return nil, false
	}
	fn(a)
	a.UpdatedAt = time.Now().UTC()
	return a, true
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

type Handler struct {
	store   *Store
	baseURL string
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/articles")
	path = strings.TrimSuffix(path, "/")

	// /articles
	if path == "" {
		switch r.Method {
		case http.MethodGet:
			h.list(w, r)
		case http.MethodPost:
			h.create(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	// /articles/{id}/publish
	if strings.HasSuffix(path, "/publish") {
		idStr := strings.TrimSuffix(strings.TrimPrefix(path, "/"), "/publish")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.publish(w, id)
		return
	}

	// /articles/{id}
	id, err := strconv.Atoi(strings.TrimPrefix(path, "/"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.get(w, id)
	case http.MethodPut:
		h.update(w, r, id)
	case http.MethodDelete:
		h.delete(w, id)
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	articles := h.store.List()
	resp := make([]ArticleResponse, len(articles))
	for i, a := range articles {
		resp[i] = ArticleResponse{Article: a, Links: articleLinks(h.baseURL, a)}
	}
	// 200 OK — resource collection returned
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) get(w http.ResponseWriter, id int) {
	a, ok := h.store.Get(id)
	if !ok {
		// 404 Not Found — resource does not exist
		writeError(w, http.StatusNotFound, "article not found")
		return
	}
	writeJSON(w, http.StatusOK, ArticleResponse{Article: a, Links: articleLinks(h.baseURL, a)})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var a Article
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if strings.TrimSpace(a.Title) == "" {
		// 422 Unprocessable Entity — request is syntactically valid but semantically wrong
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}
	created := h.store.Create(&a)
	// 201 Created — resource was created; Location header points to the new resource
	w.Header().Set("Location", fmt.Sprintf("%s/articles/%d", h.baseURL, created.ID))
	writeJSON(w, http.StatusCreated, ArticleResponse{Article: created, Links: articleLinks(h.baseURL, created)})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id int) {
	var patch Article
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	updated, ok := h.store.Update(id, func(a *Article) {
		if patch.Title != "" {
			a.Title = patch.Title
		}
		if patch.Body != "" {
			a.Body = patch.Body
		}
	})
	if !ok {
		writeError(w, http.StatusNotFound, "article not found")
		return
	}
	// 200 OK — full updated resource returned
	writeJSON(w, http.StatusOK, ArticleResponse{Article: updated, Links: articleLinks(h.baseURL, updated)})
}

func (h *Handler) delete(w http.ResponseWriter, id int) {
	if !h.store.Delete(id) {
		writeError(w, http.StatusNotFound, "article not found")
		return
	}
	// 204 No Content — success with no response body (idempotent: deleting already-gone → 404)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) publish(w http.ResponseWriter, id int) {
	updated, ok := h.store.Update(id, func(a *Article) {
		a.Published = true
	})
	if !ok {
		writeError(w, http.StatusNotFound, "article not found")
		return
	}
	// 200 OK — action (state transition) applied; full resource returned
	writeJSON(w, http.StatusOK, ArticleResponse{Article: updated, Links: articleLinks(h.baseURL, updated)})
}

// ─────────────────────────────────────────────────────────────────────────────
// TEST HARNESS
// ─────────────────────────────────────────────────────────────────────────────

func run(client *http.Client, base, method, path, body string) (int, string) {
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
	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	return resp.StatusCode, strings.TrimSpace(string(buf[:n]))
}

func main() {
	store := NewStore()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()

	h := &Handler{store: store, baseURL: base}
	mux := http.NewServeMux()
	mux.Handle("/articles", h)
	mux.Handle("/articles/", h)

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}
	ct := "application/json"
	_ = ct

	fmt.Printf("=== REST API Design — %s ===\n\n", base)

	check := func(label string, code int, wantCode int) {
		mark := "✓"
		if code != wantCode {
			mark = "✗"
		}
		fmt.Printf("  %s %-40s %d\n", mark, label, code)
	}

	// HTTP method semantics.
	fmt.Println("--- HTTP method semantics ---")
	code, _ := run(client, base, "GET", "/articles", "")
	check("GET /articles (list, safe+idempotent)", code, 200)

	code, body := run(client, base, "POST", "/articles", `{"title":"New Post","body":"content","author_id":1}`)
	check("POST /articles (create, not idempotent)", code, 201)
	_ = body

	code, _ = run(client, base, "GET", "/articles/1", "")
	check("GET /articles/1 (safe+idempotent)", code, 200)

	code, _ = run(client, base, "PUT", "/articles/1", `{"title":"Updated Title"}`)
	check("PUT /articles/1 (full replace, idempotent)", code, 200)

	code, _ = run(client, base, "DELETE", "/articles/2", "")
	check("DELETE /articles/2 (idempotent)", code, 204)

	code, _ = run(client, base, "DELETE", "/articles/2", "")
	check("DELETE /articles/2 again (idempotent → 404)", code, 404)

	// State transitions via sub-resources.
	fmt.Println()
	fmt.Println("--- State transitions ---")
	code, _ = run(client, base, "POST", "/articles/1/publish", "")
	check("POST /articles/1/publish (state transition)", code, 200)

	code, _ = run(client, base, "POST", "/articles/1/publish", "")
	check("POST /articles/1/publish again (idempotent)", code, 200)

	// Error responses.
	fmt.Println()
	fmt.Println("--- Error responses ---")
	code, _ = run(client, base, "GET", "/articles/99", "")
	check("GET /articles/99 (not found → 404)", code, 404)

	code, _ = run(client, base, "POST", "/articles", `{"title":""}`)
	check("POST invalid title (→ 422)", code, 422)

	code, _ = run(client, base, "POST", "/articles", `not-json`)
	check("POST bad JSON (→ 400)", code, 400)

	code, _ = run(client, base, "PATCH", "/articles", "")
	check("PATCH /articles (→ 405 + Allow header)", code, 405)

	// HATEOAS links.
	fmt.Println()
	fmt.Println("--- HATEOAS links on GET /articles/3 ---")
	_, resp := run(client, base, "GET", "/articles/3", "")
	var ar ArticleResponse
	json.Unmarshal([]byte(resp), &ar)
	for _, l := range ar.Links {
		fmt.Printf("  rel=%-10s method=%-7s href=%s\n", l.Rel, l.Method, l.Href)
	}
}
