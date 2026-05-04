# Chapter 88 — GC & Escape Analysis

Understanding when values escape to the heap — and how to tune the GC —
is essential for keeping latency predictable in production Go services.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | Escape analysis | Causes, compiler flags, measurement |
| 2 | GC tuning | GOGC, GOMEMLIMIT, gctrace, finalizers |
| E | GC pressure | Before/after comparison in HTTP pipeline |

## Examples

### `examples/01_escape_analysis`

Covers every common escape cause with measured allocation deltas:

- Return `*T` vs return `T` — heap vs stack
- Storing into `interface{}` — boxing overhead
- Closure capturing a local — frame lifetime issue
- Fixed `[N]T` vs `make([]T, n)` — compile-time vs runtime sizing
- `fmt.Println` forcing escape vs `strconv.AppendInt`
- Reading `-gcflags="-m"` output — "moved to heap" vs "escapes to heap"

### `examples/02_gc_tuning`

Demonstrates runtime GC control:

- `runtime.MemStats` — HeapAlloc, PauseTotalNs, NumGC
- `debug.SetGCPercent` — GOGC experiment (50/100/200/400)
- `debug.SetMemoryLimit` — GOMEMLIMIT programmatic equivalent
- Finalizer lifecycle — SetFinalizer, explicit Close, nil removal
- `GODEBUG=gctrace=1` output reference with field annotations
- `net/http/pprof` heap profiling workflow

### `exercises/01_gc_pressure`

Refactor a simulated HTTP middleware pipeline to reduce GC pressure:

- `parsePathFast` — caller-provided `[]string`, no allocation
- `buildLogLineFast` — `strconv.AppendInt` into caller buffer
- Channel-based `Response` pool with Reset on return
- Before/after: GC cycle count and total pause time

## Key Concepts

**Five escape causes**
1. Returning a pointer to a local variable
2. Storing a value into `interface{}`
3. Closure capturing a local variable
4. `make([]T, n)` where `n` is not a compile-time constant
5. Passing arguments to variadic `...any` (e.g. `fmt.Println`)

**GOGC trade-off**

```
GOGC=100 (default) → GC when heap doubles
GOGC=200           → 2× less GC, 2× more memory
GOGC=50            → 2× more GC, 50% less memory
GOMEMLIMIT=Xmib    → hard memory cap (prefer over very low GOGC)
```

**Finalizer rules**
- Non-deterministic — never use for correctness
- Always provide `Close()` / `Destroy()` alongside
- Call `runtime.SetFinalizer(r, nil)` inside `Close` to disarm
- Cyclic references prevent the finalizer from ever firing

## Running

```bash
go run ./part6_production_engineering/chapter88_gc_escape/examples/01_escape_analysis
go run ./part6_production_engineering/chapter88_gc_escape/examples/02_gc_tuning
go run ./part6_production_engineering/chapter88_gc_escape/exercises/01_gc_pressure
# See compiler escape decisions:
go build -gcflags="-m" ./part6_production_engineering/chapter88_gc_escape/examples/01_escape_analysis/
```
