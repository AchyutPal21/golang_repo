// FILE: book/part5_building_backends/chapter78_rate_limiting/examples/01_rate_limiters/main.go
// CHAPTER: 78 — Rate Limiting, Circuit Breakers, Retries
// TOPIC: Token bucket, sliding window, fixed window, and leaky bucket rate limiters.
//
// Run (from the chapter folder):
//   go run ./examples/01_rate_limiters

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// TOKEN BUCKET — allows bursts up to capacity; refills at a fixed rate
// ─────────────────────────────────────────────────────────────────────────────

type TokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	rate     float64 // tokens per second
	lastFill time.Time
}

func NewTokenBucket(capacity, ratePerSec float64) *TokenBucket {
	return &TokenBucket{
		tokens:   capacity,
		capacity: capacity,
		rate:     ratePerSec,
		lastFill: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

func (tb *TokenBucket) AllowN(n float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(tb.lastFill).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastFill = now
	if tb.tokens < n {
		return false
	}
	tb.tokens -= n
	return true
}

func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		if tb.Allow() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SLIDING WINDOW — count requests in the last N seconds using bucketed counters
// ─────────────────────────────────────────────────────────────────────────────

type SlidingWindow struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	bucketDur time.Duration
	buckets   map[int64]int // bucket key → count
}

func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	buckets := int64(10) // split window into 10 sub-buckets
	return &SlidingWindow{
		limit:     limit,
		window:    window,
		bucketDur: window / time.Duration(buckets),
		buckets:   make(map[int64]int),
	}
}

func (sw *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	bucketKey := now.UnixNano() / int64(sw.bucketDur)

	// Remove expired buckets.
	cutoff := now.Add(-sw.window).UnixNano() / int64(sw.bucketDur)
	for k := range sw.buckets {
		if k <= cutoff {
			delete(sw.buckets, k)
		}
	}

	// Count current window.
	var total int
	for _, count := range sw.buckets {
		total += count
	}
	if total >= sw.limit {
		return false
	}
	sw.buckets[bucketKey]++
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// FIXED WINDOW — simple counter per time window; resets at window boundary
// ─────────────────────────────────────────────────────────────────────────────

type FixedWindow struct {
	mu          sync.Mutex
	limit       int
	window      time.Duration
	count       int
	windowStart time.Time
}

func NewFixedWindow(limit int, window time.Duration) *FixedWindow {
	return &FixedWindow{limit: limit, window: window, windowStart: time.Now()}
}

func (fw *FixedWindow) Allow() bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	now := time.Now()
	if now.Sub(fw.windowStart) > fw.window {
		fw.count = 0
		fw.windowStart = now
	}
	if fw.count >= fw.limit {
		return false
	}
	fw.count++
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// LEAKY BUCKET — smooths out bursts; processes at a constant rate
// ─────────────────────────────────────────────────────────────────────────────

type LeakyBucket struct {
	mu       sync.Mutex
	queue    []time.Time // queued request times
	capacity int
	interval time.Duration // time between processing requests
	lastDrop time.Time
}

func NewLeakyBucket(capacity int, ratePerSec float64) *LeakyBucket {
	return &LeakyBucket{
		capacity: capacity,
		interval: time.Duration(float64(time.Second) / ratePerSec),
		lastDrop: time.Now(),
	}
}

func (lb *LeakyBucket) Allow() bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	now := time.Now()
	// Drain requests that can be processed since last check.
	elapsed := now.Sub(lb.lastDrop)
	drain := int(elapsed / lb.interval)
	if drain > 0 {
		lb.lastDrop = lb.lastDrop.Add(time.Duration(drain) * lb.interval)
		if drain > len(lb.queue) {
			lb.queue = nil
		} else {
			lb.queue = lb.queue[drain:]
		}
	}
	if len(lb.queue) >= lb.capacity {
		return false
	}
	lb.queue = append(lb.queue, now)
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// PER-KEY RATE LIMITER
// ─────────────────────────────────────────────────────────────────────────────

type PerKeyLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*TokenBucket
	capacity float64
	rate     float64
}

func NewPerKeyLimiter(capacity, ratePerSec float64) *PerKeyLimiter {
	return &PerKeyLimiter{
		limiters: make(map[string]*TokenBucket),
		capacity: capacity,
		rate:     ratePerSec,
	}
}

