// FILE: book/part6_production_engineering/chapter89_logging_strategy/exercises/01_log_pipeline/main.go
// CHAPTER: 89 — Logging Strategy
// EXERCISE: Build a log pipeline: structured JSON → PII scrubbing →
//   rate-limited sampler → buffered async writer → stdout.
//   Simulates a high-traffic service where health-check logs are sampled
//   and PII is stripped before any record reaches the output.
//
// Run:
//   go run ./part6_production_engineering/chapter89_logging_strategy/exercises/01_log_pipeline/

package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// STAGE 1: PII SCRUBBER (same as example 2, inlined for self-contained file)
// ─────────────────────────────────────────────────────────────────────────────

var emailRe = regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`)

var piiSensitiveKeys = map[string]bool{
	"password": true, "token": true, "secret": true,
	"authorization": true, "credit_card": true,
}

func scrub(v string) string {
	return emailRe.ReplaceAllString(v, "[EMAIL]")
}

type scrubbingHandler struct{ inner slog.Handler }

func (h *scrubbingHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}
func (h *scrubbingHandler) Handle(ctx context.Context, r slog.Record) error {
	clean := slog.NewRecord(r.Time, r.Level, scrub(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		clean.AddAttrs(scrubAttr(a))
		return true
	})
	return h.inner.Handle(ctx, clean)
}
func (h *scrubbingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	sa := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		sa[i] = scrubAttr(a)
	}
	return &scrubbingHandler{h.inner.WithAttrs(sa)}
}
func (h *scrubbingHandler) WithGroup(g string) slog.Handler {
	return &scrubbingHandler{h.inner.WithGroup(g)}
}
func scrubAttr(a slog.Attr) slog.Attr {
	if piiSensitiveKeys[strings.ToLower(a.Key)] {
		return slog.String(a.Key, "[REDACTED]")
	}
	if a.Value.Kind() == slog.KindString {
		return slog.String(a.Key, scrub(a.Value.String()))
	}
	return a
}

// ─────────────────────────────────────────────────────────────────────────────
// STAGE 2: SAMPLING HANDLER — 1-in-N per message key
// ─────────────────────────────────────────────────────────────────────────────

type samplingHandler struct {
	inner    slog.Handler
	every    int64
	counters sync.Map // map[string]*atomic.Int64
	dropped  atomic.Int64
}

func newSamplingHandler(inner slog.Handler, every int) *samplingHandler {
	return &samplingHandler{inner: inner, every: int64(every)}
}

func (h *samplingHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}

func (h *samplingHandler) Handle(ctx context.Context, r slog.Record) error {
	// Always pass through WARN and ERROR.
	if r.Level >= slog.LevelWarn {
		return h.inner.Handle(ctx, r)
	}
	key := r.Message
	v, _ := h.counters.LoadOrStore(key, new(atomic.Int64))
	counter := v.(*atomic.Int64)
	n := counter.Add(1)
	if n%h.every == 0 {
		enriched := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
		r.Attrs(func(a slog.Attr) bool { enriched.AddAttrs(a); return true })
		enriched.AddAttrs(slog.Int64("sampled_1_in", h.every))
		return h.inner.Handle(ctx, enriched)
	}
	h.dropped.Add(1)
	return nil
}

func (h *samplingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &samplingHandler{every: h.every, inner: h.inner.WithAttrs(attrs)}
}
func (h *samplingHandler) WithGroup(g string) slog.Handler {
	return &samplingHandler{every: h.every, inner: h.inner.WithGroup(g)}
}

// ─────────────────────────────────────────────────────────────────────────────
// STAGE 3: ASYNC BUFFERED HANDLER
// ─────────────────────────────────────────────────────────────────────────────

type asyncHandler struct {
	inner    slog.Handler
	ch       chan slog.Record
	wg       sync.WaitGroup
	done     chan struct{}
	dropped  atomic.Int64
}

func newAsyncHandler(inner slog.Handler, bufSize int) *asyncHandler {
	h := &asyncHandler{
		inner: inner,
		ch:    make(chan slog.Record, bufSize),
		done:  make(chan struct{}),
	}
	h.wg.Add(1)
	go h.drain()
	return h
}

func (h *asyncHandler) drain() {
	defer h.wg.Done()
	for {
		select {
		case r := <-h.ch:
			_ = h.inner.Handle(context.Background(), r)
		case <-h.done:
			// Flush remaining
			for {
				select {
				case r := <-h.ch:
					_ = h.inner.Handle(context.Background(), r)
				default:
					return
				}
			}
		}
	}
}

func (h *asyncHandler) Shutdown() {
	close(h.done)
	h.wg.Wait()
}

func (h *asyncHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}

func (h *asyncHandler) Handle(_ context.Context, r slog.Record) error {
	select {
	case h.ch <- r:
	default:
		h.dropped.Add(1) // back-pressure: drop oldest if full
	}
	return nil
}

func (h *asyncHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &asyncHandler{inner: h.inner.WithAttrs(attrs), ch: h.ch, done: h.done}
}
func (h *asyncHandler) WithGroup(g string) slog.Handler {
	return &asyncHandler{inner: h.inner.WithGroup(g), ch: h.ch, done: h.done}
}

// ─────────────────────────────────────────────────────────────────────────────
// SINK — captures output for inspection
// ─────────────────────────────────────────────────────────────────────────────

type captureSink struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	lines int
}

func (c *captureSink) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buf.Write(p)
	c.lines++
	return len(p), nil
}

func (c *captureSink) Lines() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lines
}

func (c *captureSink) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 89 Exercise: Log Pipeline ===")
	fmt.Println()

	// Build pipeline: JSON handler → scrubber → sampler → async
	sink := &captureSink{}
	jsonH := slog.NewJSONHandler(sink, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue("2026-01-01T00:00:00Z") // deterministic
			}
			return a
		},
	})
	scrubH := &scrubbingHandler{inner: jsonH}
	sampleH := newSamplingHandler(scrubH, 5) // 1-in-5 sampling for INFO
	asyncH := newAsyncHandler(sampleH, 256)
	logger := slog.New(asyncH)

	// ── SIMULATE TRAFFIC ──────────────────────────────────────────────────────
	fmt.Println("--- Simulating 100 health-checks + 20 real requests ---")
	// Health checks (high volume, sampled)
	for i := 0; i < 100; i++ {
		logger.Info("health_check", "status", "ok", "seq", i)
	}

	// Business events (always logged because WARN/ERROR pass through)
	logger.Warn("high_latency", "endpoint", "/api/orders", "p99_ms", 850)
	logger.Error("db_error", "query", "SELECT * FROM orders", "err", "timeout")

	// Login events with PII (should be scrubbed)
	for i := 0; i < 20; i++ {
		logger.Info("user_login",
			"email", fmt.Sprintf("user%d@example.com", i),
			"password", "hunter2",
			"user_id", i,
		)
	}

	// Shutdown flushes async buffer
	asyncH.Shutdown()
	time.Sleep(50 * time.Millisecond) // let prints complete

	// ── RESULTS ───────────────────────────────────────────────────────────────
	fmt.Printf("  Input events: 100 health + 2 business + 20 login = 122\n")
	fmt.Printf("  Output lines: %d\n", sink.Lines())
	fmt.Printf("  Async dropped: %d\n", asyncH.dropped.Load())
	fmt.Printf("  Sampler dropped: %d\n", sampleH.dropped.Load())
	fmt.Println()

	// ── SPOT CHECK OUTPUT ─────────────────────────────────────────────────────
	fmt.Println("--- Spot-check output (first 4 lines) ---")
	lines := strings.Split(strings.TrimRight(sink.String(), "\n"), "\n")
	for i, line := range lines {
		if i >= 4 {
			break
		}
		// Trim long JSON to fit display
		if len(line) > 120 {
			line = line[:117] + "..."
		}
		fmt.Printf("  [%d] %s\n", i+1, line)
	}
	fmt.Println()

	// ── PIPELINE DIAGRAM ─────────────────────────────────────────────────────
	fmt.Println("--- Pipeline diagram ---")
	fmt.Println("  logger.Info() / Warn() / Error()")
	fmt.Println("    ↓")
	fmt.Println("  asyncHandler  (non-blocking, 256-slot buffer)")
	fmt.Println("    ↓")
	fmt.Println("  samplingHandler  (1-in-5 for INFO, always pass WARN/ERROR)")
	fmt.Println("    ↓")
	fmt.Println("  scrubbingHandler  (email/password/token redacted)")
	fmt.Println("    ↓")
	fmt.Println("  slog.JSONHandler  → stdout")
}
