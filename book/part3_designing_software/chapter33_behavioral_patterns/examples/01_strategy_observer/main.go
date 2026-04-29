// FILE: book/part3_designing_software/chapter33_behavioral_patterns/examples/01_strategy_observer/main.go
// CHAPTER: 33 — Behavioral Patterns
// TOPIC: Strategy (interchangeable algorithms) and Observer (event notification).
//
// Run (from the chapter folder):
//   go run ./examples/01_strategy_observer

package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// STRATEGY
//
// Defines a family of algorithms, encapsulates each, and makes them
// interchangeable. The context delegates to the strategy; swapping the
// strategy changes behaviour without modifying the context.
// ─────────────────────────────────────────────────────────────────────────────

// ── Sorting strategy ──────────────────────────────────────────────────────────

type SortStrategy interface {
	Sort(data []int) []int
	Name() string
}

// BubbleSort — simple, O(n²). Good for nearly sorted data.
type BubbleSort struct{}

func (BubbleSort) Name() string { return "bubble" }
func (BubbleSort) Sort(data []int) []int {
	out := make([]int, len(data))
	copy(out, data)
	n := len(out)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-1-i; j++ {
			if out[j] > out[j+1] {
				out[j], out[j+1] = out[j+1], out[j]
			}
		}
	}
	return out
}

// StdSort — wraps sort.Ints; O(n log n).
type StdSort struct{}

func (StdSort) Name() string { return "stdlib" }
func (StdSort) Sort(data []int) []int {
	out := make([]int, len(data))
	copy(out, data)
	sort.Ints(out)
	return out
}

// ReverseSort — delegates to StdSort, then reverses.
type ReverseSort struct{}

func (ReverseSort) Name() string { return "reverse" }
func (ReverseSort) Sort(data []int) []int {
	out := StdSort{}.Sort(data)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

type Sorter struct{ strategy SortStrategy }

func (s *Sorter) SetStrategy(st SortStrategy) { s.strategy = st }
func (s *Sorter) Sort(data []int) []int {
	result := s.strategy.Sort(data)
	fmt.Printf("  [%s] %v\n", s.strategy.Name(), result)
	return result
}

// ── Pricing strategy ──────────────────────────────────────────────────────────

type PricingStrategy interface {
	Calculate(basePrice float64, qty int) float64
	Label() string
}

type StandardPricing struct{}

func (StandardPricing) Calculate(base float64, qty int) float64 { return base * float64(qty) }
func (StandardPricing) Label() string                           { return "standard" }

type BulkPricing struct{ Threshold, DiscountPct int }

func (b BulkPricing) Calculate(base float64, qty int) float64 {
	total := base * float64(qty)
	if qty >= b.Threshold {
		total *= (1 - float64(b.DiscountPct)/100)
	}
	return total
}
func (b BulkPricing) Label() string {
	return fmt.Sprintf("bulk-%d%%@%d+", b.DiscountPct, b.Threshold)
}

type MemberPricing struct{ MemberDiscount float64 }

func (m MemberPricing) Calculate(base float64, qty int) float64 {
	return base * float64(qty) * (1 - m.MemberDiscount)
}
func (m MemberPricing) Label() string { return fmt.Sprintf("member-%.0f%%", m.MemberDiscount*100) }

type PriceCalculator struct{ strategy PricingStrategy }

func (c *PriceCalculator) SetStrategy(s PricingStrategy) { c.strategy = s }
func (c *PriceCalculator) Quote(base float64, qty int) {
	total := c.strategy.Calculate(base, qty)
	fmt.Printf("  %-20s  qty=%d  unit=%.2f  total=%.2f\n",
		c.strategy.Label(), qty, base, total)
}

// ─────────────────────────────────────────────────────────────────────────────
// OBSERVER
//
// Defines a one-to-many dependency so when one object changes state,
// all dependents are notified automatically.
// In Go: a registry of Observer values; Notify loops over them.
// ─────────────────────────────────────────────────────────────────────────────

// ── Generic event bus ──────────────────────────────────────────────────────────

type Event struct {
	Type    string
	Payload map[string]any
}

type Observer interface {
	OnEvent(e Event)
}

type EventBus struct {
	observers map[string][]Observer // topic → observers
}

func NewEventBus() *EventBus {
	return &EventBus{observers: make(map[string][]Observer)}
}

func (b *EventBus) Subscribe(topic string, o Observer) {
	b.observers[topic] = append(b.observers[topic], o)
}

func (b *EventBus) Publish(e Event) {
	for _, o := range b.observers[e.Type] {
		o.OnEvent(e)
	}
	for _, o := range b.observers["*"] { // wildcard subscriptions
		o.OnEvent(e)
	}
}

// Concrete observers.

type AuditLogger struct{ name string }

func (a *AuditLogger) OnEvent(e Event) {
	fmt.Printf("  [AUDIT %s] %s %v\n", a.name, e.Type, e.Payload)
}

type MetricsCollector struct{ counts map[string]int }

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{counts: make(map[string]int)}
}

