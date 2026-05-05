# Capstone C — E-commerce Backend

A production-grade e-commerce backend built entirely in Go. This project ties together the saga pattern for distributed transactions, inventory management, payment processing, and order lifecycle — patterns covered in Parts IV–VI.

## What you build

A backend system where:
- A **product catalog** holds items with prices and stock levels
- A **cart** lets a user add/remove items and compute totals
- **PlaceOrder** runs a three-step saga: reserve inventory, charge payment, confirm order
- If any step fails, **compensating actions** execute in reverse order (saga rollback)
- An **order store** tracks order state through its lifecycle
- Three scenarios are simulated: happy path, failed payment, insufficient stock

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Simulation                           │
│                                                             │
│   Cart ──► OrderSaga ──► InventoryService                   │
│               │               │  Reserve(productID, qty)    │
│               │               │  Release(productID, qty)    │
│               │                                             │
│               ├──────────────► PaymentService               │
│               │                  Charge(orderID, amount)    │
│               │                  Refund(orderID)            │
│               │                                             │
│               └──────────────► OrderStore                   │
│                                  Create / UpdateStatus      │
│                                                             │
│   ProductCatalog (in-memory, sync.RWMutex)                  │
│   InventoryService (stock map, sync.Mutex)                  │
│   PaymentService   (ledger map, sync.Mutex)                 │
│   OrderStore       (order map, sync.Mutex)                  │
│   IDs via sync/atomic (uint64 counter)                      │
└─────────────────────────────────────────────────────────────┘

Saga step sequence:
  1. ReserveInventory  ◄── compensate: ReleaseInventory
  2. ChargePayment     ◄── compensate: RefundPayment
  3. ConfirmOrder      ◄── compensate: CancelOrder (terminal, no retry)

On step N failure: compensate N-1, N-2, … in reverse order.
```

## Key components

| Component | Pattern | Chapter ref |
|-----------|---------|-------------|
| `ProductCatalog` | In-memory repository with `sync.RWMutex` | Ch 34, Ch 43 |
| `Cart` | Value-object accumulator, thread-safe copy-on-write | Ch 9 |
| `OrderSaga` | Saga pattern, reverse-order compensation | Ch 96 |
| `InventoryService` | Pessimistic reservation with compensating release | Ch 96 |
| `PaymentService` | Idempotent charge with compensating refund | Ch 97 |
| `OrderStore` | State machine (Pending → Confirmed / Failed) | Ch 34 |
| Atomic ID generation | `sync/atomic` monotonic counter | Ch 43 |
| Thread safety | `sync.Mutex` / `sync.RWMutex` throughout | Ch 43 |

## Project layout

```
capstone_c_ecommerce_backend/
├── main.go               ← self-contained simulation
├── README.md
└── scaling_discussion.md
```

## Running

```bash
# Simulation (no external deps):
go run ./book/part7_capstone_projects/capstone_c_ecommerce_backend

# Or build and run:
go build ./book/part7_capstone_projects/capstone_c_ecommerce_backend
./capstone_c_ecommerce_backend
```

Expected output covers three scenarios:

```
=== Scenario 1: Successful order ===
[Saga] Step 1/3: ReserveInventory
[Saga] Step 2/3: ChargePayment
[Saga] Step 3/3: ConfirmOrder
Order #1 status: confirmed

=== Scenario 2: Payment failure ===
[Saga] Step 1/3: ReserveInventory
[Saga] Step 2/3: ChargePayment  ← fails
[Saga] Compensating step 1: ReleaseInventory
Order #2 status: failed

=== Scenario 3: Insufficient stock ===
[Saga] Step 1/3: ReserveInventory  ← fails
Order #3 status: failed
```

## What this capstone tests

- Can you implement the saga pattern with correct reverse compensation?
- Can you design service interfaces (InventoryService, PaymentService) that are independently testable?
- Can you keep shared state thread-safe under concurrent access?
- Can you model an order state machine so invalid transitions are impossible?
- Can you simulate distributed failure modes without external infrastructure?
- Do you understand why 2PC is impractical here but saga is viable?
