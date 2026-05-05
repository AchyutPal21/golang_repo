// FILE: book/part7_capstone_projects/capstone_a_url_shortener/main.go
// CAPSTONE A — URL Shortener
// Self-contained simulation: URL store, Base62 encoding, Redis-like cache,
// rate limiter, click tracker, and graceful shutdown — no external deps.
//
// Run:
//   go run ./book/part7_capstone_projects/capstone_a_url_shortener

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// BASE62 ENCODER
// ─────────────────────────────────────────────────────────────────────────────

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func encodeBase62(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = base62Chars[n%62]
		n /= 62
	}
	return string(buf[i:])
}

// ─────────────────────────────────────────────────────────────────────────────
// URL STORE INTERFACE + IN-MEMORY IMPLEMENTATION
// ─────────────────────────────────────────────────────────────────────────────

type URLEntry struct {
	Code      string
	LongURL   string
	OwnerID   string
	CreatedAt time.Time
	Clicks    int64
}

type URLStore interface {
	Save(ownerID, longURL string) (URLEntry, error)
	Get(code string) (URLEntry, bool)
	Delete(code, ownerID string) error
	IncrClick(code string)
	Stats(code string) (URLEntry, bool)
}

type memoryStore struct {
	mu      sync.RWMutex
	entries map[string]URLEntry
	counter atomic.Uint64
}

func newMemoryStore() *memoryStore { return &memoryStore{entries: map[string]URLEntry{}} }

func (s *memoryStore) Save(ownerID, longURL string) (URLEntry, error) {
	id := s.counter.Add(1)
	code := encodeBase62(id)
	e := URLEntry{Code: code, LongURL: longURL, OwnerID: ownerID, CreatedAt: time.Now()}
	s.mu.Lock()
	s.entries[code] = e
	s.mu.Unlock()
	return e, nil
}

func (s *memoryStore) Get(code string) (URLEntry, bool) {
	s.mu.RLock()
	e, ok := s.entries[code]
	s.mu.RUnlock()
	return e, ok
}

func (s *memoryStore) Delete(code, ownerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[code]
	if !ok {
		return errors.New("not found")
	}
	if e.OwnerID != ownerID {
		return errors.New("forbidden")
	}
	delete(s.entries, code)
	return nil
}

func (s *memoryStore) IncrClick(code string) {
	s.mu.Lock()
	if e, ok := s.entries[code]; ok {
		e.Clicks++
		s.entries[code] = e
	}
	s.mu.Unlock()
}

func (s *memoryStore) Stats(code string) (URLEntry, bool) { return s.Get(code) }

// ─────────────────────────────────────────────────────────────────────────────
// CACHE LAYER (read-through, simulates Redis)
// ─────────────────────────────────────────────────────────────────────────────

type cacheEntry struct {
	entry     URLEntry
	expiresAt time.Time
}

type cachedStore struct {
	mu      sync.RWMutex
	cache   map[string]cacheEntry
	ttl     time.Duration
	hits    atomic.Int64
	misses  atomic.Int64
	backing URLStore
}

func newCachedStore(backing URLStore, ttl time.Duration) *cachedStore {
	return &cachedStore{cache: map[string]cacheEntry{}, ttl: ttl, backing: backing}
}

func (c *cachedStore) Save(ownerID, longURL string) (URLEntry, error) {
	e, err := c.backing.Save(ownerID, longURL)
	if err == nil {
		c.mu.Lock()
		c.cache[e.Code] = cacheEntry{entry: e, expiresAt: time.Now().Add(c.ttl)}
		c.mu.Unlock()
	}
	return e, err
}

func (c *cachedStore) Get(code string) (URLEntry, bool) {
	c.mu.RLock()
	if ce, ok := c.cache[code]; ok && time.Now().Before(ce.expiresAt) {
		c.mu.RUnlock()
		c.hits.Add(1)
		return ce.entry, true
	}
	c.mu.RUnlock()
	c.misses.Add(1)
	e, ok := c.backing.Get(code)
	if ok {
		c.mu.Lock()
		c.cache[code] = cacheEntry{entry: e, expiresAt: time.Now().Add(c.ttl)}
		c.mu.Unlock()
	}
	return e, ok
}

