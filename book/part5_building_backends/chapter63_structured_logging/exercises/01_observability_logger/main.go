// FILE: book/part5_building_backends/chapter63_structured_logging/exercises/01_observability_logger/main.go
// CHAPTER: 63 — Structured Logging
// EXERCISE: Build an observability logger layer for an HTTP API:
//   - Multi-handler: write DEBUG+ to stderr (text), INFO+ to stdout (JSON)
//   - Request middleware: inject child logger with trace_id, method, path, user_agent
//   - Domain events: order lifecycle (created, updated, shipped, cancelled) with structured fields
//   - Slow request detector: warn if handler duration > threshold
//   - Error sampler: log 100% of 5xx, 10% of 4xx (simulated)
//   - Discard logger for tests
//
// Run (from the chapter folder):
//   go run ./exercises/01_observability_logger

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// MULTI-WRITER HANDLER
// Fans out to multiple handlers — each can have its own level filter and format.
// ─────────────────────────────────────────────────────────────────────────────

type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) *multiHandler {
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var first error
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r.Clone()); err != nil && first == nil {
				first = err
			}
		}
	}
	return first
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: hs}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: hs}
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT KEY & ACCESSORS
// ─────────────────────────────────────────────────────────────────────────────

type ctxKey int

const keyLogger ctxKey = iota

func loggerFromCtx(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(keyLogger).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

func withLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, keyLogger, l)
}

// ─────────────────────────────────────────────────────────────────────────────
// STATUS RECORDER
// ─────────────────────────────────────────────────────────────────────────────

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// ─────────────────────────────────────────────────────────────────────────────
// OBSERVABILITY MIDDLEWARE
// - Injects per-request child logger (trace_id, method, path, user_agent)
// - Logs request completed with status/latency/bytes
// - Warns on slow requests (> slowThreshold)
// - Samples 4xx logs (logs ~10%) but always logs 5xx
// ─────────────────────────────────────────────────────────────────────────────

const slowThreshold = 5 * time.Millisecond

