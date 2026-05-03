// FILE: book/part5_building_backends/chapter80_event_driven/exercises/01_order_saga/main.go
// CHAPTER: 80 — Event-Driven Architecture in Go
// TOPIC: Choreography-based saga — each service reacts to events and emits new
//        events; no central orchestrator. Includes compensation on failure.
//
// Run (from the chapter folder):
//   go run ./exercises/01_order_saga

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN EVENTS
// ─────────────────────────────────────────────────────────────────────────────

type Event struct {
	ID          int64
	Type        string
	AggregateID string
	Payload     any
	OccurredAt  time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// EVENT BUS (in-process, sync)
// ─────────────────────────────────────────────────────────────────────────────

type HandlerFunc func(ctx context.Context, e *Event) error

type EventBus struct {
	mu       sync.RWMutex
	handlers map[string][]HandlerFunc
	Errors   atomic.Int64
	seq      atomic.Int64
}

func NewEventBus() *EventBus {
	return &EventBus{handlers: make(map[string][]HandlerFunc)}
}

func (b *EventBus) Subscribe(eventType string, h HandlerFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], h)
}

func (b *EventBus) Publish(ctx context.Context, eventType, aggregateID string, payload any) *Event {
	e := &Event{
		ID:          b.seq.Add(1),
		Type:        eventType,
		AggregateID: aggregateID,
		Payload:     payload,
		OccurredAt:  time.Now(),
	}
	b.mu.RLock()
	handlers := make([]HandlerFunc, len(b.handlers[eventType]))
	copy(handlers, b.handlers[eventType])
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, e); err != nil {
			b.Errors.Add(1)
			fmt.Printf("  [bus] error handling %s: %v\n", eventType, err)
		}
	}
	return e
}

// ─────────────────────────────────────────────────────────────────────────────
// ORDER EVENT PAYLOADS
// ─────────────────────────────────────────────────────────────────────────────

type OrderPlacedPayload struct {
	OrderID    string
	CustomerID string
	Items      int
	Total      int
}

type InventoryReservedPayload struct {
	OrderID string
	Items   int
}

type InventoryFailedPayload struct {
	OrderID string
	Reason  string
}

type PaymentChargedPayload struct {
	OrderID   string
	PaymentID string
	Amount    int
}

type PaymentFailedPayload struct {
	OrderID string
	Reason  string
}

type OrderConfirmedPayload struct{ OrderID string }
type OrderCancelledPayload struct {
	OrderID string
	Reason  string
}

// ─────────────────────────────────────────────────────────────────────────────
// INVENTORY SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type InventoryService struct {
	mu       sync.Mutex
	stock    map[string]int
	reserved map[string]int // orderID → quantity reserved
	bus      *EventBus
}

func NewInventoryService(bus *EventBus) *InventoryService {
	svc := &InventoryService{
		stock:    map[string]int{"item-A": 10, "item-B": 0},
		reserved: make(map[string]int),
		bus:      bus,
	}
	bus.Subscribe("order.placed", svc.onOrderPlaced)
	bus.Subscribe("order.cancelled", svc.onOrderCancelled)
	return svc
}

func (s *InventoryService) onOrderPlaced(ctx context.Context, e *Event) error {
	p := e.Payload.(OrderPlacedPayload)
	s.mu.Lock()
	var reserved bool
	if s.stock["item-A"] >= p.Items {
		s.stock["item-A"] -= p.Items
		s.reserved[p.OrderID] = p.Items
		reserved = true
	}
	s.mu.Unlock()

	if reserved {
		fmt.Printf("  [inventory] reserved %d items for %s\n", p.Items, p.OrderID)
		s.bus.Publish(ctx, "inventory.reserved", p.OrderID, InventoryReservedPayload{
			OrderID: p.OrderID, Items: p.Items,
		})
	} else {
		fmt.Printf("  [inventory] insufficient stock for %s\n", p.OrderID)
		s.bus.Publish(ctx, "inventory.failed", p.OrderID, InventoryFailedPayload{
			OrderID: p.OrderID, Reason: "insufficient stock",
		})
	}
	return nil
}

