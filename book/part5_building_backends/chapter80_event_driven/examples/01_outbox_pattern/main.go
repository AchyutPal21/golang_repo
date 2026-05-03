// FILE: book/part5_building_backends/chapter80_event_driven/examples/01_outbox_pattern/main.go
// CHAPTER: 80 — Event-Driven Architecture in Go
// TOPIC: Domain events, transactional outbox, event relay, event handler
//        registration, and event versioning.
//
// Run (from the chapter folder):
//   go run ./examples/01_outbox_pattern

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN EVENT
// ─────────────────────────────────────────────────────────────────────────────

type DomainEvent struct {
	ID          string
	AggregateID string
	Type        string
	Version     int
	Payload     any
	OccurredAt  time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// OUTBOX — durable event persistence before publishing
// ─────────────────────────────────────────────────────────────────────────────

type OutboxEntry struct {
	EventID     string
	EventType   string
	AggregateID string
	Payload     string
	OccurredAt  time.Time
	PublishedAt *time.Time
}

type Outbox struct {
	mu      sync.Mutex
	entries []*OutboxEntry
	seq     atomic.Int64
}

func (o *Outbox) Append(eventType, aggregateID, payload string) *OutboxEntry {
	o.mu.Lock()
	defer o.mu.Unlock()
	e := &OutboxEntry{
		EventID:     fmt.Sprintf("evt-%d", o.seq.Add(1)),
		EventType:   eventType,
		AggregateID: aggregateID,
		Payload:     payload,
		OccurredAt:  time.Now(),
	}
	o.entries = append(o.entries, e)
	return e
}

func (o *Outbox) Pending() []*OutboxEntry {
	o.mu.Lock()
	defer o.mu.Unlock()
	var out []*OutboxEntry
	for _, e := range o.entries {
		if e.PublishedAt == nil {
			out = append(out, e)
		}
	}
	return out
}

func (o *Outbox) MarkPublished(eventID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	now := time.Now()
	for _, e := range o.entries {
		if e.EventID == eventID {
			e.PublishedAt = &now
			return
		}
	}
}

func (o *Outbox) All() []*OutboxEntry {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]*OutboxEntry, len(o.entries))
	copy(out, o.entries)
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// EVENT BUS (in-process)
// ─────────────────────────────────────────────────────────────────────────────

type EventHandlerFunc func(ctx context.Context, event *OutboxEntry) error

type EventBus struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandlerFunc
	Errors   atomic.Int64
}

func NewEventBus() *EventBus {
	return &EventBus{handlers: make(map[string][]EventHandlerFunc)}
}

func (b *EventBus) Subscribe(eventType string, h EventHandlerFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], h)
}

