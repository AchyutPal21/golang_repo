# Capstone C — Scaling Discussion

## Saga vs two-phase commit (2PC)

Two-phase commit coordinates multiple resource managers (databases, queues) through a central coordinator. In the prepare phase every participant votes yes/no; in the commit phase the coordinator broadcasts the final decision. The guarantee is strong: all-or-nothing atomicity across services.

| | 2PC | Saga |
|--|-----|------|
| Consistency model | Strong (ACID across services) | Eventual |
| Coordinator failure | Blocks all participants | Each step is independent |
| Network partitions | Fatal — blocked until coordinator recovers | Steps proceed; compensations run when connectivity returns |
| Latency | High — two round-trips to every participant | Low — steps execute independently |
| Practical scope | Single database cluster, XA transactions | Microservices, cross-database workflows |

For e-commerce, 2PC is impractical because the payment gateway (Stripe, Braintree) and the inventory database are **different systems with no shared transaction coordinator**. Even if both exposed XA-compatible interfaces, a coordinator failure during checkout would lock inventory records for an unbounded period, blocking all orders.

The saga trades the atomic guarantee for **compensability**: every step that can succeed must also be reversible. The result is a workflow that makes forward progress under partial failure without holding locks.

## Why eventual consistency works for e-commerce

Three observations make eventual consistency acceptable in retail:

1. **Overselling is a business decision, not a fatal error.** Most retailers accept a small number of oversells during flash sales and handle them via backorder or refund. The saga can be designed to confirm first and cancel later rather than reject upfront.

2. **The customer experience is asynchronous anyway.** A customer does not expect instant physical delivery. Showing "Order placed — we will email you when shipped" tolerates a window where the order state is in-flight.

3. **Compensation is well-understood in finance.** Payment refunds, inventory returns, and order cancellations are normal business processes. Modelling them explicitly in code (the compensate functions) is cleaner than attempting to prevent them at the infrastructure level.

The critical invariant to protect is **payment without inventory** (customer is charged for something that cannot be shipped). The saga prevents this by always attempting inventory reservation before payment.

## Outbox pattern for saga events

A naive saga emits domain events (e.g. `OrderConfirmed`) directly to a message broker after committing to the database. This creates a dual-write problem: the database write can succeed while the broker publish fails, leaving consumers unaware.

The **outbox pattern** solves this:

```
BEGIN TRANSACTION
  INSERT INTO orders (id, status, ...) VALUES (...)
  INSERT INTO outbox (event_type, payload) VALUES ('OrderConfirmed', '...')
COMMIT

-- Separate relay process:
SELECT * FROM outbox WHERE published_at IS NULL ORDER BY created_at
  → publish each event to the broker
  → UPDATE outbox SET published_at = NOW() WHERE id = $1
```

The outbox row is written in the same transaction as the state change. A background relay process polls the outbox and publishes events at-least-once. Consumers must be idempotent (see below).

In PostgreSQL the relay can use `LISTEN/NOTIFY` triggered by an `AFTER INSERT` rule on the outbox table to avoid polling latency.

## Idempotency keys on payment

A payment service call can fail in a way where the outcome is unknown: the network dropped after the gateway processed the charge but before the response arrived. Retrying without an idempotency key would charge the customer twice.

```
// First attempt
POST /v1/charges
Idempotency-Key: order-42-charge-1
Body: { amount: 4999, currency: "usd", ... }

// Retry after network error — same key
POST /v1/charges
Idempotency-Key: order-42-charge-1   ← gateway deduplicates, returns original result
```

Stripe and most payment gateways accept an `Idempotency-Key` header. The key must be:
- **Unique per logical operation** (not per HTTP call) — use `orderID + attempt-number`
- **Short-lived** — gateways expire keys after 24 hours; keep retry windows well within that

In the simulation, `PaymentService.Charge` uses `orderID` as an in-process idempotency key: a second call for the same order is a no-op and returns nil.

## Inventory reservation TTL (hold-then-release)

A reservation that is never followed by a confirmation (because the browser tab was closed mid-checkout) permanently removes stock from available inventory. The fix is a **TTL-based hold**:

1. `Reserve` sets stock aside and records a `reserved_until` timestamp (e.g. now + 15 minutes).
2. A background job (or Redis TTL + keyspace notification) calls `Release` on expired reservations.
3. `Confirm` converts the hold into a permanent deduction by deleting the `reserved_until` field.

