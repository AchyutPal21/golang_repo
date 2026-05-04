// FILE: book/part6_production_engineering/chapter90_prometheus/exercises/01_cardinality/main.go
// CHAPTER: 90 — Prometheus Metrics
// EXERCISE: Demonstrate the cardinality explosion problem and implement a
//   cardinality-safe label registry that enforces label value limits.
//
// Run:
//   go run ./part6_production_engineering/chapter90_prometheus/exercises/01_cardinality/

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// CARDINALITY-SAFE LABEL REGISTRY
// ─────────────────────────────────────────────────────────────────────────────

// LabelPolicy enforces a maximum number of distinct values per label key.
type LabelPolicy struct {
	mu       sync.Mutex
	limits   map[string]int          // key → max distinct values
	seen     map[string]map[string]bool // key → set of seen values
	overflow map[string]int64        // key → overflow count
}

const overflowLabel = "__overflow__"

func NewLabelPolicy(limits map[string]int) *LabelPolicy {
	return &LabelPolicy{
		limits:   limits,
		seen:     make(map[string]map[string]bool),
		overflow: make(map[string]int64),
	}
}

// Sanitise returns the safe value for a label key/value pair.
// If the value would exceed the cardinality limit, returns overflowLabel.
func (lp *LabelPolicy) Sanitise(key, value string) string {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	limit, ok := lp.limits[key]
	if !ok {
		return value // no limit for this key
	}
	if lp.seen[key] == nil {
		lp.seen[key] = make(map[string]bool)
	}
	if lp.seen[key][value] {
		return value // already known
	}
	if len(lp.seen[key]) >= limit {
		lp.overflow[key]++
		return overflowLabel
	}
	lp.seen[key][value] = true
	return value
}

// Stats returns per-key cardinality and overflow counts.
func (lp *LabelPolicy) Stats() map[string]struct{ distinct, overflow int } {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	result := make(map[string]struct{ distinct, overflow int })
	for k, s := range lp.seen {
		result[k] = struct{ distinct, overflow int }{len(s), int(lp.overflow[k])}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// COUNTER WITH CARDINALITY POLICY
// ─────────────────────────────────────────────────────────────────────────────

type labelKey struct{ method, path, status string }

type SafeCounter struct {
	name   string
	policy *LabelPolicy
	mu     sync.Mutex
	series map[labelKey]*atomic.Int64
}

func NewSafeCounter(name string, policy *LabelPolicy) *SafeCounter {
	return &SafeCounter{
		name:   name,
		policy: policy,
		series: make(map[labelKey]*atomic.Int64),
	}
}

func (c *SafeCounter) Inc(method, path, status string) {
	k := labelKey{
		method: c.policy.Sanitise("method", method),
		path:   c.policy.Sanitise("path", path),
		status: c.policy.Sanitise("status", status),
	}
	c.mu.Lock()
	if c.series[k] == nil {
		c.series[k] = new(atomic.Int64)
	}
	v := c.series[k]
	c.mu.Unlock()
	v.Add(1)
}

func (c *SafeCounter) NumSeries() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.series)
}

