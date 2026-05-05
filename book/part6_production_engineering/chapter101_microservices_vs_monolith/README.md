# Chapter 101 — Microservices vs Monolith

The "microservices vs monolith" question is not a technical debate — it is an operational cost trade-off. Start with a well-structured monolith. Extract services only when a specific bottleneck or team-ownership problem makes the extraction cost worth it.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | Monolith patterns | Module boundaries, domain packages, shared DB, anti-patterns to avoid |
| 2 | Service decomposition | Strangler fig, seam identification, API contracts, data ownership |
| E | Migration planner | Score a monolith's readiness; produce a decomposition roadmap |

## Examples

### `examples/01_monolith_patterns`

A well-structured monolith in Go:
- Domain packages as the primary boundary (`order`, `catalog`, `billing`)
- Package-level interfaces preventing circular imports
- In-process calls vs RPC overhead comparison
- Anti-pattern catalogue: distributed monolith, chatty services, shared schema

### `examples/02_service_decomposition`

Strangler-fig pattern and seam detection:
- `Seam` analysis: which packages are independently deployable today?
- Dependency graph rendering (adjacency list)
- Strangler fig: route subset of traffic to new service while monolith handles the rest
- Data ownership: one service owns each table; others call its API

### `exercises/01_migration_planner`

Migration readiness scorer:
- Score a package on coupling, change frequency, team ownership, data isolation
- Produce a ranked decomposition roadmap
- Estimate migration cost vs benefit

## Key Concepts

**When to stay monolith**
```
✓ Team < 10 engineers
✓ Domain not yet stable (still discovering boundaries)
✓ Deployment is already fast (< 10 min)
✓ All modules deploy together without issue
✓ No single module needs independent scaling
```

**When to extract a service**
```
✓ Module has a different scaling profile (e.g., image processing vs API)
✓ Module needs a different technology (e.g., ML model serving)
✓ Module is owned by a dedicated team and causes merge conflicts
✓ Module has a well-understood, stable API boundary
✓ Module's deployment cycle is bottlenecked by the monolith release train
```

**Decomposition patterns**

| Pattern | Description | When to use |
|---------|-------------|-------------|
| Strangler fig | Route new traffic to service; legacy handles rest | Incremental, low-risk |
| Branch by abstraction | Add interface, swap impl behind it | Easier to test boundary |
| Anti-corruption layer | Translate between old and new domain models | When models diverge |
| Saga | Distributed transactions via events | Replacing ACID with eventual consistency |

**Data ownership rule**
```
One service = one schema.
No direct cross-service DB queries.
Services communicate via API calls or events.
Violating this creates a "distributed monolith" — worst of both worlds.
```

**Distributed system costs you take on**
```
• Network latency (in-process μs → cross-service ms)
• Distributed tracing required (chapter 91)
• Service discovery and load balancing
• Partial failure modes (circuit breakers, retries, timeouts)
• Distributed transactions (saga pattern, eventual consistency)
• Multiple deployment pipelines to maintain
• API versioning and backward compatibility
• Operational complexity: N services × all of the above
```

## Running

```bash
go run ./book/part6_production_engineering/chapter101_microservices_vs_monolith/examples/01_monolith_patterns
go run ./book/part6_production_engineering/chapter101_microservices_vs_monolith/examples/02_service_decomposition
go run ./book/part6_production_engineering/chapter101_microservices_vs_monolith/exercises/01_migration_planner
```
