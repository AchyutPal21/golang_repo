# Chapter 85 Exercises — Benchmarking

## Exercise 1 — Optimisation Challenge (`exercises/01_optimize`)

Four functions each have a slow and fast implementation. Benchmark both, report the speedup, and explain the cause.

### Functions to benchmark

**1. JSON builder** — `BuildJSONSlow` vs `BuildJSONFast`
- Slow: string `+` concatenation per key-value pair
- Fast: `strings.Builder` with `Grow` hint
- Expected: ~5x speedup, ~40x fewer allocs

**2. Duplicate filter** — `DeduplicateSlow` vs `DeduplicateFast`
- Slow: O(n²) nested scan
- Fast: O(n) with `map[string]struct{}`
- Expected: grows faster with input size

**3. CSV parser** — `ParseCSVSlow` vs `ParseCSVFast`
- Slow: no pre-allocation
- Fast: `make([][]string, 0, len(lines))` pre-allocation
- Expected: modest improvement; main gain is in allocation count

**4. Cache reads** — `LockedCache` (Mutex) vs `RWCache` (RWMutex)
- Single-goroutine read: similar performance
- Concurrent reads (10 goroutines): RWMutex wins significantly
- Shows the difference only appears under contention

### Benchmark structure

```go
type Result struct {
    Name        string
    NsPerOp     float64
    AllocsPerOp uint64
    BytesPerOp  uint64
}

func bench(name string, fn func()) Result { ... }
func speedup(slow, fast Result) string { ... }
```

### Hints

- For JSON builder: measure with 20 fields — shows dramatic allocation reduction
- For dedup: use 200 items with 50% duplicates to make the O(n²) cost visible
- For CSV: parse 500 lines — the allocation difference becomes measurable
- The cache RWMutex benefit only shows under concurrent access — add a goroutine test
