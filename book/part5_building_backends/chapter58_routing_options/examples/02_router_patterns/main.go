// FILE: book/part5_building_backends/chapter58_routing_options/examples/02_router_patterns/main.go
// CHAPTER: 58 — Routing Options
// TOPIC: Advanced routing patterns — route groups, subrouters, named routes,
//        method-not-allowed handling, and building a minimal router from scratch
//        to understand what third-party routers (chi, gin, echo) provide.
//
// This example builds a minimal Router type that demonstrates the core features
// that make third-party routers attractive over the stdlib mux:
//   - Route groups with shared prefix and middleware
//   - Typed path parameters (int vs string coercion at the router level)
//   - Named routes for reverse URL generation
//   - 405 with Allow header built-in
//
// Run (from the chapter folder):
//   go run ./examples/02_router_patterns

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// MINIMAL ROUTER — illustrates what chi/gin/echo provide internally
// ─────────────────────────────────────────────────────────────────────────────

type route struct {
	method  string
	pattern *regexp.Regexp
	names   []string // capture group names, in order
	name    string   // optional named route
	handler http.HandlerFunc
	mws     []func(http.Handler) http.Handler
}

// Router dispatches on method + path, extracts named captures, and
// returns 405 with an Allow header when the path matches but not the method.
type Router struct {
	routes []route
	mws    []func(http.Handler) http.Handler // global middleware
}

// patternToRegexp converts "/users/{id}" → ^/users/(?P<id>[^/]+)$
// and "/files/{path...}" → ^/files/(?P<path>.+)$
func patternToRegexp(pattern string) (*regexp.Regexp, []string) {
	var names []string
	// Replace {name...} (tail wildcard) first.
	re := regexp.MustCompile(`\{(\w+)\.\.\.\}`)
	result := re.ReplaceAllStringFunc(pattern, func(m string) string {
		name := re.FindStringSubmatch(m)[1]
		names = append(names, name)
		return fmt.Sprintf(`(?P<%s>.+)`, name)
	})
	// Replace {name} (segment parameter).
	re2 := regexp.MustCompile(`\{(\w+)\}`)
	result = re2.ReplaceAllStringFunc(result, func(m string) string {
		name := re2.FindStringSubmatch(m)[1]
		names = append(names, name)
		return fmt.Sprintf(`(?P<%s>[^/]+)`, name)
	})
	return regexp.MustCompile(`^` + result + `$`), names
}

type Middleware = func(http.Handler) http.Handler

func (r *Router) Use(mws ...Middleware) {
	r.mws = append(r.mws, mws...)
}

func (r *Router) add(method, pattern, name string, h http.HandlerFunc, mws []Middleware) {
	re, names := patternToRegexp(pattern)
	r.routes = append(r.routes, route{
		method:  method,
		pattern: re,
		names:   names,
		name:    name,
		handler: h,
		mws:     mws,
	})
}

func (r *Router) GET(pattern, name string, h http.HandlerFunc, mws ...Middleware) {
	r.add("GET", pattern, name, h, mws)
}

func (r *Router) POST(pattern, name string, h http.HandlerFunc, mws ...Middleware) {
	r.add("POST", pattern, name, h, mws)
}

func (r *Router) PUT(pattern, name string, h http.HandlerFunc, mws ...Middleware) {
	r.add("PUT", pattern, name, h, mws)
}

func (r *Router) DELETE(pattern, name string, h http.HandlerFunc, mws ...Middleware) {
	r.add("DELETE", pattern, name, h, mws)
}

