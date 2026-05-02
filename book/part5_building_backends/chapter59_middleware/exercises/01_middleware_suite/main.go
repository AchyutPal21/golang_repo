// FILE: book/part5_building_backends/chapter59_middleware/exercises/01_middleware_suite/main.go
// CHAPTER: 59 — Middleware
// EXERCISE: Compose a reusable middleware suite with:
//   - Correlation ID middleware (generate or propagate X-Correlation-ID)
//   - Structured JSON request logger (method, path, status, latency, correlation_id)
//   - Panic recovery (500 + correlation_id in error body)
//   - Per-IP rate limiter (5 req/sec per IP using token bucket)
//   - Bearer token authentication (stores User in context)
//   - Role gate middleware factory (requires a specific role)
//   - An API that exercises all middleware layers:
//       GET /health       — no auth
//       GET /api/profile  — auth required
//       GET /api/admin    — admin role required
//       GET /api/panic    — tests recovery + correlation in error response
//
// Run (from the chapter folder):
//   go run ./exercises/01_middleware_suite

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT KEYS
// ─────────────────────────────────────────────────────────────────────────────

type ctxKey int

const (
	keyCorrID ctxKey = iota
	keyUser
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func withCorrID(r *http.Request, id string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), keyCorrID, id))
}

func getCorrID(r *http.Request) string {
	v, _ := r.Context().Value(keyCorrID).(string)
	return v
}

func withUser(r *http.Request, u *User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), keyUser, u))
}

