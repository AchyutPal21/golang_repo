# Chapter 100 Exercises — Deployment Strategies

## Exercise 1 (provided): Deploy Strategy Simulator

Location: `exercises/01_deploy_strategy/main.go`

Full deployment strategy simulation:
- Risk scorer: maps change attributes to strategy recommendation
- Blue/green simulation: parallel environments, smoke tests, atomic cutover
- Canary simulation: progressive stages with configurable metric gates
- Failure injection: randomly fail a canary stage to trigger rollback
- Decision framework: when to use each strategy

## Exercise 2 (self-directed): Traffic Splitter

Build a production-grade traffic splitter:
- `Split(weight int) bool` — returns true if request should go to canary (weight = % to canary)
- Deterministic for the same `user_id` (consistent hashing so a user always sees the same version)
- Thread-safe weight updates via `SetWeight(n int)`
- Tracks: requests routed to stable, requests routed to canary, weight changes

Acceptance criteria:
- `Split` is consistent per user_id across calls
- `SetWeight(0)` routes 100% to stable immediately
- Concurrent `Split` and `SetWeight` calls pass `-race`

## Exercise 3 (self-directed): Canary Metric Gate

Build a `MetricGate` system:
- `Gate{Name string, Check func(MetricSnapshot) error}` — a named gate with a check function
- `Pipeline` — ordered list of gates, all must pass
- Run 5 simulated metric snapshots through the pipeline
- On gate failure: return the gate name + reason
- Implement built-in gates: `ErrorRateGate(max float64)`, `LatencyP99Gate(maxMs float64)`

## Stretch Goal: Feature Flag Service

Build a simple in-memory feature flag service:
- `FlagStore` with `Set(flag, rule string)` and `Evaluate(flag, userID string) bool`
- Rule DSL: `"enabled"`, `"disabled"`, `"rollout:10"` (10% of users), `"allow:user42,user99"`
- Deterministic: the same user always gets the same answer for `rollout:N`
- Thread-safe: multiple goroutines reading/writing simultaneously
