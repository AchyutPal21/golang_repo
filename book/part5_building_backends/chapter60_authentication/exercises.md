# Chapter 60 — Exercises

## 60.1 — Auth Server

Run [`exercises/01_auth_server`](exercises/01_auth_server/main.go).

Full authentication server: bcrypt registration, session-cookie login, JWT login with access + refresh tokens, refresh rotation, and a `GET /me` endpoint that accepts either a session cookie or a JWT Bearer token. 16 test assertions cover registration (duplicate, short password), session lifecycle (login → profile → logout → 401), JWT flow (login, auth, role gate), and refresh rotation (old token revoked after first use).

Try:
- Add a `POST /auth/logout/jwt` endpoint that invalidates the refresh token by token value.
- Add a `GET /sessions` admin endpoint that lists all active sessions (count and creation times).
- Extend the `GET /me` response to include `last_login_at` and `login_count`.

## 60.2 ★ — Password reset flow

Implement a stateless password reset:

1. `POST /auth/forgot-password { email }` — generate a one-time reset token (HMAC-signed, 15-minute expiry) and print it (in a real system, email it)
2. `POST /auth/reset-password { token, new_password }` — verify the token, hash the new password, update the user record
3. The reset token must encode the user ID and expiry, be HMAC-signed, and be single-use (track used tokens in a store)

## 60.3 ★★ — OAuth2 authorization code flow (simulation)

Build a mini OAuth2 server without an external library:

- `GET /oauth/authorize?client_id=&redirect_uri=&state=` — simulates the user consent page; issues an authorization code
- `POST /oauth/token { code, client_id, client_secret, grant_type }` — exchanges code for access + refresh tokens; code is single-use and 5-minute TTL
- `GET /oauth/userinfo` — protected endpoint using the access token; returns the user profile
- Register two test clients with different `redirect_uri` values; verify CSRF protection via the `state` parameter
