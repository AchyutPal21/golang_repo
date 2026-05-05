// FILE: book/part6_production_engineering/chapter99_production_profiling/examples/02_profile_analysis/main.go
// CHAPTER: 99 — Production Profiling
// TOPIC: Reading profile output — flat vs cum, top/list/tree, flame graph
//        interpretation, profile diff, and inlining effects.
//
// Run:
//   go run ./book/part6_production_engineering/chapter99_production_profiling/examples/02_profile_analysis

package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED PROFILE DATA
// ─────────────────────────────────────────────────────────────────────────────

type ProfileEntry struct {
	Function string
	FlatMs   float64 // time IN this function only
	CumMs    float64 // time in this function + everything it calls
	Calls    int
}

// simulateTopOutput mimics `(pprof) top 10` output
func simulateTopOutput(entries []ProfileEntry, total float64) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].FlatMs > entries[j].FlatMs
	})
	fmt.Printf("  %-8s  %-8s  %-8s  %-8s  %-8s  %s\n",
		"flat", "flat%", "sum%", "cum", "cum%", "Function")
	fmt.Printf("  %s\n", strings.Repeat("-", 75))
	var sum float64
	for i, e := range entries {
		if i >= 10 {
			break
		}
		sum += e.FlatMs
		flatPct := 100 * e.FlatMs / total
		cumPct := 100 * e.CumMs / total
		sumPct := 100 * sum / total
		fmt.Printf("  %6.2fs  %6.2f%%  %6.2f%%  %6.2fs  %6.2f%%  %s\n",
			e.FlatMs/1000, flatPct, sumPct, e.CumMs/1000, cumPct, e.Function)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FLAME GRAPH INTERPRETATION
// ─────────────────────────────────────────────────────────────────────────────

type CallFrame struct {
	Name     string
	WidthPct float64 // % of total time
	Depth    int
	IsHot    bool // on-CPU leaf
}

func printFlameGraph(frames []CallFrame) {
	totalWidth := 60
	fmt.Printf("  %-35s  %s\n", "Function", "Timeline (% CPU)")
	fmt.Printf("  %s\n", strings.Repeat("-", 75))
	for _, f := range frames {
		indent := strings.Repeat("  ", f.Depth)
		bar := int(f.WidthPct / 100 * float64(totalWidth))
		if bar < 1 && f.WidthPct > 0 {
			bar = 1
		}
		marker := strings.Repeat("█", bar)
		hot := ""
		if f.IsHot {
			hot = " ← HOT (on CPU)"
		}
		fmt.Printf("  %-35s  %s %.1f%%%s\n",
			indent+f.Name, marker, f.WidthPct, hot)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PROFILE DIFF
// ─────────────────────────────────────────────────────────────────────────────

type DiffEntry struct {
	Function  string
	BaselineMs float64
	CurrentMs  float64
}

func (d DiffEntry) DeltaMs() float64  { return d.CurrentMs - d.BaselineMs }
func (d DiffEntry) DeltaPct() float64 {
	if d.BaselineMs == 0 {
		return math.Inf(1)
	}
	return 100 * (d.CurrentMs - d.BaselineMs) / d.BaselineMs
}

func printProfileDiff(diffs []DiffEntry) {
	sort.Slice(diffs, func(i, j int) bool {
		return math.Abs(diffs[i].DeltaMs()) > math.Abs(diffs[j].DeltaMs())
	})
	fmt.Printf("  %-45s  %8s  %8s  %8s  %8s  %s\n",
		"Function", "Before", "After", "Delta", "Delta%", "Verdict")
	fmt.Printf("  %s\n", strings.Repeat("-", 100))
	for _, d := range diffs {
		verdict := "stable"
		if d.DeltaPct() > 20 {
			verdict = "REGRESSION"
		} else if d.DeltaPct() < -20 {
			verdict = "improved"
		}
		fmt.Printf("  %-45s  %7.0fms  %7.0fms  %+7.0fms  %+7.1f%%  %s\n",
			truncate(d.Function, 45),
			d.BaselineMs, d.CurrentMs, d.DeltaMs(), d.DeltaPct(), verdict)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// ─────────────────────────────────────────────────────────────────────────────
// INLINING EFFECTS
// ─────────────────────────────────────────────────────────────────────────────

const inliningNote = `  Inlining and profiling:

  When the Go compiler inlines a function (small leaf functions are inlined
  by default), the call disappears from the profile — its time is attributed
  to the CALLER.

  Example:
    func add(a, b int) int { return a + b }   // inlined — disappears
    func compute(x int) int {
        return add(x, x*2)                     // add's time shows as compute's
    }

  Consequences:
    1. Hot functions that are inlined show 0 flat time — misleading
    2. The caller looks hotter than it really is
    3. Use -gcflags='-m' to see what was inlined

  Force no-inline for accurate profiling (development only):
    //go:noinline
    func add(a, b int) int { return a + b }

  Practical rule: if pprof points at a caller but the function body looks
  trivial, check if it's calling an inlined hot function.`

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 99: Profile Analysis ===")
	fmt.Println()

	// ── TOP OUTPUT ────────────────────────────────────────────────────────────
	fmt.Println("--- Simulated `pprof top 10` output ---")
	totalMs := 10_000.0
	entries := []ProfileEntry{
		{"github.com/myorg/app/internal/report.uniqueProductsSlow", 3800, 3800, 2000},
		{"github.com/myorg/app/internal/report.formatReport", 1200, 1200, 2000},
		{"runtime.mallocgc", 900, 900, 15000},
		{"github.com/myorg/app/internal/db.scanRows", 750, 1100, 2000},
		{"encoding/json.Marshal", 620, 620, 3000},
		{"github.com/myorg/app/internal/cache.Get", 480, 480, 5000},
		{"runtime.gcBgMarkWorker", 420, 420, 1},
		{"github.com/myorg/app/internal/order.Validate", 310, 1850, 2000},
		{"strings.Builder.Write", 290, 290, 8000},
		{"github.com/myorg/app/internal/report.generateReport", 80, 8200, 2000},
		{"net/http.(*ServeMux).ServeHTTP", 10, 9900, 2000},
	}
	simulateTopOutput(entries, totalMs)
	fmt.Println()
	fmt.Println("  Reading guide:")
	fmt.Println("  • uniqueProductsSlow: flat=cum → leaf function, all work done here → fix this first")
	fmt.Println("  • generateReport:     flat=80ms but cum=8200ms → it's a caller, not the hotspot")
	fmt.Println("  • order.Validate:     flat=310ms, cum=1850ms → does work AND calls slow functions")
	fmt.Println()

	// ── FLAME GRAPH ───────────────────────────────────────────────────────────
	fmt.Println("--- Flame graph interpretation ---")
	frames := []CallFrame{
		{"main.ServeHTTP", 97, 0, false},
		{"router.dispatch", 95, 1, false},
		{"handler.GetReport", 92, 2, false},
		{"report.generateReport", 88, 3, false},
		{"report.uniqueProductsSlow", 38, 4, true},
		{"report.formatReport", 12, 4, true},
		{"db.scanRows", 11, 4, false},
		{"database/sql.(*Rows).Scan", 10, 5, true},
		{"runtime.gcBgMarkWorker", 3, 0, true},
	}
	printFlameGraph(frames)
	fmt.Println()
	fmt.Println("  Reading guide:")
	fmt.Println("  • Wide boxes at BOTTOM = hot callers (reduce won't help)")
	fmt.Println("  • Wide ← HOT boxes = on-CPU leaves = where to optimize")
	fmt.Println()

	// ── PROFILE DIFF ──────────────────────────────────────────────────────────
	fmt.Println("--- Profile diff: v1.2 → v1.3 ---")
	diffs := []DiffEntry{
		{"report.uniqueProductsSlow", 3800, 4600, },
		{"report.formatReport", 1200, 850},
		{"encoding/json.Marshal", 620, 180},
		{"runtime.mallocgc", 900, 720},
		{"db.scanRows", 750, 760},
		{"cache.Get", 480, 470},
		{"order.Validate", 310, 305},
		{"strings.Builder.Write", 290, 150},
		{"report.uniqueProductsFast", 0, 85},
	}
	printProfileDiff(diffs)
	fmt.Println()
	fmt.Println("  Summary: uniqueProductsSlow regressed (+21%) — should have been replaced.")
	fmt.Println("  json.Marshal improved 71% — switched to encoding/json v2 (streaming).")
	fmt.Println()

	// ── INLINING EFFECTS ─────────────────────────────────────────────────────
	fmt.Println("--- Inlining and profiling ---")
	fmt.Println(inliningNote)
}
