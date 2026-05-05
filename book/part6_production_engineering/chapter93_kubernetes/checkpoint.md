# Chapter 93 Checkpoint — Kubernetes for Go Services

## Concepts to know

- [ ] What is the difference between `requests` and `limits` for CPU and memory?
- [ ] What happens to a pod that exceeds its memory limit? Its CPU limit?
- [ ] What is a `RollingUpdate` strategy? What do `maxSurge` and `maxUnavailable` control?
- [ ] What is the HPA desired-replica formula?
- [ ] What is a `PodDisruptionBudget`? When does it apply?
- [ ] What is the difference between `ClusterIP`, `NodePort`, and `LoadBalancer` service types?
- [ ] What is `terminationGracePeriodSeconds`? How does it interact with graceful shutdown in Go?
- [ ] What should `readinessProbe.failureThreshold` be set to? Why?
- [ ] What is a `ConfigMap`? When do you use a `Secret` instead?

## Code exercises

### 1. Resource calculator

Write a function `recommendResources(rps int, avgLatencyMs int) (cpuRequest, memRequest string)` that estimates CPU and memory requests for a Go HTTP service.

### 2. HPA simulation

Given: 3 current replicas, current CPU = 85%, target CPU = 70%.
Calculate: desired replica count using the HPA formula.

### 3. Probe configuration

An application takes 20 seconds to start. Write the `startupProbe`, `livenessProbe`, and `readinessProbe` configuration (as Go structs or YAML) that:
- Allows 30 seconds for startup before liveness kicks in
- Checks liveness every 10 seconds with 3 retries
- Checks readiness every 5 seconds

## Quick reference

```bash
# Apply manifests
kubectl apply -f deploy/

# Watch rollout
kubectl rollout status deployment/my-app

# Scale manually
kubectl scale deployment/my-app --replicas=5

# View HPA status
kubectl get hpa

# Pod resource usage
kubectl top pods

# Drain node safely (respects PDB)
kubectl drain node-1 --ignore-daemonsets --delete-emptydir-data

# View pod logs
kubectl logs -l app=my-app --tail=100 -f
```

## Expected answers

1. `requests` is what the scheduler uses to place pods; `limits` caps usage. A pod always gets its request; it can burst to its limit.
2. Memory exceeded: OOMKilled (pod restarted). CPU exceeded: throttled (not killed).
3. RollingUpdate replaces pods incrementally. `maxSurge`: extra pods allowed above desired count. `maxUnavailable`: pods that can be down simultaneously.
4. `desiredReplicas = ceil(currentReplicas × currentMetric / targetMetric)`.
5. PDB ensures minimum availability during voluntary disruptions. It blocks `kubectl drain` if draining would violate the budget.
6. ClusterIP: internal only. NodePort: external via node IP + port. LoadBalancer: external via cloud load balancer.
7. `terminationGracePeriodSeconds` (default 30s) — total time Kubernetes waits before SIGKILL. Your Go server.Shutdown timeout must be less than this.
8. `failureThreshold: 2` or `3`. Lower means faster traffic removal; too low causes flapping during GC pauses.
9. ConfigMap: non-sensitive config (feature flags, URLs). Secret: sensitive data (passwords, tokens) — stored base64-encoded in etcd, mountable as env or volume.
