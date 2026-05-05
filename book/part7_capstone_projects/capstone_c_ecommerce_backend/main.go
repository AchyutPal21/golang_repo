// CAPSTONE C — E-commerce Backend
// Simulates: product catalog, cart management, order placement via the saga
// pattern (ReserveInventory → ChargePayment → ConfirmOrder), with
// compensating actions that run in reverse on failure.
// No external dependencies — stdlib only.
//
// Run:
//
//	go run ./book/part7_capstone_projects/capstone_c_ecommerce_backend

package main

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

// OrderStatus represents the lifecycle state of an order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusFailed    OrderStatus = "failed"
)

// Product is a catalog entry with a current stock level.
type Product struct {
	ID    uint64
	Name  string
	Price float64 // USD cents to avoid float arithmetic in real code; float64 for readability here
	Stock int
}

// CartItem is a line item inside a cart.
type CartItem struct {
	ProductID uint64
	Name      string
	UnitPrice float64
	Quantity  int
}

// Order is created when a saga run begins.
type Order struct {
	ID         uint64
	CustomerID string
	Items      []CartItem
	Total      float64
	Status     OrderStatus
}

// ─────────────────────────────────────────────────────────────────────────────
// PRODUCT CATALOG
// ─────────────────────────────────────────────────────────────────────────────

// ProductCatalog is an in-memory product store. Reads dominate so it uses
// sync.RWMutex — multiple concurrent reads are allowed.
type ProductCatalog struct {
	mu       sync.RWMutex
	products map[uint64]*Product
}

func NewProductCatalog() *ProductCatalog {
	return &ProductCatalog{products: make(map[uint64]*Product)}
}

func (c *ProductCatalog) Add(p Product) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.products[p.ID] = &p
}

// Get returns a copy of the product — callers must not mutate the return value
// to change catalog state.
func (c *ProductCatalog) Get(id uint64) (Product, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.products[id]
	if !ok {
		return Product{}, false
	}
	return *p, true
}

// ─────────────────────────────────────────────────────────────────────────────
// CART
// ─────────────────────────────────────────────────────────────────────────────

// Cart accumulates items before checkout. Each Cart belongs to a single
// customer goroutine; external locking is the caller's responsibility when
// sharing across goroutines.
type Cart struct {
	mu         sync.Mutex
	CustomerID string
	items      map[uint64]*CartItem
}

func NewCart(customerID string) *Cart {
	return &Cart{
		CustomerID: customerID,
		items:      make(map[uint64]*CartItem),
	}
}

// AddItem adds qty units of the given product to the cart.
// If the product is already in the cart the quantities are summed.
func (c *Cart) AddItem(p Product, qty int) error {
	if qty <= 0 {
		return errors.New("cart: quantity must be positive")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if item, exists := c.items[p.ID]; exists {
		item.Quantity += qty
	} else {
		c.items[p.ID] = &CartItem{
			ProductID: p.ID,
			Name:      p.Name,
			UnitPrice: p.Price,
			Quantity:  qty,
		}
	}
	return nil
}

// RemoveItem removes qty units of the product. If the resulting quantity
// reaches zero the item is deleted from the cart.
func (c *Cart) RemoveItem(productID uint64, qty int) error {
	if qty <= 0 {
		return errors.New("cart: quantity must be positive")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	item, exists := c.items[productID]
	if !exists {
		return fmt.Errorf("cart: product %d not in cart", productID)
	}
	if qty > item.Quantity {
		return fmt.Errorf("cart: cannot remove %d, only %d in cart", qty, item.Quantity)
	}
	item.Quantity -= qty
	if item.Quantity == 0 {
		delete(c.items, productID)
	}
	return nil
}

// Total returns the sum of (unit price × quantity) for all items.
func (c *Cart) Total() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	var total float64
	for _, item := range c.items {
		total += item.UnitPrice * float64(item.Quantity)
	}
	return total
}

// Snapshot returns a stable slice of CartItems for use in order placement.
// The caller receives a copy — later cart mutations do not affect it.
func (c *Cart) Snapshot() []CartItem {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]CartItem, 0, len(c.items))
	for _, item := range c.items {
		out = append(out, *item)
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// ORDER STORE
// ─────────────────────────────────────────────────────────────────────────────

// OrderStore persists orders and allows status transitions.
type OrderStore struct {
	mu     sync.Mutex
	orders map[uint64]*Order
}

func NewOrderStore() *OrderStore {
	return &OrderStore{orders: make(map[uint64]*Order)}
}

func (s *OrderStore) Create(o Order) *Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[o.ID] = &o
	return s.orders[o.ID]
}

func (s *OrderStore) UpdateStatus(orderID uint64, status OrderStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[orderID]
	if !ok {
		return fmt.Errorf("order store: order %d not found", orderID)
	}
	o.Status = status
	return nil
}

func (s *OrderStore) Get(orderID uint64) (Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[orderID]
	if !ok {
		return Order{}, false
	}
	return *o, ok
}

