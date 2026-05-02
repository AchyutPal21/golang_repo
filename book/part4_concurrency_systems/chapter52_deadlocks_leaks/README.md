# Chapter 52 — Deadlocks, Leaks

## What you will learn

- What a deadlock is and how the Go runtime detects all-goroutine deadlocks
- The four common deadlock patterns: lock ordering, channel symmetry, self-deadlock, livelock
- Consistent lock ordering by account ID (or address) to prevent circular waits
- The "unlocked helper" convention for functions that assume a lock is held
- What a goroutine leak is and how to detect one with `runtime.NumGoroutine`
- Four common leak patterns: blocked receiver, blocked sender, ticker without Stop, context not propagated
- How to use context cancellation as an escape hatch for every blocking goroutine

---

## Deadlock: the four patterns

### Lock-order deadlock

Two goroutines acquire the same two mutexes in opposite order:

```
goroutine A: lock(mu1) → lock(mu2)
goroutine B: lock(mu2) → lock(mu1)   ← deadlock when A holds mu1, B holds mu2
```

**Fix**: always acquire locks in a canonical order (e.g., ascending by ID or `uintptr(unsafe.Pointer(&mu))`).

### Channel deadlock

A send has no corresponding receiver (or vice versa):

```go
ch := make(chan int)
ch <- 1 // blocks forever — fatal error: all goroutines are asleep
```

**Fix**: ensure every send has a receiver goroutine, or use a buffered channel large enough to hold the send without blocking.

### Self-deadlock

A function acquires a mutex, then calls another function that tries to acquire the same mutex:

```go
func (s *S) Outer() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.Inner() // ← Inner also calls s.mu.Lock() — deadlock
}
```

**Fix**: extract `innerLocked()` helpers that are only called while the lock is held; never call them from outside the locked section.

### Livelock

Goroutines are not blocked but constantly retry and fail to make progress. Unlike a deadlock, the CPU is busy. **Fix**: randomised exponential backoff or a designated leader.

---

## Goroutine leak

A goroutine leak occurs when a goroutine is created but never exits — it stays alive, consuming memory and a stack slot, for the lifetime of the process.

### Detecting leaks

```go
before := runtime.NumGoroutine()
// ... call the suspect code ...
after := runtime.NumGoroutine()
if after > before {
    fmt.Printf("leaked %d goroutines\n", after-before)
}
```

In tests, use [goleak](https://github.com/uber-go/goleak):
```go
defer goleak.VerifyNone(t)
```

### Common patterns and fixes

| Leak | Cause | Fix |
|---|---|---|
| Blocked receiver | Channel never closed or sent to | Pass `ctx` and select on `ctx.Done()` |
| Blocked sender | Channel full, nobody drains | Use a buffered channel or drain after cancel |
| Ticker not stopped | `time.NewTicker` running forever | `defer ticker.Stop()` |
| Context not propagated | Blocking call ignores ctx | Wrap blocking call in `select { case <-time.After(d): ... case <-ctx.Done(): return }` |

---

## The escape hatch rule

Every goroutine that blocks on a channel, a mutex, or a timer must have an escape hatch — a way to exit when the context is cancelled:

```go
go func() {
    select {
    case v := <-ch:
        process(v)
    case <-ctx.Done(): // escape hatch
        return
    }
}()
```

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_deadlock_patterns/main.go` | Lock ordering, channel symmetry, self-deadlock, livelock |
| `examples/02_goroutine_leaks/main.go` | Blocked receiver/sender, ticker, context-unaware call + fixes |

## Exercise

`exercises/01_leak_finder/main.go` — four components with labelled `TODO LEAK` comments; fix each and verify goroutine count returns to baseline.
