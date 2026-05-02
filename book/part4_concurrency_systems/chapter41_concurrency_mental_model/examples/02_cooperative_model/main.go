// FILE: book/part4_concurrency_systems/chapter41_concurrency_mental_model/examples/02_cooperative_model/main.go
// CHAPTER: 41 — Concurrency Mental Model
// TOPIC: Concurrency vs parallelism, goroutine scheduling yield points,
//        GOMAXPROCS, the happens-before relationship, and data races.
//
// Run (from the chapter folder):
//   go run ./examples/02_cooperative_model

package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CONCURRENCY ≠ PARALLELISM
//
// Concurrency = structure (multiple independent tasks, possibly interleaved).
// Parallelism = execution (tasks literally running at the same instant).
//
// A single-CPU machine with GOMAXPROCS=1 is concurrent but not parallel.
// ─────────────────────────────────────────────────────────────────────────────

func demoConcurrencyVsParallelism() {
	fmt.Println("=== Concurrency vs Parallelism ===")
	fmt.Printf("  GOMAXPROCS = %d  (logical CPUs = %d)\n",
		runtime.GOMAXPROCS(0), runtime.NumCPU())

	// Two goroutines interleave on whatever CPUs are available.
	var wg sync.WaitGroup
	tick := make(chan string, 20)

	worker := func(name string, n int) {
		defer wg.Done()
		for i := 0; i < n; i++ {
			tick <- fmt.Sprintf("%s:%d", name, i)
			runtime.Gosched() // explicit yield — like a cooperative scheduling hint
		}
	}

	wg.Add(2)
	go worker("A", 4)
	go worker("B", 4)
	go func() {
		wg.Wait()
		close(tick)
	}()

	var seq []string
	for t := range tick {
		seq = append(seq, t)
	}
	fmt.Printf("  interleaved sequence: %v\n", seq)
	fmt.Println("  (order varies — goroutines are concurrent, not sequenced)")
}

// ─────────────────────────────────────────────────────────────────────────────
// YIELD POINTS — where the scheduler can switch goroutines
//
// The Go scheduler is cooperative + asynchronously preemptible (since 1.14).
// A goroutine yields at: channel ops, function calls, syscalls, runtime.Gosched.
// ─────────────────────────────────────────────────────────────────────────────

func demoYieldPoints() {
	fmt.Println()
	fmt.Println("=== Yield points ===")

	done := make(chan struct{})
	log := make(chan string, 10)

	go func() {
		log <- "goroutine: before channel send (yield point)"
		done <- struct{}{} // channel send — scheduler may switch here
		log <- "goroutine: after channel send"
		log <- "goroutine: done"
		close(log)
	}()

	<-done // channel receive — unblocks the other goroutine
	fmt.Println("  main: received from channel (yield point)")

	for msg := range log {
		fmt.Println(" ", msg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HAPPENS-BEFORE — the guarantee that makes channel sync safe
//
// Rule: a send on a channel happens-before the corresponding receive completes.
// This means: anything the sender did before the send is visible to the
// receiver after the receive.
// ─────────────────────────────────────────────────────────────────────────────

func demoHappensBefore() {
	fmt.Println()
	fmt.Println("=== Happens-before via channel ===")

	ready := make(chan struct{})
	var data string

	go func() {
		data = "hello from goroutine" // write happens-before the send
		ready <- struct{}{}           // send
	}()

	<-ready // receive happens-after the send, therefore after the write
	fmt.Printf("  data = %q  (safely visible because of happens-before)\n", data)
}

// ─────────────────────────────────────────────────────────────────────────────
// DATA RACE — what happens without synchronisation
//
// We cannot show an actual race here (it would be non-deterministic and the
// race detector would flag it), so we demonstrate the safe atomic version and
// explain the unsafe pattern.
// ─────────────────────────────────────────────────────────────────────────────

func demoAtomicVsRace() {
	fmt.Println()
	fmt.Println("=== Atomic (safe) counter ===")

	var counter int64
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&counter, 1) // atomic — no race
		}()
	}
	wg.Wait()
	fmt.Printf("  counter = %d  (always 1000 with atomic)\n", counter)

	fmt.Println()
	fmt.Println("  Unsafe pattern (NOT executed — would be a data race):")
	fmt.Println("    var n int")
	fmt.Println("    go func() { n++ }()  // unsynchronised write")
	fmt.Println("    go func() { n++ }()  // unsynchronised write — RACE")
	fmt.Println("  Detect with: go run -race ./...")
}

// ─────────────────────────────────────────────────────────────────────────────
// GOROUTINE LIFECYCLE — created, runnable, running, blocked, dead
// ─────────────────────────────────────────────────────────────────────────────

func demoLifecycle() {
	fmt.Println()
	fmt.Println("=== Goroutine lifecycle ===")

	before := runtime.NumGoroutine()

	blocked := make(chan struct{}) // goroutines will block on this
	var started sync.WaitGroup

	for i := 0; i < 5; i++ {
		started.Add(1)
		go func() {
			started.Done()
			<-blocked // goroutine is now BLOCKED (parked by scheduler)
		}()
	}
	started.Wait()
	time.Sleep(time.Millisecond) // let all goroutines park

	during := runtime.NumGoroutine()
	close(blocked) // unblock all goroutines — they will finish and become DEAD
	time.Sleep(5 * time.Millisecond)
	after := runtime.NumGoroutine()

	fmt.Printf("  goroutines before:  %d\n", before)
	fmt.Printf("  goroutines during:  %d  (+5 blocked)\n", during)
	fmt.Printf("  goroutines after:   %d  (all finished)\n", after)
}

func main() {
	demoConcurrencyVsParallelism()
	demoYieldPoints()
	demoHappensBefore()
	demoAtomicVsRace()
	demoLifecycle()
}