// ─────────────────────────────────────────────────────────────────────────────
// INVENTORY SERVICE
// ─────────────────────────────────────────────────────────────────────────────

// ErrInsufficientStock is returned when a reservation would exceed available stock.
var ErrInsufficientStock = errors.New("inventory: insufficient stock")

// InventoryService manages the physical stock level for products.
// Reserve/Release are the two sides of the saga step.
type InventoryService struct {
	mu    sync.Mutex
	stock map[uint64]int // productID → available units
}

func NewInventoryService() *InventoryService {
	return &InventoryService{stock: make(map[uint64]int)}
}

// Seed initialises stock for a product. Called at startup, not concurrently.
func (s *InventoryService) Seed(productID uint64, qty int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stock[productID] = qty
}

// Reserve atomically decrements stock for each item in the order.
// If any product lacks stock the whole reservation is rolled back and
// ErrInsufficientStock is returned.
func (s *InventoryService) Reserve(items []CartItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// First pass: check availability without mutating.
	for _, item := range items {
		avail, ok := s.stock[item.ProductID]
		if !ok || avail < item.Quantity {
			return fmt.Errorf("%w: product %d needs %d, has %d",
				ErrInsufficientStock, item.ProductID, item.Quantity, s.stock[item.ProductID])
		}
	}
	// Second pass: commit the reservation.
	for _, item := range items {
		s.stock[item.ProductID] -= item.Quantity
	}
	return nil
}

// Release is the compensating action for Reserve. It adds the quantities back.
func (s *InventoryService) Release(items []CartItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range items {
		s.stock[item.ProductID] += item.Quantity
	}
}

// Stock returns the current available quantity for a product (for inspection).
func (s *InventoryService) Stock(productID uint64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stock[productID]
}

// ─────────────────────────────────────────────────────────────────────────────
// PAYMENT SERVICE
// ─────────────────────────────────────────────────────────────────────────────

// ErrPaymentDeclined is returned when the payment gateway rejects a charge.
var ErrPaymentDeclined = errors.New("payment: charge declined")

// PaymentService simulates a payment gateway. In production this would call
// Stripe/Braintree over HTTP with an idempotency key.
type PaymentService struct {
	mu      sync.Mutex
	charges map[uint64]float64 // orderID → charged amount
	// forceFailOrderID is set in tests to simulate a declined card.
	forceFailOrderID uint64
}

func NewPaymentService() *PaymentService {
	return &PaymentService{charges: make(map[uint64]float64)}
}

// SimulateDeclineFor configures the service to decline the next charge for
// the given order ID. Used only by the simulation scenarios below.
func (s *PaymentService) SimulateDeclineFor(orderID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.forceFailOrderID = orderID
}

// Charge attempts to capture payment. The orderID acts as an idempotency key:
// a second call for the same orderID is a no-op and returns nil.
func (s *PaymentService) Charge(orderID uint64, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Idempotency: already charged.
	if _, ok := s.charges[orderID]; ok {
		return nil
	}
	if s.forceFailOrderID == orderID {
		s.forceFailOrderID = 0 // consume the flag
		return ErrPaymentDeclined
	}
	s.charges[orderID] = amount
	return nil
}

// Refund is the compensating action for Charge. It removes the charge record.
func (s *PaymentService) Refund(orderID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.charges, orderID)
}

// ─────────────────────────────────────────────────────────────────────────────
// ORDER SAGA
// ─────────────────────────────────────────────────────────────────────────────

// sagaStep pairs a forward action with its compensating action.
type sagaStep struct {
	name      string
	execute   func() error
	compensate func()
}

// OrderSaga orchestrates order placement as a sequence of saga steps.
// If step N fails, all completed steps compensate in reverse order.
type OrderSaga struct {
	inventory *InventoryService
	payment   *PaymentService
	store     *OrderStore
	idGen     *atomic.Uint64
}

func NewOrderSaga(inv *InventoryService, pay *PaymentService, store *OrderStore) *OrderSaga {
	return &OrderSaga{
		inventory: inv,
		payment:   pay,
		store:     store,
		idGen:     &atomic.Uint64{},
	}
}

