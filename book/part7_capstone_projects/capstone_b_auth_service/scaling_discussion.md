# Capstone B — Scaling Discussion

## Access token vs refresh token: why two tokens?

| | Access Token | Refresh Token |
|--|-------------|---------------|
| Lifetime | 15 minutes | 7 days |
| Storage | Stateless (JWT, verified by signature) | Stateful (DB/Redis lookup required) |
| Revocation | Cannot revoke early (use short TTL instead) | Deleted on logout or rotation |
| Transmitted | Every API request (Authorization header) | Only to /refresh endpoint |

Short access tokens limit the blast radius of a leaked token to 15 minutes. Refresh tokens allow seamless re-issue without re-login.

## Token rotation (refresh token reuse detection)

Each `POST /refresh` call:
1. Validates and **deletes** the old refresh token (one-use).
2. Issues a new refresh token alongside the new access token.

If a stolen refresh token is replayed after legitimate rotation, the second use returns "not found" — alerting the system to invalidate the entire session family.

## Scaling the session store

The in-memory refresh store must move to Redis for multi-instance deployments:

```
SET refresh:{token} {userID} EX 604800   # 7 days TTL
GET refresh:{token}                       # lookup
DEL refresh:{token}                       # revoke
```

Redis `GETDEL` (atomic get + delete) implements the consume-on-use pattern without a race condition.

## bcrypt cost factor

```go
bcrypt.GenerateFromPassword([]byte(password), 12)
// cost=12 → ~300ms on modern hardware
// cost=10 → ~75ms
// cost=14 → ~1200ms
```

Pick the highest cost that keeps login latency under your SLA. Dedicated auth servers can afford cost=13–14. General API servers should use cost=10–11.

## MFA replay attack prevention

The `±1 window` drift tolerance means each TOTP code is valid for 90 seconds. To prevent replay:

```
redis.SET("totp_used:{userID}:{code}", "1", "EX", "90")
// Reject if key already exists
```

This blocks the same code from being used twice within its validity window.

## Kubernetes deployment

```yaml
auth-service:
  replicas: 3
  resources: {cpu: "500m", memory: "128Mi"}
  env:
    - JWT_SECRET: from SecretKeyRef
    - REDIS_URL:  from ConfigMap
  readinessProbe:
    httpGet: {path: /readyz, port: 8080}
```

Rate-limit `/login` and `/register` at the ingress layer (5 req/min per IP) to prevent credential stuffing.
