// FILE: book/part5_building_backends/chapter59_middleware/examples/02_context_values/main.go
// CHAPTER: 59 — Middleware
// TOPIC: Passing values through middleware via request context —
//        typed context keys, correlation IDs, authenticated user propagation,
//        and reading context values in handlers and downstream middleware.
//
// Pattern: middleware stores a value in context with r.WithContext(ctx);
// downstream handlers retrieve it with a typed accessor function.
// Never use raw strings as context keys — use unexported custom types.
//
// Run (from the chapter folder):
//   go run ./examples/02_context_values

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT KEY TYPES
// Using unexported types prevents key collisions across packages.
// ─────────────────────────────────────────────────────────────────────────────

type ctxKey int

const (
	keyCorrelationID ctxKey = iota
	keyUser
	keyRequestStart
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	Token string `json:"-"`
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT ACCESSORS — typed helpers to get/set context values safely
// ─────────────────────────────────────────────────────────────────────────────

func withCorrelationID(r *http.Request, id string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), keyCorrelationID, id))
}

func correlationID(r *http.Request) string {
	if v, ok := r.Context().Value(keyCorrelationID).(string); ok {
		return v
	}
	return ""
}

func withUser(r *http.Request, u *User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), keyUser, u))
}

func currentUser(r *http.Request) (*User, bool) {
	u, ok := r.Context().Value(keyUser).(*User)
	return u, ok && u != nil
}

func withRequestStart(r *http.Request, t time.Time) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), keyRequestStart, t))
}

func requestStart(r *http.Request) time.Time {
	if t, ok := r.Context().Value(keyRequestStart).(time.Time); ok {
		return t
	}
	return time.Time{}
}

// ─────────────────────────────────────────────────────────────────────────────
// TOKEN DB (simulated)
// ─────────────────────────────────────────────────────────────────────────────

var tokenDB = map[string]*User{
	"tok-alice": {ID: 1, Name: "Alice", Role: "admin", Token: "tok-alice"},
	"tok-bob":   {ID: 2, Name: "Bob", Role: "reader", Token: "tok-bob"},
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE — CORRELATION ID
// Generates or propagates a correlation ID; stores in context and response header.
// ─────────────────────────────────────────────────────────────────────────────

func correlationMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Correlation-ID")
		if id == "" {
			id = fmt.Sprintf("%x", rand.Int63())
		}
		w.Header().Set("X-Correlation-ID", id)
		next.ServeHTTP(w, withCorrelationID(r, id))
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE — AUTH
// Parses Bearer token, looks up user, stores in context.
// Does NOT reject unauthenticated requests — that's the job of requireAuth.
// ─────────────────────────────────────────────────────────────────────────────

func authMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimPrefix(auth, "Bearer ")
			if u, ok := tokenDB[token]; ok {
				r = withUser(r, u)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE — REQUIRE AUTH
// Must come after authMW. Rejects if no user is in context.
// ─────────────────────────────────────────────────────────────────────────────

func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := currentUser(r); !ok {
			corr := correlationID(r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error":          "authentication required",
				"correlation_id": corr,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE — REQUIRE ROLE
// Must come after authMW + requireAuth.
// ─────────────────────────────────────────────────────────────────────────────

func requireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, _ := currentUser(r)
			if u.Role != role {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error":          "forbidden: requires role " + role,
					"correlation_id": correlationID(r),
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE — TIMING (stores request start time in context for handlers)
// ─────────────────────────────────────────────────────────────────────────────

func timingMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = withRequestStart(r, time.Now())
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

// handleMe returns the current user — only reachable after requireAuth.
func handleMe(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"user":           u,
		"correlation_id": correlationID(r),
		"latency_us":     time.Since(requestStart(r)).Microseconds(),
	})
}

// handleAdmin requires admin role.
func handleAdmin(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"msg":            "welcome to admin panel",
		"user":           u.Name,
		"correlation_id": correlationID(r),
	})
}

// handlePublic is accessible without auth but still carries correlation ID.
func handlePublic(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"msg":            "public endpoint",
		"correlation_id": correlationID(r),
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()

	// Public route — correlation MW applies but not auth.
	mux.Handle("GET /public", correlationMW(timingMW(
		http.HandlerFunc(handlePublic),
	)))

	// Protected route — requires valid token.
	mux.Handle("GET /me", correlationMW(timingMW(authMW(requireAuth(
		http.HandlerFunc(handleMe),
	)))))

	// Admin route — requires valid token AND admin role.
	mux.Handle("GET /admin", correlationMW(timingMW(authMW(requireAuth(requireRole("admin")(
		http.HandlerFunc(handleAdmin),
	))))))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: mux}).Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}

	do := func(method, path string, headers map[string]string) (int, map[string]any, http.Header) {
		req, _ := http.NewRequest(method, base+path, nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil, nil
		}
		defer resp.Body.Close()
		var body map[string]any
		json.NewDecoder(resp.Body).Decode(&body)
		return resp.StatusCode, body, resp.Header
	}

	check := func(label string, code, want int) {
		mark := "✓"
		if code != want {
			mark = "✗"
		}
		fmt.Printf("  %s %-50s %d\n", mark, label, code)
	}

	fmt.Printf("=== Context Values Middleware — %s ===\n\n", base)

	fmt.Println("--- Correlation ID propagation ---")
	code, body, h := do("GET", "/public", nil)
	check("GET /public → 200", code, 200)
	fmt.Printf("    response body   correlation_id = %s\n", body["correlation_id"])
	fmt.Printf("    response header X-Correlation-ID = %s\n", h.Get("X-Correlation-ID"))

	// Client-supplied correlation ID is echoed back.
	code, body, h = do("GET", "/public", map[string]string{"X-Correlation-ID": "my-trace-abc"})
	check("GET /public (client-supplied ID) → 200", code, 200)
	fmt.Printf("    echoed correlation_id = %v\n", body["correlation_id"])
	fmt.Printf("    response header       = %s\n", h.Get("X-Correlation-ID"))

	fmt.Println()
	fmt.Println("--- Authentication via context ---")
	code, body, _ = do("GET", "/me", nil)
	check("GET /me (no token) → 401", code, 401)
	fmt.Printf("    error: %v\n", body["error"])

	code, body, _ = do("GET", "/me", map[string]string{"Authorization": "Bearer tok-bob"})
	check("GET /me (tok-bob) → 200", code, 200)
	if u, ok := body["user"].(map[string]any); ok {
		fmt.Printf("    user: id=%v name=%v role=%v\n", u["id"], u["name"], u["role"])
	}

	fmt.Println()
	fmt.Println("--- Role-based access control via context ---")
	code, body, _ = do("GET", "/admin", map[string]string{"Authorization": "Bearer tok-bob"})
	check("GET /admin (bob=reader) → 403", code, 403)
	fmt.Printf("    error: %v\n", body["error"])

	code, body, _ = do("GET", "/admin", map[string]string{"Authorization": "Bearer tok-alice"})
	check("GET /admin (alice=admin) → 200", code, 200)
	fmt.Printf("    msg: %v\n", body["msg"])

	fmt.Println()
	fmt.Println("--- Latency from context timing middleware ---")
	code, body, _ = do("GET", "/me", map[string]string{"Authorization": "Bearer tok-alice"})
	check("GET /me → 200", code, 200)
	fmt.Printf("    latency from context: %v µs\n", body["latency_us"])

	fmt.Println()
	fmt.Println("--- Context key collision prevention ---")
	fmt.Println("  Using typed ctxKey avoids string collisions across packages.")
	fmt.Println("  ctxKey(0) ≠ \"correlationID\" ≠ any other package's key.")
}
