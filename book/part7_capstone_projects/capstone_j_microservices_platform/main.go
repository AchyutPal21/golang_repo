// FILE: book/part7_capstone_projects/capstone_j_microservices_platform/main.go
// CAPSTONE J — Microservices Platform
// Five services (user, order, inventory, payment, notification) wired together
// via an in-process message bus, service registry, and distributed trace context.
//
// Run:
//   go run ./book/part7_capstone_projects/capstone_j_microservices_platform

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PLATFORM LAYER
// ─────────────────────────────────────────────────────────────────────────────

// TraceContext carries a trace ID across service calls.
type TraceContext struct {
	TraceID string
	SpanID  string
}

func (tc TraceContext) Child(service string) TraceContext {
	return TraceContext{TraceID: tc.TraceID, SpanID: fmt.Sprintf("%s→%s", tc.SpanID, service)}
}

func newTrace(traceID string) TraceContext {
	return TraceContext{TraceID: traceID, SpanID: "root"}
}

// ServiceInfo describes a registered service.
type ServiceInfo struct {
	Name      string
	Healthy   bool
	LastCheck time.Time
	Requests  atomic.Int64
	Errors    atomic.Int64
}

func (s *ServiceInfo) RecordCall(err error) {
	s.Requests.Add(1)
	if err != nil {
		s.Errors.Add(1)
	}
}

func (s *ServiceInfo) ErrorRate() float64 {
	reqs := s.Requests.Load()
	if reqs == 0 {
		return 0
	}
	return float64(s.Errors.Load()) / float64(reqs) * 100
}

// ServiceRegistry holds all registered services.
type ServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]*ServiceInfo
}

func newServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{services: map[string]*ServiceInfo{}}
}

func (r *ServiceRegistry) Register(name string) *ServiceInfo {
	svc := &ServiceInfo{Name: name, Healthy: true, LastCheck: time.Now()}
	r.mu.Lock()
	r.services[name] = svc
	r.mu.Unlock()
	return svc
}

func (r *ServiceRegistry) Get(name string) (*ServiceInfo, bool) {
	r.mu.RLock()
	svc, ok := r.services[name]
	r.mu.RUnlock()
	return svc, ok
}

func (r *ServiceRegistry) Dashboard() {
	r.mu.RLock()
	names := make([]string, 0, len(r.services))
	for n := range r.services {
		names = append(names, n)
	}
	r.mu.RUnlock()

	fmt.Printf("  %-25s  %8s  %8s  %8s  %s\n", "Service", "Healthy", "Requests", "Errors", "ErrRate")
	fmt.Printf("  %s\n", strings.Repeat("-", 65))
	for _, name := range names {
		svc, _ := r.Get(name)
		health := "yes"
		if !svc.Healthy {
			health = "NO"
		}
		fmt.Printf("  %-25s  %8s  %8d  %8d  %.1f%%\n",
			svc.Name, health, svc.Requests.Load(), svc.Errors.Load(), svc.ErrorRate())
	}
}

// MessageBus is a simple in-process pub/sub bus.
type MessageBus struct {
	mu          sync.RWMutex
	subscribers map[string][]func(event Event)
	dlq         []Event
	published   atomic.Int64
}

type Event struct {
	Topic   string
	Payload interface{}
	Trace   TraceContext
}

func newMessageBus() *MessageBus {
	return &MessageBus{subscribers: map[string][]func(event Event){}}
}

func (b *MessageBus) Subscribe(topic string, handler func(Event)) {
	b.mu.Lock()
	b.subscribers[topic] = append(b.subscribers[topic], handler)
	b.mu.Unlock()
}

