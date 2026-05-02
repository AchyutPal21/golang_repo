// FILE: book/part4_concurrency_systems/chapter42_goroutines_internals/examples/02_goroutine_patterns/main.go
// CHAPTER: 42 — Goroutines: Internals
// TOPIC: Goroutine lifetime management — done channels, WaitGroup,
//        errgroup, goroutine leaks, and the loop-variable capture trap.
//
// Run (from the chapter folder):
//   go run ./examples/02_goroutine_patterns

package main

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// GOROUTINE LEAK — a goroutine that is never collected
//
// A goroutine blocked on a channel with no sender/receiver will run forever,
// consuming memory and a scheduler slot. Always ensure every goroutine has a
// path to exit.
// ─────────────────────────────────────────────────────────────────────────────

func leakyWorker(id int) {
	// This goroutine blocks forever — it's a leak.
	ch := make(chan struct{})
	go func() {
		<-ch // nobody will ever send here
		fmt.Println("never printed", id)
	}()
}

func nonLeakyWorker(id int, done <-chan struct{}) {
	go func() {
		select {
		case <-done:
			// caller signals done — goroutine exits cleanly
		case <-time.After(100 * time.Millisecond):
			// timeout fallback
		}
	}()
}

func demoLeaks() {
	fmt.Println("=== Goroutine Leak vs Clean Exit ===")

	before := runtime.NumGoroutine()

	// Leak: 3 goroutines that will never exit.
	for i := 0; i < 3; i++ {
		leakyWorker(i)
	}
	time.Sleep(time.Millisecond) // let them start
	leaked := runtime.NumGoroutine()

	// Clean: 3 goroutines with a done channel.
	done := make(chan struct{})
	for i := 0; i < 3; i++ {
		nonLeakyWorker(i, done)
	}
	time.Sleep(time.Millisecond)
	withClean := runtime.NumGoroutine()

	close(done) // signal all clean workers to exit
	time.Sleep(10 * time.Millisecond)
	afterClose := runtime.NumGoroutine()

	fmt.Printf("  before:      %d\n", before)
	fmt.Printf("  +3 leaky:    %d  (leaks persist)\n", leaked)
	fmt.Printf("  +3 clean:    %d  (total goroutines)\n", withClean)
	fmt.Printf("  after done:  %d  (clean ones exited; leaks remain)\n", afterClose)
}

// ─────────────────────────────────────────────────────────────────────────────
// LOOP-VARIABLE CAPTURE TRAP (Go < 1.22)
//
// In Go < 1.22, the loop variable is a single address reused each iteration.
// A goroutine that captures it by reference sees whatever value the variable
// holds when it runs — usually the final value. Go 1.22 fixed this by giving
// each iteration its own variable.
// ─────────────────────────────────────────────────────────────────────────────

func demoLoopCapture() {
	fmt.Println()
	fmt.Println("=== Loop Variable Capture ===")

	// Go 1.22+: each iteration has its own 'i' — all goroutines see their value.
	var wg sync.WaitGroup
	results := make([]int, 5)
	for i := range 5 {
		wg.Add(1)
		go func() { // Go 1.22: 'i' is per-iteration
			defer wg.Done()
			results[i] = i * i
		}()
	}
	wg.Wait()
	fmt.Printf("  squares (Go 1.22 loop vars): %v\n", results)

	// Pre-1.22 workaround: pass as argument to bind the value.
	results2 := make([]int, 5)
	for i := range 5 {
		wg.Add(1)
		go func(n int) { // n is a copy — safe in any Go version
			defer wg.Done()
			results2[n] = n * n
		}(i)
	}
	wg.Wait()
	fmt.Printf("  squares (arg copy pattern):  %v\n", results2)
}

// ─────────────────────────────────────────────────────────────────────────────
// ERRGROUP — WaitGroup + first-error collection
//
// golang.org/x/sync/errgroup is the standard solution for "run N goroutines,
// collect the first error." We implement a minimal version here so the chapter
// has no external dependencies.
// ─────────────────────────────────────────────────────────────────────────────

type errGroup struct {
	wg      sync.WaitGroup
	once    sync.Once
	firstErr error
}

func (g *errGroup) Go(fn func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := fn(); err != nil {
			g.once.Do(func() { g.firstErr = err })
		}
	}()
}

func (g *errGroup) Wait() error {
	g.wg.Wait()
	return g.firstErr
}

func demoErrGroup() {
	fmt.Println()
	fmt.Println("=== errGroup (first-error collection) ===")

	// All tasks succeed.
	var g1 errGroup
	for i := range 4 {
		n := i
		g1.Go(func() error {
			time.Sleep(time.Duration(n) * 5 * time.Millisecond)
			return nil
		})
	}
	fmt.Printf("  all success: err=%v\n", g1.Wait())

	// One task fails — first error is captured.
	var g2 errGroup
	for i := range 4 {
		n := i
		g2.Go(func() error {
			if n == 2 {
				return errors.New("task 2 failed")
			}
			return nil
		})
	}
	fmt.Printf("  one failure: err=%v\n", g2.Wait())
}

// ─────────────────────────────────────────────────────────────────────────────
// GOROUTINE CREATION COST — demonstrate that goroutines are cheap
// ─────────────────────────────────────────────────────────────────────────────

func demoCreationCost() {
	fmt.Println()
	fmt.Println("=== Goroutine Creation Cost ===")

	const n = 100_000
	start := time.Now()

	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			wg.Done()
		}()
	}
	wg.Wait()

	elapsed := time.Since(start)
	fmt.Printf("  created and joined %d goroutines in %s\n", n, elapsed.Round(time.Millisecond))
	fmt.Printf("  average per goroutine: ~%dns\n", elapsed.Nanoseconds()/n)
}

func main() {
	demoLeaks()
	demoLoopCapture()
	demoErrGroup()
	demoCreationCost()
}
