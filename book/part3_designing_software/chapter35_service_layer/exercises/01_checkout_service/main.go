// EXERCISE 35.1 — Build a CheckoutService that coordinates three domains.
//
// CheckoutService orchestrates:
//   - InventoryService  (reserve stock)
//   - PricingService    (calculate total with discounts)
//   - PaymentService    (charge and record receipt)
//
// If any step fails, earlier steps must be compensated (stock released, etc.)
//
// Run (from the chapter folder):
//   go run ./exercises/01_checkout_service

package main

import (
	"errors"
	"fmt"
	"strings"
)

// ─── Domain ───────────────────────────────────────────────────────────────────

type CartItem struct {
	SKU string
	Qty int
}

type Receipt struct {
	OrderID     string
	Items       []CartItem
	Subtotal    float64
	Discount    float64
	Total       float64
	ChargeID    string
}

var (
	ErrOutOfStock    = errors.New("out of stock")
	ErrPaymentFailed = errors.New("payment failed")
)

// ─── Ports ────────────────────────────────────────────────────────────────────

type InventoryPort interface {
	Reserve(sku string, qty int) error
	Release(sku string, qty int) error
	UnitPrice(sku string) (float64, error)
}

type PricingPort interface {
	Calculate(items []CartItem, prices map[string]float64, coupon string) (subtotal, discount float64)
}

type PaymentPort interface {
	Charge(userID string, amountCents int64) (chargeID string, err error)
}

// ─── CheckoutService ──────────────────────────────────────────────────────────

type CheckoutService struct {
	inventory InventoryPort
	pricing   PricingPort
	payment   PaymentPort
	orderSeq  int
}

func NewCheckoutService(inv InventoryPort, price PricingPort, pay PaymentPort) *CheckoutService {
	return &CheckoutService{inventory: inv, pricing: price, payment: pay}
}

func (s *CheckoutService) Checkout(userID string, items []CartItem, coupon string) (Receipt, error) {
	// Step 1: reserve all items (collect what was reserved for rollback).
	var reserved []CartItem
	for _, item := range items {
		if err := s.inventory.Reserve(item.SKU, item.Qty); err != nil {
			// Compensate: release already-reserved items.
			for _, r := range reserved {
				_ = s.inventory.Release(r.SKU, r.Qty)
			}
			return Receipt{}, fmt.Errorf("Checkout: reserve %s: %w", item.SKU, err)
		}
		reserved = append(reserved, item)
	}

	// Step 2: gather prices.
	prices := make(map[string]float64)
	for _, item := range items {
		price, err := s.inventory.UnitPrice(item.SKU)
		if err != nil {
			for _, r := range reserved {
				_ = s.inventory.Release(r.SKU, r.Qty)
			}
			return Receipt{}, fmt.Errorf("Checkout: price %s: %w", item.SKU, err)
		}
		prices[item.SKU] = price
	}

	// Step 3: calculate total.
	subtotal, discount := s.pricing.Calculate(items, prices, coupon)
	total := subtotal - discount
	totalCents := int64(total * 100)

	// Step 4: charge.
	chargeID, err := s.payment.Charge(userID, totalCents)
	if err != nil {
		// Compensate: release all reserved stock.
		for _, r := range reserved {
			_ = s.inventory.Release(r.SKU, r.Qty)
		}
		return Receipt{}, fmt.Errorf("Checkout: %w: %v", ErrPaymentFailed, err)
	}

	s.orderSeq++
	return Receipt{
		OrderID:  fmt.Sprintf("ORD-%04d", s.orderSeq),
		Items:    items,
		Subtotal: subtotal,
		Discount: discount,
		Total:    total,
		ChargeID: chargeID,
	}, nil
}

// ─── Infrastructure ───────────────────────────────────────────────────────────

type memInventory struct {
	stock  map[string]int
	prices map[string]float64
}

func newMemInventory() *memInventory {
	return &memInventory{
		stock:  map[string]int{"WIDGET": 10, "GADGET": 3, "TOOL": 20},
		prices: map[string]float64{"WIDGET": 9.99, "GADGET": 49.99, "TOOL": 19.99},
	}
}

func (m *memInventory) Reserve(sku string, qty int) error {
	avail, ok := m.stock[sku]
	if !ok || avail < qty {
		return fmt.Errorf("%w: %s (available=%d requested=%d)", ErrOutOfStock, sku, avail, qty)
	}
	m.stock[sku] -= qty
	fmt.Printf("  [INV] reserved %s x%d  (remaining=%d)\n", sku, qty, m.stock[sku])
	return nil
}

func (m *memInventory) Release(sku string, qty int) error {
	m.stock[sku] += qty
	fmt.Printf("  [INV] released %s x%d  (remaining=%d)\n", sku, qty, m.stock[sku])
	return nil
}

func (m *memInventory) UnitPrice(sku string) (float64, error) {
	p, ok := m.prices[sku]
	if !ok {
		return 0, fmt.Errorf("unknown SKU: %s", sku)
	}
	return p, nil
}

type simplePricing struct{}

func (simplePricing) Calculate(items []CartItem, prices map[string]float64, coupon string) (float64, float64) {
	subtotal := 0.0
	for _, item := range items {
		subtotal += prices[item.SKU] * float64(item.Qty)
	}
	discount := 0.0
	if strings.ToUpper(coupon) == "SAVE10" {
		discount = subtotal * 0.10
		fmt.Printf("  [PRICE] coupon SAVE10 applied: -$%.2f\n", discount)
	}
	return subtotal, discount
}

type fakePayment struct {
	failFor map[string]bool
	seq     int
}

func (p *fakePayment) Charge(userID string, amountCents int64) (string, error) {
	if p.failFor[userID] {
		return "", fmt.Errorf("card declined for %s", userID)
	}
	p.seq++
	id := fmt.Sprintf("CH-%04d", p.seq)
	fmt.Printf("  [PAY] charged %s $%.2f → %s\n", userID, float64(amountCents)/100, id)
	return id, nil
}

func main() {
	inv := newMemInventory()
	svc := NewCheckoutService(inv, simplePricing{}, &fakePayment{failFor: map[string]bool{}})

	fmt.Println("=== Happy path ===")
	r, err := svc.Checkout("alice", []CartItem{
		{"WIDGET", 2},
		{"GADGET", 1},
	}, "SAVE10")
	if err != nil {
		fmt.Println("  error:", err)
	} else {
		fmt.Printf("  %s  subtotal=$%.2f  discount=$%.2f  total=$%.2f  charge=%s\n",
			r.OrderID, r.Subtotal, r.Discount, r.Total, r.ChargeID)
	}

	fmt.Println()
	fmt.Println("=== Out of stock (GADGET only 2 left) ===")
	_, err = svc.Checkout("bob", []CartItem{
		{"WIDGET", 1},
		{"GADGET", 5}, // only 2 remaining after alice's order
	}, "")
	fmt.Println("  error:", err)
	fmt.Println("  is ErrOutOfStock:", errors.Is(err, ErrOutOfStock))

	fmt.Println()
	fmt.Println("=== Payment failure → stock released ===")
	badPay := &fakePayment{failFor: map[string]bool{"bad-user": true}}
	svc2 := NewCheckoutService(inv, simplePricing{}, badPay)
	_, err = svc2.Checkout("bad-user", []CartItem{{"TOOL", 3}}, "")
	fmt.Println("  error:", err)
	fmt.Println("  is ErrPaymentFailed:", errors.Is(err, ErrPaymentFailed))
}
