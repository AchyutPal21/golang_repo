// FILE: book/part5_building_backends/chapter61_authorization/examples/01_rbac/main.go
// CHAPTER: 61 — Authorization
// TOPIC: Role-Based Access Control (RBAC) —
//        permissions, roles with inheritance, HTTP middleware,
//        and a test harness demonstrating viewer/editor/admin access.
//
// Run (from the chapter folder):
//   go run ./examples/01_rbac

package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// PERMISSIONS
// ─────────────────────────────────────────────────────────────────────────────

type Permission string

const (
	ReadArticles  Permission = "read:articles"
	WriteArticles Permission = "write:articles"
	DeleteArticles Permission = "delete:articles"
	ReadUsers     Permission = "read:users"
	ManageUsers   Permission = "manage:users"
)

// ─────────────────────────────────────────────────────────────────────────────
// RBAC ENGINE
// ─────────────────────────────────────────────────────────────────────────────

// RBAC holds role definitions and an optional parent (inheritance) map.
type RBAC struct {
	// direct permissions granted to each role
	grants map[string][]Permission
	// parent maps a child role to a parent role whose permissions it inherits
	parent map[string]string
}

// NewRBAC builds an RBAC instance with the provided role definitions and
// an inheritance chain expressed as child → parent pairs.
func NewRBAC(grants map[string][]Permission, inheritance map[string]string) *RBAC {
	return &RBAC{grants: grants, parent: inheritance}
}

// permissionsFor returns the full permission set for a role, walking the
// inheritance chain to include parent permissions.
func (r *RBAC) permissionsFor(role string) map[Permission]struct{} {
	set := make(map[Permission]struct{})
	visited := make(map[string]bool)
	cur := role
	for cur != "" && !visited[cur] {
		visited[cur] = true
		for _, p := range r.grants[cur] {
			set[p] = struct{}{}
		}
		cur = r.parent[cur]
	}
	return set
}

// HasPermission returns true if the given role holds the given permission
// (including permissions inherited from parent roles).
func (r *RBAC) HasPermission(role string, perm Permission) bool {
	set := r.permissionsFor(role)
	_, ok := set[perm]
	return ok
}

// ─────────────────────────────────────────────────────────────────────────────
// BUILD THE APPLICATION'S RBAC POLICY
// ─────────────────────────────────────────────────────────────────────────────

func buildRBAC() *RBAC {
	grants := map[string][]Permission{
		// viewer: read articles only
		"viewer": {ReadArticles},
		// editor: adds write on top of viewer (inherits viewer via parent chain)
		"editor": {WriteArticles},
		// admin: adds delete + user management on top of editor
		"admin": {DeleteArticles, ReadUsers, ManageUsers},
	}
	// admin inherits editor; editor inherits viewer
	inheritance := map[string]string{
		"editor": "viewer",
		"admin":  "editor",
	}
	return NewRBAC(grants, inheritance)
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE
// ─────────────────────────────────────────────────────────────────────────────

// RequirePermission reads the user's role from the Authorization header:
//
//	Authorization: Bearer role-<rolename>
//
// It then checks the role against rbac.HasPermission.  Missing auth → 401,
// insufficient role → 403, permission granted → calls next.
func RequirePermission(rbac *RBAC, perm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer role-") {
				http.Error(w, `{"error":"missing or malformed Authorization header"}`, http.StatusUnauthorized)
				return
			}
			role := strings.TrimPrefix(auth, "Bearer role-")
			if role == "" {
				http.Error(w, `{"error":"empty role"}`, http.StatusUnauthorized)
				return
			}
			if !rbac.HasPermission(role, perm) {
				http.Error(w, fmt.Sprintf(`{"error":"role %q lacks permission %q"}`, role, perm), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"ok":true,"msg":%q}`, msg)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	rbac := buildRBAC()

	// ── demonstrate the permission table ─────────────────────────────────────
	fmt.Println("=== RBAC Permission Table ===")
	roles := []string{"viewer", "editor", "admin"}
	perms := []Permission{ReadArticles, WriteArticles, DeleteArticles, ReadUsers, ManageUsers}
	header := fmt.Sprintf("%-20s", "permission")
	for _, role := range roles {
		header += fmt.Sprintf("  %-8s", role)
	}
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)))
	for _, perm := range perms {
		line := fmt.Sprintf("%-20s", perm)
		for _, role := range roles {
			mark := "no"
			if rbac.HasPermission(role, perm) {
				mark = "YES"
			}
			line += fmt.Sprintf("  %-8s", mark)
		}
		fmt.Println(line)
	}
	fmt.Println()

	// ── build HTTP server ─────────────────────────────────────────────────────
	mux := http.NewServeMux()

	protect := func(perm Permission, msg string) http.Handler {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jsonOK(w, msg)
		})
		return RequirePermission(rbac, perm)(inner)
	}

	mux.Handle("GET /articles", protect(ReadArticles, "articles list"))
	mux.Handle("POST /articles", protect(WriteArticles, "article created"))
	mux.Handle("DELETE /articles/{id}", protect(DeleteArticles, "article deleted"))
	mux.Handle("GET /users", protect(ReadUsers, "users list"))
	mux.Handle("DELETE /users/{id}", protect(ManageUsers, "user deleted"))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: mux}).Serve(ln) //nolint:errcheck

	client := &http.Client{}

	do := func(method, path, role string) int {
		req, _ := http.NewRequest(method, base+path, nil)
		if role != "" {
			req.Header.Set("Authorization", "Bearer role-"+role)
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
		fmt.Printf("  %s %-55s got=%d want=%d\n", mark, label, got, want)
	}

	fmt.Printf("=== HTTP Authorization Tests — %s ===\n\n", base)

	fmt.Println("--- No auth (401 expected) ---")
	check("GET /articles (no auth)", do("GET", "/articles", ""), 401)
	check("POST /articles (no auth)", do("POST", "/articles", ""), 401)

	fmt.Println()
	fmt.Println("--- viewer (can read articles, nothing else) ---")
	check("GET  /articles  (viewer)", do("GET", "/articles", "viewer"), 200)
	check("POST /articles  (viewer)", do("POST", "/articles", "viewer"), 403)
	check("DELETE /articles/1 (viewer)", do("DELETE", "/articles/1", "viewer"), 403)
	check("GET  /users     (viewer)", do("GET", "/users", "viewer"), 403)

	fmt.Println()
	fmt.Println("--- editor (inherits viewer: read+write, not delete/users) ---")
	check("GET  /articles  (editor)", do("GET", "/articles", "editor"), 200)
	check("POST /articles  (editor)", do("POST", "/articles", "editor"), 200)
	check("DELETE /articles/1 (editor)", do("DELETE", "/articles/1", "editor"), 403)
	check("GET  /users     (editor)", do("GET", "/users", "editor"), 403)

	fmt.Println()
	fmt.Println("--- admin (inherits all: full access) ---")
	check("GET  /articles     (admin)", do("GET", "/articles", "admin"), 200)
	check("POST /articles     (admin)", do("POST", "/articles", "admin"), 200)
	check("DELETE /articles/1 (admin)", do("DELETE", "/articles/1", "admin"), 200)
	check("GET  /users        (admin)", do("GET", "/users", "admin"), 200)
	check("DELETE /users/1    (admin)", do("DELETE", "/users/1", "admin"), 200)

	fmt.Println()
	fmt.Println("--- unknown role → 403 (no matching grants) ---")
	check("GET  /articles (unknown role)", do("GET", "/articles", "guest"), 403)
}
