# Chapter 41 — Concurrency Mental Model

> **Part IV · Concurrency & Systems** | Estimated reading time: 18 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Go's concurrency story is different. Most languages bolt concurrency on top of an existing threading model. Go built it in from day one, shaped by Tony Hoare's CSP paper (1978). Before writing a single `go` keyword, you need the right mental model — otherwise you'll write Go that looks like Java with channels.

---

## 41.1 — Concurrency vs Parallelism

These are not the same thing:

- **Concurrency** — structure. Multiple independent tasks that can overlap in time, possibly on one CPU.
- **Parallelism** — execution. Tasks that literally run at the same instant on multiple CPUs.

A single-CPU machine can be concurrent (time-sliced) but not parallel. A multi-CPU machine with a sequential program is parallel in hardware but not concurrent in structure.

Go's scheduler is concurrent by design. Whether your program is also parallel depends on `GOMAXPROCS` (defaults to the number of logical CPUs).

---

## 41.2 — CSP: Communicating Sequential Processes

Go's concurrency model is based on CSP. The core idea:

> **Don't communicate by sharing memory. Share memory by communicating.**

In practice: instead of having multiple goroutines lock/unlock a shared variable, have **one goroutine own the state** and let others send messages to it through a channel.

| Shared memory model | CSP model |
|---|---|
| Multiple goroutines access shared data | One goroutine owns data; others message it |
| Requires locks to prevent corruption | Channel ops provide synchronisation |
| Deadlock risk grows with lock count | Simpler reasoning about ownership |
| Any goroutine can corrupt state | State mutation is single-threaded |

---

## 41.3 — Goroutines

A goroutine is a lightweight independently-scheduled function. It is not a thread — the Go runtime multiplexes thousands of goroutines onto a small pool of OS threads.

```go
go f()           // start f concurrently; returns immediately
go func() { ... }() // anonymous goroutine
```

Key properties:
- Starts with a 2–8 KB stack (grows/shrinks as needed, up to 1 GB by default).
- Cheaper than an OS thread by ~100×.
- The runtime parks blocked goroutines (channel wait, sleep, I/O) and switches to runnable ones.

---

## 41.4 — Channels

A channel is a typed, directional, synchronising message pipe:

```go
ch := make(chan int)       // unbuffered: send blocks until recv
ch := make(chan int, 10)   // buffered: send blocks only when full

ch <- 42        // send
v := <-ch       // receive
v, ok := <-ch   // receive with close check
close(ch)       // signal no more sends
```

**Unbuffered channel** — sender and receiver rendezvous. The send blocks until someone receives. This is a synchronisation point.

**Buffered channel** — sender doesn't block until the buffer is full. Receiver doesn't block while buffer is non-empty.

---

## 41.5 — Pipelines

The fundamental CSP pattern: each stage is a goroutine that reads from an input channel and writes to an output channel.

```go
func generate(n int) <-chan int { ... }
func square(in <-chan int) <-chan int { ... }
func filter(in <-chan int, pred func(int) bool) <-chan int { ... }

result := collect(filter(square(generate(8)), isEven))
```

Each stage runs concurrently. Data flows through channels. No shared state.

---

## 41.6 — Happens-before

The Go memory model defines when a write in one goroutine is guaranteed to be visible in another. The key rule for channels:

> A **send** on a channel happens-before the corresponding **receive** from that channel completes.

This means: everything the sender did before the send is guaranteed visible to the receiver after the receive.

```go
data := ""
ready := make(chan struct{})

go func() {
    data = "set"   // happens-before the send
    ready <- struct{}{}
}()

<-ready            // happens-after the send → data is guaranteed visible
fmt.Println(data)  // safe: prints "set"
```

Without this synchronisation (e.g., just starting the goroutine and reading `data` immediately), you have a data race.

---

## 41.7 — Data races

A data race occurs when two goroutines access the same memory concurrently and at least one is a write, with no synchronisation between them.

Data races are undefined behaviour in Go — the program may crash, return wrong results, or appear to work.

Always run with `-race` during development and CI:

```bash
go run -race ./...
go test -race ./...
```

---

## Running the examples

```bash
cd book/part4_concurrency_systems/chapter41_concurrency_mental_model

go run ./examples/01_csp_goroutines    # CSP vs shared-memory, pipeline, fire-and-collect
go run ./examples/02_cooperative_model # concurrency vs parallelism, yield points, happens-before, atomic

go run ./exercises/01_concurrent_counter # mutex vs actor vs atomic — all correct
```

---

## Key takeaways

1. **Concurrency ≠ parallelism** — concurrency is structure, parallelism is execution.
2. **CSP rule** — one goroutine owns state; others communicate via channels, not shared memory.
3. **Goroutines are cheap** — thousands is normal; each starts at ~2 KB stack.
4. **Channels synchronise** — an unbuffered channel send/receive is a rendezvous and a happens-before point.
5. **Pipelines** — compose goroutines with channels; each stage runs concurrently with zero shared state.
6. **-race flag** — always use it in development and CI.

---

## Cross-references

- **Chapter 42** — Goroutines: Internals: G/M/P scheduler model
- **Chapter 43** — Channels: Internals: hchan structure, send/receive paths
- **Chapter 45** — sync Primitives: when shared memory + mutex is the right choice
