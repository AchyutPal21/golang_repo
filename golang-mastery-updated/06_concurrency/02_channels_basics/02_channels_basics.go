// 02_channels_basics.go
//
// CHANNELS — Go's core communication primitive
//
// "Do not communicate by sharing memory; instead, share memory by communicating."
//                                               — Rob Pike (Go Proverbs)
//
// WHY CHANNELS?
// -------------
// Traditional concurrent programming shares memory between threads and uses
// locks (mutexes) to prevent simultaneous access. This leads to:
//   - Deadlocks (A waits for B, B waits for A)
//   - Race conditions (unsynchronized reads/writes)
//   - Complex lock ordering rules
//
// Channels provide a different model: goroutines communicate by passing values
// through channels. The channel itself is the synchronization primitive.
//
// A channel is a typed conduit — only values of type T can flow through chan T.
//
// CHANNEL MECHANICS (UNDER THE HOOD)
// ------------------------------------
// Internally, a channel is a struct (runtime/chan.go → hchan) containing:
//   - A ring buffer (for buffered channels)
//   - A send queue (goroutines blocked on send)
//   - A receive queue (goroutines blocked on receive)
//   - A mutex protecting the internal state
//   - Size, element type info
//
// When you send on a channel, the runtime checks:
//   1. Is there a receiver already waiting? → hand off directly (no buffer copy)
//   2. Is there buffer space? → put in buffer, continue
//   3. Neither? → add sender to send queue, suspend the goroutine (context switch)
//
// The inverse applies for receiving.

package main

import (
	"fmt"
	"time"
)

// =============================================================================
// SECTION 1: Channel declaration and the zero value
// =============================================================================

func demoChannelDeclaration() {
	fmt.Println("=== Channel Declaration ===")

	// make(chan T) — creates an UNBUFFERED channel of type T
	// make(chan T, n) — creates a BUFFERED channel with capacity n
	ch1 := make(chan int)          // unbuffered int channel
	ch2 := make(chan string, 5)    // buffered string channel, capacity 5
	ch3 := make(chan struct{})     // zero-size signal channel (very common idiom)
	ch4 := make(chan []byte, 10)   // buffered channel of byte slices

	// Zero value of a channel is nil.
	// A nil channel blocks forever on send AND receive.
	// This sounds useless but is actually useful in select statements (see 04_select).
	var ch5 chan int // nil channel
	fmt.Printf("ch1 (unbuffered int):      %v, len=%d, cap=%d\n", ch1, len(ch1), cap(ch1))
	fmt.Printf("ch2 (buffered string,5):   %v, len=%d, cap=%d\n", ch2, len(ch2), cap(ch2))
	fmt.Printf("ch3 (struct{} signal):     %v\n", ch3)
	fmt.Printf("ch4 (buffered []byte,10):  %v, len=%d, cap=%d\n", ch4, len(ch4), cap(ch4))
	fmt.Printf("ch5 (nil channel):         %v\n", ch5)

	_ = ch1; _ = ch2; _ = ch3; _ = ch4
	fmt.Println()
}

// =============================================================================
// SECTION 2: Send and receive syntax
// =============================================================================
//
// ch <- value   SEND value into channel ch (blocks if ch is full/unbuffered and no receiver)
// value := <-ch RECEIVE from channel ch (blocks if ch is empty and no sender)
// <-ch          RECEIVE and discard the value (just for synchronization)

func demoSendReceiveSyntax() {
	fmt.Println("=== Send and Receive Syntax ===")

	ch := make(chan int, 3) // buffered so we don't need a goroutine

	// Send values — won't block because buffer has space
	ch <- 10
	ch <- 20
	ch <- 30

	fmt.Printf("buffer length: %d, capacity: %d\n", len(ch), cap(ch))

	// Receive values
	a := <-ch
	b := <-ch
	c := <-ch
	fmt.Printf("received: %d, %d, %d\n", a, b, c)

	// Receive with ok-idiom (important for closed channels):
	ch2 := make(chan string, 2)
	ch2 <- "hello"
	ch2 <- "world"
	close(ch2) // close the channel — no more sends allowed

	// val, ok := <-ch
	// ok is true if a value was received, false if channel is closed and empty
	for {
		val, ok := <-ch2
		if !ok {
			fmt.Println("channel closed, no more values")
			break
		}
		fmt.Printf("received from ch2: %q\n", val)
	}

	fmt.Println()
}

// =============================================================================
// SECTION 3: Unbuffered channels — synchronous rendezvous
// =============================================================================
//
// An unbuffered channel has NO storage. A send BLOCKS until a receiver is ready.
// A receive BLOCKS until a sender is ready. They must rendezvous simultaneously.
//
// This makes unbuffered channels great for:
//   - Synchronization (guaranteeing ordering between goroutines)
//   - Handing off ownership of data
//   - Signaling events

