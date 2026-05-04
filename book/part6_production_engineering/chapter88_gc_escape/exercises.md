# Chapter 88 Exercises — GC & Escape Analysis

## Exercise 1 (provided): GC Pressure Reduction

Location: `exercises/01_gc_pressure/main.go`

Refactors an HTTP-pipeline simulation to reduce GC cycles by:
- Caller-provided path-segment slice
- `strconv.AppendInt` log-line builder
- Channel-based Response pool

## Exercise 2 (self-directed): Escape Hunting

Given the following function, use `go build -gcflags="-m"` to identify every
escape. Then rewrite the function to have zero heap allocations (except the
final return value):

```go
func summarise(events []Event) string {
    counts := map[string]int{}
    for _, e := range events {
        counts[e.Type]++
    }
    lines := []string{}
    for k, v := range counts {
        lines = append(lines, fmt.Sprintf("%s=%d", k, v))
    }
    sort.Strings(lines)
    return strings.Join(lines, ",")
}
```

Hints: replace `map` with a sorted `[]pair`, replace `fmt.Sprintf` with
`strconv.AppendInt`, replace `strings.Join` with `strings.Builder`.

## Exercise 3 (self-directed): GC Pause Histogram

Instrument a long-running loop (100k iterations, each allocating 4 KB) to
capture per-GC-cycle pause latencies from `runtime.MemStats.PauseNs` ring
buffer. Print a simple histogram of pause durations in 100 µs buckets.

## Stretch Goal: GOGC Auto-Tuner

Build a goroutine that monitors `runtime.MemStats.NumGC` every 500 ms and
adjusts `debug.SetGCPercent` dynamically:
- If GC frequency > 5/s → increase GOGC by 25 (up to 400)
- If RSS > 80% of GOMEMLIMIT → decrease GOGC by 25 (down to 50)
- Print a log line each time it adjusts