func (c *cachedStore) Delete(code, ownerID string) error {
	err := c.backing.Delete(code, ownerID)
	if err == nil {
		c.mu.Lock()
		delete(c.cache, code)
		c.mu.Unlock()
	}
	return err
}

func (c *cachedStore) IncrClick(code string) { c.backing.IncrClick(code) }
func (c *cachedStore) Stats(code string) (URLEntry, bool) {
	return c.backing.Stats(code)
}

func (c *cachedStore) CacheStats() string {
	return fmt.Sprintf("hits=%d misses=%d ratio=%.0f%%",
		c.hits.Load(), c.misses.Load(),
		100*float64(c.hits.Load())/float64(max64(c.hits.Load()+c.misses.Load(), 1)))
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// ─────────────────────────────────────────────────────────────────────────────
// RATE LIMITER (token bucket per key)
// ─────────────────────────────────────────────────────────────────────────────

type tokenBucket struct {
	tokens   float64
	capacity float64
	rate     float64 // tokens per second
	lastFill time.Time
	mu       sync.Mutex
}

func newTokenBucket(capacity, ratePerSec float64) *tokenBucket {
	return &tokenBucket{tokens: capacity, capacity: capacity, rate: ratePerSec, lastFill: time.Now()}
}

func (tb *tokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(tb.lastFill).Seconds()
	tb.tokens = min64f(tb.capacity, tb.tokens+elapsed*tb.rate)
	tb.lastFill = now
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

func min64f(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	cap     float64
	rate    float64
}

func newRateLimiter(cap, ratePerSec float64) *rateLimiter {
	return &rateLimiter{buckets: map[string]*tokenBucket{}, cap: cap, rate: ratePerSec}
}

func (rl *rateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	b, ok := rl.buckets[key]
	if !ok {
		b = newTokenBucket(rl.cap, rl.rate)
		rl.buckets[key] = b
	}
	rl.mu.Unlock()
	return b.Allow()
}

// ─────────────────────────────────────────────────────────────────────────────
// CLICK TRACKER (async batch writer)
// ─────────────────────────────────────────────────────────────────────────────

type clickTracker struct {
	ch    chan string
	store URLStore
	wg    sync.WaitGroup
}

func newClickTracker(store URLStore, bufSize int) *clickTracker {
	ct := &clickTracker{ch: make(chan string, bufSize), store: store}
	ct.wg.Add(1)
	go ct.run()
	return ct
}

func (ct *clickTracker) run() {
	defer ct.wg.Done()
	for code := range ct.ch {
		ct.store.IncrClick(code)
	}
}

func (ct *clickTracker) Track(code string) {
	select {
	case ct.ch <- code:
	default: // drop if buffer full — acceptable under extreme load
	}
}

func (ct *clickTracker) Shutdown() {
	close(ct.ch)
	ct.wg.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type URLShortener struct {
	store   URLStore
	rl      *rateLimiter
	tracker *clickTracker
}

func NewURLShortener(store URLStore) *URLShortener {
	return &URLShortener{
		store:   store,
		rl:      newRateLimiter(10, 5), // 10 burst, 5/s refill
		tracker: newClickTracker(store, 512),
	}
}

func (s *URLShortener) Shorten(ownerID, longURL string) (string, error) {
	if !s.rl.Allow(ownerID) {
		return "", errors.New("rate limited")
	}
	if !strings.HasPrefix(longURL, "http://") && !strings.HasPrefix(longURL, "https://") {
		return "", errors.New("invalid URL: must start with http:// or https://")
	}
	e, err := s.store.Save(ownerID, longURL)
	if err != nil {
		return "", err
	}
	return e.Code, nil
}

func (s *URLShortener) Redirect(code string) (string, error) {
	e, ok := s.store.Get(code)
	if !ok {
		return "", errors.New("not found")
	}
	s.tracker.Track(code)
	return e.LongURL, nil
}

func (s *URLShortener) Stats(code string) (URLEntry, error) {
	e, ok := s.store.Stats(code)
	if !ok {
		return URLEntry{}, errors.New("not found")
	}
	return e, nil
}

func (s *URLShortener) Delete(code, ownerID string) error {
	return s.store.Delete(code, ownerID)
}

func (s *URLShortener) Shutdown(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		s.tracker.Shutdown()
		close(done)
	}()
	select {
	case <-done:
		fmt.Println("  [shutdown] click tracker drained cleanly")
	case <-ctx.Done():
		fmt.Println("  [shutdown] click tracker drain timed out")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN — simulate the full lifecycle
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Capstone A: URL Shortener ===")
	fmt.Println()

	backing := newMemoryStore()
	store := newCachedStore(backing, 1*time.Hour)
	svc := NewURLShortener(store)

	// ── SHORTEN URLs ──────────────────────────────────────────────────────────
	fmt.Println("--- Shortening URLs ---")
	urls := []struct{ owner, url string }{
		{"user-1", "https://example.com/very/long/path/to/some/article?ref=newsletter"},
		{"user-1", "https://github.com/golang/go/issues/12345"},
		{"user-2", "https://pkg.go.dev/net/http"},
		{"user-2", "https://go.dev/blog/context"},
	}
	codes := make([]string, 0, len(urls))
	for _, u := range urls {
		code, err := svc.Shorten(u.owner, u.url)
		if err != nil {
			fmt.Printf("  ERROR shorten: %v\n", err)
			continue
		}
		codes = append(codes, code)
		fmt.Printf("  %-8s → https://sho.rt/%s\n", u.owner, code)
	}
	fmt.Println()

	// ── REDIRECT (cache miss first, then cache hit) ────────────────────────────
	fmt.Println("--- Redirect lookups ---")
	for _, code := range codes {
		target, err := svc.Redirect(code)
		if err != nil {
			fmt.Printf("  [404] %s\n", code)
			continue
		}
		fmt.Printf("  /%-6s → 301 %s\n", code, target)
	}
	// hit same codes again — these should be cache hits
	for _, code := range codes {
		svc.Redirect(code) //nolint:errcheck
	}
	fmt.Printf("  Cache: %s\n", store.CacheStats())
	fmt.Println()

	// ── INVALID URL ───────────────────────────────────────────────────────────
	fmt.Println("--- Validation ---")
	_, err := svc.Shorten("user-1", "not-a-url")
	fmt.Printf("  Invalid URL: %v\n", err)

	// ── RATE LIMITING ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Rate limiter (burst=10, 5/s) ---")
	limited := 0
	for i := 0; i < 15; i++ {
		_, err := svc.Shorten("user-3", "https://example.com/page")
		if err != nil && err.Error() == "rate limited" {
			limited++
		}
	}
	fmt.Printf("  15 rapid requests → %d rate-limited\n", limited)
	fmt.Println()

	// ── STATS ─────────────────────────────────────────────────────────────────
	fmt.Println("--- Click stats ---")
	time.Sleep(10 * time.Millisecond) // let async tracker flush
	for _, code := range codes[:2] {
		e, err := svc.Stats(code)
		if err != nil {
			continue
		}
		fmt.Printf("  code=%-6s clicks=%d url=%s\n", e.Code, e.Clicks, truncate(e.LongURL, 50))
	}
	fmt.Println()

	// ── DELETE ────────────────────────────────────────────────────────────────
	fmt.Println("--- Delete ---")
	err = svc.Delete(codes[0], "user-1")
	fmt.Printf("  Delete by owner:     %v\n", err)
	err = svc.Delete(codes[1], "wrong-owner")
	fmt.Printf("  Delete wrong owner:  %v\n", err)
	_, err = svc.Redirect(codes[0])
	fmt.Printf("  Redirect deleted:    %v\n", err)
	fmt.Println()

	// ── GRACEFUL SHUTDOWN ─────────────────────────────────────────────────────
	fmt.Println("--- Graceful shutdown ---")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	svc.Shutdown(ctx)
	fmt.Println()

	// ── BASE62 DEMO ───────────────────────────────────────────────────────────
	fmt.Println("--- Base62 code distribution (IDs 1–10) ---")
	for i := uint64(1); i <= 10; i++ {
		fmt.Printf("  ID %-4d → %s\n", i, encodeBase62(i))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
