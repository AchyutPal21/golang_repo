// FILE: book/part5_building_backends/chapter61_authorization/exercises/01_permission_system/main.go
// CHAPTER: 61 — Authorization
// EXERCISE: Combined RBAC + resource-level permission system.
//
// Rules:
//   - viewer  : read any doc, cannot write/delete anything
//   - editor  : read any doc, edit any doc, cannot delete
//   - admin   : full CRUD on any doc
//   - owner   : can always edit and delete their own doc (overrides viewer role)
//
// Users:
//   alice  (admin)   — can do everything
//   bob    (editor)  — can read+edit any, cannot delete
//   carol  (viewer)  — read-only
//   dave   (viewer)  — owns doc 1; can edit+delete his own doc
//
// Run (from the chapter folder):
//   go run ./exercises/01_permission_system

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
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Role string

const (
	RoleViewer Role = "viewer"
	RoleEditor Role = "editor"
	RoleAdmin  Role = "admin"
)

type Action string

const (
	ActionRead   Action = "read"
	ActionEdit   Action = "edit"
	ActionDelete Action = "delete"
)

type User struct {
	ID   string
	Name string
	Role Role
}

type Document struct {
	ID      int
	Title   string
	Content string
	OwnerID string
}

// ─────────────────────────────────────────────────────────────────────────────
// PERMISSION LOGIC
// ─────────────────────────────────────────────────────────────────────────────

// roleAllows returns true if the role allows the action, ignoring ownership.
func roleAllows(role Role, action Action) bool {
	switch role {
	case RoleAdmin:
		return true
	case RoleEditor:
		return action == ActionRead || action == ActionEdit
	case RoleViewer:
		return action == ActionRead
	}
	return false
}

// CanPerform is the combined RBAC + resource-level authorisation check.
// An owner can always edit or delete their own document (ownership override).
func CanPerform(user User, action Action, doc Document) bool {
	// Role check first (covers admin and editor permissions)
	if roleAllows(user.Role, action) {
		return true
	}
	// Ownership override: owner can edit or delete their own document
	if user.ID == doc.OwnerID && (action == ActionEdit || action == ActionDelete) {
		return true
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// IN-MEMORY STORE
// ─────────────────────────────────────────────────────────────────────────────

type Store struct {
	mu   sync.RWMutex
	docs map[int]Document
	next int
}

func newStore() *Store {
	s := &Store{docs: make(map[int]Document), next: 3}
	s.docs[1] = Document{ID: 1, Title: "Dave's Guide", Content: "content of doc 1", OwnerID: "dave"}
	s.docs[2] = Document{ID: 2, Title: "Alice's Report", Content: "content of doc 2", OwnerID: "alice"}
	return s
}

func (s *Store) List() []Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Document, 0, len(s.docs))
	for _, d := range s.docs {
		out = append(out, d)
	}
	return out
}

func (s *Store) Get(id int) (Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.docs[id]
	return d, ok
}

func (s *Store) Create(title, content, ownerID string) Document {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.next
	s.next++
	d := Document{ID: id, Title: title, Content: content, OwnerID: ownerID}
	s.docs[id] = d
	return d
}

func (s *Store) Update(id int, title, content string) (Document, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.docs[id]
	if !ok {
		return Document{}, false
	}
	if title != "" {
		d.Title = title
	}
	if content != "" {
		d.Content = content
	}
	s.docs[id] = d
	return d, true
}

func (s *Store) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.docs[id]; !ok {
		return false
	}
	delete(s.docs, id)
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP HELPERS
// ─────────────────────────────────────────────────────────────────────────────

var userDB = map[string]User{
	"alice": {ID: "alice", Name: "Alice", Role: RoleAdmin},
	"bob":   {ID: "bob", Name: "Bob", Role: RoleEditor},
	"carol": {ID: "carol", Name: "Carol", Role: RoleViewer},
	"dave":  {ID: "dave", Name: "Dave", Role: RoleViewer},
}

// userFromHeader parses "Authorization: Bearer <username>" for demo purposes.
func userFromHeader(r *http.Request) (User, bool) {
	auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	u, ok := userDB[auth]
	return u, ok
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func handleListDocs(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFromHeader(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		// All authenticated users can list (read)
		docs := store.List()
		// Filter to only docs the user can read (all roles can read here)
		_ = user
		writeJSON(w, http.StatusOK, docs)
	}
}

func handleCreateDoc(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFromHeader(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		// Creating a document is a pure role-level check: only admins and editors
		// may create.  Ownership override does NOT apply here (there is no existing
		// document to own yet), so we use roleAllows directly.
		if !roleAllows(user.Role, ActionEdit) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden: insufficient role to create documents"})
			return
		}
		var body struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		doc := store.Create(body.Title, body.Content, user.ID)
		writeJSON(w, http.StatusCreated, doc)
	}
}

func handleGetDoc(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFromHeader(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}
		doc, found := store.Get(id)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if !CanPerform(user, ActionRead, doc) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		writeJSON(w, http.StatusOK, doc)
	}
}

