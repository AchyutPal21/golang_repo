// FILE: book/part6_production_engineering/chapter91_opentelemetry/examples/01_traces_spans/main.go
// CHAPTER: 91 — OpenTelemetry
// TOPIC: Traces, spans, parent-child relationships, attributes, events —
//        pure in-process simulation without an OTLP collector.
//
// Run:
//   go run ./part6_production_engineering/chapter91_opentelemetry/examples/01_traces_spans/

package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// IN-PROCESS TRACER — simulates OpenTelemetry trace/span API
// ─────────────────────────────────────────────────────────────────────────────

type TraceID [16]byte
type SpanID  [8]byte

func newTraceID() TraceID {
	var id TraceID
	for i := range id {
		id[i] = byte(rand.IntN(256))
	}
	return id
}

func newSpanID() SpanID {
	var id SpanID
	for i := range id {
		id[i] = byte(rand.IntN(256))
	}
	return id
}

func (t TraceID) String() string {
	return fmt.Sprintf("%x", t[:])
}

func (s SpanID) String() string {
	return fmt.Sprintf("%x", s[:])
}

// SpanStatus mirrors OTEL's status codes.
type SpanStatus int

const (
	StatusUnset SpanStatus = iota
	StatusOK
	StatusError
)

// Attribute is a typed key-value pair.
type Attribute struct{ Key, Value string }

func Attr(k, v string) Attribute { return Attribute{k, v} }

// Span represents a single unit of work in a trace.
type Span struct {
	traceID    TraceID
	spanID     SpanID
	parentID   SpanID
	name       string
	startTime  time.Time
	endTime    time.Time
	attributes []Attribute
	events     []spanEvent
	status     SpanStatus
	statusMsg  string
	mu         sync.Mutex
	finished   bool
}

type spanEvent struct {
	name string
	time time.Time
	attrs []Attribute
}

func (s *Span) SetAttribute(k, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attributes = append(s.attributes, Attribute{k, v})
}

func (s *Span) AddEvent(name string, attrs ...Attribute) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, spanEvent{name, time.Now(), attrs})
}

func (s *Span) SetStatus(status SpanStatus, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
	s.statusMsg = msg
}

func (s *Span) End() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.finished {
		s.endTime = time.Now()
		s.finished = true
	}
}

func (s *Span) Duration() time.Duration {
	return s.endTime.Sub(s.startTime)
}

// ─────────────────────────────────────────────────────────────────────────────
// IN-PROCESS TRACER
// ─────────────────────────────────────────────────────────────────────────────

type Tracer struct {
	mu    sync.Mutex
	spans []*Span
}

var globalTracer = &Tracer{}

type spanCtxKey struct{}

// Start creates a new span. If ctx already has a span, the new span is a child.
func (t *Tracer) Start(ctx context.Context, name string, attrs ...Attribute) (context.Context, *Span) {
	var traceID TraceID
	var parentID SpanID
	if parent, ok := ctx.Value(spanCtxKey{}).(*Span); ok {
		traceID = parent.traceID
		parentID = parent.spanID
	} else {
		traceID = newTraceID()
	}
	span := &Span{
		traceID:    traceID,
		spanID:     newSpanID(),
		parentID:   parentID,
		name:       name,
		startTime:  time.Now(),
		attributes: attrs,
	}
	t.mu.Lock()
	t.spans = append(t.spans, span)
	t.mu.Unlock()
	return context.WithValue(ctx, spanCtxKey{}, span), span
}

// Spans returns all collected spans.
func (t *Tracer) Spans() []*Span {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make([]*Span, len(t.spans))
	copy(result, t.spans)
	return result
}

// SpanFromCtx retrieves the current span from context.
func SpanFromCtx(ctx context.Context) *Span {
	s, _ := ctx.Value(spanCtxKey{}).(*Span)
	return s
}

// ─────────────────────────────────────────────────────────────────────────────
// RENDERING — human-readable trace output
// ─────────────────────────────────────────────────────────────────────────────

