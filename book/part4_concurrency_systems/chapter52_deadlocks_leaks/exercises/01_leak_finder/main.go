// FILE: book/part4_concurrency_systems/chapter52_deadlocks_leaks/exercises/01_leak_finder/main.go
// CHAPTER: 52 — Deadlocks, Leaks
// EXERCISE: A "service" has four goroutine leaks hidden inside it.
//           Each leak is labelled with a TODO LEAK comment.
//           Use runtime.NumGoroutine() before/after calls to locate leaks,
//           then fix each one so goroutinesBefore == goroutinesAfter.
//
// Run:
//   go run ./exercises/01_leak_finder

package main

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// HELPER
// ─────────────────────────────────────────────────────────────────────────────

func goroutines() int { return runtime.NumGoroutine() }

// ─────────────────────────────────────────────────────────────────────────────
// LEAK 1: background worker that never stops
// ─────────────────────────────────────────────────────────────────────────────

// StartPoller polls every interval but has no stop mechanism.
// TODO LEAK 1: add a done <-chan struct{} parameter and return when done closes.
func StartPoller(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		// TODO LEAK 1: defer ticker.Stop() is missing
		for range ticker.C {
			// poll work (noop)
		}
	}()
}

// ─────────────────────────────────────────────────────────────────────────────
// LEAK 2: request handler that spawns a goroutine but abandons it
// ─────────────────────────────────────────────────────────────────────────────

// HandleRequest sends work to a goroutine but the goroutine blocks if
// the context is cancelled before it can send its result.
// TODO LEAK 2: the result goroutine must select on ctx.Done().
func HandleRequest(ctx context.Context, workDuration time.Duration) string {
	ch := make(chan string) // unbuffered
	go func() {
		time.Sleep(workDuration) // simulate work — ignores ctx
		ch <- "result"           // TODO LEAK 2: blocks if caller already returned
	}()

	select {
	case res := <-ch:
		return res
	case <-ctx.Done():
		return "timeout"
		// goroutine is now blocked trying to send to ch
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// LEAK 3: fan-out with no error path drain
// ─────────────────────────────────────────────────────────────────────────────

// FanOut launches n goroutines but the caller only reads the first result
// via early return. The remaining goroutines block trying to send.
// TODO LEAK 3: use a buffered channel (cap n) so all goroutines can send.
func FanOut(ctx context.Context, n int) string {
	results := make(chan string) // unbuffered — TODO LEAK 3: should be make(chan string, n)
	for i := range n {
		go func(id int) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(id+1) * 5 * time.Millisecond):
				results <- fmt.Sprintf("result-%d", id) // blocks if nobody reads
			}
		}(i)
	}
	return <-results // only reads one — remaining senders block
}

// ─────────────────────────────────────────────────────────────────────────────
// LEAK 4: subscriber never unsubscribes
// ─────────────────────────────────────────────────────────────────────────────

type EventStream struct {
	ch chan string
}

func NewEventStream() *EventStream {
	es := &EventStream{ch: make(chan string, 10)}
	go func() {
		for i := 0; ; i++ {
			es.ch <- fmt.Sprintf("event-%d", i)
			time.Sleep(5 * time.Millisecond)
		}
	}()
	return es
}

// Subscribe reads events until its done channel closes.
// TODO LEAK 4: the subscribe goroutine never stops because done is never closed.
func (es *EventStream) Subscribe() {
	go func() {
		for range es.ch { // TODO LEAK 4: no done channel or context
			// handle event (noop)
		}
	}()
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN — measures goroutine counts before/after each component
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Leak Finder: locate and fix 4 goroutine leaks ===")
	fmt.Println()

	// Give the runtime a moment to settle.
	time.Sleep(20 * time.Millisecond)
	base := goroutines()
	fmt.Printf("base goroutine count: %d\n\n", base)

	// --- Leak 1 ---
	b1 := goroutines()
	StartPoller(10 * time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	a1 := goroutines()
	fmt.Printf("Leak 1 (poller):     before=%d after=%d leaked=%d\n", b1, a1, a1-b1)

	// --- Leak 2 ---
	b2 := goroutines()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel2()
	result := HandleRequest(ctx2, 100*time.Millisecond) // work longer than timeout
	time.Sleep(20 * time.Millisecond)
	a2 := goroutines()
	fmt.Printf("Leak 2 (handler):    before=%d after=%d leaked=%d  result=%q\n", b2, a2, a2-b2, result)

	// --- Leak 3 ---
	b3 := goroutines()
	ctx3, cancel3 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel3()
	r := FanOut(ctx3, 5)
	time.Sleep(50 * time.Millisecond) // let blocked goroutines show up
	a3 := goroutines()
	fmt.Printf("Leak 3 (fan-out):    before=%d after=%d leaked=%d  first=%q\n", b3, a3, a3-b3, r)

	// --- Leak 4 ---
	b4 := goroutines()
	es := NewEventStream()
	es.Subscribe()
	time.Sleep(30 * time.Millisecond)
	a4 := goroutines()
	fmt.Printf("Leak 4 (subscriber): before=%d after=%d leaked=%d\n", b4, a4, a4-b4)

	fmt.Println()
	total := goroutines() - base
	fmt.Printf("total leaked goroutines (above base): %d\n", total)
	fmt.Println()
	fmt.Println("Fix each TODO LEAK comment and re-run to see count reach 0.")
}
