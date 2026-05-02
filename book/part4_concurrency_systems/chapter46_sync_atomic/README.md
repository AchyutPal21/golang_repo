# Chapter 46 — sync/atomic

> **Part IV · Concurrency & Systems** | Estimated reading time: 18 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

`sync/atomic` provides hardware-level atomic operations — single-instruction reads, writes, and compare-and-swap that require no locks and have no scheduler overhead. For counters, flags, and read-heavy shared pointers, they outperform mutexes by an order of magnitude.

---

## 46.1 — The typed API (Go 1.19+)

Go 1.19 introduced value types that embed the atomic operations:

| Type | Operations |
|---|---|
| `atomic.Int32`, `Int64` | `Load`, `Store`, `Add`, `Swap`, `CompareAndSwap` |
| `atomic.Uint32`, `Uint64` | same |
| `atomic.Bool` | `Load`, `Store`, `Swap`, `CompareAndSwap` |
| `atomic.Pointer[T]` | `Load`, `Store`, `Swap`, `CompareAndSwap` |
| `atomic.Value` | `Load`, `Store`, `Swap`, `CompareAndSwap` (any type) |

Prefer the typed API over the legacy package-level functions — it prevents alignment bugs and is harder to misuse.

---

## 46.2 — Basic operations

```go
var n atomic.Int64

n.Store(42)           // write
v := n.Load()         // read
old := n.Swap(100)    // replace, return old
n.Add(5)              // fetch-and-add

var flag atomic.Bool
flag.Store(true)
if flag.Load() { ... }
```

---

## 46.3 — CompareAndSwap (CAS)

The foundation of all lock-free algorithms:

```go
// Atomically: if current == old { current = new; return true }
//             else               { return false }
swapped := n.CompareAndSwap(old, new)
```

CAS-loop pattern for lock-free mutation:

```go
for {
    old := n.Load()
    if n.CompareAndSwap(old, transform(old)) {
        break  // success
    }
    // another goroutine changed n — retry
}
```

---

## 46.4 — atomic.Value — store any immutable value

```go
var cfg atomic.Value
cfg.Store(Config{...})       // first store determines the type
c := cfg.Load().(Config)     // type-assert; always succeeds if type is consistent
```

Rules:
- All `Store` calls must use the same concrete type.
- Values stored in `atomic.Value` should be **immutable** — never modify a stored value in place.
- `Load` returns `nil` if nothing has been stored yet.

Use case: hot-reloadable configuration, routing tables, feature flags — replace the whole value atomically with no read lock.

---

## 46.5 — atomic.Pointer[T]

Type-safe pointer swap (Go 1.19+):

```go
var ptr atomic.Pointer[RouteTable]
ptr.Store(&RouteTable{...})
rt := ptr.Load()                    // zero-copy, no lock
old := ptr.Swap(&newRouteTable)     // atomic replace
```

---

## 46.6 — When to use atomic vs mutex

| Situation | Use |
|---|---|
| Simple counter / flag | `atomic.Int64` / `atomic.Bool` |
| Read-heavy pointer swap | `atomic.Value` / `atomic.Pointer[T]` |
| Protecting a struct with multiple fields | `sync.Mutex` / `sync.RWMutex` |
| Complex invariant spanning multiple variables | `sync.Mutex` |
| Channel-based ownership transfer | channel |

Atomic operations are composable only within a single variable. If you need to update two variables as a unit, you need a mutex.

---

## Running the examples

```bash
cd book/part4_concurrency_systems/chapter46_sync_atomic

go run ./examples/01_atomic_ops    # Int64, Bool, CAS, legacy functions, spinlock illustration
go run ./examples/02_atomic_value  # atomic.Value hot-reload, hot-reload pattern, atomic.Pointer

go run ./exercises/01_metrics      # lock-free Counter/Gauge registry, rate counter, CAS max tracker
```

---

## Key takeaways

1. **Typed API** (Go 1.19+) — use `atomic.Int64`, `atomic.Bool`, `atomic.Pointer[T]` over raw functions.
2. **CAS** — the building block of lock-free algorithms; retry on failure with a loop.
3. **atomic.Value** — atomically replace any immutable value; all stores must use the same type.
4. **Immutability rule** — never modify a value after storing it in `atomic.Value` or `atomic.Pointer`.
5. **Scope** — atomics only protect a single variable; use mutex for multi-variable invariants.

---

## Cross-references

- **Chapter 45** — sync Primitives: when mutex is the right choice
- **Chapter 41** — Concurrency Mental Model: the Go memory model and happens-before
- **Chapter 51** — Race Detector: atomics prevent races; the detector confirms correctness
