// FILE: book/part5_building_backends/chapter63_structured_logging/examples/02_log_middleware/main.go
// CHAPTER: 63 — Structured Logging
// TOPIC: Structured logging middleware for HTTP —
//        per-request child logger injected via context, request/response logging,
//        structured error events, and log output suitable for Loki/CloudWatch.
//
// Run (from the chapter folder):
//   go run ./examples/02_log_middleware

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
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT KEY
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
// LOGGING MIDDLEWARE
// Injects a per-request logger (with request_id, method, path) into context.
// Logs the completed request: status, latency_ms, bytes.
// ─────────────────────────────────────────────────────────────────────────────

func loggingMW(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := fmt.Sprintf("%x", rand.Int63())

			// Create a child logger with request-level fields.
			// Every log line in this request will carry these attributes.
			reqLogger := base.With(
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
			)

			reqLogger.Info("request started")

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r.WithContext(withLogger(r.Context(), reqLogger)))

			latency := time.Since(start)
			logFn := reqLogger.Info
			if rec.status >= 500 {
				logFn = reqLogger.Error
			} else if rec.status >= 400 {
				logFn = reqLogger.Warn
			}
			logFn("request completed",
				slog.Int("status", rec.status),
				slog.Int64("latency_ms", latency.Milliseconds()),
				slog.Int("bytes", rec.bytes),
			)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS — use logger from context to log domain events
// ─────────────────────────────────────────────────────────────────────────────

type Order struct {
	ID     string `json:"id"`
	Item   string `json:"item"`
	Amount int    `json:"amount_cents"`
}

var orderDB = map[string]*Order{
	"ord-1": {ID: "ord-1", Item: "Widget", Amount: 999},
	"ord-2": {ID: "ord-2", Item: "Gadget", Amount: 4999},
}

func handleGetOrder(w http.ResponseWriter, r *http.Request) {
	log := loggerFromCtx(r.Context())
	id := r.PathValue("id")

	log.Info("fetching order", slog.String("order_id", id))

	o, ok := orderDB[id]
	if !ok {
		log.Warn("order not found", slog.String("order_id", id))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "order not found"})
		return
	}

	log.Info("order found", slog.String("order_id", id), slog.String("item", o.Item))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	log := loggerFromCtx(r.Context())

	var o Order
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		log.Error("failed to decode request body", slog.String("error", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(o.Item) == "" {
		log.Warn("validation failed", slog.String("field", "item"), slog.String("reason", "required"))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "item required"})
		return
	}

	o.ID = fmt.Sprintf("ord-%x", rand.Int31())
	orderDB[o.ID] = &o

	log.Info("order created",
		slog.String("order_id", o.ID),
		slog.String("item", o.Item),
		slog.Int("amount_cents", o.Amount),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(o)
}

func handlePanicOrder(w http.ResponseWriter, r *http.Request) {
	log := loggerFromCtx(r.Context())
	log.Info("about to panic intentionally")
	panic("intentional panic for demo")
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// JSON logger — every output line is a valid JSON object.
	baseLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(baseLogger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /orders/{id}", handleGetOrder)
	mux.HandleFunc("POST /orders", handleCreateOrder)
	mux.HandleFunc("GET /panic", handlePanicOrder)

	// Recovery middleware (outermost) + logging middleware.
	recoveryMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					log := loggerFromCtx(r.Context())
					log.Error("panic recovered",
						slog.Any("panic", v),
						slog.String("path", r.URL.Path),
					)
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}

	// Chain: recovery → logging → mux
	handler := recoveryMW(loggingMW(baseLogger)(mux))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: handler}).Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}

	do := func(method, path, body string) int {
		var br *strings.Reader
		if body != "" {
			br = strings.NewReader(body)
		} else {
			br = strings.NewReader("")
		}
		req, _ := http.NewRequest(method, base+path, br)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0
		}
		resp.Body.Close()
		return resp.StatusCode
	}

	fmt.Printf("=== HTTP Logging Middleware — %s ===\n", base)
	fmt.Println("(All JSON log lines below; request_id ties all logs for a single request)")
	fmt.Println()

	fmt.Println("--- GET /orders/ord-1 (200) ---")
	do("GET", "/orders/ord-1", "")

	fmt.Println()
	fmt.Println("--- GET /orders/missing (404) ---")
	do("GET", "/orders/missing", "")

	fmt.Println()
	fmt.Println("--- POST /orders (201) ---")
	do("POST", "/orders", `{"item":"Thingamajig","amount_cents":2500}`)

	fmt.Println()
	fmt.Println("--- POST /orders (422 — missing item) ---")
	do("POST", "/orders", `{"amount_cents":100}`)

	fmt.Println()
	fmt.Println("--- GET /panic (500 — panic recovered) ---")
	do("GET", "/panic", "")
}
