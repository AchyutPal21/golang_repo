// 01_goroutines.go
//
// GOROUTINES — Go's lightweight concurrency primitive
//
// A goroutine is a function executing concurrently with other goroutines in the
// same address space. The keyword "go" before a function call starts a new goroutine.
//
// WHY GOROUTINES EXIST
// --------------------
// Traditional concurrency uses OS threads. OS threads are expensive:
//   - Each thread gets a fixed stack (typically 1-8 MB on Linux)
//   - Thread creation/destruction involves kernel syscalls
//   - Context switching between threads is expensive (~1-2 µs)
//   - The OS scheduler doesn't know about your program's logic
//
// Goroutines solve all of these:
//   - Goroutine starts with a 2KB stack (Go 1.4+), grows/shrinks dynamically
//   - Created/destroyed entirely in user space (no kernel involvement)
//   - Context switching is ~200ns (10x faster than threads)
//   - The Go scheduler knows when goroutines are blocked (I/O, channel ops)
//
// M:N SCHEDULING (The Go Scheduler)
// -----------------------------------
// Go uses an M:N scheduler: M goroutines are multiplexed onto N OS threads.
// The scheduler is part of the Go runtime, not the OS.
//
//   Goroutines (G) — your lightweight tasks
//   OS Threads (M) — "Machine" threads, actual kernel threads
//   Processors (P) — logical processors, GOMAXPROCS controls how many
//
// Each P has a local run queue of goroutines. When a goroutine blocks on a
// syscall, the P detaches from that M and attaches to another, keeping all
// Ps busy. This is "work stealing".
//
// GOROUTINE LIFECYCLE
// -------------------
//   1. "go f()" puts the goroutine in a run queue
//   2. A P picks it up and runs it on an M (OS thread)
//   3. If it blocks (I/O, channel, syscall), the scheduler suspends it
//   4. When unblocked, it's re-queued and eventually resumed
//   5. When the function returns, the goroutine is garbage collected
//
// GOROUTINE LEAKS
// ---------------
// A goroutine leak occurs when a goroutine is started but never terminates.
// Since goroutines are GC roots (the runtime keeps them alive), leaked
// goroutines accumulate memory indefinitely and can exhaust system resources.
//
// Common causes:
//   - Goroutine blocked waiting to send on a channel nobody reads
//   - Goroutine blocked waiting to receive on a channel nobody writes to
//   - Goroutine in an infinite loop with no exit condition
//   - Goroutine waiting on a sync.WaitGroup that never reaches zero
//
// GOROUTINE IDs — WHY THEY DON'T EXIST IN THE PUBLIC API
// --------------------------------------------------------
// Unlike threads, Go intentionally does NOT expose goroutine IDs in the
// standard library. Why?
//
// Thread-local storage (TLS) — storing data per-thread — is a common pattern
// in languages like Java. Java's ThreadLocal is essentially a map keyed by
// thread ID. The Go designers considered this harmful because:
//   1. It makes goroutine identity semantically significant, discouraging
//      goroutines from being freely created and pooled.
//   2. It encourages "goroutine-local" state which makes code harder to reason
//      about and test.
//   3. The correct Go idiom is to pass context explicitly.
//
// You CAN extract the goroutine ID by parsing runtime.Stack() output (the
// stack trace starts with "goroutine 7 [running]:"), but this is hacky and
// not supported. We'll demonstrate it below for educational purposes only.

package main

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// SECTION 1: The "go" keyword — starting a goroutine
// =============================================================================

func sayHello(name string) {
	fmt.Printf("Hello from goroutine, %s!\n", name)
}

func demoBasicGoroutine() {
	fmt.Println("=== Basic Goroutine ===")

	// Without "go": runs synchronously, blocks until done
	sayHello("synchronous call")

	// With "go": starts a new goroutine, returns immediately
	// The goroutine runs concurrently; main does NOT wait for it.
	go sayHello("goroutine")

	// PROBLEM: if main returns before the goroutine runs, you'll never see
	// "Hello from goroutine". The program exits, killing all goroutines.
	// The sleep below simulates "waiting" — but this is NOT how you do it
	// in real code. We use sync.WaitGroup (shown later).
	time.Sleep(10 * time.Millisecond)

	fmt.Println()
}

// =============================================================================
// SECTION 2: Multiple goroutines — ordering is non-deterministic
// =============================================================================

func demoMultipleGoroutines() {
	fmt.Println("=== Multiple Goroutines (non-deterministic order) ===")

	// Start 5 goroutines. Their execution order is NOT guaranteed.
	// The Go scheduler decides when each runs. Run this multiple times
	// and you'll see different orderings.
	for i := 0; i < 5; i++ {
		// GOTCHA: "i" is a loop variable captured by closure.
		// If we wrote:   go func() { fmt.Println(i) }()
		// By the time the goroutine runs, i may already be 5.
		// FIX: pass i as a parameter to capture its current value.
		go func(n int) {
			fmt.Printf("goroutine %d running\n", n)
		}(i) // <-- i is evaluated HERE and passed as n
	}

	time.Sleep(20 * time.Millisecond) // crude wait — replaced by WaitGroup below
	fmt.Println()
}

