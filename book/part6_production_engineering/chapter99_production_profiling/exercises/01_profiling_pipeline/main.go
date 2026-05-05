// FILE: book/part6_production_engineering/chapter99_production_profiling/exercises/01_profiling_pipeline/main.go
// CHAPTER: 99 — Production Profiling
// EXERCISE: Automated profiling pipeline — baseline capture, periodic
//           snapshots, regression detection, and heap growth alerting.
//
// Run:
//   go run ./book/part6_production_engineering/chapter99_production_profiling/exercises/01_profiling_pipeline

package main

import (
	"bytes"
	"fmt"
	"math"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// HEAP SNAPSHOT
// ─────────────────────────────────────────────────────────────────────────────

type HeapSnapshot struct {
	Timestamp   time.Time
	HeapAllocB  uint64
	HeapObjects uint64
	NumGC       uint32
	ProfileSize int
}

func takeHeapSnapshot() HeapSnapshot {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	var buf bytes.Buffer
	pprof.WriteHeapProfile(&buf) //nolint:errcheck
	return HeapSnapshot{
		Timestamp:   time.Now(),
		HeapAllocB:  m.Alloc,
		HeapObjects: m.HeapObjects,
		NumGC:       m.NumGC,
		ProfileSize: buf.Len(),
	}
}

func (s HeapSnapshot) String() string {
	return fmt.Sprintf("heap=%.2fMB objects=%d gc=%d profile=%dB",
		float64(s.HeapAllocB)/(1024*1024), s.HeapObjects, s.NumGC, s.ProfileSize)
}

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED FUNCTION TIMING (replaces a real CPU profile)
// ─────────────────────────────────────────────────────────────────────────────

type FuncProfile map[string]float64 // function → flat ms

func simulateBaseline() FuncProfile {
	return FuncProfile{
		"report.generateReport":      80,
		"report.uniqueProducts":      120,
		"report.formatReport":        1200,
		"db.scanRows":                750,
		"cache.Get":                  480,
		"order.Validate":             310,
		"encoding/json.Marshal":      620,
		"runtime.mallocgc":           900,
	}
}

func simulateCurrent(introduceRegression bool) FuncProfile {
	p := FuncProfile{
		"report.generateReport":      85,
		"report.uniqueProducts":      130,
		"report.formatReport":        1180,
		"db.scanRows":                760,
		"cache.Get":                  470,
		"order.Validate":             305,
		"encoding/json.Marshal":      600,
		"runtime.mallocgc":           880,
	}
	if introduceRegression {
		p["report.uniqueProducts"] = 980 // O(n²) crept back in
		p["encoding/json.Marshal"] = 1400 // marshaling large structs now
	}
	return p
}

// ─────────────────────────────────────────────────────────────────────────────
// REGRESSION DETECTOR
// ─────────────────────────────────────────────────────────────────────────────

type RegressionResult struct {
	Function   string
	BaselineMs float64
	CurrentMs  float64
	DeltaPct   float64
}

func detectRegressions(baseline, current FuncProfile, threshold float64) []RegressionResult {
	var regressions []RegressionResult
	for fn, cur := range current {
		base, ok := baseline[fn]
		if !ok {
			continue
		}
		if base == 0 {
			continue
		}
		delta := 100 * (cur - base) / base
		if delta > threshold {
			regressions = append(regressions, RegressionResult{fn, base, cur, delta})
		}
	}
	sort.Slice(regressions, func(i, j int) bool {
		return regressions[i].DeltaPct > regressions[j].DeltaPct
	})
	return regressions
}

// ─────────────────────────────────────────────────────────────────────────────
// HEAP GROWTH DETECTOR
// ─────────────────────────────────────────────────────────────────────────────

type HeapGrowthAnalysis struct {
	Samples     []HeapSnapshot
	Slope       float64 // bytes per second
	IsLeaking   bool
	Explanation string
}

func analyzeHeapGrowth(samples []HeapSnapshot) HeapGrowthAnalysis {
	if len(samples) < 2 {
		return HeapGrowthAnalysis{Samples: samples, Explanation: "not enough samples"}
	}
	// Simple linear regression
	n := float64(len(samples))
	var sumX, sumY, sumXY, sumX2 float64
	t0 := samples[0].Timestamp
	for i, s := range samples {
		x := s.Timestamp.Sub(t0).Seconds()
		y := float64(s.HeapAllocB)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		_ = i
	}
	denom := n*sumX2 - sumX*sumX
	var slope float64
	if math.Abs(denom) > 1e-9 {
		slope = (n*sumXY - sumX*sumY) / denom
	}
	isLeaking := slope > 10*1024 // > 10 KB/s sustained growth
	explanation := "stable"
	if slope > 0 && isLeaking {
		explanation = fmt.Sprintf("growing at %.1f KB/s — possible leak", slope/1024)
	} else if slope > 0 {
		explanation = fmt.Sprintf("slow growth %.1f B/s — monitor", slope)
	} else if slope < 0 {
		explanation = "shrinking (GC effective)"
	}
	return HeapGrowthAnalysis{
		Samples:     samples,
		Slope:       slope,
		IsLeaking:   isLeaking,
		Explanation: explanation,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DIFF RENDERER
// ─────────────────────────────────────────────────────────────────────────────

func renderDiff(baseline, current FuncProfile) {
	type row struct {
		fn       string
		base     float64
		cur      float64
		deltaPct float64
	}
	var rows []row
	seen := map[string]bool{}
	for fn, base := range baseline {
		cur := current[fn]
		rows = append(rows, row{fn, base, cur, 100 * (cur - base) / base})
		seen[fn] = true
	}
	for fn, cur := range current {
		if !seen[fn] {
			rows = append(rows, row{fn, 0, cur, math.Inf(1)})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return math.Abs(rows[i].deltaPct) > math.Abs(rows[j].deltaPct)
	})
	fmt.Printf("  %-45s  %8s  %8s  %8s\n", "Function", "Before", "After", "Delta%")
	fmt.Printf("  %s\n", strings.Repeat("-", 80))
	for _, r := range rows {
		verdict := ""
		if r.deltaPct > 20 {
			verdict = " ← REGRESSION"
		} else if r.deltaPct < -20 {
			verdict = " ← improved"
		}
		fmt.Printf("  %-45s  %7.0fms  %7.0fms  %+7.1f%%%s\n",
			truncate(r.fn, 45), r.base, r.cur, r.deltaPct, verdict)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 99 Exercise: Profiling Pipeline ===")
	fmt.Println()

	// ── BASELINE CAPTURE ──────────────────────────────────────────────────────
	fmt.Println("--- Baseline heap snapshot at startup ---")
	baseline := takeHeapSnapshot()
	fmt.Printf("  Baseline: %s\n\n", baseline)

	// ── PERIODIC SNAPSHOTS ────────────────────────────────────────────────────
	fmt.Println("--- Taking 3 periodic snapshots (simulated workload) ---")
	var heapSamples []HeapSnapshot
	heapSamples = append(heapSamples, baseline)

	// Simulate workload: allocate progressively more
	sink := make([][]byte, 0, 3)
	for i := 1; i <= 3; i++ {
		// simulate allocation growth
		buf := make([]byte, 512*1024*(1+i)) // 1MB, 1.5MB, 2MB
		sink = append(sink, buf)
		time.Sleep(20 * time.Millisecond)
		snap := takeHeapSnapshot()
		heapSamples = append(heapSamples, snap)
		fmt.Printf("  Snapshot %d: %s\n", i, snap)
	}
	_ = sink
	fmt.Println()

	// ── HEAP GROWTH ANALYSIS ──────────────────────────────────────────────────
	fmt.Println("--- Heap growth analysis ---")
	analysis := analyzeHeapGrowth(heapSamples)
	fmt.Printf("  Slope:     %.1f KB/s\n", analysis.Slope/1024)
	fmt.Printf("  Leaking:   %v\n", analysis.IsLeaking)
	fmt.Printf("  Verdict:   %s\n\n", analysis.Explanation)

	// ── REGRESSION DETECTION (no regression) ─────────────────────────────────
	fmt.Println("--- Profile diff: healthy deploy (no regression) ---")
	baseProfile := simulateBaseline()
	healthyCurrent := simulateCurrent(false)
	renderDiff(baseProfile, healthyCurrent)
	regs := detectRegressions(baseProfile, healthyCurrent, 20.0)
	fmt.Printf("  Regressions detected: %d\n\n", len(regs))

	// ── REGRESSION DETECTION (with regression) ────────────────────────────────
	fmt.Println("--- Profile diff: bad deploy (regression introduced) ---")
	badCurrent := simulateCurrent(true)
	renderDiff(baseProfile, badCurrent)
	regs2 := detectRegressions(baseProfile, badCurrent, 20.0)
	fmt.Printf("\n  Regressions detected: %d\n", len(regs2))
	if len(regs2) > 0 {
		fmt.Println("  ALERT: performance regression detected — consider rollback")
		for _, r := range regs2 {
			fmt.Printf("    • %s: +%.1f%% (%.0fms → %.0fms)\n",
				r.Function, r.DeltaPct, r.BaselineMs, r.CurrentMs)
		}
	}
	fmt.Println()

	// ── PIPELINE WORKFLOW ─────────────────────────────────────────────────────
	fmt.Println("--- Automated pipeline workflow ---")
	fmt.Println(`  1. Deploy new version
  2. Wait 5 minutes for traffic warm-up
  3. Capture CPU profile (30s) + heap snapshot
  4. Compare against stored baseline with diff
  5. If any function regresses > 20%:
       → Fire alert: "CPU regression in <function>"
       → Block promotion to next stage
       → Offer rollback
  6. If heap slope > 10 KB/s:
       → Fire alert: "Possible memory leak detected"
  7. On green: promote baseline → replace with current
  8. Store profiles for 30 days (regression history)`)
}
