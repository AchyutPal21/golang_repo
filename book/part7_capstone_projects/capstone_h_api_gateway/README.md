# Capstone H — API Gateway

A reverse-proxy API gateway with route matching, upstream load balancing, rate limiting, request/response transformation, circuit breaking, and auth enforcement — all implemented without external dependencies.

## What you build

- Route matching: `GET /api/orders/*` → `orders-service`
- Upstream selection: round-robin across healthy backends
- Rate limiting: per-API-key token bucket
- Circuit breaker: open/half-open/closed per upstream
- Request transformation: inject `X-Request-ID`, strip internal headers
- Auth enforcement: require valid API key for protected routes
- Health check: `/healthz` bypass (always forwarded)
- Metrics: request count, error rate, p99 latency per route

## Architecture

```
Client
  │
  ▼
Gateway Router
  ├── Auth Middleware      (API key → allowed routes)
  ├── Rate Limiter         (per API key, token bucket)
  ├── Request Transformer  (inject headers, strip internals)
  │
  ├── Route Table
  │     ├── /api/orders/* → OrdersUpstream
  │     ├── /api/users/*  → UsersUpstream
  │     └── /api/catalog/* → CatalogUpstream
  │
  └── Upstream (per service)
        ├── LoadBalancer    (round-robin)
        ├── CircuitBreaker  (per backend)
        └── ResponseMetrics (latency histogram)
```

## Key components

| Component | Pattern | Chapter ref |
|-----------|---------|-------------|
| Route trie | Prefix tree | Ch 58 |
| Round-robin LB | Atomic counter | Ch 46 |
| Circuit breaker | State machine | Ch 95 |
| Rate limiter | Token bucket | Ch 78 |
| Request ID | crypto/rand | Ch 97 |
| Metrics | Atomic counters + histogram | Ch 90 |

## Running

```bash
go run ./book/part7_capstone_projects/capstone_h_api_gateway
```

## What this capstone tests

- Can you implement a route table that handles prefix and wildcard matching?
- Can you combine circuit breaking with round-robin load balancing correctly?
- Can you track p99 latency without a metrics library?
- Can you enforce auth at the gateway layer without leaking it to upstreams?
