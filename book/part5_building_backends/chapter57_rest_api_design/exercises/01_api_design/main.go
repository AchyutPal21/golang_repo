// FILE: book/part5_building_backends/chapter57_rest_api_design/exercises/01_api_design/main.go
// CHAPTER: 57 — REST API Design
// EXERCISE: Build a Books API applying REST principles:
//   - Proper resource naming (/books, /books/{id})
//   - Full CRUD with correct HTTP method semantics
//   - Status codes: 200, 201, 204, 400, 404, 405, 422
//   - Location header on 201
//   - HATEOAS _links on every response
//   - Cursor-based pagination on GET /books
//   - URL versioning: /v1/books returns title+author,
//     /v2/books adds ISBN and year
//
// Run (from the chapter folder):
//   go run ./exercises/01_api_design

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

type Book struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	ISBN      string    `json:"isbn"`
	Year      int       `json:"year"`
	CreatedAt time.Time `json:"created_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// HATEOAS
// ─────────────────────────────────────────────────────────────────────────────

type Link struct {
	Href   string `json:"href"`
	Method string `json:"method"`
	Rel    string `json:"rel"`
}

type BookResponse struct {
	*Book
	Links []Link `json:"_links"`
}

func bookLinks(base string, b *Book) []Link {
	return []Link{
		{Href: fmt.Sprintf("%s/v1/books/%d", base, b.ID), Method: "GET", Rel: "self"},
		{Href: fmt.Sprintf("%s/v1/books/%d", base, b.ID), Method: "PUT", Rel: "update"},
		{Href: fmt.Sprintf("%s/v1/books/%d", base, b.ID), Method: "DELETE", Rel: "delete"},
		{Href: fmt.Sprintf("%s/v1/books", base), Method: "GET", Rel: "collection"},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// STORE
// ─────────────────────────────────────────────────────────────────────────────

type Store struct {
	mu     sync.RWMutex
	items  map[int]*Book
	nextID int
}

func NewStore() *Store {
	s := &Store{items: make(map[int]*Book), nextID: 1}
	seed := []struct{ title, author, isbn string; year int }{
		{"The Go Programming Language", "Donovan & Kernighan", "978-0-13-419562-1", 2015},
		{"Concurrency in Go", "Katherine Cox-Buday", "978-1-491-94119-5", 2017},
		{"Clean Code", "Robert C. Martin", "978-0-13-235088-4", 2008},
		{"Designing Data-Intensive Applications", "Martin Kleppmann", "978-1-449-37332-0", 2017},
		{"The Pragmatic Programmer", "Hunt & Thomas", "978-0-13-595705-9", 2019},
	}
	for _, b := range seed {
		s.items[s.nextID] = &Book{
			ID:        s.nextID,
			Title:     b.title,
			Author:    b.author,
			ISBN:      b.isbn,
			Year:      b.year,
			CreatedAt: time.Now().UTC(),
		}
		s.nextID++
	}
	return s
}

func (s *Store) ListAfter(afterID, limit int) ([]*Book, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Book
	for id := afterID + 1; id < s.nextID; id++ {
		if b, ok := s.items[id]; ok {
			result = append(result, b)
			if len(result) == limit+1 {
				break
			}
		}
	}
	hasMore := len(result) > limit
	if hasMore {
		result = result[:limit]
	}
	return result, hasMore
}

func (s *Store) Get(id int) (*Book, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.items[id]
	return b, ok
}

func (s *Store) Create(b *Book) *Book {
	s.mu.Lock()
	defer s.mu.Unlock()
	b.ID = s.nextID
	b.CreatedAt = time.Now().UTC()
	s.items[s.nextID] = b
	s.nextID++
	return b
}

func (s *Store) Update(id int, fn func(*Book)) (*Book, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.items[id]
	if !ok {
		return nil, false
	}
	fn(b)
	return b, true
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
// HANDLER — V1 (title + author only in list/detail)
// ─────────────────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

type Handler struct {
	store   *Store
	baseURL string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Strip /v1/books prefix to get the resource path.
	path := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/books"), "/")

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
		h.deleteBook(w, id)
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

type CursorPage struct {
	Data       []BookResponse `json:"data"`
	NextCursor int            `json:"next_cursor,omitempty"`
	HasMore    bool           `json:"has_more"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	after := 0
	if s := r.URL.Query().Get("after"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			after = v
		}
	}
	limit := 5
	if s := r.URL.Query().Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 20 {
			limit = v
		}
	}
	books, hasMore := h.store.ListAfter(after, limit)
	resp := make([]BookResponse, len(books))
	for i, b := range books {
		resp[i] = BookResponse{Book: b, Links: bookLinks(h.baseURL, b)}
	}
	page := CursorPage{Data: resp, HasMore: hasMore}
	if hasMore && len(books) > 0 {
		page.NextCursor = books[len(books)-1].ID
	}
	if page.Data == nil {
		page.Data = []BookResponse{}
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) get(w http.ResponseWriter, id int) {
	b, ok := h.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "book not found")
		return
	}
	writeJSON(w, http.StatusOK, BookResponse{Book: b, Links: bookLinks(h.baseURL, b)})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var b Book
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if strings.TrimSpace(b.Title) == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}
	if strings.TrimSpace(b.Author) == "" {
		writeError(w, http.StatusUnprocessableEntity, "author is required")
		return
	}
	created := h.store.Create(&b)
	w.Header().Set("Location", fmt.Sprintf("%s/v1/books/%d", h.baseURL, created.ID))
	writeJSON(w, http.StatusCreated, BookResponse{Book: created, Links: bookLinks(h.baseURL, created)})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id int) {
	var patch Book
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	updated, ok := h.store.Update(id, func(b *Book) {
		if patch.Title != "" {
			b.Title = patch.Title
		}
		if patch.Author != "" {
			b.Author = patch.Author
		}
		if patch.ISBN != "" {
			b.ISBN = patch.ISBN
		}
		if patch.Year != 0 {
			b.Year = patch.Year
		}
	})
	if !ok {
		writeError(w, http.StatusNotFound, "book not found")
		return
	}
	writeJSON(w, http.StatusOK, BookResponse{Book: updated, Links: bookLinks(h.baseURL, updated)})
}