func (c *SafeCounter) Print() {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Printf("  Metric: %s (%d series)\n", c.name, len(c.series))
	for k, v := range c.series {
		fmt.Printf("    {method=%q,path=%q,status=%q} = %d\n",
			k.method, k.path, k.status, v.Load())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CARDINALITY EXPLOSION SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

// simulateExplosion shows what happens without cardinality control.
func simulateExplosion(numUsers int) int {
	// Suppose someone naively uses user_id as a label.
	type series struct{ status, userID string }
	seen := make(map[series]bool)
	for i := 0; i < numUsers; i++ {
		seen[series{"200", fmt.Sprintf("user-%d", i)}] = true
	}
	return len(seen)
}

// simulateSafe shows the safe version with cardinality policy.
func simulateSafe(policy *LabelPolicy, numUsers int) int {
	type series struct{ status, userID string }
	seen := make(map[series]bool)
	for i := 0; i < numUsers; i++ {
		userID := policy.Sanitise("user_id", fmt.Sprintf("user-%d", i))
		seen[series{"200", userID}] = true
	}
	return len(seen)
}

// ─────────────────────────────────────────────────────────────────────────────
// MEMORY COST ESTIMATE
// ─────────────────────────────────────────────────────────────────────────────

// Prometheus stores ~1 KB per time series in memory.
const bytesPerSeries = 1024 // bytes

func memMB(series int) float64 {
	return float64(series*bytesPerSeries) / 1024 / 1024
}

// ─────────────────────────────────────────────────────────────────────────────
// ANTI-PATTERN DEMONSTRATION
// ─────────────────────────────────────────────────────────────────────────────

const antiPatterns = `
Common cardinality anti-patterns:

  Anti-pattern 1: using request_id as a label
    http_requests_total{request_id="abc-123"} ← NEVER
    → Millions of unique series, OOM Prometheus.

  Anti-pattern 2: using user_id as a label
    http_requests_total{user_id="42"} ← NEVER
    → Use logs or traces for per-user investigation instead.

  Anti-pattern 3: unbounded error messages
    errors_total{message="connection refused: tcp 10.0.0.1:5432 (attempt 7)"} ← NEVER
    → Normalise to: errors_total{type="connection_refused"}

  Anti-pattern 4: full URL path with IDs
    http_requests_total{path="/api/users/12345/orders/99"} ← NEVER
    → Normalise to: {path="/api/users/:id/orders/:id"}

  Safe patterns:
    http_requests_total{method="GET", status_class="2xx", service="api"}
    db_queries_total{query_type="SELECT", table="orders"}
    cache_operations_total{operation="get", result="hit"}
`

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 90 Exercise: Label Cardinality ===")
	fmt.Println()

	// ── EXPLOSION DEMO ────────────────────────────────────────────────────────
	fmt.Println("--- Cardinality explosion without control ---")
	for _, n := range []int{100, 1_000, 10_000, 100_000} {
		series := simulateExplosion(n)
		fmt.Printf("  %7d users → %7d series (~%.1f MB)\n", n, series, memMB(series))
	}
	fmt.Println()

	// ── SAFE COUNTER ──────────────────────────────────────────────────────────
	fmt.Println("--- SafeCounter with cardinality policy ---")
	policy := NewLabelPolicy(map[string]int{
		"method": 5,
		"path":   20,
		"status": 10,
		"user_id": 0, // 0 = unlimited (not a label here, just policy placeholder)
	})

	sc := NewSafeCounter("http_requests_total", policy)

	// Simulate realistic traffic
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	paths := []string{"/api/users", "/api/orders", "/health", "/metrics",
		"/api/payments", "/api/products", "/api/reviews"}
	statuses := []string{"200", "201", "400", "404", "500"}

	for i := 0; i < 1000; i++ {
		m := methods[i%len(methods)]
		p := paths[i%len(paths)]
		s := statuses[i%len(statuses)]
		sc.Inc(m, p, s)
	}
	sc.Print()
	fmt.Printf("  Series count: %d (bounded)\n", sc.NumSeries())
	fmt.Println()

	// ── OVERFLOW DEMONSTRATION ────────────────────────────────────────────────
	fmt.Println("--- Overflow: too many path variants ---")
	policy2 := NewLabelPolicy(map[string]int{
		"path": 3, // limit to 3 distinct path values
	})
	sc2 := NewSafeCounter("requests_total", policy2)
	paths2 := []string{
		"/api/users", "/api/orders", "/api/payments",
		"/api/items/1", "/api/items/2", "/api/items/3", // overflow
	}
	for i, p := range paths2 {
		sc2.Inc("GET", p, "200")
		fmt.Printf("  Inc path=%q → sanitised=%q\n",
			p, policy2.Sanitise("path", p))
		_ = i
	}
	fmt.Println()
	stats := policy2.Stats()
	for k, s := range stats {
		fmt.Printf("  Label %q: distinct=%d overflow_count=%d\n", k, s.distinct, s.overflow)
	}
	fmt.Printf("  Series with overflow: %d (bounded at 3+1=%d)\n",
		sc2.NumSeries(), 3+1)
	fmt.Println()

	// ── SAFE SIMULATION ───────────────────────────────────────────────────────
	fmt.Println("--- Safe cardinality (100k users, capped at 1 value) ---")
	policy3 := NewLabelPolicy(map[string]int{"user_id": 1})
	safeSeries := simulateSafe(policy3, 100_000)
	fmt.Printf("  100 000 users → %d series (all user_ids → __overflow__)\n", safeSeries)
	fmt.Printf("  Memory: ~%.3f MB (vs %.1f MB uncapped)\n",
		memMB(safeSeries), memMB(100_000))
	fmt.Println()

	fmt.Println("--- Anti-patterns ---")
	fmt.Print(antiPatterns)
}
