// EXERCISE 28.1 — Wire a full application manually.
//
// Build an OrderService with three dependencies:
//   - ProductCatalog (read prices)
//   - PaymentGateway (charge cards)
//   - OrderStore (persist orders)
//
// Wire it in main() with real stubs.
// Then re-wire with fakes and verify the happy path and error paths.
//
// Run (from the chapter folder):
//   go run ./exercises/01_service_wiring

package main

import (
	"errors"
	"fmt"
)

// ─── Interfaces ───────────────────────────────────────────────────────────────

type ProductCatalog interface {
	Price(productID string) (float64, error)
}

type PaymentGateway interface {
	Charge(cardToken string, amount float64) (txID string, err error)
}

type OrderStore interface {
	Save(order Order) error
}

// ─── Domain ───────────────────────────────────────────────────────────────────

type Order struct {
	ID        string
	ProductID string
	Quantity  int
	Amount    float64
	TxID      string
}

// ─── Service ──────────────────────────────────────────────────────────────────

type OrderService struct {
	catalog  ProductCatalog
	payment  PaymentGateway
	store    OrderStore
	nextID   int
}

func NewOrderService(c ProductCatalog, p PaymentGateway, s OrderStore) *OrderService {
	return &OrderService{catalog: c, payment: p, store: s, nextID: 1}
}

var ErrInsufficientStock = errors.New("insufficient stock")

func (svc *OrderService) PlaceOrder(cardToken, productID string, qty int) (Order, error) {
	price, err := svc.catalog.Price(productID)
	if err != nil {
		return Order{}, fmt.Errorf("PlaceOrder: %w", err)
	}

	total := price * float64(qty)
	txID, err := svc.payment.Charge(cardToken, total)
	if err != nil {
		return Order{}, fmt.Errorf("PlaceOrder: charge failed: %w", err)
	}

	order := Order{
		ID:        fmt.Sprintf("ORD-%04d", svc.nextID),
		ProductID: productID,
		Quantity:  qty,
		Amount:    total,
		TxID:      txID,
	}
	svc.nextID++

	if err := svc.store.Save(order); err != nil {
		return Order{}, fmt.Errorf("PlaceOrder: save failed: %w", err)
	}
	return order, nil
}

// ─── Stub implementations ────────────────────────────────────────────────────

type stubCatalog struct{ prices map[string]float64 }

func (c *stubCatalog) Price(id string) (float64, error) {
	p, ok := c.prices[id]
	if !ok {
		return 0, fmt.Errorf("product %q not found", id)
	}
	return p, nil
}

type stubPayment struct{ nextTx int }

func (p *stubPayment) Charge(token string, amount float64) (string, error) {
	if token == "bad-card" {
		return "", errors.New("card declined")
	}
	p.nextTx++
	fmt.Printf("  [PAY] charged %.2f on %s\n", amount, token)
	return fmt.Sprintf("TX-%04d", p.nextTx), nil
}

type stubOrderStore struct{ orders []Order }

func (s *stubOrderStore) Save(o Order) error {
	s.orders = append(s.orders, o)
	return nil
}

func main() {
	catalog := &stubCatalog{prices: map[string]float64{
		"WIDGET": 9.99,
		"GADGET": 49.99,
	}}
	payment := &stubPayment{}
	store := &stubOrderStore{}

	svc := NewOrderService(catalog, payment, store)

	fmt.Println("=== happy path ===")
	order, err := svc.PlaceOrder("tok_visa", "WIDGET", 3)
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Printf("order: %s  amount=%.2f  tx=%s\n", order.ID, order.Amount, order.TxID)
	}

	fmt.Println()
	fmt.Println("=== unknown product ===")
	_, err = svc.PlaceOrder("tok_visa", "UNKNOWN", 1)
	fmt.Println("error:", err)

	fmt.Println()
	fmt.Println("=== card declined ===")
	_, err = svc.PlaceOrder("bad-card", "GADGET", 1)
	fmt.Println("error:", err)

	fmt.Println()
	fmt.Printf("orders saved: %d\n", len(store.orders))
}
