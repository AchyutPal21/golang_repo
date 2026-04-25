// FILE: book/part1_foundations/chapter01_why_go_exists/examples/02_concurrent_clock/main.go
// CHAPTER: 01 — Why Go Exists
// TOPIC: Goroutines and channels, the simplest possible introduction.
//
// Run (from the chapter folder):
//   go run ./examples/02_concurrent_clock
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   To show that "concurrency in Go" is not a library, not a framework, and
//   not a future. Two background goroutines run cooperatively; a channel
//   coordinates their shutdown; the program exits cleanly after a fixed
//   duration. Every language we know has *some* answer to this — but in
//   Go, this is *the* answer, and it ships in the language.
//
//   You will not understand all of this on a first read. That is fine. The
//   point is to put the *shape* of Go concurrency in your head: spawn with
//   `go`, communicate with channels, terminate with `close` or context.
//   Chapters 41–47 unpack every line.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"time"
)

// ─── The first goroutine: a clock that prints the wall-clock time ───────────
//
// This is just an ordinary function. It becomes a goroutine when we call it
// with `go clock(done)` below.
//
// It takes a single parameter: a *receive-only* channel of empty struct
// values. The `<-chan struct{}` type means "I can read from this channel,
// but I cannot write to it." The empty struct (`struct{}`) carries no
// data — it's a zero-byte signal. The pattern "channel of struct{} used
// as a done signal" is one of the most idiomatic forms in Go.
func clock(done <-chan struct{}) {
	// time.Tick returns a channel that emits the current time on a regular
	// interval. We'll create one that fires every 250 ms.
	//
	// In production code prefer time.NewTicker so you can Stop() it; we'll
	// see that pattern in Chapter 47. time.Tick is fine for short-lived
	// programs like this teaching example.
	tick := time.Tick(250 * time.Millisecond)

	for {
		// `select` is Go's multi-way wait. It picks whichever case is
		// ready first. If both are ready, it picks one at random.
		select {
		case t := <-tick:
			// A tick arrived. Print the time, formatted to milliseconds.
			fmt.Printf("[clock]   %s\n", t.Format("15:04:05.000"))
		case <-done:
			// The done channel was closed (or had a value sent). Either
			// way, our `<-done` receive succeeds, which means it's time
			// to exit. Returning from a goroutine is how it terminates;
			// the runtime cleans up its stack.
			fmt.Println("[clock]   shutting down")
			return
		}
	}
}

// ─── The second goroutine: a counter that prints a number every 400 ms ──────
//
// Same shape: receives the done signal, ticks on its own cadence, prints,
// returns when done.
func counter(done <-chan struct{}) {
	tick := time.Tick(400 * time.Millisecond)
	n := 0
	for {
		select {
		case <-tick:
			n++
			fmt.Printf("[counter] %d\n", n)
		case <-done:
			fmt.Println("[counter] shutting down")
			return
		}
	}
}

// ─── main: spawn, wait, signal, exit ────────────────────────────────────────
func main() {
	// `done` is a channel of empty struct. We never *send* on it; we only
	// *close* it, which causes every receive on it to return immediately.
	// This is the standard "broadcast a stop signal to N goroutines"
	// idiom in Go. Closing a channel is *visible to all receivers* — that
	// is why we don't have to send N values for N goroutines.
	done := make(chan struct{})

	// Spawn the two goroutines. The `go` keyword schedules the function
	// to run concurrently and returns immediately. Neither call blocks.
	//
	// Note: we pass `done` by *value*. Channels in Go are reference types
	// under the hood — both goroutines and main share the same channel.
	go clock(done)
	go counter(done)

	// The main goroutine sleeps for 3 seconds. While it's sleeping, the
	// two goroutines we spawned are doing their work. The Go scheduler
	// multiplexes all three onto the OS threads available to the
	// process; we don't have to think about it.
	fmt.Println("[main]    will run for 3 seconds...")
	time.Sleep(3 * time.Second)

	// Now we ask the workers to stop. Closing the channel is the broadcast.
	// After this line, every `<-done` in clock() and counter() returns,
	// their loops exit, and they print their shutdown messages.
	fmt.Println("[main]    closing done channel")
	close(done)

	// We don't have a clean "wait for goroutines to exit" primitive in
	// this naive version. We sleep briefly so their shutdown messages
	// have time to print before main returns and the program exits.
	//
	// In real code, we use sync.WaitGroup (Chapter 45) or errgroup
	// (Chapter 48) to do this properly. The point of *this* file is the
	// `go` keyword, not the cleanup.
	time.Sleep(50 * time.Millisecond)

	fmt.Println("[main]    bye")
}
