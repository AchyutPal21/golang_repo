# Chapter 97 — Security for Go Services

Security is not a feature to add at the end — it is woven into how you handle input, authenticate requests, manage secrets, and keep dependencies up to date.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | OWASP patterns | Input validation, SQL injection, XSS, path traversal |
| 2 | JWT & rate limiting | Token validation, claims verification, rate limiter |
| E | Secure service | Defense-in-depth: validation + auth + rate limit + audit log |

## Examples

### `examples/01_owasp_patterns`

Common OWASP Top 10 vulnerabilities and their Go mitigations:
- SQL injection: parameterized queries
- XSS: HTML escaping
- Path traversal: filepath.Clean + sandbox check
- SSRF: allowlist-based URL validation
- Command injection: `exec.Command` with args (never shell string)

### `examples/02_jwt_ratelimit`

JWT authentication and rate limiting:
- HMAC-SHA256 JWT creation and validation
- Claims: sub, exp, iat, iss, roles
- Token expiry and clock skew handling
- Sliding window rate limiter
- Per-IP and per-user rate limiting

### `exercises/01_security_hardening`

Production security hardening checklist implementation:
- Input sanitization pipeline
- Audit log with structured events
- Secret scanning in config
- Security headers middleware
- `govulncheck` integration reference

## Key Concepts

**Input validation rules**
1. Validate at the boundary — never trust input that crosses a trust boundary
2. Whitelist over blacklist — define what is allowed, reject everything else
3. Sanitize for the output context — HTML encoding for HTML, parameterization for SQL

**JWT best practices**
- Always verify `exp`, `iss`, `aud` claims
- Use RS256 or ES256 in production (asymmetric) — not HS256
- Short expiry (15min access token) + refresh token rotation
- Store in `HttpOnly` cookie, not `localStorage`

**Security headers**
```
Content-Security-Policy: default-src 'self'
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
Strict-Transport-Security: max-age=31536000; includeSubDomains
Referrer-Policy: strict-origin-when-cross-origin
```

**govulncheck**
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

## Running

```bash
go run ./book/part6_production_engineering/chapter97_security/examples/01_owasp_patterns
go run ./book/part6_production_engineering/chapter97_security/examples/02_jwt_ratelimit
go run ./book/part6_production_engineering/chapter97_security/exercises/01_security_hardening
```
