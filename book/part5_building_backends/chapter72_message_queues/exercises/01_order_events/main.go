// FILE: book/part5_building_backends/chapter72_message_queues/exercises/01_order_events/main.go
// CHAPTER: 72 — Message Queues
// TOPIC: Order lifecycle event system — priority queue, event sourcing, middleware pipeline.
//
// Run (from the chapter folder):
//   go run ./exercises/01_order_events

package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PRIORITY QUEUE
// ─────────────────────────────────────────────────────────────────────────────

type Priority int

const (
	PriorityLow    Priority = 1
	PriorityNormal Priority = 5
	PriorityHigh   Priority = 10
)

type PriorityMessage struct {
	ID        string
	Topic     string
	Payload   any
	Priority  Priority
	Timestamp time.Time
	Attempts  int
}

type PriorityQueue struct {
	mu       sync.Mutex
	items    []*PriorityMessage
	maxRetry int
	dlq      []*PriorityMessage
}

func NewPriorityQueue(maxRetry int) *PriorityQueue {
	return &PriorityQueue{maxRetry: maxRetry}
}

func (pq *PriorityQueue) Enqueue(msg *PriorityMessage) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.items = append(pq.items, msg)
	// Keep sorted descending by priority.
	sort.Slice(pq.items, func(i, j int) bool {
		return pq.items[i].Priority > pq.items[j].Priority
	})
}

// Dequeue returns the highest-priority message, or nil if empty.
func (pq *PriorityQueue) Dequeue() *PriorityMessage {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if len(pq.items) == 0 {
		return nil
	}
	msg := pq.items[0]
	pq.items = pq.items[1:]
	return msg
}

func (pq *PriorityQueue) Nack(msg *PriorityMessage) {
	msg.Attempts++
	if msg.Attempts >= pq.maxRetry {
		pq.mu.Lock()
		pq.dlq = append(pq.dlq, msg)
		pq.mu.Unlock()
		return
	}
	pq.Enqueue(msg)
}

func (pq *PriorityQueue) DLQ() []*PriorityMessage {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	out := make([]*PriorityMessage, len(pq.dlq))
	copy(out, pq.dlq)
	return out
}

func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.items)
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE PIPELINE
// ─────────────────────────────────────────────────────────────────────────────

type Event struct {
	ID        string
	Topic     string
	Payload   any
	Timestamp time.Time
}

type HandlerFunc func(ctx context.Context, e Event) error

type Middleware func(next HandlerFunc) HandlerFunc

func Chain(h HandlerFunc, mws ...Middleware) HandlerFunc {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// LoggingMW prints before/after the handler.
func LoggingMW(next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, e Event) error {
		fmt.Printf("  [log] handling event id=%s topic=%s\n", e.ID, e.Topic)
		err := next(ctx, e)
		if err != nil {
			fmt.Printf("  [log] event id=%s failed: %v\n", e.ID, err)
		}
		return err
	}
}

// RetryMW retries on error up to maxAttempts times with a fixed delay.
func RetryMW(maxAttempts int, delay time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, e Event) error {
			var err error
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				err = next(ctx, e)
				if err == nil {
					return nil
				}
				if attempt < maxAttempts {
					fmt.Printf("  [retry] attempt %d/%d failed, retrying...\n", attempt, maxAttempts)
					time.Sleep(delay)
				}
			}
			return fmt.Errorf("all %d attempts failed: %w", maxAttempts, err)
		}
	}
}

// DeduplicationMW drops events whose ID has already been processed.
type Dedup struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewDedup() *Dedup { return &Dedup{seen: make(map[string]struct{})} }

func (d *Dedup) Middleware(next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, e Event) error {
		d.mu.Lock()
		if _, ok := d.seen[e.ID]; ok {
			d.mu.Unlock()
			fmt.Printf("  [dedup] dropping duplicate event id=%s\n", e.ID)
			return nil
		}
		d.seen[e.ID] = struct{}{}
		d.mu.Unlock()
		return next(ctx, e)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ORDER DOMAIN EVENTS
// ─────────────────────────────────────────────────────────────────────────────

type OrderPlaced struct {
	OrderID    string
	CustomerID string
	Items      int
	Total      int
}

type PaymentFailed struct {
	OrderID string
	Reason  string
}

type OrderFulfilled struct {
	OrderID        string
	TrackingNumber string
}

// ─────────────────────────────────────────────────────────────────────────────
// EVENT STORE (append-only log)
// ─────────────────────────────────────────────────────────────────────────────

type EventStore struct {
	mu     sync.RWMutex
	events []Event
}

func (es *EventStore) Append(e Event) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.events = append(es.events, e)
}

func (es *EventStore) All() []Event {
	es.mu.RLock()
	defer es.mu.RUnlock()
	out := make([]Event, len(es.events))
	copy(out, es.events)
	return out
}

