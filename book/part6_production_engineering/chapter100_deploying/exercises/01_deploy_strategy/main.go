// FILE: book/part6_production_engineering/chapter100_deploying/exercises/01_deploy_strategy/main.go
// CHAPTER: 100 — Deployment Strategies
// EXERCISE: Deployment strategy decision engine + full simulation of both
//           blue/green and canary with configurable failure injection.
//
// Run:
//   go run ./book/part6_production_engineering/chapter100_deploying/exercises/01_deploy_strategy

package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// RISK SCORER
// ─────────────────────────────────────────────────────────────────────────────

type ChangeProfile struct {
	Name              string
	FilesChanged      int
	HasDBMigration    bool
	TestCoverage      float64 // 0–100
	TrafficPerSecond  int
	HasFeatureFlag    bool
	RollbackSLAMin    int // maximum acceptable rollback time (minutes)
}

type Strategy string

const (
	StrategyRolling    Strategy = "rolling update"
	StrategyBlueGreen  Strategy = "blue/green"
	StrategyCanary     Strategy = "canary"
	StrategyFeatureFlag Strategy = "feature flag"
)

type Recommendation struct {
	Strategy Strategy
	Score    int // 0–100 risk score
	Reasons  []string
}

func scoreChange(c ChangeProfile) Recommendation {
	score := 0
	var reasons []string

	if c.FilesChanged > 50 {
		score += 20
		reasons = append(reasons, fmt.Sprintf("large diff (%d files)", c.FilesChanged))
	} else if c.FilesChanged > 20 {
		score += 10
		reasons = append(reasons, fmt.Sprintf("medium diff (%d files)", c.FilesChanged))
	}

	if c.HasDBMigration {
		score += 30
		reasons = append(reasons, "has DB migration (rollback requires schema compatibility)")
	}

	if c.TestCoverage < 60 {
		score += 20
		reasons = append(reasons, fmt.Sprintf("low test coverage (%.0f%%)", c.TestCoverage))
	} else if c.TestCoverage < 80 {
		score += 10
		reasons = append(reasons, fmt.Sprintf("moderate test coverage (%.0f%%)", c.TestCoverage))
	}

	if c.TrafficPerSecond > 1000 {
		score += 15
		reasons = append(reasons, fmt.Sprintf("high traffic (%d rps)", c.TrafficPerSecond))
	}

	if c.RollbackSLAMin <= 1 {
		score += 15
		reasons = append(reasons, fmt.Sprintf("strict rollback SLA (%d min)", c.RollbackSLAMin))
	}

	// Determine strategy
	var strategy Strategy
	switch {
	case c.HasFeatureFlag && c.FilesChanged < 20:
		strategy = StrategyFeatureFlag
	case c.HasDBMigration || c.RollbackSLAMin <= 1:
		strategy = StrategyBlueGreen
	case c.TrafficPerSecond >= 500 && score >= 30:
		strategy = StrategyCanary
	case score < 25:
		strategy = StrategyRolling
	default:
		strategy = StrategyBlueGreen
	}

	return Recommendation{Strategy: strategy, Score: score, Reasons: reasons}
}

