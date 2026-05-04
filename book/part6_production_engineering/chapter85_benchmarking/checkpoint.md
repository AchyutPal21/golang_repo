# Chapter 85 Checkpoint — Benchmarking

## Concepts to know

- [ ] What is `b.N` in a Go benchmark? Who sets it and why?
- [ ] Why call `b.ResetTimer()` in a benchmark?
- [ ] What does `-benchmem` add to benchmark output?
- [ ] What is `b.ReportAllocs()`? Is it needed when using `-benchmem`?
- [ ] What is benchstat and why use it instead of eyeballing benchmark output?
- [ ] What does a p-value < 0.05 mean in benchstat output?
- [ ] How do you generate a CPU profile from a benchmark?
- [ ] What does "escapes to heap" mean in `-gcflags='-m'` output?
- [ ] When should you use `sync.Pool`?
- [ ] Why is pre-sizing a map or slice important for performance?

## Code exercises

### 1. String builder benchmark

Write benchmarks comparing these three approaches for building a comma-separated string from a 1000-element slice:
- `+` concatenation in a loop
- `strings.Builder`
- `strings.Join`

Report allocs/op. Explain which is fastest and why.

### 2. Sub-benchmark sizes

Write a benchmark for a sorting function with sub-benchmarks for n=10, n=100, n=1000, and n=10000. Run it and observe how ns/op scales — is it O(n log n)?

### 3. Allocation reduction

Given:
```go
func Format(items []Item) string {
    var parts []string
    for _, item := range items {
        parts = append(parts, fmt.Sprintf("%s:%d", item.Name, item.Price))
    }
    return strings.Join(parts, ",")
}
```
Rewrite to reduce allocations. Measure before/after with `b.ReportAllocs()`.

## Quick reference

```bash
# Run benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkConcat -benchmem -count=5 ./...

# Compare with benchstat
go install golang.org/x/perf/cmd/benchstat@latest
benchstat old.txt new.txt

# CPU profile
go test -bench=BenchmarkFoo -cpuprofile=cpu.prof ./...
go tool pprof -http=:8080 cpu.prof

# Memory profile
go test -bench=BenchmarkFoo -memprofile=mem.prof ./...
go tool pprof -alloc_objects mem.prof

# Escape analysis
go build -gcflags='-m' ./...
```

```go
// Benchmark template
func BenchmarkFoo(b *testing.B) {
    setup := expensive()
    b.ResetTimer()
    for b.Loop() {   // or: i := 0; i < b.N; i++
        Foo(setup)
    }
}
```

## What to remember

- `b.ResetTimer()` excludes fixture setup from the measurement.
- Always use `-count=5` or more for benchstat comparisons — single runs are noisy.
- Allocation reduction is usually more impactful than CPU optimisation in Go.
- `sync.Pool` is for reusing temporary objects — it doesn't help for objects with different lifetimes.
- Profile before optimising — the bottleneck is rarely where you think it is.