func (s *InventoryService) onOrderCancelled(ctx context.Context, e *Event) error {
	p := e.Payload.(OrderCancelledPayload)
	s.mu.Lock()
	qty, ok := s.reserved[p.OrderID]
	if ok {
		s.stock["item-A"] += qty
		delete(s.reserved, p.OrderID)
	}
	s.mu.Unlock()
	if ok {
		fmt.Printf("  [inventory] released %d items for %s (compensation)\n", qty, p.OrderID)
	}
	return nil
}

func (s *InventoryService) Stock() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stock["item-A"]
}

// ─────────────────────────────────────────────────────────────────────────────
// PAYMENT SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type PaymentService struct {
	mu         sync.Mutex
	payments   map[string]string // orderID → paymentID
	failOrders map[string]bool   // orders that should fail payment
	seq        atomic.Int64
	bus        *EventBus
}

func NewPaymentService(bus *EventBus, failOrders ...string) *PaymentService {
	fails := make(map[string]bool)
	for _, o := range failOrders {
		fails[o] = true
	}
	svc := &PaymentService{
		payments:   make(map[string]string),
		failOrders: fails,
		bus:        bus,
	}
	bus.Subscribe("inventory.reserved", svc.onInventoryReserved)
	return svc
}

func (s *PaymentService) onInventoryReserved(ctx context.Context, e *Event) error {
	p := e.Payload.(InventoryReservedPayload)
	s.mu.Lock()
	shouldFail := s.failOrders[p.OrderID]
	var payID string
	if !shouldFail {
		payID = fmt.Sprintf("pay-%d", s.seq.Add(1))
		s.payments[p.OrderID] = payID
	}
	s.mu.Unlock()

	if shouldFail {
		fmt.Printf("  [payment] charge failed for %s (simulated)\n", p.OrderID)
		s.bus.Publish(ctx, "payment.failed", p.OrderID, PaymentFailedPayload{
			OrderID: p.OrderID, Reason: "card declined",
		})
	} else {
		fmt.Printf("  [payment] charged %s → %s\n", p.OrderID, payID)
		s.bus.Publish(ctx, "payment.charged", p.OrderID, PaymentChargedPayload{
			OrderID: p.OrderID, PaymentID: payID, Amount: 9999,
		})
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ORDER SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type OrderStatus string

const (
	StatusPlaced    OrderStatus = "placed"
	StatusConfirmed OrderStatus = "confirmed"
	StatusCancelled OrderStatus = "cancelled"
)

type OrderState struct {
	ID     string
	Status OrderStatus
	Reason string
}

type OrderService struct {
	mu     sync.Mutex
	orders map[string]*OrderState
	bus    *EventBus
}

func NewOrderService(bus *EventBus) *OrderService {
	svc := &OrderService{
		orders: make(map[string]*OrderState),
		bus:    bus,
	}
	bus.Subscribe("payment.charged", svc.onPaymentCharged)
	bus.Subscribe("inventory.failed", svc.onFailed)
	bus.Subscribe("payment.failed", svc.onFailed)
	return svc
}

func (s *OrderService) PlaceOrder(ctx context.Context, orderID, customerID string, items, total int) {
	s.mu.Lock()
	s.orders[orderID] = &OrderState{ID: orderID, Status: StatusPlaced}
	s.mu.Unlock()
	fmt.Printf("  [order] placed %s\n", orderID)
	s.bus.Publish(ctx, "order.placed", orderID, OrderPlacedPayload{
		OrderID: orderID, CustomerID: customerID, Items: items, Total: total,
	})
}

func (s *OrderService) onPaymentCharged(ctx context.Context, e *Event) error {
	p := e.Payload.(PaymentChargedPayload)
	s.mu.Lock()
	if o, ok := s.orders[p.OrderID]; ok {
		o.Status = StatusConfirmed
	}
	s.mu.Unlock()
	// Publish outside the lock to avoid re-entrant deadlock via bus callbacks.
	fmt.Printf("  [order] confirmed %s\n", p.OrderID)
	s.bus.Publish(ctx, "order.confirmed", p.OrderID, OrderConfirmedPayload{OrderID: p.OrderID})
	return nil
}

func (s *OrderService) onFailed(ctx context.Context, e *Event) error {
	var orderID, reason string
	switch p := e.Payload.(type) {
	case InventoryFailedPayload:
		orderID, reason = p.OrderID, p.Reason
	case PaymentFailedPayload:
		orderID, reason = p.OrderID, p.Reason
	}
	s.mu.Lock()
	if o, ok := s.orders[orderID]; ok {
		o.Status = StatusCancelled
		o.Reason = reason
	}
	s.mu.Unlock()
	// Publish outside the lock.
	fmt.Printf("  [order] cancelled %s (%s)\n", orderID, reason)
	s.bus.Publish(ctx, "order.cancelled", orderID, OrderCancelledPayload{
		OrderID: orderID, Reason: reason,
	})
	return nil
}

func (s *OrderService) State(orderID string) *OrderState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.orders[orderID]
}

// ─────────────────────────────────────────────────────────────────────────────
// NOTIFICATION SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type NotificationService struct {
	Sent atomic.Int64
}

