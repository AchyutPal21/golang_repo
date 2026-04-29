// EXERCISE 30.1 — Add a new adapter without changing the application layer.
//
// The application service (InventoryService) is identical to example 02.
// Your task: add a JSONAdapter primary adapter that reads commands from a
// slice of JSON-like structs and drives the application service.
// The service must remain unchanged.
//
// Run (from the chapter folder):
//   go run ./exercises/01_add_adapter

package main

import (
	"fmt"
	"strings"
)

// ─── Domain ───────────────────────────────────────────────────────────────────

type Product struct {
	SKU   string
	Name  string
	Price float64
	Stock int
}

var ErrOutOfStock = fmt.Errorf("out of stock")
var ErrProductNotFound = fmt.Errorf("product not found")

// ─── Ports ────────────────────────────────────────────────────────────────────

type InventoryUseCase interface {
	GetProduct(sku string) (Product, error)
	Reserve(sku string, qty int) error
	Restock(sku string, qty int) error
}

type ProductStore interface {
	FindBySKU(sku string) (Product, error)
	Update(p Product) error
	All() ([]Product, error)
}

type EventBus interface {
	Publish(topic string, payload map[string]any) error
}

// ─── Application service (unchanged from example 02) ─────────────────────────

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
	_ = s.bus.Publish("inventory.reserved", map[string]any{"sku": sku, "qty": qty})
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
	_ = s.bus.Publish("inventory.restocked", map[string]any{"sku": sku, "qty": qty})
	return nil
}

// ─── Infrastructure adapters ──────────────────────────────────────────────────

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

type logEventBus struct{}

func (logEventBus) Publish(topic string, payload map[string]any) error {
	var parts []string
	for k, v := range payload {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	fmt.Printf("  [EVENT] %s {%s}\n", topic, strings.Join(parts, ", "))
	return nil
}

// ─── NEW: JSONAdapter primary adapter ─────────────────────────────────────────
//
// Simulates receiving a batch of commands encoded as simple structs
// (real code would use encoding/json). The adapter translates each
// command to an application use-case call — it does NOT contain business logic.

type Command struct {
	Op  string // "reserve" | "restock" | "get"
	SKU string
	Qty int
}

type JSONAdapter struct{ uc InventoryUseCase }

func (j *JSONAdapter) Process(cmds []Command) {
	fmt.Println("=== JSONAdapter: processing commands ===")
	for _, cmd := range cmds {
		switch cmd.Op {
		case "get":
			p, err := j.uc.GetProduct(cmd.SKU)
			if err != nil {
				fmt.Printf("  GET  %s → error: %v\n", cmd.SKU, err)
			} else {
				fmt.Printf("  GET  %s → stock=%d price=%.2f\n", p.SKU, p.Stock, p.Price)
			}
		case "reserve":
			err := j.uc.Reserve(cmd.SKU, cmd.Qty)
			if err != nil {
				fmt.Printf("  RSRV %s qty=%d → error: %v\n", cmd.SKU, cmd.Qty, err)
			} else {
				fmt.Printf("  RSRV %s qty=%d → ok\n", cmd.SKU, cmd.Qty)
			}
		case "restock":
			err := j.uc.Restock(cmd.SKU, cmd.Qty)
			if err != nil {
				fmt.Printf("  RSTO %s qty=%d → error: %v\n", cmd.SKU, cmd.Qty, err)
			} else {
				fmt.Printf("  RSTO %s qty=%d → ok\n", cmd.SKU, cmd.Qty)
			}
		default:
			fmt.Printf("  unknown op: %q\n", cmd.Op)
		}
	}
}

// ─── Composition root ─────────────────────────────────────────────────────────

func main() {
	store := newMemProductStore(
		Product{SKU: "ALPHA", Name: "Alpha Widget", Price: 14.99, Stock: 10},
		Product{SKU: "BETA", Name: "Beta Gadget", Price: 29.99, Stock: 3},
	)
	svc := NewInventoryService(store, logEventBus{})

	adapter := &JSONAdapter{uc: svc}
	adapter.Process([]Command{
		{Op: "get", SKU: "ALPHA"},
		{Op: "reserve", SKU: "ALPHA", Qty: 4},
		{Op: "reserve", SKU: "BETA", Qty: 5},  // should fail — only 3 in stock
		{Op: "restock", SKU: "BETA", Qty: 20},
		{Op: "reserve", SKU: "BETA", Qty: 5},  // should succeed now
		{Op: "get", SKU: "ALPHA"},
		{Op: "get", SKU: "BETA"},
		{Op: "get", SKU: "UNKNOWN"}, // not found
	})
}
