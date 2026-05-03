// FILE: book/part5_building_backends/chapter72_message_queues/examples/02_pub_sub/main.go
// CHAPTER: 72 — Message Queues
// TOPIC: Pub/Sub event bus — topic subscriptions, fan-out,
//        typed events, ordered and unordered delivery.
//
// Run (from the chapter folder):
//   go run ./examples/02_pub_sub

package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// EVENT
// ─────────────────────────────────────────────────────────────────────────────

type Event struct {
	ID        string
	Topic     string
	Payload   any
	Timestamp time.Time
}

type HandlerFunc func(event Event)

// ─────────────────────────────────────────────────────────────────────────────
// EVENT BUS
// ─────────────────────────────────────────────────────────────────────────────

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]subscription
	nextSubID   int
}

type subscription struct {
	id      int
	handler HandlerFunc
}

func NewEventBus() *EventBus {
	return &EventBus{subscribers: make(map[string][]subscription)}
}

// Subscribe registers a handler for a topic. Returns an unsubscribe function.
func (b *EventBus) Subscribe(topic string, handler HandlerFunc) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextSubID++
	id := b.nextSubID
	b.subscribers[topic] = append(b.subscribers[topic], subscription{id: id, handler: handler})

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subs := b.subscribers[topic]
		for i, s := range subs {
			if s.id == id {
				b.subscribers[topic] = append(subs[:i], subs[i+1:]...)
				return
			}
		}
	}
}

// Publish fans out the event to all subscribers of the topic.
func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	subs := make([]subscription, len(b.subscribers[event.Topic]))
	copy(subs, b.subscribers[event.Topic])
	b.mu.RUnlock()

	for _, s := range subs {
		s.handler(event)
	}
}

// PublishAsync publishes in a separate goroutine for each subscriber.
func (b *EventBus) PublishAsync(ctx context.Context, event Event) {
	b.mu.RLock()
	subs := make([]subscription, len(b.subscribers[event.Topic]))
	copy(subs, b.subscribers[event.Topic])
	b.mu.RUnlock()

	var wg sync.WaitGroup
	for _, s := range subs {
		wg.Add(1)
		go func(h HandlerFunc) {
			defer wg.Done()
			h(event)
		}(s.handler)
	}
	wg.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN EVENTS
// ─────────────────────────────────────────────────────────────────────────────

type OrderCreated struct {
	OrderID    string
	CustomerID string
	Total      int
}

type PaymentProcessed struct {
	OrderID string
	Amount  int
}

type OrderShipped struct {
	OrderID        string
	TrackingNumber string
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Pub/Sub Event Bus ===")
	fmt.Println()

	bus := NewEventBus()
	ctx := context.Background()

	// ── SINGLE SUBSCRIBER ─────────────────────────────────────────────────────
	fmt.Println("--- Single subscriber ---")
	var received []string
	unsub := bus.Subscribe("orders.created", func(e Event) {
		oc := e.Payload.(OrderCreated)
		received = append(received, oc.OrderID)
		fmt.Printf("  [order-service] order created: %s customer=%s total=%d\n",
			oc.OrderID, oc.CustomerID, oc.Total)
	})

	bus.Publish(Event{
		ID:        "evt-1",
		Topic:     "orders.created",
		Payload:   OrderCreated{OrderID: "ord-1", CustomerID: "c-101", Total: 9999},
		Timestamp: time.Now(),
	})
	fmt.Printf("  received: %v\n", received)

	// ── FAN-OUT: multiple subscribers ────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Fan-out: 3 subscribers on orders.created ---")

	var mu sync.Mutex
	calls := map[string]int{}

	services := []struct{ name, role string }{
		{"inventory-svc", "deduct stock"},
		{"email-svc", "send confirmation"},
		{"analytics-svc", "record event"},
	}
	for _, svc := range services {
		name := svc.name
		role := svc.role
		bus.Subscribe("orders.created", func(e Event) {
			oc := e.Payload.(OrderCreated)
			fmt.Printf("  [%s] %s for order %s\n", name, role, oc.OrderID)
			mu.Lock()
			calls[name]++
			mu.Unlock()
		})
	}

	bus.Publish(Event{
		ID: "evt-2", Topic: "orders.created",
		Payload:   OrderCreated{OrderID: "ord-2", CustomerID: "c-102", Total: 4999},
		Timestamp: time.Now(),
	})

	// ── MULTIPLE TOPICS ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Multiple topics ---")

	bus.Subscribe("payments.processed", func(e Event) {
		pp := e.Payload.(PaymentProcessed)
		fmt.Printf("  [fulfillment-svc] payment confirmed for order %s amount=%d\n",
			pp.OrderID, pp.Amount)
	})

	bus.Subscribe("orders.shipped", func(e Event) {
		os_ := e.Payload.(OrderShipped)
		fmt.Printf("  [email-svc] order %s shipped, tracking=%s\n",
			os_.OrderID, os_.TrackingNumber)
	})

	bus.Publish(Event{ID: "evt-3", Topic: "payments.processed",
		Payload: PaymentProcessed{OrderID: "ord-2", Amount: 4999}, Timestamp: time.Now()})
	bus.Publish(Event{ID: "evt-4", Topic: "orders.shipped",
		Payload: OrderShipped{OrderID: "ord-2", TrackingNumber: "1Z999AA1"}, Timestamp: time.Now()})

	// ── UNSUBSCRIBE ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Unsubscribe ---")
	unsub() // remove first subscriber
	fmt.Printf("  unsubscribed original order-service handler\n")
	bus.Publish(Event{ID: "evt-5", Topic: "orders.created",
		Payload:   OrderCreated{OrderID: "ord-3", CustomerID: "c-103", Total: 1999},
		Timestamp: time.Now()})
	fmt.Printf("  (only inventory, email, analytics receive ord-3)\n")

	// ── ASYNC PUBLISH ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Async publish (concurrent handlers) ---")
	asyncBus := NewEventBus()
	var wg sync.WaitGroup
	for i := 1; i <= 3; i++ {
		idx := i
		asyncBus.Subscribe("work", func(e Event) {
			time.Sleep(time.Duration(idx) * time.Millisecond)
			fmt.Printf("  worker-%d done\n", idx)
		})
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		asyncBus.PublishAsync(ctx, Event{ID: "e", Topic: "work", Payload: nil, Timestamp: time.Now()})
	}()
	wg.Wait()

	fmt.Println()
	fmt.Printf("  calls per service: %v\n", calls)
}
