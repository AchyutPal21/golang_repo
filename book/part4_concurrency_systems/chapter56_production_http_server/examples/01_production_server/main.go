// FILE: book/part4_concurrency_systems/chapter56_production_http_server/examples/01_production_server/main.go
// CHAPTER: 56 — Production HTTP Server
// TOPIC: Everything you need for a production Go HTTP server — graceful
//        shutdown with OS signal handling, structured request logging,
//        panic recovery middleware, health/readiness probes, request timeout
//        middleware, rate limiting, and pprof debug endpoints.
//
// Run (from the chapter folder):
//   go run ./examples/01_production_server

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	_ "net/http/pprof" // registers /debug/pprof handlers on DefaultServeMux
	"os"
	"os/signal"
	"runtime/debug"
	"sync/atomic"
	"syscall"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE: panic recovery
// ─────────────────────────────────────────────────────────────────────────────

func recoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					"panic", rec,
					"stack", string(debug.Stack()),
					"method", r.Method,
					"path", r.URL.Path,
				)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE: structured request logging
// ─────────────────────────────────────────────────────────────────────────────

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE: per-request timeout
// ─────────────────────────────────────────────────────────────────────────────

func timeoutMiddleware(d time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE: simple token-bucket rate limiter
// ─────────────────────────────────────────────────────────────────────────────

type rateLimiter struct {
	tokens   atomic.Int64
	cap      int64
	interval time.Duration
}

func newRateLimiter(cap int64, refillInterval time.Duration) *rateLimiter {
	rl := &rateLimiter{cap: cap, interval: refillInterval}
	rl.tokens.Store(cap)
	go func() {
		ticker := time.NewTicker(refillInterval)
		for range ticker.C {
			rl.tokens.Store(cap)
		}
	}()
	return rl
}

func (rl *rateLimiter) allow() bool {
	for {
		cur := rl.tokens.Load()
		if cur <= 0 {
			return false
		}
		if rl.tokens.CompareAndSwap(cur, cur-1) {
			return true
		}
	}
}

func rateLimitMiddleware(rl *rateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow() {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// HEALTH / READINESS
// ─────────────────────────────────────────────────────────────────────────────

type healthState struct {
	ready atomic.Bool
}

func healthHandler(hs *healthState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	}
}

func readinessHandler(hs *healthState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !hs.ready.Load() {
			http.Error(w, `{"status":"not ready"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func echoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	select {
	case <-ctx.Done():
		http.Error(w, "request timeout", http.StatusGatewayTimeout)
		return
	case <-time.After(5 * time.Millisecond):
		fmt.Fprintln(w, "echo: "+r.URL.Query().Get("msg"))
	}
}

func panicHandler(w http.ResponseWriter, r *http.Request) {
	panic("intentional panic for demo")
}

// ─────────────────────────────────────────────────────────────────────────────
// SERVER SETUP
// ─────────────────────────────────────────────────────────────────────────────

func buildServer(logger *slog.Logger, addr string) (*http.Server, *healthState) {
	hs := &healthState{}
	rl := newRateLimiter(100, time.Second)

	mux := http.NewServeMux()

	// Application routes.
	mux.HandleFunc("/echo", echoHandler)
	mux.HandleFunc("/panic", panicHandler) // intentional panic demo

	// Infra routes (bypass rate limiter + timeout).
	mux.HandleFunc("/health", healthHandler(hs))
	mux.HandleFunc("/ready", readinessHandler(hs))

	// Build middleware stack (outermost first).
	handler := recoveryMiddleware(logger,
		loggingMiddleware(logger,
			rateLimitMiddleware(rl,
				timeoutMiddleware(500*time.Millisecond, mux))))

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
		ErrorLog:     log.New(os.Stderr, "[server] ", log.LstdFlags),
	}
	return srv, hs
}

// ─────────────────────────────────────────────────────────────────────────────
// SELF-TEST + GRACEFUL SHUTDOWN
// ─────────────────────────────────────────────────────────────────────────────

func runTests(base string, logger *slog.Logger) {
	client := &http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		path string
		want int
	}{
		{"/health", 200},
		{"/ready", 503}, // not ready yet
		{"/echo?msg=hello", 200},
		{"/panic", 500},
	}

	logger.Info("running self-tests", "base", base)
	for _, tt := range tests {
		resp, err := client.Get(base + tt.path)
		if err != nil {
			logger.Warn("test failed", "path", tt.path, "error", err)
			continue
		}
		resp.Body.Close()
		status := "✓"
		if resp.StatusCode != tt.want {
			status = "✗"
		}
		logger.Info("test",
			"status", status,
			"path", tt.path,
			"got", resp.StatusCode,
			"want", tt.want,
		)
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		logger.Error("listen failed", "error", err)
		os.Exit(1)
	}

	srv, hs := buildServer(logger, ln.Addr().String())

	// Start serving.
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error("serve failed", "error", err)
		}
	}()
	logger.Info("server started", "addr", ln.Addr().String())

	// Small pause so the server goroutine starts listening.
	time.Sleep(5 * time.Millisecond)

	// Run tests BEFORE marking ready — /ready should return 503.
	runTests("http://"+ln.Addr().String(), logger)

	// Now mark ready.
	hs.ready.Store(true)
	logger.Info("server ready")

	// Wait for OS signal (SIGTERM/SIGINT) or short timeout for demo.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	// For demo: auto-shutdown after 200ms.
	go func() {
		time.Sleep(200 * time.Millisecond)
		quit <- syscall.SIGTERM
	}()

	sig := <-quit
	logger.Info("shutdown signal received", "signal", sig)

	// Graceful shutdown: drain in-flight for up to 30 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown failed", "error", err)
	}
	logger.Info("server stopped")
}
