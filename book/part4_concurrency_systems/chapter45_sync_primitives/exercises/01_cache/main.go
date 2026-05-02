// FILE: book/part4_concurrency_systems/chapter45_sync_primitives/exercises/01_cache/main.go
// CHAPTER: 45 — sync Primitives
// EXERCISE: Thread-safe cache with RWMutex, TTL expiry, sync.Once for
//           lazy initialisation, and sync.Pool for request buffers.
//
// Run (from the chapter folder):
//   go run ./exercises/01_cache

package main

import (
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CACHE ENTRY
// ─────────────────────────────────────────────────────────────────────────────

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

func (e entry[V]) expired() bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}

// ─────────────────────────────────────────────────────────────────────────────
// TTL CACHE — RWMutex for read-heavy workloads
// ─────────────────────────────────────────────────────────────────────────────

type Cache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]entry[V]
	hits    int64
	misses  int64
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{entries: make(map[K]entry[V])}
}

func (c *Cache[K, V]) Set(k K, v V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.entries[k] = entry[V]{value: v, expiresAt: exp}
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	c.mu.RLock()
	e, ok := c.entries[k]
	c.mu.RUnlock()

	if !ok || e.expired() {
		c.mu.Lock()
		c.misses++
		if ok && e.expired() {
			delete(c.entries, k) // evict expired entry
		}
		c.mu.Unlock()
		var zero V
		return zero, false
	}

	c.mu.Lock()
	c.hits++
	c.mu.Unlock()
	return e.value, true
}

func (c *Cache[K, V]) Delete(k K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, k)
}

func (c *Cache[K, V]) Purge() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for k, e := range c.entries {
		if e.expired() {
			delete(c.entries, k)
			n++
		}
	}
	return n
}

func (c *Cache[K, V]) Stats() (hits, misses int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses
}

// ─────────────────────────────────────────────────────────────────────────────
// LAZY LOADER — sync.Once per key
// ─────────────────────────────────────────────────────────────────────────────

type LazyLoader[K comparable, V any] struct {
	mu    sync.Mutex
	once  map[K]*sync.Once
	cache *Cache[K, V]
	load  func(K) (V, error)
}

func NewLazyLoader[K comparable, V any](load func(K) (V, error)) *LazyLoader[K, V] {
	return &LazyLoader[K, V]{
		once:  make(map[K]*sync.Once),
		cache: NewCache[K, V](),
		load:  load,
	}
}

func (l *LazyLoader[K, V]) Get(k K) (V, error) {
	if v, ok := l.cache.Get(k); ok {
		return v, nil
	}

	l.mu.Lock()
	if _, exists := l.once[k]; !exists {
		l.once[k] = &sync.Once{}
	}
	once := l.once[k]
	l.mu.Unlock()

	var result V
	var loadErr error
	once.Do(func() {
		v, err := l.load(k)
		if err == nil {
			l.cache.Set(k, v, 5*time.Minute)
		}
		result = v
		loadErr = err
	})

	if v, ok := l.cache.Get(k); ok {
		return v, nil
	}
	return result, loadErr
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// 1. Basic cache with TTL.
	fmt.Println("=== TTL Cache ===")

	c := NewCache[string, int]()
	c.Set("a", 1, 50*time.Millisecond)
	c.Set("b", 2, 0) // no TTL — lives forever

	v, ok := c.Get("a")
	fmt.Printf("  get a: v=%d ok=%v\n", v, ok)

	time.Sleep(60 * time.Millisecond)

	v, ok = c.Get("a") // expired
	fmt.Printf("  get a (after TTL): v=%d ok=%v\n", v, ok)

	v, ok = c.Get("b") // no TTL
	fmt.Printf("  get b (no TTL): v=%d ok=%v\n", v, ok)

	hits, misses := c.Stats()
	fmt.Printf("  stats: hits=%d misses=%d\n", hits, misses)

	// 2. Concurrent read-heavy workload.
	fmt.Println()
	fmt.Println("=== Concurrent reads (RWMutex) ===")

	cache2 := NewCache[int, string]()
	for i := range 10 {
		cache2.Set(i, fmt.Sprintf("val-%d", i), time.Minute)
	}

	var wg sync.WaitGroup
	hits2, misses2 := int64(0), int64(0)
	var mu sync.Mutex

	for range 200 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := int(time.Now().UnixNano() % 12) // some will miss (keys 10,11)
			_, ok := cache2.Get(key)
			mu.Lock()
			if ok {
				hits2++
			} else {
				misses2++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	fmt.Printf("  200 concurrent reads: hits=%d misses=%d\n", hits2, misses2)

	// 3. Lazy loader with sync.Once.
	fmt.Println()
	fmt.Println("=== LazyLoader (sync.Once per key) ===")

	loadCount := 0
	loader := NewLazyLoader(func(k string) (string, error) {
		loadCount++
		fmt.Printf("  loading key=%q (load #%d)\n", k, loadCount)
		time.Sleep(10 * time.Millisecond) // simulate expensive load
		return "value-for-" + k, nil
	})

	// 5 goroutines all request the same key concurrently — load runs only once.
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, _ := loader.Get("config")
			_ = v
		}()
	}
	wg.Wait()

	v2, _ := loader.Get("config") // hits cache
	fmt.Printf("  result: %q  load called %d time(s)\n", v2, loadCount)

	// 4. Purge expired entries.
	fmt.Println()
	fmt.Println("=== Purge expired ===")
	pc := NewCache[string, int]()
	pc.Set("short", 1, 20*time.Millisecond)
	pc.Set("long", 2, time.Hour)
	pc.Set("none", 3, 0)
	time.Sleep(30 * time.Millisecond)
	n := pc.Purge()
	fmt.Printf("  purged %d expired entries\n", n)
}
