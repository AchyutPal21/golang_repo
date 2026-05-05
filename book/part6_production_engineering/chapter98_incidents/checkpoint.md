# Chapter 98 Checkpoint — Incident Management & Debugging

## Concepts to know

- [ ] How do you capture a goroutine dump from a running Go process without restarting it?
- [ ] What is a goroutine leak? How do you detect one in a test?
- [ ] What does `recover()` return? When does it return nil?
- [ ] Why must `recover()` be called directly in a `defer` function, not in a nested function?
- [ ] What information should a panic recovery handler capture?
- [ ] What is MTTR? What is MTTD?
- [ ] What is a postmortem? What makes it "blameless"?
- [ ] Name three tools for capturing a heap snapshot from a running Go process.
- [ ] What is the `runtime.ReadMemStats` call sequence for accurate measurement?

## Code exercises

### 1. Stack trace capture

Write a function `captureStack(all bool) string` that:
- Returns the current goroutine's stack if `all=false`
- Returns all goroutine stacks if `all=true`
- Truncates at 64KB

### 2. Panic recovery middleware

Write `RecoverMiddleware(next http.Handler) http.Handler` that:
- Recovers from panics
- Writes HTTP 500 to the client
- Logs: panic value, stack trace, request path, request ID

### 3. Goroutine leak test

Write a test helper `AssertNoGoroutineLeak(t *testing.T, fn func())` that:
- Records `runtime.NumGoroutine()` before calling `fn`
- Waits up to 100ms for goroutines to settle
- Fails the test if count increased

## Quick reference

```go
// Goroutine dump
buf := make([]byte, 1<<20)
n := runtime.Stack(buf, true)
fmt.Printf("%s", buf[:n])

// Heap snapshot
f, _ := os.Create("heap.prof")
pprof.WriteHeapProfile(f)
f.Close()

// Accurate memory measurement
runtime.GC()
var m runtime.MemStats
runtime.ReadMemStats(&m)

// Send SIGQUIT to running process
kill -QUIT $(pgrep myapp)

// pprof endpoints
GET /debug/pprof/goroutine?debug=2   → all goroutine stacks
GET /debug/pprof/heap                → heap snapshot
GET /debug/pprof/profile?seconds=30  → 30s CPU profile
```

## Expected answers

1. `kill -QUIT <pid>` dumps all stacks to stderr; `/debug/pprof/goroutine?debug=2` via HTTP endpoint.
2. A goroutine leak is a goroutine that is stuck waiting forever (usually on a channel or lock). Detect by comparing `runtime.NumGoroutine()` before and after.
3. `recover()` returns the value passed to `panic()`, or nil if there's no active panic.
4. `recover()` only works when called directly in the deferred function — the runtime checks call depth.
5. Panic value, full stack trace, request ID, user, timestamp. Report to error tracking (Sentry/Honeybadger).
6. MTTR: Mean Time To Recovery (detect → resolved). MTTD: Mean Time To Detection (incident starts → alert fires).
7. A postmortem analyses what went wrong and how to prevent recurrence. "Blameless" means focusing on systems/processes rather than blaming individuals.
8. `pprof.WriteHeapProfile`, `curl /debug/pprof/heap`, `go tool pprof`.
9. `runtime.GC()` before `runtime.ReadMemStats()` to flush allocation records; `GC()` after workload to collect garbage before comparing.
