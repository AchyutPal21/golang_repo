# Chapter 99 — Production Profiling

Chapter 86 showed you how to profile locally with benchmarks. Production profiling is different: you profile a live service under real traffic, continuously, with minimal overhead, and diff profiles across deploys to catch regressions.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | Continuous profiling | Always-on pprof, overhead budget, profile collection loop |
| 2 | Profile analysis | Flame graph interpretation, top/list/tree, profile diff |
| E | Profiling pipeline | Automated regression detection: baseline vs current |

## Examples

### `examples/01_continuous_profiling`

Always-on profiling infrastructure:
- `net/http/pprof` endpoint setup and endpoint inventory
- Overhead measurement: profiling cost vs workload cost
- Profile collection scheduler: CPU every 5 min, heap on demand
- Self-profiling: the profiler profiling itself
- `GOMAXPROCS` and its effect on profiling accuracy

### `examples/02_profile_analysis`

Reading and interpreting profile data:
- `runtime/pprof` write to file + read back simulation
- Flat vs cumulative time interpretation
- Identifying hot paths from top N output
- Inlining and its effect on profile granularity
- Profile diff: "what changed between v1.2 and v1.3?"

### `exercises/01_profiling_pipeline`

Automated profiling pipeline:
- Baseline capture at startup
- Periodic comparison: current vs baseline
- Regression alert: flag if any function grows > 20%
- Memory leak detector: heap growth over time

## Key Concepts

**Always-on pprof (production-safe)**
```go
import _ "net/http/pprof"
go http.ListenAndServe(":6060", nil)
```
CPU profiling costs ~5% CPU when active. Heap profiles are nearly free.  
**Never expose `:6060` publicly** — restrict to internal network or loopback.

**Profile endpoints**

| Endpoint | What it captures | Overhead |
|----------|-----------------|----------|
| `/debug/pprof/profile?seconds=30` | 30s CPU profile | ~5% CPU while active |
| `/debug/pprof/heap` | Heap snapshot | Negligible |
| `/debug/pprof/goroutine?debug=2` | All goroutine stacks | Momentary STW |
| `/debug/pprof/allocs` | All allocations (past) | Negligible |
| `/debug/pprof/mutex` | Mutex contention | Must enable first |
| `/debug/pprof/block` | Blocking events | Must enable first |

**Enabling mutex and block profiling**
```go
runtime.SetMutexProfileFraction(5)   // sample 1-in-5 mutex contentions
runtime.SetBlockProfileRate(1000)    // sample 1-in-1000ns of blocking
```

**Continuous profiling tools**
- **Parca** (open source, Prometheus-style for profiles)
- **Pyroscope** (open source, flame graph UI)
- **Google Cloud Profiler** (GCP-native, very low overhead)
- **Datadog Continuous Profiler** (commercial)

**Reading a flame graph**
```
Width  = % of total time spent in this function + descendants
Color  = arbitrary (not meaningful in Go flame graphs)
Top    = the function actually on-CPU (leaf = hot spot)
Bottom = program entry (main → ...)

Hotspot rule: wide boxes near the TOP of a stack are where to optimize.
Wide boxes at the bottom are just callers — reducing them won't help.
```

**Profile diff workflow**
```bash
# Capture baseline at v1.2.3
curl http://localhost:6060/debug/pprof/profile?seconds=30 > baseline.prof

# Deploy v1.3.0, capture again
curl http://localhost:6060/debug/pprof/profile?seconds=30 > current.prof

# Diff: shows what got slower (+) or faster (-)
go tool pprof -http=:8080 -diff_base=baseline.prof current.prof
```

## Running

```bash
go run ./book/part6_production_engineering/chapter99_production_profiling/examples/01_continuous_profiling
go run ./book/part6_production_engineering/chapter99_production_profiling/examples/02_profile_analysis
go run ./book/part6_production_engineering/chapter99_production_profiling/exercises/01_profiling_pipeline
```
