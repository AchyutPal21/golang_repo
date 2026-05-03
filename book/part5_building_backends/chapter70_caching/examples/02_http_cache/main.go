// Chapter 70 — HTTP-level caching: ResponseCache middleware, Cache-Control,
// ETag / If-None-Match (304 Not Modified), Vary header.
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── Types ────────────────────────────────────────────────────────────────────

// CachedResponse stores a serialised HTTP response body with metadata.
type CachedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	ETag       string
	CachedAt   time.Time
	MaxAge     int // seconds; 0 means no max-age directive
}

// isStale reports whether the cached entry has exceeded its max-age.
func (c *CachedResponse) isStale() bool {
	if c.MaxAge <= 0 {
		return false // no expiry directive — treat as fresh
	}
	return time.Since(c.CachedAt) > time.Duration(c.MaxAge)*time.Second
}

// ─── ResponseCache ────────────────────────────────────────────────────────────

// ResponseCache is an HTTP middleware that caches GET responses.
// Cache key = path + (optional) Accept header when Vary: Accept is present.
type ResponseCache struct {
	mu      sync.RWMutex
	entries map[string]*CachedResponse
	hits    int
	misses  int
}

func NewResponseCache() *ResponseCache {
	return &ResponseCache{entries: make(map[string]*CachedResponse)}
}

// cacheKey builds the lookup key from path and relevant Vary headers.
func (rc *ResponseCache) cacheKey(r *http.Request) string {
	key := r.URL.Path
	// Simple Vary: Accept support — append Accept to key.
	if accept := r.Header.Get("Accept"); accept != "" {
		key += "|" + accept
	}
	return key
}

// recordingWriter captures the response body and headers before forwarding.
type recordingWriter struct {
	http.ResponseWriter
	buf    bytes.Buffer
	status int
	hdr    http.Header
}

func newRecordingWriter(w http.ResponseWriter) *recordingWriter {
	return &recordingWriter{ResponseWriter: w, status: http.StatusOK, hdr: w.Header()}
}

func (rw *recordingWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *recordingWriter) Write(b []byte) (int, error) {
	rw.buf.Write(b)
	return rw.ResponseWriter.Write(b)
}

// Middleware wraps the next handler with HTTP caching logic.
func (rc *ResponseCache) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only cache GET/HEAD requests.
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		key := rc.cacheKey(r)

		// ── Check request Cache-Control directives ────────────────────────
		reqCC := r.Header.Get("Cache-Control")
		noCache := strings.Contains(reqCC, "no-cache")

		rc.mu.RLock()
		cached, found := rc.entries[key]
		rc.mu.RUnlock()

		if found && !noCache && !cached.isStale() {
			// ── ETag / If-None-Match ──────────────────────────────────────
			if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
				if ifNoneMatch == cached.ETag {
					rc.mu.Lock()
					rc.hits++
					rc.mu.Unlock()
					w.Header().Set("ETag", cached.ETag)
					w.Header().Set("X-Cache", "HIT (304)")
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
			// ── Serve from cache ──────────────────────────────────────────
			rc.mu.Lock()
			rc.hits++
			rc.mu.Unlock()
			for k, vs := range cached.Headers {
				for _, v := range vs {
					w.Header().Set(k, v)
				}
			}
			w.Header().Set("ETag", cached.ETag)
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(cached.StatusCode)
			w.Write(cached.Body)
			return
		}

		// ── Cache miss — pass to real handler ─────────────────────────────
		rc.mu.Lock()
		rc.misses++
		rc.mu.Unlock()

		rw := newRecordingWriter(w)
		next.ServeHTTP(rw, r)

		// ── Parse response Cache-Control ──────────────────────────────────
		respCC := rw.hdr.Get("Cache-Control")
		if strings.Contains(respCC, "no-store") {
			return // do not cache
		}

		maxAge := 0
		for _, part := range strings.Split(respCC, ",") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "max-age=") {
				if n, err := strconv.Atoi(strings.TrimPrefix(part, "max-age=")); err == nil {
					maxAge = n
				}
			}
		}

		body := rw.buf.Bytes()
		etag := fmt.Sprintf(`"%x"`, sha256.Sum256(body))

		rc.mu.Lock()
		rc.entries[key] = &CachedResponse{
			StatusCode: rw.status,
			Headers:    rw.hdr.Clone(),
			Body:       body,
			ETag:       etag,
			CachedAt:   time.Now(),
			MaxAge:     maxAge,
		}
		rc.mu.Unlock()

		w.Header().Set("ETag", etag)
		w.Header().Set("X-Cache", "MISS")
	})
}

