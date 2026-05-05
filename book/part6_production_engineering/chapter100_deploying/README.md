# Chapter 100 — Deployment Strategies

Getting code to production safely is a discipline. Rolling updates (chapter 93) are the baseline. Blue/green and canary deployments give you faster rollbacks and risk-controlled rollouts respectively.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | Blue/Green | Instant cutover, full parallel environment, zero-downtime swap |
| 2 | Canary | Traffic splitting, metric gates, progressive rollout |
| E | Deploy strategy | Decision framework + simulation of both strategies |

## Examples

### `examples/01_blue_green`

Blue/green deployment simulation:
- Two environments (blue=active, green=staging)
- Smoke test before cutover
- Atomic traffic switch (DNS / load balancer weight)
- Instant rollback: swap back in < 1 minute
- Cost: 2× infrastructure during deployment window

### `examples/02_canary`

Canary deployment with automatic promotion/rollback:
- Traffic split: 5% → 25% → 50% → 100%
- Metric gates at each stage: error rate, p99 latency
- Automatic rollback if gate fails
- Manual override: pause / force-promote / abort
- Comparison: canary vs baseline side-by-side

### `exercises/01_deploy_strategy`

Deployment strategy decision engine:
- Risk scoring: change size, test coverage, traffic sensitivity
- Recommends: rolling / blue-green / canary / feature flag
- Simulates a full canary rollout with configurable failure injection

## Key Concepts

**Deployment strategy comparison**

| Strategy | Rollback time | Risk | Cost | Best for |
|----------|--------------|------|------|---------|
| Rolling update | Minutes | Medium | 1× | Stateless services, low risk |
| Blue/green | Seconds | Low | 2× | Database migrations, instant rollback needed |
| Canary | Seconds | Very low | 1.1× | High-traffic, risk-averse changes |
| Feature flag | Instant | Very low | 1× | Per-user gradual rollout |

**Blue/green flow**
```
      Load Balancer
          │
    ┌─────┴─────┐
    │           │
  Blue        Green
(v1.2, live) (v1.3, staging)
    │
    ▼ smoke tests pass → switch LB → green becomes live
    
Rollback: switch LB back to blue (< 60s)
```

**Canary stages**
```
Stage 1: 5%  traffic → v1.3   (monitor 5min, gate: error_rate < 0.1%)
Stage 2: 25% traffic → v1.3   (monitor 10min, gate: p99 < 200ms)
Stage 3: 50% traffic → v1.3   (monitor 15min)
Stage 4: 100% traffic → v1.3  (rollout complete)

Any gate fail → automatic rollback to 0%
```

**Kubernetes traffic splitting**
```yaml
# With Argo Rollouts or Flagger
apiVersion: argoproj.io/v1alpha1
kind: Rollout
spec:
  strategy:
    canary:
      steps:
      - setWeight: 5
      - pause: {duration: 5m}
      - setWeight: 25
      - analysis:
          templates: [{templateName: error-rate-check}]
      - setWeight: 100
```

## Running

```bash
go run ./book/part6_production_engineering/chapter100_deploying/examples/01_blue_green
go run ./book/part6_production_engineering/chapter100_deploying/examples/02_canary
go run ./book/part6_production_engineering/chapter100_deploying/exercises/01_deploy_strategy
```
