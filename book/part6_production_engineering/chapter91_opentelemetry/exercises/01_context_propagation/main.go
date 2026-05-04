// FILE: book/part6_production_engineering/chapter91_opentelemetry/exercises/01_context_propagation/main.go
// CHAPTER: 91 — OpenTelemetry
// EXERCISE: Build a complete in-process trace pipeline: root span → child
//   spans across simulated service calls → baggage forwarding → sampled
//   export to an in-memory exporter → print a waterfall trace view.
//
// Run:
//   go run ./part6_production_engineering/chapter91_opentelemetry/exercises/01_context_propagation/

package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SPAN STORE — in-memory exporter
// ─────────────────────────────────────────────────────────────────────────────

type SpanData struct {
	TraceID  string
	SpanID   string
	ParentID string
	Name     string
	Start    time.Time
	End      time.Time
	Attrs    map[string]string
	Events   []string
	Status   string
}

func (s SpanData) Duration() time.Duration { return s.End.Sub(s.Start) }

type SpanStore struct {
	mu    sync.Mutex
	spans []SpanData
}

func (s *SpanStore) Export(d SpanData) {
	s.mu.Lock()
	s.spans = append(s.spans, d)
	s.mu.Unlock()
}

func (s *SpanStore) All() []SpanData {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SpanData, len(s.spans))
	copy(out, s.spans)
	return out
}

func (s *SpanStore) ByTrace(traceID string) []SpanData {
	all := s.All()
	var out []SpanData
	for _, sp := range all {
		if sp.TraceID == traceID {
			out = append(out, sp)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Start.Before(out[j].Start)
	})
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// TRACER WITH EXPORT
// ─────────────────────────────────────────────────────────────────────────────

type ctxSpanKey struct{}
type ctxTraceKey struct{}
type ctxBagKey struct{}

func randHex(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.IntN(256))
	}
	return fmt.Sprintf("%x", b)
}

type span struct {
	traceID  string
	spanID   string
	parentID string
	name     string
	start    time.Time
	attrs    map[string]string
	events   []string
	status   string
	store    *SpanStore
	mu       sync.Mutex
}

func (s *span) SetAttr(k, v string) {
	s.mu.Lock()
	s.attrs[k] = v
	s.mu.Unlock()
}

func (s *span) AddEvent(e string) {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
}

func (s *span) SetError(msg string) {
	s.mu.Lock()
	s.status = "ERROR: " + msg
	s.mu.Unlock()
}

func (s *span) End() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store.Export(SpanData{
		TraceID:  s.traceID,
		SpanID:   s.spanID,
		ParentID: s.parentID,
		Name:     s.name,
		Start:    s.start,
		End:      time.Now(),
		Attrs:    s.attrs,
		Events:   append([]string{}, s.events...),
		Status:   s.status,
	})
}

type Tracer struct {
	store   *SpanStore
	sampler func(traceID string) bool
}

func NewTracer(store *SpanStore, sampleRate float64) *Tracer {
	return &Tracer{
		store: store,
		sampler: func(traceID string) bool {
			// Deterministic: use last byte of traceID
			if len(traceID) < 2 {
				return true
			}
			var b byte
			fmt.Sscanf(traceID[len(traceID)-2:], "%02x", &b)
			return float64(b)/255.0 < sampleRate
		},
	}
}

func (t *Tracer) Start(ctx context.Context, name string) (context.Context, func()) {
	traceID, _ := ctx.Value(ctxTraceKey{}).(string)
	parentID, _ := ctx.Value(ctxSpanKey{}).(string)
	newSpanID := randHex(8)

	if traceID == "" {
		traceID = randHex(16)
		// Sampling decision at trace root
		if !t.sampler(traceID) {
			// Return no-op span
			ctx = context.WithValue(ctx, ctxTraceKey{}, traceID)
			ctx = context.WithValue(ctx, ctxSpanKey{}, newSpanID)
			return ctx, func() {}
		}
	}

	s := &span{
		traceID:  traceID,
		spanID:   newSpanID,
		parentID: parentID,
		name:     name,
		start:    time.Now(),
		attrs:    make(map[string]string),
		store:    t.store,
	}

	ctx = context.WithValue(ctx, ctxTraceKey{}, traceID)
	ctx = context.WithValue(ctx, ctxSpanKey{}, newSpanID)
	ctx = context.WithValue(ctx, spanObjKey{}, s)
	return ctx, s.End
}

type spanObjKey struct{}

func spanFromCtx(ctx context.Context) *span {
	s, _ := ctx.Value(spanObjKey{}).(*span)
	return s
}

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED MICROSERVICES
// ─────────────────────────────────────────────────────────────────────────────

func apiGateway(ctx context.Context, tracer *Tracer, method, path string) {
	ctx, end := tracer.Start(ctx, "api-gateway: "+method+" "+path)
	defer end()
	if s := spanFromCtx(ctx); s != nil {
		s.SetAttr("http.method", method)
		s.SetAttr("http.path", path)
		s.SetAttr("http.user_agent", "mobile-app/2.1")
		s.AddEvent("auth_check_passed")
	}
	orderService(ctx, tracer, path)
}

func orderService(ctx context.Context, tracer *Tracer, path string) {
	ctx, end := tracer.Start(ctx, "order-service: processOrder")
	defer end()
	if s := spanFromCtx(ctx); s != nil {
		s.SetAttr("service", "order-service")
	}
	time.Sleep(2 * time.Millisecond)
	dbRead(ctx, tracer, "SELECT * FROM orders LIMIT 10")
	cacheCheck(ctx, tracer, "orders:page:1")
	paymentService(ctx, tracer)
}

