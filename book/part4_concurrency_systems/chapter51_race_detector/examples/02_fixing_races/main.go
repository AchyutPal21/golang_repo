// FILE: book/part4_concurrency_systems/chapter51_race_detector/examples/02_fixing_races/main.go
// CHAPTER: 51 — Race Detector
// TOPIC: Fixing each of the five race patterns from 01_race_patterns —
//        atomic counter, sync.RWMutex map, channel-based append,
//        explicit loop-var capture, and sync.Once init.
//        This file passes go run -race clean.
//
// Run (from the chapter folder):
//   go run -race ./examples/02_fixing_races

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// FIX 1: atomic counter
// ─────────────────────────────────────────────────────────────────────────────

func fixCounter() {
	fmt.Println("=== Fix 1: atomic counter ===")
	var count atomic.Int64
	var wg sync.WaitGroup
	for range 1000 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count.Add(1) // atomic — no race
		}()
	}
	wg.Wait()
	fmt.Printf("  result: %d  (always exactly 1000)\n", count.Load())
}

// ─────────────────────────────────────────────────────────────────────────────
// FIX 2: mutex-protected struct
// ─────────────────────────────────────────────────────────────────────────────

type SafeStats struct {
	mu     sync.Mutex
	Reads  int
	Writes int
}

func fixStruct() {
	fmt.Println()
	fmt.Println("=== Fix 2: mutex-protected struct fields ===")

	var s SafeStats
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1000 {
			s.mu.Lock()
			s.Reads++
			s.mu.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1000 {
			s.mu.Lock()
			s.Writes++
			s.mu.Unlock()
		}
	}()

	wg.Wait()
	fmt.Printf("  Reads=%d Writes=%d (always 1000 each)\n", s.Reads, s.Writes)
}

// ─────────────────────────────────────────────────────────────────────────────
// FIX 3: channel-based append (single collector goroutine owns the slice)
// ─────────────────────────────────────────────────────────────────────────────

func fixAppend() {
	fmt.Println()
	fmt.Println("=== Fix 3: single-owner append via channel ===")

	ch := make(chan int, 100)
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ch <- n // send — no race on channel
		}(i)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var results []int
	for v := range ch { // only this goroutine appends — no race
		results = append(results, v)
	}
	fmt.Printf("  len(results) = %d  (always exactly 100)\n", len(results))
}

// ─────────────────────────────────────────────────────────────────────────────
// FIX 4: explicit loop-variable capture
// ─────────────────────────────────────────────────────────────────────────────

func fixClosure() {
	fmt.Println()
	fmt.Println("=== Fix 4: explicit loop-variable capture ===")

	var shared atomic.Int64
	var wg sync.WaitGroup

	for i := range 5 {
		wg.Add(1)
		go func(val int) { // val is a copy, not a closure over i
			defer wg.Done()
			shared.Add(int64(val))
		}(i)
	}
	wg.Wait()
	// 0+1+2+3+4 = 10
	fmt.Printf("  shared = %d  (always exactly 10)\n", shared.Load())
}

// ─────────────────────────────────────────────────────────────────────────────
// FIX 5: sync.Once for one-time initialisation
// ─────────────────────────────────────────────────────────────────────────────

var (
	initOnce sync.Once
	cfg      string
)

func fixInit() {
	fmt.Println()
	fmt.Println("=== Fix 5: sync.Once for init ===")

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			initOnce.Do(func() { // guaranteed to run exactly once
				cfg = "loaded"
			})
		}()
	}
	wg.Wait()
	fmt.Printf("  cfg=%q (always \"loaded\", set exactly once)\n", cfg)
}

// ─────────────────────────────────────────────────────────────────────────────
// BONUS: sync.Map — concurrent-safe map without external locking
// ─────────────────────────────────────────────────────────────────────────────

func fixMap() {
	fmt.Println()
	fmt.Println("=== Bonus: sync.Map for concurrent-safe map ===")

	var m sync.Map
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			m.Store(n, n*n)
		}(i)
	}
	wg.Wait()

	count := 0
	m.Range(func(_, _ any) bool {
		count++
		return true
	})
	fmt.Printf("  sync.Map has %d entries\n", count)
}

func main() {
	fixCounter()
	fixStruct()
	fixAppend()
	fixClosure()
	fixInit()
	fixMap()

	fmt.Println()
	fmt.Println("Run with -race to confirm zero races reported.")
}
