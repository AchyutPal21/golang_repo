# Capstone A — Scaling Discussion

## Current design limits

The in-process implementation handles ~50,000 redirects/sec on a single core. The bottleneck is the `sync.RWMutex` on the URL store, not the cache lookup.

## Path to 1 billion short links

### 1. ID generation at scale

The simple auto-increment counter works for a single node but becomes a bottleneck with multiple servers. Options:

| Approach | Pros | Cons |
|----------|------|------|
| DB sequence (Postgres `SERIAL`) | Guaranteed unique | Single DB write per shorten |
| Snowflake ID (time + node + seq) | Distributed, k-sorted | Requires node ID assignment |
| Random Base62 (6 chars = 56B combos) | No coordination | Collision check needed |

**Recommended:** Snowflake ID. Encode the 64-bit ID in Base62 → 7–8 char codes, globally unique, no DB round-trip.

### 2. Read path (redirect) — the hot path

Redirects vastly outnumber shortens (read:write ≈ 1000:1).

```
Client → CDN edge (cache 301/302 for popular codes)
          ↓ miss
       Redis cluster (in-memory lookup, sub-ms)
          ↓ miss
       PostgreSQL read replica (persistent store)
```

- Cache 301s at the CDN level for popular codes — eliminates most traffic before it hits your service.
- Use `302 Found` (not `301 Moved Permanently`) if you want analytics; browsers cache 301s aggressively.
- Redis cluster with consistent hashing across 6 nodes gives ~3M ops/sec.

### 3. Write path (shorten)

- Rate limit per user IP and per API key.
- Validate URL reachability asynchronously (HEAD request in background worker).
- Store in PostgreSQL with a partial index on `code` column.

```sql
CREATE TABLE short_urls (
    id        BIGINT PRIMARY KEY,
    code      VARCHAR(12) NOT NULL,
    long_url  TEXT NOT NULL,
    owner_id  UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    clicks    BIGINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX idx_short_urls_code ON short_urls (code);
```

### 4. Click tracking at scale

The buffered-channel approach works up to ~100K clicks/sec. Beyond that:

```
Redirect handler → Kafka topic "clicks"
                       ↓
                   Consumer group
                       ↓
                   Batch UPSERT into PostgreSQL (every 5s or 1000 events)
```

This decouples the hot redirect path from the write path completely. Click counts become eventually consistent (acceptable for analytics).

### 5. Custom domains

Users want `https://myco.com/abc` not `https://sho.rt/abc`. Implement:
- `custom_domains` table linking domain → owner
- TLS cert provisioning via ACME (Let's Encrypt) on first request
- SNI-based routing in the HTTP server

### 6. Kubernetes deployment

```yaml
# Two deployments: redirect (high replica) and admin (low replica)
redirect-service:  replicas: 20, CPU: 0.5, mem: 128Mi
admin-service:     replicas: 3,  CPU: 1,   mem: 256Mi

# Redis cluster via Helm
# PostgreSQL via managed service (RDS, Cloud SQL)
# CDN: Cloudflare or AWS CloudFront in front of redirect-service
```

### 7. Numbers

| Metric | Value |
|--------|-------|
| 1B links stored | ~200GB in Postgres (200 bytes/row average) |
| 10K redirects/sec | 3 Redis nodes, 5 redirect pods |
| 1M redirects/sec | CDN offloads 99%, ~10K req/sec reach origin |
| P99 redirect latency | < 5ms (cache hit), < 20ms (DB fallback) |

## What this capstone proved

- Repository pattern makes swapping in-memory → Redis → Postgres trivial
- Async click tracking decouples the hot path from write amplification
- Token bucket rate limiting prevents abuse at the handler layer
- Graceful shutdown drains the write buffer before exit — no lost clicks
