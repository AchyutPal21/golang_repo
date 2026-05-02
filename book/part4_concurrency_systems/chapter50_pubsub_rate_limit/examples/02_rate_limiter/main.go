// FILE: book/part4_concurrency_systems/chapter50_pubsub_rate_limit/examples/02_rate_limiter/main.go
// CHAPTER: 50 — Pub/Sub, Rate Limit, Throttle
// TOPIC: Token-bucket rate limiter, sliding-window rate limiter,
//        throttle (debounce), and leaky-bucket analogue.
//
// Run (from the chapter folder):
//   go run ./examples/02_rate_limiter

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// TOKEN BUCKET — allow bursts up to cap, refill at rate tokens/sec
// ─────────────────────────────────────────────────────────────────────────────

type TokenBucket struct {
	tokens   float64
	cap      float64
	rate     float64 // tokens per second
	lastTime time.Time
	mu       sync.Mutex
}

func NewTokenBucket(cap, ratePerSec float64) *TokenBucket {
	return &TokenBucket{
		tokens:   cap,
		cap:      cap,
		rate:     ratePerSec,
		lastTime: time.Now(),
	}
}

// Allow returns true if one token is available and consumes it.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()
	tb.lastTime = now

	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.cap {
		tb.tokens = tb.cap
	}
	if tb.tokens < 1 {
		return false
	}
	tb.tokens--
	return true
}

// Wait blocks until a token is available or ctx is cancelled.
func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		if tb.Allow() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second / time.Duration(tb.rate)):
		}
	}
}

func demoTokenBucket() {
	fmt.Println("=== Token bucket: 5 rps, burst cap 3 ===")

	tb := NewTokenBucket(3, 5) // cap=3 tokens, refill at 5/sec
	ctx := context.Background()

	var accepted, rejected int64
	start := time.Now()

	// Fire 15 requests in a short burst.
	for i := range 15 {
		if tb.Allow() {
			accepted++
			fmt.Printf("  req %2d: accepted at %s\n", i+1, time.Since(start).Round(time.Millisecond))
		} else {
			rejected++
			fmt.Printf("  req %2d: rejected\n", i+1)
		}
		time.Sleep(20 * time.Millisecond)
	}

	fmt.Printf("  accepted: %d  rejected: %d\n", accepted, rejected)

	// Blocking mode — wait for tokens.
	fmt.Println("  --- blocking wait mode ---")
	start = time.Now()
	for i := range 4 {
		if err := tb.Wait(ctx); err != nil {
			fmt.Printf("  wait error: %v\n", err)
			return
		}
		fmt.Printf("  req %d admitted at %s\n", i+1, time.Since(start).Round(time.Millisecond))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SLIDING WINDOW RATE LIMITER — allow at most N requests in the last window
// ─────────────────────────────────────────────────────────────────────────────

type SlidingWindow struct {
	limit    int
	window   time.Duration
	mu       sync.Mutex
	requests []time.Time
}

func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	return &SlidingWindow{limit: limit, window: window}
}

func (sw *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)

	// Evict requests outside the window.
	start := 0
	for start < len(sw.requests) && sw.requests[start].Before(cutoff) {
		start++
	}
	sw.requests = sw.requests[start:]

	if len(sw.requests) >= sw.limit {
		return false
	}
	sw.requests = append(sw.requests, now)
	return true
}

func demoSlidingWindow() {
	fmt.Println()
	fmt.Println("=== Sliding window: 5 req / 100ms ===")

	sw := NewSlidingWindow(5, 100*time.Millisecond)
	accepted := 0

	for i := range 15 {
		allowed := sw.Allow()
		status := "rejected"
		if allowed {
			accepted++
			status = "accepted"
		}
		fmt.Printf("  req %2d at %dms: %s\n", i+1, i*20, status)
		time.Sleep(20 * time.Millisecond)
	}
	fmt.Printf("  total accepted: %d of 15\n", accepted)
}

// ─────────────────────────────────────────────────────────────────────────────
// THROTTLE — run fn at most once per interval (debounce leading-edge)
// ─────────────────────────────────────────────────────────────────────────────

type Throttle struct {
	interval time.Duration
	lastRun  time.Time
	mu       sync.Mutex
}

func NewThrottle(interval time.Duration) *Throttle {
	return &Throttle{interval: interval}
}

// Do calls fn only if interval has elapsed since the last call.
func (t *Throttle) Do(fn func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if time.Since(t.lastRun) >= t.interval {
		t.lastRun = time.Now()
		fn()
	}
}

func demoThrottle() {
	fmt.Println()
	fmt.Println("=== Throttle: execute at most once per 50ms ===")

	th := NewThrottle(50 * time.Millisecond)
	executed := atomic.Int64{}

	for i := range 10 {
		i := i
		th.Do(func() {
			executed.Add(1)
			fmt.Printf("  executed at iteration %d\n", i+1)
		})
		time.Sleep(15 * time.Millisecond)
	}
	fmt.Printf("  fn executed %d times out of 10 calls\n", executed.Load())
}

// ─────────────────────────────────────────────────────────────────────────────
// LEAKY BUCKET (via channel + drainer goroutine)
// ─────────────────────────────────────────────────────────────────────────────

type LeakyBucket struct {
	in  chan struct{}
	out chan struct{}
}

func NewLeakyBucket(cap int, drainRate time.Duration) *LeakyBucket {
	lb := &LeakyBucket{
		in:  make(chan struct{}, cap),
		out: make(chan struct{}, cap),
	}
	go func() {
		ticker := time.NewTicker(drainRate)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case item := <-lb.in:
				lb.out <- item
			default:
				// bucket empty — nothing to drain
			}
		}
	}()
	return lb
}

// TryAdd adds to the bucket; returns false if full (overflow).
func (lb *LeakyBucket) TryAdd() bool {
	select {
	case lb.in <- struct{}{}:
		return true
	default:
		return false
	}
}

func demoLeakyBucket() {
	fmt.Println()
	fmt.Println("=== Leaky bucket: cap=5, drain 1/50ms ===")

	lb := NewLeakyBucket(5, 50*time.Millisecond)

	// Drain output in background.
	drained := atomic.Int64{}
	go func() {
		for range lb.out {
			drained.Add(1)
		}
	}()

	overflow := 0
	for i := range 15 {
		if !lb.TryAdd() {
			overflow++
			fmt.Printf("  req %2d: overflow\n", i+1)
		}
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(400 * time.Millisecond) // let bucket drain
	fmt.Printf("  overflow: %d  drained: %d\n", overflow, drained.Load())
}

func main() {
	demoTokenBucket()
	demoSlidingWindow()
	demoThrottle()
	demoLeakyBucket()
}
