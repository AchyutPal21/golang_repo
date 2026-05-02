# Chapter 60 — Authentication

## What you will learn

- Password hashing with `golang.org/x/crypto/bcrypt` — cost factor, why bcrypt is slow by design
- Session-based authentication: session ID generation with `crypto/rand`, in-memory session store, server-side session deletion on logout
- `Set-Cookie` flags: `HttpOnly`, `SameSite`, `Secure`, `MaxAge`, `Path`
- JWT structure: Base64URL-encoded header + payload + HMAC-SHA256 signature
- JWT claims: `sub`, `iat`, `exp`, `type` ("access" vs "refresh")
- Short-lived access tokens (15 min) + long-lived refresh tokens (7 days)
- Refresh token rotation: single-use — consumed on exchange, new refresh token issued
- JWT validation middleware: signature check, expiry check, type assertion
- Choosing between sessions and JWTs: server state vs stateless, revocation, scalability

---

## bcrypt password hashing

```go
import "golang.org/x/crypto/bcrypt"

// Hash on registration:
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// Verify on login:
err := bcrypt.CompareHashAndPassword(hash, []byte(password))
// err == nil → password matches
```

`bcrypt.DefaultCost` = 10 (about 100ms per hash). Increase to 12–14 for higher security. Never use `MD5`, `SHA1`, or `SHA256` for passwords — they are too fast.

---

## Session authentication flow

```
POST /login  →  verify password  →  create session  →  Set-Cookie: session_id=<opaque>
GET  /profile →  read Cookie     →  look up session  →  return user data
POST /logout  →  delete session  →  Set-Cookie: session_id=; MaxAge=-1
```

Cookie flags:
- `HttpOnly: true` — not readable by JavaScript (prevents XSS token theft)
- `SameSite: Lax` — cookie not sent on cross-site POST (CSRF mitigation)
- `Secure: true` — HTTPS only (always enable in production)

---

## JWT structure

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.    ← header
eyJzdWIiOiJ1MSIsImV4cCI6MTcwMDAwMDAwMH0.  ← payload (claims)
SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c  ← HMAC-SHA256 signature
```

Three Base64URL-encoded segments, separated by `.`. The signature is computed over `header.payload` — tampering with either invalidates it.

---

## Access + refresh token pattern

```
POST /login → { access_token (15m), refresh_token (7d, opaque) }

Every API call: Authorization: Bearer <access_token>

When access token expires:
POST /refresh { refresh_token } → { new_access_token, new_refresh_token }
(old refresh token is deleted — rotation prevents reuse)
```

Refresh tokens are stored server-side. Access tokens are stateless (validated by signature alone). To revoke an access token before expiry, use a token blocklist or shorten the TTL.

---

## Sessions vs JWTs

| | Sessions | JWTs |
|---|---|---|
| **State** | Server-side | Stateless (client holds all data) |
| **Revocation** | Instant (delete session) | Difficult (blocklist needed) |
| **Horizontal scaling** | Shared session store needed | No shared state |
| **Size** | Small cookie (ID only) | Token grows with claims |
| **Best for** | Web apps, single-server | APIs, microservices, mobile |

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_session_auth/main.go` | bcrypt, session store, Set-Cookie flags, login/logout/profile flow |
| `examples/02_jwt_auth/main.go` | JWT from scratch (HS256), access + refresh tokens, rotation, validation middleware |

## Exercise

`exercises/01_auth_server/main.go` — Full auth server: registration (bcrypt), session login, JWT login, refresh rotation, GET /me accepting either session or JWT, admin endpoint requiring JWT + admin role.
