// FILE: book/part6_production_engineering/chapter90_prometheus/examples/01_metrics_types/main.go
// CHAPTER: 90 — Prometheus Metrics
// TOPIC: Counter, Gauge, Histogram, Summary — pure in-process simulation
//        that shows correct usage, registration, and text-format rendering.
//        No external Prometheus server required.
//
// Run:
//   go run ./part6_production_engineering/chapter90_prometheus/examples/01_metrics_types/

package main

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// IN-PROCESS COUNTER
// ─────────────────────────────────────────────────────────────────────────────

// Counter is a monotonically increasing value (never decrements).
type Counter struct {
	name   string
	labels map[string]string
	value  atomic.Int64
}

func NewCounter(name string, labels map[string]string) *Counter {
	return &Counter{name: name, labels: labels}
}

func (c *Counter) Inc()          { c.value.Add(1) }
func (c *Counter) Add(n int64)   { c.value.Add(n) }
func (c *Counter) Value() int64  { return c.value.Load() }

// ─────────────────────────────────────────────────────────────────────────────
// IN-PROCESS GAUGE
// ─────────────────────────────────────────────────────────────────────────────

// Gauge can go up and down (active connections, queue depth, memory).
type Gauge struct {
	name   string
	labels map[string]string
	mu     sync.Mutex
	value  float64
}

func NewGauge(name string, labels map[string]string) *Gauge {
	return &Gauge{name: name, labels: labels}
}

func (g *Gauge) Set(v float64)  { g.mu.Lock(); g.value = v; g.mu.Unlock() }
func (g *Gauge) Inc()            { g.mu.Lock(); g.value++; g.mu.Unlock() }
func (g *Gauge) Dec()            { g.mu.Lock(); g.value--; g.mu.Unlock() }
func (g *Gauge) Add(v float64)  { g.mu.Lock(); g.value += v; g.mu.Unlock() }
func (g *Gauge) Value() float64 { g.mu.Lock(); defer g.mu.Unlock(); return g.value }

// ─────────────────────────────────────────────────────────────────────────────
// IN-PROCESS HISTOGRAM
// ─────────────────────────────────────────────────────────────────────────────

// Histogram observes values and buckets them.
// The standard HTTP latency buckets (in seconds).
var DefaultBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

type Histogram struct {
	name    string
	labels  map[string]string
	mu      sync.Mutex
	buckets []float64 // upper bounds
	counts  []int64   // count[i] = observations <= buckets[i]
	sum     float64
	count   int64
}

func NewHistogram(name string, labels map[string]string, buckets []float64) *Histogram {
	sorted := make([]float64, len(buckets))
	copy(sorted, buckets)
	sort.Float64s(sorted)
	return &Histogram{
		name:    name,
		labels:  labels,
		buckets: sorted,
		counts:  make([]int64, len(sorted)),
	}
}

func (h *Histogram) Observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, b := range h.buckets {
		if v <= b {
			h.counts[i]++
		}
	}
	h.sum += v
	h.count++
}

// Percentile returns an approximate pN from the bucket distribution.
func (h *Histogram) Percentile(p float64) float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.count == 0 {
		return 0
	}
	target := int64(float64(h.count) * p)
	for i, b := range h.buckets {
		if h.counts[i] >= target {
			return b
		}
	}
	return h.buckets[len(h.buckets)-1]
}

func (h *Histogram) Mean() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.count == 0 {
		return 0
	}
	return h.sum / float64(h.count)
}

// ─────────────────────────────────────────────────────────────────────────────
// IN-PROCESS SUMMARY (simple sliding-window quantiles)
// ─────────────────────────────────────────────────────────────────────────────

// Summary keeps a bounded window of observations and computes quantiles.
type Summary struct {
	name     string
	labels   map[string]string
	mu       sync.Mutex
	window   []float64
	maxSize  int
	sum      float64
	count    int64
}

func NewSummary(name string, labels map[string]string, windowSize int) *Summary {
	return &Summary{name: name, labels: labels, window: make([]float64, 0, windowSize), maxSize: windowSize}
}

func (s *Summary) Observe(v float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.window) >= s.maxSize {
		// Drop oldest (ring-like truncation for simplicity).
		s.window = s.window[1:]
	}
	s.window = append(s.window, v)
	s.sum += v
	s.count++
}

func (s *Summary) Quantile(q float64) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.window) == 0 {
		return 0
	}
	sorted := make([]float64, len(s.window))
	copy(sorted, s.window)
	sort.Float64s(sorted)
	idx := int(float64(len(sorted)-1) * q)
	return sorted[idx]
}

// ─────────────────────────────────────────────────────────────────────────────
// PROMETHEUS TEXT FORMAT RENDERER
// ─────────────────────────────────────────────────────────────────────────────

