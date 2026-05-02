// FILE: book/part5_building_backends/chapter58_routing_options/examples/01_stdlib_routing/main.go
// CHAPTER: 58 — Routing Options
// TOPIC: net/http 1.22 enhanced routing — method-qualified patterns, path parameters,
//        wildcard matching, route specificity, and middleware chaining on the stdlib mux.
//
// Go 1.22 introduced method-qualified patterns ("GET /path") and named path
// wildcards ("{id}") directly into net/http.ServeMux, eliminating the need for
// a third-party router in most CRUD APIs.
//
// Run (from the chapter folder):
//   go run ./examples/01_stdlib_routing

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// writeJSON is a minimal helper to write a JSON response.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE HELPERS
// ─────────────────────────────────────────────────────────────────────────────

type middleware func(http.Handler) http.Handler

// chain applies middlewares right-to-left so the first in the list is outermost.
func chain(h http.Handler, mws ...middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rw, r)
		fmt.Printf("    [log] %-7s %-30s %d %dms\n",
			r.Method, r.URL.Path, rw.code, time.Since(start).Milliseconds())
	})
}

type responseWriter struct {
	http.ResponseWriter
	code int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.code = code
	rw.ResponseWriter.WriteHeader(code)
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func listUsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []map[string]any{
		{"id": "1", "name": "Alice"},
		{"id": "2", "name": "Bob"},
	})
}

func createUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Location", "/users/3")
	writeJSON(w, http.StatusCreated, map[string]any{"id": "3", "name": "Carol"})
}

func getUser(w http.ResponseWriter, r *http.Request) {
	// Go 1.22: r.PathValue("id") extracts the {id} wildcard.
	id := r.PathValue("id")
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "name": "User " + id})
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "updated": true})
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func getPost(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	postID := r.PathValue("postID")
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
		"post_id": postID,
		"title":   fmt.Sprintf("Post %s by user %s", postID, userID),
	})
}

// catchAll handles /files/{path...} — wildcard captures the remainder.
func catchAll(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	writeJSON(w, http.StatusOK, map[string]any{"file": path})
}

// ─────────────────────────────────────────────────────────────────────────────
// ROUTE REGISTRATION — Go 1.22 syntax
// ─────────────────────────────────────────────────────────────────────────────

func buildMux() http.Handler {
	mux := http.NewServeMux()

	// Method + path patterns (Go 1.22).
	// The mux returns 405 automatically when the path matches but the method does not.
	mux.HandleFunc("GET /users", listUsers)
	mux.HandleFunc("POST /users", createUser)
	mux.HandleFunc("GET /users/{id}", getUser)
	mux.HandleFunc("PUT /users/{id}", updateUser)
	mux.HandleFunc("DELETE /users/{id}", deleteUser)

	// Nested resources: /users/{userID}/posts/{postID}
	mux.HandleFunc("GET /users/{userID}/posts/{postID}", getPost)

	// Wildcard tail: {path...} matches the rest of the URL.
	mux.HandleFunc("GET /files/{path...}", catchAll)

	// Static route is more specific than wildcard — mux picks the longest match.
	mux.HandleFunc("GET /files/index.html", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"file": "index.html (exact match)"})
	})

	return chain(mux, logging)
}

// ─────────────────────────────────────────────────────────────────────────────
// TEST HARNESS
// ─────────────────────────────────────────────────────────────────────────────

func get(client *http.Client, url string) (int, string) {
	resp, err := client.Get(url)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	return resp.StatusCode, strings.TrimSpace(string(buf[:n]))
}

func do(client *http.Client, method, url string) int {
	req, _ := http.NewRequest(method, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	resp.Body.Close()
	return resp.StatusCode
}

func main() {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()

	srv := &http.Server{Handler: buildMux()}
	go srv.Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}

	fmt.Printf("=== stdlib Routing (Go 1.22) — %s ===\n\n", base)

	check := func(label string, code, want int) {
		mark := "✓"
		if code != want {
			mark = "✗"
		}
		fmt.Printf("  %s %-48s %d\n", mark, label, code)
	}

	fmt.Println("--- Method-qualified patterns ---")
	check("GET /users", do(client, "GET", base+"/users"), 200)
	check("POST /users", do(client, "POST", base+"/users"), 201)
	// Go 1.22 mux returns 405 automatically for method mismatch.
	check("DELETE /users → 405", do(client, "DELETE", base+"/users"), 405)

	fmt.Println()
	fmt.Println("--- Path parameters {id} ---")
	check("GET /users/42", do(client, "GET", base+"/users/42"), 200)
	code, body := get(client, base+"/users/42")
	_ = code
	fmt.Printf("    body: %s\n", body)

	check("PUT /users/42", do(client, "PUT", base+"/users/42"), 200)
	check("DELETE /users/42", do(client, "DELETE", base+"/users/42"), 204)

	fmt.Println()
	fmt.Println("--- Nested resources ---")
	check("GET /users/1/posts/99", do(client, "GET", base+"/users/1/posts/99"), 200)
	_, body = get(client, base+"/users/1/posts/99")
	fmt.Printf("    body: %s\n", body)

	fmt.Println()
	fmt.Println("--- Wildcard tail {path...} ---")
	check("GET /files/a/b/c.txt", do(client, "GET", base+"/files/a/b/c.txt"), 200)
	_, body = get(client, base+"/files/a/b/c.txt")
	fmt.Printf("    body: %s\n", body)

	// Exact match wins over wildcard.
	check("GET /files/index.html (exact > wildcard)", do(client, "GET", base+"/files/index.html"), 200)
	_, body = get(client, base+"/files/index.html")
	fmt.Printf("    body: %s\n", body)

	fmt.Println()
	fmt.Println("--- Middleware (logging above) ---")
	do(client, "GET", base+"/users")
	do(client, "GET", base+"/users/7")
	do(client, "DELETE", base+"/users") // triggers 405
}
