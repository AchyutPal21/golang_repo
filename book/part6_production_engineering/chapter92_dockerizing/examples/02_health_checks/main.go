// FILE: book/part6_production_engineering/chapter92_dockerizing/examples/02_health_checks/main.go
// CHAPTER: 92 — Dockerizing Go Services
// TOPIC: HTTP health checks for Kubernetes probes — liveness, readiness,
//        startup delays, and graceful shutdown on SIGTERM.
//
// Run:
//   go run ./book/part6_production_engineering/chapter92_dockerizing/examples/02_health_checks

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SERVICE STATE
// ─────────────────────────────────────────────────────────────────────────────

// ready is set to 1 after the startup warmup completes.
var ready atomic.Int32

// shuttingDown is set to 1 when SIGTERM is received.
var shuttingDown atomic.Int32

// requestCount tracks total handled requests.
var requestCount atomic.Int64

const warmupDuration = 3 * time.Second

// ─────────────────────────────────────────────────────────────────────────────
// PROBE HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

// livenessHandler — Kubernetes restarts the container if this returns non-200.
// It should only fail if the process is truly broken (deadlock, OOM).
func livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"alive","goroutines":%d}`, runtime.NumGoroutine())
}

// readinessHandler — Kubernetes stops routing traffic if this returns non-200.
// Returns 503 during startup and shutdown.
func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if shuttingDown.Load() == 1 {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"shutting_down"}`)
		return
	}
	if ready.Load() == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"starting"}`)
		return
	}
	fmt.Fprintf(w, `{"status":"ready"}`)
}

// metricsHandler — basic process metrics (no Prometheus dependency).
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "goroutines %d\n", runtime.NumGoroutine())
	fmt.Fprintf(w, "heap_alloc_bytes %d\n", m.Alloc)
	fmt.Fprintf(w, "total_alloc_bytes %d\n", m.TotalAlloc)
	fmt.Fprintf(w, "gc_count %d\n", m.NumGC)
	fmt.Fprintf(w, "requests_total %d\n", requestCount.Load())
}

// appHandler — simulated application endpoint.
func appHandler(w http.ResponseWriter, r *http.Request) {
	requestCount.Add(1)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message":"ok","requests":%d}`, requestCount.Load())
}

// ─────────────────────────────────────────────────────────────────────────────
// PROBE SIMULATION (runs in-process without starting a real HTTP server)
// ─────────────────────────────────────────────────────────────────────────────

type fakeResponseWriter struct {
	status int
	body   string
	header http.Header
}

func newFakeRW() *fakeResponseWriter {
	return &fakeResponseWriter{status: 200, header: make(http.Header)}
}

func (f *fakeResponseWriter) Header() http.Header         { return f.header }
func (f *fakeResponseWriter) WriteHeader(status int)      { f.status = status }
func (f *fakeResponseWriter) Write(b []byte) (int, error) { f.body += string(b); return len(b), nil }

func simulateProbe(name string, handler http.HandlerFunc) {
	rw := newFakeRW()
	req, _ := http.NewRequest("GET", "/"+name, nil)
	handler(rw, req)
	fmt.Printf("  GET /%s → %d %s\n", name, rw.status, rw.body)
}

// ─────────────────────────────────────────────────────────────────────────────
// GRACEFUL SHUTDOWN DEMO
// ─────────────────────────────────────────────────────────────────────────────

func demonstrateGracefulShutdown() {
	fmt.Println()
	fmt.Println("--- Graceful shutdown sequence ---")
	fmt.Println(`  1. Kubernetes sends SIGTERM to container PID 1
  2. Container's signal handler sets shuttingDown=1
  3. /readyz starts returning 503 → kube-proxy drains connections (terminationGracePeriodSeconds)
  4. In-flight requests complete (server.Shutdown context with timeout)
  5. Process exits 0

  Go implementation:
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
    defer stop()
    server := &http.Server{Addr: ":8080", Handler: mux}
    go server.ListenAndServe()
    <-ctx.Done()                        // blocked until signal
    shuttingDown.Store(1)               // readyz returns 503
    shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    server.Shutdown(shutCtx)            // waits for in-flight requests`)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 92: Health Checks & Graceful Shutdown ===")
	fmt.Println()

	// ── STARTUP PROBE SEQUENCE ────────────────────────────────────────────────
	fmt.Println("--- Probe states during startup ---")
	fmt.Println("  [t=0s] Service starting, not yet ready:")
	ready.Store(0)
	shuttingDown.Store(0)
	simulateProbe("healthz", livenessHandler)
	simulateProbe("readyz", readinessHandler)

	fmt.Printf("\n  [t=%v] Simulating warmup delay...\n", warmupDuration)
	time.Sleep(100 * time.Millisecond) // shortened for demo
	ready.Store(1)
	fmt.Println("  [t=ready] Service warmed up:")
	simulateProbe("healthz", livenessHandler)
	simulateProbe("readyz", readinessHandler)

	// ── REQUEST HANDLING ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Request handling ---")
	for i := 0; i < 3; i++ {
		simulateProbe("api", appHandler)
	}

	// ── METRICS ───────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Metrics endpoint ---")
	simulateProbe("metrics", metricsHandler)

	// ── SHUTDOWN SEQUENCE ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Probe states during shutdown ---")
	shuttingDown.Store(1)
	simulateProbe("healthz", livenessHandler) // still 200 — process is alive
	simulateProbe("readyz", readinessHandler) // 503 — stop routing traffic

	demonstrateGracefulShutdown()

	// ── KUBERNETES CONFIG REFERENCE ───────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Kubernetes probe configuration ---")
	fmt.Println(`  livenessProbe:
    httpGet:
      path: /healthz
      port: 8080
    initialDelaySeconds: 5
    periodSeconds: 10
    failureThreshold: 3

  readinessProbe:
    httpGet:
      path: /readyz
      port: 8080
    initialDelaySeconds: 5
    periodSeconds: 5
    failureThreshold: 2

  startupProbe:           # Use for slow-starting apps (prevents liveness kill during init)
    httpGet:
      path: /readyz
      port: 8080
    failureThreshold: 30
    periodSeconds: 2      # 60s total startup window`)

	// Ensure signal imports are used (demonstration).
	_ = os.Interrupt
	_ = syscall.SIGTERM
	_ = signal.NotifyContext
	_ = context.Background
}
