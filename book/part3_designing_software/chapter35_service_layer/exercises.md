# Chapter 35 — Exercises

## 35.1 — Checkout service with compensation

Run [`exercises/01_checkout_service`](exercises/01_checkout_service/main.go).

`CheckoutService` coordinates three ports with compensating rollback on failure.

Try:
- Add a `NotificationPort` that sends an order confirmation email. It should be called after the charge succeeds but must not block the checkout if it fails.
- Add an `OrderRepository` port. After a successful charge, save a `Receipt` and return its `OrderID`. On save failure, refund the charge.
- Add a `quantity limit` validation: no single item can have `Qty > 10`. Return a validation error before touching any port.

## 35.2 ★ — Retry with idempotency

Extend `SubscriptionService` from example 02. Add a `RetrySubscribe` method that retries up to 3 times on transient payment errors (simulate with a payment gateway that fails the first 2 calls, succeeds on the 3rd). The idempotency key ensures no double charge even if the caller retries independently.

## 35.3 ★★ — Saga pattern

Build a `ShipmentSaga` that coordinates:
1. `Reserve` inventory
2. `CreateShipment` with a shipping provider
3. `DeductStock` permanently

If step 3 fails after step 2 succeeds, cancel the shipment. Model each step as a `SagaStep` interface with `Execute()` and `Compensate()`. Run the saga with a generic `Runner` that calls `Compensate` in reverse order on any failure.
