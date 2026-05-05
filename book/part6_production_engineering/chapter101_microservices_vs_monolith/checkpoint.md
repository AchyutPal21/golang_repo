# Chapter 101 Checkpoint — Microservices vs Monolith

## Concepts to know

- [ ] What is a "distributed monolith"? Why is it worse than either a real monolith or real microservices?
- [ ] What is the strangler fig pattern? Walk through its three phases.
- [ ] What is the "data ownership rule" for microservices? Why does cross-service DB access break it?
- [ ] Name three signals that a monolith module is ready to be extracted as a service.
- [ ] Name three signals that microservices are the wrong choice for a team.
- [ ] What is an anti-corruption layer (ACL)? When do you need one?
- [ ] What is a saga? How does it replace a database transaction across services?
- [ ] What does "API contract" mean between services? What breaks backward compatibility?
- [ ] What is Conway's Law? How does it affect architecture decisions?
- [ ] What operational costs do microservices add that a monolith doesn't have?

## Code exercises

### 1. Dependency scorer

Write `func CouplingScore(pkg Package) int` where `Package` has:
- `Imports []string` — packages this one imports
- `ImportedBy []string` — packages that import this one
- `SharedTables []string` — DB tables touched

Score 0–100: higher means harder to extract. Penalize: high fan-in, high fan-out, shared tables.

### 2. Strangler router

Write a `StranglerRouter` that:
- Has `Register(path string, handler string)` — routes a URL path to "legacy" or "new-service"
- Has `Route(path string) string` — returns which handler serves the path
- `MigratedPct() float64` — what percentage of paths are now on "new-service"

### 3. API contract checker

Write `CheckCompatibility(old, new APIContract) []BreakingChange` where `APIContract` has:
- `Endpoints []Endpoint` — each with `Method, Path, RequiredFields []string`
- Breaking changes: removed endpoint, removed required field, changed method

## Quick reference

```
Monolith first. Extract when you feel the pain.
Pain signals: merge conflicts, independent scaling needs, team ownership clashes.

Strangler fig phases:
  1. Add abstraction (interface) in front of the module
  2. Deploy new service behind the interface (dark launch)
  3. Flip traffic to new service; delete old code

Data ownership:
  Orders service → owns `orders`, `order_items` tables
  Billing service → owns `invoices`, `payments` tables
  Never: SELECT * FROM billing.payments FROM orders service

Saga pattern:
  step 1: reserve inventory → publish InventoryReserved event
  step 2: charge payment    → on success: publish PaymentCharged
                              on failure: publish PaymentFailed → compensate step 1
```

## Expected answers

1. A distributed monolith has microservice infrastructure (separate deployments, network calls) but still shares a database and has tight coupling — you pay all microservice costs with none of the benefits.
2. Strangler fig: (1) add an interface in front of the legacy module; (2) build a new service implementing that interface; (3) migrate traffic to the new service and delete the legacy code.
3. Each service owns its own schema; others access data via API only. Cross-service DB queries create hidden coupling — a schema change in one service breaks another silently.
4. Ready-to-extract signals: dedicated team ownership, stable and well-understood API boundary, independent scaling needs.
5. Wrong choice signals: team < 10, domain not yet understood, no independent scaling requirement.
6. ACL translates between two different domain models at a service boundary, preventing the upstream model from "leaking" into the downstream service.
7. A saga replaces a ACID transaction with a sequence of local transactions + compensating transactions on failure. Each step publishes an event; failure triggers a rollback event handled by prior steps.
8. API contract: the set of endpoints, methods, request/response fields a service exposes. Breaking changes: removing an endpoint, removing a required field, changing a field type.
9. Conway's Law: systems mirror the communication structure of the organizations that build them. Align service boundaries with team boundaries, not the other way around.
10. Microservice costs: network latency, distributed tracing, service discovery, partial failure handling, distributed transactions, multiple CI/CD pipelines, API versioning.
