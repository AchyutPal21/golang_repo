# Capstone B — Auth Service

A standalone authentication and authorisation service. Covers the full session lifecycle: registration, login, JWT access tokens, opaque refresh tokens, MFA (TOTP), and RBAC policy enforcement.

## What you build

- `POST /register` — create user account (bcrypt password hash)
- `POST /login` — verify credentials → issue access token (15m) + refresh token (7d)
- `POST /refresh` — exchange valid refresh token for new access token
- `POST /logout` — revoke refresh token
- `POST /mfa/setup` — generate TOTP secret + QR URI
- `POST /mfa/verify` — verify TOTP code, upgrade session
- `GET  /me` — return claims from access token
- RBAC middleware: `RequireRole("admin")`, `RequirePermission("orders:write")`

## Architecture

```
Client
  │
  ▼
Auth Handler
  ├── PasswordHasher (bcrypt, cost=12)
  ├── TokenIssuer    (HMAC-SHA256 JWT)
  ├── RefreshStore   (opaque token → user mapping, Redis/DB)
  ├── TOTPService    (RFC 6238, 30s window, ±1 drift tolerance)
  └── RBACPolicy     (role → permissions map)
```

## Key components

| Component | Pattern | Chapter ref |
|-----------|---------|-------------|
| bcrypt hashing | Adaptive cost factor | Ch 97 |
| JWT (HMAC-SHA256) | Stateless access token | Ch 60 |
| Opaque refresh tokens | crypto/rand, DB-backed | Ch 60 |
| TOTP | RFC 6238, HMAC-SHA1 | Ch 97 |
| RBAC | Role → permission set | Ch 61 |
| Token revocation | Refresh token store | Ch 60 |

## Running

```bash
go run ./book/part7_capstone_projects/capstone_b_auth_service
```

## What this capstone tests

- Can you implement the full access/refresh token rotation pattern?
- Can you implement TOTP without an external library?
- Can you enforce RBAC at the middleware layer cleanly?
- Do you understand why short-lived access tokens + long-lived refresh tokens is the right model?