func (h *Handler) deleteBook(w http.ResponseWriter, id int) {
	if !h.store.Delete(id) {
		writeError(w, http.StatusNotFound, "book not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────────────────────
// V2 — adds isbn and year fields to list response
// ─────────────────────────────────────────────────────────────────────────────

type BookV2 struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	ISBN   string `json:"isbn"`
	Year   int    `json:"year"`
}

func listV2(store *Store, w http.ResponseWriter, r *http.Request) {
	books, _ := store.ListAfter(0, 10)
	out := make([]BookV2, len(books))
	for i, b := range books {
		out[i] = BookV2{ID: b.ID, Title: b.Title, Author: b.Author, ISBN: b.ISBN, Year: b.Year}
	}
	writeJSON(w, http.StatusOK, out)
}

// ─────────────────────────────────────────────────────────────────────────────
// TEST HARNESS
// ─────────────────────────────────────────────────────────────────────────────

func request(client *http.Client, method, url, body string) (int, string, string) {
	var br *strings.Reader
	if body != "" {
		br = strings.NewReader(body)
	} else {
		br = strings.NewReader("")
	}
	req, _ := http.NewRequest(method, url, br)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err.Error()
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	buf := make([]byte, 8192)
	n, _ := resp.Body.Read(buf)
	return resp.StatusCode, loc, strings.TrimSpace(string(buf[:n]))
}

func main() {
	store := NewStore()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()

	h := &Handler{store: store, baseURL: base}
	mux := http.NewServeMux()
	mux.Handle("/v1/books", h)
	mux.Handle("/v1/books/", h)
	mux.HandleFunc("/v2/books", func(w http.ResponseWriter, r *http.Request) {
		listV2(store, w, r)
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}

	fmt.Printf("=== Books API — %s ===\n\n", base)

	check := func(label string, code, want int) {
		mark := "✓"
		if code != want {
			mark = "✗"
		}
		fmt.Printf("  %s %-48s %d\n", mark, label, code)
	}

	// CRUD
	fmt.Println("--- CRUD operations ---")
	code, _, _ := request(client, "GET", base+"/v1/books", "")
	check("GET /v1/books (list, cursor page 1)", code, 200)

	code, loc, body := request(client, "POST", base+"/v1/books",
		`{"title":"Domain-Driven Design","author":"Eric Evans","isbn":"978-0-32-112521-7","year":2003}`)
	check("POST /v1/books (create → 201)", code, 201)
	fmt.Printf("    Location: %s\n", loc)

	code, _, _ = request(client, "GET", base+"/v1/books/1", "")
	check("GET /v1/books/1 (single resource)", code, 200)

	code, _, _ = request(client, "PUT", base+"/v1/books/1", `{"title":"The Go Programming Language (2nd Ed.)"}`)
	check("PUT /v1/books/1 (update → 200)", code, 200)

	code, _, _ = request(client, "DELETE", base+"/v1/books/3", "")
	check("DELETE /v1/books/3 → 204", code, 204)

	code, _, _ = request(client, "DELETE", base+"/v1/books/3", "")
	check("DELETE /v1/books/3 again (idempotent → 404)", code, 404)

	// Error cases
	fmt.Println()
	fmt.Println("--- Error responses ---")
	code, _, _ = request(client, "GET", base+"/v1/books/99", "")
	check("GET /v1/books/99 (→ 404)", code, 404)

	code, _, _ = request(client, "POST", base+"/v1/books", `{"author":"No Title"}`)
	check("POST missing title (→ 422)", code, 422)

	code, _, _ = request(client, "POST", base+"/v1/books", `{"title":"No Author"}`)
	check("POST missing author (→ 422)", code, 422)

	code, _, _ = request(client, "POST", base+"/v1/books", `not-json`)
	check("POST bad JSON (→ 400)", code, 400)

	code, _, _ = request(client, "PATCH", base+"/v1/books/1", "")
	check("PATCH /v1/books/1 (→ 405)", code, 405)

	// Pagination
	fmt.Println()
	fmt.Println("--- Cursor pagination ---")
	code, _, body = request(client, "GET", base+"/v1/books?limit=3", "")
	check("GET /v1/books?limit=3 (first page)", code, 200)
	var pg CursorPage
	json.Unmarshal([]byte(body), &pg)
	fmt.Printf("    returned=%d has_more=%v next_cursor=%d\n", len(pg.Data), pg.HasMore, pg.NextCursor)

	// HATEOAS
	fmt.Println()
	fmt.Println("--- HATEOAS links on GET /v1/books/1 ---")
	_, _, body = request(client, "GET", base+"/v1/books/1", "")
	var br BookResponse
	json.Unmarshal([]byte(body), &br)
	for _, l := range br.Links {
		fmt.Printf("    rel=%-12s method=%-8s href=%s\n", l.Rel, l.Method, l.Href)
	}

	// URL versioning
	fmt.Println()
	fmt.Println("--- URL versioning ---")
	code, _, body = request(client, "GET", base+"/v2/books", "")
	check("GET /v2/books (extended fields)", code, 200)
	var v2books []BookV2
	json.Unmarshal([]byte(body), &v2books)
	if len(v2books) > 0 {
		fmt.Printf("    first book: id=%d title=%q isbn=%s year=%d\n",
			v2books[0].ID, v2books[0].Title, v2books[0].ISBN, v2books[0].Year)
	}
}
