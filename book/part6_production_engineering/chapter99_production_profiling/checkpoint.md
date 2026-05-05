# Chapter 99 Checkpoint — Production Profiling

## Concepts to know

- [ ] What is the CPU overhead of an active pprof CPU profile? Is it safe to run in production?
- [ ] What is the difference between `/debug/pprof/allocs` and `/debug/pprof/heap`?
- [ ] How do you enable mutex contention profiling? Block profiling?
- [ ] What does a "wide box near the top" of a flame graph mean?
- [ ] What is the difference between `flat` and `cum` in `pprof top` output?
- [ ] How do you diff two profiles to find what regressed between deploys?
- [ ] Why should pprof endpoints never be exposed on a public port?
- [ ] What is `runtime.SetMutexProfileFraction`? What does the argument mean?
- [ ] Name two managed continuous profiling services for Go.

## Code exercises

### 1. Profile collector

Write a `ProfileCollector` that:
- Captures a heap snapshot via `pprof.WriteHeapProfile` to an `io.Writer`
- Captures a goroutine dump via `pprof.Lookup("goroutine").WriteTo`
- Reports the sizes of both outputs

### 2. Overhead estimator

Given a workload that runs at 10,000 ops/sec and CPU profiling adds 5% overhead, calculate: how many ops/sec are "lost" to profiling. At what ops/sec does this become unacceptable (> 1% budget)?

### 3. Regression detector

Write `DetectRegression(baseline, current map[string]float64, threshold float64) []string` that returns function names where `current[fn] / baseline[fn] > (1 + threshold)`.

## Quick reference

```bash
# CPU profile (30s)
curl -s http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof
(pprof) top 10
(pprof) list FunctionName
(pprof) web          # flame graph (needs graphviz)

# Profile diff
go tool pprof -http=:8080 -diff_base=baseline.prof current.prof

# Memory
curl -s http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof -alloc_space heap.prof

# Enable mutex + block profiling at startup
runtime.SetMutexProfileFraction(5)
runtime.SetBlockProfileRate(1)
curl http://localhost:6060/debug/pprof/mutex > mutex.prof
curl http://localhost:6060/debug/pprof/block > block.prof
```

## Expected answers

1. ~5% CPU overhead while the profile is active (30s window). Safe for production if infrequent (once every few minutes); do not run continuously.
2. `/allocs` shows all allocations since process start (cumulative). `/heap` shows what is alive right now (live objects). Use allocs to find allocation hotspots; heap to find memory leaks.
3. `runtime.SetMutexProfileFraction(n)` — samples 1-in-n mutex contention events. `runtime.SetBlockProfileRate(ns)` — samples blocking events longer than `ns` nanoseconds.
4. A wide box near the top of a flame graph is a function that consumes significant CPU directly (on-CPU leaf). That is where optimization effort pays off.
5. `flat`: time spent executing this function body only. `cum`: time in this function and everything it calls.
6. `go tool pprof -diff_base=baseline.prof current.prof` — positive values are regressions, negative are improvements.
7. Pprof dumps contain goroutine stacks, heap contents, and timing data. Exposed publicly, an attacker can map the entire call graph and memory layout of your service.
8. `SetMutexProfileFraction(n)` samples 1-in-n contentions. Argument 1 = sample everything (highest overhead). 0 = disabled.
9. Google Cloud Profiler, Datadog Continuous Profiler, Parca, Pyroscope.
