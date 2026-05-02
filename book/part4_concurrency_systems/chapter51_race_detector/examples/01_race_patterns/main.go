// FILE: book/part4_concurrency_systems/chapter51_race_detector/examples/01_race_patterns/main.go
// CHAPTER: 51 — Race Detector
// TOPIC: Common data race patterns — unsynchronised counter, map write,
//        slice append, closure capture, and bool-flag init.
//        Run with -race to see the detector report; run without -race
//        to observe the silent wrong results that races produce.
//
//   go run                           ./examples/01_race_patterns   # silent wrong output
//   go run -race                     ./examples/01_race_patterns   # race reports + exit 66
//   go build -race -o /tmp/rp        ./examples/01_race_patterns
//   GORACE="halt_on_error=0 log_path=/tmp/race.log" /tmp/rp
//
// NOTE: None of these patterns panic without -race; the races are real
//       but manifest as wrong results rather than crashes (except for map,
//       which is why that pattern uses only 2 goroutines to reduce crash odds).

package main

import (
	"fmt"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// RACE 1: unsynchronised counter
//
// 1000 goroutines each do count++. Read-modify-write is not atomic;
// updates are silently lost. Typical result: 950–999 instead of 1000.
// ─────────────────────────────────────────────────────────────────────────────

func demoRacyCounter() {
	fmt.Println("=== Race 1: unsynchronised counter ===")
	var count int
	var wg sync.WaitGroup
	for range 1000 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count++ // DATA RACE: concurrent read-modify-write
		}()
	}
	wg.Wait()
	fmt.Printf("  result: %d  (expected 1000 — lost updates make it less)\n", count)
}

// ─────────────────────────────────────────────────────────────────────────────
// RACE 2: concurrent struct field writes
//
// Two goroutines write to different fields of the same struct.
// Although the fields are logically independent, they share a cache line and
// a memory address space; the race detector reports this correctly.
// ─────────────────────────────────────────────────────────────────────────────

type Stats struct {
	Reads  int
	Writes int
}

func demoRacyStruct() {
	fmt.Println()
	fmt.Println("=== Race 2: concurrent struct field writes ===")

	var s Stats
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1000 {
			s.Reads++ // DATA RACE: concurrent write to s.Reads
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 1000 {
			s.Writes++ // concurrent write to s.Writes
		}
	}()

	wg.Wait()
	// Values will be lower than 1000 due to lost updates.
	fmt.Printf("  Reads=%d Writes=%d (both expected 1000)\n", s.Reads, s.Writes)
}

// ─────────────────────────────────────────────────────────────────────────────
// NOTE: concurrent map writes
//
// Concurrent map writes always panic in Go (the runtime has a built-in check).
// Example code (DO NOT run as shown):
//
//   m := make(map[int]int)
//   go func() { m[1] = 1 }()   // goroutine A
//   go func() { m[2] = 2 }()   // goroutine B
//   // runtime panics: "concurrent map writes"
//
// Fix: protect with sync.RWMutex or use sync.Map.
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// RACE 3: concurrent slice append
//
// Goroutines share a single slice header. Concurrent appends produce a slice
// shorter than expected because goroutines overwrite each other's length field.
// ─────────────────────────────────────────────────────────────────────────────

func demoRacyAppend() {
	fmt.Println()
	fmt.Println("=== Race 3: concurrent slice append ===")

	var results []int
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			results = append(results, n) // DATA RACE: concurrent read+write of slice header
		}(i)
	}
	wg.Wait()
	fmt.Printf("  len(results) = %d  (expected 100; some appends overwrite each other)\n", len(results))
}

// ─────────────────────────────────────────────────────────────────────────────
// RACE 4: closure capturing a shared variable
//
// Each goroutine reads `shared` while the loop is still modifying it.
// The -race detector identifies the exact goroutine and line numbers.
// ─────────────────────────────────────────────────────────────────────────────

func demoRacyClosure() {
	fmt.Println()
	fmt.Println("=== Race 4: closure over shared variable ===")

	shared := 0
	var wg sync.WaitGroup

	for i := range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			shared = i // DATA RACE: reading and writing shared across goroutines
		}()
	}
	wg.Wait()
	fmt.Printf("  shared = %d  (value is non-deterministic)\n", shared)
}

// ─────────────────────────────────────────────────────────────────────────────
// RACE 5: bool-flag DIY "once" initialisation
//
// Multiple goroutines check the same boolean flag without synchronisation.
// The result can be that `config` is initialised multiple times or not at all.
// ─────────────────────────────────────────────────────────────────────────────

var initialized bool
var config string

func demoRacyInit() {
	fmt.Println()
	fmt.Println("=== Race 5: bool-flag DIY once init ===")

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !initialized { // DATA RACE: concurrent read
				config = "loaded"    // DATA RACE: concurrent write
				initialized = true   // DATA RACE: concurrent write
			}
		}()
	}
	wg.Wait()
	fmt.Printf("  initialized=%v  config=%q\n", initialized, config)
	fmt.Println("  (sync.Once is the correct fix — see examples/02_fixing_races)")
}

func main() {
	demoRacyCounter()
	demoRacyStruct()
	demoRacyAppend()
	demoRacyClosure()
	demoRacyInit()

	fmt.Println()
	fmt.Println("─────────────────────────────────────────────")
	fmt.Println("Run with -race to see the full detector report")
	fmt.Println("─────────────────────────────────────────────")
}
