# Chapter 46 — Exercises

## 46.1 — Lock-free metrics registry

Run [`exercises/01_metrics`](exercises/01_metrics/main.go).

`Counter`, `Gauge`, `Registry` with atomic snapshot export, `RateCounter`, and CAS-based max tracker.

Try:
- Add a `Histogram` type that tracks min, max, and sum using three `atomic.Int64` fields. Update min/max with CAS loops. Verify correctness under 1000 concurrent observations.
- Add `Reset() Snapshot` to `Registry` that atomically resets all counters to zero and returns their last values (useful for delta-reporting to a metrics backend).
- Run `go run -race ./exercises/01_metrics` and confirm no races.

## 46.2 ★ — Lock-free stack

Implement a lock-free stack using `atomic.Pointer[node[T]]` and a CAS loop:

```go
type Stack[T any] struct{ top atomic.Pointer[node[T]] }
type node[T any] struct { val T; next *node[T] }

func (s *Stack[T]) Push(v T)
func (s *Stack[T]) Pop() (T, bool)
func (s *Stack[T]) Len() int
```

Test with 1000 concurrent pushes and 1000 concurrent pops. Verify no value is lost or duplicated.

## 46.3 ★★ — Seqlock

Implement a seqlock — a synchronisation primitive that allows multiple concurrent readers without blocking writers, using sequence numbers:

- A writer increments a sequence counter (odd = writing), updates data, then increments again (even = done).
- A reader reads the sequence, reads data, reads sequence again. If both reads are even and equal, the read is consistent; otherwise retry.

Use `atomic.Uint64` for the sequence. Demonstrate correctness with concurrent readers and a writer that updates 10 times per second.
