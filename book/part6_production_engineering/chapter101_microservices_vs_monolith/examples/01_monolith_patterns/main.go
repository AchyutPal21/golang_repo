// FILE: book/part6_production_engineering/chapter101_microservices_vs_monolith/examples/01_monolith_patterns/main.go
// CHAPTER: 101 — Microservices vs Monolith
// TOPIC: Well-structured monolith — domain packages, interface boundaries,
//        in-process call costs, and anti-pattern catalogue.
//
// Run:
//   go run ./book/part6_production_engineering/chapter101_microservices_vs_monolith/examples/01_monolith_patterns

package main

import (
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN INTERFACES — prevent circular imports between packages
// ─────────────────────────────────────────────────────────────────────────────

// OrderStore is the interface the orders domain exposes to callers.
type OrderStore interface {
	Create(order Order) (string, error)
	GetByID(id string) (Order, bool)
}

// CatalogStore is the interface the catalog domain exposes.
type CatalogStore interface {
	GetProduct(id string) (Product, bool)
	Reserve(productID string, qty int) error
}

// BillingService is the interface the billing domain exposes.
type BillingService interface {
	Charge(orderID string, amountCents int) error
}

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Order struct {
	ID          string
	CustomerID  string
	ProductID   string
	Quantity    int
	AmountCents int
	Status      string
}

type Product struct {
	ID         string
	Name       string
	PriceCents int
	Stock      int
}

// ─────────────────────────────────────────────────────────────────────────────
// IN-PROCESS IMPLEMENTATIONS
// ─────────────────────────────────────────────────────────────────────────────

type inMemoryOrderStore struct {
	orders map[string]Order
}

func newOrderStore() *inMemoryOrderStore {
	return &inMemoryOrderStore{orders: map[string]Order{}}
}

func (s *inMemoryOrderStore) Create(o Order) (string, error) {
	s.orders[o.ID] = o
	return o.ID, nil
}

func (s *inMemoryOrderStore) GetByID(id string) (Order, bool) {
	o, ok := s.orders[id]
	return o, ok
}

type inMemoryCatalog struct {
	products map[string]Product
}

func newCatalogStore() *inMemoryCatalog {
	return &inMemoryCatalog{products: map[string]Product{
		"p1": {ID: "p1", Name: "Laptop", PriceCents: 129999, Stock: 10},
		"p2": {ID: "p2", Name: "Keyboard", PriceCents: 8999, Stock: 50},
	}}
}

func (c *inMemoryCatalog) GetProduct(id string) (Product, bool) {
	p, ok := c.products[id]
	return p, ok
}

func (c *inMemoryCatalog) Reserve(productID string, qty int) error {
	p, ok := c.products[productID]
	if !ok {
		return fmt.Errorf("product %s not found", productID)
	}
	if p.Stock < qty {
		return fmt.Errorf("insufficient stock: have %d, want %d", p.Stock, qty)
	}
	p.Stock -= qty
	c.products[productID] = p
	return nil
}

type inMemoryBilling struct{}

func (b *inMemoryBilling) Charge(orderID string, amountCents int) error {
	_ = orderID
	_ = amountCents
	return nil // simulate success
}

// ─────────────────────────────────────────────────────────────────────────────
// APPLICATION SERVICE — orchestrates across domains (in-process)
// ─────────────────────────────────────────────────────────────────────────────

type PlaceOrderRequest struct {
	CustomerID string
	ProductID  string
	Quantity   int
}

type OrderService struct {
	orders  OrderStore
	catalog CatalogStore
	billing BillingService
}

func NewOrderService(o OrderStore, c CatalogStore, b BillingService) *OrderService {
	return &OrderService{orders: o, catalog: c, billing: b}
}

func (s *OrderService) PlaceOrder(req PlaceOrderRequest) (string, error) {
	product, ok := s.catalog.GetProduct(req.ProductID)
	if !ok {
		return "", fmt.Errorf("product not found: %s", req.ProductID)
	}

	if err := s.catalog.Reserve(req.ProductID, req.Quantity); err != nil {
		return "", fmt.Errorf("reserve failed: %w", err)
	}

	totalCents := product.PriceCents * req.Quantity
	orderID := fmt.Sprintf("ord-%d", time.Now().UnixNano())

	order := Order{
		ID: orderID, CustomerID: req.CustomerID,
		ProductID: req.ProductID, Quantity: req.Quantity,
		AmountCents: totalCents, Status: "pending",
	}

	if _, err := s.orders.Create(order); err != nil {
		return "", fmt.Errorf("create order failed: %w", err)
	}

	if err := s.billing.Charge(orderID, totalCents); err != nil {
		return "", fmt.Errorf("billing failed: %w", err)
	}

	return orderID, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// LATENCY COMPARISON: in-process vs RPC
// ─────────────────────────────────────────────────────────────────────────────

type CallCost struct {
	Name      string
	TypicalUs int // microseconds
	Note      string
}

func printLatencyComparison() {
	calls := []CallCost{
		{"In-process function call", 1, "L1 cache, branch prediction"},
		{"In-process via interface", 2, "vtable dispatch, same goroutine"},
		{"Goroutine channel (same host)", 500, "goroutine switch + alloc"},
		{"localhost TCP (loopback)", 50_000, "syscall + kernel TCP stack"},
		{"Same-DC gRPC call", 500_000, "DNS + TCP + TLS + framing + goroutine"},
		{"Cross-region HTTP", 30_000_000, "speed-of-light + routing + TLS"},
	}

	fmt.Printf("  %-40s  %12s  %s\n", "Call type", "Typical", "Notes")
	fmt.Printf("  %s\n", strings.Repeat("-", 80))
	for _, c := range calls {
		var formatted string
		switch {
		case c.TypicalUs < 1000:
			formatted = fmt.Sprintf("%d μs", c.TypicalUs)
		case c.TypicalUs < 1_000_000:
			formatted = fmt.Sprintf("%.1f ms", float64(c.TypicalUs)/1000)
		default:
			formatted = fmt.Sprintf("%.0f ms", float64(c.TypicalUs)/1000)
		}
		fmt.Printf("  %-40s  %12s  %s\n", c.Name, formatted, c.Note)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ANTI-PATTERN CATALOGUE
// ─────────────────────────────────────────────────────────────────────────────

type AntiPattern struct {
	Name        string
	Description string
	Signal      string
	Fix         string
}

func printAntiPatterns() {
	patterns := []AntiPattern{
		{
			Name:        "Distributed monolith",
			Description: "Services deployed separately but sharing a database schema.",
			Signal:      "Team deploys service A but must coordinate with team B because B reads A's tables.",
			Fix:         "Enforce data ownership: one service per schema. Others call the owning service's API.",
		},
		{
			Name:        "Chatty services",
			Description: "Service A makes 10+ synchronous RPC calls to complete a single request.",
			Signal:      "Request latency is 10× higher than the slowest individual service.",
			Fix:         "Batch calls, use async events, or merge the chattiest pair back into one service.",
		},
		{
			Name:        "Shared library monolith",
			Description: "All services share a common library that contains domain logic.",
			Signal:      "Upgrading the shared library requires coordinating all 12 service releases.",
			Fix:         "Libraries should contain utilities (logging, tracing), not domain logic.",
		},
		{
			Name:        "No interface inside monolith",
			Description: "Packages call each other directly, creating hidden coupling.",
			Signal:      "Changing one package forces changes in 6 others. Tests require the full app.",
			Fix:         "Each domain package exposes a narrow interface. Callers depend on the interface.",
		},
		{
			Name:        "Premature decomposition",
			Description: "Services extracted before domain boundaries are understood.",
			Signal:      "Every feature requires changes to 4+ services. API contracts break weekly.",
			Fix:         "Start with a monolith. Extract only stable, well-understood boundaries.",
		},
	}

	for _, p := range patterns {
		fmt.Printf("  ✗ %s\n", p.Name)
		fmt.Printf("    What: %s\n", p.Description)
		fmt.Printf("    Signal: %s\n", p.Signal)
		fmt.Printf("    Fix: %s\n\n", p.Fix)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 101: Monolith Patterns ===")
	fmt.Println()

	// ── IN-PROCESS DOMAIN CALL ────────────────────────────────────────────────
	fmt.Println("--- In-process order placement (3 domain packages, 0 network calls) ---")
	svc := NewOrderService(newOrderStore(), newCatalogStore(), &inMemoryBilling{})

	start := time.Now()
	orderID, err := svc.PlaceOrder(PlaceOrderRequest{
		CustomerID: "cust-42",
		ProductID:  "p1",
		Quantity:   2,
	})
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("  Order failed: %v\n", err)
	} else {
		fmt.Printf("  Order placed: %s  (elapsed: %v)\n", orderID, elapsed)
	}

	// Insufficient stock
	_, err = svc.PlaceOrder(PlaceOrderRequest{
		CustomerID: "cust-43",
		ProductID:  "p1",
		Quantity:   100,
	})
	fmt.Printf("  Insufficient stock: %v\n", err)
	fmt.Println()

	// ── LATENCY COMPARISON ────────────────────────────────────────────────────
	fmt.Println("--- Call latency: in-process vs RPC ---")
	printLatencyComparison()
	fmt.Println()

	// ── MONOLITH STRENGTHS ────────────────────────────────────────────────────
	fmt.Println("--- Monolith strengths ---")
	fmt.Println(`  • In-process calls: μs latency (no serialization, no network)
  • Single deployment artifact: one binary, one config
  • ACID transactions across the entire domain
  • Easy refactoring: rename a type, the compiler catches every caller
  • Simple local debugging: one process, one trace
  • No distributed state: no saga needed for multi-step operations
  • Lower operational overhead: one CI pipeline, one observability setup`)
	fmt.Println()

	// ── ANTI-PATTERNS ─────────────────────────────────────────────────────────
	fmt.Println("--- Monolith and microservice anti-patterns ---")
	printAntiPatterns()

	// ── WHEN TO STAY MONOLITH ─────────────────────────────────────────────────
	fmt.Println("--- When to stay monolith ---")
	fmt.Println(`  ✓ Team size < 10 engineers (communication overhead is low)
  ✓ Domain still evolving (boundaries not yet clear)
  ✓ Deployment already fast and reliable (< 10 min)
  ✓ No module has a radically different scaling profile
  ✓ Operational simplicity matters more than theoretical flexibility
  ✓ Starting a new product (premature decomposition kills velocity)`)
}
