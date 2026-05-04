// FILE: book/part6_production_engineering/chapter90_prometheus/examples/02_red_use/main.go
// CHAPTER: 90 — Prometheus Metrics
// TOPIC: RED (Rate/Error/Duration) for services, USE (Utilisation/Saturation/
//        Errors) for resources, label cardinality rules, and example PromQL.
//
// Run:
//   go run ./part6_production_engineering/chapter90_prometheus/examples/02_red_use/

package main

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// MINIMAL METRIC PRIMITIVES (self-contained, no external library)
// ─────────────────────────────────────────────────────────────────────────────

type counter struct{ v atomic.Int64 }

func (c *counter) Inc()         { c.v.Add(1) }
func (c *counter) Add(n int64)  { c.v.Add(n) }
func (c *counter) Get() int64   { return c.v.Load() }

type gauge struct {
	mu sync.Mutex
	v  float64
}

func (g *gauge) Set(v float64) { g.mu.Lock(); g.v = v; g.mu.Unlock() }
func (g *gauge) Inc()          { g.mu.Lock(); g.v++; g.mu.Unlock() }
func (g *gauge) Dec()          { g.mu.Lock(); g.v--; g.mu.Unlock() }
func (g *gauge) Get() float64  { g.mu.Lock(); defer g.mu.Unlock(); return g.v }

// ─────────────────────────────────────────────────────────────────────────────
// RED METRICS — Rate / Errors / Duration
// ─────────────────────────────────────────────────────────────────────────────

// ServiceMetrics holds RED metrics for a single service endpoint.
type ServiceMetrics struct {
	name           string
	requestsTotal  counter
	errorsTotal    counter
	activeRequests gauge

	mu          sync.Mutex
	durations   []float64 // rolling window for percentile calc
	windowLimit int
}

func NewServiceMetrics(name string) *ServiceMetrics {
	return &ServiceMetrics{name: name, windowLimit: 1000}
}

// RecordRequest simulates recording one request.
func (m *ServiceMetrics) RecordRequest(dur float64, isError bool) {
	m.requestsTotal.Inc()
	if isError {
		m.errorsTotal.Inc()
	}
	m.mu.Lock()
	if len(m.durations) >= m.windowLimit {
		m.durations = m.durations[1:]
	}
	m.durations = append(m.durations, dur)
	m.mu.Unlock()
}

func (m *ServiceMetrics) Rate(windowSecs float64) float64 {
	return float64(m.requestsTotal.Get()) / windowSecs
}

func (m *ServiceMetrics) ErrorRate() float64 {
	total := m.requestsTotal.Get()
	if total == 0 {
		return 0
	}
	return float64(m.errorsTotal.Get()) / float64(total)
}

func (m *ServiceMetrics) P99() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.durations) == 0 {
		return 0
	}
	sorted := make([]float64, len(m.durations))
	copy(sorted, m.durations)
	// simple insertion sort (small window)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	idx := int(0.99 * float64(len(sorted)-1))
	return sorted[idx]
}

// ─────────────────────────────────────────────────────────────────────────────
// USE METRICS — Utilisation / Saturation / Errors
// ─────────────────────────────────────────────────────────────────────────────

// ResourceMetrics holds USE metrics for a resource (e.g., DB connection pool).
type ResourceMetrics struct {
	name         string
	capacity     int
	inUse        gauge
	waitingQueue gauge
	errors       counter
}

func NewResourceMetrics(name string, capacity int) *ResourceMetrics {
	return &ResourceMetrics{name: name, capacity: capacity}
}

func (r *ResourceMetrics) Utilisation() float64 {
	return r.inUse.Get() / float64(r.capacity)
}

func (r *ResourceMetrics) IsSaturated() bool {
	return r.waitingQueue.Get() > 0
}

func (r *ResourceMetrics) Acquire() {
	r.inUse.Inc()
}

func (r *ResourceMetrics) Release() {
	r.inUse.Dec()
}

func (r *ResourceMetrics) RecordError() {
	r.errors.Inc()
}

// ─────────────────────────────────────────────────────────────────────────────
// LABEL CARDINALITY RULES
// ─────────────────────────────────────────────────────────────────────────────

const cardinalityGuide = `
Label cardinality rules:

  LOW cardinality (acceptable as labels):
    method         — GET, POST, PUT, DELETE, PATCH   (5 values)
    status_class   — 2xx, 3xx, 4xx, 5xx              (4 values)
    service        — api-gateway, order-service, ...  (~10 values)
    env            — production, staging              (2–3 values)
    region         — us-east-1, eu-west-1, ...        (~10 values)

  HIGH cardinality (NEVER use as labels — put in log fields instead):
    user_id        — millions of values → OOM in Prometheus
    request_id     — unique per request  → TSDB explosion
    email          — PII + high cardinality
    ip_address     — billions of IPs
    trace_id       — unique per trace

  Rule of thumb:
    If a label can have > 100 distinct values, it is too high cardinality.
    Prefer to use a histogram for the distribution, not separate series.

  Anti-pattern (status_code as label with 404/200/500/503/...):
    http_requests_total{status_code="404"}   ← OK (bounded)
    http_requests_total{user_id="12345"}     ← NEVER (unbounded)
`

