// EXERCISE 34.1 — Wire multiple repositories and demonstrate cross-repo queries.
//
// Build an order management system with two repositories:
//   - OrderRepository
//   - CustomerRepository
//
// The OrderService performs cross-repo operations.
//
// Run (from the chapter folder):
//   go run ./exercises/01_multi_repo

package main

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// ─── Domain ───────────────────────────────────────────────────────────────────

type CustomerID int
type OrderID int

type Customer struct {
	ID    CustomerID
	Email string
	Name  string
	Tier  string // "standard" | "premium"
}

type OrderItem struct {
	SKU   string
	Qty   int
	Price float64
}

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusConfirmed OrderStatus = "confirmed"
	StatusShipped   OrderStatus = "shipped"
)

type Order struct {
	ID         OrderID
	CustomerID CustomerID
	Items      []OrderItem
	Status     OrderStatus
	PlacedAt   time.Time
}

func (o Order) Total() float64 {
	total := 0.0
	for _, item := range o.Items {
		total += item.Price * float64(item.Qty)
	}
	return total
}

var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrOrderNotFound    = errors.New("order not found")
)

// ─── Repositories ─────────────────────────────────────────────────────────────

type CustomerRepository interface {
	Save(c Customer) (Customer, error)
	FindByID(id CustomerID) (Customer, error)
	FindAll() ([]Customer, error)
}

type OrderRepository interface {
	Save(o Order) (Order, error)
	FindByID(id OrderID) (Order, error)
	FindByCustomer(customerID CustomerID) ([]Order, error)
	UpdateStatus(id OrderID, status OrderStatus) error
}

// ─── In-memory implementations ────────────────────────────────────────────────

type memCustomerRepo struct {
	data   map[CustomerID]Customer
	nextID CustomerID
}

func NewMemCustomerRepo() CustomerRepository {
	return &memCustomerRepo{data: make(map[CustomerID]Customer), nextID: 1}
}

func (r *memCustomerRepo) Save(c Customer) (Customer, error) {
	if c.ID == 0 {
		c.ID = r.nextID
		r.nextID++
	}
	r.data[c.ID] = c
	return c, nil
}

func (r *memCustomerRepo) FindByID(id CustomerID) (Customer, error) {
	c, ok := r.data[id]
	if !ok {
		return Customer{}, ErrCustomerNotFound
	}
	return c, nil
}

func (r *memCustomerRepo) FindAll() ([]Customer, error) {
	result := make([]Customer, 0, len(r.data))
	for _, c := range r.data {
		result = append(result, c)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

type memOrderRepo struct {
	data   map[OrderID]Order
	nextID OrderID
}

func NewMemOrderRepo() OrderRepository {
	return &memOrderRepo{data: make(map[OrderID]Order), nextID: 1}
}

func (r *memOrderRepo) Save(o Order) (Order, error) {
	if o.ID == 0 {
		o.ID = r.nextID
		r.nextID++
		if o.PlacedAt.IsZero() {
			o.PlacedAt = time.Now()
		}
	}
	r.data[o.ID] = o
	return o, nil
}

func (r *memOrderRepo) FindByID(id OrderID) (Order, error) {
	o, ok := r.data[id]
	if !ok {
		return Order{}, ErrOrderNotFound
	}
	return o, nil
}

func (r *memOrderRepo) FindByCustomer(customerID CustomerID) ([]Order, error) {
	var orders []Order
	for _, o := range r.data {
		if o.CustomerID == customerID {
			orders = append(orders, o)
		}
	}
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	return orders, nil
}

func (r *memOrderRepo) UpdateStatus(id OrderID, status OrderStatus) error {
	o, ok := r.data[id]
	if !ok {
		return ErrOrderNotFound
	}
	o.Status = status
	r.data[id] = o
	return nil
}

// ─── Application service ─────────────────────────────────────────────────────

type OrderService struct {
	customers CustomerRepository
	orders    OrderRepository
}

func NewOrderService(customers CustomerRepository, orders OrderRepository) *OrderService {
	return &OrderService{customers: customers, orders: orders}
}

func (s *OrderService) PlaceOrder(customerID CustomerID, items []OrderItem) (Order, error) {
	cust, err := s.customers.FindByID(customerID)
	if err != nil {
		return Order{}, fmt.Errorf("PlaceOrder: %w", err)
	}

	// Premium customers get 10% off automatically.
	if cust.Tier == "premium" {
		for i := range items {
			items[i].Price *= 0.90
		}
		fmt.Printf("  [DISCOUNT] premium customer %s — 10%% off applied\n", cust.Name)
	}

	order := Order{
		CustomerID: customerID,
		Items:      items,
		Status:     StatusPending,
	}
	return s.orders.Save(order)
}

func (s *OrderService) ConfirmOrder(orderID OrderID) error {
	return s.orders.UpdateStatus(orderID, StatusConfirmed)
}

func (s *OrderService) CustomerReport(customerID CustomerID) {
	cust, err := s.customers.FindByID(customerID)
	if err != nil {
		fmt.Println("customer not found:", customerID)
		return
	}
	orders, _ := s.orders.FindByCustomer(customerID)
	total := 0.0
	for _, o := range orders {
		total += o.Total()
	}
	fmt.Printf("  Customer: %s (%s) — %d orders, lifetime total: $%.2f\n",
		cust.Name, cust.Tier, len(orders), total)
	for _, o := range orders {
		fmt.Printf("    [%d] status=%-10s  total=$%.2f  items=%d\n",
			o.ID, o.Status, o.Total(), len(o.Items))
	}
}

func main() {
	customers := NewMemCustomerRepo()
	orders := NewMemOrderRepo()
	svc := NewOrderService(customers, orders)

	// Register customers.
	alice, _ := customers.Save(Customer{Email: "alice@example.com", Name: "Alice", Tier: "premium"})
	bob, _ := customers.Save(Customer{Email: "bob@example.com", Name: "Bob", Tier: "standard"})

	fmt.Println("=== Place orders ===")
	o1, _ := svc.PlaceOrder(alice.ID, []OrderItem{
		{"WIDGET", 3, 9.99},
		{"GADGET", 1, 49.99},
	})
	fmt.Printf("  order %d placed: total=$%.2f  status=%s\n", o1.ID, o1.Total(), o1.Status)

	o2, _ := svc.PlaceOrder(bob.ID, []OrderItem{
		{"TOOL", 2, 19.99},
	})
	fmt.Printf("  order %d placed: total=$%.2f  status=%s\n", o2.ID, o2.Total(), o2.Status)

	o3, _ := svc.PlaceOrder(alice.ID, []OrderItem{
		{"GADGET", 2, 49.99},
	})
	fmt.Printf("  order %d placed: total=$%.2f  status=%s\n", o3.ID, o3.Total(), o3.Status)

	fmt.Println()
	fmt.Println("=== Confirm orders ===")
	_ = svc.ConfirmOrder(o1.ID)
	_ = svc.ConfirmOrder(o2.ID)
	fmt.Println("  o1 and o2 confirmed")

	fmt.Println()
	fmt.Println("=== Customer reports ===")
	svc.CustomerReport(alice.ID)
	fmt.Println()
	svc.CustomerReport(bob.ID)

	fmt.Println()
	fmt.Println("=== Not found error ===")
	_, err := svc.PlaceOrder(999, []OrderItem{{"X", 1, 1.0}})
	fmt.Println("  error:", err)
	fmt.Println("  is ErrCustomerNotFound:", errors.Is(err, ErrCustomerNotFound))
}
