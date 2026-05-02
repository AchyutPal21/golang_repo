# Chapter 45 — sync Primitives

> **Part IV · Concurrency & Systems** | Estimated reading time: 20 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Channels are not always the right tool. When you have a shared data structure that is read far more often than written, or when you need one-time initialisation, or when you want to reuse objects to reduce GC pressure, the `sync` package primitives give you the right abstraction.

---

## 45.1 — sync.Mutex

An exclusive lock. Exactly one goroutine can hold it at a time.

```go
type Counter struct {
    mu    sync.Mutex
    value int
}

func (c *Counter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.value++
}
```

Rules:
- Zero value is an unlocked, usable mutex — no constructor needed.
- **Never copy** a mutex after first use (this includes structs containing one).
- Use `defer mu.Unlock()` immediately after `Lock()` — ensures unlock on every return path.
- `TryLock()` (Go 1.18+) returns `false` instead of blocking if the lock is held.

---

## 45.2 — sync.RWMutex

Allows multiple concurrent readers **or** one exclusive writer.

```go
var mu sync.RWMutex

// Read — multiple goroutines can hold RLock simultaneously:
mu.RLock()
defer mu.RUnlock()
return m[key]

// Write — exclusive:
mu.Lock()
defer mu.Unlock()
m[key] = value
```

Use `RWMutex` when reads are far more frequent than writes and reads are long enough for the contention savings to matter. For short critical sections a plain `Mutex` is often faster due to lower overhead.

---

## 45.3 — Lock ordering

When acquiring multiple mutexes, always acquire them in the same order across all goroutines. Inconsistent ordering → deadlock.

```go
// Always lock lower ID first:
if a.id < b.id {
    a.mu.Lock(); b.mu.Lock()
} else {
    b.mu.Lock(); a.mu.Lock()
}
```

---

## 45.4 — sync.Cond

A condition variable: goroutines wait for a condition, and other goroutines signal when the condition changes.

```go
cond := sync.NewCond(&mu)

// Waiter:
mu.Lock()
for !conditionMet() {
    cond.Wait()   // atomically unlocks mu and suspends
}
// condition is true here, mu is locked
mu.Unlock()

// Signaller:
mu.Lock()
makeConditionTrue()
cond.Signal()    // wake one waiter
// or cond.Broadcast() to wake all
mu.Unlock()
```

`Wait` must be called in a `for` loop — spurious wakeups are possible.

---

## 45.5 — sync.Once

Runs a function exactly once, even under concurrent access:

```go
var once sync.Once
var conn *DB

func getConn() *DB {
    once.Do(func() {
        conn = openDB()    // runs exactly once
    })
    return conn
}
```

Zero value is ready to use. Cannot be reset — create a new `Once` if you need to re-run.

---

## 45.6 — sync.Pool

A thread-safe free list. `Get` returns a pooled object (or calls `New` if empty). `Put` returns an object for reuse. Objects may be evicted by the GC at any time.

```go
var pool = sync.Pool{
    New: func() any { return &bytes.Buffer{} },
}

buf := pool.Get().(*bytes.Buffer)
buf.Reset() // always reset before use
defer pool.Put(buf)
// ... use buf
```

Ideal for: allocating scratch buffers, encoder/decoder objects, or any short-lived allocation that appears in a hot path.

---

## 45.7 — sync.WaitGroup

Waits for a group of goroutines to finish:

```go
var wg sync.WaitGroup
wg.Add(n)               // add BEFORE launching goroutines
for i := 0; i < n; i++ {
    go func() {
        defer wg.Done()
        // work
    }()
}
wg.Wait()
```

**Never** call `Add` inside the goroutine — it races with `Wait`.

---

## Running the examples

```bash
cd book/part4_concurrency_systems/chapter45_sync_primitives

go run ./examples/01_mutex_rwmutex   # Mutex, RWMutex, lock ordering, TryLock
go run ./examples/02_cond_once_pool  # Cond, Once, Pool, WaitGroup

go run ./exercises/01_cache          # TTL cache with RWMutex + LazyLoader with Once
```

---

## Key takeaways

1. **Mutex** — exclusive lock; use `defer Unlock()` immediately after `Lock()`; never copy.
2. **RWMutex** — multiple readers or one writer; prefer when reads >> writes.
3. **Lock ordering** — always acquire multiple mutexes in the same order to prevent deadlock.
4. **Cond** — wait in a `for` loop, not an `if`; `Broadcast` wakes all, `Signal` wakes one.
5. **Once** — guarantees exactly one execution; zero value is ready; cannot be reset.
6. **Pool** — reduces GC pressure for short-lived objects; always `Reset` before use; objects may disappear.
7. **WaitGroup** — `Add` before launching goroutines; `Done` deferred in each goroutine.

---

## Cross-references

- **Chapter 46** — sync/atomic: lower-level, lock-free primitives for simple counters and flags
- **Chapter 41** — Concurrency Mental Model: when to use sync vs channels
- **Chapter 52** — Deadlocks, Leaks: diagnosing lock-related deadlocks
