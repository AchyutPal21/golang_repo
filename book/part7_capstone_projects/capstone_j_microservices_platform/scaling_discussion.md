# Capstone J — Scaling Discussion

## From in-process bus to Kafka

The `MessageBus` here is synchronous and in-process. For real microservices replace it with Kafka:

```
order-service  →  Kafka topic "order.placed"
                      ↓
                inventory-service (consumer group: inventory)
                      ↓
                Kafka topic "inventory.reserved"
                      ↓
                payment-service (consumer group: payment)
```

Each service is a Kafka consumer group. Horizontal scaling is free — add instances, Kafka rebalances partitions. The message bus interface stays the same; only the implementation changes.

## Service-to-service auth

In production, every inter-service call must be authenticated. Two patterns:

| Pattern | How | When |
|---------|-----|------|
| mTLS | Each service has a certificate; peer verifies it | East-west traffic in same cluster |
| JWT service token | Short-lived JWT signed by auth-service; passed in `Authorization` | Cross-cluster or external calls |

Kubernetes service meshes (Istio, Linkerd) can enforce mTLS transparently without code changes.

## Distributed tracing in production

Replace `TraceContext` with OpenTelemetry (chapter 91):

```go
ctx, span := tracer.Start(ctx, "order-service.PlaceOrder")
defer span.End()
// Propagate via W3C traceparent header on HTTP calls
```

The trace spans appear in Jaeger or Tempo, showing exactly where time was spent across all 5 services for a single request.

## Data ownership

Each service owns its own database. The shared domain types (`Order`, `Product`) are duplicated by design — services have their own projections:

```
order-service DB:    orders table (id, user_id, status, total)
inventory DB:        products table (id, stock)
payment DB:          payments table (id, order_id, amount, status)
notification DB:     delivery_log table (id, user_id, channel, sent_at)
```

No JOINs across service databases. Reporting that needs a cross-service view uses an event-sourced read model or a dedicated analytics DB fed by Kafka.

## Saga compensation flow

```
PlaceOrder
  ├── ReserveInventory (compensate: ReleaseInventory)
  ├── ChargePayment    (compensate: RefundPayment)
  └── ConfirmOrder     (no compensation — terminal state)

Failure at ChargePayment:
  → RefundPayment skipped (charge never happened)
  → ReleaseInventory fires  ← compensation
  → OrderFailed event published
```

All compensations are idempotent — safe to retry if the compensating action itself fails.

## Kubernetes deployment (all 5 services)

```yaml
# Each service gets its own Deployment + Service
user-service:         replicas: 2,  cpu: "250m", mem: "64Mi"
order-service:        replicas: 3,  cpu: "500m", mem: "128Mi"
inventory-service:    replicas: 2,  cpu: "250m", mem: "64Mi"
payment-service:      replicas: 2,  cpu: "500m", mem: "128Mi"
notification-service: replicas: 2,  cpu: "250m", mem: "64Mi"

# Shared infrastructure
kafka:        3-broker cluster (Helm: bitnami/kafka)
postgres:     1 cluster per service (or schemas on shared instance for small teams)
redis:        1 cluster (rate limiting, caching)
jaeger:       1 instance (distributed tracing)
prometheus:   1 instance + grafana dashboard
```

## What the platform summary proves

After completing all 10 capstones, a reader can:
1. Design a service boundary that avoids a distributed monolith (Capstone J)
2. Build each service with production-grade internals (Capstones A–I)
3. Operate the platform: deploy (Ch 100), observe (Ch 91/90), respond to incidents (Ch 98)
4. Scale independently: each service has its own replica count and resource profile

The full stack — 5 services, 1 message bus, 1 registry, distributed tracing — runs in a single `go run` with zero infrastructure. That's the power of writing Go from first principles.
