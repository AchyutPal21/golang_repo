// FILE: 06_concurrency/06_sync_atomic.go
// TOPIC: sync/atomic — lock-free operations for simple shared state
//
// Run: go run 06_concurrency/06_sync_atomic.go

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: sync/atomic")
	fmt.Println("════════════════════════════════════════")

	// ── WHY ATOMICS? ──────────────────────────────────────────────────────
	// For a SINGLE shared variable (counter, flag), a mutex is overkill.
	// Atomics perform the read-modify-write in ONE CPU instruction — no lock.
	// Use atomics for: counters, flags, one-value shared state.
	// Use mutex for: multiple related variables that must change together.

	// ── atomic.Int64 (Go 1.19+ typed atomics — preferred) ────────────────
	fmt.Println("\n── atomic.Int64 counter ──")
	var counter atomic.Int64

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1)  // atomic increment — no race condition
		}()
	}
	wg.Wait()
	fmt.Printf("  Counter after 1000 goroutines: %d  (always exactly 1000)\n", counter.Load())

	// ── atomic.Bool ───────────────────────────────────────────────────────
	fmt.Println("\n── atomic.Bool ──")
	var initialized atomic.Bool
	fmt.Printf("  initialized: %v\n", initialized.Load())
	initialized.Store(true)
	fmt.Printf("  After Store(true): %v\n", initialized.Load())

	// CompareAndSwap: only stores if current value matches expected.
	// This is the foundation of lock-free algorithms.
	swapped := initialized.CompareAndSwap(true, false)
	fmt.Printf("  CAS(true→false): swapped=%v, value=%v\n", swapped, initialized.Load())

	// ── atomic.Value — store any type atomically ──────────────────────────
	// Use case: config object that is read constantly but updated rarely.
	// Any number of goroutines can Load concurrently — no lock needed.
	fmt.Println("\n── atomic.Value (hot config) ──")
	type Config struct{ MaxConns int; Timeout int }
	var cfg atomic.Value
	cfg.Store(Config{MaxConns: 10, Timeout: 30})

	// Readers (goroutines) load the current config atomically:
	current := cfg.Load().(Config)
	fmt.Printf("  Config: %+v\n", current)

	// Writer updates the whole config atomically:
	cfg.Store(Config{MaxConns: 20, Timeout: 60})
	updated := cfg.Load().(Config)
	fmt.Printf("  Updated config: %+v\n", updated)

	// ── Low-level atomic functions (pre-1.19 style) ───────────────────────
	fmt.Println("\n── Low-level atomic funcs (int64) ──")
	var n int64
	atomic.AddInt64(&n, 5)
	atomic.AddInt64(&n, 3)
	fmt.Printf("  After two AddInt64: %d\n", atomic.LoadInt64(&n))
	atomic.StoreInt64(&n, 100)
	fmt.Printf("  After StoreInt64(100): %d\n", atomic.LoadInt64(&n))

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  atomic.Int64 / Bool / Pointer → typed, preferred (Go 1.19+)")
	fmt.Println("  atomic.Value → store any type, great for hot config")
	fmt.Println("  CompareAndSwap → conditional update (lock-free algorithms)")
	fmt.Println("  Use atomics: single variable, read-heavy")
	fmt.Println("  Use mutex: multiple related variables, complex invariants")
}