// ─────────────────────────────────────────────────────────────────────────────
// PROMQL REFERENCE
// ─────────────────────────────────────────────────────────────────────────────

const promQLRef = `
PromQL reference for RED:

  Rate (requests/second over 5m):
    rate(http_requests_total[5m])

  Error ratio:
    rate(http_requests_total{status=~"5.."}[5m])
    / rate(http_requests_total[5m])

  p95 latency from histogram:
    histogram_quantile(0.95,
      sum(rate(http_request_duration_seconds_bucket[5m])) by (le, service)
    )

  Apdex score (target=100ms):
    (
      rate(http_request_duration_seconds_bucket{le="0.1"}[5m])
      + rate(http_request_duration_seconds_bucket{le="0.4"}[5m])
    ) / 2 / rate(http_request_duration_seconds_count[5m])

PromQL for USE:

  CPU utilisation:
    1 - avg(rate(node_cpu_seconds_total{mode="idle"}[5m])) by (instance)

  Memory saturation (swap use → saturation signal):
    node_memory_SwapFree_bytes / node_memory_SwapTotal_bytes < 0.2

  Connection pool utilisation:
    db_connections_in_use / db_connections_capacity
`

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 90: RED/USE Methods ===")
	fmt.Println()

	// ── RED — SERVICE METRICS ─────────────────────────────────────────────────
	fmt.Println("--- RED metrics simulation ---")
	svc := NewServiceMetrics("order-service")
	start := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			dur := 0.01 + rand.Float64()*0.2
			isErr := rand.Float64() < 0.05 // 5% error rate
			svc.RecordRequest(dur, isErr)
		}(i)
	}
	wg.Wait()

	elapsed := time.Since(start).Seconds()
	fmt.Printf("  Service: %s\n", svc.name)
	fmt.Printf("  R (Rate):     %.1f req/s\n", svc.Rate(elapsed))
	fmt.Printf("  E (ErrorRate): %.2f%% (%d/%d)\n",
		svc.ErrorRate()*100, svc.errorsTotal.Get(), svc.requestsTotal.Get())
	fmt.Printf("  D (p99 lat):  %.3fs\n", svc.P99())
	fmt.Println()

	// ── USE — RESOURCE METRICS ────────────────────────────────────────────────
	fmt.Println("--- USE metrics simulation (DB connection pool) ---")
	pool := NewResourceMetrics("db-pool", 20)

	// Simulate connections being used
	var wg2 sync.WaitGroup
	for i := 0; i < 18; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			pool.Acquire()
			time.Sleep(time.Millisecond)
			pool.Release()
		}()
	}
	// Check while in use
	time.Sleep(500 * time.Microsecond)
	fmt.Printf("  Resource: %s (capacity=%d)\n", pool.name, pool.capacity)
	fmt.Printf("  U (Utilisation):  %.0f%% (%d/%d in use)\n",
		pool.Utilisation()*100, int(pool.inUse.Get()), pool.capacity)
	fmt.Printf("  S (Saturation):   waiting=%d\n", int(pool.waitingQueue.Get()))
	fmt.Printf("  E (Errors):       %d\n", pool.errors.Get())
	wg2.Wait()
	fmt.Println()

	// ── MULTI-SERVICE DASHBOARD ───────────────────────────────────────────────
	fmt.Println("--- Multi-service RED dashboard ---")
	services := []struct {
		name     string
		rps      float64
		errPct   float64
		p99      float64
	}{
		{"api-gateway", 1250, 0.12, 0.042},
		{"order-service", 87, 0.45, 0.210},
		{"payment-service", 23, 2.10, 0.890},
		{"notification-service", 340, 0.01, 0.015},
	}
	fmt.Printf("  %-25s  %8s  %7s  %7s\n", "Service", "Rate/s", "Err%", "p99(s)")
	fmt.Printf("  %s\n", "-------------------------------------------------------")
	for _, s := range services {
		alert := ""
		if s.errPct > 1.0 {
			alert = " ⚑ HIGH ERROR"
		}
		if s.p99 > 0.5 {
			alert += " SLOW"
		}
		fmt.Printf("  %-25s  %8.1f  %6.2f%%  %7.3f%s\n",
			s.name, s.rps, s.errPct, s.p99, alert)
	}
	fmt.Println()

	// ── CARDINALITY ───────────────────────────────────────────────────────────
	fmt.Println("--- Label cardinality ---")
	fmt.Print(cardinalityGuide)

	// ── PROMQL ────────────────────────────────────────────────────────────────
	fmt.Println("--- PromQL reference ---")
	fmt.Print(promQLRef)
}
