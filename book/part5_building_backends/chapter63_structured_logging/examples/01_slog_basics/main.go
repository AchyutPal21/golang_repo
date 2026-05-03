// FILE: book/part5_building_backends/chapter63_structured_logging/examples/01_slog_basics/main.go
// CHAPTER: 63 — Structured Logging
// TOPIC: Go standard library log/slog fundamentals —
//        log levels, structured attributes, JSON vs text handlers,
//        child loggers with pre-set attributes, log groups, context logging,
//        and a custom handler that adds a service field to every record.
//
// Run (from the chapter folder):
//   go run ./examples/01_slog_basics

package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// ─────────────────────────────────────────────────────────────────────────────
// CUSTOM HANDLER — wraps any handler to add a "service" attribute to all logs
// ─────────────────────────────────────────────────────────────────────────────

type serviceHandler struct {
	inner   slog.Handler
	service string
}

func newServiceHandler(inner slog.Handler, service string) *serviceHandler {
	return &serviceHandler{inner: inner, service: service}
}

func (h *serviceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *serviceHandler) Handle(ctx context.Context, r slog.Record) error {
	// Add the service attribute to every record before passing to the inner handler.
	r.AddAttrs(slog.String("service", h.service))
	return h.inner.Handle(ctx, r)
}

func (h *serviceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &serviceHandler{inner: h.inner.WithAttrs(attrs), service: h.service}
}

func (h *serviceHandler) WithGroup(name string) slog.Handler {
	return &serviceHandler{inner: h.inner.WithGroup(name), service: h.service}
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT KEY for request ID
// ─────────────────────────────────────────────────────────────────────────────

type ctxKey int

const keyRequestID ctxKey = iota

func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyRequestID, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT HANDLER — reads request-id from context and adds it to every log
// ─────────────────────────────────────────────────────────────────────────────

type contextHandler struct {
	inner slog.Handler
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value(keyRequestID).(string); ok && id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.inner.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{inner: h.inner.WithGroup(name)}
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func section(title string) {
	fmt.Printf("\n--- %s ---\n", title)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== log/slog Structured Logging ===")

	// ── TEXT HANDLER ─────────────────────────────────────────────────────────
	section("Text handler (human-readable)")
	textLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	textLogger.Debug("debug message", "key", "value")
	textLogger.Info("server started", "port", 8080)
	textLogger.Warn("disk usage high", "percent", 85)
	textLogger.Error("connection failed", "host", "db.example.com", "err", "timeout")

	// ── JSON HANDLER ─────────────────────────────────────────────────────────
	section("JSON handler (machine-readable)")
	jsonLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo, // Debug messages filtered out
	}))
	jsonLogger.Debug("this is filtered out") // below INFO level
	jsonLogger.Info("user logged in",
		slog.String("user_id", "u-123"),
		slog.String("ip", "192.168.1.1"),
		slog.Int("duration_ms", 42),
	)

	// ── LEVEL FILTERING ───────────────────────────────────────────────────────
	section("Level filtering")
	warnLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))
	warnLogger.Info("this is suppressed (below WARN)")
	warnLogger.Warn("this appears", "component", "cache")
	warnLogger.Error("this appears too", "component", "db")

	// ── STRUCTURED ATTRIBUTES ────────────────────────────────────────────────
	section("Structured attribute types")
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("request completed",
		slog.String("method", "GET"),
		slog.String("path", "/api/users"),
		slog.Int("status", 200),
		slog.Int64("bytes", 4096),
		slog.Float64("latency_ms", 12.5),
		slog.Bool("cache_hit", true),
		slog.Any("user_ids", []int{1, 2, 3}),
	)

	// ── CHILD LOGGER WITH PRE-SET ATTRS ──────────────────────────────────────
	section("Child logger (With) — pre-set attributes")
	requestLogger := logger.With(
		slog.String("request_id", "req-abc-123"),
		slog.String("user_id", "u-456"),
	)
	// All log lines from requestLogger share these attributes.
	requestLogger.Info("handler started")
	requestLogger.Info("db query",
		slog.String("table", "users"),
		slog.Int("rows", 3),
	)
	requestLogger.Info("handler finished", slog.Int("status", 200))

	// ── LOG GROUPS ───────────────────────────────────────────────────────────
	section("Log groups — nested attribute namespacing")
	logger.Info("http request",
		slog.Group("request",
			slog.String("method", "POST"),
			slog.String("path", "/api/orders"),
			slog.String("user_agent", "Go-http-client/1.1"),
		),
		slog.Group("response",
			slog.Int("status", 201),
			slog.Int("bytes", 512),
		),
	)

	// ── CONTEXT LOGGING ──────────────────────────────────────────────────────
	section("Context logging — request ID from context")
	ctxHandler := &contextHandler{inner: slog.NewJSONHandler(os.Stdout, nil)}
	ctxLogger := slog.New(ctxHandler)

	ctx := withRequestID(context.Background(), "req-xyz-789")
	ctxLogger.InfoContext(ctx, "processing order", slog.String("order_id", "ord-1"))
	ctxLogger.InfoContext(ctx, "payment processed", slog.String("amount", "99.99"))

	// Without request ID in context — field simply absent.
	ctxLogger.InfoContext(context.Background(), "background task ran")

	// ── CUSTOM HANDLER ───────────────────────────────────────────────────────
	section("Custom handler — service field on every log record")
	baseHandler := slog.NewJSONHandler(os.Stdout, nil)
	svcLogger := slog.New(newServiceHandler(baseHandler, "order-service"))
	svcLogger.Info("order created", slog.String("order_id", "ord-42"))
	svcLogger.Error("payment failed", slog.String("order_id", "ord-43"), slog.String("reason", "card declined"))

	// ── DISCARD HANDLER ──────────────────────────────────────────────────────
	section("Discard handler (for tests)")
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	discardLogger.Info("this goes nowhere — useful in tests")
	fmt.Println("  (discard handler swallows all output)")
}
