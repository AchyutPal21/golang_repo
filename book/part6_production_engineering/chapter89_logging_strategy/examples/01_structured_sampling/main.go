// FILE: book/part6_production_engineering/chapter89_logging_strategy/examples/01_structured_sampling/main.go
// CHAPTER: 89 — Logging Strategy
// TOPIC: Structured logging with log/slog, log sampling (rate limiting),
//        context propagation of request IDs, and log level strategy.
//
// Run:
//   go run ./part6_production_engineering/chapter89_logging_strategy/examples/01_structured_sampling/

package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT KEY — request-scoped logger
// ─────────────────────────────────────────────────────────────────────────────

type ctxKey string

const loggerKey ctxKey = "logger"
const requestIDKey ctxKey = "request_id"

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func LoggerFromCtx(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// ─────────────────────────────────────────────────────────────────────────────
// LOG SAMPLER — drops high-volume events probabilistically
// ─────────────────────────────────────────────────────────────────────────────

// Sampler rate-limits log emission.
// Every N events, exactly 1 is logged (head-based sampling).
type Sampler struct {
	every    int64       // log 1 in every N events
	counter  atomic.Int64
	dropped  atomic.Int64
	delegate *slog.Logger
}

func NewSampler(delegate *slog.Logger, every int) *Sampler {
	if every < 1 {
		every = 1
	}
	return &Sampler{every: int64(every), delegate: delegate}
}

func (s *Sampler) Info(msg string, args ...any) {
	c := s.counter.Add(1)
	if c%s.every == 0 {
		// Append a "sampled" marker so consumers know this is sampled.
		args = append(args, "sampled_rate", s.every)
		s.delegate.Info(msg, args...)
	} else {
		s.dropped.Add(1)
	}
}

// DroppedCount returns the number of events dropped since creation.
func (s *Sampler) DroppedCount() int64 { return s.dropped.Load() }

// ─────────────────────────────────────────────────────────────────────────────
// RATE-BASED SAMPLER — emit at most N events per second per key
// ─────────────────────────────────────────────────────────────────────────────

type rateSampler struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	maxRate int // events per second per key
}

type rateBucket struct {
	tokens   float64
	lastTick time.Time
}

func newRateSampler(maxRate int) *rateSampler {
	return &rateSampler{buckets: make(map[string]*rateBucket), maxRate: maxRate}
}

// Allow returns true if an event with the given key should be logged.
func (rs *rateSampler) Allow(key string) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	now := time.Now()
	b, ok := rs.buckets[key]
	if !ok {
		b = &rateBucket{tokens: float64(rs.maxRate), lastTick: now}
		rs.buckets[key] = b
	}
	elapsed := now.Sub(b.lastTick).Seconds()
	b.lastTick = now
	b.tokens += elapsed * float64(rs.maxRate)
	if b.tokens > float64(rs.maxRate) {
		b.tokens = float64(rs.maxRate)
	}
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// LOG LEVEL STRATEGY
// ─────────────────────────────────────────────────────────────────────────────

const levelStrategy = `
Log Level Strategy:

  DEBUG  — developer-only, never in production
           Expensive: may contain raw payloads, SQL queries, loop traces.
           Enable per-request via dynamic level change.

  INFO   — normal business events (request served, job started, user login)
           Sample high-volume endpoints (e.g., health checks) at 1-in-100.

  WARN   — recoverable anomaly (retry succeeded, quota nearing, deprecated API)
           Always emit; alert on sustained rate > threshold.

  ERROR  — operation failed, requires investigation
           Always emit; page on spike.

  FATAL  — cannot continue (config missing, DB unreachable at startup)
           log.Fatal / os.Exit(1). Never use inside request handlers.

Dynamic level change (without restart):
  var level slog.LevelVar            // atomic, safe for concurrent use
  level.Set(slog.LevelDebug)         // enable debug at runtime
  handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: &level})
`

// ─────────────────────────────────────────────────────────────────────────────
// STRUCTURED LOG FIELDS
// ─────────────────────────────────────────────────────────────────────────────