func demoUnbufferedChannel() {
	fmt.Println("=== Unbuffered Channel (synchronous) ===")

	ch := make(chan int) // unbuffered

	// WRONG (would deadlock):
	// ch <- 42      // send blocks — nobody is receiving
	// val := <-ch   // receive blocks — nobody is sending (unreachable)

	// CORRECT: sender and receiver in separate goroutines
	go func() {
		fmt.Println("  sender: about to send 42...")
		ch <- 42 // blocks here until someone receives
		fmt.Println("  sender: send complete (receiver is now running)")
	}()

	time.Sleep(50 * time.Millisecond) // let sender block first (for demo clarity)
	fmt.Println("  receiver: about to receive...")
	val := <-ch // rendezvous — both unblock simultaneously
	fmt.Printf("  receiver: got %d\n", val)

	// KEY INSIGHT: when ch <- 42 completes, we KNOW the receiver has the value.
	// With a buffered channel, the sender would continue before the receiver processes.
	// Unbuffered = synchronous handoff = stronger guarantee.

	fmt.Println()
}

// =============================================================================
// SECTION 4: Buffered channels — asynchronous up to capacity
// =============================================================================
//
// A buffered channel has internal storage for N values.
// Sending blocks only when the buffer is FULL.
// Receiving blocks only when the buffer is EMPTY.
//
// Use buffered channels when:
//   - You want to decouple sender speed from receiver speed
//   - Implementing a work queue (channel as a job queue)
//   - Bounding concurrency (semaphore pattern)
//
// COMMON MISTAKE: using buffered channels to "fix" a deadlock.
// A deadlock caused by incorrect logic is NOT fixed by adding a buffer —
// you're just deferring the problem. Use buffers intentionally.

func demoBufferedChannel() {
	fmt.Println("=== Buffered Channel (asynchronous) ===")

	ch := make(chan string, 3) // buffer of 3

	// We can send up to 3 times without a receiver, because buffer absorbs them
	ch <- "first"
	ch <- "second"
	ch <- "third"
	// ch <- "fourth" // would block here — buffer is full

	fmt.Printf("after 3 sends: len=%d, cap=%d\n", len(ch), cap(ch))

	// We can receive without blocking because buffer has values
	fmt.Println("receiving:", <-ch)
	fmt.Println("receiving:", <-ch)
	fmt.Println("receiving:", <-ch)

	fmt.Printf("after 3 receives: len=%d, cap=%d\n", len(ch), cap(ch))

	// -------------------------------------------------------------------------
	// Semaphore pattern: use a buffered channel to limit concurrency
	// -------------------------------------------------------------------------
	// Only allow 3 goroutines to run at once, regardless of total goroutines.
	sem := make(chan struct{}, 3) // semaphore with capacity 3

	fmt.Println("\nSemaphore demo (max 3 concurrent):")
	done := make(chan struct{})
	count := 0

	for i := 0; i < 8; i++ {
		go func(id int) {
			sem <- struct{}{} // acquire: blocks if 3 are already running
			// --- critical section ---
			fmt.Printf("  worker %d running (sem len=%d)\n", id, len(sem))
			time.Sleep(30 * time.Millisecond)
			// --- end critical section ---
			<-sem              // release: allow another goroutine in
			done <- struct{}{} // signal completion
		}(i)
	}

	for i := 0; i < 8; i++ {
		<-done
		count++
	}
	fmt.Printf("all %d workers completed\n", count)
	fmt.Println()
}

// =============================================================================
// SECTION 5: Channel directions — send-only and receive-only
// =============================================================================
//
// Channel direction types restrict what operations are allowed:
//
//   chan T      — bidirectional (can send AND receive)
//   chan<- T    — send-only (can only send, "chan arrow T")
//   <-chan T    — receive-only (can only receive, "arrow chan T")
//
// WHY USE DIRECTIONAL CHANNELS?
//   - Self-documenting: function signature tells you its role
//   - Safety: compiler catches accidental misuse
//   - Principle of least privilege: give each goroutine only what it needs
//
// A bidirectional chan T converts implicitly to chan<- T or <-chan T.
// The reverse is NOT allowed (cannot widen restrictions).

// producer takes a send-only channel. It cannot accidentally read from it.
func producer(out chan<- int) {
	for i := 0; i < 5; i++ {
		out <- i * i // send squares
	}
	close(out) // producers close their output channels
	// Note: it's the SENDER's responsibility to close, not the receiver's.
	// Sending to a closed channel panics. Receiving from closed channel returns zero+false.
}

// consumer takes a receive-only channel. It cannot accidentally write to it.
func consumer(in <-chan int) {
	for val := range in { // range on channel receives until closed
		fmt.Printf("  consumed: %d\n", val)
	}
}

func demoChannelDirections() {
	fmt.Println("=== Channel Directions ===")

	// ch is bidirectional — we pass it to functions that each get a restricted view
	ch := make(chan int, 5)
	producer(ch) // ch implicitly converts to chan<- int inside producer
	consumer(ch) // ch implicitly converts to <-chan int inside consumer

	// The compiler would catch:
	// func wrongProducer(out chan<- int) { val := <-out } // ERROR: receive from send-only
	// func wrongConsumer(in <-chan int) { in <- 5 }       // ERROR: send to receive-only

	fmt.Println()
}