func (p *PerKeyLimiter) Allow(key string) bool {
	p.mu.RLock()
	lb, ok := p.limiters[key]
	p.mu.RUnlock()
	if !ok {
		p.mu.Lock()
		lb = NewTokenBucket(p.capacity, p.rate)
		p.limiters[key] = lb
		p.mu.Unlock()
	}
	return lb.Allow()
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Rate Limiters ===")
	fmt.Println()

	// ── TOKEN BUCKET ──────────────────────────────────────────────────────────
	fmt.Println("--- Token bucket (capacity=5, rate=10/s) ---")
	tb := NewTokenBucket(5, 10)

	var allowed, denied int
	for i := 0; i < 8; i++ {
		if tb.Allow() {
			allowed++
			fmt.Printf("  request %d: allowed\n", i+1)
		} else {
			denied++
			fmt.Printf("  request %d: denied\n", i+1)
		}
	}
	fmt.Printf("  burst result: allowed=%d denied=%d\n", allowed, denied)

	// Refill and try again.
	time.Sleep(300 * time.Millisecond) // refill ~3 tokens
	allowed2 := 0
	for i := 0; i < 4; i++ {
		if tb.Allow() {
			allowed2++
		}
	}
	fmt.Printf("  after 300ms refill: allowed=%d/4\n", allowed2)

	// Wait-based usage.
	fmt.Println()
	fmt.Println("--- Token bucket Wait() ---")
	tb2 := NewTokenBucket(2, 20) // burst 2, refill 20/s
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	var waitAllowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if err := tb2.Wait(ctx); err == nil {
				waitAllowed.Add(1)
				fmt.Printf("  goroutine %d: got token\n", n)
			}
		}(i)
	}
	wg.Wait()
	fmt.Printf("  wait result: %d/5 goroutines passed\n", waitAllowed.Load())

	// ── SLIDING WINDOW ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Sliding window (5 req per 100ms) ---")
	sw := NewSlidingWindow(5, 100*time.Millisecond)
	swAllowed, swDenied := 0, 0
	for i := 0; i < 8; i++ {
		if sw.Allow() {
			swAllowed++
		} else {
			swDenied++
		}
	}
	fmt.Printf("  burst: allowed=%d denied=%d\n", swAllowed, swDenied)
	time.Sleep(110 * time.Millisecond)
	swAllowed2 := 0
	for i := 0; i < 5; i++ {
		if sw.Allow() {
			swAllowed2++
		}
	}
	fmt.Printf("  after window: allowed=%d/5\n", swAllowed2)

	// ── FIXED WINDOW ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Fixed window (3 req per 50ms) ---")
	fw := NewFixedWindow(3, 50*time.Millisecond)
	fwAllowed, fwDenied := 0, 0
	for i := 0; i < 5; i++ {
		if fw.Allow() {
			fwAllowed++
		} else {
			fwDenied++
		}
	}
	fmt.Printf("  window 1: allowed=%d denied=%d\n", fwAllowed, fwDenied)
	time.Sleep(55 * time.Millisecond)
	fwAllowed2 := 0
	for i := 0; i < 3; i++ {
		if fw.Allow() {
			fwAllowed2++
		}
	}
	fmt.Printf("  window 2: allowed=%d/3\n", fwAllowed2)

	// ── LEAKY BUCKET ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Leaky bucket (capacity=3, rate=10/s) ---")
	leaky := NewLeakyBucket(3, 10)
	lkAllowed, lkDenied := 0, 0
	for i := 0; i < 6; i++ {
		if leaky.Allow() {
			lkAllowed++
		} else {
			lkDenied++
		}
	}
	fmt.Printf("  burst: allowed=%d denied=%d (smoothing output)\n", lkAllowed, lkDenied)

	// ── PER-KEY LIMITER ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Per-key limiter (user-level, 3 burst, 5/s) ---")
	pkl := NewPerKeyLimiter(3, 5)
	for _, user := range []string{"user-A", "user-B", "user-A", "user-A", "user-A"} {
		result := "allowed"
		if !pkl.Allow(user) {
			result = "denied"
		}
		fmt.Printf("  [%s]: %s\n", user, result)
	}

	// ── COMPARISON ────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Algorithm comparison ---")
	fmt.Println(`  Token bucket:   allows burst up to capacity; refills at rate/s
  Sliding window: accurate count over rolling window; no burst spike at boundary
  Fixed window:   simple; bursts possible at window edge (2x spike)
  Leaky bucket:   smoothed output at constant rate; drops excess immediately`)
}
