# Chapter 42 — Goroutines: Internals

> **Part IV · Concurrency & Systems** | Estimated reading time: 20 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Goroutines feel like magic until you understand the scheduler. Once you know the G/M/P model you can reason about performance, diagnose leaks, and avoid common pitfalls like the loop-variable capture trap.

---

## 42.1 — The G/M/P model

The Go scheduler multiplexes thousands of goroutines onto a small pool of OS threads using three abstractions:

| Symbol | Name | What it is |
|---|---|---|
| G | Goroutine | Unit of work: goroutine struct + stack |
| M | Machine | OS thread — the actual execution vehicle |
| P | Processor | Scheduling context: holds a run queue of Gs |

**Rules:**
- An M must hold a P to run Go code.
- `GOMAXPROCS` = number of Ps = maximum parallelism.
- When M blocks on a syscall, it releases P so another M can acquire it.
- When the syscall returns, M tries to reclaim its P or steal an idle one.

---

## 42.2 — Goroutine stack

Every goroutine has its own stack, separate from the OS thread stack:

- Starts at **2 KB** (64-bit, Go 1.4+).
- Grows dynamically by copying to a new, larger allocation when needed.
- Shrinks when the goroutine returns from deep frames.
- Maximum stack size defaults to **1 GB** (configurable with `runtime/debug.SetMaxStack`).

Stack growth is automatic and invisible to the programmer — no stack overflow panics for reasonable recursion depths.

---

## 42.3 — Work stealing

Each P has a local run queue. When a P's queue is empty it steals half the goroutines from another P's queue. This keeps all Ps busy without a central lock on a global queue.

```
P0 (run queue: G1 G2 G3 G4)   P1 (run queue: empty)
→ P1 steals G3 G4 from P0
P0 (run queue: G1 G2)          P1 (run queue: G3 G4)
```

---

## 42.4 — Preemption

Since Go 1.14, goroutines are **asynchronously preemptible** — the runtime can stop a goroutine at any safe point (function call, memory allocation), not just at explicit yield points. This prevents a CPU-bound goroutine from starving others.

Before 1.14: preemption only happened at function calls. A tight loop with no function calls could monopolise a P.

---

## 42.5 — Goroutine leaks

A goroutine that is blocked on a channel with no sender (or vice versa) and has no other exit path runs forever. Leaks accumulate memory and scheduler slots.

**Every goroutine must have a guaranteed exit path:**

```go
// Leak — nobody sends to ch
go func() { <-ch }()

// Clean — done channel provides an exit
go func() {
    select {
    case <-ch:
    case <-done: // caller closes this when finished
    }
}()
```

Use `runtime.NumGoroutine()` in tests to detect leaks.

---

## 42.6 — Loop-variable capture

In **Go < 1.22**, the loop variable is a single address reused each iteration. Goroutines that close over it see the value at the time they run — usually the final value.

```go
// Go < 1.22 — BUG: all goroutines print 5
for i := 0; i < 5; i++ {
    go func() { fmt.Println(i) }()
}

// Workaround — pass as argument (works in all Go versions)
for i := 0; i < 5; i++ {
    go func(n int) { fmt.Println(n) }(i)
}
```

**Go 1.22+** gives each loop iteration its own variable — the bug is gone and no workaround is needed.

---

## 42.7 — errgroup pattern

`sync.WaitGroup` tracks completion but can't collect errors. The standard solution is `golang.org/x/sync/errgroup`:

```go
var g errgroup.Group
for _, url := range urls {
    url := url
    g.Go(func() error {
        return fetch(url)
    })
}
if err := g.Wait(); err != nil {
    // first error from any goroutine
}
```

---

## Running the examples

```bash
cd book/part4_concurrency_systems/chapter42_goroutines_internals

go run ./examples/01_gmp_scheduler    # G/M/P, stack growth, work stealing, syscall handoff
go run ./examples/02_goroutine_patterns # leaks, loop capture, errgroup, creation cost

go run ./exercises/01_goroutine_pool  # bounded pool with done channel and error collection

# Observe scheduler internals:
GODEBUG=schedtrace=200 go run ./examples/01_gmp_scheduler
```

---

## Key takeaways

1. **G/M/P** — goroutines (G) run on OS threads (M) via scheduling contexts (P). `GOMAXPROCS` = number of Ps.
2. **Stacks grow automatically** — goroutines start at 2 KB and grow as needed; no programmer action required.
3. **Work stealing** — idle Ps steal from busy Ps; all CPUs stay utilized.
4. **Preemption** — since Go 1.14, goroutines are preemptible at any function call; no goroutine can starve others.
5. **Leaks** — always give goroutines a done-channel exit path.
6. **Loop capture** — Go 1.22 fixed the loop-variable capture bug; for older code use argument passing.

---

## Cross-references

- **Chapter 41** — Concurrency Mental Model: CSP, channels, happens-before
- **Chapter 43** — Channels: Internals: hchan, send/receive scheduler interaction
- **Chapter 52** — Deadlocks, Leaks: detecting leaks with pprof goroutine profiles