// =============================================================================
// SECTION 3: sync.WaitGroup — the correct way to wait for goroutines
// =============================================================================
//
// sync.WaitGroup maintains a counter:
//   Add(n)  — increment by n (call before launching goroutines)
//   Done()  — decrement by 1 (call in the goroutine when it finishes)
//   Wait()  — block until the counter reaches 0
//
// Think of it as: "I'm expecting n goroutines to finish; wake me when done."

func demoWaitGroup() {
	fmt.Println("=== sync.WaitGroup ===")

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1) // Increment BEFORE launching the goroutine.
		// If you call Add inside the goroutine, the race condition
		// means Wait() might see 0 and return before Add(1) is called.

		go func(n int) {
			defer wg.Done() // Always defer Done to ensure it's called even on panic.
			// defer executes when the surrounding function returns.
			// Without defer: if the goroutine panics, Done() is never called,
			// and Wait() blocks forever.
			fmt.Printf("worker %d: doing work...\n", n)
			time.Sleep(time.Duration(n*10) * time.Millisecond)
			fmt.Printf("worker %d: done\n", n)
		}(i)
	}

	fmt.Println("main: waiting for all workers...")
	wg.Wait() // Blocks here until all goroutines call Done()
	fmt.Println("main: all workers finished")
	fmt.Println()
}

// =============================================================================
// SECTION 4: Goroutine stack growth — why goroutines are cheap
// =============================================================================
//
// Each goroutine starts with a small stack (2KB in Go 1.14+).
// The stack grows automatically when needed (up to a configurable max, default 1GB).
// This is done via "stack copying": when a stack overflow is detected, a larger
// stack is allocated, all data is copied over, and pointers are adjusted.
//
// Contrast with OS threads: Linux threads get 8MB stacks by default. You can
// have ~100 goroutines for every OS thread in terms of memory. Programs with
// thousands of goroutines are perfectly normal in Go.

func demoGoroutineCount() {
	fmt.Println("=== Goroutine Count ===")

	// runtime.NumGoroutine() returns the number of currently running goroutines.
	fmt.Printf("goroutines before spawning: %d\n", runtime.NumGoroutine())

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond) // simulate brief work
		}()
	}

	// While they're running:
	fmt.Printf("goroutines during work: %d\n", runtime.NumGoroutine())

	wg.Wait()
	// After all goroutines finish, the count drops back down.
	// Give the runtime a moment to clean up.
	time.Sleep(5 * time.Millisecond)
	fmt.Printf("goroutines after all done: %d\n", runtime.NumGoroutine())
	fmt.Println()
}

// =============================================================================
// SECTION 5: GOMAXPROCS — controlling parallelism
// =============================================================================
//
// GOMAXPROCS sets the number of logical processors (P) the Go runtime uses.
// Default: the number of CPU cores (since Go 1.5).
//
// With GOMAXPROCS=1: concurrency, but not true parallelism — goroutines take
//                    turns on one OS thread.
// With GOMAXPROCS=4: up to 4 goroutines can truly run in parallel on 4 cores.
//
// You can set it programmatically or via the GOMAXPROCS environment variable.
// In production, leave it at the default unless you have a specific reason.

func demoGOMAXPROCS() {
	fmt.Println("=== GOMAXPROCS ===")

	current := runtime.GOMAXPROCS(0) // 0 = query without changing
	fmt.Printf("current GOMAXPROCS: %d\n", current)
	fmt.Printf("number of CPU cores: %d\n", runtime.NumCPU())

	// You can change it at runtime:
	old := runtime.GOMAXPROCS(2)
	fmt.Printf("changed to 2, was: %d\n", old)
	runtime.GOMAXPROCS(old) // restore
	fmt.Println()
}

// =============================================================================
// SECTION 6: Goroutine leak — demonstration and detection
// =============================================================================

// leakyFunction starts a goroutine that blocks forever waiting on a channel
// that will never be sent to. This goroutine will never exit.
func leakyFunction() {
	ch := make(chan int) // unbuffered, nobody sends to it

	go func() {
		// This goroutine blocks here forever.
		// After leakyFunction returns, ch goes out of scope in the caller,
		// but the goroutine still holds a reference to ch — preventing GC.
		val := <-ch
		fmt.Println("received:", val) // never reached
	}()

	// leakyFunction returns, but the goroutine above lives on forever.
	// Each call to leakyFunction leaks one goroutine.
}

