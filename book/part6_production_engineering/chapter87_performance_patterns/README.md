# Chapter 87 — Performance Patterns

High-performance Go requires minimizing allocations, reusing memory, and
keeping objects on the stack. This chapter covers three complementary
techniques that production Go services use to stay fast under load.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | `sync.Pool` | Object reuse, correct reset, GC interaction |
| 2 | Zero-allocation patterns | `strconv` vs `fmt`, `strings.Builder`, `unsafe` |
| E | Slab allocator | Fixed-pool, channel-backed, zero heap alloc |

## Examples

### `examples/01_sync_pool`

Demonstrates `sync.Pool` for `bytes.Buffer` and struct reuse:

- `acquireBuffer` / `releaseBuffer` — buffer lifecycle with cap guard
- `acquireRequest` / `releaseRequest` — struct zeroing before Put
- Side-by-side allocation measurement: 100 k iterations with/without pool
- GC-reset demonstration — pool objects disappear on any GC cycle

### `examples/02_zero_alloc`

Shows how to eliminate allocations in hot paths:

- `unsafe.Slice` / `unsafe.String` — string↔[]byte without copy
- `strconv.AppendInt` / `AppendFloat` — zero-alloc number formatting
- `strings.Builder` with `Grow()` — predictable string concatenation
- `make([]T, 0, n)` — pre-allocated slices
- Value vs pointer return — stack vs heap placement

### `exercises/01_buffer_slab`

Build a slab allocator backed by a channel of free indices:

- `SlabCount` fixed slots of `SlabSize` bytes each
- Lock-free acquire via `select` on buffered channel
- Slot zeroing on release to prevent data leakage
- `PacketProcessor` using the slab under concurrent load
- Comparison: slab vs per-call `make([]byte, SlabSize)`

## Key Concepts

**sync.Pool rules**
1. Always `Reset()` / zero the object *before* use, not before `Put`.
2. Never pool objects larger than ~64 KB — they waste RSS.
3. Pool is cleared on every GC — it is a reuse hint, not a cache.
4. Pool is most effective for frequent, short-lived, same-shape objects.

**Zero-alloc checklist**
- `strconv.AppendInt/Float/Bool` instead of `fmt.Sprintf` in hot paths
- `make([]T, 0, knownCap)` to avoid growth reallocations
- `strings.Builder` + `Grow` for building strings from pieces
- Small structs returned by value stay on the stack
- `unsafe` string/byte conversions only with provably immutable data

**Slab vs sync.Pool**
| Property | sync.Pool | Slab |
|----------|-----------|------|
| Max objects | Unlimited | Fixed (SlabCount) |
| Cleared by GC | Yes | No |
| Backpressure on overload | No | Yes (ErrSlabFull) |
| Best for | Generic objects | Fixed-size network buffers |

## Running

```bash
go run ./part6_production_engineering/chapter87_performance_patterns/examples/01_sync_pool
go run ./part6_production_engineering/chapter87_performance_patterns/examples/02_zero_alloc
go run ./part6_production_engineering/chapter87_performance_patterns/exercises/01_buffer_slab
```