func NewNotificationService(bus *EventBus) *NotificationService {
	ns := &NotificationService{}
	bus.Subscribe("order.confirmed", func(ctx context.Context, e *Event) error {
		ns.Sent.Add(1)
		fmt.Printf("  [notify] confirmation email for %s\n", e.AggregateID)
		return nil
	})
	bus.Subscribe("order.cancelled", func(ctx context.Context, e *Event) error {
		ns.Sent.Add(1)
		fmt.Printf("  [notify] cancellation notice for %s\n", e.AggregateID)
		return nil
	})
	return ns
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Choreography-Based Saga ===")
	fmt.Println()
	ctx := context.Background()

	// ── HAPPY PATH ────────────────────────────────────────────────────────────
	fmt.Println("--- Happy path: inventory reserved + payment charged ---")

	bus1 := NewEventBus()
	inv1 := NewInventoryService(bus1)
	NewPaymentService(bus1)
	orders1 := NewOrderService(bus1)
	notify1 := NewNotificationService(bus1)

	orders1.PlaceOrder(ctx, "ord-1", "cust-1", 2, 9999)

	fmt.Println()
	state1 := orders1.State("ord-1")
	fmt.Printf("  order status: %s\n", state1.Status)
	fmt.Printf("  inventory remaining: %d\n", inv1.Stock())
	fmt.Printf("  notifications sent: %d\n", notify1.Sent.Load())

	// ── COMPENSATION: inventory ok but payment fails ───────────────────────────
	fmt.Println()
	fmt.Println("--- Compensation: payment fails → inventory released ---")

	bus2 := NewEventBus()
	inv2 := NewInventoryService(bus2)
	NewPaymentService(bus2, "ord-2") // ord-2 will fail payment
	orders2 := NewOrderService(bus2)
	notify2 := NewNotificationService(bus2)

	orders2.PlaceOrder(ctx, "ord-2", "cust-2", 3, 14999)

	fmt.Println()
	state2 := orders2.State("ord-2")
	fmt.Printf("  order status: %s (reason: %s)\n", state2.Status, state2.Reason)
	fmt.Printf("  inventory remaining: %d (restored to 10)\n", inv2.Stock())
	fmt.Printf("  notifications sent: %d (cancellation notice)\n", notify2.Sent.Load())

	// ── INVENTORY FAILURE ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Inventory failure: stock = 0 ---")

	bus3 := NewEventBus()
	NewInventoryService(bus3) // item-B stock = 0
	NewPaymentService(bus3)
	orders3 := NewOrderService(bus3)

	// Deplete item-A first, then try another order.
	orders3.PlaceOrder(ctx, "ord-3", "cust-3", 15, 5000) // 15 > 10 stock

	fmt.Println()
	state3 := orders3.State("ord-3")
	fmt.Printf("  order status: %s (reason: %s)\n", state3.Status, state3.Reason)

	// ── CHOREOGRAPHY SUMMARY ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Choreography vs Orchestration ---")
	fmt.Println(`  Choreography (this example):
    - Each service listens for events and reacts independently.
    - No central coordinator; services are loosely coupled.
    - Compensation triggered by failure events (e.g. payment.failed → cancel → release).
    - Harder to trace: the flow is implicit in subscriptions.

  Orchestration (alternative):
    - A saga orchestrator drives the workflow explicitly.
    - Calls each step in sequence; knows the full flow.
    - Easier to trace; tighter coupling to orchestrator.
    - Best when the flow is complex with many branches.`)
}
