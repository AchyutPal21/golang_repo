# Chapter 101 Exercises — Microservices vs Monolith

## Exercise 1 (provided): Migration Planner

Location: `exercises/01_migration_planner/main.go`

Full decomposition planning tool:
- `Package` struct: name, imports, imported-by, shared tables, change frequency, team owner
- `CouplingScore(pkg)` — 0–100 score (fan-in, fan-out, shared tables)
- `ExtractabilityScore(pkg)` — inverse of coupling; high = easier to extract
- `MigrationPlan` — ranked list of packages with cost/benefit estimates
- Decision framework: when to extract vs leave in monolith

## Exercise 2 (self-directed): Strangler Router

Build a `StranglerRouter`:
- `Register(path, handler string)` — register a path to "legacy" or "service-X"
- `Route(path string) string` — return which handler serves the path (exact or prefix match)
- `MigratedPct() float64` — percentage of paths NOT served by "legacy"
- `MigrationReport() []MigrationEntry` — each path with its handler and status

Acceptance criteria:
- Exact paths take precedence over prefix matches
- `MigratedPct()` returns 0.0 when all paths are on "legacy"
- Concurrent `Register` and `Route` calls pass `-race`

## Exercise 3 (self-directed): API Contract Checker

Build `CheckCompatibility(old, new APIContract) []BreakingChange`:
- `APIContract{Endpoints []Endpoint}`
- `Endpoint{Method, Path string; RequiredFields []string; Optional []string}`
- Breaking changes: removed endpoint, added required field, removed required field, changed method
- Non-breaking: added optional field, added new endpoint

Return a `BreakingChange{Type, Description string}` for each violation.

## Stretch Goal: Distributed Monolith Detector

Given a `ServiceGraph` (services + their cross-service DB queries):
- Flag pairs of services that share a database table
- Compute a "distributed monolith score" (0–100)
- Suggest which shared tables to refactor into owned APIs first
