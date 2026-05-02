# Chapter 43 — Channels: Internals

> **Part IV · Concurrency & Systems** | Estimated reading time: 20 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Channels are Go's primary synchronisation primitive. Understanding what happens inside `hchan` — the blocking queues, the buffer ring, the close flag — lets you write correct channel code without superstition and diagnose subtle bugs like deadlocks and panics.

---

## 43.1 — hchan anatomy

Every channel is backed by a runtime struct `hchan`:

```
hchan {
    buf      *[cap]T   // circular ring buffer (nil for unbuffered)
    sendx    uint      // send index into buf
    recvx    uint      // receive index into buf
    qcount   uint      // items currently in buf
    dataqsiz uint      // capacity (cap(ch))
    sendq    waitq     // list of blocked senders
    recvq    waitq     // list of blocked receivers
    closed   uint32    // set by close()
    lock     mutex     // internal lock
}
```

`sendq` and `recvq` are linked lists of goroutines parked waiting for the channel to become available.

---

## 43.2 — Buffered vs unbuffered

| | Unbuffered (cap=0) | Buffered (cap=N) |
|---|---|---|
| Send blocks? | Until a receiver is ready | Until buffer is full |
| Receive blocks? | Until a sender is ready | Until buffer is non-empty |
| Synchronisation? | Yes — rendezvous | Only at full/empty boundaries |
| Use case | Handoff, synchronisation | Decoupling, burst absorption |

```go
ch := make(chan int)     // unbuffered
ch := make(chan int, 10) // buffered, capacity 10
```

---

## 43.3 — Closing rules

| Operation | Result |
|---|---|
| `close(ch)` | Sets closed flag; wakes all receivers in recvq |
| Send to closed | **panic** |
| Receive from closed, buffer non-empty | Returns buffered value, `ok=true` |
| Receive from closed, buffer empty | Returns zero value, `ok=false` |
| Close already-closed | **panic** |

**Rule:** only the sender (the goroutine responsible for writing) should close the channel. Closing from the receiver side is unsafe if multiple senders exist.

```go
v, ok := <-ch   // ok=false means channel is closed and drained
for v := range ch { ... } // exits when channel is closed and drained
```

---

## 43.4 — Nil channels

Sending to or receiving from a `nil` channel blocks forever. In a `select`, a case on a nil channel is never selected.

This is useful to disable a case without restructuring the select:

```go
var ch chan int // nil
select {
case v := <-ch:   // never selected
case v := <-other:
}

// Disable a case mid-loop:
if done {
    ch = nil // this case won't fire any more
}
```

---

## 43.5 — Directional channels

Use directional types to enforce producer/consumer contracts at compile time:

```go
func produce(ch chan<- int) { ch <- 1 }   // send-only
func consume(ch <-chan int) { <-ch }       // receive-only

ch := make(chan int)
go produce(ch)   // bidirectional auto-converts to send-only
consume(ch)      // bidirectional auto-converts to receive-only
```

The compiler rejects reads from `chan<- T` and writes to `<-chan T`.

---

## 43.6 — Key patterns

**Done channel** — broadcast cancellation:
```go
done := make(chan struct{})
go func() {
    select {
    case <-done: return
    case v := <-work: process(v)
    }
}()
close(done) // wakes all goroutines blocked on <-done
```

**Semaphore** — cap-N buffered channel limits concurrency:
```go
sem := make(chan struct{}, 3)
sem <- struct{}{}  // acquire
<-sem              // release
```

**One-time signal** — `close` broadcasts to all waiters at once:
```go
ready := make(chan struct{})
close(ready)       // all <-ready unblock immediately and forever
```

**Ownership transfer** — sender must not touch value after send:
```go
ch <- buf   // ownership of buf transferred to receiver
// buf = nil  // good practice: make the transfer explicit
```

---

## Running the examples

```bash
cd book/part4_concurrency_systems/chapter43_channels_internals

go run ./examples/01_channel_mechanics # buffered/unbuffered, closing, range, nil, directional, hchan
go run ./examples/02_channel_patterns  # done channel, semaphore, ownership, start gun, channel-mutex

go run ./exercises/01_ring_buffer      # generic ring buffer with backpressure, timeout send, pipeline
```

---

## Key takeaways

1. **hchan** — channels are a ring buffer + two wait queues (sendq, recvq) + a lock.
2. **Unbuffered = rendezvous** — sender and receiver must meet; this is a happens-before point.
3. **Close rules** — only the sender closes; receive from closed returns zero + `ok=false` after draining.
4. **Nil channel = disabled case** — send/recv on nil blocks forever; useful in select to silence a case.
5. **Directional channels** — `chan<- T` (send-only) and `<-chan T` (receive-only) enforce contracts at compile time.
6. **One-time signal** — `close(ch)` broadcasts to all current and future receivers immediately.

---

## Cross-references

- **Chapter 41** — Concurrency Mental Model: CSP and why channels are the primary primitive
- **Chapter 44** — select / Timeouts / Cancel: composing multiple channels
- **Chapter 52** — Deadlocks, Leaks: diagnosing goroutines blocked on channels
