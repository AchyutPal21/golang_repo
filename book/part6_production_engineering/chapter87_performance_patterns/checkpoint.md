# Chapter 87 Checkpoint — Performance Patterns

Answer each question before moving to Chapter 88.

## Concept checks

1. Why must you `Reset()` a `bytes.Buffer` *after* calling `Get()`, not before
   calling `Put()`?

2. A `sync.Pool` has 1000 objects stored. A GC cycle runs. How many objects
   remain in the pool?

3. What is the fundamental difference between `fmt.Sprintf("%d", n)` and
   `strconv.AppendInt(buf, int64(n), 10)` in terms of allocations?

4. When is it safe to use `unsafe.String(&b[0], len(b))` to convert `[]byte`
   to `string` without copying?

5. Explain the trade-off between a slab allocator and `sync.Pool` when
   handling bursts of traffic beyond the slab's slot count.

## Code review

```go
type worker struct{ buf bytes.Buffer }

var pool = sync.Pool{
    New: func() any { return &worker{} },
}

func handle(data []byte) {
    w := pool.Get().(*worker)
    w.buf.Write(data)
    result := w.buf.String()
    pool.Put(w)
    _ = result
}
```

What is the bug? How do you fix it?

## Expected answers

1. If you Reset before Put you clear data that was already used (fine), but
   the *next* caller gets a clean object — this works. However the canonical
   pattern is to Reset *after* Get so the object is always clean when obtained,
   regardless of what the previous user did. Never reset before Put because the
   buffer may be used by another goroutine immediately after Put.

2. Zero — the GC clears all pool objects. This is by specification.

3. `fmt.Sprintf` always allocates a new `string`; `strconv.AppendInt` writes
   digits directly into the caller-supplied `[]byte` with zero heap allocation.

4. When the underlying `[]byte` is not mutated for the lifetime of the
   resulting `string`. The string header points directly into the byte slice's
   memory.

5. A slab provides deterministic backpressure — callers get `ErrSlabFull` and
   can shed load. `sync.Pool` has no upper bound; under a burst it calls `New`
   repeatedly, causing GC pressure. Slabs win for bounded-memory scenarios;
   pools win for convenience and unbounded workloads.

**Bug**: `w.buf` is not reset before use. After `Put`, the next caller's `Get`
returns a buffer with leftover bytes. Fix: `w.buf.Reset()` immediately after
`pool.Get()`.
