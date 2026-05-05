# Chapter 93 Exercises — Kubernetes for Go Services

## Exercise 1 (provided): K8s Manifest Generator

Location: `exercises/01_k8s_manifests/main.go`

Generates a complete Kubernetes manifest set:
- `Deployment` with rolling update, resource limits, probes
- `Service` (ClusterIP)
- `HorizontalPodAutoscaler` (CPU target 70%)
- `PodDisruptionBudget` (minAvailable: 1)
- YAML rendering via formatted strings

## Exercise 2 (self-directed): Rolling Update Simulator

Build a rolling update simulator:
- Start with N pods running version "v1"
- Simulate a rolling update to "v2" with `maxSurge=1` and `maxUnavailable=0`
- Print the pod state at each step (which are running, which are terminating, which are starting)
- Verify that at least N pods are always available during the update
- Include a readiness check delay: new pods take 2 "ticks" to become ready

## Exercise 3 (self-directed): HPA Autoscaler

Build an HPA simulator:
- Start with 2 replicas at 0% CPU
- Feed CPU load in 10% increments every tick
- Calculate desired replicas using the HPA formula at each tick
- Apply a cooldown (3 ticks after scale-up before scaling down)
- Print a timeline showing: tick, CPU%, current replicas, desired replicas, action

## Stretch Goal: ConfigMap Hot-Reload

Simulate a ConfigMap hot-reload:
- Store config in a `sync.Map` (key=string, value=string)
- Start a "watcher" goroutine that polls for config changes every second
- Expose a `GetConfig(key string) string` function that always returns the latest value
- Demonstrate updating a feature flag without restarting the service