func handleUpdateDoc(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFromHeader(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}
		doc, found := store.Get(id)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if !CanPerform(user, ActionEdit, doc) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden: cannot edit this document"})
			return
		}
		var body struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		updated, _ := store.Update(id, body.Title, body.Content)
		writeJSON(w, http.StatusOK, updated)
	}
}

func handleDeleteDoc(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFromHeader(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}
		doc, found := store.Get(id)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if !CanPerform(user, ActionDelete, doc) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden: cannot delete this document"})
			return
		}
		store.Delete(id)
		writeJSON(w, http.StatusOK, map[string]string{"msg": "deleted"})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	store := newStore()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /docs", handleListDocs(store))
	mux.HandleFunc("POST /docs", handleCreateDoc(store))
	mux.HandleFunc("GET /docs/{id}", handleGetDoc(store))
	mux.HandleFunc("PUT /docs/{id}", handleUpdateDoc(store))
	mux.HandleFunc("DELETE /docs/{id}", handleDeleteDoc(store))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: mux}).Serve(ln) //nolint:errcheck

	client := &http.Client{Timeout: 3 * time.Second}

	do := func(method, path, user, body string) int {
		var bodyR *strings.Reader
		if body != "" {
			bodyR = strings.NewReader(body)
		} else {
			bodyR = strings.NewReader("")
		}
		req, _ := http.NewRequest(method, base+path, bodyR)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if user != "" {
			req.Header.Set("Authorization", "Bearer "+user)
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0
		}
		resp.Body.Close()
		return resp.StatusCode
	}

	check := func(label string, got, want int) {
		mark := "✓"
		if got != want {
			mark = "✗"
		}
		fmt.Printf("  %s %-62s got=%d want=%d\n", mark, label, got, want)
	}

	fmt.Printf("=== Combined RBAC + Resource-Level Permissions — %s ===\n\n", base)
	fmt.Println("Users:")
	fmt.Println("  alice (admin)  — full access to everything")
	fmt.Println("  bob   (editor) — read + edit any doc, cannot delete")
	fmt.Println("  carol (viewer) — read-only, cannot edit or delete")
	fmt.Println("  dave  (viewer) — owns doc 1, can edit+delete his own doc")
	fmt.Println()
	fmt.Println("Documents: doc 1 (owner=dave), doc 2 (owner=alice)")
	fmt.Println()

	fmt.Println("--- carol (viewer): read-only ---")
	check("GET  /docs          (carol=viewer)", do("GET", "/docs", "carol", ""), 200)
	check("GET  /docs/1        (carol=viewer)", do("GET", "/docs/1", "carol", ""), 200)
	check("POST /docs          (carol=viewer) → 403", do("POST", "/docs", "carol", `{"title":"t","content":"c"}`), 403)
	check("PUT  /docs/1        (carol=viewer) → 403", do("PUT", "/docs/1", "carol", `{"title":"updated"}`), 403)
	check("DELETE /docs/1      (carol=viewer) → 403", do("DELETE", "/docs/1", "carol", ""), 403)

	fmt.Println()
	fmt.Println("--- dave (viewer + owner of doc 1): ownership override ---")
	check("GET  /docs/1        (dave=viewer, owner)", do("GET", "/docs/1", "dave", ""), 200)
	check("PUT  /docs/1        (dave=viewer, owner) → 200", do("PUT", "/docs/1", "dave", `{"title":"Dave Updated"}`), 200)
	check("DELETE /docs/1      (dave=viewer, owner) → 200", do("DELETE", "/docs/1", "dave", ""), 200)
	// doc 2 is owned by alice, dave cannot edit/delete it
	check("PUT  /docs/2        (dave=viewer, not owner) → 403", do("PUT", "/docs/2", "dave", `{"title":"hack"}`), 403)
	check("DELETE /docs/2      (dave=viewer, not owner) → 403", do("DELETE", "/docs/2", "dave", ""), 403)

	fmt.Println()
	fmt.Println("--- bob (editor): can edit any doc, cannot delete ---")
	check("GET  /docs/2        (bob=editor)", do("GET", "/docs/2", "bob", ""), 200)
	check("POST /docs          (bob=editor) → 201", do("POST", "/docs", "bob", `{"title":"Bob's Doc","content":"hello"}`), 201)
	check("PUT  /docs/2        (bob=editor) → 200", do("PUT", "/docs/2", "bob", `{"title":"Bob Updated"}`), 200)
	check("DELETE /docs/2      (bob=editor) → 403", do("DELETE", "/docs/2", "bob", ""), 403)

	fmt.Println()
	fmt.Println("--- alice (admin): full access ---")
	check("GET  /docs          (alice=admin)", do("GET", "/docs", "alice", ""), 200)
	check("POST /docs          (alice=admin) → 201", do("POST", "/docs", "alice", `{"title":"Alice's New Doc","content":"x"}`), 201)
	check("PUT  /docs/2        (alice=admin) → 200", do("PUT", "/docs/2", "alice", `{"title":"Alice Updated"}`), 200)
	check("DELETE /docs/2      (alice=admin) → 200", do("DELETE", "/docs/2", "alice", ""), 200)

	fmt.Println()
	fmt.Println("--- unauthenticated ---")
	check("GET  /docs          (no auth) → 401", do("GET", "/docs", "", ""), 401)
}
