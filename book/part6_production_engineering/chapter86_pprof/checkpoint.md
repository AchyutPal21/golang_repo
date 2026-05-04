# Chapter 86 Checkpoint — pprof

## Concepts to know

- [ ] What are the three ways to collect a pprof profile in Go?
- [ ] What is the difference between `flat` and `cum` in pprof `top` output?
- [ ] What are the four memory profile modes (`-alloc_objects`, `-alloc_space`, `-inuse_objects`, `-inuse_space`)? When do you use each?
- [ ] What does "escapes to heap" mean in `-gcflags='-m'` output?
- [ ] Name three common causes of heap escape.
- [ ] What is `sync.Pool`? When should you use it?
- [ ] How do you detect goroutine leaks in tests?
- [ ] What does `net/http/pprof` import do?
- [ ] Which pprof endpoint gives you a CPU profile over 30 seconds?

## Code exercises

### 1. CPU hotspot

Write a function `SumSquaresSlow(n int) int` that computes the sum of squares naively. Write `SumSquaresFast` using the closed form `n*(n+1)*(2n+1)/6`. Measure the speedup.

### 2. Memory hotspot

Write a function that builds a `[]string` in a loop using `append` with a new slice vs `make([]string, 0, n)`. Measure the allocation difference with `runtime.ReadMemStats`.

### 3. sync.Pool demo

Write a JSON serialiser that creates a `bytes.Buffer` per call. Rewrite it using `sync.Pool`. Benchmark both and report `allocs/op`.

## Quick reference

```bash
# CPU profile from benchmark
go test -bench=. -cpuprofile=cpu.prof ./...
go tool pprof -http=:8080 cpu.prof

# Memory profile from benchmark
go test -bench=. -memprofile=mem.prof -benchmem ./...
go tool pprof -alloc_space mem.prof

# Always-on endpoint
import _ "net/http/pprof"
go http.ListenAndServe(":6060", nil)
# curl :6060/debug/pprof/profile?seconds=30 > cpu.prof

# Escape analysis
go build -gcflags='-m' ./...

# Goroutine count
runtime.NumGoroutine()
```

## What to remember

- Profile before optimising — the bottleneck is almost never where you expect.
- `flat` time = work done IN the function; `cum` = work done by the function AND all it calls.
- `alloc_space` shows where allocations happen; `inuse_space` shows what's alive now.
- `sync.Pool` only helps for objects you allocate many times with consistent size.
- Import `_ "net/http/pprof"` in your production binary to get on-demand profiling for free.
