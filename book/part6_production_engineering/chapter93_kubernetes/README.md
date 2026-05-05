# Chapter 93 — Kubernetes for Go Services

Kubernetes orchestrates containers at scale. Go services need to be designed around Kubernetes primitives: Deployments for rolling updates, ConfigMaps/Secrets for config, Services for discovery, and HPA for autoscaling.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | Deployment & manifests | Deployment, Service, ConfigMap, resource limits |
| 2 | HPA & probes | HorizontalPodAutoscaler, probe tuning, PodDisruptionBudget |
| E | K8s manifests | Complete production manifest set for a Go service |

## Examples

### `examples/01_manifests`

Simulates Kubernetes manifest generation and validation:
- `Deployment` with rolling update strategy
- `Service` with ClusterIP and NodePort
- `ConfigMap` for application config
- `HorizontalPodAutoscaler` targeting 70% CPU
- Resource request/limit calculation

### `examples/02_probes_hpa`

Probe configuration and HPA scaling logic:
- Probe timing interactions (startup → liveness → readiness)
- HPA desired replica calculation
- Pod disruption budget enforcement
- Rolling update availability guarantees

### `exercises/01_k8s_manifests`

Complete Go service manifest generator:
- Structured manifest types matching Kubernetes API
- YAML rendering without external dependencies
- Environment-driven configuration injection
- Resource limit recommendations based on app profile

## Key Concepts

**Deployment rolling update**
```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1          # +1 pod during update
    maxUnavailable: 0    # zero downtime
```

**Resource requests vs limits**
- `requests` — what Kubernetes guarantees and schedules on
- `limits` — maximum allowed; exceeding CPU → throttled; exceeding memory → OOMKilled

**Recommended starting values for a Go HTTP service**
```yaml
resources:
  requests:
    cpu: "100m"      # 0.1 vCPU
    memory: "64Mi"
  limits:
    cpu: "500m"      # 0.5 vCPU (burst)
    memory: "128Mi"  # hard limit — tune to actual usage
```

**HPA formula**
```
desiredReplicas = ceil(currentReplicas × currentMetric / targetMetric)
```

**PodDisruptionBudget** ensures at least N pods remain available during voluntary disruptions (node drain, rolling update).

## Running

```bash
go run ./book/part6_production_engineering/chapter93_kubernetes/examples/01_manifests
go run ./book/part6_production_engineering/chapter93_kubernetes/examples/02_probes_hpa
go run ./book/part6_production_engineering/chapter93_kubernetes/exercises/01_k8s_manifests
```