// requestLog emits a structured request completion log.
func requestLog(ctx context.Context, method, path string, status int, dur time.Duration, err error) {
	logger := LoggerFromCtx(ctx)
	reqID, _ := ctx.Value(requestIDKey).(string)

	fields := []any{
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", dur.Milliseconds(),
		"request_id", reqID,
	}
	if err != nil {
		fields = append(fields, "error", err.Error())
		logger.Error("request completed with error", fields...)
		return
	}
	if status >= 500 {
		logger.Error("server error", fields...)
	} else if status >= 400 {
		logger.Warn("client error", fields...)
	} else {
		logger.Info("request completed", fields...)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 89: Structured Logging & Sampling ===")
	fmt.Println()

	// ── BASIC STRUCTURED LOGGER ───────────────────────────────────────────────
	fmt.Println("--- JSON structured logger ---")
	var levelVar slog.LevelVar
	levelVar.Set(slog.LevelInfo)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: &levelVar,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			// Normalise time to RFC3339 milliseconds.
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339Nano))
			}
			return a
		},
	})
	base := slog.New(handler)
	base.Info("server started", "port", 8080, "version", "1.2.3")
	base.Warn("high memory usage", "heap_mb", 412, "threshold_mb", 400)
	fmt.Println()

	// ── CONTEXT PROPAGATION ───────────────────────────────────────────────────
	fmt.Println("--- Context-scoped logger ---")
	logger := base.With("service", "api-gateway", "env", "production")
	ctx := WithLogger(context.Background(), logger)
	ctx = WithRequestID(ctx, "req-abc-123")

	requestLog(ctx, "GET", "/api/users/42", 200, 12*time.Millisecond, nil)
	requestLog(ctx, "DELETE", "/api/items/99", 404, 3*time.Millisecond, nil)
	fmt.Println()

	// ── HEAD-BASED SAMPLER ────────────────────────────────────────────────────
	fmt.Println("--- Head-based sampler (1-in-10) ---")
	sampled := NewSampler(base, 10)
	total := 100
	for i := 0; i < total; i++ {
		sampled.Info("health_check", "status", "ok", "iter", i)
	}
	fmt.Printf("  %d events → dropped=%d emitted=%d\n",
		total, sampled.DroppedCount(), int64(total)-sampled.DroppedCount())
	fmt.Println()

	// ── RATE-BASED SAMPLER ────────────────────────────────────────────────────
	fmt.Println("--- Rate-based sampler (2/s per endpoint) ---")
	rs := newRateSampler(2)
	endpoints := []string{"/api/users", "/health", "/metrics"}
	var allowed, denied int
	for i := 0; i < 30; i++ {
		ep := endpoints[i%len(endpoints)]
		if rs.Allow(ep) {
			allowed++
		} else {
			denied++
		}
		// No real sleep — just simulate time passing occasionally
		if i%10 == 9 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	fmt.Printf("  30 rapid events → allowed=%d denied=%d\n", allowed, denied)
	fmt.Println()

	// ── DYNAMIC LEVEL CHANGE ──────────────────────────────────────────────────
	fmt.Println("--- Dynamic level change ---")
	fmt.Printf("  Current level: %s\n", levelVar.Level())
	base.Debug("this is hidden at INFO level", "detail", "x")
	fmt.Println("  (debug not shown above — level is INFO)")
	levelVar.Set(slog.LevelDebug)
	fmt.Printf("  Changed level to: %s\n", levelVar.Level())
	base.Debug("now debug is visible", "detail", "x")
	levelVar.Set(slog.LevelInfo)
	fmt.Println()

	// ── LEVEL STRATEGY ────────────────────────────────────────────────────────
	fmt.Println("--- Level strategy reference ---")
	fmt.Print(levelStrategy)
	fmt.Println()

	// ── CARDINALITY WARNING ───────────────────────────────────────────────────
	fmt.Println("--- Cardinality in log fields ---")
	fmt.Println("  HIGH cardinality fields (per-request unique values):")
	fmt.Println("    request_id, user_id, trace_id — fine in logs, avoid in metrics")
	fmt.Println("  LOW cardinality fields (bounded set):")
	fmt.Println("    method, status_class (2xx/4xx/5xx), service, env")
	fmt.Println("  Rule: log high-cardinality; only metric low-cardinality labels")
	fmt.Println()

	_ = rand.Int() // suppress import
}
