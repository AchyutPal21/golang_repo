// FILE: book/part6_production_engineering/chapter100_deploying/examples/01_blue_green/main.go
// CHAPTER: 100 — Deployment Strategies
// TOPIC: Blue/green deployment — parallel environments, smoke tests,
//        atomic traffic cutover, and instant rollback.
//
// Run:
//   go run ./book/part6_production_engineering/chapter100_deploying/examples/01_blue_green

package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// ENVIRONMENT
// ─────────────────────────────────────────────────────────────────────────────

type EnvColor string

const (
	Blue  EnvColor = "blue"
	Green EnvColor = "green"
)

type Environment struct {
	Color    EnvColor
	Version  string
	Healthy  bool
	Replicas int
}

func (e Environment) String() string {
	health := "healthy"
	if !e.Healthy {
		health = "UNHEALTHY"
	}
	return fmt.Sprintf("%s (version=%s, replicas=%d, %s)", e.Color, e.Version, e.Replicas, health)
}

// ─────────────────────────────────────────────────────────────────────────────
// LOAD BALANCER
// ─────────────────────────────────────────────────────────────────────────────

type LoadBalancer struct {
	mu     sync.RWMutex
	active EnvColor
	counts map[EnvColor]*atomic.Int64
}

func NewLoadBalancer(active EnvColor) *LoadBalancer {
	return &LoadBalancer{
		active: active,
		counts: map[EnvColor]*atomic.Int64{
			Blue:  {},
			Green: {},
		},
	}
}

func (lb *LoadBalancer) Route() EnvColor {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	active := lb.active
	lb.counts[active].Add(1)
	return active
}

func (lb *LoadBalancer) Switch(to EnvColor) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.active = to
}

func (lb *LoadBalancer) Active() EnvColor {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.active
}

func (lb *LoadBalancer) Stats() string {
	return fmt.Sprintf("blue=%d green=%d",
		lb.counts[Blue].Load(), lb.counts[Green].Load())
}

// ─────────────────────────────────────────────────────────────────────────────
// SMOKE TEST SUITE
// ─────────────────────────────────────────────────────────────────────────────

type SmokeTest struct {
	Name   string
	Target EnvColor
	Check  func(env Environment) error
}

func runSmokeTests(tests []SmokeTest, env Environment) (passed, failed int) {
	for _, t := range tests {
		err := t.Check(env)
		if err != nil {
			fmt.Printf("    [FAIL] %s: %v\n", t.Name, err)
			failed++
		} else {
			fmt.Printf("    [PASS] %s\n", t.Name)
			passed++
		}
	}
	return
}

