# Chapter 52 — Exercises

## 52.1 — Leak finder

Run [`exercises/01_leak_finder`](exercises/01_leak_finder/main.go).

Four components with labelled `TODO LEAK` comments: `StartPoller`, `HandleRequest`, `FanOut`, `EventStream.Subscribe`. The baseline prints 8 leaked goroutines above the base count.

Expected fixes:
- Leak 1: add `done <-chan struct{}` and `defer ticker.Stop()`
- Leak 2: make `ch` buffered (cap 1) so the work goroutine can send even after the caller times out
- Leak 3: make `results` buffered (cap `n`) so all fan-out goroutines can send
- Leak 4: add a `done chan struct{}` to `Subscribe`; close it to stop the goroutine

After all fixes: `total leaked goroutines (above base): 0`.

## 52.2 ★ — Detect deadlock potential in existing code

Write a function `LockCheck` that verifies a lock-ordering invariant:

```go
type LockOrder struct{ mu sync.Mutex }

func (lo *LockOrder) Acquire(id int) (unlock func())
// Records the order each goroutine acquires locks.
// After the test: lo.Violations() returns pairs (A, B) where A was
// acquired while holding B in some goroutine, and B was acquired while
// holding A in another goroutine — a potential deadlock cycle.
```

Use `sync.Map` to track per-goroutine lock stacks (goroutine ID via `runtime.Stack`). Test with the crossing pattern from `examples/01_deadlock_patterns`.

## 52.3 ★★ — goleak integration

Add `TestNoLeaks(t *testing.T)` that uses `goleak.VerifyNone(t)` to verify the main scenarios from `exercises/01_leak_finder`:
1. `StartPoller` with a proper `done` channel — assert no leak
2. `HandleRequest` with a 10ms ctx and 100ms work — assert no leak after cancel
3. `FanOut` with a buffered channel — assert no leak

This requires the `go.uber.org/goleak` package; add it with `go get go.uber.org/goleak`.