func printTrace(spans []*Span) {
	if len(spans) == 0 {
		return
	}
	root := spans[0]
	fmt.Printf("Trace: %s\n", root.traceID)
	for _, s := range spans {
		depth := 0
		// Find depth by counting ancestors.
		cur := s
		for {
			found := false
			for _, other := range spans {
				if other.spanID == cur.parentID {
					depth++
					cur = other
					found = true
					break
				}
			}
			if !found {
				break
			}
		}
		indent := strings.Repeat("  ", depth)
		status := ""
		switch s.status {
		case StatusError:
			status = " [ERROR: " + s.statusMsg + "]"
		case StatusOK:
			status = " [OK]"
		}
		fmt.Printf("  %s%-30s %8v%s\n", indent, s.name, s.Duration().Round(time.Microsecond), status)
		for _, a := range s.attributes {
			fmt.Printf("  %s  attr: %s=%s\n", indent, a.Key, a.Value)
		}
		for _, e := range s.events {
			fmt.Printf("  %s  event: %s\n", indent, e.name)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED SERVICE HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func handleHTTPRequest(ctx context.Context, method, path string) {
	ctx, span := globalTracer.Start(ctx, "HTTP "+method+" "+path,
		Attr("http.method", method),
		Attr("http.path", path),
	)
	defer span.End()

	span.AddEvent("request_received")

	// Call the service layer
	if err := serviceLayer(ctx, path); err != nil {
		span.SetStatus(StatusError, err.Error())
		span.SetAttribute("error", err.Error())
	} else {
		span.SetStatus(StatusOK, "")
		span.SetAttribute("http.status_code", "200")
	}
	span.AddEvent("response_sent")
}

func serviceLayer(ctx context.Context, path string) error {
	ctx, span := globalTracer.Start(ctx, "service.process",
		Attr("service", "order-service"),
	)
	defer span.End()

	span.AddEvent("validating_input")
	time.Sleep(time.Duration(rand.IntN(3)+1) * time.Millisecond)

	if err := dbQuery(ctx, "SELECT * FROM orders WHERE path=?"); err != nil {
		return err
	}
	if err := cacheGet(ctx, "orders:"+path); err != nil {
		span.AddEvent("cache_miss")
	}
	return nil
}

func dbQuery(ctx context.Context, sql string) error {
	_, span := globalTracer.Start(ctx, "db.query",
		Attr("db.system", "postgresql"),
		Attr("db.statement", sql),
	)
	defer span.End()
	time.Sleep(time.Duration(rand.IntN(5)+2) * time.Millisecond)
	span.SetStatus(StatusOK, "")
	return nil
}

func cacheGet(ctx context.Context, key string) error {
	_, span := globalTracer.Start(ctx, "cache.get",
		Attr("cache.key", key),
		Attr("cache.system", "redis"),
	)
	defer span.End()
	time.Sleep(time.Millisecond)
	// Simulate occasional cache miss (return error to signal miss)
	if rand.Float64() < 0.4 {
		return fmt.Errorf("cache miss: %s", key)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// OTEL CONCEPTS REFERENCE
// ─────────────────────────────────────────────────────────────────────────────

const otelRef = `
OpenTelemetry concepts:

  Signal types:
    Traces  — distributed request flow (this chapter)
    Metrics — aggregated counters/histograms (chapter 90)
    Logs    — structured log records (chapter 89)

  Trace anatomy:
    Trace  — a tree of spans sharing a TraceID
    Span   — one unit of work: name, start/end, attributes, events, status
    Context — propagates TraceID + SpanID across process boundaries

  Context propagation formats:
    W3C TraceContext (traceparent header)  ← standard, recommended
    B3 (Zipkin)                            ← legacy
    Jaeger                                 ← legacy

  Sampling strategies:
    AlwaysOn        — 100% (dev/test only)
    AlwaysOff       — 0% (disable tracing)
    TraceIDRatio    — e.g., 0.01 = 1% of traces
    ParentBased     — follow parent's sampling decision (recommended)
    Tail-based      — sample after seeing the full trace (Jaeger/Tempo)

  Span attributes (semantic conventions):
    http.method, http.status_code, http.url
    db.system, db.statement, db.name
    messaging.system, messaging.destination
    rpc.system, rpc.method, rpc.service
`

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 91: Traces & Spans ===")
	fmt.Println()

	// ── SINGLE TRACE ─────────────────────────────────────────────────────────
	fmt.Println("--- Single request trace ---")
	ctx := context.Background()
	handleHTTPRequest(ctx, "GET", "/api/orders/42")
	printTrace(globalTracer.Spans())
	fmt.Println()

	// ── CONCURRENT TRACES ─────────────────────────────────────────────────────
	fmt.Println("--- Concurrent request traces ---")
	globalTracer = &Tracer{} // reset
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			handleHTTPRequest(context.Background(), "POST", fmt.Sprintf("/api/items/%d", n))
		}(i)
	}
	wg.Wait()
	spans := globalTracer.Spans()
	fmt.Printf("  %d spans collected across 3 concurrent traces\n", len(spans))
	// Count unique traces
	traces := make(map[TraceID]bool)
	for _, s := range spans {
		traces[s.traceID] = true
	}
	fmt.Printf("  Unique traces: %d\n", len(traces))
	fmt.Println()

	// ── OTEL REFERENCE ────────────────────────────────────────────────────────
	fmt.Println("--- OpenTelemetry concepts ---")
	fmt.Print(otelRef)
}
