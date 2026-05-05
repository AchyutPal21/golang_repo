// FILE: book/part6_production_engineering/chapter95_reliability/exercises/01_slo_service/main.go
// CHAPTER: 95 — Reliability Engineering
// EXERCISE: Complete SLO-aware service — error budget tracking, burn rate
//           alerts, and policy enforcement (feature freeze when budget < 10%).
//
// Run:
//   go run ./book/part6_production_engineering/chapter95_reliability/exercises/01_slo_service

package main

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SLO CONFIGURATION
// ─────────────────────────────────────────────────────────────────────────────

type SLO struct {
	ServiceName     string
	AvailTarget     float64 // e.g. 0.999
	LatencyP99Ms    float64 // p99 target in ms
	ErrorRateTarget float64 // max error rate (e.g. 0.001)
	WindowDays      int
}

func (s SLO) errorBudgetMinutes() float64 {
	return (1 - s.AvailTarget) * float64(s.WindowDays) * 24 * 60
}

// ─────────────────────────────────────────────────────────────────────────────
// REAL-TIME METRICS
// ─────────────────────────────────────────────────────────────────────────────

type Metrics struct {
	Requests atomic.Int64
	Errors   atomic.Int64
	LatencyTotalMs atomic.Int64
}

func (m *Metrics) RecordRequest(latencyMs int64, isError bool) {
	m.Requests.Add(1)
	m.LatencyTotalMs.Add(latencyMs)
	if isError {
		m.Errors.Add(1)
	}
}

func (m *Metrics) ErrorRate() float64 {
	total := m.Requests.Load()
	if total == 0 {
		return 0
	}
	return float64(m.Errors.Load()) / float64(total)
}

func (m *Metrics) Availability() float64 {
	return 1 - m.ErrorRate()
}

func (m *Metrics) AvgLatencyMs() float64 {
	total := m.Requests.Load()
	if total == 0 {
		return 0
	}
	return float64(m.LatencyTotalMs.Load()) / float64(total)
}

// ─────────────────────────────────────────────────────────────────────────────
// ERROR BUDGET TRACKER
// ─────────────────────────────────────────────────────────────────────────────

type BudgetTracker struct {
	slo            SLO
	startTime      time.Time
	errorMinutes   float64
	totalMinutes   float64
}

func NewBudgetTracker(slo SLO) *BudgetTracker {
	return &BudgetTracker{slo: slo, startTime: time.Now()}
}

func (bt *BudgetTracker) Advance(durationMin float64, isViolation bool) {
	bt.totalMinutes += durationMin
	if isViolation {
		bt.errorMinutes += durationMin
	}
}

func (bt *BudgetTracker) BudgetTotal() float64    { return bt.slo.errorBudgetMinutes() }
func (bt *BudgetTracker) BudgetUsed() float64     { return bt.errorMinutes }
func (bt *BudgetTracker) BudgetRemaining() float64 { return bt.BudgetTotal() - bt.errorMinutes }
func (bt *BudgetTracker) BudgetRemainingPct() float64 {
	return 100 * bt.BudgetRemaining() / bt.BudgetTotal()
}
func (bt *BudgetTracker) BurnRate() float64 {
	if bt.totalMinutes == 0 {
		return 0
	}
	actualRate := bt.errorMinutes / bt.totalMinutes
	budgetRate := 1 - bt.slo.AvailTarget
	if budgetRate == 0 {
		return 0
	}
	return actualRate / budgetRate
}

// ─────────────────────────────────────────────────────────────────────────────
// POLICY ENGINE
// ─────────────────────────────────────────────────────────────────────────────

type PolicyState int

const (
	PolicyNormal PolicyState = iota
	PolicyWarning
	PolicyFreeze // < 10% budget remaining
)

func (p PolicyState) String() string {
	switch p {
	case PolicyNormal:
		return "NORMAL"
	case PolicyWarning:
		return "WARNING"
	case PolicyFreeze:
		return "FEATURE FREEZE"
	default:
		return "UNKNOWN"
	}
}

