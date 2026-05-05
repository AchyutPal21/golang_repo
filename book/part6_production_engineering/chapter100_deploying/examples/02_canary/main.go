// FILE: book/part6_production_engineering/chapter100_deploying/examples/02_canary/main.go
// CHAPTER: 100 — Deployment Strategies
// TOPIC: Canary deployment — progressive traffic splitting, metric gates,
//        automatic rollback, and promotion to 100%.
//
// Run:
//   go run ./book/part6_production_engineering/chapter100_deploying/examples/02_canary

package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// METRIC SNAPSHOT
// ─────────────────────────────────────────────────────────────────────────────

type MetricSnapshot struct {
	Stage        int
	ErrorRatePct float64 // 0–100
	P99LatencyMs float64
	RequestCount int
}

func (m MetricSnapshot) String() string {
	return fmt.Sprintf("stage=%d requests=%d error=%.2f%% p99=%.0fms",
		m.Stage, m.RequestCount, m.ErrorRatePct, m.P99LatencyMs)
}

// ─────────────────────────────────────────────────────────────────────────────
// METRIC GATE
// ─────────────────────────────────────────────────────────────────────────────

type MetricGate struct {
	Name  string
	Check func(MetricSnapshot) error
}

func ErrorRateGate(maxPct float64) MetricGate {
	return MetricGate{
		Name: fmt.Sprintf("error_rate < %.1f%%", maxPct),
		Check: func(m MetricSnapshot) error {
			if m.ErrorRatePct > maxPct {
				return fmt.Errorf("error rate %.2f%% exceeds max %.1f%%", m.ErrorRatePct, maxPct)
			}
			return nil
		},
	}
}

func LatencyP99Gate(maxMs float64) MetricGate {
	return MetricGate{
		Name: fmt.Sprintf("p99_latency < %.0fms", maxMs),
		Check: func(m MetricSnapshot) error {
			if m.P99LatencyMs > maxMs {
				return fmt.Errorf("p99 %.0fms exceeds max %.0fms", m.P99LatencyMs, maxMs)
			}
			return nil
		},
	}
}

func runGates(gates []MetricGate, snap MetricSnapshot) (passed, failed int, firstFailure string) {
	for _, g := range gates {
		if err := g.Check(snap); err != nil {
			failed++
			if firstFailure == "" {
				firstFailure = fmt.Sprintf("%s: %v", g.Name, err)
			}
		} else {
			passed++
		}
	}
	return
}

// ─────────────────────────────────────────────────────────────────────────────
// CANARY STAGE
// ─────────────────────────────────────────────────────────────────────────────

type CanaryStage struct {
	WeightPct    int           // percentage of traffic routed to canary
	MonitorFor   time.Duration // how long to observe before gating
	Gates        []MetricGate
}

