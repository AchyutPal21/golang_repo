// FILE: book/part4_concurrency_systems/chapter43_channels_internals/examples/01_channel_mechanics/main.go
// CHAPTER: 43 — Channels: Internals
// TOPIC: hchan anatomy, buffered vs unbuffered, closing rules,
//        nil channel behaviour, directional channels, and range-over-channel.
//
// Run (from the chapter folder):
//   go run ./examples/01_channel_mechanics

package main

import (
	"fmt"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// UNBUFFERED vs BUFFERED
//
// Unbuffered (cap=0): send blocks until a receiver is ready (rendezvous).
// Buffered (cap=N):   send only blocks when buffer is full.
//                     receive only blocks when buffer is empty.
// ─────────────────────────────────────────────────────────────────────────────

func demoBuffering() {
	fmt.Println("=== Unbuffered vs Buffered ===")

	// Unbuffered — the goroutine must be ready before main can send.
	unbuf := make(chan int)
	go func() { fmt.Printf("  unbuffered received: %d\n", <-unbuf) }()
	unbuf <- 42 // blocks until the goroutine above receives

	// Buffered — main can send without a waiting receiver (up to cap).
	buf := make(chan int, 3)
	buf <- 1
	buf <- 2
	buf <- 3
	// buf <- 4  // would block — buffer full
	fmt.Printf("  buffered len=%d cap=%d\n", len(buf), cap(buf))
	fmt.Printf("  buffered drain: %d %d %d\n", <-buf, <-buf, <-buf)
}

// ─────────────────────────────────────────────────────────────────────────────
// CLOSING RULES
//
// 1. Only the sender should close a channel.
// 2. Sending on a closed channel panics.
// 3. Receiving from a closed channel always succeeds — returns zero values
//    once the buffer is drained.
// 4. The two-value receive (v, ok) lets you detect closure.
// ─────────────────────────────────────────────────────────────────────────────

func demoClosing() {
	fmt.Println()
	fmt.Println("=== Closing Rules ===")

	ch := make(chan int, 3)
	ch <- 10
	ch <- 20
	ch <- 30
	close(ch)

	// Read until drained.
	for {
		v, ok := <-ch
		if !ok {
			fmt.Println("  channel closed, done")
			break
		}
		fmt.Printf("  received: %d (ok=%v)\n", v, ok)
	}

	// Zero values after close.
	v, ok := <-ch
	fmt.Printf("  after drain: v=%d ok=%v\n", v, ok)
}

// ─────────────────────────────────────────────────────────────────────────────
// RANGE OVER CHANNEL — idiomatic way to drain until close
// ─────────────────────────────────────────────────────────────────────────────

func producer(n int) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch) // close signals the range loop to stop
		for i := range n {
			ch <- i * i
		}
	}()
	return ch
}

func demoRange() {
	fmt.Println()
	fmt.Println("=== Range over channel ===")

	var results []int
	for v := range producer(6) {
		results = append(results, v)
	}
	fmt.Printf("  squares: %v\n", results)
}

// ─────────────────────────────────────────────────────────────────────────────
// NIL CHANNEL — blocks forever on send AND receive
//
// A nil channel is useful in select to disable a case without removing it.
// ─────────────────────────────────────────────────────────────────────────────

func demoNilChannel() {
	fmt.Println()
	fmt.Println("=== Nil channel in select ===")

	a := make(chan string, 1)
	b := make(chan string, 1)
	a <- "from a"
	b <- "from b"

	// Disable channel 'a' after first receive by setting it to nil.
	var results []string
	for range 2 {
		select {
		case v, ok := <-a:
			if !ok {
				a = nil // disable this case
				continue
			}
			results = append(results, v)
			a = nil // read once, then disable
		case v := <-b:
			results = append(results, v)
		}
	}
	fmt.Printf("  results: %v\n", results)
}

// ─────────────────────────────────────────────────────────────────────────────
// DIRECTIONAL CHANNELS — read-only and write-only
//
// Directional channels enforce the contract at the type level.
// The compiler rejects writes to a receive-only (<-chan T) channel.
// ─────────────────────────────────────────────────────────────────────────────

// writeOnly accepts only a send channel — cannot receive from it.
func writeOnly(ch chan<- int, values []int) {
	for _, v := range values {
		ch <- v
	}
	close(ch)
}

// readOnly accepts only a receive channel — cannot send to it.
func readOnly(ch <-chan int) []int {
	var out []int
	for v := range ch {
		out = append(out, v)
	}
	return out
}

func demoDirectional() {
	fmt.Println()
	fmt.Println("=== Directional channels ===")

	ch := make(chan int, 5)
	writeOnly(ch, []int{10, 20, 30, 40, 50})
	fmt.Printf("  read-only result: %v\n", readOnly(ch))
}

// ─────────────────────────────────────────────────────────────────────────────
// HCHAN ANATOMY (conceptual)
//
// The runtime's hchan struct holds:
//   buf      — circular ring buffer (for buffered channels)
//   sendq    — list of blocked senders (goroutines waiting to send)
//   recvq    — list of blocked receivers (goroutines waiting to receive)
//   closed   — flag set by close()
//   lock     — a low-level mutex protecting the struct
//
// When a goroutine sends to a full buffered channel, it is added to sendq
// and parked. When a receiver drains one slot, it wakes the first sender
// from sendq and copies its value directly into the buffer slot.
// ─────────────────────────────────────────────────────────────────────────────

func demoHchanConcept() {
	fmt.Println()
	fmt.Println("=== hchan: blocking queues ===")

	ch := make(chan int, 1) // cap=1
	var wg sync.WaitGroup

	// sender 1 fills the buffer
	ch <- 100

	// sender 2 blocks (sendq) because buffer is full
	wg.Add(1)
	go func() {
		defer wg.Done()
		ch <- 200 // parks in sendq until receiver drains
		fmt.Println("  sender 2: unblocked after receiver drained")
	}()

	// receiver drains slot 1, waking sender 2
	v1 := <-ch
	fmt.Printf("  received: %d  (buffer had one slot)\n", v1)

	wg.Wait()
	v2 := <-ch
	fmt.Printf("  received: %d  (sent by unblocked sender 2)\n", v2)
}

func main() {
	demoBuffering()
	demoClosing()
	demoRange()
	demoNilChannel()
	demoDirectional()
	demoHchanConcept()
}