func (b *EventBus) Publish(ctx context.Context, entry *OutboxEntry) {
	b.mu.RLock()
	handlers := make([]EventHandlerFunc, len(b.handlers[entry.EventType]))
	copy(handlers, b.handlers[entry.EventType])
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, entry); err != nil {
			b.Errors.Add(1)
			fmt.Printf("  [bus] handler error for %s: %v\n", entry.EventType, err)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ORDER AGGREGATE
// ─────────────────────────────────────────────────────────────────────────────

type OrderStatus string

const (
	OrderPlaced    OrderStatus = "placed"
	OrderConfirmed OrderStatus = "confirmed"
	OrderShipped   OrderStatus = "shipped"
	OrderDelivered OrderStatus = "delivered"
)

type Order struct {
	ID         string
	CustomerID string
	Items      int
	Total      int
	Status     OrderStatus
	outbox     *Outbox // shared outbox
}

func (o *Order) Place() {
	o.Status = OrderPlaced
	o.outbox.Append("order.placed",
		o.ID,
		fmt.Sprintf(`{"orderID":%q,"customerID":%q,"items":%d,"total":%d}`,
			o.ID, o.CustomerID, o.Items, o.Total),
	)
}

func (o *Order) Confirm() {
	o.Status = OrderConfirmed
	o.outbox.Append("order.confirmed", o.ID,
		fmt.Sprintf(`{"orderID":%q}`, o.ID))
}

func (o *Order) Ship(tracking string) {
	o.Status = OrderShipped
	o.outbox.Append("order.shipped", o.ID,
		fmt.Sprintf(`{"orderID":%q,"tracking":%q}`, o.ID, tracking))
}

// ─────────────────────────────────────────────────────────────────────────────
// RELAY — polls outbox and publishes pending events
// ─────────────────────────────────────────────────────────────────────────────

type Relay struct {
	outbox *Outbox
	bus    *EventBus
}

func (r *Relay) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Millisecond):
			for _, entry := range r.outbox.Pending() {
				r.bus.Publish(ctx, entry)
				r.outbox.MarkPublished(entry.EventID)
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Outbox Pattern ===")
	fmt.Println()

	outbox := &Outbox{}
	bus := NewEventBus()

	// Subscribe handlers.
	var inventoryUpdates, emailsSent, analyticsRecorded atomic.Int32

	bus.Subscribe("order.placed", func(ctx context.Context, e *OutboxEntry) error {
		inventoryUpdates.Add(1)
		fmt.Printf("  [inventory] reserve stock for %s\n", e.AggregateID)
		return nil
	})
	bus.Subscribe("order.placed", func(ctx context.Context, e *OutboxEntry) error {
		emailsSent.Add(1)
		fmt.Printf("  [email] send confirmation for %s\n", e.AggregateID)
		return nil
	})
	bus.Subscribe("order.placed", func(ctx context.Context, e *OutboxEntry) error {
		analyticsRecorded.Add(1)
		fmt.Printf("  [analytics] record order.placed %s\n", e.AggregateID)
		return nil
	})
	bus.Subscribe("order.confirmed", func(ctx context.Context, e *OutboxEntry) error {
		fmt.Printf("  [fulfillment] start fulfillment for %s\n", e.AggregateID)
		return nil
	})
	bus.Subscribe("order.shipped", func(ctx context.Context, e *OutboxEntry) error {
		fmt.Printf("  [email] shipping notification for %s\n", e.AggregateID)
		return nil
	})

	// ── ORDER LIFECYCLE ───────────────────────────────────────────────────────
	fmt.Println("--- Order lifecycle via outbox ---")

	order := &Order{ID: "ord-1", CustomerID: "cust-1", Items: 3, Total: 14999, outbox: outbox}
	order.Place()
	order.Confirm()
	order.Ship("1Z999AA1")

	fmt.Printf("  %d events in outbox, 0 published yet\n", len(outbox.All()))

	// ── RELAY ─────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Relay publishes pending events ---")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	relay := &Relay{outbox: outbox, bus: bus}
	go relay.Run(ctx)

	<-ctx.Done()
	time.Sleep(10 * time.Millisecond) // allow last relay tick

	fmt.Println()
	pending := outbox.Pending()
	fmt.Printf("  pending after relay: %d\n", len(pending))
	fmt.Printf("  inventory updates: %d\n", inventoryUpdates.Load())
	fmt.Printf("  emails sent: %d\n", emailsSent.Load())
	fmt.Printf("  analytics: %d\n", analyticsRecorded.Load())

	// ── EVENT VERSIONING ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Event versioning ---")
	fmt.Println(`  Problem: event schema evolves; old consumers must still work.

  Strategy 1 — additive changes (safe):
    v1: {"orderID":"ord-1","total":9999}
    v2: {"orderID":"ord-1","total":9999,"currency":"USD"}  // new optional field
    Consumers ignore unknown fields → backwards compatible.

  Strategy 2 — version field in event:
    {"version":2,"orderID":"ord-1","total":9999}
    Consumers dispatch on version; old consumers handle v1, new ones handle v2.

  Strategy 3 — schema registry (production):
    Event schema stored in Confluent Schema Registry or a simple DB table.
    Producer validates before publishing; consumer validates on receive.`)

	// ── IDEMPOTENT HANDLER ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Idempotent event handler ---")
	seen := make(map[string]bool)
	var mu sync.Mutex
	dedup := func(ctx context.Context, e *OutboxEntry) error {
		mu.Lock()
		defer mu.Unlock()
		if seen[e.EventID] {
			fmt.Printf("  [dedup] duplicate event %s skipped\n", e.EventID)
			return nil
		}
		seen[e.EventID] = true
		fmt.Printf("  [dedup] processed %s\n", e.EventID)
		return nil
	}

	bus2 := NewEventBus()
	bus2.Subscribe("order.placed", dedup)

	entry := outbox.All()[0] // ord-1 placed event
	bus2.Publish(context.Background(), entry)
	bus2.Publish(context.Background(), entry) // duplicate
	bus2.Publish(context.Background(), entry) // duplicate
}
