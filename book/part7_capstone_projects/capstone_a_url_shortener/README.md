# Capstone A — URL Shortener

A production-grade URL shortener built entirely in Go. This project ties together HTTP routing, database persistence, Redis caching, rate limiting, graceful shutdown, and deployment — all patterns covered in Parts IV–VI.

## What you build

A service where:
- `POST /shorten` accepts a long URL and returns a short code (e.g. `https://sho.rt/xK9mP2`)
- `GET /:code` redirects to the original URL (301 or 302)
- `GET /stats/:code` returns click count, referrer breakdown, creation time
- `DELETE /shorten/:code` removes a link (owner-authenticated)

## Architecture

```
Client
  │
  ▼
HTTP Handler (net/http)
  │
  ├── Rate Limiter (token bucket per IP)
  │
  ├── URL Store (interface)
  │     ├── PostgreSQL  (source of truth)
  │     └── Redis cache (L1, TTL=1h)
  │
  ├── Click Tracker (async, buffered channel → batch DB write)
  │
  └── Auth Middleware (HMAC-signed owner token)
```

## Key components

| Component | Pattern | Chapter ref |
|-----------|---------|-------------|
| Short code generation | Base62 encoding of auto-increment ID | Ch 9 |
| URL store interface | Repository pattern | Ch 34 |
| Redis cache layer | Read-through, stampede protection | Ch 71 |
| Rate limiter | Token bucket per IP | Ch 78 |
| Click tracker | Buffered channel + batch flush | Ch 48 |
| Graceful shutdown | context cancel + drain | Ch 56 |
| Health/ready probes | `/healthz`, `/readyz` | Ch 92 |
| Structured logging | log/slog | Ch 63 |

## Running

```bash
# Simulation (no external deps):
go run ./book/part7_capstone_projects/capstone_a_url_shortener

# Full stack (requires Docker):
docker-compose up
```

## Project layout

```
capstone_a_url_shortener/
├── main.go              ← self-contained simulation
├── README.md
└── scaling_discussion.md
```

## What this capstone tests

- Can you design a clean repository interface and swap implementations?
- Can you protect a hot path with an in-process cache and avoid stampedes?
- Can you drain an async write buffer before shutdown without data loss?
- Can you write a Base62 encoder that produces short, URL-safe codes?
- Can you make the `/redirect` path sub-millisecond with Redis?
