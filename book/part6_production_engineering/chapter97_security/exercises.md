# Chapter 97 Exercises — Security for Go Services

## Exercise 1 (provided): Security Hardening

Location: `exercises/01_security_hardening/main.go`

Production security hardening demonstration:
- Input validation pipeline (username, email, URL)
- Audit log with structured security events
- Secret scanning in environment variables
- Security headers middleware pattern
- govulncheck integration reference

## Exercise 2 (self-directed): Secure API Handler

Build an HTTP handler that:
- Validates a `Bearer` token in the `Authorization` header (HMAC-SHA256 signature)
- Extracts claims: `sub`, `exp`, `roles`
- Returns 401 for missing/invalid tokens, 403 for insufficient roles
- Applies per-user rate limiting (10 req/s)
- Logs every request with: timestamp, user, path, status, duration

Acceptance criteria:
- Expired tokens → 401
- Valid token, wrong role → 403
- Rate limit exceeded → 429
- Valid request → 200

## Exercise 3 (self-directed): Dependency Audit

Write a program that:
- Parses a `go.mod` file (provided as a string)
- Extracts all `require` entries
- Checks each against a mock vulnerability database (map of module → CVE)
- Reports: module, version, CVE ID, severity, recommendation
- Exits 1 if any CRITICAL vulnerabilities are found

## Stretch Goal: CSRF Protection

Implement CSRF token middleware:
- Generate a random 32-byte token per session (`crypto/rand`)
- Store it in a `HttpOnly` cookie
- On `POST`/`PUT`/`DELETE`: verify the token from the request header matches the cookie
- Return 403 if the token is missing or mismatched
- Use `subtle.ConstantTimeCompare` for the comparison
