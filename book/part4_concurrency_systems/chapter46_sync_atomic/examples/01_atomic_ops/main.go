// FILE: book/part4_concurrency_systems/chapter46_sync_atomic/examples/01_atomic_ops/main.go
// CHAPTER: 46 — sync/atomic
// TOPIC: atomic.Int64, atomic.Uint64, atomic.Bool, CompareAndSwap (CAS),
//        memory ordering, and when to prefer atomic over mutex.
//
// Run (from the chapter folder):
//   go run ./examples/01_atomic_ops

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// BASIC ATOMIC OPERATIONS
//
// sync/atomic provides lock-free operations on integer types.
// All operations are atomic with respect to other goroutines — no mutex needed.
// ─────────────────────────────────────────────────────────────────────────────

func demoBasicAtomics() {
	fmt.Println("=== Basic atomic operations ===")

	// atomic.Int64 (Go 1.19+ typed API — preferred over raw functions)
	var counter atomic.Int64

	var wg sync.WaitGroup
	for range 1000 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1)
		}()
	}
	wg.Wait()
	fmt.Printf("  Add (1000 goroutines): %d\n", counter.Load())

	// Store / Load
	counter.Store(42)
	fmt.Printf("  Store/Load: %d\n", counter.Load())

	// Swap — returns old value
	old := counter.Swap(100)
	fmt.Printf("  Swap(100): old=%d new=%d\n", old, counter.Load())

	// atomic.Bool (Go 1.19+)
	var flag atomic.Bool
	fmt.Printf("  Bool initial: %v\n", flag.Load())
	flag.Store(true)
	fmt.Printf("  Bool after Store(true): %v\n", flag.Load())
	prev := flag.Swap(false)
	fmt.Printf("  Bool Swap(false): prev=%v now=%v\n", prev, flag.Load())
}

// ─────────────────────────────────────────────────────────────────────────────
// COMPARE-AND-SWAP (CAS) — the foundation of lock-free algorithms
//
// CompareAndSwap(addr, old, new) atomically:
//   if *addr == old { *addr = new; return true }
//   else            { return false }
//
// CAS is the building block for all lock-free data structures.
// ─────────────────────────────────────────────────────────────────────────────

func demoCAS() {
	fmt.Println()
	fmt.Println("=== CompareAndSwap ===")

	var val atomic.Int64
	val.Store(10)

	// CAS succeeds when current value matches expected.
	swapped := val.CompareAndSwap(10, 20)
	fmt.Printf("  CAS(10→20): swapped=%v new=%d\n", swapped, val.Load())

	// CAS fails when current value does NOT match expected.
	swapped = val.CompareAndSwap(10, 30) // current is 20, not 10
	fmt.Printf("  CAS(10→30): swapped=%v (expected false — value was 20)\n", swapped)
	fmt.Printf("  value unchanged: %d\n", val.Load())

	// Lock-free increment using CAS loop (illustrates the pattern).
	var n atomic.Int64
	n.Store(0)
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				old := n.Load()
				if n.CompareAndSwap(old, old+1) {
					return
				}
				// CAS failed — another goroutine incremented first; retry
			}
		}()
	}
	wg.Wait()
	fmt.Printf("  lock-free increment result: %d  (always 100)\n", n.Load())
}

// ─────────────────────────────────────────────────────────────────────────────
// OLD-STYLE PACKAGE-LEVEL FUNCTIONS (pre-Go-1.19)
//
// These are still valid and used in code that predates Go 1.19 or that needs
// to operate on plain int64 variables (not atomic.Int64 values).
// ─────────────────────────────────────────────────────────────────────────────

func demoLegacyFunctions() {
	fmt.Println()
	fmt.Println("=== Legacy atomic functions (pre-1.19 style) ===")

	var n int64

	// These are equivalent to atomic.Int64 methods but operate on *int64.
	atomic.StoreInt64(&n, 100)
	fmt.Printf("  StoreInt64: %d\n", atomic.LoadInt64(&n))

	atomic.AddInt64(&n, 50)
	fmt.Printf("  AddInt64(50): %d\n", atomic.LoadInt64(&n))

	old := atomic.SwapInt64(&n, 0)
	fmt.Printf("  SwapInt64(0): old=%d new=%d\n", old, atomic.LoadInt64(&n))

	ok := atomic.CompareAndSwapInt64(&n, 0, 999)
	fmt.Printf("  CAS(0→999): ok=%v n=%d\n", ok, atomic.LoadInt64(&n))
}

// ─────────────────────────────────────────────────────────────────────────────
// SPIN-LOCK using atomic.Bool — for illustration only
//
// In practice, always use sync.Mutex. A spin-lock burns CPU waiting.
// But it illustrates how CAS underpins all locks.
// ─────────────────────────────────────────────────────────────────────────────

type SpinLock struct{ held atomic.Bool }

func (s *SpinLock) Lock() {
	for !s.held.CompareAndSwap(false, true) {
		// spin — yield to avoid monopolising the CPU
	}
}

func (s *SpinLock) Unlock() { s.held.Store(false) }

func demoSpinLock() {
	fmt.Println()
	fmt.Println("=== SpinLock (CAS-based, illustration only) ===")

	var sl SpinLock
	count := 0
	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sl.Lock()
			count++
			sl.Unlock()
		}()
	}
	wg.Wait()
	fmt.Printf("  spin-lock counter = %d  (always 100)\n", count)
	fmt.Println("  (in production, use sync.Mutex — it parks instead of spinning)")
}

func main() {
	demoBasicAtomics()
	demoCAS()
	demoLegacyFunctions()
	demoSpinLock()
}
