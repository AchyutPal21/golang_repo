// FILE: 08_standard_library/10_sync_and_atomic_stdlib.go
// TOPIC: sync.Map, sync.Pool, runtime package — deeper stdlib concurrency
//
// Run: go run 08_standard_library/10_sync_and_atomic_stdlib.go

package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: sync.Map, sync.Pool, runtime")
	fmt.Println("════════════════════════════════════════")

	// ── sync.Map — concurrent-safe map ────────────────────────────────────
	// sync.Map is a specialized concurrent map optimized for two use cases:
	//   1. Write-once, read-many (e.g., caches, registries)
	//   2. Keys disjoint per goroutine (no key shared between goroutines)
	// For general concurrent access: use map + sync.RWMutex (faster for most cases).
	// sync.Map has more overhead for write-heavy workloads.

	fmt.Println("\n── sync.Map ──")
	var sm sync.Map

	// Store:
	sm.Store("key1", "value1")
	sm.Store("key2", 42)

	// Load (returns interface{}, must type-assert):
	if v, ok := sm.Load("key1"); ok {
		fmt.Printf("  Load key1: %v (type: %T)\n", v, v)
	}

	// LoadOrStore — atomic: returns existing, or stores and returns new:
	actual, loaded := sm.LoadOrStore("key1", "new-value")
	fmt.Printf("  LoadOrStore key1: value=%v, loaded(existing)=%v\n", actual, loaded)

	actual2, loaded2 := sm.LoadOrStore("key3", "brand-new")
	fmt.Printf("  LoadOrStore key3: value=%v, loaded(existing)=%v\n", actual2, loaded2)

	// Delete:
	sm.Delete("key2")

	// Range — iterate (no guaranteed order):
	fmt.Println("  Range:")
	sm.Range(func(k, v interface{}) bool {
		fmt.Printf("    %v → %v\n", k, v)
		return true  // return false to stop iteration early
	})

	// Concurrent usage:
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sm.Store(fmt.Sprintf("goroutine-%d", n), n*n)
		}(i)
	}
	wg.Wait()
	count := 0
	sm.Range(func(k, v interface{}) bool { count++; return true })
	fmt.Printf("  After concurrent stores: %d entries\n", count)

	// ── sync.Pool — object reuse pool ──────────────────────────────────────
	// sync.Pool reduces GC pressure by reusing allocated objects.
	// Objects in the pool may be collected at any GC cycle.
	// Ideal for: buffers, encoder/decoder instances, temporary workspaces.
	fmt.Println("\n── sync.Pool ──")
	type Buffer struct {
		data [4096]byte
		size int
	}
	bufPool := sync.Pool{
		New: func() interface{} {
			fmt.Println("  Pool: allocating new Buffer")
			return &Buffer{}
		},
	}

	// Get buffer from pool (calls New if empty):
	buf := bufPool.Get().(*Buffer)
	buf.size = 100
	fmt.Printf("  Got buffer: size=%d\n", buf.size)

	// Return buffer to pool (reset first!):
	buf.size = 0  // MUST reset before returning
	bufPool.Put(buf)

	// Get again — reuses existing:
	buf2 := bufPool.Get().(*Buffer)
	fmt.Printf("  Got again: size=%d  (reused, no allocation log)\n", buf2.size)
	bufPool.Put(buf2)

	// ── atomic.Value — hot config pattern ────────────────────────────────
	fmt.Println("\n── atomic.Value (hot config) ──")
	type Config struct {
		MaxConns int
		Timeout  time.Duration
	}

	var cfg atomic.Value
	cfg.Store(&Config{MaxConns: 10, Timeout: 30 * time.Second})

	// Many readers:
	for i := 0; i < 3; i++ {
		go func(id int) {
			c := cfg.Load().(*Config)
			fmt.Printf("  Reader %d: MaxConns=%d\n", id, c.MaxConns)
		}(i)
	}

	// One writer (rare):
	time.Sleep(1 * time.Millisecond)
	cfg.Store(&Config{MaxConns: 50, Timeout: 10 * time.Second})
	fmt.Printf("  Updated config: MaxConns=%d\n", cfg.Load().(*Config).MaxConns)

	// ── runtime package ─────────────────────────────────────────────────
	fmt.Println("\n── runtime package ──")
	fmt.Printf("  runtime.GOOS:       %s\n", runtime.GOOS)
	fmt.Printf("  runtime.GOARCH:     %s\n", runtime.GOARCH)
	fmt.Printf("  runtime.GOVERSION:  %s\n", runtime.Version())
	fmt.Printf("  runtime.NumCPU:     %d  (logical CPUs)\n", runtime.NumCPU())
	fmt.Printf("  runtime.GOMAXPROCS: %d  (goroutines in parallel)\n", runtime.GOMAXPROCS(0))

	// Goroutine count:
	fmt.Printf("  runtime.NumGoroutine: %d\n", runtime.NumGoroutine())

	// Memory stats:
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	fmt.Printf("  Alloc:      %d KB  (live heap)\n", memStats.Alloc/1024)
	fmt.Printf("  TotalAlloc: %d KB  (total allocated ever)\n", memStats.TotalAlloc/1024)
	fmt.Printf("  NumGC:      %d     (GC cycles run)\n", memStats.NumGC)

	// GOMAXPROCS — how many goroutines can run in parallel
	// Default: runtime.NumCPU() (since Go 1.5)
	// Set via GOMAXPROCS env var or runtime.GOMAXPROCS(n)
	// For CPU-bound work: set to NumCPU
	// For I/O-bound work: can be higher
	prev := runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Printf("  Set GOMAXPROCS to %d (was %d)\n", runtime.NumCPU(), prev)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  sync.Map: concurrent map for write-once/read-many or disjoint keys")
	fmt.Println("  sync.Pool: reuse objects, reduce GC; always reset before Put()")
	fmt.Println("  atomic.Value: atomic storage for any type (great for hot config)")
	fmt.Println("  runtime.NumCPU: set GOMAXPROCS for optimal CPU-bound parallelism")
	fmt.Println("  runtime.MemStats: monitor heap size, GC cycles, allocations")
}
