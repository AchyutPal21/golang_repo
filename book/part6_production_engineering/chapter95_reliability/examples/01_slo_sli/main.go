// FILE: book/part6_production_engineering/chapter95_reliability/examples/01_slo_sli/main.go
// CHAPTER: 95 — Reliability Engineering
// TOPIC: SLOs, SLIs, error budgets, and burn rate calculation.
//
// Run:
//   go run ./book/part6_production_engineering/chapter95_reliability/examples/01_slo_sli

package main

import (
	"fmt"
	"math"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// SLO DEFINITIONS
// ─────────────────────────────────────────────────────────────────────────────

type SLO struct {
	Name       string
	Target     float64 // e.g. 0.999 for 99.9%
	WindowDays int
}

func (s SLO) errorBudgetMinutes() float64 {
	return (1 - s.Target) * float64(s.WindowDays) * 24 * 60
}

func (s SLO) errorBudgetRate() float64 {
	// error budget consumed per minute (fraction)
	return (1 - s.Target) / float64(s.WindowDays*24*60)
}

// ─────────────────────────────────────────────────────────────────────────────
// SLI MEASUREMENT
// ─────────────────────────────────────────────────────────────────────────────

type RequestWindow struct {
	Total    int64
	Errors   int64
	LatencyP99Ms float64
}

func (w RequestWindow) availability() float64 {
	if w.Total == 0 {
		return 1.0
	}
	return 1 - float64(w.Errors)/float64(w.Total)
}

func (w RequestWindow) errorRate() float64 {
	if w.Total == 0 {
		return 0
	}
	return float64(w.Errors) / float64(w.Total)
}

// ─────────────────────────────────────────────────────────────────────────────
// ERROR BUDGET TRACKER
// ─────────────────────────────────────────────────────────────────────────────

type BudgetTracker struct {
	SLO              SLO
	TotalMinutes     float64 // elapsed minutes in window
	ErrorMinutes     float64 // minutes where SLO was violated
}

func NewBudgetTracker(slo SLO) *BudgetTracker {
	return &BudgetTracker{SLO: slo}
}

func (b *BudgetTracker) Record(w RequestWindow, durationMinutes float64) {
	b.TotalMinutes += durationMinutes
	if w.availability() < b.SLO.Target {
		b.ErrorMinutes += durationMinutes
	}
}

func (b *BudgetTracker) BudgetRemaining() float64 {
	return b.SLO.errorBudgetMinutes() - b.ErrorMinutes
}

func (b *BudgetTracker) BudgetConsumedPct() float64 {
	return 100 * b.ErrorMinutes / b.SLO.errorBudgetMinutes()
}

func (b *BudgetTracker) BurnRate() float64 {
	if b.TotalMinutes == 0 {
		return 0
	}
	actualErrorRate := b.ErrorMinutes / b.TotalMinutes
	budgetRate := 1 - b.SLO.Target
	if budgetRate == 0 {
		return 0
	}
	return actualErrorRate / budgetRate
}

// ─────────────────────────────────────────────────────────────────────────────
// MULTI-WINDOW BURN RATE ALERTS
// ─────────────────────────────────────────────────────────────────────────────

type BurnRateWindow struct {
	Label         string
	DurationHours float64
	PageThreshold float64
}

var burnRateWindows = []BurnRateWindow{
	{"1h", 1, 14.4},
	{"6h", 6, 6.0},
	{"24h", 24, 3.0},
	{"3d", 72, 1.0},
}

func checkBurnRateAlerts(slo SLO, observedErrorPct float64) {
	fmt.Printf("  SLO: %.3f%%  Observed error rate: %.4f%%\n",
		slo.Target*100, observedErrorPct)
	fmt.Printf("  Error budget rate: %.6f%% per hour\n", (1-slo.Target)*100/float64(slo.WindowDays*24))
	fmt.Println()
	fmt.Printf("  %-8s  %12s  %12s  %6s  %s\n", "Window", "Burn Rate", "Threshold", "Alert", "Action")
	fmt.Printf("  %s\n", strings.Repeat("-", 65))

	budgetPerHour := (1 - slo.Target) / float64(slo.WindowDays*24)
	for _, w := range burnRateWindows {
		burnRate := observedErrorPct / 100 / budgetPerHour
		alert := "none"
		if burnRate > w.PageThreshold {
			alert = "PAGE"
		}
		action := "Monitor"
		if alert == "PAGE" {
			action = "Wake oncall immediately"
		}
		fmt.Printf("  %-8s  %12.2f  %12.1f  %6s  %s\n",
			w.Label, burnRate, w.PageThreshold, alert, action)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SLO COMPARISON TABLE
// ─────────────────────────────────────────────────────────────────────────────

var commonSLOs = []SLO{
	{"99%", 0.99, 30},
	{"99.5%", 0.995, 30},
	{"99.9%", 0.999, 30},
	{"99.95%", 0.9995, 30},
	{"99.99%", 0.9999, 30},
}

func printSLOTable() {
	fmt.Printf("  %-8s  %18s  %18s\n", "SLO", "Budget (min/month)", "Budget (h:mm/month)")
	fmt.Printf("  %s\n", strings.Repeat("-", 50))
	for _, s := range commonSLOs {
		mins := s.errorBudgetMinutes()
		h := int(mins) / 60
		m := int(mins) % 60
		fmt.Printf("  %-8s  %18.1f  %14d:%02d\n", s.Name, mins, h, m)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 95: SLOs, SLIs & Error Budgets ===")
	fmt.Println()

	// ── SLO TABLE ─────────────────────────────────────────────────────────────
	fmt.Println("--- Error budgets by SLO (30-day window) ---")
	printSLOTable()
	fmt.Println()

	// ── ERROR BUDGET TRACKING ─────────────────────────────────────────────────
	fmt.Println("--- Error budget tracker: order-service (SLO: 99.9%) ---")
	slo := SLO{"order-service", 0.999, 30}
	tracker := NewBudgetTracker(slo)

	// Simulate a month of data: mostly good, one outage
	type period struct {
		label   string
		window  RequestWindow
		minutes float64
	}
	periods := []period{
		{"Week 1 (healthy)", RequestWindow{1_000_000, 100, 45}, 7 * 24 * 60},
		{"Week 2 (healthy)", RequestWindow{1_200_000, 80, 42}, 7 * 24 * 60},
		{"Week 3 (incident)", RequestWindow{800_000, 4800, 120}, 7 * 24 * 60},
		{"Week 4 (recovery)", RequestWindow{1_100_000, 110, 48}, 9 * 24 * 60},
	}

	fmt.Printf("  Budget total: %.1f minutes\n\n", slo.errorBudgetMinutes())
	for _, p := range periods {
		tracker.Record(p.window, p.minutes)
		fmt.Printf("  %-30s  avail=%.4f%%  budget_remaining=%.1fmin  consumed=%.1f%%\n",
			p.label,
			p.window.availability()*100,
			tracker.BudgetRemaining(),
			tracker.BudgetConsumedPct())
	}
	fmt.Printf("\n  Final burn rate: %.2f× (should be ≤1.0 at month end)\n", tracker.BurnRate())
	fmt.Println()

	// ── BURN RATE ALERTS ──────────────────────────────────────────────────────
	fmt.Println("--- Burn rate alerts: payment-service ---")
	fmt.Println("  Scenario: 2% error rate observed (SLO: 99.9%, 30-day window)")
	fmt.Println()
	checkBurnRateAlerts(SLO{"payment-service", 0.999, 30}, 2.0)
	fmt.Println()

	// ── LATENCY SLO ───────────────────────────────────────────────────────────
	fmt.Println("--- Latency SLO example ---")
	latencyTarget := 200.0 // ms p99 target
	observations := []float64{45, 67, 89, 120, 180, 210, 250, 310, 450, 1200}
	// p99 estimate: 99th percentile from sorted sample
	// (simplified: use max of top 1% of sorted list)
	p99 := observations[int(math.Ceil(float64(len(observations))*0.99))-1]
	fmt.Printf("  Target: p99 < %.0fms\n", latencyTarget)
	fmt.Printf("  Observed p99: %.0fms\n", p99)
	if p99 > latencyTarget {
		fmt.Printf("  Status: BREACHING SLO (%.0fms > %.0fms)\n", p99, latencyTarget)
	} else {
		fmt.Printf("  Status: meeting SLO\n")
	}
}
