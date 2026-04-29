# Chapter 30 — Exercises

## 30.1 — Add a new adapter

Run [`exercises/01_add_adapter`](exercises/01_add_adapter/main.go).

A `JSONAdapter` primary adapter drives the same `InventoryService` without modifying it.

Try:
- Add a `WebhookEventBus` driven adapter that prints events as HTTP POST bodies (simulate with `fmt.Printf`). Swap it in at the composition root — the service must not change.
- Add a `ReadonlyStore` adapter that wraps `memProductStore` and rejects all `Update` calls. Wire it to a second service instance and verify `Reserve` returns an error.
- Add a `MetricsEventBus` adapter that counts events by topic in a `map[string]int`. After running the adapter, print the counts.

## 30.2 ★ — Extract the domain layer

Take any existing example from Part II or Part III that mixes domain logic with infrastructure (print statements, map operations, etc.) and restructure it into the four-layer model:

1. Domain: pure structs + domain errors
2. Application: use-case service + port interfaces
3. Infrastructure: concrete adapters
4. Transport: `main()` composition root

No framework is allowed. The domain layer must have zero imports from other layers.

## 30.3 ★★ — Dual adapter

Build a `TaskService` that can be driven by both a CLI adapter and an in-memory "HTTP simulation" adapter (a struct with `Handle(method, path string)` that routes to use-case methods). Both adapters must work against the same `TaskService` instance in `main()`.
