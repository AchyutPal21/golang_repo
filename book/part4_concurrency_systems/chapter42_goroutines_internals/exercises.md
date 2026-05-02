# Chapter 42 — Exercises

## 42.1 — Bounded goroutine pool

Run [`exercises/01_goroutine_pool`](exercises/01_goroutine_pool/main.go).

A `Pool` with configurable worker count, job submission, result collection, graceful close, and immediate shutdown via done channel.

Try:
- Add a `Metrics()` method that returns `(submitted, processed, failed int)` counts using atomic counters.
- Add a `WithTimeout(d time.Duration) Option` that cancels the entire pool after duration `d` regardless of pending jobs. Use `time.AfterFunc` to close the done channel.
- Run `go run -race ./exercises/01_goroutine_pool` and confirm no data race is reported.

## 42.2 ★ — Leak detector

Write a `LeakChecker` helper for tests:

```go
type LeakChecker struct{ before int }
func NewLeakChecker() LeakChecker { return LeakChecker{runtime.NumGoroutine()} }
func (c LeakChecker) Check(t *testing.T) {
    time.Sleep(10 * time.Millisecond)
    after := runtime.NumGoroutine()
    if after > c.before {
        t.Errorf("goroutine leak: started with %d, now %d", c.before, after)
    }
}
```

Write three test functions: one that passes (no leak), one that leaks a goroutine blocked on a channel, and one that leaks due to a missing `close(done)`. Observe `LeakChecker.Check` catching the leaks.

## 42.3 ★★ — Bounded semaphore

Implement `Semaphore` backed by a buffered channel that provides `Acquire()` and `Release()`. Use it to limit a fan-out to at most 3 concurrent goroutines fetching URLs (simulated with `time.Sleep`). Verify with `runtime.NumGoroutine()` that the peak count never exceeds `3 + baseline`.