func defaultStages() []CanaryStage {
	return []CanaryStage{
		{
			WeightPct:  5,
			MonitorFor: 10 * time.Millisecond, // compressed for simulation
			Gates:      []MetricGate{ErrorRateGate(0.5), LatencyP99Gate(200)},
		},
		{
			WeightPct:  25,
			MonitorFor: 10 * time.Millisecond,
			Gates:      []MetricGate{ErrorRateGate(0.5), LatencyP99Gate(200)},
		},
		{
			WeightPct:  50,
			MonitorFor: 10 * time.Millisecond,
			Gates:      []MetricGate{ErrorRateGate(0.5), LatencyP99Gate(200)},
		},
		{
			WeightPct:  100,
			MonitorFor: 10 * time.Millisecond,
			Gates:      []MetricGate{ErrorRateGate(0.5), LatencyP99Gate(200)},
		},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TRAFFIC SPLITTER
// ─────────────────────────────────────────────────────────────────────────────

type TrafficSplitter struct {
	weight  atomic.Int64 // % to canary (0–100)
	stable  atomic.Int64
	canary  atomic.Int64
}

func NewTrafficSplitter() *TrafficSplitter { return &TrafficSplitter{} }

func (ts *TrafficSplitter) SetWeight(pct int) {
	if pct < 0 {
		pct = 0
	} else if pct > 100 {
		pct = 100
	}
	ts.weight.Store(int64(pct))
}

// Route returns true if this request should go to canary.
func (ts *TrafficSplitter) Route() bool {
	w := int(ts.weight.Load())
	toCanary := rand.Intn(100) < w
	if toCanary {
		ts.canary.Add(1)
	} else {
		ts.stable.Add(1)
	}
	return toCanary
}

func (ts *TrafficSplitter) Stats() string {
	return fmt.Sprintf("stable=%d canary=%d weight=%d%%",
		ts.stable.Load(), ts.canary.Load(), ts.weight.Load())
}

// ─────────────────────────────────────────────────────────────────────────────
// METRIC COLLECTOR (simulated)
// ─────────────────────────────────────────────────────────────────────────────

type MetricCollector struct {
	injectFailureAtStage int // 0 = no failure
}

func (mc *MetricCollector) Collect(stage, weightPct int) MetricSnapshot {
	// Base healthy metrics
	errorRate := 0.05 + rand.Float64()*0.05 // 0.05–0.10%
	p99 := 120.0 + rand.Float64()*30        // 120–150ms
	requests := weightPct * 100

	if mc.injectFailureAtStage != 0 && stage >= mc.injectFailureAtStage {
		errorRate = 2.0 + rand.Float64()*3 // spike: 2–5%
		p99 = 350 + rand.Float64()*100     // spike: 350–450ms
	}

	return MetricSnapshot{
		Stage:        stage,
		ErrorRatePct: errorRate,
		P99LatencyMs: p99,
		RequestCount: requests,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CANARY CONTROLLER
// ─────────────────────────────────────────────────────────────────────────────

type CanaryResult struct {
	Promoted bool
	Reason   string // empty on success
	Stage    int    // last stage attempted
}

type CanaryController struct {
	splitter  *TrafficSplitter
	collector *MetricCollector
	stages    []CanaryStage
	version   string
}

func NewCanaryController(version string, failAtStage int) *CanaryController {
	return &CanaryController{
		splitter:  NewTrafficSplitter(),
		collector: &MetricCollector{injectFailureAtStage: failAtStage},
		stages:    defaultStages(),
		version:   version,
	}
}

func (cc *CanaryController) Run() CanaryResult {
	fmt.Printf("  Starting canary rollout for %s\n", cc.version)
	fmt.Printf("  Stages: %s\n\n",
		strings.Join(func() []string {
			var s []string
			for _, st := range cc.stages {
				s = append(s, fmt.Sprintf("%d%%", st.WeightPct))
			}
			return s
		}(), " → "))

	for i, stage := range cc.stages {
		stageNum := i + 1
		fmt.Printf("  ── Stage %d: %d%% traffic to canary ──\n", stageNum, stage.WeightPct)
		cc.splitter.SetWeight(stage.WeightPct)

		// Simulate traffic during monitoring window
		for j := 0; j < 20; j++ {
			cc.splitter.Route()
		}

		time.Sleep(stage.MonitorFor)

		// Collect metrics
		snap := cc.collector.Collect(stageNum, stage.WeightPct)
		fmt.Printf("    Metrics:  %s\n", snap)

		// Evaluate gates
		passed, failed, reason := runGates(stage.Gates, snap)
		fmt.Printf("    Gates:    %d passed, %d failed\n", passed, failed)

		if failed > 0 {
			fmt.Printf("    [GATE FAIL] %s\n", reason)
			cc.splitter.SetWeight(0)
			fmt.Printf("    Rollback: weight → 0%% (stable gets 100%% of traffic)\n")
			fmt.Printf("    Traffic:  %s\n", cc.splitter.Stats())
			return CanaryResult{Promoted: false, Reason: reason, Stage: stageNum}
		}

		fmt.Printf("    [GATE PASS] promoting to next stage\n")
		fmt.Printf("    Traffic:  %s\n\n", cc.splitter.Stats())
	}

	return CanaryResult{Promoted: true, Stage: len(cc.stages)}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 100: Canary Deployment ===")
	fmt.Println()

	// ── HEALTHY ROLLOUT ───────────────────────────────────────────────────────
	fmt.Println("--- Healthy canary rollout (no failures) ---")
	ctrl := NewCanaryController("v1.3.0", 0)
	result := ctrl.Run()
	if result.Promoted {
		fmt.Printf("  Canary promoted at stage %d — 100%% traffic on v1.3.0\n", result.Stage)
	}
	fmt.Println()

	// ── FAILING ROLLOUT ───────────────────────────────────────────────────────
	fmt.Println("--- Canary rollout with stage-2 failure (error spike) ---")
	ctrl2 := NewCanaryController("v1.4.0-buggy", 2)
	result2 := ctrl2.Run()
	if !result2.Promoted {
		fmt.Printf("  Rollback at stage %d: %s\n", result2.Stage, result2.Reason)
		fmt.Println("  Stable version continues serving 100% of traffic.")
	}
	fmt.Println()

	// ── METRIC GATE DEMO ──────────────────────────────────────────────────────
	fmt.Println("--- Metric gate evaluation examples ---")
	gates := []MetricGate{ErrorRateGate(0.5), LatencyP99Gate(200)}
	scenarios := []struct {
		label string
		snap  MetricSnapshot
	}{
		{"healthy",   MetricSnapshot{Stage: 1, ErrorRatePct: 0.08, P99LatencyMs: 145, RequestCount: 500}},
		{"high error", MetricSnapshot{Stage: 1, ErrorRatePct: 1.2, P99LatencyMs: 145, RequestCount: 500}},
		{"high p99",   MetricSnapshot{Stage: 2, ErrorRatePct: 0.05, P99LatencyMs: 280, RequestCount: 2500}},
		{"both bad",   MetricSnapshot{Stage: 3, ErrorRatePct: 3.5, P99LatencyMs: 420, RequestCount: 5000}},
	}
	for _, sc := range scenarios {
		_, failed, reason := runGates(gates, sc.snap)
		status := "PASS"
		detail := ""
		if failed > 0 {
			status = "FAIL"
			detail = " — " + reason
		}
		fmt.Printf("  [%s] %-10s %s%s\n", status, sc.label, sc.snap, detail)
	}
	fmt.Println()

	// ── TRAFFIC SPLITTER DEMO ─────────────────────────────────────────────────
	fmt.Println("--- Traffic splitter weight changes ---")
	ts := NewTrafficSplitter()
	for _, w := range []int{0, 5, 25, 50, 100} {
		ts.SetWeight(w)
		for i := 0; i < 100; i++ {
			ts.Route()
		}
		fmt.Printf("  weight=%3d%%  %s\n", w, ts.Stats())
	}
	fmt.Println()

	// ── COMPARISON ────────────────────────────────────────────────────────────
	fmt.Println("--- Canary vs blue/green comparison ---")
	fmt.Println(`  Canary:
    • Traffic split: 5% → 25% → 50% → 100% (gradual)
    • Rollback: set weight=0 (seconds, no LB flip needed)
    • Cost: 1.05× infrastructure (tiny canary pool)
    • Risk: only a fraction of users hit the bad version
    • Requirement: app must handle mixed-version traffic

  Blue/Green:
    • Traffic split: 0% → 100% (atomic cutover)
    • Rollback: flip LB back (< 60s, clean)
    • Cost: 2× infrastructure during deploy window
    • Risk: all users affected if green is bad (but smoke tests help)
    • Requirement: full parallel environment must be healthy first

  Rule of thumb:
    • High traffic, risk-averse → canary
    • Database migration, strict rollback SLA → blue/green
    • Small change, low traffic → rolling update`)
	fmt.Println()

	// ── KUBERNETES REFERENCE ──────────────────────────────────────────────────
	fmt.Println("--- Argo Rollouts canary reference ---")
	fmt.Println(strings.TrimSpace(`  # Argo Rollouts canary strategy
  kubectl argo rollouts get rollout my-app --watch
  kubectl argo rollouts promote my-app     # manual gate pass
  kubectl argo rollouts abort my-app       # instant rollback to 0%

  # Check canary metrics
  kubectl argo rollouts status my-app

  # Manual weight override
  kubectl argo rollouts set weight my-app 50`))
}
