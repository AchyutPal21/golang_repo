# Chapter 43 — Exercises

## 43.1 — Generic ring buffer

Run [`exercises/01_ring_buffer`](exercises/01_ring_buffer/main.go).

A generic `RingBuffer[T]` backed by a buffered channel with `TrySend`, `SendTimeout`, `Drain`, and backpressure semantics.

Try:
- Add a `Stats()` method that returns `(len, cap, dropped int)` using atomic counters to track dropped items across all `TrySend` calls.
- Add a `RecvTimeout(d time.Duration) (T, bool)` method that returns the zero value and `false` if no item arrives within `d`.
- Run `go run -race ./exercises/01_ring_buffer` and confirm no data race is reported.

## 43.2 ★ — Multi-producer single-consumer

Implement a pattern where 5 producer goroutines each send 20 integers to a shared buffered channel (capacity 10), and a single consumer goroutine reads all 100 values and computes their sum. Use a `sync.WaitGroup` on the producer side and close the channel only after all producers finish (hint: use a coordinator goroutine).

## 43.3 ★★ — Typed event bus via channels

Build an `EventBus[T]` that:
- Has a `Publish(event T)` method (non-blocking, drops if no subscriber is ready)
- Has a `Subscribe() <-chan T` method that returns a per-subscriber channel
- Has an `Unsubscribe(ch <-chan T)` method to remove a subscriber
- Uses a single goroutine internally to fan-out events to all subscribers

Test with 3 subscribers, publish 10 events, verify each subscriber receives all 10.
