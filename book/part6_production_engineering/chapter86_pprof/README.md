# Chapter 86 — pprof

Go ships with `runtime/pprof` and `net/http/pprof` for profiling CPU usage, memory allocations, goroutine stacks, and more. Profiling is how you find what to optimise — without it, you're guessing.

## Profiling methods

### 1. Benchmark-based (best for focused optimisation)

```bash
go test -bench=BenchmarkFoo -cpuprofile=cpu.prof ./...
go test -bench=BenchmarkFoo -memprofile=mem.prof ./...
go tool pprof -http=:8080 cpu.prof
```

### 2. runtime/pprof in main (for scripts and CLIs)

```go
import "runtime/pprof"

f, _ := os.Create("cpu.prof")
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()
// ... workload ...
```

### 3. net/http/pprof endpoint (always-on, production-safe)

```go
import _ "net/http/pprof"
go http.ListenAndServe(":6060", nil)
```

Endpoints:
- `GET /debug/pprof/profile?seconds=30` — 30s CPU profile
- `GET /debug/pprof/heap` — heap snapshot
- `GET /debug/pprof/goroutine?debug=2` — all goroutine stacks

## Reading pprof output

```
(pprof) top 10
    flat  flat%   sum%        cum   cum%
   3.80s 89.20% 89.20%      3.80s 89.20%  main.isPrimeSlow

flat  = time IN this function (not counting callees)
cum   = time IN this function AND everything it calls

(pprof) list isPrimeSlow   → annotated source with per-line times
(pprof) web                → flame graph in browser (needs graphviz)
```

## Memory profile modes

```bash
go tool pprof -alloc_objects mem.prof   # allocation count (past)
go tool pprof -alloc_space   mem.prof   # bytes allocated (past)
go tool pprof -inuse_objects mem.prof   # live object count
go tool pprof -inuse_space   mem.prof   # live bytes (current heap)
```

## Escape analysis

```bash
go build -gcflags='-m' ./...
# "escapes to heap" means a heap allocation — look for surprises in hot paths
```

Common escape causes:
- Returning a pointer to a local variable
- Storing a value in an interface
- Closures capturing outer variables
- Slices/maps that grow beyond a size threshold

## sync.Pool

Reuse frequently-allocated objects:

```go
var bufPool = sync.Pool{
    New: func() any {
        b := make([]byte, 0, 4096)
        return &b
    },
}

buf := bufPool.Get().(*[]byte)
*buf = (*buf)[:0]  // reset
// ... use buf ...
bufPool.Put(buf)
```

Rules: only for short-lived objects of consistent size; always `Reset()` before reuse.

## Goroutine leak detection

```bash
# Check goroutine count before/after in tests
before := runtime.NumGoroutine()
// ... operation ...
after := runtime.NumGoroutine()
if after > before { t.Errorf("goroutine leak") }

# In production
curl http://localhost:6060/debug/pprof/goroutine?debug=2 | head -100
```

## Examples in this chapter

| File | Topic |
|------|-------|
| `examples/01_cpu_profile/main.go` | CPU profiling, hotspot identification, pprof workflow |
| `examples/02_memory_profile/main.go` | Memory profiling, escape analysis, sync.Pool, leak detection |
| `exercises/01_hotspot/main.go` | Three bottlenecks: O(n²) dedup, string concat, no caching |
