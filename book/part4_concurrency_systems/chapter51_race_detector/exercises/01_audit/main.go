// FILE: book/part4_concurrency_systems/chapter51_race_detector/exercises/01_audit/main.go
// CHAPTER: 51 — Race Detector
// EXERCISE: Audit a small "service" that has five hidden races. Each race is
//           labelled with a TODO comment. The task is to find and fix all
//           races so the binary passes:
//             go run -race ./exercises/01_audit
//           with no race reports.
//
// Run before fixing:
//   go run -race ./exercises/01_audit
// Run after fixing to verify:
//   go run -race ./exercises/01_audit

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// COMPONENT 1: request counter — tracks in-flight and total requests
// ─────────────────────────────────────────────────────────────────────────────

type RequestCounter struct {
	inFlight int64     // FIX: use atomic.Int64
	total    int64     // FIX: use atomic.Int64
}

func (rc *RequestCounter) Begin() {
	rc.inFlight++     // TODO RACE 1: not atomic
	rc.total++        // TODO RACE 1: not atomic
}

func (rc *RequestCounter) End() {
	rc.inFlight--     // TODO RACE 1: not atomic
}

func (rc *RequestCounter) Snapshot() (inFlight, total int64) {
	return rc.inFlight, rc.total // TODO RACE 1: not atomic read
}

// ─────────────────────────────────────────────────────────────────────────────
// COMPONENT 2: simple in-memory cache with TTL
// ─────────────────────────────────────────────────────────────────────────────

type cacheEntry struct {
	value  string
	expiry time.Time
}

type Cache struct {
	// TODO RACE 2: the map is accessed from multiple goroutines without a lock.
	// FIX: add sync.RWMutex mu; use mu.Lock/RLock in Get and Set.
	entries map[string]cacheEntry
}

func NewCache() *Cache {
	return &Cache{entries: make(map[string]cacheEntry)}
}

func (c *Cache) Set(key, value string, ttl time.Duration) {
	c.entries[key] = cacheEntry{value: value, expiry: time.Now().Add(ttl)} // TODO RACE 2
}

func (c *Cache) Get(key string) (string, bool) {
	e, ok := c.entries[key] // TODO RACE 2
	if !ok || time.Now().After(e.expiry) {
		return "", false
	}
	return e.value, true
}

// ─────────────────────────────────────────────────────────────────────────────
// COMPONENT 3: result collector — accumulates results from workers
// ─────────────────────────────────────────────────────────────────────────────

type Collector struct {
	// TODO RACE 3: results slice is written by multiple goroutines.
	// FIX: use a channel or protect with sync.Mutex.
	results []string
}

func (col *Collector) Add(s string) {
	col.results = append(col.results, s) // TODO RACE 3: concurrent append
}

func (col *Collector) All() []string {
	return col.results
}

// ─────────────────────────────────────────────────────────────────────────────
// COMPONENT 4: lazy config loader
// ─────────────────────────────────────────────────────────────────────────────

type Config struct {
	// TODO RACE 4: loaded bool and value string are accessed without sync.
	// FIX: use sync.Once.
	loaded bool
	value  string
}

func (cfg *Config) Load() string {
	if !cfg.loaded { // TODO RACE 4: concurrent read
		time.Sleep(1 * time.Millisecond) // simulate slow load
		cfg.value = "db://localhost:5432"
		cfg.loaded = true // TODO RACE 4: concurrent write
	}
	return cfg.value
}

// ─────────────────────────────────────────────────────────────────────────────
// COMPONENT 5: event broadcaster
// ─────────────────────────────────────────────────────────────────────────────

type Broadcaster struct {
	// TODO RACE 5: listeners slice is read (broadcast) and written (subscribe)
	// concurrently without a lock.
	// FIX: add sync.RWMutex; use mu.Lock in Subscribe, mu.RLock in Broadcast.
	listeners []chan string
}

func (b *Broadcaster) Subscribe(bufSize int) <-chan string {
	ch := make(chan string, bufSize)
	b.listeners = append(b.listeners, ch) // TODO RACE 5: concurrent append
	return ch
}

func (b *Broadcaster) Broadcast(msg string) {
	for _, ch := range b.listeners { // TODO RACE 5: concurrent range
		select {
		case ch <- msg:
		default:
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DRIVER — exercises all five components concurrently
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Audit: find and fix 5 hidden races ===")
	fmt.Println("Run with -race to see the reports, then fix each TODO RACE comment.")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup

	// --- Race 1: request counter ---
	counter := &RequestCounter{}
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Begin()
			time.Sleep(time.Millisecond)
			counter.End()
		}()
	}
	wg.Wait()
	inFlight, total := counter.Snapshot()
	fmt.Printf("Counter: inFlight=%d total=%d (expected 0, 50)\n", inFlight, total)

	// --- Race 2: cache ---
	cache := NewCache()
	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", n)
			cache.Set(key, fmt.Sprintf("value-%d", n), 10*time.Second)
			_, _ = cache.Get(key)
		}(i)
	}
	wg.Wait()
	fmt.Println("Cache: writes and reads complete")

	// --- Race 3: collector ---
	col := &Collector{}
	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			col.Add(fmt.Sprintf("result-%d", n))
		}(i)
	}
	wg.Wait()
	fmt.Printf("Collector: %d results (expected 20)\n", len(col.All()))

	// --- Race 4: lazy config ---
	cfg := &Config{}
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cfg.Load()
		}()
	}
	wg.Wait()
	fmt.Printf("Config: %q\n", cfg.value)

	// --- Race 5: broadcaster ---
	bc := &Broadcaster{}
	subs := make([]<-chan string, 5)
	for i := range 5 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			subs[n] = bc.Subscribe(10)
		}(i)
	}
	wg.Wait()

	bc.Broadcast("hello")
	time.Sleep(10 * time.Millisecond)

	received := atomic.Int64{}
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		select {
		case <-sub:
			received.Add(1)
		default:
		}
	}
	fmt.Printf("Broadcaster: %d of 5 subscribers received message\n", received.Load())

	_ = ctx
	fmt.Println()
	fmt.Println("Fix all TODO RACE comments, then run with -race to verify.")
}