func (m *MetricsCollector) OnEvent(e Event) {
	m.counts[e.Type]++
}

func (m *MetricsCollector) Report() {
	keys := make([]string, 0, len(m.counts))
	for k := range m.counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m.counts[k]))
	}
	fmt.Printf("  [METRICS] %s\n", strings.Join(parts, "  "))
}

type EmailAlerter struct{ threshold int }

func (e *EmailAlerter) OnEvent(ev Event) {
	if val, ok := ev.Payload["cpu"].(int); ok && val > e.threshold {
		fmt.Printf("  [ALERT] CPU %d%% exceeds threshold %d%% — paging oncall\n",
			val, e.threshold)
	}
}

// Stock price observer pattern.
type StockWatcher struct{ name string; threshold float64 }

func (s *StockWatcher) OnEvent(e Event) {
	price, _ := e.Payload["price"].(float64)
	ticker, _ := e.Payload["ticker"].(string)
	if math.Abs(price-s.threshold)/s.threshold > 0.05 {
		dir := "up"
		if price < s.threshold {
			dir = "down"
		}
		fmt.Printf("  [WATCH %s] %s moved %s to %.2f (threshold %.2f)\n",
			s.name, ticker, dir, price, s.threshold)
	}
}

func main() {
	fmt.Println("=== Strategy: sorting ===")
	data := []int{5, 3, 8, 1, 9, 2, 7, 4, 6}
	sorter := &Sorter{strategy: StdSort{}}
	sorter.Sort(data)
	sorter.SetStrategy(BubbleSort{})
	sorter.Sort(data)
	sorter.SetStrategy(ReverseSort{})
	sorter.Sort(data)

	fmt.Println()
	fmt.Println("=== Strategy: pricing ===")
	calc := &PriceCalculator{strategy: StandardPricing{}}
	calc.Quote(9.99, 5)
	calc.SetStrategy(BulkPricing{Threshold: 10, DiscountPct: 15})
	calc.Quote(9.99, 5)
	calc.Quote(9.99, 15)
	calc.SetStrategy(MemberPricing{MemberDiscount: 0.20})
	calc.Quote(9.99, 8)

	fmt.Println()
	fmt.Println("=== Observer: event bus ===")
	bus := NewEventBus()
	audit := &AuditLogger{name: "security"}
	metrics := NewMetricsCollector()
	alerter := &EmailAlerter{threshold: 80}

	bus.Subscribe("user.login", audit)
	bus.Subscribe("user.login", metrics)
	bus.Subscribe("user.logout", metrics)
	bus.Subscribe("system.alert", alerter)
	bus.Subscribe("*", metrics)

	bus.Publish(Event{"user.login", map[string]any{"user": "alice", "ip": "10.0.0.1"}})
	bus.Publish(Event{"user.login", map[string]any{"user": "bob", "ip": "10.0.0.2"}})
	bus.Publish(Event{"user.logout", map[string]any{"user": "alice"}})
	bus.Publish(Event{"system.alert", map[string]any{"cpu": 92}})
	bus.Publish(Event{"system.alert", map[string]any{"cpu": 45}})
	bus.Publish(Event{"user.login", map[string]any{"user": "carol"}})

	metrics.Report()

	fmt.Println()
	fmt.Println("=== Observer: stock watcher ===")
	stockBus := NewEventBus()
	stockBus.Subscribe("stock.tick", &StockWatcher{"Alice", 150.0})
	stockBus.Subscribe("stock.tick", &StockWatcher{"Bob", 200.0})

	prices := []float64{148.0, 152.5, 145.0, 158.0, 210.0, 195.0}
	for _, p := range prices {
		stockBus.Publish(Event{"stock.tick", map[string]any{"ticker": "GOOG", "price": p}})
	}
}
