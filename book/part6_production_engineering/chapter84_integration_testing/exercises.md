# Chapter 84 Exercises — Integration Testing

## Exercise 1 — Order Repository Suite (`exercises/01_postgres_suite`)

Build a full integration test suite for an order repository, covering CRUD, status transitions, customer listing, concurrent inserts, and constraint enforcement.

### Repository interface

```go
func (r *OrderRepository) Create(ctx, customerID string, total int) (string, error)
func (r *OrderRepository) Get(ctx, id string) (*OrderRow, error)
func (r *OrderRepository) UpdateStatus(ctx, id, status string) error
func (r *OrderRepository) ListByCustomer(ctx, customerID string) ([]*OrderRow, error)
func (r *OrderRepository) Delete(ctx, id string) error
```

### Constraints to enforce

- `customerID` must not be empty → return `ConstraintError`
- `total` must be non-negative → return `ConstraintError`
- `Get`/`UpdateStatus`/`Delete` on unknown ID → return error

### Test groups

1. **CRUD** — create, get found, get not found, constraint violations
2. **Status transitions** — update pending → shipped; update nonexistent
3. **List by customer** — multiple orders for one customer; empty list for unknown
4. **Delete** — removes row; subsequent Get fails; delete nonexistent returns error
5. **Concurrent writes** — 20 goroutines each inserting one order; final count = 20

### Hints

- Use a fresh `*OrderDB` per test group for isolation
- The concurrent test verifies that the DB's mutex prevents data races — run with `go run -race`
- `ListByCustomer` should return the orders in any order; test by building a set of returned IDs