func printRecommendation(c ChangeProfile, r Recommendation) {
	riskLabel := "LOW"
	if r.Score >= 50 {
		riskLabel = "HIGH"
	} else if r.Score >= 25 {
		riskLabel = "MEDIUM"
	}
	fmt.Printf("  Change:   %s\n", c.Name)
	fmt.Printf("  Risk:     %s (%d/100)\n", riskLabel, r.Score)
	fmt.Printf("  Strategy: %s\n", r.Strategy)
	if len(r.Reasons) > 0 {
		fmt.Printf("  Factors:  %s\n", strings.Join(r.Reasons, "; "))
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// BLUE/GREEN SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

type BGEnv struct {
	Name    string
	Version string
	Healthy bool
}

type BGController struct {
	active  string
	envs    map[string]*BGEnv
	traffic map[string]int
}

func newBGController(stableVersion string) *BGController {
	return &BGController{
		active: "blue",
		envs: map[string]*BGEnv{
			"blue":  {Name: "blue", Version: stableVersion, Healthy: true},
			"green": {Name: "green", Version: "", Healthy: false},
		},
		traffic: map[string]int{"blue": 0, "green": 0},
	}
}

func (bg *BGController) sendTraffic(n int) {
	bg.traffic[bg.active] += n
}

func (bg *BGController) deploy(version string, healthy bool) bool {
	fmt.Printf("    Deploying %s to green...\n", version)
	time.Sleep(10 * time.Millisecond)
	bg.envs["green"].Version = version
	bg.envs["green"].Healthy = healthy

	fmt.Printf("    Running smoke tests against green (%s)...\n", version)
	if !healthy {
		fmt.Printf("    [FAIL] Smoke tests failed — aborting, blue stays active\n")
		return false
	}
	fmt.Printf("    [PASS] All smoke tests passed\n")
	return true
}

func (bg *BGController) cutover() {
	bg.active = "green"
	fmt.Printf("    Cutover: traffic → green (%s)\n", bg.envs["green"].Version)
}

func (bg *BGController) rollback() {
	bg.active = "blue"
	fmt.Printf("    Rollback: traffic → blue (%s)\n", bg.envs["blue"].Version)
}

func (bg *BGController) stats() string {
	return fmt.Sprintf("active=%s  blue=%d requests  green=%d requests",
		bg.active, bg.traffic["blue"], bg.traffic["green"])
}

func simulateBlueGreen(scenario string, newVersion string, greenHealthy bool) {
	fmt.Printf("  [blue/green] Scenario: %s\n", scenario)
	ctrl := newBGController("v2.1.0")
	ctrl.sendTraffic(100)

	ok := ctrl.deploy(newVersion, greenHealthy)
	if ok {
		ctrl.sendTraffic(50) // more traffic before cutover
		ctrl.cutover()
		ctrl.sendTraffic(100)

		// Simulate post-cutover incident check
		if rand.Float64() < 0.3 && scenario == "with incident" {
			fmt.Printf("    [alert] Error rate spiked after cutover\n")
			ctrl.rollback()
			ctrl.sendTraffic(50)
		}
	}
	fmt.Printf("    Final: %s\n\n", ctrl.stats())
}

// ─────────────────────────────────────────────────────────────────────────────
// CANARY SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

type CanaryStep struct {
	Weight     int
	ErrorRate  float64 // simulated
	P99Ms      float64 // simulated
}

func simulateCanary(scenario string, version string, steps []CanaryStep) {
	fmt.Printf("  [canary] Scenario: %s (%s)\n", scenario, version)
	stableTraffic, canaryTraffic := 0, 0
	aborted := false

	for i, step := range steps {
		stageNum := i + 1
		stableTraffic += (100 - step.Weight) * 10
		canaryTraffic += step.Weight * 10

		fmt.Printf("    Stage %d (%d%%)  error=%.2f%%  p99=%.0fms",
			stageNum, step.Weight, step.ErrorRate, step.P99Ms)

		if step.ErrorRate > 0.5 {
			fmt.Printf("  → [GATE FAIL] error_rate > 0.5%%\n")
			fmt.Printf("    Rollback: canary weight → 0%%\n")
			aborted = true
			break
		}
		if step.P99Ms > 200 {
			fmt.Printf("  → [GATE FAIL] p99 > 200ms\n")
			fmt.Printf("    Rollback: canary weight → 0%%\n")
			aborted = true
			break
		}
		fmt.Printf("  → [PASS]\n")
		time.Sleep(5 * time.Millisecond)
	}

	if aborted {
		fmt.Printf("    Canary aborted — stable serves 100%% of traffic\n")
	} else {
		fmt.Printf("    Canary fully promoted — %s serving 100%%\n", version)
	}
	fmt.Printf("    Traffic routed: stable=%d  canary=%d\n\n",
		stableTraffic, canaryTraffic)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 100 Exercise: Deployment Strategy Decision Engine ===")
	fmt.Println()

	// ── RISK SCORING ──────────────────────────────────────────────────────────
	fmt.Println("--- Risk scoring & strategy recommendations ---")
	fmt.Println()
	changes := []ChangeProfile{
		{
			Name: "CSS typo fix", FilesChanged: 2,
			HasDBMigration: false, TestCoverage: 90,
			TrafficPerSecond: 200, HasFeatureFlag: true,
			RollbackSLAMin: 30,
		},
		{
			Name: "Payment service refactor", FilesChanged: 60,
			HasDBMigration: true, TestCoverage: 55,
			TrafficPerSecond: 800, HasFeatureFlag: false,
			RollbackSLAMin: 1,
		},
		{
			Name: "New search algorithm", FilesChanged: 35,
			HasDBMigration: false, TestCoverage: 75,
			TrafficPerSecond: 1500, HasFeatureFlag: false,
			RollbackSLAMin: 5,
		},
		{
			Name: "Config value update", FilesChanged: 1,
			HasDBMigration: false, TestCoverage: 85,
			TrafficPerSecond: 50, HasFeatureFlag: false,
			RollbackSLAMin: 60,
		},
	}
	for _, c := range changes {
		rec := scoreChange(c)
		printRecommendation(c, rec)
	}

	// ── BLUE/GREEN SIMULATIONS ─────────────────────────────────────────────────
	fmt.Println("--- Blue/green simulations ---")
	fmt.Println()
	simulateBlueGreen("clean deploy", "v2.2.0", true)
	simulateBlueGreen("smoke test failure", "v2.3.0-broken", false)

	// ── CANARY SIMULATIONS ─────────────────────────────────────────────────────
	fmt.Println("--- Canary simulations ---")
	fmt.Println()
	simulateCanary("clean rollout", "v3.1.0", []CanaryStep{
		{Weight: 5, ErrorRate: 0.04, P99Ms: 130},
		{Weight: 25, ErrorRate: 0.06, P99Ms: 142},
		{Weight: 50, ErrorRate: 0.08, P99Ms: 148},
		{Weight: 100, ErrorRate: 0.07, P99Ms: 139},
	})

	simulateCanary("error spike at stage 2", "v3.2.0-buggy", []CanaryStep{
		{Weight: 5, ErrorRate: 0.05, P99Ms: 135},
		{Weight: 25, ErrorRate: 1.8, P99Ms: 145}, // error gate fails
		{Weight: 50, ErrorRate: 1.9, P99Ms: 140},
		{Weight: 100, ErrorRate: 1.7, P99Ms: 138},
	})

	simulateCanary("latency spike at stage 3", "v3.3.0-slow", []CanaryStep{
		{Weight: 5, ErrorRate: 0.03, P99Ms: 145},
		{Weight: 25, ErrorRate: 0.04, P99Ms: 160},
		{Weight: 50, ErrorRate: 0.05, P99Ms: 310}, // p99 gate fails
		{Weight: 100, ErrorRate: 0.04, P99Ms: 315},
	})

	// ── DECISION FRAMEWORK ────────────────────────────────────────────────────
	fmt.Println("--- Decision framework summary ---")
	fmt.Println(`  ┌─────────────────────────────────────────────────────────────────┐
  │ Change type                  → Recommended strategy             │
  ├─────────────────────────────────────────────────────────────────┤
  │ Single config / flag change  → feature flag (instant rollback)  │
  │ Small change, low risk       → rolling update                   │
  │ Has DB migration             → blue/green (schema isolation)    │
  │ Strict rollback SLA (< 1min) → blue/green (flip LB back)        │
  │ High traffic, complex change → canary (gate per stage)          │
  │ Cannot afford 2× infra cost  → canary or rolling update         │
  └─────────────────────────────────────────────────────────────────┘`)
}
