# Chapter 85 — Benchmarking

Go's `testing` package includes a first-class benchmark runner. Benchmarks measure the speed and allocation behaviour of your code so you can identify hotspots, compare alternatives, and prove that an optimisation actually helped.

## Writing a benchmark

```go
func BenchmarkConcat(b *testing.B) {
    parts := makeParts(100)
    b.ResetTimer()           // don't count setup in the measurement
    for b.Loop() {           // Go 1.24+; or i := 0; i < b.N; i++
        ConcatBuilder(parts)
    }
}
```

`b.N` is automatically tuned by the runner until the total time is stable.

## Measuring allocations

```go
b.ReportAllocs()  // report allocs/op and B/op
```

Or run with `-benchmem`:

```bash
go test -bench=BenchmarkConcat -benchmem ./...
# output: 312 ns/op   1 allocs/op   256 B/op
```

## Sub-benchmarks for input sizes

```go
func BenchmarkConcat(b *testing.B) {
    for _, n := range []int{10, 100, 1000} {
        b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
            parts := makeParts(n)
            b.ResetTimer()
            for b.Loop() { ConcatBuilder(parts) }
        })
    }
}
```

## Comparing with benchstat

```bash
go test -bench=. -benchmem -count=5 ./... > old.txt
# make your change
go test -bench=. -benchmem -count=5 ./... > new.txt
benchstat old.txt new.txt
```

Output shows the delta and p-value. p < 0.05 means statistically significant. Use `-count=10` for tighter intervals.

## Profiling from benchmarks

```bash
# CPU profile
go test -bench=BenchmarkWordFreq -cpuprofile=cpu.prof ./...
go tool pprof -http=:8080 cpu.prof

# Memory profile
go test -bench=BenchmarkWordFreq -memprofile=mem.prof ./...
go tool pprof -alloc_objects mem.prof
```

Inside pprof: `top`, `list FunctionName`, `web` (flame graph).

## Escape analysis

```bash
go build -gcflags='-m' ./...
# "escapes to heap" means an allocation; look for unexpected ones in hot paths
```

## Optimisation checklist

1. **Profile first** — never optimise without measurement.
2. **Reduce allocations** — the #1 Go perf cost.
   - Pre-size: `make([]T, 0, n)`, `make(map[K]V, n)`
   - Reuse: `sync.Pool` for frequently allocated buffers
   - Avoid interface boxing in tight loops
3. **Use `strings.Builder`** for string assembly in loops.
4. **RWMutex for read-heavy caches** — `sync.Mutex` blocks all readers.
5. **Profile again** to confirm improvement and check for regressions.

## Examples in this chapter

| File | Topic |
|------|-------|
| `examples/01_benchmark_basics/main.go` | Benchmark patterns, alloc measurement, benchstat |
| `examples/02_profiling_benchmarks/main.go` | Profiling workflow, escape analysis, optimisation checklist |
| `exercises/01_optimize/main.go` | Four real optimisations with measured speedups |