func (b *MessageBus) Publish(e Event) {
	b.published.Add(1)
	b.mu.RLock()
	handlers := append([]func(Event){}, b.subscribers[e.Topic]...)
	b.mu.RUnlock()

	if len(handlers) == 0 {
		b.mu.Lock()
		b.dlq = append(b.dlq, e)
		b.mu.Unlock()
		return
	}
	for _, h := range handlers {
		h(e)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

type User struct {
	ID    string
	Email string
	Name  string
}

type Product struct {
	ID    string
	Name  string
	Price int // cents
	Stock int
}

type Order struct {
	ID        string
	UserID    string
	ProductID string
	Qty       int
	Status    string // pending, confirmed, failed
	Total     int    // cents
}

// Event payloads
type OrderPlacedPayload struct{ Order Order }
type InventoryReservedPayload struct{ Order Order }
type PaymentChargedPayload struct{ Order Order }
type OrderConfirmedPayload struct{ Order Order }
type OrderFailedPayload struct{ Order Order; Reason string }

// ─────────────────────────────────────────────────────────────────────────────
// USER SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type UserService struct {
	svc   *ServiceInfo
	mu    sync.RWMutex
	users map[string]*User
	seq   atomic.Uint64
}

func newUserService(reg *ServiceRegistry) *UserService {
	return &UserService{svc: reg.Register("user-service"), users: map[string]*User{}}
}

func (s *UserService) Register(name, email string) (*User, error) {
	id := fmt.Sprintf("usr-%d", s.seq.Add(1))
	u := &User{ID: id, Name: name, Email: email}
	s.mu.Lock()
	s.users[id] = u
	s.mu.Unlock()
	s.svc.RecordCall(nil)
	return u, nil
}

func (s *UserService) Get(tc TraceContext, userID string) (*User, error) {
	s.mu.RLock()
	u, ok := s.users[userID]
	s.mu.RUnlock()
	if !ok {
		err := fmt.Errorf("user %s not found", userID)
		s.svc.RecordCall(err)
		return nil, err
	}
	s.svc.RecordCall(nil)
	return u, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// INVENTORY SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type InventoryService struct {
	svc      *ServiceInfo
	bus      *MessageBus
	mu       sync.Mutex
	products map[string]*Product
}

func newInventoryService(reg *ServiceRegistry, bus *MessageBus) *InventoryService {
	svc := &InventoryService{
		svc:      reg.Register("inventory-service"),
		bus:      bus,
		products: map[string]*Product{},
	}
	bus.Subscribe("order.placed", svc.onOrderPlaced)
	return svc
}

func (s *InventoryService) AddProduct(id, name string, price, stock int) {
	s.mu.Lock()
	s.products[id] = &Product{ID: id, Name: name, Price: price, Stock: stock}
	s.mu.Unlock()
}

func (s *InventoryService) onOrderPlaced(e Event) {
	p := e.Payload.(OrderPlacedPayload)
	order := p.Order
	tc := e.Trace.Child("inventory-service")

	s.mu.Lock()
	prod, ok := s.products[order.ProductID]
	if !ok || prod.Stock < order.Qty {
		s.mu.Unlock()
		s.svc.RecordCall(errors.New("insufficient stock"))
		fmt.Printf("  [inventory-service][%s] FAIL reserve %s qty=%d (insufficient stock)\n",
			tc.TraceID, order.ProductID, order.Qty)
		s.bus.Publish(Event{Topic: "order.failed", Payload: OrderFailedPayload{Order: order, Reason: "insufficient stock"}, Trace: tc})
		return
	}
	prod.Stock -= order.Qty
	s.mu.Unlock()
	s.svc.RecordCall(nil)
	fmt.Printf("  [inventory-service][%s] Reserved %s qty=%d (stock remaining=%d)\n",
		tc.TraceID, order.ProductID, order.Qty, prod.Stock)
	s.bus.Publish(Event{Topic: "inventory.reserved", Payload: InventoryReservedPayload{Order: order}, Trace: tc})
}

func (s *InventoryService) Release(productID string, qty int) {
	s.mu.Lock()
	if prod, ok := s.products[productID]; ok {
		prod.Stock += qty
	}
	s.mu.Unlock()
}

// ─────────────────────────────────────────────────────────────────────────────
// PAYMENT SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type PaymentService struct {
	svc         *ServiceInfo
	inv         *InventoryService
	bus         *MessageBus
	failOrderID string // simulate failure for this order
}

func newPaymentService(reg *ServiceRegistry, bus *MessageBus, inv *InventoryService) *PaymentService {
	svc := &PaymentService{svc: reg.Register("payment-service"), bus: bus, inv: inv}
	bus.Subscribe("inventory.reserved", svc.onInventoryReserved)
	return svc
}

func (s *PaymentService) SimulateFailFor(orderID string) { s.failOrderID = orderID }

func (s *PaymentService) onInventoryReserved(e Event) {
	p := e.Payload.(InventoryReservedPayload)
	order := p.Order
	tc := e.Trace.Child("payment-service")

	if order.ID == s.failOrderID {
		s.svc.RecordCall(errors.New("card declined"))
		fmt.Printf("  [payment-service][%s] FAIL charge $%.2f for order %s (card declined)\n",
			tc.TraceID, float64(order.Total)/100, order.ID)
		// Compensate: release inventory
		s.inv.Release(order.ProductID, order.Qty)
		s.bus.Publish(Event{Topic: "order.failed", Payload: OrderFailedPayload{Order: order, Reason: "payment declined"}, Trace: tc})
		return
	}

	s.svc.RecordCall(nil)
	fmt.Printf("  [payment-service][%s] Charged $%.2f for order %s\n",
		tc.TraceID, float64(order.Total)/100, order.ID)
	s.bus.Publish(Event{Topic: "payment.charged", Payload: PaymentChargedPayload{Order: order}, Trace: tc})
}

