# Chapter 100 Checkpoint — Deployment Strategies

## Concepts to know

- [ ] What is the key difference between blue/green and canary deployment?
- [ ] What does "rollback time" mean for each strategy? Why does blue/green win here?
- [ ] What is a smoke test in the context of blue/green deployment?
- [ ] What is a metric gate in a canary rollout? Give two examples of metrics to gate on.
- [ ] What is a feature flag? How is it different from a canary deployment?
- [ ] When is blue/green the wrong choice? (Hint: state)
- [ ] What is Argo Rollouts? What is Flagger?
- [ ] What is `terminationGracePeriodSeconds` and why does it matter for zero-downtime deploys?
- [ ] What is a `PreStop` hook? When do you need it?

## Code exercises

### 1. Traffic splitter

Write a `TrafficSplitter` that:
- Routes requests to `v1` or `v2` based on a configurable weight (0–100)
- Is thread-safe
- Supports `SetWeight(pct int)` to change the split at runtime
- Counts requests per version

### 2. Canary gate

Write `EvaluateGate(metrics MetricSnapshot, gate GateCriteria) (pass bool, reason string)` where `GateCriteria` has `MaxErrorRatePct` and `MaxP99Ms`. Return a human-readable reason when the gate fails.

### 3. Rollback detector

Write a `RollbackDetector` that:
- Accepts a stream of error rates (one per second)
- Returns `true` if the rolling 60-second error rate exceeds a threshold
- Ignores the first 30 seconds (warmup period)

## Quick reference

```bash
# Kubernetes: check rollout status
kubectl rollout status deployment/my-app

# Instant rollback
kubectl rollout undo deployment/my-app

# Blue/green with kubectl
kubectl set image deployment/my-app-green app=myimage:v1.3
kubectl rollout status deployment/my-app-green
# After smoke tests:
kubectl patch service my-app -p '{"spec":{"selector":{"version":"green"}}}'

# Canary with Argo Rollouts
kubectl argo rollouts get rollout my-app --watch
kubectl argo rollouts promote my-app     # manual promote
kubectl argo rollouts abort my-app       # rollback
```

## Expected answers

1. Blue/green switches 100% of traffic at once (instant). Canary gradually shifts traffic in stages.
2. Rollback time: blue/green = seconds (flip LB back). Canary = seconds (set weight to 0%). Blue/green wins because no partial traffic state to clean up.
3. A smoke test is a minimal set of critical-path requests run against the green environment before cutting over, to verify it's functional.
4. Metric gates check error rate, p99 latency, success rate, etc. at each canary stage. Example gates: `error_rate < 0.1%`, `p99_latency < 200ms`.
5. A feature flag controls a feature at the application level (per-user, per-tenant). A canary controls which version of the binary serves traffic.
6. Blue/green is wrong when the new version has a database migration — you can't roll back the schema while green is writing to it.
7. Argo Rollouts and Flagger are Kubernetes controllers that automate canary and blue/green rollouts with metric-based promotion.
8. `terminationGracePeriodSeconds` (default 30s) is total time before SIGKILL. Must be > your shutdown drain time to avoid in-flight request drops.
9. `PreStop` hook runs before SIGTERM. Use it to deregister from service discovery (e.g., Consul) before connections are terminated.
