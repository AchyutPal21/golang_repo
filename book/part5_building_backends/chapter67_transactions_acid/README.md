# Chapter 67 — Transactions & ACID

## What you'll learn

The four ACID properties that make database transactions reliable, SQL isolation levels and the anomalies each prevents, optimistic locking with version columns, savepoints for partial rollback, deadlock detection and prevention, and how to structure complex multi-table transactions in Go.

## ACID in one sentence each

| Property | Guarantee |
|---|---|
| **Atomicity** | All operations in a transaction succeed, or none do — no partial updates |
| **Consistency** | Constraints (CHECK, FK, NOT NULL, UNIQUE) are enforced — invalid state can never be committed |
| **Isolation** | Concurrent transactions don't interfere with each other — each sees a consistent snapshot |
| **Durability** | Committed data survives process crashes, power failures, and restarts |

## Isolation levels

| Level | Dirty Read | Non-Repeatable Read | Phantom Read |
|---|---|---|---|
| READ UNCOMMITTED | possible | possible | possible |
| READ COMMITTED *(default)* | prevented | possible | possible |
| REPEATABLE READ | prevented | prevented | possible* |
| SERIALIZABLE | prevented | prevented | prevented |

*MySQL REPEATABLE READ prevents phantoms via gap locks; PostgreSQL uses full snapshot isolation.

## Files

| File | Topic |
|---|---|
| `examples/01_acid_properties/main.go` | Atomicity (rollback on failure), consistency (constraint enforcement), isolation, durability, savepoints |
| `examples/02_isolation_levels/main.go` | Isolation level reference, optimistic locking with version column, deadlock pattern, transaction patterns summary |
| `exercises/01_order_saga/main.go` | Multi-table order saga: inventory deduction, payment, cancellation + refund — all atomic |

## Transaction pattern

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil { return err }
defer tx.Rollback()  // safe no-op after Commit

// ... multiple operations using tx ...

return tx.Commit()
```

## Optimistic locking

Use when contention is rare — avoids holding locks:

```go
// Read with version
var qty, version int
db.QueryRow(`SELECT qty, version FROM inventory WHERE id = ?`, id).Scan(&qty, &version)

// Compute new value, then update with version check
res, _ := db.Exec(
    `UPDATE inventory SET qty = ?, version = version + 1 WHERE id = ? AND version = ?`,
    newQty, id, version,
)
if n, _ := res.RowsAffected(); n == 0 {
    // Conflict — another writer modified the row; re-read and retry
}
```

## Savepoints

Partial rollback within a transaction (useful in loops or complex flows):

```sql
SAVEPOINT sp_before_risky
-- risky operation
ROLLBACK TO SAVEPOINT sp_before_risky  -- undoes the risky part, keeps outer tx open
RELEASE SAVEPOINT sp_before_risky
```

## Deadlock prevention

Always acquire locks in the same order across all code paths:

```go
// In a transfer: always lock lower ID first
if fromID > toID { fromID, toID = toID, fromID }
// Then SELECT ... FOR UPDATE on fromID, then toID
```

PostgreSQL will detect and abort one transaction with `SQLSTATE 40P01`. Retry that transaction.