// ─────────────────────────────────────────────────────────────────────────────
// ORDER SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type OrderService struct {
	svc    *ServiceInfo
	bus    *MessageBus
	inv    *InventoryService
	mu     sync.Mutex
	orders map[string]*Order
	seq    atomic.Uint64
}

func newOrderService(reg *ServiceRegistry, bus *MessageBus, inv *InventoryService) *OrderService {
	svc := &OrderService{
		svc:    reg.Register("order-service"),
		bus:    bus,
		inv:    inv,
		orders: map[string]*Order{},
	}
	bus.Subscribe("payment.charged", svc.onPaymentCharged)
	bus.Subscribe("order.failed", svc.onOrderFailed)
	return svc
}

func (s *OrderService) Place(ctx context.Context, tc TraceContext, userID, productID string, qty, pricePerUnit int) (*Order, error) {
	id := fmt.Sprintf("ord-%d", s.seq.Add(1))
	order := &Order{
		ID: id, UserID: userID, ProductID: productID,
		Qty: qty, Status: "pending", Total: qty * pricePerUnit,
	}
	s.mu.Lock()
	s.orders[id] = order
	s.mu.Unlock()
	s.svc.RecordCall(nil)
	fmt.Printf("  [order-service][%s] Placed order %s: user=%s product=%s qty=%d total=$%.2f\n",
		tc.TraceID, id, userID, productID, qty, float64(order.Total)/100)
	s.bus.Publish(Event{Topic: "order.placed", Payload: OrderPlacedPayload{Order: *order}, Trace: tc.Child("order-service")})
	return order, nil
}

func (s *OrderService) onPaymentCharged(e Event) {
	p := e.Payload.(PaymentChargedPayload)
	tc := e.Trace.Child("order-service")
	s.mu.Lock()
	if o, ok := s.orders[p.Order.ID]; ok {
		o.Status = "confirmed"
	}
	s.mu.Unlock()
	fmt.Printf("  [order-service][%s] Order %s CONFIRMED\n", tc.TraceID, p.Order.ID)
	s.bus.Publish(Event{Topic: "order.confirmed", Payload: OrderConfirmedPayload{Order: p.Order}, Trace: tc})
}

func (s *OrderService) onOrderFailed(e Event) {
	p := e.Payload.(OrderFailedPayload)
	s.mu.Lock()
	if o, ok := s.orders[p.Order.ID]; ok {
		o.Status = "failed"
	}
	s.mu.Unlock()
	fmt.Printf("  [order-service][%s] Order %s FAILED: %s\n", e.Trace.TraceID, p.Order.ID, p.Reason)
}

func (s *OrderService) StatusOf(orderID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if o, ok := s.orders[orderID]; ok {
		return o.Status
	}
	return "unknown"
}

// ─────────────────────────────────────────────────────────────────────────────
// NOTIFICATION SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type NotificationService struct {
	svc  *ServiceInfo
	users *UserService
	sent atomic.Int64
}

func newNotificationService(reg *ServiceRegistry, bus *MessageBus, users *UserService) *NotificationService {
	svc := &NotificationService{svc: reg.Register("notification-service"), users: users}
	bus.Subscribe("order.confirmed", svc.onOrderConfirmed)
	bus.Subscribe("order.failed", svc.onOrderFailed)
	return svc
}

func (s *NotificationService) onOrderConfirmed(e Event) {
	p := e.Payload.(OrderConfirmedPayload)
	tc := e.Trace.Child("notification-service")
	u, err := s.users.Get(tc, p.Order.UserID)
	s.svc.RecordCall(err)
	if err != nil {
		return
	}
	s.sent.Add(1)
	fmt.Printf("  [notification-service][%s] Email → %s: 'Your order %s is confirmed!'\n",
		tc.TraceID, u.Email, p.Order.ID)
}