// =============================================================================
// SECTION 6: Closing a channel
// =============================================================================
//
// close(ch) signals that no more values will be sent on ch.
// Rules:
//   1. Only the SENDER should close a channel (never the receiver).
//   2. Sending to a closed channel causes a PANIC.
//   3. Closing an already-closed channel causes a PANIC.
//   4. Receiving from a closed channel returns immediately with the zero value.
//   5. Closing a nil channel causes a PANIC.
//
// Why close? It's how producers signal "I'm done" to consumers.
// range over a channel loops until the channel is closed.

func demoClosingChannel() {
	fmt.Println("=== Closing a Channel ===")

	ch := make(chan int, 5)
	for i := 0; i < 5; i++ {
		ch <- i
	}
	close(ch)

	// Receiving from a closed, empty channel returns zero value + false
	val, ok := <-ch // still has buffered values
	fmt.Printf("receive 1: val=%d, ok=%v\n", val, ok)

	val, ok = <-ch
	fmt.Printf("receive 2: val=%d, ok=%v\n", val, ok)

	// ... eventually, buffer is drained:
	for range 3 { // Go 1.22 range-over-integer
		val, ok = <-ch
		fmt.Printf("receive: val=%d, ok=%v\n", val, ok)
	}

	// After all buffered values consumed, channel is closed and empty:
	val, ok = <-ch
	fmt.Printf("after drain: val=%d, ok=%v (zero value + false)\n", val, ok)

	// Panics to be aware of (commented out to not crash the demo):
	// close(ch)   // panic: close of closed channel
	// ch <- 99    // panic: send on closed channel
	// var nilCh chan int
	// close(nilCh) // panic: close of nil channel

	fmt.Println()
}

// =============================================================================
// SECTION 7: The nil channel — blocks forever
// =============================================================================
//
// A nil channel (var ch chan T) blocks forever on both send and receive.
// This sounds like a bug, but it's INTENTIONAL in select statements.
//
// Use case: dynamically disable a select case by setting the channel to nil.
// We'll see this fully in 04_select_statement.go; previewed here.

func demoNilChannel() {
	fmt.Println("=== Nil Channel Behavior ===")

	var ch chan int // nil channel
	fmt.Printf("nil channel: %v\n", ch)

	// These would block forever (deadlock in a single goroutine):
	// <-ch      // blocks
	// ch <- 1   // blocks

	// The useful pattern: set a channel variable to nil to "disable" it in select.
	// Example (non-blocking because we use select with default):
	select {
	case val := <-ch: // ch is nil, so this case never fires
		fmt.Println("received:", val)
	default:
		fmt.Println("nil channel case did not fire (as expected)")
	}

	fmt.Println()
}

// =============================================================================
// SECTION 8: Channel as a future/promise
// =============================================================================
//
// A common Go pattern: a function starts a goroutine and returns a channel.
// The caller can do other work and later receive the result from the channel.
// This is Go's equivalent of a "future" or "promise".

func computeAsync(x, y int) <-chan int {
	result := make(chan int, 1) // buffered 1 so goroutine doesn't block
	go func() {
		time.Sleep(50 * time.Millisecond) // simulate slow computation
		result <- x * y
		// No close needed — single value, receiver knows exactly one value comes.
		// But if multiple values or ranging: close after all sends.
	}()
	return result
}

func demoChannelAsFuture() {
	fmt.Println("=== Channel as Future/Promise ===")

	// Start two async computations concurrently
	future1 := computeAsync(6, 7)
	future2 := computeAsync(3, 14)

	// Do other work here while computations run in background...
	fmt.Println("waiting for results...")

	// Block until results are ready
	r1 := <-future1
	r2 := <-future2
	fmt.Printf("6 × 7 = %d\n", r1)
	fmt.Printf("3 × 14 = %d\n", r2)
	fmt.Println()
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║         CHANNELS BASICS — Deep Dive                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	demoChannelDeclaration()
	demoSendReceiveSyntax()
	demoUnbufferedChannel()
	demoBufferedChannel()
	demoChannelDirections()
	demoClosingChannel()
	demoNilChannel()
	demoChannelAsFuture()

	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("KEY TAKEAWAYS:")
	fmt.Println("  1. make(chan T) = unbuffered; make(chan T,n) = buffered n")
	fmt.Println("  2. Unbuffered: send blocks until receiver is ready (rendezvous)")
	fmt.Println("  3. Buffered: send blocks only when buffer is full")
	fmt.Println("  4. chan<- T = send-only;  <-chan T = receive-only")
	fmt.Println("  5. Only SENDERS close channels; sending to closed = panic")
	fmt.Println("  6. Receiving from closed empty channel gives zero+false")
	fmt.Println("  7. nil channel blocks forever — useful to disable select cases")
	fmt.Println("  8. val, ok := <-ch detects channel closure")
	fmt.Println("═══════════════════════════════════════════════════════")
}