// Stats returns hit and miss counts.
func (rc *ResponseCache) Stats() (hits, misses int) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.hits, rc.misses
}

// ─── Application handlers ─────────────────────────────────────────────────────

var requestCount int

func productsHandler(w http.ResponseWriter, r *http.Request) {
	requestCount++
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=60")
	w.Header().Set("Vary", "Accept")
	fmt.Fprintf(w, `{"products":[{"id":1,"name":"Widget"},{"id":2,"name":"Gadget"}],"served_count":%d}`, requestCount)
}

func noCacheHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintf(w, `{"secret":"password123","ts":%d}`, time.Now().UnixNano())
}

func privateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=5")
	fmt.Fprintf(w, `{"user":"alice","balance":100}`)
}

// ─── main — demo with httptest.Server ─────────────────────────────────────────

func main() {
	cache := NewResponseCache()

	mux := http.NewServeMux()
	mux.HandleFunc("/products", productsHandler)
	mux.HandleFunc("/secret", noCacheHandler)
	mux.HandleFunc("/profile", privateHandler)

	handler := cache.Middleware(mux)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := srv.Client()

	// Helper to make a GET and print result.
	get := func(path string, extraHeaders map[string]string) *http.Response {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		for k, v := range extraHeaders {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  request error: %v\n", err)
			return resp
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("  GET %-12s  status=%-3d  X-Cache=%-12s  ETag=%-20s  body=%s\n",
			path, resp.StatusCode, resp.Header.Get("X-Cache"), resp.Header.Get("ETag"), truncate(string(body), 60))
		return resp
	}

	fmt.Println("=== HTTP Cache Demo ===")
	fmt.Println()

	fmt.Println("--- 1. Basic caching (cache-miss, then cache-hit) ---")
	get("/products", nil)
	get("/products", nil)
	hits, misses := cache.Stats()
	fmt.Printf("  stats: hits=%d misses=%d  (handler called %d times)\n", hits, misses, requestCount)

	fmt.Println("\n--- 2. ETag / If-None-Match → 304 Not Modified ---")
	resp := get("/products", nil) // should be HIT, returns ETag
	etag := resp.Header.Get("ETag")
	get("/products", map[string]string{"If-None-Match": etag})

	fmt.Println("\n--- 3. no-store → never cached ---")
	get("/secret", nil)
	get("/secret", nil)
	hits2, misses2 := cache.Stats()
	fmt.Printf("  stats: hits=%d misses=%d\n", hits2, misses2)

	fmt.Println("\n--- 4. Cache-Control: no-cache in request → bypass cache ---")
	get("/profile", nil) // warm cache
	get("/profile", map[string]string{"Cache-Control": "no-cache"}) // bypass
	hits3, misses3 := cache.Stats()
	fmt.Printf("  stats: hits=%d misses=%d\n", hits3, misses3)

	fmt.Println("\n--- 5. Vary: Accept — different Accept headers → separate cache entries ---")
	get("/products", map[string]string{"Accept": "application/json"})
	get("/products", map[string]string{"Accept": "application/xml"}) // different key
	hits4, misses4 := cache.Stats()
	fmt.Printf("  stats: hits=%d misses=%d\n", hits4, misses4)

	fmt.Println("\n--- 6. max-age=5 expiry demo (profile) ---")
	get("/profile", nil)
	fmt.Println("  (simulating TTL expiry by mutating cache entry)")
	// Force stale by backdating the cache entry.
	key := "/profile"
	cache.mu.Lock()
	if e, ok := cache.entries[key]; ok {
		e.CachedAt = time.Now().Add(-10 * time.Second)
	}
	cache.mu.Unlock()
	get("/profile", nil) // stale → re-fetch
	hits5, misses5 := cache.Stats()
	fmt.Printf("  stats: hits=%d misses=%d\n", hits5, misses5)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