func buildSmokeTests() []SmokeTest {
	return []SmokeTest{
		{"GET /healthz returns 200", Green, func(e Environment) error {
			if !e.Healthy {
				return errors.New("healthz: 503")
			}
			return nil
		}},
		{"GET /readyz returns 200", Green, func(e Environment) error {
			if !e.Healthy {
				return errors.New("readyz: 503")
			}
			return nil
		}},
		{"GET /api/orders returns 200", Green, func(e Environment) error {
			if e.Version == "" {
				return errors.New("no version tag")
			}
			return nil
		}},
		{"POST /api/orders accepts payload", Green, func(e Environment) error {
			return nil // always passes in this simulation
		}},
		{"Database migration complete", Green, func(e Environment) error {
			return nil
		}},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// BLUE/GREEN CONTROLLER
// ─────────────────────────────────────────────────────────────────────────────

type BlueGreenController struct {
	lb   *LoadBalancer
	envs map[EnvColor]*Environment
}

func NewController(blueVersion, greenVersion string) *BlueGreenController {
	return &BlueGreenController{
		lb: NewLoadBalancer(Blue),
		envs: map[EnvColor]*Environment{
			Blue:  {Blue, blueVersion, true, 5},
			Green: {Green, greenVersion, false, 0},
		},
	}
}

func (c *BlueGreenController) printState(label string) {
	fmt.Printf("  [%s]\n", label)
	fmt.Printf("    Active: %s\n", c.lb.Active())
	fmt.Printf("    Blue:  %s\n", c.envs[Blue])
	fmt.Printf("    Green: %s\n", c.envs[Green])
	fmt.Printf("    LB traffic: %s\n", c.lb.Stats())
}

func (c *BlueGreenController) Deploy(newVersion string) error {
	fmt.Printf("\n  Deploying %s to green environment...\n", newVersion)
	time.Sleep(20 * time.Millisecond)
	c.envs[Green].Version = newVersion
	c.envs[Green].Replicas = 5
	c.envs[Green].Healthy = true
	fmt.Printf("  Green environment ready: %s\n", c.envs[Green])

	fmt.Println("\n  Running smoke tests against green...")
	tests := buildSmokeTests()
	passed, failed := runSmokeTests(tests, *c.envs[Green])
	fmt.Printf("  Smoke tests: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		c.envs[Green].Healthy = false
		c.envs[Green].Replicas = 0
		return fmt.Errorf("smoke tests failed: aborting deploy")
	}
	return nil
}

func (c *BlueGreenController) Cutover() {
	fmt.Println("\n  Cutting over: blue → green")
	c.lb.Switch(Green)
	fmt.Printf("  Traffic now routing to: %s\n", c.lb.Active())
}

func (c *BlueGreenController) Rollback() {
	fmt.Println("\n  ROLLBACK: green → blue")
	c.lb.Switch(Blue)
	fmt.Printf("  Traffic restored to: %s\n", c.lb.Active())
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 100: Blue/Green Deployment ===")
	fmt.Println()

	ctrl := NewController("v1.2.3", "")

	// ── INITIAL STATE ─────────────────────────────────────────────────────────
	ctrl.printState("Initial state")

	// Simulate some live traffic hitting blue
	for i := 0; i < 10; i++ {
		ctrl.lb.Route()
	}

	// ── DEPLOY TO GREEN ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Deploying v1.3.0 to green ---")
	if err := ctrl.Deploy("v1.3.0"); err != nil {
		fmt.Printf("  Deploy failed: %v\n", err)
		return
	}
	ctrl.printState("After deploy to green")

	// ── CUTOVER ───────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Cutting over to green ---")
	start := time.Now()
	ctrl.Cutover()
	for i := 0; i < 10; i++ {
		ctrl.lb.Route()
	}
	fmt.Printf("  Cutover took: %v\n", time.Since(start).Round(time.Microsecond))
	ctrl.printState("After cutover")

	// ── ROLLBACK SCENARIO ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Simulating post-cutover incident → rollback ---")
	fmt.Println("  [alert] Error rate spiked to 8% on green")
	start = time.Now()
	ctrl.Rollback()
	fmt.Printf("  Rollback took: %v\n", time.Since(start).Round(time.Microsecond))
	ctrl.printState("After rollback")

	// ── FAILED SMOKE TEST SCENARIO ────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Deploy with smoke test failure ---")
	ctrl2 := NewController("v1.2.3", "")
	ctrl2.envs[Green].Healthy = false // force unhealthy
	err := ctrl2.Deploy("v1.4.0-broken")
	if err != nil {
		fmt.Printf("  Deploy aborted: %v\n", err)
		fmt.Println("  Blue remains active — no impact to production traffic.")
	}
	fmt.Println()

	// ── COMPARISON TABLE ──────────────────────────────────────────────────────
	fmt.Println("--- Blue/green pros and cons ---")
	fmt.Println(`  Pros:
    • Instant rollback (< 60s) — just flip the LB
    • Full environment parity — green is identical to blue
    • Smoke tests before any live traffic hits new version
    • Database migration can be tested in isolation

  Cons:
    • 2× infrastructure cost during deploy window
    • Stateful services are hard: sessions, in-flight transactions
    • DB schema changes that aren't backward-compatible break rollback
    • Requires idempotent DB migrations (additive only)

  When to use:
    • Services with strict rollback SLA (< 1 minute)
    • After a database migration
    • Initial deploy to production of a new service
    • When canary traffic analysis is not possible (low traffic)`)
	fmt.Println()

	// ── KUBERNETES MANIFESTS REFERENCE ───────────────────────────────────────
	fmt.Println("--- Kubernetes blue/green reference ---")
	fmt.Println(strings.TrimSpace(`  # Two Deployments (blue and green), one Service
  kubectl apply -f deploy-green.yaml          # deploy new version
  kubectl rollout status deployment/app-green # wait for ready
  # Run smoke tests against green ClusterIP
  kubectl patch svc app -p '{"spec":{"selector":{"slot":"green"}}}'  # cutover
  # Rollback:
  kubectl patch svc app -p '{"spec":{"selector":{"slot":"blue"}}}'`))
}
