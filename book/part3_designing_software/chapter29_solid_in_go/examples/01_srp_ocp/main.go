// FILE: book/part3_designing_software/chapter29_solid_in_go/examples/01_srp_ocp/main.go
// CHAPTER: 29 — SOLID in Go
// TOPIC: Single Responsibility Principle and Open/Closed Principle in idiomatic Go.
//
// Run (from the chapter folder):
//   go run ./examples/01_srp_ocp

package main

import (
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// S — Single Responsibility Principle
//
// A type should have one reason to change.
// In Go: keep types small; separate concerns into distinct types.
// ─────────────────────────────────────────────────────────────────────────────

// ── BAD: OrderProcessor does everything ──────────────────────────────────────

type badOrderProcessor struct{}

// One struct handles pricing, tax, email, and persistence.
// Any change to tax rules, email templates, or DB schema requires editing this.
func (b *badOrderProcessor) Process(productID string, qty int) error {
	price := 9.99 * float64(qty) // hard-coded pricing
	tax := price * 0.08          // hard-coded tax rule
	total := price + tax

	fmt.Printf("[BAD] total=%.2f (includes tax)\n", total)
	fmt.Printf("[BAD] sending email: Your order for %s x%d is confirmed\n", productID, qty)
	fmt.Printf("[BAD] INSERT INTO orders (product, qty, total) VALUES (%s, %d, %.2f)\n",
		productID, qty, total)
	return nil
}

// ── GOOD: each type has one reason to change ─────────────────────────────────

type PriceCalculator struct {
	unitPrices map[string]float64
}

func NewPriceCalculator() *PriceCalculator {
	return &PriceCalculator{unitPrices: map[string]float64{
		"WIDGET": 9.99,
		"GADGET": 49.99,
	}}
}

func (p *PriceCalculator) UnitPrice(productID string) (float64, error) {
	price, ok := p.unitPrices[productID]
	if !ok {
		return 0, fmt.Errorf("unknown product: %s", productID)
	}
	return price, nil
}

// TaxPolicy — only reason to change: tax rules change.
type TaxPolicy struct{ rate float64 }

func (t TaxPolicy) Apply(amount float64) float64 { return amount * (1 + t.rate) }

// ConfirmationMailer — only reason to change: email template or SMTP config.
type ConfirmationMailer struct{ from string }

func (m *ConfirmationMailer) Send(to, productID string, qty int) {
	fmt.Printf("[MAIL] from=%s to=%s: order %s x%d confirmed\n", m.from, to, productID, qty)
}

// OrderRecord is pure data — no behavior, just what we persist.
type OrderRecord struct {
	ID        string
	ProductID string
	Quantity  int
	Total     float64
	CreatedAt time.Time
}

// OrderRepository — only reason to change: storage mechanism.
type OrderRepository struct{ orders []OrderRecord }

func (r *OrderRepository) Save(o OrderRecord) {
	r.orders = append(r.orders, o)
	fmt.Printf("[DB] saved order %s total=%.2f\n", o.ID, o.Total)
}

// OrderCoordinator wires the separate concerns together.
type OrderCoordinator struct {
	pricing *PriceCalculator
	tax     TaxPolicy
	mailer  *ConfirmationMailer
	repo    *OrderRepository
	nextID  int
}

func NewOrderCoordinator(
	pricing *PriceCalculator,
	tax TaxPolicy,
	mailer *ConfirmationMailer,
	repo *OrderRepository,
) *OrderCoordinator {
	return &OrderCoordinator{pricing: pricing, tax: tax, mailer: mailer, repo: repo, nextID: 1}
}

func (c *OrderCoordinator) PlaceOrder(customerEmail, productID string, qty int) error {
	unit, err := c.pricing.UnitPrice(productID)
	if err != nil {
		return err
	}
	total := c.tax.Apply(unit * float64(qty))
	rec := OrderRecord{
		ID:        fmt.Sprintf("ORD-%04d", c.nextID),
		ProductID: productID,
		Quantity:  qty,
		Total:     total,
		CreatedAt: time.Now(),
	}
	c.nextID++
	c.repo.Save(rec)
	c.mailer.Send(customerEmail, productID, qty)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// O — Open/Closed Principle
//
// Open for extension, closed for modification.
// In Go: express extension points as interfaces; add behaviour by adding new
// types that satisfy the interface — not by editing existing types.
// ─────────────────────────────────────────────────────────────────────────────

// ── BAD: switch on type — must modify every time a new discount type is added ─

type badDiscountType int

const (
	noDiscount   badDiscountType = iota
	percentOff                   // requires modifying Apply every time
	absoluteOff                  // same
)

func applyBadDiscount(t badDiscountType, price, value float64) float64 {
	switch t {
	case percentOff:
		return price * (1 - value/100)
	case absoluteOff:
		return price - value
	default:
		return price
	}
}

// ── GOOD: Discount is an interface — extend by adding types, not editing ──────

type Discount interface {
	Apply(price float64) float64
	Description() string
}

type NoDiscount struct{}

func (NoDiscount) Apply(price float64) float64 { return price }
func (NoDiscount) Description() string         { return "no discount" }

type PercentDiscount struct{ Percent float64 }

func (d PercentDiscount) Apply(price float64) float64 { return price * (1 - d.Percent/100) }
func (d PercentDiscount) Description() string {
	return fmt.Sprintf("%.0f%% off", d.Percent)
}

type AbsoluteDiscount struct{ Amount float64 }

func (d AbsoluteDiscount) Apply(price float64) float64 { return price - d.Amount }
func (d AbsoluteDiscount) Description() string {
	return fmt.Sprintf("$%.2f off", d.Amount)
}

// BuyNGetMDiscount — added without touching any existing code.
type BuyNGetMDiscount struct{ N, M int }

func (d BuyNGetMDiscount) Apply(price float64) float64 {
	if d.N+d.M == 0 {
		return price
	}
	paidFraction := float64(d.N) / float64(d.N+d.M)
	return price * paidFraction
}
func (d BuyNGetMDiscount) Description() string {
	return fmt.Sprintf("buy %d get %d free", d.N, d.M)
}

// Checkout is closed for modification — open for new Discount types.
func Checkout(price float64, discounts []Discount) float64 {
	descs := make([]string, 0, len(discounts))
	for _, d := range discounts {
		price = d.Apply(price)
		if d.Description() != "no discount" {
			descs = append(descs, d.Description())
		}
	}
	applied := "none"
	if len(descs) > 0 {
		applied = strings.Join(descs, " + ")
	}
	fmt.Printf("[CHECKOUT] applied: %-30s  final=%.2f\n", applied, price)
	return price
}

func main() {
	fmt.Println("=== SRP: bad (monolith) ===")
	bad := &badOrderProcessor{}
	_ = bad.Process("WIDGET", 2)

	fmt.Println()
	fmt.Println("=== SRP: good (separated concerns) ===")
	coord := NewOrderCoordinator(
		NewPriceCalculator(),
		TaxPolicy{rate: 0.08},
		&ConfirmationMailer{from: "orders@example.com"},
		&OrderRepository{},
	)
	if err := coord.PlaceOrder("alice@example.com", "WIDGET", 3); err != nil {
		fmt.Println("error:", err)
	}

	fmt.Println()
	fmt.Println("=== OCP: bad (switch) ===")
	fmt.Printf("percent off: %.2f\n", applyBadDiscount(percentOff, 100.0, 10))
	fmt.Printf("absolute off: %.2f\n", applyBadDiscount(absoluteOff, 100.0, 15))

	fmt.Println()
	fmt.Println("=== OCP: good (interface extension) ===")
	base := 100.0
	Checkout(base, []Discount{NoDiscount{}})
	Checkout(base, []Discount{PercentDiscount{10}})
	Checkout(base, []Discount{AbsoluteDiscount{15}})
	Checkout(base, []Discount{BuyNGetMDiscount{N: 2, M: 1}}) // 33% effective discount
	Checkout(base, []Discount{PercentDiscount{10}, AbsoluteDiscount{5}})
}
