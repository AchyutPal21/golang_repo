# Capstone H — Scaling Discussion

## Why an API Gateway?

Without a gateway, every service must implement: auth, rate limiting, TLS termination, request ID injection, observability, and CORS. A gateway centralises these cross-cutting concerns so services stay focused on domain logic.

## Route matching performance

The prefix-sorted slice works for < 1000 routes. For larger route tables, use a radix/trie:

```
/api/orders       → orders-service
/api/orders/bulk  → orders-service (more specific, wins)
/api              → catch-all
```

A trie does O(path_length) lookups vs O(routes) for sorted prefix scan.

## Load balancing algorithms

| Algorithm | When to use |
|-----------|-------------|
| Round-robin | Homogeneous backends, uniform request cost |
| Weighted round-robin | Backends with different capacity (e.g. 8-core vs 4-core) |
| Least connections | Long-lived requests (gRPC streams, WebSockets) |
| Consistent hashing | Stateful backends (cache locality, sticky sessions) |

Round-robin is correct for stateless REST services. Switch to least-connections when request durations vary widely.

## Circuit breaker tuning

```
maxFailures = 5        # open after 5 consecutive failures
cooldown    = 30s      # wait before half-open probe
halfOpenMax = 1        # only 1 request probes during half-open
```

Per-backend circuit breakers (not per-upstream) allow partial degradation: if backends-1 is down, backends-2 and 3 still serve traffic normally.

## Connection pooling to upstreams

In a real implementation, the gateway maintains an HTTP/2 or gRPC connection pool to each backend:

```go
transport := &http.Transport{
    MaxIdleConnsPerHost: 100,
    IdleConnTimeout:     90 * time.Second,
    DisableCompression:  true, // backend already compresses
}
```

Without pooling, each proxy hop incurs a TCP + TLS handshake (~100ms).

## Rate limiting strategies

| Scope | Key | Use case |
|-------|-----|----------|
| Per API key | apiKey | Tenant quota enforcement |
| Per IP | IP | Abuse prevention |
| Per endpoint | apiKey+path | Expensive endpoint protection |
| Global | service name | Protect a weak downstream |

Store rate limit state in Redis (`INCR` + `EXPIRE`) for multi-instance gateways. In-process buckets are correct for single-node deployments.

## Request/response transformation

Common transformations at the gateway layer:
- Inject `X-Request-ID` (correlation for distributed tracing)
- Strip `X-Internal-*` headers (prevent header injection from clients)
- Add `X-Forwarded-For` (pass real client IP to backends)
- Rewrite paths: `/v1/orders` → `/api/orders` (API versioning at the edge)
- Response: inject `X-Response-Time`, `X-Cache` headers

## Kubernetes deployment

```yaml
gateway:
  replicas: 3
  resources: {cpu: "1", memory: "256Mi"}
  env:
    - UPSTREAM_TIMEOUT:      "30s"
    - RATE_LIMIT_REDIS_URL:  "redis://redis:6379"
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/path:   "/metrics"
```

Use `PodDisruptionBudget: minAvailable=2` so rolling updates never drop below 2 gateway replicas. The gateway is the critical path — treat it like a load balancer, not a regular service.

## Observability must-haves

Every request through the gateway should emit:
- Trace span: `gateway.route` with upstream name, route prefix, status code
- Metric: `gateway_requests_total{route, method, status}`
- Metric: `gateway_request_duration_seconds{route}` histogram
- Log line: `{level:"info", requestID, route, upstream, status, latencyMs}`