```sql
-- Reservation table
CREATE TABLE inventory_holds (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id  BIGINT NOT NULL REFERENCES products(id),
    order_id    BIGINT,
    quantity    INT NOT NULL CHECK (quantity > 0),
    held_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    released_at TIMESTAMPTZ,
    CONSTRAINT not_expired CHECK (released_at IS NOT NULL OR expires_at > now())
);

-- Available stock view
CREATE VIEW available_stock AS
SELECT
    p.id,
    p.total_stock - COALESCE(SUM(h.quantity), 0) AS available
FROM products p
LEFT JOIN inventory_holds h
    ON h.product_id = p.id
    AND h.released_at IS NULL
    AND h.expires_at  > now()
GROUP BY p.id;
```

The TTL approach enables optimistic checkout flows: a customer can reach the payment page knowing their items are held. If they abandon checkout, stock automatically returns after the hold expires.

## PostgreSQL schema for orders and inventory

```sql
-- Products
CREATE TABLE products (
    id           BIGSERIAL PRIMARY KEY,
    name         TEXT        NOT NULL,
    price_cents  INT         NOT NULL CHECK (price_cents > 0),
    total_stock  INT         NOT NULL CHECK (total_stock >= 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Inventory holds (reservation TTL — see above)
CREATE TABLE inventory_holds (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id  BIGINT      NOT NULL REFERENCES products(id),
    order_id    BIGINT,                          -- NULL until order is created
    quantity    INT         NOT NULL CHECK (quantity > 0),
    held_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    released_at TIMESTAMPTZ
);

CREATE INDEX ON inventory_holds (product_id, released_at, expires_at);

-- Orders
CREATE TABLE orders (
    id            BIGSERIAL   PRIMARY KEY,
    customer_id   TEXT        NOT NULL,
    status        TEXT        NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending','confirmed','failed')),
    total_cents   INT         NOT NULL CHECK (total_cents >= 0),
    idempotency_key TEXT      UNIQUE,           -- prevents duplicate submissions
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Order line items
CREATE TABLE order_items (
    id         BIGSERIAL PRIMARY KEY,
    order_id   BIGINT    NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id BIGINT    NOT NULL REFERENCES products(id),
    quantity   INT       NOT NULL CHECK (quantity > 0),
    unit_price_cents INT NOT NULL
);

CREATE INDEX ON order_items (order_id);

-- Outbox for saga domain events
CREATE TABLE outbox (
    id           BIGSERIAL   PRIMARY KEY,
    event_type   TEXT        NOT NULL,
    aggregate_id BIGINT      NOT NULL,          -- order ID
    payload      JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX ON outbox (published_at) WHERE published_at IS NULL;
```

Key design decisions:
- Prices are stored as integers (cents) to avoid floating-point drift. The simulation uses `float64` for readability, but production code should use `int64` or a `decimal` library.
- `idempotency_key` on orders prevents double-submission from the frontend (e.g. double-click on "Pay").
- The `outbox` table is written in the same transaction as the order state change, eliminating dual-write inconsistency.

## Kubernetes deployment notes

**Stateless pods.** The OrderSaga, InventoryService, and PaymentService hold no in-process state that cannot be reconstructed from the database. Horizontal pod autoscaling based on CPU or request queue depth is safe.

**Database connection pooling.** Each pod opens a PgBouncer sidecar or uses `pgx`'s built-in pool. Target `max_connections` per pod ≤ 10; scale out pods rather than database connections.

**Saga coordinator placement.** The saga runs inside the `order-service` pod. There is no separate saga coordinator process — the logic is embedded in the service. This is the **choreography** variant (each service knows its own compensation); the alternative is **orchestration** (a separate saga service drives the steps via commands).

**Leader election for the outbox relay.** Only one pod should poll and publish the outbox to avoid duplicate event delivery. Use a Kubernetes `Lease` object (via `client-go/coordination`) or a PostgreSQL advisory lock (`pg_try_advisory_lock`) to elect a single relay leader.

**Health probes.**

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 3
  periodSeconds: 5
```

`/readyz` should check database connectivity and return 503 if the connection pool is exhausted. This prevents the load balancer from routing traffic to a pod that cannot complete orders.

**Graceful shutdown.** On `SIGTERM`, stop accepting new orders, allow in-flight sagas to complete (or time out and compensate), drain the outbox relay, then exit. A `preStop` hook with a 15-second sleep gives the load balancer time to deregister the pod before connections are closed.