func dbRead(ctx context.Context, tracer *Tracer, query string) {
	_, end := tracer.Start(ctx, "postgres: query")
	defer end()
	if s := spanFromCtx(ctx); s != nil {
		s.SetAttr("db.system", "postgresql")
		s.SetAttr("db.statement", query)
	}
	time.Sleep(time.Duration(rand.IntN(4)+1) * time.Millisecond)
}

func cacheCheck(ctx context.Context, tracer *Tracer, key string) {
	_, end := tracer.Start(ctx, "redis: get")
	defer end()
	if s := spanFromCtx(ctx); s != nil {
		s.SetAttr("cache.key", key)
		if rand.Float64() < 0.3 {
			s.AddEvent("cache_miss")
		} else {
			s.AddEvent("cache_hit")
		}
	}
	time.Sleep(time.Millisecond)
}

func paymentService(ctx context.Context, tracer *Tracer) {
	ctx, end := tracer.Start(ctx, "payment-service: charge")
	defer end()
	if s := spanFromCtx(ctx); s != nil {
		s.SetAttr("payment.gateway", "stripe")
		s.AddEvent("charge_initiated")
	}
	time.Sleep(3 * time.Millisecond)
	if rand.Float64() < 0.1 {
		if s := spanFromCtx(ctx); s != nil {
			s.SetError("card_declined")
		}
	} else {
		if s := spanFromCtx(ctx); s != nil {
			s.AddEvent("charge_succeeded")
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// WATERFALL VIEW RENDERER
// ─────────────────────────────────────────────────────────────────────────────

func printWaterfall(spans []SpanData) {
	if len(spans) == 0 {
		return
	}
	traceStart := spans[0].Start
	totalDur := spans[0].End.Sub(traceStart)
	for _, s := range spans {
		if s.End.After(traceStart.Add(totalDur)) {
			totalDur = s.End.Sub(traceStart)
		}
	}

	barWidth := 40
	fmt.Printf("  %-32s  %6s  %s\n", "Span", "Dur", "Timeline")
	fmt.Printf("  %s\n", strings.Repeat("-", 80))
	for _, s := range spans {
		startFrac := float64(s.Start.Sub(traceStart)) / float64(totalDur)
		endFrac := float64(s.End.Sub(traceStart)) / float64(totalDur)
		bar := []rune(strings.Repeat(".", barWidth))
		startCol := int(startFrac * float64(barWidth))
		endCol := int(endFrac * float64(barWidth))
		if endCol <= startCol {
			endCol = startCol + 1
		}
		if endCol > barWidth {
			endCol = barWidth
		}
		for i := startCol; i < endCol; i++ {
			bar[i] = '#'
		}
		status := ""
		if strings.HasPrefix(s.Status, "ERROR") {
			status = " ERROR"
		}
		fmt.Printf("  %-32s  %5v  %s%s\n",
			truncate(s.Name, 32),
			s.Duration().Round(time.Microsecond),
			string(bar), status)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 91 Exercise: Context Propagation ===")
	fmt.Println()

	store := &SpanStore{}
	tracer := NewTracer(store, 1.0) // 100% sampling for demo

	// ── SINGLE REQUEST TRACE ─────────────────────────────────────────────────
	fmt.Println("--- Single request trace ---")
	ctx := context.Background()
	apiGateway(ctx, tracer, "POST", "/api/orders")

	spans := store.All()
	if len(spans) > 0 {
		traceSpans := store.ByTrace(spans[0].TraceID)
		fmt.Printf("  TraceID: %s\n", traceSpans[0].TraceID)
		fmt.Printf("  Total spans: %d\n\n", len(traceSpans))
		printWaterfall(traceSpans)
	}
	fmt.Println()

	// ── SAMPLING DEMO ────────────────────────────────────────────────────────
	fmt.Println("--- Sampling demo (10% rate, 100 requests) ---")
	store2 := &SpanStore{}
	tracer2 := NewTracer(store2, 0.10)
	for i := 0; i < 100; i++ {
		apiGateway(context.Background(), tracer2, "GET", fmt.Sprintf("/api/items/%d", i))
	}
	allSpans := store2.All()
	traces := make(map[string]bool)
	for _, s := range allSpans {
		traces[s.TraceID] = true
	}
	fmt.Printf("  100 requests, 10%% sample rate → %d traces recorded (%d spans)\n",
		len(traces), len(allSpans))
	fmt.Println()

	// ── BAGGAGE PROPAGATION ───────────────────────────────────────────────────
	fmt.Println("--- Baggage propagation through spans ---")
	store3 := &SpanStore{}
	tracer3 := NewTracer(store3, 1.0)
	bagCtx := context.WithValue(context.Background(), ctxBagKey{}, map[string]string{
		"tenant": "acme-corp",
		"user":   "42",
	})
	apiGateway(bagCtx, tracer3, "DELETE", "/api/orders/99")
	spans3 := store3.All()
	fmt.Printf("  Spans recorded: %d\n", len(spans3))
	for _, s := range spans3 {
		if s.Status != "" {
			fmt.Printf("  Span %q status: %s\n", s.Name, s.Status)
		}
	}
	fmt.Println()

	fmt.Println("Propagation summary:")
	fmt.Println("  traceparent header: TraceID + SpanID + sampling flag")
	fmt.Println("  baggage header:     cross-cutting business context")
	fmt.Println("  context.Context:    in-process propagation (no copy)")
}