// PlaceOrder runs the three-step saga for the given cart and customer.
// It returns the Order regardless of outcome; check order.Status to determine
// whether the order succeeded or was rolled back.
func (s *OrderSaga) PlaceOrder(cart *Cart, customerID string) *Order {
	items := cart.Snapshot()
	var total float64
	for _, item := range items {
		total += item.UnitPrice * float64(item.Quantity)
	}

	orderID := s.idGen.Add(1)
	_ = s.store.Create(Order{
		ID:         orderID,
		CustomerID: customerID,
		Items:      items,
		Total:      total,
		Status:     OrderStatusPending,
	})

	steps := []sagaStep{
		{
			name: "ReserveInventory",
			execute: func() error {
				return s.inventory.Reserve(items)
			},
			compensate: func() {
				s.inventory.Release(items)
			},
		},
		{
			name: "ChargePayment",
			execute: func() error {
				return s.payment.Charge(orderID, total)
			},
			compensate: func() {
				s.payment.Refund(orderID)
			},
		},
		{
			name: "ConfirmOrder",
			execute: func() error {
				return s.store.UpdateStatus(orderID, OrderStatusConfirmed)
			},
			// ConfirmOrder is the terminal step; its compensating action marks
			// the order failed. In a real system this would also emit a
			// domain event so downstream services (shipping, email) can react.
			compensate: func() {
				_ = s.store.UpdateStatus(orderID, OrderStatusFailed)
			},
		},
	}

	completed := make([]sagaStep, 0, len(steps))

	for i, step := range steps {
		fmt.Printf("  [Saga] Step %d/%d: %s\n", i+1, len(steps), step.name)
		if err := step.execute(); err != nil {
			fmt.Printf("  [Saga] Step %s failed: %v\n", step.name, err)
			fmt.Printf("  [Saga] Running compensations in reverse...\n")
			// Compensate all completed steps in LIFO order.
			for j := len(completed) - 1; j >= 0; j-- {
				fmt.Printf("  [Saga] Compensating: %s\n", completed[j].name)
				completed[j].compensate()
			}
			_ = s.store.UpdateStatus(orderID, OrderStatusFailed)
			o, _ := s.store.Get(orderID)
			return &o
		}
		completed = append(completed, step)
	}

	o, _ := s.store.Get(orderID)
	return &o
}

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATION SCENARIOS
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// ── Infrastructure setup ────────────────────────────────────────────────
	catalog := NewProductCatalog()
	inventory := NewInventoryService()
	payment := NewPaymentService()
	store := NewOrderStore()
	saga := NewOrderSaga(inventory, payment, store)

	// ── Seed product catalog ────────────────────────────────────────────────
	products := []Product{
		{ID: 1, Name: "Go Programming Book", Price: 49.99, Stock: 10},
		{ID: 2, Name: "Mechanical Keyboard", Price: 129.00, Stock: 3},
		{ID: 3, Name: "USB-C Hub", Price: 35.50, Stock: 5},
	}
	for _, p := range products {
		catalog.Add(p)
		inventory.Seed(p.ID, p.Stock)
	}

	fmt.Println("Initial inventory:")
	for _, p := range products {
		fmt.Printf("  Product %d (%s): %d units\n", p.ID, p.Name, inventory.Stock(p.ID))
	}
	fmt.Println()

	// ── Scenario 1: Successful order ────────────────────────────────────────
	fmt.Println("=== Scenario 1: Successful order ===")
	cart1 := NewCart("alice")
	book, _ := catalog.Get(1)
	kb, _ := catalog.Get(2)
	_ = cart1.AddItem(book, 2)
	_ = cart1.AddItem(kb, 1)
	fmt.Printf("Cart total: $%.2f\n", cart1.Total())

	order1 := saga.PlaceOrder(cart1, "alice")
	fmt.Printf("Order #%d status: %s\n", order1.ID, order1.Status)
	fmt.Printf("Inventory after scenario 1 — Book: %d, Keyboard: %d\n",
		inventory.Stock(1), inventory.Stock(2))
	fmt.Println()

	// ── Scenario 2: Payment failure — inventory must be released ────────────
	fmt.Println("=== Scenario 2: Payment failure (triggers inventory compensation) ===")
	cart2 := NewCart("bob")
	hub, _ := catalog.Get(3)
	_ = cart2.AddItem(hub, 2)
	fmt.Printf("Cart total: $%.2f\n", cart2.Total())

	// The next order will get ID=2; tell the payment service to decline it.
	payment.SimulateDeclineFor(2)

	order2 := saga.PlaceOrder(cart2, "bob")
	fmt.Printf("Order #%d status: %s\n", order2.ID, order2.Status)
	fmt.Printf("Inventory after scenario 2 — Hub: %d (should be 5, reservation released)\n",
		inventory.Stock(3))
	fmt.Println()

	// ── Scenario 3: Insufficient stock ──────────────────────────────────────
	fmt.Println("=== Scenario 3: Insufficient stock ===")
	cart3 := NewCart("carol")
	kb2, _ := catalog.Get(2)
	// Only 2 keyboards remain after scenario 1; request 5.
	_ = cart3.AddItem(kb2, 5)
	fmt.Printf("Cart total: $%.2f\n", cart3.Total())

	order3 := saga.PlaceOrder(cart3, "carol")
	fmt.Printf("Order #%d status: %s\n", order3.ID, order3.Status)
	fmt.Printf("Inventory after scenario 3 — Keyboard: %d (unchanged, saga aborted at step 1)\n",
		inventory.Stock(2))
	fmt.Println()

	// ── Final state ─────────────────────────────────────────────────────────
	fmt.Println("=== Final order states ===")
	for _, id := range []uint64{1, 2, 3} {
		if o, ok := store.Get(id); ok {
			fmt.Printf("  Order #%d  customer=%-6s  total=$%6.2f  status=%s\n",
				o.ID, o.CustomerID, o.Total, o.Status)
		}
	}
}
