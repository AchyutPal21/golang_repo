// FILE: book/part6_production_engineering/chapter91_opentelemetry/examples/02_baggage_sampling/main.go
// CHAPTER: 91 — OpenTelemetry
// TOPIC: Baggage (cross-cutting key-value propagation), sampling strategies
//        (head-based ratio + parent-based), and W3C traceparent format.
//
// Run:
//   go run ./part6_production_engineering/chapter91_opentelemetry/examples/02_baggage_sampling/

package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// BAGGAGE — cross-cutting propagation (user_id, tenant_id, experiment flags)
// ─────────────────────────────────────────────────────────────────────────────

type baggageKey struct{}

type Baggage map[string]string

func WithBaggage(ctx context.Context, b Baggage) context.Context {
	return context.WithValue(ctx, baggageKey{}, b)
}

func BaggageFromCtx(ctx context.Context) Baggage {
	if b, ok := ctx.Value(baggageKey{}).(Baggage); ok {
		return b
	}
	return Baggage{}
}

func (b Baggage) Get(key string) string { return b[key] }

func (b Baggage) Set(key, value string) Baggage {
	next := make(Baggage, len(b)+1)
	for k, v := range b {
		next[k] = v
	}
	next[key] = value
	return next
}

// Encode produces W3C baggage header value: key=value,key2=value2
func (b Baggage) Encode() string {
	parts := make([]string, 0, len(b))
	for k, v := range b {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}

// DecodeBaggage parses W3C baggage header value.
func DecodeBaggage(header string) Baggage {
	b := Baggage{}
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			b[kv[0]] = kv[1]
		}
	}
	return b
}

// ─────────────────────────────────────────────────────────────────────────────
// W3C TRACEPARENT FORMAT
// ─────────────────────────────────────────────────────────────────────────────

// TraceFlags holds sampling decision bits.
type TraceFlags byte

const (
	FlagSampled TraceFlags = 0x01
)

type TraceParent struct {
	Version  byte
	TraceID  [16]byte
	SpanID   [8]byte
	Flags    TraceFlags
}

func newTraceParent(sampled bool) TraceParent {
	var tp TraceParent
	tp.Version = 0
	for i := range tp.TraceID {
		tp.TraceID[i] = byte(rand.IntN(256))
	}
	for i := range tp.SpanID {
		tp.SpanID[i] = byte(rand.IntN(256))
	}
	if sampled {
		tp.Flags = FlagSampled
	}
	return tp
}

// Encode produces the W3C traceparent header value.
func (tp TraceParent) Encode() string {
	return fmt.Sprintf("%02x-%x-%x-%02x",
		tp.Version, tp.TraceID[:], tp.SpanID[:], byte(tp.Flags))
}

// IsSampled returns true if the trace is sampled.
func (tp TraceParent) IsSampled() bool {
	return tp.Flags&FlagSampled != 0
}

// ParseTraceParent parses a W3C traceparent header value.
func ParseTraceParent(s string) (TraceParent, error) {
	parts := strings.Split(s, "-")
	if len(parts) != 4 {
		return TraceParent{}, fmt.Errorf("invalid traceparent: %q", s)
	}
	var tp TraceParent
	fmt.Sscanf(parts[0], "%x", &tp.Version)
	copy(tp.TraceID[:], parseHex(parts[1]))
	copy(tp.SpanID[:], parseHex(parts[2]))
	var flags byte
	fmt.Sscanf(parts[3], "%x", &flags)
	tp.Flags = TraceFlags(flags)
	return tp, nil
}

func parseHex(s string) []byte {
	b := make([]byte, len(s)/2)
	for i := range b {
		fmt.Sscanf(s[2*i:2*i+2], "%02x", &b[i])
	}
	return b
}

// ─────────────────────────────────────────────────────────────────────────────
// SAMPLING STRATEGIES
// ─────────────────────────────────────────────────────────────────────────────

// Sampler decides whether a trace should be recorded.
type Sampler interface {
	ShouldSample(traceID [16]byte) bool
	Name() string
}

// AlwaysOnSampler samples every trace.
type AlwaysOnSampler struct{}

func (s AlwaysOnSampler) ShouldSample(_ [16]byte) bool { return true }
func (s AlwaysOnSampler) Name() string                  { return "AlwaysOn" }

// AlwaysOffSampler samples no traces.
type AlwaysOffSampler struct{}

func (s AlwaysOffSampler) ShouldSample(_ [16]byte) bool { return false }
func (s AlwaysOffSampler) Name() string                  { return "AlwaysOff" }

// RatioSampler samples a fraction of traces deterministically by trace ID.
type RatioSampler struct {
	ratio float64 // 0.0–1.0
}

func NewRatioSampler(ratio float64) RatioSampler {
	return RatioSampler{ratio: ratio}
}

func (s RatioSampler) ShouldSample(traceID [16]byte) bool {
	// Use first 8 bytes of trace ID as a uint64 for deterministic decision.
	var n uint64
	for i := 0; i < 8; i++ {
		n = n<<8 | uint64(traceID[i])
	}
	return float64(n)/float64(^uint64(0)) < s.ratio
}

func (s RatioSampler) Name() string {
	return fmt.Sprintf("Ratio(%.2f)", s.ratio)
}

