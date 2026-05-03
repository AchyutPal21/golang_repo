# Chapter 67 Exercises — Transactions & ACID

## Exercise 1 — Order Saga (`exercises/01_order_saga`)

Build an e-commerce order processing system that uses atomic multi-table transactions for every state change.

### Schema

```sql
customers  (id, name, email, balance INT CHECK(balance >= 0))
products   (id, name, price INT, stock INT CHECK(stock >= 0), version INT)
orders     (id, customer_id, status TEXT, total_cents INT, created_at)
order_items(id, order_id, product_id, qty, unit_price)  -- price snapshot
payments   (id, order_id, amount, status TEXT, created_at)
```

### Operations

**PlaceOrder(ctx, customerID, items)**
- Atomically in a single transaction:
  1. Create order row (status=pending)
  2. For each item: read stock+version, check sufficient stock, deduct via optimistic lock
  3. Insert order_items with unit_price snapshot
  4. Update order status to `completed`, set total
  5. Charge customer (UPDATE WHERE balance >= total → 0 rows = insufficient funds)
  6. Insert payment record
- Return `*OrderResult{OrderID, Total, PaymentID, ItemsCount}`

**CancelOrder(ctx, orderID)**
- Atomically:
  1. Read order — fail with `errOrderNotFound` if missing
  2. Fail with `errAlreadyCancelled` if status is already `cancelled`
  3. Restore inventory for each item (increment stock + version)
  4. Mark order `cancelled`
  5. Refund customer balance
  6. Insert payment record with `status='refunded'`

### Error types

```go
var (
    errInsufficientStock   = errors.New("insufficient stock")
    errInsufficientBalance = errors.New("insufficient balance")
    errOrderNotFound       = errors.New("order not found")
    errAlreadyCancelled    = errors.New("order already cancelled")
    errOptimisticConflict  = errors.New("optimistic lock conflict")
)
```

### Expected behavior

| Scenario | Result |
|---|---|
| Valid order with stock available | ✓ order created, stock decremented, customer charged |
| Insufficient stock | ✗ `errInsufficientStock` — no changes committed |
| Insufficient balance | ✗ `errInsufficientBalance` — stock unchanged, order rolled back |
| Cancel pending order | ✓ stock restored, customer refunded, status=cancelled |
| Cancel already-cancelled order | ✗ `errAlreadyCancelled` |
| Stock after failed order | Unchanged from before the attempt |

### Key implementation notes

- Optimistic locking: `UPDATE products SET stock = stock - ?, version = version + 1 WHERE id = ? AND version = ?` — if 0 rows affected, return `errOptimisticConflict`
- Unit price snapshot: store `price` at the time of order, not a FK to current price — prices change
- `defer tx.Rollback()` immediately after `BeginTx` — rollback is a no-op after Commit
- All money values in integer cents — avoids float rounding errors
