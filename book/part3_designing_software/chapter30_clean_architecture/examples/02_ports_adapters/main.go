// FILE: book/part3_designing_software/chapter30_clean_architecture/examples/02_ports_adapters/main.go
// CHAPTER: 30 — Clean / Hexagonal Architecture
// TOPIC: Ports and Adapters (Hexagonal Architecture) — multiple adapters for
//        the same port, swapped at the composition root.
//
// Run (from the chapter folder):
//   go run ./examples/02_ports_adapters

package main

import (
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type Product struct {
	SKU   string
	Name  string
	Price float64
	Stock int
}

var ErrOutOfStock = fmt.Errorf("out of stock")
var ErrProductNotFound = fmt.Errorf("product not found")

// ─────────────────────────────────────────────────────────────────────────────
// APPLICATION PORTS (interfaces defined by the application layer)
//
// Primary (driving) port  — how the outside world calls into the app.
// Secondary (driven) port — how the app calls outward (storage, events, etc.).
// ─────────────────────────────────────────────────────────────────────────────

// Primary port: what the application exposes to any driver (HTTP, CLI, gRPC…).
type InventoryUseCase interface {
	GetProduct(sku string) (Product, error)
	Reserve(sku string, qty int) error
	Restock(sku string, qty int) error
}

// Secondary ports: what the application needs from infrastructure.
type ProductStore interface {
	FindBySKU(sku string) (Product, error)
	Update(p Product) error
	All() ([]Product, error)
}

type EventBus interface {
	Publish(topic string, payload map[string]any) error
}

// ─────────────────────────────────────────────────────────────────────────────
// APPLICATION SERVICE (implements the primary port, uses secondary ports)
// ─────────────────────────────────────────────────────────────────────────────

type InventoryService struct {
	store ProductStore
	bus   EventBus
}

func NewInventoryService(store ProductStore, bus EventBus) *InventoryService {
	return &InventoryService{store: store, bus: bus}
}

func (s *InventoryService) GetProduct(sku string) (Product, error) {
	return s.store.FindBySKU(sku)
}

func (s *InventoryService) Reserve(sku string, qty int) error {
	p, err := s.store.FindBySKU(sku)
	if err != nil {
		return err
	}
	if p.Stock < qty {
		return ErrOutOfStock
	}
	p.Stock -= qty
	if err := s.store.Update(p); err != nil {
		return err
	}
	_ = s.bus.Publish("inventory.reserved", map[string]any{
		"sku": sku, "qty": qty, "remaining": p.Stock,
	})
	return nil
}

func (s *InventoryService) Restock(sku string, qty int) error {
	p, err := s.store.FindBySKU(sku)
	if err != nil {
		return err
	}
	p.Stock += qty
	if err := s.store.Update(p); err != nil {
		return err
	}
	_ = s.bus.Publish("inventory.restocked", map[string]any{
		"sku": sku, "qty": qty, "total": p.Stock,
	})
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECONDARY ADAPTERS (infrastructure implementations of driven ports)
// ─────────────────────────────────────────────────────────────────────────────

// memProductStore — in-memory adapter.
type memProductStore struct{ products map[string]Product }

func newMemProductStore(products ...Product) *memProductStore {
	s := &memProductStore{products: make(map[string]Product)}
	for _, p := range products {
		s.products[p.SKU] = p
	}
	return s
}

func (s *memProductStore) FindBySKU(sku string) (Product, error) {
	p, ok := s.products[sku]
	if !ok {
		return Product{}, ErrProductNotFound
	}
	return p, nil
}

func (s *memProductStore) Update(p Product) error { s.products[p.SKU] = p; return nil }
func (s *memProductStore) All() ([]Product, error) {
	out := make([]Product, 0, len(s.products))
	for _, p := range s.products {
		out = append(out, p)
	}
	return out, nil
}

// stdoutEventBus — logs events to stdout.
type stdoutEventBus struct{}

func (stdoutEventBus) Publish(topic string, payload map[string]any) error {
	var parts []string
	for k, v := range payload {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	fmt.Printf("  [EVENT] %s  {%s}\n", topic, strings.Join(parts, ", "))
	return nil
}

// noopEventBus — discards events (used in tests or offline mode).
type noopEventBus struct{ published []string }

func (b *noopEventBus) Publish(topic string, _ map[string]any) error {
	b.published = append(b.published, topic)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// PRIMARY ADAPTERS (driving adapters — translate external callers to the port)
// ─────────────────────────────────────────────────────────────────────────────

// CLIAdapter drives the application from a command-line style loop.
type CLIAdapter struct{ uc InventoryUseCase }

func (c *CLIAdapter) Run() {
	fmt.Println("=== CLI driver ===")
	p, err := c.uc.GetProduct("WIDGET-A")
	if err != nil {
		fmt.Println("  error:", err)
		return
	}
	fmt.Printf("  product: %s  price=%.2f  stock=%d\n", p.Name, p.Price, p.Stock)

	fmt.Println("  reserving 3 units…")
	if err := c.uc.Reserve("WIDGET-A", 3); err != nil {
		fmt.Println("  reserve error:", err)
	}

	fmt.Println("  reserving 10 more (should fail)…")
	if err := c.uc.Reserve("WIDGET-A", 10); err != nil {
		fmt.Println("  expected error:", err)
	}

	fmt.Println("  restocking 20 units…")
	if err := c.uc.Restock("WIDGET-A", 20); err != nil {
		fmt.Println("  restock error:", err)
	}

	p, _ = c.uc.GetProduct("WIDGET-A")
	fmt.Printf("  final stock: %d\n", p.Stock)
}

// BatchAdapter simulates a scheduled job driver.
type BatchAdapter struct {
	uc    InventoryUseCase
	store ProductStore
}

func (b *BatchAdapter) ReplenishLowStock(threshold, fillTo int) {
	fmt.Println()
	fmt.Printf("=== batch replenish (threshold=%d, fillTo=%d) ===\n", threshold, fillTo)
	products, err := b.store.All()
	if err != nil {
		fmt.Println("  error fetching products:", err)
		return
	}
	for _, p := range products {
		if p.Stock < threshold {
			qty := fillTo - p.Stock
			fmt.Printf("  %s: stock=%d < %d — restocking +%d\n", p.SKU, p.Stock, threshold, qty)
			if err := b.uc.Restock(p.SKU, qty); err != nil {
				fmt.Println("  restock error:", err)
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// COMPOSITION ROOT
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	_ = time.RFC3339 // keep import; would be used for timestamps in a real app

	store := newMemProductStore(
		Product{SKU: "WIDGET-A", Name: "Widget Alpha", Price: 9.99, Stock: 12},
		Product{SKU: "GADGET-B", Name: "Gadget Beta", Price: 49.99, Stock: 2},
	)
	bus := stdoutEventBus{}
	svc := NewInventoryService(store, bus)

	// Two different primary adapters drive the same application service.
	cli := &CLIAdapter{uc: svc}
	cli.Run()

	batch := &BatchAdapter{uc: svc, store: store}
	batch.ReplenishLowStock(5, 20)

	// Swap to noop bus — same store, different event adapter.
	fmt.Println()
	fmt.Println("=== noop event bus (silent events) ===")
	noop := &noopEventBus{}
	svc2 := NewInventoryService(store, noop)
	_ = svc2.Reserve("GADGET-B", 1)
	fmt.Printf("  events captured (not printed): %v\n", noop.published)
}
