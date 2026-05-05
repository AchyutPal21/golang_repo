// FILE: book/part6_production_engineering/chapter93_kubernetes/examples/02_probes_hpa/main.go
// CHAPTER: 93 — Kubernetes for Go Services
// TOPIC: Probe timing interactions, HPA desired-replica calculation,
//        rolling update availability guarantees.
//
// Run:
//   go run ./book/part6_production_engineering/chapter93_kubernetes/examples/02_probes_hpa

package main

import (
	"fmt"
	"math"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// PROBE TIMING CALCULATOR
// ─────────────────────────────────────────────────────────────────────────────

type ProbeConfig struct {
	Name                string
	InitialDelaySeconds int
	PeriodSeconds       int
	FailureThreshold    int
	TimeoutSeconds      int
}

// firstDetectionTime returns the worst-case seconds until a failure is detected.
func (p ProbeConfig) firstDetectionTime() int {
	return p.InitialDelaySeconds + p.PeriodSeconds*p.FailureThreshold + p.TimeoutSeconds*p.FailureThreshold
}

// startupWindow returns the total time before liveness kicks in (startup probe).
func startupWindow(startup ProbeConfig) int {
	return startup.InitialDelaySeconds + startup.PeriodSeconds*startup.FailureThreshold
}

func printProbeAnalysis(startup, liveness, readiness ProbeConfig) {
	fmt.Printf("  %-12s  initialDelay=%ds  period=%ds  failThresh=%d  → detect failure in ≤%ds\n",
		startup.Name, startup.InitialDelaySeconds, startup.PeriodSeconds, startup.FailureThreshold,
		startup.firstDetectionTime())
	fmt.Printf("  %-12s  initialDelay=%ds  period=%ds  failThresh=%d  → detect failure in ≤%ds\n",
		liveness.Name, liveness.InitialDelaySeconds, liveness.PeriodSeconds, liveness.FailureThreshold,
		liveness.firstDetectionTime())
	fmt.Printf("  %-12s  initialDelay=%ds  period=%ds  failThresh=%d  → detect failure in ≤%ds\n",
		readiness.Name, readiness.InitialDelaySeconds, readiness.PeriodSeconds, readiness.FailureThreshold,
		readiness.firstDetectionTime())
	fmt.Printf("  Startup window: %ds (liveness starts after startup passes)\n", startupWindow(startup))
}

// ─────────────────────────────────────────────────────────────────────────────
// HPA CALCULATOR
// ─────────────────────────────────────────────────────────────────────────────

func hpaDesiredReplicas(current int, currentPct, targetPct float64) int {
	desired := math.Ceil(float64(current) * currentPct / targetPct)
	if desired < 1 {
		desired = 1
	}
	return int(desired)
}

type HPAScenario struct {
	Description string
	Current     int
	CurrentPct  float64
	TargetPct   float64
	Min, Max    int
}

func (s HPAScenario) compute() int {
	desired := hpaDesiredReplicas(s.Current, s.CurrentPct, s.TargetPct)
	if desired < s.Min {
		return s.Min
	}
	if desired > s.Max {
		return s.Max
	}
	return desired
}

// ─────────────────────────────────────────────────────────────────────────────
// ROLLING UPDATE SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

type PodState struct {
	Name    string
	Version string
	State   string // Running, Terminating, Starting, Ready
}

func simulateRollingUpdate(desired int, maxSurge, maxUnavailable int) {
	fmt.Printf("  Rolling update: desired=%d maxSurge=%d maxUnavailable=%d\n",
		desired, maxSurge, maxUnavailable)
	fmt.Println("  " + strings.Repeat("-", 60))

	// Build initial pod set
	pods := make([]PodState, desired)
	for i := range pods {
		pods[i] = PodState{fmt.Sprintf("pod-%d", i+1), "v1", "Running"}
	}

	step := 0
	printPods := func(label string) {
		step++
		fmt.Printf("  Step %d: %s\n", step, label)
		for _, p := range pods {
			fmt.Printf("    %-8s  %-3s  %s\n", p.Name, p.Version, p.State)
		}
		available := 0
		for _, p := range pods {
			if p.State == "Running" || p.State == "Ready" {
				available++
			}
		}
		fmt.Printf("    Available: %d/%d (min required: %d)\n", available, desired, desired-maxUnavailable)
		fmt.Println()
	}

	printPods("Initial state (all v1)")

	// Simulate update: add surge pods, then remove old ones
	for i := 0; i < desired; i++ {
		// Start a new v2 pod (surge)
		pods = append(pods, PodState{fmt.Sprintf("pod-%d-new", i+1), "v2", "Starting"})
		printPods(fmt.Sprintf("Started new v2 pod (surge +1)"))

		// Mark new pod ready
		pods[len(pods)-1].State = "Ready"
		printPods("New pod is Ready")

		// Terminate old v1 pod
		pods[i].State = "Terminating"
		printPods(fmt.Sprintf("Terminating old pod %s", pods[i].Name))

		// Remove terminated pod
		pods = append(pods[:i], pods[i+1:]...)
		// Rename for clarity
		pods[len(pods)-1].State = "Running"
		pods[len(pods)-1].Name = fmt.Sprintf("pod-%d", i+1)
	}
	printPods("Update complete (all v2)")
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 93: Probes, HPA & Rolling Updates ===")
	fmt.Println()

	// ── PROBE ANALYSIS ────────────────────────────────────────────────────────
	fmt.Println("--- Probe timing analysis ---")
	startup := ProbeConfig{"startup", 0, 2, 30, 1}    // 60s window
	liveness := ProbeConfig{"liveness", 0, 10, 3, 1}  // runs after startup passes
	readiness := ProbeConfig{"readiness", 0, 5, 2, 1} // traffic removed in ≤12s
	printProbeAnalysis(startup, liveness, readiness)
	fmt.Println()

	// ── HPA SCENARIOS ─────────────────────────────────────────────────────────
	fmt.Println("--- HPA desired-replica scenarios ---")
	scenarios := []HPAScenario{
		{"Scale up: 3 pods at 85% (target 70%)", 3, 85, 70, 2, 10},
		{"Scale down: 5 pods at 40% (target 70%)", 5, 40, 70, 2, 10},
		{"At target: 4 pods at 70%", 4, 70, 70, 2, 10},
		{"Hit max: 8 pods at 95%", 8, 95, 70, 2, 10},
		{"Hit min: 2 pods at 10%", 2, 10, 70, 2, 10},
	}
	fmt.Printf("  %-42s  %7s  %7s  %8s  %s\n", "Scenario", "Current", "CPU%", "Target%", "Desired")
	fmt.Printf("  %s\n", strings.Repeat("-", 80))
	for _, s := range scenarios {
		raw := hpaDesiredReplicas(s.Current, s.CurrentPct, s.TargetPct)
		clamped := s.compute()
		clampNote := ""
		if raw != clamped {
			clampNote = fmt.Sprintf(" (raw=%d, clamped by min/max)", raw)
		}
		fmt.Printf("  %-42s  %7d  %6.0f%%  %7.0f%%  %d%s\n",
			s.Description, s.Current, s.CurrentPct, s.TargetPct, clamped, clampNote)
	}
	fmt.Println()

	// ── ROLLING UPDATE ────────────────────────────────────────────────────────
	fmt.Println("--- Rolling update simulation (3 replicas, maxSurge=1, maxUnavailable=0) ---")
	simulateRollingUpdate(3, 1, 0)
}