func renderCounter(c *Counter) string {
	return fmt.Sprintf("# TYPE %s counter\n%s %d\n", c.name, c.name, c.Value())
}

func renderGauge(g *Gauge) string {
	return fmt.Sprintf("# TYPE %s gauge\n%s %g\n", g.name, g.name, g.Value())
}

func renderHistogram(h *Histogram) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := fmt.Sprintf("# TYPE %s histogram\n", h.name)
	for i, b := range h.buckets {
		out += fmt.Sprintf("%s_bucket{le=\"%g\"} %d\n", h.name, b, h.counts[i])
	}
	out += fmt.Sprintf("%s_bucket{le=\"+Inf\"} %d\n", h.name, h.count)
	out += fmt.Sprintf("%s_sum %g\n", h.name, h.sum)
	out += fmt.Sprintf("%s_count %d\n", h.name, h.count)
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 90: Prometheus Metric Types ===")
	fmt.Println()

	// ── COUNTER ───────────────────────────────────────────────────────────────
	fmt.Println("--- Counter ---")
	httpRequests := NewCounter("http_requests_total", map[string]string{"method": "GET", "path": "/api/users", "status": "200"})
	errors := NewCounter("http_errors_total", map[string]string{"type": "5xx"})

	for i := 0; i < 142; i++ {
		httpRequests.Inc()
		if i%20 == 0 {
			errors.Inc()
		}
	}
	fmt.Println(renderCounter(httpRequests))
	fmt.Println(renderCounter(errors))
	fmt.Println("  Rules: never decrement, reset only on process restart.")
	fmt.Println("  PromQL: rate(http_requests_total[5m]) — per-second rate")
	fmt.Println()

	// ── GAUGE ─────────────────────────────────────────────────────────────────
	fmt.Println("--- Gauge ---")
	activeConns := NewGauge("active_connections", nil)
	queueDepth := NewGauge("queue_depth", map[string]string{"queue": "orders"})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			activeConns.Inc()
			time.Sleep(time.Millisecond)
			activeConns.Dec()
		}()
	}
	wg.Wait()
	queueDepth.Set(37)
	fmt.Println(renderGauge(activeConns))
	fmt.Println(renderGauge(queueDepth))
	fmt.Println("  Use Gauge for: queue depth, goroutines, heap MB, CPU%.")
	fmt.Println()

	// ── HISTOGRAM ─────────────────────────────────────────────────────────────
	fmt.Println("--- Histogram ---")
	latency := NewHistogram("http_request_duration_seconds", map[string]string{"handler": "/api"}, DefaultBuckets)

	// Simulate a bimodal latency distribution.
	for i := 0; i < 1000; i++ {
		var dur float64
		if rand.Float64() < 0.9 {
			dur = rand.Float64() * 0.1 // 90% fast: 0–100ms
		} else {
			dur = 0.1 + rand.Float64()*0.9 // 10% slow: 100ms–1s
		}
		latency.Observe(dur)
	}
	fmt.Println(renderHistogram(latency))
	fmt.Printf("  Approx p50=%.3fs  p95=%.3fs  p99=%.3fs  mean=%.3fs\n",
		latency.Percentile(0.50), latency.Percentile(0.95),
		latency.Percentile(0.99), latency.Mean())
	fmt.Println("  PromQL: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))")
	fmt.Println()

	// ── SUMMARY ───────────────────────────────────────────────────────────────
	fmt.Println("--- Summary ---")
	dbDur := NewSummary("db_query_duration_seconds", map[string]string{"query": "select"}, 500)
	for i := 0; i < 500; i++ {
		dbDur.Observe(0.001 + rand.Float64()*0.049)
	}
	fmt.Printf("  p50=%.4fs  p90=%.4fs  p99=%.4fs\n",
		dbDur.Quantile(0.50), dbDur.Quantile(0.90), dbDur.Quantile(0.99))
	fmt.Println()
	fmt.Println("  Histogram vs Summary:")
	fmt.Println("    Histogram: aggregatable across instances (use in production)")
	fmt.Println("    Summary:   exact quantiles, but cannot aggregate (use per-instance)")
	fmt.Println()

	// ── PROMETHEUS TEXT FORMAT ────────────────────────────────────────────────
	fmt.Println("--- Prometheus text exposition format (excerpt) ---")
	format := `# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",status="200"} 142

# HELP http_request_duration_seconds HTTP request latency
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.005"} 12
http_request_duration_seconds_bucket{le="0.01"}  34
http_request_duration_seconds_bucket{le="+Inf"}  142
http_request_duration_seconds_sum   8.7
http_request_duration_seconds_count 142`
	fmt.Println(format)
}