func evaluatePolicy(tracker *BudgetTracker) PolicyState {
	pct := tracker.BudgetRemainingPct()
	burnRate := tracker.BurnRate()
	switch {
	case pct < 10 || burnRate > 14.4:
		return PolicyFreeze
	case pct < 30 || burnRate > 6:
		return PolicyWarning
	default:
		return PolicyNormal
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DASHBOARD RENDERER
// ─────────────────────────────────────────────────────────────────────────────

func printDashboard(slo SLO, m *Metrics, tracker *BudgetTracker, policy PolicyState) {
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("  SERVICE: %-20s  Policy: %s\n", slo.ServiceName, policy)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("  SLO Target:       %.3f%%  (latency p99 < %.0fms)\n",
		slo.AvailTarget*100, slo.LatencyP99Ms)
	fmt.Printf("  Availability:     %.4f%%\n", m.Availability()*100)
	fmt.Printf("  Error Rate:       %.4f%%\n", m.ErrorRate()*100)
	fmt.Printf("  Avg Latency:      %.0fms\n", m.AvgLatencyMs())
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("  Error Budget:     %.1f min total\n", tracker.BudgetTotal())
	fmt.Printf("  Budget Used:      %.2f min (%.1f%%)\n",
		tracker.BudgetUsed(), 100-tracker.BudgetRemainingPct())
	fmt.Printf("  Budget Remaining: %.2f min (%.1f%%)\n",
		tracker.BudgetRemaining(), tracker.BudgetRemainingPct())
	fmt.Printf("  Burn Rate:        %.2f× (1.0 = on track)\n", tracker.BurnRate())
	fmt.Println(strings.Repeat("═", 60))
}

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

type simScenario struct {
	label        string
	daysElapsed  float64
	totalReqs    int64
	errorReqs    int64
	avgLatencyMs int64
}

func runSimulation(slo SLO, scenarios []simScenario) {
	tracker := NewBudgetTracker(slo)
	m := &Metrics{}

	for _, sc := range scenarios {
		m.Requests.Store(sc.totalReqs)
		m.Errors.Store(sc.errorReqs)
		m.LatencyTotalMs.Store(sc.avgLatencyMs * sc.totalReqs)

		durationMin := sc.daysElapsed * 24 * 60
		violation := m.Availability() < slo.AvailTarget
		tracker.Advance(durationMin, violation)

		policy := evaluatePolicy(tracker)
		fmt.Printf("\n%s\n", sc.label)
		printDashboard(slo, m, tracker, policy)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 95 Exercise: SLO-Aware Service ===")
	fmt.Println()

	slo := SLO{
		ServiceName:     "checkout-service",
		AvailTarget:     0.999,
		LatencyP99Ms:    200,
		ErrorRateTarget: 0.001,
		WindowDays:      30,
	}

	fmt.Printf("Service: %s | SLO: %.3f%% | Budget: %.1f min/month\n\n",
		slo.ServiceName, slo.AvailTarget*100, slo.errorBudgetMinutes())

	scenarios := []simScenario{
		{"[Day 5] Healthy week", 5, 5_000_000, 300, 45},
		{"[Day 12] Minor degradation", 7, 6_000_000, 3600, 95},
		{"[Day 19] Incident — 5% error rate", 7, 4_000_000, 200_000, 310},
		{"[Day 26] Post-incident recovery", 7, 5_500_000, 550, 52},
		{"[Day 30] Month end", 4, 4_800_000, 480, 48},
	}

	runSimulation(slo, scenarios)

	// ── POLICY SUMMARY ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("Error budget policy:")
	fmt.Println(`  Budget > 30%:   NORMAL — ship freely
  Budget 10-30%:  WARNING — review changes carefully, no risky deploys
  Budget < 10%:   FEATURE FREEZE — only rollbacks and reliability fixes
  Burn rate >14×: PAGE oncall, consider emergency rollback`)
}