func getUser(r *http.Request) (*User, bool) {
	u, ok := r.Context().Value(keyUser).(*User)
	return u, ok && u != nil
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

type statusRec struct {
	http.ResponseWriter
	status int
}

func (r *statusRec) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

type MW = func(http.Handler) http.Handler

func chain(h http.Handler, mws ...MW) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE: CORRELATION ID
// ─────────────────────────────────────────────────────────────────────────────

func corrIDMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Correlation-ID")
		if id == "" {
			id = fmt.Sprintf("cid-%x", rand.Int63())
		}
		w.Header().Set("X-Correlation-ID", id)
		next.ServeHTTP(w, withCorrID(r, id))
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE: STRUCTURED JSON LOGGER
// ─────────────────────────────────────────────────────────────────────────────

func jsonLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRec{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		fmt.Printf(`{"method":%q,"path":%q,"status":%d,"ms":%d,"cid":%q}`+"\n",
			r.Method, r.URL.Path, rec.status,
			time.Since(start).Milliseconds(), getCorrID(r))
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE: PANIC RECOVERY
// ─────────────────────────────────────────────────────────────────────────────

func panicRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error":          "internal server error",
					"correlation_id": getCorrID(r),
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE: PER-IP TOKEN BUCKET RATE LIMITER
// ─────────────────────────────────────────────────────────────────────────────

type ipBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

type ipRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*ipBucket
	max     float64
	rate    float64
}

func newIPRateLimiter(maxTokens, refillPerSec float64) *ipRateLimiter {
	return &ipRateLimiter{
		buckets: make(map[string]*ipBucket),
		max:     maxTokens,
		rate:    refillPerSec,
	}
}

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[ip]
	if !ok {
		b = &ipBucket{tokens: l.max, maxTokens: l.max, refillRate: l.rate, lastRefill: time.Now()}
		l.buckets[ip] = b
	}
	elapsed := time.Since(b.lastRefill).Seconds()
	b.tokens = min(b.maxTokens, b.tokens+elapsed*b.refillRate)
	b.lastRefill = time.Now()
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func perIPRateLimit(limiter *ipRateLimiter) MW {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if idx := strings.LastIndex(ip, ":"); idx >= 0 {
				ip = ip[:idx]
			}
			if !limiter.allow(ip) {
				w.Header().Set("Retry-After", "1")
				writeJSON(w, http.StatusTooManyRequests, map[string]string{
					"error":          "rate limit exceeded",
					"correlation_id": getCorrID(r),
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE: BEARER AUTH
// ─────────────────────────────────────────────────────────────────────────────

var users = map[string]*User{
	"tok-admin": {ID: 1, Name: "Admin", Role: "admin"},
	"tok-user":  {ID: 2, Name: "User", Role: "user"},
}

func bearerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if tok, ok := strings.CutPrefix(auth, "Bearer "); ok {
			if u, found := users[tok]; found {
				r = withUser(r, u)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE: REQUIRE AUTH
// ─────────────────────────────────────────────────────────────────────────────

func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getUser(r); !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "authentication required", "correlation_id": getCorrID(r),
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE: ROLE GATE
// ─────────────────────────────────────────────────────────────────────────────

func requireRole(role string) MW {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, _ := getUser(r)
			if u.Role != role {
				writeJSON(w, http.StatusForbidden, map[string]string{
					"error": "forbidden: requires " + role, "correlation_id": getCorrID(r),
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleProfile(w http.ResponseWriter, r *http.Request) {
	u, _ := getUser(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"user": u, "correlation_id": getCorrID(r),
	})
}

func handleAdminPanel(w http.ResponseWriter, r *http.Request) {
	u, _ := getUser(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"panel": "admin", "user": u.Name, "correlation_id": getCorrID(r),
	})
}

func handlePanicRoute(w http.ResponseWriter, r *http.Request) {
	panic("demo panic")
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	limiter := newIPRateLimiter(20, 10)

	// Global stack applied to every request.
	// corrIDMW is outermost so the correlation ID is in context before recovery fires.
	globalMWs := func(h http.Handler) http.Handler {
		return chain(h,
			corrIDMW,
			panicRecovery,
			jsonLogger,
			perIPRateLimit(limiter),
		)
	}

	mux := http.NewServeMux()

	// Public.
	mux.Handle("GET /health", globalMWs(http.HandlerFunc(handleHealth)))

	// Auth-required routes.
	authed := func(h http.Handler) http.Handler {
		return globalMWs(chain(h, bearerAuth, requireAuth))
	}
	mux.Handle("GET /api/profile", authed(http.HandlerFunc(handleProfile)))

	// Admin-only routes.
	admin := func(h http.Handler) http.Handler {
		return globalMWs(chain(h, bearerAuth, requireAuth, requireRole("admin")))
	}
	mux.Handle("GET /api/admin", admin(http.HandlerFunc(handleAdminPanel)))

	// Panic test — recovery should kick in.
	mux.Handle("GET /api/panic", globalMWs(http.HandlerFunc(handlePanicRoute)))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: mux}).Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}

	do := func(method, path string, hdrs map[string]string) (int, map[string]any, http.Header) {
		req, _ := http.NewRequest(method, base+path, nil)
		for k, v := range hdrs {
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
		fmt.Printf("  %s %-54s %d\n", mark, label, code)
	}

	adminH := map[string]string{"Authorization": "Bearer tok-admin"}
	userH := map[string]string{"Authorization": "Bearer tok-user"}

	fmt.Printf("=== Middleware Suite — %s ===\n\n", base)

	fmt.Println("--- Public endpoint ---")
	code, _, h := do("GET", "/health", nil)
	check("GET /health → 200", code, 200)
	fmt.Printf("    X-Correlation-ID: %s\n", h.Get("X-Correlation-ID"))

	fmt.Println()
	fmt.Println("--- Auth required ---")
	code, body, _ := do("GET", "/api/profile", nil)
	check("GET /api/profile (no token) → 401", code, 401)
	fmt.Printf("    error: %v  cid: %v\n", body["error"], body["correlation_id"])

	code, body, _ = do("GET", "/api/profile", userH)
	check("GET /api/profile (user token) → 200", code, 200)
	if u, ok := body["user"].(map[string]any); ok {
		fmt.Printf("    user: %v role: %v\n", u["name"], u["role"])
	}

	fmt.Println()
	fmt.Println("--- Role gate ---")
	code, body, _ = do("GET", "/api/admin", userH)
	check("GET /api/admin (user=reader) → 403", code, 403)
	fmt.Printf("    error: %v\n", body["error"])

	code, body, _ = do("GET", "/api/admin", adminH)
	check("GET /api/admin (admin token) → 200", code, 200)
	fmt.Printf("    panel: %v user: %v\n", body["panel"], body["user"])

	fmt.Println()
	fmt.Println("--- Panic recovery with correlation ID ---")
	code, body, h = do("GET", "/api/panic", map[string]string{"X-Correlation-ID": "trace-xyz"})
	check("GET /api/panic → 500 (recovered)", code, 500)
	fmt.Printf("    error:          %v\n", body["error"])
	fmt.Printf("    correlation_id: %v\n", body["correlation_id"])
	fmt.Printf("    X-Correlation-ID header: %s\n", h.Get("X-Correlation-ID"))

	fmt.Println()
	fmt.Println("--- Per-IP rate limiting (burst=5, firing 7 rapid) ---")
	fastLimiter := newIPRateLimiter(5, 0.1)
	fastMux := http.NewServeMux()
	fastMux.Handle("GET /ping", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"pong": "ok"})
		}),
		corrIDMW,
		perIPRateLimit(fastLimiter),
	))
	ln2, _ := net.Listen("tcp", "127.0.0.1:0")
	base2 := "http://" + ln2.Addr().String()
	go (&http.Server{Handler: fastMux}).Serve(ln2)

	var ok, rejected int
	for i := 0; i < 7; i++ {
		req, _ := http.NewRequest("GET", base2+"/ping", nil)
		resp, _ := client.Do(req)
		if resp != nil {
			if resp.StatusCode == 200 {
				ok++
			} else if resp.StatusCode == 429 {
				rejected++
			}
			resp.Body.Close()
		}
	}
	fmt.Printf("  ✓ 7 requests → %d OK, %d rate-limited (burst=5)\n", ok, rejected)

	fmt.Println()
	fmt.Println("--- Log output ---")
	fmt.Println("  (see JSON log lines above for each request)")
}
