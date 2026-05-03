// FILE: book/part5_building_backends/chapter79_idempotency/examples/02_idempotency_patterns/main.go
// CHAPTER: 79 — Idempotency at the API Boundary
// TOPIC: Transactional outbox pattern, inbox deduplication, saga compensation,
//        and at-most-once vs at-least-once delivery semantics.
//
// Run (from the chapter folder):
//   go run ./examples/02_idempotency_patterns

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// TRANSACTIONAL INBOX — deduplication at the consumer side
// Stores processed event IDs; rejects duplicates.
// ─────────────────────────────────────────────────────────────────────────────

type Inbox struct {
	mu        sync.Mutex
	processed map[string]time.Time // eventID → processedAt
	ttl       time.Duration
	Applied   atomic.Int64
	Rejected  atomic.Int64
}

func NewInbox(ttl time.Duration) *Inbox {
	return &Inbox{processed: make(map[string]time.Time), ttl: ttl}
}

// TryProcess returns true and marks the event if not yet processed.
// Returns false if already processed (duplicate).
func (ib *Inbox) TryProcess(eventID string) bool {
	ib.mu.Lock()
	defer ib.mu.Unlock()
	// Cleanup expired entries.
	now := time.Now()
	for id, at := range ib.processed {
		if now.Sub(at) > ib.ttl {
			delete(ib.processed, id)
		}
	}
	if _, ok := ib.processed[eventID]; ok {
		ib.Rejected.Add(1)
		return false
	}
	ib.processed[eventID] = now
	ib.Applied.Add(1)
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// OUTBOX PATTERN — write event atomically with the business change
// Event is published to the message broker only after being persisted.
// ─────────────────────────────────────────────────────────────────────────────

type OutboxEvent struct {
	ID        string
	Type      string
	Payload   string
	CreatedAt time.Time
	Published bool
}

type Outbox struct {
	mu     sync.Mutex
	events []*OutboxEvent
	seq    atomic.Int64
}

func (o *Outbox) Write(eventType, payload string) *OutboxEvent {
	o.mu.Lock()
	defer o.mu.Unlock()
	evt := &OutboxEvent{
		ID:        fmt.Sprintf("evt-%d", o.seq.Add(1)),
		Type:      eventType,
		Payload:   payload,
		CreatedAt: time.Now(),
	}
	o.events = append(o.events, evt)
	return evt
}

// Unpublished returns events not yet forwarded to the broker.
func (o *Outbox) Unpublished() []*OutboxEvent {
	o.mu.Lock()
	defer o.mu.Unlock()
	var out []*OutboxEvent
	for _, e := range o.events {
		if !e.Published {
			out = append(out, e)
		}
	}
	return out
}

func (o *Outbox) MarkPublished(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, e := range o.events {
		if e.ID == id {
			e.Published = true
			return
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SAGA STEPS — compensatable operations
// ─────────────────────────────────────────────────────────────────────────────

type SagaStep struct {
	Name      string
	Execute   func() error
	Compensate func() error
}

type Saga struct {
	steps     []SagaStep
	completed []int // indices of completed steps
}

func (s *Saga) Add(step SagaStep) {
	s.steps = append(s.steps, step)
}

func (s *Saga) Run() error {
	for i, step := range s.steps {
		fmt.Printf("  [saga] executing step: %s\n", step.Name)
		if err := step.Execute(); err != nil {
			fmt.Printf("  [saga] step %s failed: %v — rolling back\n", step.Name, err)
			// Compensate in reverse order.
			for j := len(s.completed) - 1; j >= 0; j-- {
				idx := s.completed[j]
				comp := s.steps[idx]
				fmt.Printf("  [saga] compensating: %s\n", comp.Name)
				comp.Compensate()
			}
			return fmt.Errorf("saga failed at step %q: %w", step.Name, err)
		}
		s.completed = append(s.completed, i)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// AT-LEAST-ONCE vs AT-MOST-ONCE DEMO
// ─────────────────────────────────────────────────────────────────────────────

type DeliveryMode string

const (
	AtMostOnce  DeliveryMode = "at-most-once"
	AtLeastOnce DeliveryMode = "at-least-once"
	ExactlyOnce DeliveryMode = "exactly-once"
)

type DeliverySimulator struct {
	inbox     *Inbox
	processed atomic.Int64
	dropped   atomic.Int64
}

func (ds *DeliverySimulator) Process(mode DeliveryMode, eventID string, action func()) {
	switch mode {
	case AtMostOnce:
		// Fire and forget — may lose the event, never duplicates.
		action()
		ds.processed.Add(1)
	case AtLeastOnce:
		// Retry on failure — may duplicate; consumer must be idempotent.
		action()
		ds.processed.Add(1)
	case ExactlyOnce:
		// At-least-once + inbox dedup.
		if ds.inbox.TryProcess(eventID) {
			action()
			ds.processed.Add(1)
		} else {
			ds.dropped.Add(1)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Idempotency Patterns ===")
	fmt.Println()

	// ── INBOX DEDUPLICATION ───────────────────────────────────────────────────
	fmt.Println("--- Inbox deduplication ---")
	inbox := NewInbox(time.Minute)

	events := []string{"evt-1", "evt-2", "evt-1", "evt-3", "evt-2", "evt-3"}
	var processedCount, duplicateCount int
	for _, id := range events {
		if inbox.TryProcess(id) {
			processedCount++
			fmt.Printf("  processed %s\n", id)
		} else {
			duplicateCount++
			fmt.Printf("  duplicate %s — skipped\n", id)
		}
	}
	fmt.Printf("  applied=%d rejected=%d\n", inbox.Applied.Load(), inbox.Rejected.Load())

	// ── OUTBOX PATTERN ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Outbox pattern ---")
	outbox := &Outbox{}

	// Simulate: DB write + outbox write happen together (atomically in production).
	// Relay process polls unpublished outbox entries and publishes them.
	outbox.Write("order.placed", `{"orderID":"ord-1","total":9999}`)
	outbox.Write("payment.processed", `{"orderID":"ord-1","amount":9999}`)
	outbox.Write("order.shipped", `{"orderID":"ord-1","tracking":"1Z999"}`)

	fmt.Printf("  %d events in outbox\n", len(outbox.Unpublished()))

	// Relay publishes events.
	for _, evt := range outbox.Unpublished() {
		fmt.Printf("  [relay] publishing %s: %s → %s\n", evt.ID, evt.Type, evt.Payload)
		outbox.MarkPublished(evt.ID)
	}
	fmt.Printf("  unpublished after relay: %d\n", len(outbox.Unpublished()))

	// Simulate relay crash and re-run (outbox already published — no duplicates).
	fmt.Printf("  relay re-run: %d to publish (already done)\n", len(outbox.Unpublished()))

	// ── SAGA PATTERN ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Saga: order fulfillment (step 3 fails → compensate) ---")

	var reserved, charged, notified bool
	saga := &Saga{}

	saga.Add(SagaStep{
		Name:    "reserve-inventory",
		Execute: func() error { reserved = true; fmt.Println("  inventory reserved"); return nil },
		Compensate: func() error {
			reserved = false
			fmt.Println("  inventory reservation cancelled")
			return nil
		},
	})
	saga.Add(SagaStep{
		Name:    "charge-card",
		Execute: func() error { charged = true; fmt.Println("  card charged"); return nil },
		Compensate: func() error {
			charged = false
			fmt.Println("  card charge refunded")
			return nil
		},
	})
	saga.Add(SagaStep{
		Name:    "send-notification",
		Execute: func() error { return fmt.Errorf("email service down") },
		Compensate: func() error {
			notified = false
			return nil
		},
	})

	err := saga.Run()
	fmt.Printf("  saga result: err=%v\n", err)
	fmt.Printf("  state: reserved=%v charged=%v notified=%v\n", reserved, charged, notified)

	// ── DELIVERY SEMANTICS ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Delivery semantics comparison ---")

	dedupInbox := NewInbox(time.Minute)
	ds := &DeliverySimulator{inbox: dedupInbox}

	var actions int
	action := func() { actions++ }

	// Simulate event delivered twice (at-least-once).
	fmt.Println("  at-least-once (no dedup): event delivered twice → 2 actions")
	for range 2 {
		ds.Process(AtLeastOnce, "e-1", action)
	}

	// Exactly-once: same event twice → only 1 action.
	actions = 0
	fmt.Println("  exactly-once (inbox dedup): event delivered twice → 1 action")
	ds2 := &DeliverySimulator{inbox: NewInbox(time.Minute)}
	for range 2 {
		ds2.Process(ExactlyOnce, "e-2", action)
	}
	fmt.Printf("  exactly-once actions=%d dropped=%d\n", ds2.processed.Load(), ds2.dropped.Load())

	fmt.Println()
	fmt.Println("--- Summary ---")
	fmt.Println(`  at-most-once:  no retry; may lose events; use for metrics/logs
  at-least-once: retry on failure; may duplicate; consumer must be idempotent
  exactly-once:  at-least-once + inbox dedup; strongest guarantee; highest cost`)
}