// Replay calls fn for each event whose topic matches (empty = all topics).
func (es *EventStore) Replay(topic string, fn func(Event)) {
	for _, e := range es.All() {
		if topic == "" || e.Topic == topic {
			fn(e)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Order Events Exercise ===")
	fmt.Println()

	ctx := context.Background()

	// ── PRIORITY QUEUE ────────────────────────────────────────────────────────
	fmt.Println("--- Priority Queue ---")
	pq := NewPriorityQueue(3)

	pq.Enqueue(&PriorityMessage{ID: "msg-1", Topic: "orders", Priority: PriorityLow, Payload: "background sync"})
	pq.Enqueue(&PriorityMessage{ID: "msg-2", Topic: "payments", Priority: PriorityHigh, Payload: "fraud alert"})
	pq.Enqueue(&PriorityMessage{ID: "msg-3", Topic: "orders", Priority: PriorityNormal, Payload: "new order"})
	pq.Enqueue(&PriorityMessage{ID: "msg-4", Topic: "payments", Priority: PriorityHigh, Payload: "payment retry"})

	fmt.Printf("  queued %d messages\n", pq.Len())
	for pq.Len() > 0 {
		msg := pq.Dequeue()
		fmt.Printf("  dequeued id=%s priority=%d payload=%v\n", msg.ID, msg.Priority, msg.Payload)
	}

	// ── DEAD LETTER via NACK ─────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Dead Letter Queue (maxRetry=3) ---")
	pq2 := NewPriorityQueue(3)
	bad := &PriorityMessage{ID: "fail-1", Topic: "payments", Priority: PriorityHigh, Payload: "unprocessable"}
	pq2.Enqueue(bad)

	for {
		msg := pq2.Dequeue()
		if msg == nil {
			break
		}
		fmt.Printf("  attempt %d: NACK\n", msg.Attempts+1)
		pq2.Nack(msg)
	}
	for _, d := range pq2.DLQ() {
		fmt.Printf("  DLQ: id=%s attempts=%d\n", d.ID, d.Attempts)
	}

	// ── MIDDLEWARE PIPELINE ───────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Middleware Pipeline ---")

	var callCount atomic.Int32
	base := func(ctx context.Context, e Event) error {
		callCount.Add(1)
		op := e.Payload.(OrderPlaced)
		fmt.Printf("  [fulfillment] processing order %s items=%d total=%d\n",
			op.OrderID, op.Items, op.Total)
		return nil
	}

	dedup := NewDedup()
	handler := Chain(base, LoggingMW, dedup.Middleware)

	events := []Event{
		{ID: "e-1", Topic: "orders.placed", Payload: OrderPlaced{OrderID: "ord-10", CustomerID: "c-1", Items: 3, Total: 2999}, Timestamp: time.Now()},
		{ID: "e-1", Topic: "orders.placed", Payload: OrderPlaced{OrderID: "ord-10", CustomerID: "c-1", Items: 3, Total: 2999}, Timestamp: time.Now()}, // duplicate
		{ID: "e-2", Topic: "orders.placed", Payload: OrderPlaced{OrderID: "ord-11", CustomerID: "c-2", Items: 1, Total: 999}, Timestamp: time.Now()},
	}
	for _, e := range events {
		handler(ctx, e)
	}
	fmt.Printf("  base handler called %d times (1 duplicate dropped)\n", callCount.Load())

	// ── RETRY MIDDLEWARE ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Retry Middleware ---")

	var failCount atomic.Int32
	flakyBase := func(ctx context.Context, e Event) error {
		n := failCount.Add(1)
		if n < 3 {
			return fmt.Errorf("transient error (call %d)", n)
		}
		fmt.Printf("  [payment-svc] payment processed for %s\n", e.ID)
		return nil
	}
	retryHandler := Chain(flakyBase, RetryMW(5, time.Millisecond))
	err := retryHandler(ctx, Event{ID: "pay-evt-1", Topic: "payments.process", Timestamp: time.Now()})
	if err != nil {
		fmt.Printf("  final error: %v\n", err)
	}

	// ── EVENT STORE / REPLAY ─────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Event Store and Replay ---")

	store := &EventStore{}
	recorder := Chain(base, func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, e Event) error {
			store.Append(e)
			return next(ctx, e)
		}
	})

	_ = recorder(ctx, Event{
		ID: "e-10", Topic: "orders.placed",
		Payload:   OrderPlaced{OrderID: "ord-20", CustomerID: "c-5", Items: 2, Total: 1599},
		Timestamp: time.Now(),
	})
	_ = recorder(ctx, Event{
		ID: "e-11", Topic: "orders.placed",
		Payload:   OrderPlaced{OrderID: "ord-21", CustomerID: "c-6", Items: 5, Total: 8999},
		Timestamp: time.Now(),
	})

	fmt.Println()
	fmt.Printf("  stored %d events; replaying orders.placed:\n", len(store.All()))
	store.Replay("orders.placed", func(e Event) {
		op := e.Payload.(OrderPlaced)
		fmt.Printf("  [replay] id=%s order=%s total=%d\n", e.ID, op.OrderID, op.Total)
	})
}