// fixedFunction properly cancels the goroutine using a done channel.
func fixedFunction() func() {
	ch := make(chan int)
	done := make(chan struct{}) // struct{} costs 0 bytes — idiomatic for signals

	go func() {
		select {
		case val := <-ch:
			fmt.Println("received:", val)
		case <-done:
			fmt.Println("goroutine received cancel signal, exiting cleanly")
			return // goroutine exits, gets garbage collected
		}
	}()

	// Return a cancel function the caller can invoke to stop the goroutine.
	cancel := func() {
		close(done) // closing a channel unblocks all receivers
	}
	return cancel
}

func demoGoroutineLeak() {
	fmt.Println("=== Goroutine Leak Demo ===")

	before := runtime.NumGoroutine()
	fmt.Printf("goroutines before: %d\n", before)

	// Leak 3 goroutines
	leakyFunction()
	leakyFunction()
	leakyFunction()

	time.Sleep(5 * time.Millisecond) // let goroutines start
	during := runtime.NumGoroutine()
	fmt.Printf("goroutines after leaking 3: %d (leaked %d)\n", during, during-before)

	// The leaked goroutines will never be collected.
	// In a real server running for hours, this becomes a serious memory leak.

	fmt.Println("\n--- Leak Prevention with done channel ---")
	cancel := fixedFunction()
	time.Sleep(5 * time.Millisecond)
	cancel() // signal the goroutine to stop
	time.Sleep(5 * time.Millisecond)
	fmt.Println()
}

// =============================================================================
// SECTION 7: Goroutine ID — extracting it the hacky way (educational only)
// =============================================================================
//
// Go does NOT expose goroutine IDs. This function parses the runtime stack
// trace to extract the ID. DO NOT use this in production code.
// It exists here purely to show what the scheduler sees internally.

func goroutineID() int64 {
	// runtime.Stack writes the stack trace of current (and optionally all) goroutines.
	// A stack trace starts with: "goroutine 7 [running]:\n..."
	var buf [64]byte
	n := runtime.Stack(buf[:], false) // false = only current goroutine
	// buf now contains something like: "goroutine 18 [running]:\n..."
	s := string(buf[:n])
	// Extract the number between "goroutine " and " ["
	s = strings.TrimPrefix(s, "goroutine ")
	s = s[:strings.IndexByte(s, ' ')]
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

func demoGoroutineID() {
	fmt.Println("=== Goroutine ID (educational, not for production) ===")
	fmt.Printf("main goroutine ID: %d\n", goroutineID())

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			fmt.Printf("goroutine %d has runtime ID: %d\n", n, goroutineID())
		}(i)
	}
	wg.Wait()

	fmt.Println()
	fmt.Println("NOTE: The correct Go idiom is to pass context/state explicitly,")
	fmt.Println("not to key off goroutine ID. Never use this in production.")
	fmt.Println()
}

// =============================================================================
// SECTION 8: Anonymous goroutines vs named functions
// =============================================================================

func worker(id int, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("named worker %d starting\n", id)
	time.Sleep(time.Duration(id*5) * time.Millisecond)
	fmt.Printf("named worker %d done\n", id)
}

func demoNamedVsAnonymous() {
	fmt.Println("=== Named vs Anonymous Goroutines ===")

	var wg sync.WaitGroup

	// Named function goroutines — better for stack traces and profiling.
	// The stack trace will show "main.worker" rather than "main.demoNamedVsAnonymous.func1".
	// For anything non-trivial, prefer named functions.
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go worker(i, &wg) // pass *wg — worker needs to call wg.Done()
	}

	// Anonymous goroutines — fine for simple, short, inline logic.
	// Closure captures variables from the enclosing scope.
	for i := 4; i <= 6; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fmt.Printf("anonymous worker %d\n", id)
		}(i)
	}

	wg.Wait()
	fmt.Println()
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║         GOROUTINES — Deep Dive                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	demoBasicGoroutine()
	demoMultipleGoroutines()
	demoWaitGroup()
	demoGoroutineCount()
	demoGOMAXPROCS()
	demoGoroutineLeak()
	demoGoroutineID()
	demoNamedVsAnonymous()

	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("KEY TAKEAWAYS:")
	fmt.Println("  1. 'go f()' launches a goroutine; main doesn't wait for it")
	fmt.Println("  2. Goroutines are ~2KB stack vs ~1MB for OS threads")
	fmt.Println("  3. M:N scheduler: many goroutines on few OS threads")
	fmt.Println("  4. Use sync.WaitGroup to wait for goroutines to finish")
	fmt.Println("  5. Always provide a way for goroutines to exit (done channel)")
	fmt.Println("  6. Capture loop vars by parameter, not by closure reference")
	fmt.Println("  7. Goroutine IDs are not exposed — pass context explicitly")
	fmt.Println("═══════════════════════════════════════════════════════")
}
