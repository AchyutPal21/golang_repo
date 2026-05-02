# Chapter 34 — Exercises

## 34.1 — Multi-repository order service

Run [`exercises/01_multi_repo`](exercises/01_multi_repo/main.go).

`OrderService` is wired with `CustomerRepository` + `OrderRepository`. Cross-repo operations apply premium discounts and wrap domain errors.

Try:
- Add a `ShipOrder(orderID OrderID) error` method that checks the order is `confirmed` before transitioning to `shipped`.
- Add `CustomerLifetimeValue(customerID CustomerID) (float64, error)` that sums all order totals for a customer across both statuses.
- Add a `read-through cache` wrapper around `CustomerRepository` that memoises `FindByID` results in a `map[CustomerID]Customer`.

## 34.2 ★ — Specification for orders

Add a `Spec` system to `OrderRepository`:

```go
type OrderSpec interface { Matches(o Order) bool }

func ByStatus(s OrderStatus) OrderSpec
func ByCustomer(id CustomerID) OrderSpec
func PlacedAfter(t time.Time) OrderSpec
func And(a, b OrderSpec) OrderSpec
```

Replace `FindByCustomer` with `Query(spec OrderSpec) ([]Order, error)`.

## 34.3 ★★ — Transactional save

Add a `SaveAll(orders []Order) error` method to `OrderRepository` that saves all orders atomically — if any `Save` fails, none are persisted. Demonstrate the rollback by injecting a failing store (returns an error on the 3rd call).