func observabilityMW(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			traceID := fmt.Sprintf("%x", rand.Int63())

			reqLogger := base.With(
				slog.String("trace_id", traceID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("user_agent", r.UserAgent()),
			)
			w.Header().Set("X-Trace-Id", traceID)

			reqLogger.Info("request started")

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r.WithContext(withLogger(r.Context(), reqLogger)))

			latency := time.Since(start)

			// Slow request warning.
			if latency > slowThreshold {
				reqLogger.Warn("slow request detected",
					slog.Int64("latency_ms", latency.Milliseconds()),
					slog.String("threshold", slowThreshold.String()),
				)
			}

			attrs := []any{
				slog.Int("status", rec.status),
				slog.Int64("latency_ms", latency.Milliseconds()),
				slog.Int("bytes", rec.bytes),
			}

			switch {
			case rec.status >= 500:
				reqLogger.Error("request completed", attrs...)
			case rec.status >= 400:
				// Sample 4xx at ~10% to reduce noise.
				if rand.Intn(10) == 0 {
					reqLogger.Warn("request completed (sampled)", attrs...)
				}
			default:
				reqLogger.Info("request completed", attrs...)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ORDER DOMAIN EVENTS
// ─────────────────────────────────────────────────────────────────────────────

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusShipped   OrderStatus = "shipped"
	StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID     string      `json:"id"`
	Item   string      `json:"item"`
	Amount int         `json:"amount_cents"`
	Status OrderStatus `json:"status"`
}

var orderDB = map[string]*Order{
	"ord-1": {ID: "ord-1", Item: "Widget", Amount: 999, Status: StatusPending},
	"ord-2": {ID: "ord-2", Item: "Gadget", Amount: 4999, Status: StatusPending},
}

func logOrderEvent(log *slog.Logger, event string, o *Order, extra ...any) {
	attrs := []any{
		slog.String("order_id", o.ID),
		slog.String("item", o.Item),
		slog.Int("amount_cents", o.Amount),
		slog.String("status", string(o.Status)),
	}
	attrs = append(attrs, extra...)
	log.Info(event, attrs...)
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func handleGetOrder(w http.ResponseWriter, r *http.Request) {
	log := loggerFromCtx(r.Context())
	id := r.PathValue("id")

	o, ok := orderDB[id]
	if !ok {
		log.Warn("order not found", slog.String("order_id", id))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		return
	}

	logOrderEvent(log, "order fetched", o)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	log := loggerFromCtx(r.Context())

	var o Order
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		log.Error("invalid request body", slog.String("error", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(o.Item) == "" {
		log.Warn("validation failed", slog.String("field", "item"))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "item required"})
		return
	}

	o.ID = fmt.Sprintf("ord-%x", rand.Int31())
	o.Status = StatusPending
	orderDB[o.ID] = &o

	logOrderEvent(log, "order created", &o)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(o)
}

func handleShipOrder(w http.ResponseWriter, r *http.Request) {
	log := loggerFromCtx(r.Context())
	id := r.PathValue("id")

	o, ok := orderDB[id]
	if !ok {
		log.Warn("ship: order not found", slog.String("order_id", id))
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if o.Status != StatusPending {
		log.Warn("ship: invalid state transition",
			slog.String("order_id", id),
			slog.String("current_status", string(o.Status)),
		)
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "order not in pending state"})
		return
	}

	prev := o.Status
	o.Status = StatusShipped
	logOrderEvent(log, "order shipped", o, slog.String("prev_status", string(prev)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func handleSlowOrder(w http.ResponseWriter, r *http.Request) {
	log := loggerFromCtx(r.Context())
	log.Debug("slow handler starting")
	// Simulate slow processing — triggers slow request warning.
	time.Sleep(10 * time.Millisecond)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "done"})
}

// ─────────────────────────────────────────────────────────────────────────────
// TEST HELPER — discard logger
// ─────────────────────────────────────────────────────────────────────────────

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// Multi-handler: text (DEBUG+) → stderr; JSON (INFO+) → stdout.
	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	base := slog.New(newMultiHandler(debugHandler, jsonHandler))
	slog.SetDefault(base)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /orders/{id}", handleGetOrder)
	mux.HandleFunc("POST /orders", handleCreateOrder)
	mux.HandleFunc("POST /orders/{id}/ship", handleShipOrder)
	mux.HandleFunc("GET /slow", handleSlowOrder)

	handler := observabilityMW(base)(mux)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := "http://" + ln.Addr().String()
	go (&http.Server{Handler: handler}).Serve(ln)

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	do := func(method, path, body string) (int, string) {
		var br io.Reader = strings.NewReader("")
		if body != "" {
			br = strings.NewReader(body)
		}
		req, _ := http.NewRequest(method, addr+path, br)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("User-Agent", "upskill-go-test/1.0")
		resp, err := client.Do(req)
		if err != nil {
			return 0, ""
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, strings.TrimSpace(string(b))
	}

	check := func(label string, got, want int) {
		if got == want {
			fmt.Printf("  ✓ %s: %d\n", label, got)
		} else {
			fmt.Printf("  ✗ %s: got %d, want %d\n", label, got, want)
		}
	}

	fmt.Printf("=== Observability Logger — %s ===\n", addr)
	fmt.Println("(JSON lines appear on stdout; debug text on stderr)")
	fmt.Println()

	fmt.Println("--- GET /orders/ord-1 (200) ---")
	code, _ := do("GET", "/orders/ord-1", "")
	check("get order", code, 200)

	fmt.Println()
	fmt.Println("--- GET /orders/missing (404) ---")
	code, _ = do("GET", "/orders/missing", "")
	check("not found", code, 404)

	fmt.Println()
	fmt.Println("--- POST /orders (201 created) ---")
	code, body := do("POST", "/orders", `{"item":"Sprocket","amount_cents":750}`)
	check("create order", code, 201)
	var created Order
	json.Unmarshal([]byte(body), &created)
	fmt.Printf("  created.id = %s  status = %s\n", created.ID, created.Status)

	fmt.Println()
	fmt.Println("--- POST /orders/:id/ship (200 shipped) ---")
	code, _ = do("POST", "/orders/ord-1/ship", "")
	check("ship order", code, 200)

	fmt.Println()
	fmt.Println("--- POST /orders/:id/ship again (409 conflict) ---")
	code, _ = do("POST", "/orders/ord-1/ship", "")
	check("ship again conflict", code, 409)

	fmt.Println()
	fmt.Println("--- GET /slow (triggers slow-request warning) ---")
	code, _ = do("GET", "/slow", "")
	check("slow handler", code, 200)

	fmt.Println()
	fmt.Println("--- POST /orders (422 missing item) ---")
	code, _ = do("POST", "/orders", `{"amount_cents":100}`)
	check("validation error", code, 422)

	fmt.Println()
	fmt.Println("--- Discard logger (for tests) ---")
	testLogger := newDiscardLogger()
	testLogger.Info("this goes nowhere — no output expected")
	fmt.Println("  ✓ discard logger: no output above")
}
