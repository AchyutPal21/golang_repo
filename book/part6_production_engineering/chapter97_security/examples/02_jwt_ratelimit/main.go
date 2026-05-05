// FILE: book/part6_production_engineering/chapter97_security/examples/02_jwt_ratelimit/main.go
// CHAPTER: 97 — Security for Go Services
// TOPIC: JWT creation and validation, per-user rate limiting,
//        timing-safe comparison, and claims verification.
//
// Run:
//   go run ./book/part6_production_engineering/chapter97_security/examples/02_jwt_ratelimit

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SIMPLE JWT (HMAC-SHA256 / HS256)
// ─────────────────────────────────────────────────────────────────────────────

var ErrTokenExpired   = errors.New("token expired")
var ErrTokenInvalid   = errors.New("token invalid")
var ErrTokenBadClaims = errors.New("claims validation failed")

type Claims struct {
	Sub   string   `json:"sub"`
	Iss   string   `json:"iss"`
	Aud   string   `json:"aud"`
	Roles []string `json:"roles"`
	Exp   int64    `json:"exp"`
	Iat   int64    `json:"iat"`
}

func createToken(claims Claims, secret []byte) (string, error) {
	header := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadEnc := base64.RawURLEncoding.EncodeToString(payload)
	sigInput := header + "." + payloadEnc
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(sigInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return sigInput + "." + sig, nil
}

func validateToken(token string, secret []byte, expectedIss, expectedAud string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrTokenInvalid
	}
	sigInput := parts[0] + "." + parts[1]
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(sigInput))
	expectedSigBytes := h.Sum(nil)

	// Timing-safe comparison: decode received signature then compare bytes
	receivedSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, ErrTokenInvalid
	}
	if !hmac.Equal(receivedSig, expectedSigBytes) {
		return Claims{}, ErrTokenInvalid
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrTokenInvalid
	}
	var claims Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return Claims{}, ErrTokenInvalid
	}

	now := time.Now().Unix()
	if claims.Exp > 0 && now > claims.Exp+30 { // 30s clock skew
		return Claims{}, ErrTokenExpired
	}
	if claims.Iss != expectedIss {
		return Claims{}, fmt.Errorf("%w: iss mismatch: got %q want %q", ErrTokenBadClaims, claims.Iss, expectedIss)
	}
	if claims.Aud != expectedAud {
		return Claims{}, fmt.Errorf("%w: aud mismatch: got %q want %q", ErrTokenBadClaims, claims.Aud, expectedAud)
	}
	return claims, nil
}

func hasRole(claims Claims, role string) bool {
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// SLIDING WINDOW RATE LIMITER
// ─────────────────────────────────────────────────────────────────────────────

type RateLimiter struct {
	mu       sync.Mutex
	windows  map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		windows: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}
}

func (rl *RateLimiter) Allow(key string) (allowed bool, remaining int, resetAt time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Prune old timestamps
	times := rl.windows[key]
	valid := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	rl.windows[key] = valid

	if len(valid) >= rl.limit {
		resetAt = valid[0].Add(rl.window)
		return false, 0, resetAt
	}
	rl.windows[key] = append(rl.windows[key], now)
	remaining = rl.limit - len(rl.windows[key])
	return true, remaining, now.Add(rl.window)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 97: JWT & Rate Limiting ===")
	fmt.Println()

	secret := []byte("super-secret-signing-key-min-32-bytes")
	iss := "auth-service"
	aud := "api-gateway"

	// ── TOKEN CREATION ────────────────────────────────────────────────────────
	fmt.Println("--- JWT creation ---")
	now := time.Now()
	claims := Claims{
		Sub:   "user-42",
		Iss:   iss,
		Aud:   aud,
		Roles: []string{"user", "admin"},
		Exp:   now.Add(15 * time.Minute).Unix(),
		Iat:   now.Unix(),
	}
	token, err := createToken(claims, secret)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		return
	}
	parts := strings.Split(token, ".")
	fmt.Printf("  Header:    %s\n", parts[0])
	fmt.Printf("  Payload:   %s\n", parts[1])
	fmt.Printf("  Signature: %s...\n", parts[2][:16])
	fmt.Println()

	// ── TOKEN VALIDATION ──────────────────────────────────────────────────────
	fmt.Println("--- JWT validation ---")
	type testCase struct {
		label    string
		token    string
		iss, aud string
	}
	// Create expired token
	expiredClaims := claims
	expiredClaims.Exp = now.Add(-1 * time.Hour).Unix()
	expiredToken, _ := createToken(expiredClaims, secret)
	// Create tampered token
	tamperedToken := token[:len(token)-5] + "XXXXX"

	tests := []testCase{
		{"valid token", token, iss, aud},
		{"expired token", expiredToken, iss, aud},
		{"tampered signature", tamperedToken, iss, aud},
		{"wrong issuer", token, "other-service", aud},
		{"wrong audience", token, iss, "other-service"},
	}
	for _, tc := range tests {
		c, err := validateToken(tc.token, secret, tc.iss, tc.aud)
		if err != nil {
			fmt.Printf("  %-25s → ERROR: %v\n", tc.label, err)
		} else {
			fmt.Printf("  %-25s → OK (sub=%s, roles=%v)\n", tc.label, c.Sub, c.Roles)
		}
	}
	fmt.Println()

	// ── ROLE CHECK ────────────────────────────────────────────────────────────
	fmt.Println("--- Role-based access ---")
	validClaims, _ := validateToken(token, secret, iss, aud)
	fmt.Printf("  has 'user' role:  %v\n", hasRole(validClaims, "user"))
	fmt.Printf("  has 'admin' role: %v\n", hasRole(validClaims, "admin"))
	fmt.Printf("  has 'billing':    %v\n", hasRole(validClaims, "billing"))
	fmt.Println()

	// ── RATE LIMITER ──────────────────────────────────────────────────────────
	fmt.Println("--- Rate limiter (5 req/s per key) ---")
	rl := NewRateLimiter(5, time.Second)
	for i := 1; i <= 8; i++ {
		allowed, rem, _ := rl.Allow("user-42")
		status := "ALLOWED"
		if !allowed {
			status = "RATE LIMITED"
		}
		fmt.Printf("  Request %d: %-12s  remaining=%d\n", i, status, rem)
	}
	fmt.Println()

	// ── JWT BEST PRACTICES ────────────────────────────────────────────────────
	fmt.Println("--- JWT best practices ---")
	fmt.Println(`  Algorithm:
    HS256: shared secret — all verifiers can also forge (use only if single service)
    RS256: private key signs, public key verifies — recommended for multi-service
    ES256: ECDSA P-256 — smaller keys than RS256, same security

  Claims to always verify:
    exp: reject tokens past expiry (±30s clock skew allowed)
    iss: reject tokens from unexpected issuers
    aud: reject tokens not intended for this service

  Storage:
    HttpOnly cookie → inaccessible to JavaScript (XSS-safe)
    localStorage   → accessible to JS (vulnerable to XSS)
    sessionStorage → ditto

  Expiry strategy:
    Access token:  15 min (short-lived)
    Refresh token: 7 days (rotated on use, invalidated on logout)`)
}