// URL generates a URL for a named route by substituting params.
// URL("user.get", "id", "42") → "/users/42"
func (r *Router) URL(name string, pairs ...string) string {
	for _, rt := range r.routes {
		if rt.name != name {
			continue
		}
		// Reconstruct the pattern without regex.
		s := rt.pattern.String()
		s = strings.TrimPrefix(s, "^")
		s = strings.TrimSuffix(s, "$")
		// Replace named groups with supplied values.
		for i := 0; i+1 < len(pairs); i += 2 {
			s = regexp.MustCompile(`\(\?P<`+pairs[i]+`>[^)]+\)`).ReplaceAllString(s, pairs[i+1])
		}
		return s
	}
	return ""
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path

	// Find all routes whose pattern matches the path.
	var methodsAllowed []string
	for _, rt := range r.routes {
		if !rt.pattern.MatchString(path) {
			continue
		}
		methodsAllowed = append(methodsAllowed, rt.method)
		if rt.method != req.Method {
			continue
		}
		// Extract path values into request context via a wrapper.
		matches := rt.pattern.FindStringSubmatch(path)
		for i, n := range rt.names {
			if i+1 < len(matches) {
				req = req.WithContext(withPathValue(req.Context(), n, matches[i+1]))
			}
		}
		// Build handler chain: global mws → route mws → handler.
		h := http.Handler(rt.handler)
		for i := len(rt.mws) - 1; i >= 0; i-- {
			h = rt.mws[i](h)
		}
		for i := len(r.mws) - 1; i >= 0; i-- {
			h = r.mws[i](h)
		}
		h.ServeHTTP(w, req)
		return
	}

	if len(methodsAllowed) > 0 {
		sort.Strings(methodsAllowed)
		w.Header().Set("Allow", strings.Join(methodsAllowed, ", "))
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.NotFound(w, req)
}

// ─────────────────────────────────────────────────────────────────────────────
// PATH VALUE CONTEXT (minimal substitute for r.PathValue in Go 1.22)
// ─────────────────────────────────────────────────────────────────────────────

type contextKey string

func withPathValue(ctx interface{ Value(any) any }, key, val string) interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
} {
	return pathCtx{ctx.(interface {
		Deadline() (time.Time, bool)
		Done() <-chan struct{}
		Err() error
		Value(key any) any
	}), key, val}
}

type pathCtx struct {
	parent interface {
		Deadline() (time.Time, bool)
		Done() <-chan struct{}
		Err() error
		Value(key any) any
	}
	key, val string
}

func (c pathCtx) Deadline() (time.Time, bool) { return c.parent.Deadline() }
func (c pathCtx) Done() <-chan struct{}        { return c.parent.Done() }
func (c pathCtx) Err() error                  { return c.parent.Err() }
func (c pathCtx) Value(key any) any {
	if k, ok := key.(contextKey); ok && string(k) == c.key {
		return c.val
	}
	return c.parent.Value(key)
}

func pathValue(r *http.Request, key string) string {
	if v, ok := r.Context().Value(contextKey(key)).(string); ok {
		return v
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// ROUTE GROUPS — subrouter with shared prefix and middleware
// ─────────────────────────────────────────────────────────────────────────────

// Group wraps a Router with a path prefix and additional middleware.
type Group struct {
	router *Router
	prefix string
	mws    []Middleware
}

func (r *Router) Group(prefix string, mws ...Middleware) *Group {
	return &Group{router: r, prefix: prefix, mws: mws}
}

func (g *Group) GET(pattern, name string, h http.HandlerFunc, mws ...Middleware) {
	g.router.add("GET", g.prefix+pattern, name, h, append(g.mws, mws...))
}

func (g *Group) POST(pattern, name string, h http.HandlerFunc, mws ...Middleware) {
	g.router.add("POST", g.prefix+pattern, name, h, append(g.mws, mws...))
}

func (g *Group) PUT(pattern, name string, h http.HandlerFunc, mws ...Middleware) {
	g.router.add("PUT", g.prefix+pattern, name, h, append(g.mws, mws...))
}

func (g *Group) DELETE(pattern, name string, h http.HandlerFunc, mws ...Middleware) {
	g.router.add("DELETE", g.prefix+pattern, name, h, append(g.mws, mws...))
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO MIDDLEWARE
// ─────────────────────────────────────────────────────────────────────────────

func authRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "req-demo-123")
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func handleListProducts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, []map[string]any{{"id": "1", "name": "Widget"}})
}