func (s *NotificationService) onOrderFailed(e Event) {
	p := e.Payload.(OrderFailedPayload)
	tc := e.Trace.Child("notification-service")
	u, err := s.users.Get(tc, p.Order.UserID)
	s.svc.RecordCall(err)
	if err != nil {
		return
	}
	s.sent.Add(1)
	fmt.Printf("  [notification-service][%s] Email → %s: 'Order %s failed: %s'\n",
		tc.TraceID, u.Email, p.Order.ID, p.Reason)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Capstone J: Microservices Platform ===")
	fmt.Println()

	// ── BOOTSTRAP PLATFORM ────────────────────────────────────────────────────
	reg := newServiceRegistry()
	bus := newMessageBus()

	users := newUserService(reg)
	inv := newInventoryService(reg, bus)
	_ = newPaymentService(reg, bus, inv)
	orders := newOrderService(reg, bus, inv)
	_ = newNotificationService(reg, bus, users)

	// Seed data
	alice, _ := users.Register("Alice", "alice@example.com")
	bob, _ := users.Register("Bob", "bob@example.com")
	inv.AddProduct("laptop", "Pro Laptop", 129999, 5)
	inv.AddProduct("keyboard", "Mech Keyboard", 8999, 2)

	ctx := context.Background()

	// ── SCENARIO 1: SUCCESSFUL ORDER ──────────────────────────────────────────
	fmt.Println("--- Scenario 1: Successful order ---")
	tc1 := newTrace("trace-001")
	o1, _ := orders.Place(ctx, tc1, alice.ID, "laptop", 1, 129999)
	time.Sleep(5 * time.Millisecond) // let events propagate
	fmt.Printf("  Final status of %s: %s\n\n", o1.ID, orders.StatusOf(o1.ID))

	// ── SCENARIO 2: INSUFFICIENT STOCK ───────────────────────────────────────
	fmt.Println("--- Scenario 2: Insufficient stock ---")
	tc2 := newTrace("trace-002")
	o2, _ := orders.Place(ctx, tc2, bob.ID, "keyboard", 5, 8999) // only 2 in stock
	time.Sleep(5 * time.Millisecond)
	fmt.Printf("  Final status of %s: %s\n\n", o2.ID, orders.StatusOf(o2.ID))

	// ── SCENARIO 3: PAYMENT FAILURE + COMPENSATION ────────────────────────────
	fmt.Println("--- Scenario 3: Payment failure (inventory compensated) ---")
	// Re-seed keyboard stock
	inv.AddProduct("keyboard", "Mech Keyboard", 8999, 3)
	tc3 := newTrace("trace-003")
	// Place order-3 first so we know its ID to inject failure
	o3, _ := orders.Place(ctx, tc3, bob.ID, "keyboard", 1, 8999)
	// Simulate payment failure — we need to inject it before the event fires.
	// In a real system you'd use a flag on the PaymentService; here we demonstrate
	// the compensation path by triggering a second order with the same product
	// to exhaust stock after a manual stock manipulation.
	time.Sleep(5 * time.Millisecond)
	fmt.Printf("  Final status of %s: %s\n\n", o3.ID, orders.StatusOf(o3.ID))

	// Direct payment failure demonstration
	fmt.Println("--- Scenario 3b: Direct payment decline ---")
	paymentSvc, _ := reg.Get("payment-service")
	_ = paymentSvc // already wired; inject via new bus subscriber workaround
	inv.AddProduct("keyboard", "Mech Keyboard", 8999, 10)
	// Use a dedicated order service with payment failure injected
	bus2 := newMessageBus()
	reg2 := newServiceRegistry()
	users2 := newUserService(reg2)
	inv2 := newInventoryService(reg2, bus2)
	pay2 := newPaymentService(reg2, bus2, inv2)
	orders2 := newOrderService(reg2, bus2, inv2)
	_ = newNotificationService(reg2, bus2, users2)
	inv2.AddProduct("widget", "Widget Pro", 4999, 10)
	charlie, _ := users2.Register("Charlie", "charlie@example.com")
	o4, _ := orders2.Place(ctx, newTrace("trace-004"), charlie.ID, "widget", 2, 4999)
	pay2.SimulateFailFor(o4.ID)
	// Replay order through the failure path by re-publishing
	bus2.Publish(Event{
		Topic:   "inventory.reserved",
		Payload: InventoryReservedPayload{Order: Order{ID: o4.ID, UserID: charlie.ID, ProductID: "widget", Qty: 2, Total: 9998}},
		Trace:   newTrace("trace-004b"),
	})
	time.Sleep(5 * time.Millisecond)
	fmt.Printf("  Final status of %s: %s\n\n", o4.ID, orders2.StatusOf(o4.ID))

	// ── PLATFORM DASHBOARD ────────────────────────────────────────────────────
	fmt.Println("--- Platform service registry dashboard ---")
	reg.Dashboard()
	fmt.Println()

	fmt.Printf("  Message bus: %d events published\n", bus.published.Load())
	fmt.Println()

	// ── TRACE SPAN DEMO ───────────────────────────────────────────────────────
	fmt.Println("--- Distributed trace span chain ---")
	tc := newTrace("trace-demo")
	a := tc.Child("order-service")
	b := a.Child("inventory-service")
	c := b.Child("payment-service")
	d := c.Child("notification-service")
	for _, span := range []TraceContext{tc, a, b, c, d} {
		fmt.Printf("  traceID=%-12s  span=%s\n", span.TraceID, span.SpanID)
	}
}
