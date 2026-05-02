// FILE: book/part4_concurrency_systems/chapter56_production_http_server/exercises/01_observability/main.go
// CHAPTER: 56 — Production HTTP Server
// EXERCISE: Add observability to the production server — Prometheus-style
//           metrics (counters, histograms) collected in memory and exposed
//           on /metrics, correlation IDs injected into context, and a
//           structured error handler that returns consistent JSON errors.
//
// Run (from the chapter folder):
//   go run ./exercises/01_observability

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// METRICS REGISTRY
// ─────────────────────────────────────────────────────────────────────────────

type Counter struct{ v atomic.Int64 }
func (c *Counter) Inc()          { c.v.Add(1) }
func (c *Counter) Add(n int64)   { c.v.Add(n) }
func (c *Counter) Value() int64  { return c.v.Load() }

type Histogram struct {
	mu      sync.Mutex
	buckets []float64 // upper bounds in ms
	counts  []int64
	sum     float64
	total   int64
}

func NewHistogram(buckets []float64) *Histogram {
	return &Histogram{buckets: buckets, counts: make([]int64, len(buckets)+1)}
}

func (h *Histogram) Observe(ms float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sum += ms
	h.total++
	for i, b := range h.buckets {
		if ms <= b {
			h.counts[i]++
			return
		}
	}
	h.counts[len(h.buckets)]++ // +Inf bucket
}

type Registry struct {
	mu         sync.RWMutex
	counters   map[string]*Counter
	histograms map[string]*Histogram
}

func NewRegistry() *Registry {
	return &Registry{
		counters:   make(map[string]*Counter),
		histograms: make(map[string]*Histogram),
	}
}

func (r *Registry) Counter(name string) *Counter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.counters[name]; ok {
		return c
	}
	c := &Counter{}
	r.counters[name] = c
	return c
}

func (r *Registry) Histogram(name string, buckets []float64) *Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.histograms[name]; ok {
		return h
	}
	h := NewHistogram(buckets)
	r.histograms[name] = h
	return h
}

func (r *Registry) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		r.mu.RLock()
		defer r.mu.RUnlock()

		// Sorted counter names.
		cnames := make([]string, 0, len(r.counters))
		for n := range r.counters {
			cnames = append(cnames, n)
		}
		sort.Strings(cnames)
		for _, n := range cnames {
			fmt.Fprintf(w, "# COUNTER %s\n%s %d\n\n", n, n, r.counters[n].Value())
		}

		hnames := make([]string, 0, len(r.histograms))
		for n := range r.histograms {
			hnames = append(hnames, n)
		}
		sort.Strings(hnames)
		for _, n := range hnames {
			h := r.histograms[n]
			h.mu.Lock()
			fmt.Fprintf(w, "# HISTOGRAM %s\n", n)
			cumulative := int64(0)
			for i, b := range h.buckets {
				cumulative += h.counts[i]
				fmt.Fprintf(w, "%s_bucket{le=\"%.0f\"} %d\n", n, b, cumulative)
			}
			cumulative += h.counts[len(h.buckets)]
			fmt.Fprintf(w, "%s_bucket{le=\"+Inf\"} %d\n", n, cumulative)
			fmt.Fprintf(w, "%s_sum %.2f\n", n, h.sum)
			fmt.Fprintf(w, "%s_count %d\n\n", n, h.total)
			h.mu.Unlock()
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CORRELATION ID MIDDLEWARE
// ─────────────────────────────────────────────────────────────────────────────

type correlationIDKey struct{}

func correlationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Correlation-ID")
		if id == "" {
			id = fmt.Sprintf("%016x", rand.Int63())
		}
		ctx := context.WithValue(r.Context(), correlationIDKey{}, id)
		w.Header().Set("X-Correlation-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func correlationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return id
	}
	return "unknown"
}

// ─────────────────────────────────────────────────────────────────────────────
// METRICS MIDDLEWARE
// ─────────────────────────────────────────────────────────────────────────────

type metricsRecorder struct {
	http.ResponseWriter
	status int
}

func (m *metricsRecorder) WriteHeader(code int) {
	m.status = code
	m.ResponseWriter.WriteHeader(code)
}

func metricsMiddleware(reg *Registry, next http.Handler) http.Handler {
	reqTotal := reg.Counter("http_requests_total")
	reqErrors := reg.Counter("http_requests_errors_total")
	latency := reg.Histogram("http_request_duration_ms",
		[]float64{1, 5, 10, 25, 50, 100, 250, 500})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &metricsRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)

		ms := float64(time.Since(start).Microseconds()) / 1000.0
		reqTotal.Inc()
		latency.Observe(ms)
		if rec.status >= 500 {
			reqErrors.Inc()
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// JSON ERROR HELPER
// ─────────────────────────────────────────────────────────────────────────────

type APIError struct {
	Code          int    `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id"`
}

func writeAPIError(w http.ResponseWriter, r *http.Request, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(APIError{
		Code:          code,
		Message:       message,
		CorrelationID: correlationID(r.Context()),
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func slowHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	delay := time.Duration(10+rand.Intn(90)) * time.Millisecond
	select {
	case <-time.After(delay):
		cid := correlationID(ctx)
		json.NewEncoder(w).Encode(map[string]string{
			"correlation_id": cid,
			"delay_ms":       fmt.Sprintf("%d", delay.Milliseconds()),
		})
	case <-ctx.Done():
		writeAPIError(w, r, http.StatusGatewayTimeout, "request timed out")
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	writeAPIError(w, r, http.StatusInternalServerError, "something went wrong")
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	reg := NewRegistry()

	mux := http.NewServeMux()
	mux.HandleFunc("/work", slowHandler)
	mux.HandleFunc("/error", errorHandler)
	mux.HandleFunc("/metrics", reg.Handler())

	handler := correlationMiddleware(metricsMiddleware(reg, mux))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	srv := &http.Server{Handler: handler, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second}
	go srv.Serve(ln)
	base := "http://" + ln.Addr().String()

	logger.Info("server started", "addr", base)

	// Send 20 requests to /work and 3 to /error.
	client := &http.Client{Timeout: 5 * time.Second}
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(base + "/work")
			if err == nil {
				resp.Body.Close()
			}
		}()
	}
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(base + "/error")
			if err == nil {
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()

	// Print metrics.
	fmt.Println()
	resp, _ := client.Get(base + "/metrics")
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	resp.Body.Close()
	fmt.Println("=== /metrics output ===")
	fmt.Print(string(buf[:n]))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
