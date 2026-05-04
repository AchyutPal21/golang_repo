# Chapter 86 Exercises — pprof

## Exercise 1 — Hotspot Finder (`exercises/01_hotspot`)

Profile a sales report pipeline and identify three bottlenecks, then verify each fix with measurements.

### System: report generation pipeline

```go
func generateReport(sales []Sale) *Report
func formatReport(r *Report) string
```

### The three bottlenecks

**Bottleneck 1: O(n²) deduplication**
- `uniqueProductsSlow` uses a nested loop to find unique product IDs
- Fix: replace with `map[string]struct{}` lookup
- Impact: quadratic → linear

**Bottleneck 2: String concatenation**
- `formatReportSlow` uses `result += ...` in a loop
- Fix: `strings.Builder` with `Grow` hint
- Impact: reduces allocs/op significantly

**Bottleneck 3: No caching**
- Report regenerated on every call even with the same data
- Fix: `ReportCache` with `sync.RWMutex`
- Impact: 1 compute + N cache hits

### Measurement

For each fix, measure:
- Wall time (before/after)
- `runtime.MemStats.TotalAlloc` delta

### Hints

- Use `runtime.GC()` before `runtime.ReadMemStats` for accurate before/after measurements
- `sales` slice of 2000 entries makes O(n²) measurably slow but still fast enough to demo
- The cache benefit is most visible with 100 repeated calls on the same key
