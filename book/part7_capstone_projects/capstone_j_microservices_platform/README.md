# Capstone J — Microservices Platform

The final capstone. Five services collaborating over a simulated message bus, with a service registry, health checks, distributed tracing correlation, and a platform layer that ties everything together. This is where every concept in the book converges.

## The five services

| Service | Responsibility |
|---------|---------------|
| **user-service** | User registration, profile lookup |
| **order-service** | Place orders, track status |
| **inventory-service** | Reserve/release product stock |
| **payment-service** | Charge/refund payments |
| **notification-service** | Send emails/SMS on events |

## Architecture

```
                    ┌─────────────────────────────┐
                    │       Platform Layer         │
                    │  ServiceRegistry  │  TraceID │
                    └──────────┬──────────────────-┘
                               │
          ┌────────────────────┼────────────────────┐
          │                    │                    │
   user-service         order-service         inventory-service
          │                    │                    │
          └────────────────────┼────────────────────┘
                               │
                    ┌──────────┴──────────┐
                    │    Message Bus      │  ← event-driven
                    │  (in-process pub/sub)│
                    └──────────┬──────────┘
                               │
                    ┌──────────┴──────────┐
                    │   payment-service   │
                    │ notification-service│
                    └─────────────────────┘
```

## Platform layer components

| Component | What it does | Chapter ref |
|-----------|-------------|-------------|
| ServiceRegistry | Service discovery, health tracking | Ch 95, 96 |
| MessageBus | Typed pub/sub with dead-letter queue | Ch 72, 80 |
| TraceContext | Request correlation across services | Ch 91 |
| CircuitBreaker | Per-service failure isolation | Ch 95 |
| RateLimiter | Per-service call budget | Ch 78 |

## Event flow (place order)

```
order-service.PlaceOrder(userID, productID, qty)
  → InventoryReserved event
     → payment-service charges
        → PaymentCharged event
           → order-service marks confirmed
              → OrderConfirmed event
                 → notification-service sends email
```

## Running

```bash
go run ./book/part7_capstone_projects/capstone_j_microservices_platform
```

## What this capstone tests

This is the complete integration of every Part V–VI concept:
- Can you wire 5 services together without coupling them directly?
- Can you trace a request across service boundaries with a shared trace ID?
- Can you handle partial failure (payment down) with compensation?
- Can you observe the platform through a single metrics dashboard?
