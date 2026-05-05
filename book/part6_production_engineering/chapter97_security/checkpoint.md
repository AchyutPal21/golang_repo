# Chapter 97 Checkpoint — Security for Go Services

## Concepts to know

- [ ] What is SQL injection? How do parameterized queries prevent it?
- [ ] What is XSS? What is the difference between stored and reflected XSS?
- [ ] What is path traversal? How do you prevent it in Go?
- [ ] What is SSRF? Name two mitigations.
- [ ] What is the difference between authentication and authorization?
- [ ] What three claims must you always verify in a JWT?
- [ ] Why is HS256 weaker than RS256 for multi-service JWTs?
- [ ] What is a timing-safe comparison and when do you need it?
- [ ] Name four HTTP security response headers and their purposes.
- [ ] What does `govulncheck` check that `go mod tidy` does not?

## Code exercises

### 1. Input validator

Write a `ValidateUsername(s string) error` that:
- Rejects empty strings
- Allows only `[a-zA-Z0-9_-]`
- Rejects strings longer than 32 characters
- Returns a descriptive error for each case

### 2. Rate limiter

Write a `RateLimiter` that:
- Allows at most N requests per second per key (sliding window)
- Returns `(allowed bool, remaining int, resetAt time.Time)`
- Is safe for concurrent use

### 3. JWT claims validation

Write `ValidateClaims(claims map[string]any, issuer, audience string) error` that checks:
- `exp` is in the future (with 30s clock skew)
- `iss` matches the expected issuer
- `aud` includes the expected audience

## Quick reference

```go
// Parameterized query (safe)
db.QueryRow("SELECT * FROM users WHERE id = ?", userID)

// Path traversal prevention
clean := filepath.Clean(filepath.Join(baseDir, userInput))
if !strings.HasPrefix(clean, baseDir) {
    return errors.New("path traversal detected")
}

// Timing-safe comparison
hmac.Equal(received, expected)  // constant time
// Never: received == expected  // short-circuit, timing leak

// HTML escape
html.EscapeString(userInput)

// Security headers middleware
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("Content-Security-Policy", "default-src 'self'")
```

## Expected answers

1. SQL injection: user input changes query structure. Prevented by parameterized queries where input is always treated as data, never as SQL.
2. XSS: attacker injects scripts into pages. Stored: persisted in DB; Reflected: in request/response. Prevented by HTML escaping output.
3. Path traversal: `../` sequences to escape sandbox. Prevented by `filepath.Clean` + prefix check.
4. SSRF: server makes requests to internal services on attacker's behalf. Mitigate with URL allowlist + no-redirect policy.
5. Authentication: who are you? Authorization: what can you do?
6. `exp` (expiry), `iss` (issuer), `aud` (audience). Also `sub` for identity.
7. HS256 uses a shared secret — any service that verifies can also forge tokens. RS256 uses a private key to sign; services only need the public key to verify.
8. `hmac.Equal` and `subtle.ConstantTimeCompare` prevent timing attacks where an attacker learns string length/prefix by measuring comparison time.
9. `X-Frame-Options: DENY` (clickjacking), `X-Content-Type-Options: nosniff` (MIME sniffing), `Strict-Transport-Security` (HTTPS only), `Content-Security-Policy` (script/resource origin).
10. `govulncheck` checks whether your code actually calls vulnerable code paths, not just whether the vulnerable package is imported.