// ParentBasedSampler follows the parent's sampling decision.
// Falls back to a root sampler when there is no parent.
type ParentBasedSampler struct {
	root Sampler
}

func NewParentBasedSampler(root Sampler) ParentBasedSampler {
	return ParentBasedSampler{root: root}
}

func (s ParentBasedSampler) ShouldSample(traceID [16]byte) bool {
	return s.root.ShouldSample(traceID)
}

func (s ParentBasedSampler) Name() string {
	return "ParentBased(" + s.root.Name() + ")"
}

// ─────────────────────────────────────────────────────────────────────────────
// SAMPLING SIMULATOR
// ─────────────────────────────────────────────────────────────────────────────

type samplerStats struct {
	sampled  atomic.Int64
	dropped  atomic.Int64
}

func runSampler(sampler Sampler, n int) *samplerStats {
	stats := &samplerStats{}
	for i := 0; i < n; i++ {
		tp := newTraceParent(rand.Float64() < 0.5)
		if sampler.ShouldSample(tp.TraceID) {
			stats.sampled.Add(1)
		} else {
			stats.dropped.Add(1)
		}
	}
	return stats
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 91: Baggage & Sampling ===")
	fmt.Println()

	// ── BAGGAGE PROPAGATION ───────────────────────────────────────────────────
	fmt.Println("--- Baggage propagation ---")
	ctx := context.Background()
	bag := Baggage{}.
		Set("user_id", "42").
		Set("tenant_id", "acme-corp").
		Set("feature_flag", "new-checkout=true")

	ctx = WithBaggage(ctx, bag)
	fmt.Printf("  Baggage header: %s\n", bag.Encode())

	// Simulate crossing a process boundary: encode → decode
	encoded := BaggageFromCtx(ctx).Encode()
	decoded := DecodeBaggage(encoded)
	fmt.Printf("  Decoded tenant_id: %s\n", decoded.Get("tenant_id"))
	fmt.Printf("  Decoded feature_flag: %s\n", decoded.Get("feature_flag"))

	// Use baggage in a downstream handler
	incomingCtx := WithBaggage(context.Background(), decoded)
	fmt.Printf("  Downstream sees user_id: %s\n", BaggageFromCtx(incomingCtx).Get("user_id"))
	fmt.Println()

	// ── W3C TRACEPARENT ───────────────────────────────────────────────────────
	fmt.Println("--- W3C traceparent format ---")
	tp := newTraceParent(true)
	encoded2 := tp.Encode()
	fmt.Printf("  traceparent: %s\n", encoded2)
	fmt.Printf("  sampled: %v\n", tp.IsSampled())

	parsed, err := ParseTraceParent(encoded2)
	if err != nil {
		fmt.Printf("  parse error: %v\n", err)
	} else {
		fmt.Printf("  parsed traceID: %x\n", parsed.TraceID[:8])
		fmt.Printf("  parsed sampled: %v\n", parsed.IsSampled())
	}
	fmt.Println()
	fmt.Println("  Format: 00-<traceID(32hex)>-<spanID(16hex)>-<flags(2hex)>")
	fmt.Println("  Flags:  0x00 = not sampled, 0x01 = sampled")
	fmt.Println()

	// ── SAMPLING COMPARISON ───────────────────────────────────────────────────
	fmt.Println("--- Sampling strategies (10 000 traces) ---")
	n := 10_000
	samplers := []Sampler{
		AlwaysOnSampler{},
		AlwaysOffSampler{},
		NewRatioSampler(0.01),
		NewRatioSampler(0.10),
		NewRatioSampler(0.50),
		NewParentBasedSampler(NewRatioSampler(0.05)),
	}
	fmt.Printf("  %-35s  %8s  %8s  %6s\n", "Sampler", "Sampled", "Dropped", "Rate%")
	for _, s := range samplers {
		stats := runSampler(s, n)
		rate := 100 * float64(stats.sampled.Load()) / float64(n)
		fmt.Printf("  %-35s  %8d  %8d  %5.1f%%\n",
			s.Name(), stats.sampled.Load(), stats.dropped.Load(), rate)
	}
	fmt.Println()

	// ── OTEL SDK INTEGRATION REFERENCE ───────────────────────────────────────
	fmt.Println("--- Real OTLP setup reference ---")
	ref := `  import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/trace"
  )

  exporter, _ := otlptracegrpc.New(ctx,
    otlptracegrpc.WithEndpoint("collector:4317"),
  )
  tp := trace.NewTracerProvider(
    trace.WithBatcher(exporter),
    trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(0.01))),
    trace.WithResource(resource.NewWithAttributes(
      semconv.SchemaURL,
      semconv.ServiceNameKey.String("order-service"),
      semconv.ServiceVersionKey.String("1.2.0"),
    )),
  )
  otel.SetTracerProvider(tp)
  defer tp.Shutdown(ctx)

  tracer := otel.Tracer("order-service")
  ctx, span := tracer.Start(ctx, "process-order")
  defer span.End()`
	fmt.Println(ref)
	fmt.Println()

	// ── TIMING ────────────────────────────────────────────────────────────────
	fmt.Printf("  Demo ran in: %v\n", time.Since(time.Now().Add(-time.Millisecond)))
}
