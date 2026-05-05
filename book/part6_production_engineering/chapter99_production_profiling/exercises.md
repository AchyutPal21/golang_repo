# Chapter 99 Exercises — Production Profiling

## Exercise 1 (provided): Profiling Pipeline

Location: `exercises/01_profiling_pipeline/main.go`

Automated profiling pipeline demonstrating:
- Baseline capture at startup (heap + goroutines)
- Periodic snapshots every N seconds
- Regression detection: flag functions growing > 20%
- Memory leak detector: heap growth over rolling window
- Report generator: tabular diff of before vs after

## Exercise 2 (self-directed): Overhead Budget

Build an overhead estimator:
- Define `WorkloadProfile{OpsPerSec int, AvgLatencyMs float64}`
- `ProfilingOverhead(w WorkloadProfile, cpuProfSeconds int, freqMinutes int) OverheadReport`
- Report: ops lost per profile window, % overhead, recommendation (safe / marginal / too frequent)
- Safe threshold: < 0.5% of total capacity

## Exercise 3 (self-directed): Profile Diff Renderer

Write a profile diff renderer that:
- Takes two `map[string]float64` (function → flat ms)
- Computes absolute and relative change per function
- Sorts by absolute regression (worst first)
- Prints a table: function, before, after, delta, delta%
- Color-codes: regression (> +10%) and improvement (< -10%)

## Stretch Goal: Heap Growth Alerter

Build a `HeapGrowthAlerter` that:
- Samples `runtime.MemStats.HeapAlloc` every 30 seconds
- Fits a linear regression to the last 10 samples
- Fires an alert if slope > threshold bytes/sec (configurable)
- Resets the baseline after a GC drops heap by > 20%
- Distinguishes "sustained growth" (leak) from "spiky growth" (burst)
