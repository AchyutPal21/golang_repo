# Chapter 28 — Exercises

## 28.1 — Service wiring

Run [`exercises/01_service_wiring`](exercises/01_service_wiring/main.go).

`OrderService` depends on three injected interfaces: `ProductCatalog`, `PaymentGateway`, and `OrderStore`. All wiring happens in `main()` using stub implementations.

Try:
- Add a `Logger` dependency to `OrderService` that records every successful order; inject a `stdoutLogger` in main and a `fakeLogger` in test wiring.
- Add an `InventoryChecker` interface (`Reserve(productID string, qty int) error`). Wire it between price lookup and payment — if reservation fails, the order is aborted.
- Add a `Clock` dependency and stamp `Order.PlacedAt time.Time`; inject a fixed clock and assert the timestamp is deterministic.

## 28.2 ★ — Functional options server

Extend `examples/02_functional_options` with two new options:

1. `WithReadTimeout(d time.Duration) Option` — rejects `d == 0`
2. `WithRateLimit(rps int) Option` — rejects `rps <= 0`

Add a `productionPreset` option slice that bundles `WithTLS()`, `WithReadTimeout(30*time.Second)`, and `WithRateLimit(1000)`. Apply the preset via spread: `NewServer(productionPreset...)`.

## 28.3 ★★ — Layered composition root

Build a small application with three layers:

```
Repository layer  →  implements storage interfaces
Service layer     →  receives repositories via constructor
Handler layer     →  receives services via constructor
main()            →  wires all layers together (composition root)
```

Each layer must depend only on interfaces defined in its own package. `main()` is the only place where concrete types are imported across layers.
