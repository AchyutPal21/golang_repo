# Chapter 44 — select / Timeouts / Cancel

> **Part IV · Concurrency & Systems** | Estimated reading time: 18 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

`select` is the control-flow statement for concurrent Go. Every non-trivial concurrent program uses it: to wait on multiple channels, to implement timeouts, to check for cancellation, and to build non-blocking operations. Knowing `select` well means knowing how to compose goroutines without polling.

---

## 44.1 — select mechanics

`select` blocks until one of its cases is ready, then executes that case:

```go
select {
case v := <-ch1:   // receives from ch1
case ch2 <- x:     // sends x to ch2
case <-done:       // cancellation check
default:           // runs immediately if no other case is ready
}
```

Rules:
- If multiple cases are ready simultaneously, **one is chosen at random** (uniform pseudo-random).
- If no case is ready and there is a `default`, `default` runs immediately (non-blocking).
- If no case is ready and there is no `default`, `select` blocks until a case becomes ready.
- A nil channel case is never selected.

---

## 44.2 — Non-blocking operations

Add a `default` to make any channel operation non-blocking:

```go
// Non-blocking receive (try)
select {
case v := <-ch:
    process(v)
default:
    // nothing ready
}

// Non-blocking send (drop if full)
select {
case ch <- v:
default:
    // dropped
}
```

---

## 44.3 — Priority select

`select` is randomly fair — it does not honour case order. To implement priority:

```go
for {
    // Drain high-priority first
    select {
    case v := <-high:
        handle(v)
        continue
    default:
    }
    // Only process low when high is empty
    select {
    case v := <-high:
        handle(v)
    case v := <-low:
        handle(v)
    }
}
```

---

## 44.4 — time.After vs time.NewTimer

`time.After(d)` returns a channel that receives after `d`. Simple, but **leaks the timer** until it fires — avoid in tight loops.

`time.NewTimer(d)` returns a reusable `*Timer`. Use `Stop` + drain + `Reset` for safe reuse:

```go
timer := time.NewTimer(timeout)
defer timer.Stop()

// Safe reset pattern:
timer.Stop()
select { case <-timer.C: default: }
timer.Reset(timeout)
```

`time.NewTicker(d)` fires repeatedly every `d`. Always `defer ticker.Stop()`.

---

## 44.5 — Timeout patterns

**Per-call timeout:**
```go
select {
case v := <-work:
    return v, nil
case <-time.After(timeout):
    return "", ErrTimeout
}
```

**Overall deadline (shared across calls):**
```go
deadline := time.After(overallTimeout)
for _, op := range ops {
    select {
    case result := <-op():
        process(result)
    case <-deadline:
        return ErrDeadlineExceeded
    }
}
```

**Back-off between retries:**
```go
select {
case <-time.After(backoff):
case <-deadline:
    return ErrDeadline
}
backoff = min(backoff*2, maxBackoff)
```

---

## 44.6 — Heartbeat pattern

A worker goroutine sends on a heartbeat channel at a fixed interval, letting a supervisor detect stalls:

```go
go func() {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    for {
        select {
        case <-done: return
        case t := <-ticker.C:
            select { case heartbeat <- t: default: } // non-blocking
            doWork()
        }
    }
}()
```

The supervisor selects on both `results` and `heartbeat`, and sets a deadline if heartbeat stops arriving.

---

## Running the examples

```bash
cd book/part4_concurrency_systems/chapter44_select_timeouts_cancel

go run ./examples/01_select_basics     # select mechanics, default, priority, nil, send/recv mix
go run ./examples/02_timeout_patterns  # time.After, timer.Reset, ticker, deadline, heartbeat

go run ./exercises/01_retry_with_timeout # retry + per-attempt timeout + overall deadline + backoff
```

---

## Key takeaways

1. **select** waits on multiple channel operations; if multiple are ready, one is chosen at random.
2. **default** makes select non-blocking — use for try-send and try-receive.
3. **Priority** — use nested select with default to drain a high-priority channel first.
4. **Nil case** — a nil channel in select is never selected; use to disable a case dynamically.
5. **time.After leaks** in loops — use `time.NewTimer` + Stop + drain + Reset for safe reuse.
6. **Overall deadline** — create `time.After` once outside the loop; include it in every inner select.

---

## Cross-references

- **Chapter 43** — Channels: Internals: nil channels, buffered vs unbuffered
- **Chapter 47** — context Package: context.WithTimeout / WithDeadline build on these patterns
- **Chapter 52** — Deadlocks, Leaks: select with no ready cases and no default causes deadlock
