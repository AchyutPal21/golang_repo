// FILE: book/part4_concurrency_systems/chapter43_channels_internals/exercises/01_ring_buffer/main.go
// CHAPTER: 43 — Channels: Internals
// EXERCISE: Build a typed ring buffer backed by a buffered channel.
//           Demonstrate producer/consumer with backpressure, timeout-based
//           non-blocking sends, and drain-on-close semantics.
//
// Run (from the chapter folder):
//   go run ./exercises/01_ring_buffer

package main

import (
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// RING BUFFER — bounded FIFO backed by a buffered channel
// ─────────────────────────────────────────────────────────────────────────────

type RingBuffer[T any] struct {
	ch chan T
}

func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{ch: make(chan T, capacity)}
}

// TrySend attempts a non-blocking send. Returns false if full.
func (r *RingBuffer[T]) TrySend(v T) bool {
	select {
	case r.ch <- v:
		return true
	default:
		return false
	}
}

// SendTimeout sends within deadline. Returns false on timeout.
func (r *RingBuffer[T]) SendTimeout(v T, d time.Duration) bool {
	select {
	case r.ch <- v:
		return true
	case <-time.After(d):
		return false
	}
}

// Recv receives one item, blocking until available.
func (r *RingBuffer[T]) Recv() (T, bool) {
	v, ok := <-r.ch
	return v, ok
}

// Len returns the number of items currently in the buffer.
func (r *RingBuffer[T]) Len() int { return len(r.ch) }

// Cap returns the capacity of the buffer.
func (r *RingBuffer[T]) Cap() int { return cap(r.ch) }

// Close signals no more sends. Consumers can drain via Drain().
func (r *RingBuffer[T]) Close() { close(r.ch) }

// Drain returns all remaining items after Close() has been called.
func (r *RingBuffer[T]) Drain() []T {
	var out []T
	for v := range r.ch {
		out = append(out, v)
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 1 — backpressure: slow consumer, fast producer
// ─────────────────────────────────────────────────────────────────────────────

func demoBackpressure() {
	fmt.Println("=== Backpressure ===")

	buf := NewRingBuffer[int](3)
	var wg sync.WaitGroup

	// Producer: tries to send 10 items as fast as possible.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer buf.Close()
		sent, dropped := 0, 0
		for i := range 10 {
			if buf.TrySend(i) {
				sent++
			} else {
				dropped++ // buffer full — backpressure applied
			}
			time.Sleep(2 * time.Millisecond)
		}
		fmt.Printf("  producer: sent=%d dropped=%d\n", sent, dropped)
	}()

	// Consumer: reads slowly.
	wg.Add(1)
	go func() {
		defer wg.Done()
		received := 0
		for {
			_, ok := buf.Recv()
			if !ok {
				break
			}
			received++
			time.Sleep(5 * time.Millisecond) // slower than producer
		}
		fmt.Printf("  consumer: received=%d\n", received)
	}()

	wg.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 2 — timeout-based send (do not block if consumer is stalled)
// ─────────────────────────────────────────────────────────────────────────────

func demoSendTimeout() {
	fmt.Println()
	fmt.Println("=== Send with timeout ===")

	buf := NewRingBuffer[string](2)
	buf.TrySend("item-1")
	buf.TrySend("item-2") // buffer now full

	// This send will time out because the buffer is full and nobody is reading.
	ok := buf.SendTimeout("item-3", 10*time.Millisecond)
	fmt.Printf("  send item-3 (full buffer, 10ms timeout): ok=%v\n", ok)

	// Drain one item to make room.
	v, _ := buf.Recv()
	fmt.Printf("  drained: %q\n", v)

	// Now the timeout send should succeed.
	ok = buf.SendTimeout("item-3", 10*time.Millisecond)
	fmt.Printf("  send item-3 (after drain, 10ms timeout): ok=%v\n", ok)

	buf.Close()
	fmt.Printf("  remaining: %v\n", buf.Drain())
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 3 — producer/consumer pipeline with metrics
// ─────────────────────────────────────────────────────────────────────────────

func demoPipeline() {
	fmt.Println()
	fmt.Println("=== Pipeline with metrics ===")

	input := NewRingBuffer[int](5)
	output := NewRingBuffer[int](5)

	// Stage 1: generate numbers.
	go func() {
		defer input.Close()
		for i := range 12 {
			input.ch <- i
		}
	}()

	// Stage 2: square numbers.
	go func() {
		defer output.Close()
		for v := range input.ch {
			output.ch <- v * v
		}
	}()

	// Collect results.
	results := output.Drain()
	fmt.Printf("  squares: %v\n", results)
	fmt.Printf("  count: %d\n", len(results))
}

func main() {
	demoBackpressure()
	demoSendTimeout()
	demoPipeline()
}
