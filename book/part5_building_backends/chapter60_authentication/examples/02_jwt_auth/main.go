// FILE: book/part5_building_backends/chapter60_authentication/examples/02_jwt_auth/main.go
// CHAPTER: 60 — Authentication
// TOPIC: JWT (JSON Web Token) authentication from scratch —
//        HMAC-SHA256 signing, Base64URL encoding, claims structure,
//        access token + refresh token pattern, token validation middleware.
//
// This example implements JWT without an external library to show exactly
// how the format works: header.payload.signature (all Base64URL-encoded).
// In production, use a library like github.com/golang-jwt/jwt/v5.
//
// Run (from the chapter folder):
//   go run ./examples/02_jwt_auth

package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// JWT IMPLEMENTATION (HMAC-SHA256 / HS256)
// ─────────────────────────────────────────────────────────────────────────────

var secretKey []byte

func init() {
	secretKey = make([]byte, 32)
	rand.Read(secretKey)
}

type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type Claims struct {
	Sub  string `json:"sub"`            // subject (user ID)
	Name string `json:"name"`
	Role string `json:"role"`
	Iat  int64  `json:"iat"`            // issued at
	Exp  int64  `json:"exp"`            // expiry
	Type string `json:"type,omitempty"` // "access" or "refresh"
}

func b64(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func sign(data []byte) []byte {
	mac := hmac.New(sha256.New, secretKey)
	mac.Write(data)
	return mac.Sum(nil)
}

// IssueToken creates a signed JWT with the given claims.
func IssueToken(claims Claims) string {
	hdr, _ := json.Marshal(Header{Alg: "HS256", Typ: "JWT"})
	payload, _ := json.Marshal(claims)
	unsigned := b64(hdr) + "." + b64(payload)
	sig := sign([]byte(unsigned))
	return unsigned + "." + b64(sig)
}

// ValidateToken parses and verifies a JWT; returns claims if valid.
func ValidateToken(token string) (*Claims, error) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token")
	}
	unsigned := parts[0] + "." + parts[1]
	expectedSig := b64(sign([]byte(unsigned)))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding")
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, fmt.Errorf("invalid payload JSON")
	}
	if time.Now().Unix() > c.Exp {
		return nil, fmt.Errorf("token expired")
	}
	return &c, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// USER DB
// ─────────────────────────────────────────────────────────────────────────────

type User struct {
	ID       string
	Name     string
	Role     string
	Password string // plaintext for demo — use bcrypt in production
}

var users = map[string]*User{
	"alice": {ID: "u1", Name: "Alice", Role: "admin", Password: "secret"},
	"bob":   {ID: "u2", Name: "Bob", Role: "user", Password: "password"},
}

// ─────────────────────────────────────────────────────────────────────────────
// REFRESH TOKEN STORE
// Refresh tokens are opaque random strings, stored server-side.
// When a client presents one, we issue a new access token.
// ─────────────────────────────────────────────────────────────────────────────

type RefreshToken struct {
	UserID    string
	UserName  string
	Role      string
	ExpiresAt time.Time
}

type RefreshStore struct {
	mu    sync.RWMutex
	store map[string]*RefreshToken
}

var refreshStore = &RefreshStore{store: make(map[string]*RefreshToken)}

