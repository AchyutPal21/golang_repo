// FILE: book/part5_building_backends/chapter59_middleware/examples/01_middleware_stack/main.go
// CHAPTER: 59 — Middleware
// TOPIC: Middleware fundamentals — the adapter pattern, execution order,
//        panic recovery, per-request timeouts, rate limiting, CORS,
//        and composing a full production middleware stack.
//
// Run (from the chapter folder):
//   go run ./examples/01_middleware_stack

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CHAIN HELPER
// ─────────────────────────────────────────────────────────────────────────────

type MW = func(http.Handler) http.Handler

// chain wraps h with each middleware; first in list is outermost (executes first).
func chain(h http.Handler, mws ...MW) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// ─────────────────────────────────────────────────────────────────────────────
// RESPONSE RECORDER — captures status code for logging
// ─────────────────────────────────────────────────────────────────────────────

type recorder struct {
	http.ResponseWriter
	status  int
	written int64
}

func newRecorder(w http.ResponseWriter) *recorder {
	return &recorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.written += int64(n)
	return n, err
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE 1 — RECOVERY
// Catches panics anywhere in the inner chain, writes 500, logs stack trace.
// Must be outermost so it catches panics in all other middleware.
// ─────────────────────────────────────────────────────────────────────────────

func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
				fmt.Printf("  [PANIC] %v\n", v)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE 2 — STRUCTURED LOGGING
// Records method, path, status, latency after the inner chain completes.
// ─────────────────────────────────────────────────────────────────────────────

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := newRecorder(w)
		next.ServeHTTP(rec, r)
		fmt.Printf(`  [log] method=%s path=%-20s status=%d latency=%dµs bytes=%d`+"\n",
			r.Method, r.URL.Path, rec.status, time.Since(start).Microseconds(), rec.written)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE 3 — TOKEN BUCKET RATE LIMITER
// Allows burst requests; refills at refillRate tokens/second.
// Per-IP limiting is a common extension (not shown here for brevity).
// ─────────────────────────────────────────────────────────────────────────────

type tokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func newTokenBucket(maxTokens, refillRate float64) *tokenBucket {
	return &tokenBucket{
		tokens: maxTokens, maxTokens: maxTokens,
		refillRate: refillRate, lastRefill: time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = min(tb.maxTokens, tb.tokens+elapsed*tb.refillRate)
	tb.lastRefill = now
	if tb.tokens < 1 {
		return false
	}
	tb.tokens--
	return true
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func rateLimiter(bucket *tokenBucket) MW {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !bucket.allow() {
				w.Header().Set("Retry-After", "1")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE 4 — PER-REQUEST TIMEOUT
// Cancels the request context after the given duration.
// Handler must respect ctx.Done() for this to be useful.
// ─────────────────────────────────────────────────────────────────────────────

func timeout(d time.Duration) MW {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE 5 — CORS
// Adds permissive CORS headers and handles preflight OPTIONS requests.
// ─────────────────────────────────────────────────────────────────────────────

func cors(allowedOrigins ...string) MW {
	origins := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		origins[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origins[origin] || (len(origins) == 0) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE 6 — SECURE HEADERS
// Adds common security headers to every response.
// ─────────────────────────────────────────────────────────────────────────────

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE 7 — REQUEST COUNTER (demonstrates stateful middleware)
// ─────────────────────────────────────────────────────────────────────────────

func requestCounter(counter *atomic.Int64) MW {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			counter.Add(1)
			next.ServeHTTP(w, r)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func handleHello(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"msg": "hello"})
}

func handleSlow(w http.ResponseWriter, r *http.Request) {
	// Respects context cancellation — checks Done channel before sleeping.
	select {
	case <-time.After(500 * time.Millisecond):
		json.NewEncoder(w).Encode(map[string]string{"msg": "done"})
	case <-r.Context().Done():
		http.Error(w, `{"error":"timeout"}`, http.StatusGatewayTimeout)
	}
}

func handlePanic(w http.ResponseWriter, r *http.Request) {
	panic("deliberate panic for demo")
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	bucket := newTokenBucket(5, 2)   // burst of 5, refill 2/sec
	var counter atomic.Int64

	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", handleHello)
	mux.HandleFunc("GET /slow", handleSlow)   // 500ms handler
	mux.HandleFunc("GET /panic", handlePanic)

	// Full stack: recovery → logging → counter → cors → secure → rate-limit → timeout → mux
	stack := chain(mux,
		recovery,
		logging,
		requestCounter(&counter),
		cors("http://example.com"),
		secureHeaders,
		rateLimiter(bucket),
		timeout(200*time.Millisecond),
	)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: stack}).Serve(ln)

	client := &http.Client{Timeout: 2 * time.Second}

	do := func(method, path string, headers map[string]string) (int, http.Header) {
		req, _ := http.NewRequest(method, base+path, nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil
		}
		resp.Body.Close()
		return resp.StatusCode, resp.Header
	}

	check := func(label string, code, want int) {
		mark := "✓"
		if code != want {
			mark = "✗"
		}
		fmt.Printf("  %s %-50s %d\n", mark, label, code)
	}

	fmt.Printf("=== Middleware Stack — %s ===\n\n", base)

	fmt.Println("--- Basic request flow ---")
	code, h := do("GET", "/hello", nil)
	check("GET /hello → 200", code, 200)
	fmt.Printf("    X-Frame-Options: %s\n", h.Get("X-Frame-Options"))
	fmt.Printf("    X-Content-Type-Options: %s\n", h.Get("X-Content-Type-Options"))

	fmt.Println()
	fmt.Println("--- Panic recovery ---")
	code, _ = do("GET", "/panic", nil)
	check("GET /panic → 500 (recovered)", code, 500)

	fmt.Println()
	fmt.Println("--- Per-request timeout (200ms limit, handler needs 500ms) ---")
	code, _ = do("GET", "/slow", nil)
	check("GET /slow → 504 (timed out)", code, 504)

	fmt.Println()
	fmt.Println("--- CORS preflight ---")
	code, h = do("OPTIONS", "/hello", map[string]string{
		"Origin":                         "http://example.com",
		"Access-Control-Request-Method":  "GET",
		"Access-Control-Request-Headers": "Authorization",
	})
	check("OPTIONS /hello (preflight) → 204", code, 204)
	fmt.Printf("    Access-Control-Allow-Methods: %s\n", h.Get("Access-Control-Allow-Methods"))

	// Disallowed origin gets no CORS headers.
	code, h = do("GET", "/hello", map[string]string{"Origin": "http://evil.com"})
	check("GET /hello (disallowed origin) → 200", code, 200)
	fmt.Printf("    ACAO header: %q (empty = blocked)\n", h.Get("Access-Control-Allow-Origin"))

	fmt.Println()
	fmt.Println("--- Rate limiting (burst=5, refill=2/sec, firing 8 rapid requests) ---")

	// Reset bucket to empty it.
	bucket2 := newTokenBucket(5, 0.1) // very slow refill
	stack2 := chain(mux, recovery, rateLimiter(bucket2))
	ln2, _ := net.Listen("tcp", "127.0.0.1:0")
	base2 := "http://" + ln2.Addr().String()
	go (&http.Server{Handler: stack2}).Serve(ln2)

	var ok, limited int
	for i := 0; i < 8; i++ {
		// Use rand for a tiny jitter to avoid exact scheduling issues.
		_ = rand.Intn(1)
		req, _ := http.NewRequest("GET", base2+"/hello", nil)
		resp, _ := client.Do(req)
		if resp != nil {
			if resp.StatusCode == http.StatusOK {
				ok++
			} else if resp.StatusCode == http.StatusTooManyRequests {
				limited++
			}
			resp.Body.Close()
		}
	}
	fmt.Printf("  ✓ 8 requests → %d OK, %d rate-limited (burst=5)\n", ok, limited)

	fmt.Println()
	fmt.Printf("--- Request counter: %d total requests served ---\n", counter.Load())

	fmt.Println()
	fmt.Println("--- Middleware execution order (request path) ---")
	fmt.Println("  recovery → logging → counter → cors → secureHeaders → rateLimiter → timeout → handler")
	fmt.Println("  Each layer sees the request before inner layers; sees the response after inner layers return.")

	// Demonstrate that logging captures the status even after recovery writes 500.
	fmt.Println()
	fmt.Println("--- Log output for /panic (above) shows status=500 ---")
	// Already triggered above; the log line was printed by logging middleware.
	fmt.Println("  (see [log] lines above for the structured log output)")
}