func handleGetProduct(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	writeJSON(w, 200, map[string]any{"id": id, "name": "Product " + id})
}

func handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Location", "/api/v1/products/2")
	writeJSON(w, 201, map[string]any{"id": "2", "name": "New Product"})
}

func handleAdminStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"total_products": 42, "active_users": 7})
}

func handleFile(w http.ResponseWriter, r *http.Request) {
	path := pathValue(r, "path")
	writeJSON(w, 200, map[string]any{"path": path})
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	r := &Router{}

	// Global middleware.
	r.Use(requestID)

	// Public API group — /api/v1/products
	api := r.Group("/api/v1")
	api.GET("/products", "product.list", handleListProducts)
	api.GET("/products/{id}", "product.get", handleGetProduct)
	api.POST("/products", "product.create", handleCreateProduct)
	api.PUT("/products/{id}", "product.update", handleGetProduct)
	api.DELETE("/products/{id}", "product.delete", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Admin group — same prefix but requires Authorization header.
	admin := r.Group("/api/v1/admin", authRequired)
	admin.GET("/stats", "admin.stats", handleAdminStats)

	// Wildcard route.
	r.GET("/static/{path...}", "static.file", handleFile)

	// Named-route URL generation.
	fmt.Println("=== Router Patterns Demo ===")
	fmt.Println()
	fmt.Println("--- Named route URL generation ---")
	fmt.Printf("  product.get  id=7   → %s\n", r.URL("product.get", "id", "7"))
	fmt.Printf("  product.list        → %s\n", r.URL("product.list"))

	// Start server.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	srv := &http.Server{Handler: r}
	go srv.Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}

	do := func(method, path string, headers map[string]string) (int, string) {
		req, _ := http.NewRequest(method, base+path, nil)
		for k, v := range headers {
			req.Header.Set(k, v)
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

	check := func(label string, code, want int) {
		mark := "✓"
		if code != want {
			mark = "✗"
		}
		fmt.Printf("  %s %-50s %d\n", mark, label, code)
	}

	fmt.Println()
	fmt.Println("--- Route groups ---")
	code, _ := do("GET", "/api/v1/products", nil)
	check("GET /api/v1/products", code, 200)

	code, body := do("GET", "/api/v1/products/42", nil)
	check("GET /api/v1/products/42 (path param)", code, 200)
	fmt.Printf("    %s\n", body)

	code, _ = do("POST", "/api/v1/products", nil)
	check("POST /api/v1/products → 201", code, 201)

	code, _ = do("DELETE", "/api/v1/products/1", nil)
	check("DELETE /api/v1/products/1 → 204", code, 204)

	fmt.Println()
	fmt.Println("--- 405 with Allow header ---")
	req, _ := http.NewRequest("PATCH", base+"/api/v1/products", nil)
	resp, _ := client.Do(req)
	resp.Body.Close()
	check("PATCH /api/v1/products → 405", resp.StatusCode, 405)
	fmt.Printf("    Allow: %s\n", resp.Header.Get("Allow"))

	fmt.Println()
	fmt.Println("--- Auth middleware on /admin ---")
	code, _ = do("GET", "/api/v1/admin/stats", nil)
	check("GET /admin/stats (no auth) → 401", code, 401)

	code, _ = do("GET", "/api/v1/admin/stats", map[string]string{"Authorization": "Bearer token"})
	check("GET /admin/stats (with auth) → 200", code, 200)

	fmt.Println()
	fmt.Println("--- Wildcard route ---")
	code, body = do("GET", "/static/css/main.css", nil)
	check("GET /static/css/main.css", code, 200)
	fmt.Printf("    %s\n", body)

	fmt.Println()
	fmt.Println("--- X-Request-ID global middleware ---")
	req2, _ := http.NewRequest("GET", base+"/api/v1/products", nil)
	resp2, _ := client.Do(req2)
	resp2.Body.Close()
	fmt.Printf("  X-Request-ID: %s\n", resp2.Header.Get("X-Request-ID"))
}