func (rs *RefreshStore) Issue(userID, name, role string) string {
	b := make([]byte, 32)
	rand.Read(b)
	tok := base64.URLEncoding.EncodeToString(b)
	rs.mu.Lock()
	rs.store[tok] = &RefreshToken{
		UserID: userID, UserName: name, Role: role,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	rs.mu.Unlock()
	return tok
}

func (rs *RefreshStore) Consume(tok string) (*RefreshToken, bool) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rt, ok := rs.store[tok]
	if !ok || time.Now().After(rt.ExpiresAt) {
		return nil, false
	}
	delete(rs.store, tok) // single-use: delete on consumption
	return rt, true
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT
// ─────────────────────────────────────────────────────────────────────────────

type ctxKey int

const keyClaims ctxKey = iota

// withClaims is unused in this demo (we use a sync.Map to keep imports minimal).
// In production, use context.WithValue(r.Context(), keyClaims, c).
var _ = func(r *http.Request, c *Claims) *http.Request { _ = c; return r }

// For this demo we use a package-level map keyed on request pointer.
// Never do this in production — only context.WithValue is safe for concurrency.
var requestClaims sync.Map

func setClaims(r *http.Request, c *Claims) {
	requestClaims.Store(r, c)
}

func getClaims(r *http.Request) (*Claims, bool) {
	v, ok := requestClaims.Load(r)
	if !ok {
		return nil, false
	}
	return v.(*Claims), true
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// ─────────────────────────────────────────────────────────────────────────────
// MIDDLEWARE — JWT AUTH
// ─────────────────────────────────────────────────────────────────────────────

func jwtAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		claims, err := ValidateToken(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		if claims.Type != "access" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not an access token"})
			return
		}
		setClaims(r, claims)
		next.ServeHTTP(w, r)
		requestClaims.Delete(r) // cleanup
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	u, ok := users[creds.Username]
	if !ok || u.Password != creds.Password {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	now := time.Now().Unix()
	accessToken := IssueToken(Claims{
		Sub: u.ID, Name: u.Name, Role: u.Role,
		Iat: now, Exp: now + 15*60, // 15 minutes
		Type: "access",
	})
	refreshToken := refreshStore.Issue(u.ID, u.Name, u.Role)

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900, // seconds
	})
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	rt, ok := refreshStore.Consume(body.RefreshToken)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired refresh token"})
		return
	}
	now := time.Now().Unix()
	newAccess := IssueToken(Claims{
		Sub: rt.UserID, Name: rt.UserName, Role: rt.Role,
		Iat: now, Exp: now + 15*60,
		Type: "access",
	})
	// Issue a new refresh token (rotation — old one is already deleted).
	newRefresh := refreshStore.Issue(rt.UserID, rt.UserName, rt.Role)

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  newAccess,
		"refresh_token": newRefresh,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	c, _ := getClaims(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"sub":  c.Sub,
		"name": c.Name,
		"role": c.Role,
	})
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	c, _ := getClaims(r)
	if c.Role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"msg": "admin panel", "user": c.Name})
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", handleLogin)
	mux.HandleFunc("POST /refresh", handleRefresh)
	mux.Handle("GET /me", jwtAuth(http.HandlerFunc(handleMe)))
	mux.Handle("GET /admin", jwtAuth(http.HandlerFunc(handleAdmin)))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: mux}).Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}

	do := func(method, path, body, bearer string) (int, map[string]any) {
		var br *strings.Reader
		if body != "" {
			br = strings.NewReader(body)
		} else {
			br = strings.NewReader("")
		}
		req, _ := http.NewRequest(method, base+path, br)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil
		}
		defer resp.Body.Close()
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}

	check := func(label string, code, want int) {
		mark := "✓"
		if code != want {
			mark = "✗"
		}
		fmt.Printf("  %s %-52s %d\n", mark, label, code)
	}

	fmt.Printf("=== JWT Authentication — %s ===\n\n", base)

	fmt.Println("--- JWT structure ---")
	// Show what the token looks like.
	sampleClaims := Claims{Sub: "u1", Name: "Alice", Role: "admin",
		Iat: time.Now().Unix(), Exp: time.Now().Add(15 * time.Minute).Unix(), Type: "access"}
	sample := IssueToken(sampleClaims)
	parts := strings.SplitN(sample, ".", 3)
	fmt.Printf("  header:    %s\n", parts[0])
	fmt.Printf("  payload:   %s\n", parts[1])
	fmt.Printf("  signature: %s...\n", parts[2][:16])
	hdrB, _ := base64.RawURLEncoding.DecodeString(parts[0])
	payB, _ := base64.RawURLEncoding.DecodeString(parts[1])
	fmt.Printf("  header decoded:  %s\n", hdrB)
	fmt.Printf("  payload decoded: %s\n", payB)

	fmt.Println()
	fmt.Println("--- Login and token issuance ---")
	code, resp := do("POST", "/login", `{"username":"alice","password":"secret"}`, "")
	check("POST /login (correct) → 200", code, 200)
	accessToken, _ := resp["access_token"].(string)
	refreshToken, _ := resp["refresh_token"].(string)
	fmt.Printf("    access_token:  %s...\n", accessToken[:20])
	fmt.Printf("    refresh_token: %s...\n", refreshToken[:10])
	fmt.Printf("    expires_in:    %v seconds\n", resp["expires_in"])

	fmt.Println()
	fmt.Println("--- Protected endpoints ---")
	code, _ = do("GET", "/me", "", "")
	check("GET /me (no token) → 401", code, 401)

	code, body := do("GET", "/me", "", accessToken)
	check("GET /me (valid token) → 200", code, 200)
	fmt.Printf("    user: %v  role: %v\n", body["name"], body["role"])

	// Bob's token cannot access admin endpoint.
	code, bobResp := do("POST", "/login", `{"username":"bob","password":"password"}`, "")
	check("POST /login (bob) → 200", code, 200)
	bobToken, _ := bobResp["access_token"].(string)
	code, _ = do("GET", "/admin", "", bobToken)
	check("GET /admin (bob=user) → 403", code, 403)

	code, _ = do("GET", "/admin", "", accessToken)
	check("GET /admin (alice=admin) → 200", code, 200)

	fmt.Println()
	fmt.Println("--- Refresh token rotation ---")
	code, refreshResp := do("POST", "/refresh", `{"refresh_token":"`+refreshToken+`"}`, "")
	check("POST /refresh (valid) → 200", code, 200)
	newAccess, _ := refreshResp["access_token"].(string)
	newRefresh, _ := refreshResp["refresh_token"].(string)
	fmt.Printf("    new access token:  %s...\n", newAccess[:20])
	fmt.Printf("    new refresh token: %s...\n", newRefresh[:10])

	// Old refresh token is revoked after rotation.
	code, _ = do("POST", "/refresh", `{"refresh_token":"`+refreshToken+`"}`, "")
	check("POST /refresh (old token, revoked) → 401", code, 401)

	// New access token works.
	code, _ = do("GET", "/me", "", newAccess)
	check("GET /me (new access token) → 200", code, 200)

	fmt.Println()
	fmt.Println("--- Token tampering ---")
	tampered := accessToken[:len(accessToken)-4] + "XXXX"
	code, _ = do("GET", "/me", "", tampered)
	check("GET /me (tampered signature) → 401", code, 401)

	// Craft a token with expired time.
	expiredClaims := Claims{Sub: "u1", Name: "Alice", Role: "admin",
		Iat: time.Now().Add(-1 * time.Hour).Unix(),
		Exp: time.Now().Add(-30 * time.Minute).Unix(),
		Type: "access"}
	expired := IssueToken(expiredClaims)
	code, _ = do("GET", "/me", "", expired)
	check("GET /me (expired token) → 401", code, 401)
}
