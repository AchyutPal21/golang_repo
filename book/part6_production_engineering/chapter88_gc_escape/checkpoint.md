# Chapter 88 Checkpoint — GC & Escape Analysis

## Concept checks

1. Name the five most common causes of heap escape in Go.

2. A service is OOM-killed at 2 GB RSS. You set `GOMEMLIMIT=1500MiB`. What
   does the Go runtime do differently once the heap approaches 1500 MiB?

3. You run `go build -gcflags="-m"` and see:
   ```
   ./handler.go:22:6: moved to heap: buf
   ./handler.go:37:15: n escapes to heap
   ```
   What is the difference between "moved to heap" and "escapes to heap"?

4. A finalizer is set on a struct. The struct's `Close()` method is called.
   Will the finalizer still fire? Why or why not — and what should `Close`
   do to prevent it?

5. You set `GOGC=500` to reduce GC frequency. What is the risk?

## Code review

```go
func process(items []string) []string {
    var out []string
    for _, item := range items {
        out = append(out, strings.ToUpper(item))
    }
    return out
}
```

Identify two allocation hot-spots and propose fixes.

## Expected answers

1. (a) returning a pointer to a local, (b) storing into interface{}, (c)
   closure capturing a local, (d) dynamic-size `make([]T, n)`, (e) passing
   values to variadic `...any` (fmt.Println etc.).

2. The GC runs more aggressively (triggers at a lower heap occupancy) to
   keep RSS below the limit. This trades CPU time for memory safety.

3. "moved to heap" = the compiler decided a variable you defined in this
   package must live on the heap. "escapes to heap" = a value is forced to
   the heap by code in another package (e.g., `fmt` boxing via `any`). You
   can fix the former; the latter often requires restructuring the call.

4. The finalizer will NOT fire if `Close` removes it via
   `runtime.SetFinalizer(r, nil)`. This is the correct pattern — `Close`
   disarms the finalizer so it does not run a second cleanup.

5. Higher GOGC means the heap can grow to (live * 6) before GC triggers.
   Under sudden load spikes the RSS can spike dramatically, potentially
   exceeding container memory limits and causing OOM kills.

**Hot-spots**: (a) `var out []string` — no capacity hint, causes repeated
reallocation; fix: `out := make([]string, 0, len(items))`. (b) each
`strings.ToUpper` allocates a new string; if the upper-cased strings are
only needed briefly, consider a `strings.Builder` or `bytes.ToUpper` with a
pre-allocated sink.
