// FILE: book/part4_concurrency_systems/chapter46_sync_atomic/examples/02_atomic_value/main.go
// CHAPTER: 46 — sync/atomic
// TOPIC: atomic.Value — store and load any value atomically,
//        hot-reload config pattern, and atomic.Pointer[T].
//
// Run (from the chapter folder):
//   go run ./examples/02_atomic_value

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// atomic.Value — store any immutable value atomically
//
// Rules:
//   - Store must always store a value of the same concrete type.
//   - Stored values should be treated as immutable — never modify after storing.
//   - Load returns nil if nothing has been stored yet.
// ─────────────────────────────────────────────────────────────────────────────

type Config struct {
	LogLevel   string
	MaxRetries int
	Timeout    time.Duration
}

func demoAtomicValue() {
	fmt.Println("=== atomic.Value (hot-reload config) ===")

	var cfgVal atomic.Value

	// Initial config.
	cfgVal.Store(Config{
		LogLevel:   "info",
		MaxRetries: 3,
		Timeout:    5 * time.Second,
	})

	// 10 readers load the config concurrently.
	var wg sync.WaitGroup
	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cfg := cfgVal.Load().(Config) // type assertion always succeeds here
			_ = cfg
		}(i)
	}

	// One writer updates the config atomically while readers are running.
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond)
		cfgVal.Store(Config{
			LogLevel:   "debug",
			MaxRetries: 5,
			Timeout:    10 * time.Second,
		})
		fmt.Println("  config updated atomically")
	}()

	wg.Wait()
	cfg := cfgVal.Load().(Config)
	fmt.Printf("  final config: log=%s retries=%d timeout=%s\n",
		cfg.LogLevel, cfg.MaxRetries, cfg.Timeout)
}

// ─────────────────────────────────────────────────────────────────────────────
// HOT-RELOAD PATTERN — a watcher goroutine updates the config;
//                      handlers read it on every request with zero locking.
// ─────────────────────────────────────────────────────────────────────────────

type HotConfig struct {
	val atomic.Value
}

func (h *HotConfig) Load() Config {
	v := h.val.Load()
	if v == nil {
		return Config{LogLevel: "info", MaxRetries: 3, Timeout: 5 * time.Second}
	}
	return v.(Config)
}

func (h *HotConfig) Update(c Config) {
	h.val.Store(c)
}

func demoHotReload() {
	fmt.Println()
	fmt.Println("=== Hot-reload pattern ===")

	cfg := &HotConfig{}
	cfg.Update(Config{LogLevel: "info", MaxRetries: 3, Timeout: 5 * time.Second})

	done := make(chan struct{})
	var wg sync.WaitGroup

	// Config watcher goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		versions := []Config{
			{LogLevel: "debug", MaxRetries: 5, Timeout: 2 * time.Second},
			{LogLevel: "warn", MaxRetries: 1, Timeout: 30 * time.Second},
		}
		for _, v := range versions {
			time.Sleep(20 * time.Millisecond)
			cfg.Update(v)
			fmt.Printf("  watcher: updated to log=%s\n", v.LogLevel)
		}
		close(done)
	}()

	// Request handlers — load config on each "request".
	seen := make(map[string]bool)
	ticker := time.NewTicker(8 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c := cfg.Load()
			if !seen[c.LogLevel] {
				seen[c.LogLevel] = true
				fmt.Printf("  handler: using config log=%s retries=%d\n",
					c.LogLevel, c.MaxRetries)
			}
		case <-done:
			wg.Wait()
			fmt.Println("  hot-reload demo complete")
			return
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// atomic.Pointer[T] (Go 1.19+) — typed pointer swap
//
// Equivalent to atomic.Value but type-safe for pointer types.
// ─────────────────────────────────────────────────────────────────────────────

type RouteTable struct {
	routes map[string]string
}

func demoAtomicPointer() {
	fmt.Println()
	fmt.Println("=== atomic.Pointer[T] ===")

	var ptr atomic.Pointer[RouteTable]

	// Initial routing table.
	ptr.Store(&RouteTable{routes: map[string]string{
		"/api/users": "users-service",
		"/api/orders": "orders-service",
	}})

	var wg sync.WaitGroup

	// Reader goroutines.
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rt := ptr.Load() // zero-copy pointer load
			_ = rt.routes["/api/users"]
		}()
	}

	// Swap to a new routing table atomically.
	wg.Add(1)
	go func() {
		defer wg.Done()
		newRT := &RouteTable{routes: map[string]string{
			"/api/users":    "users-v2-service",
			"/api/orders":   "orders-v2-service",
			"/api/products": "products-service",
		}}
		old := ptr.Swap(newRT)
		fmt.Printf("  swapped: old had %d routes, new has %d routes\n",
			len(old.routes), len(newRT.routes))
	}()

	wg.Wait()
	rt := ptr.Load()
	fmt.Printf("  final routing: %v\n", rt.routes)
}

func main() {
	demoAtomicValue()
	demoHotReload()
	demoAtomicPointer()
}
