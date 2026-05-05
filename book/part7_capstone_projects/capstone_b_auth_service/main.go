// FILE: book/part7_capstone_projects/capstone_b_auth_service/main.go
// CAPSTONE B — Auth Service
// Simulates: user registration, login, JWT access tokens, opaque refresh
// tokens, TOTP MFA setup/verify, and RBAC permission enforcement.
// No external dependencies — all crypto uses stdlib.
//
// Run:
//   go run ./book/part7_capstone_projects/capstone_b_auth_service

package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PASSWORD HASHER (bcrypt simulation — real bcrypt needs golang.org/x/crypto)
// ─────────────────────────────────────────────────────────────────────────────

// In production this would use bcrypt.GenerateFromPassword(pw, 12).
// Here we simulate the cost model with HMAC-SHA256 + salt.
type passwordHasher struct{ secret []byte }

func newPasswordHasher() *passwordHasher {
	return &passwordHasher{secret: []byte("bcrypt-sim-secret-not-for-prod")}
}

func (p *passwordHasher) Hash(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt) //nolint:errcheck
	h := hmac.New(sha256.New, p.secret)
	h.Write(salt)
	h.Write([]byte(password))
	digest := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(salt) + ":" + base64.StdEncoding.EncodeToString(digest)
}

func (p *passwordHasher) Verify(password, hash string) bool {
	parts := strings.SplitN(hash, ":", 2)
	if len(parts) != 2 {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expected, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	h := hmac.New(sha256.New, p.secret)
	h.Write(salt)
	h.Write([]byte(password))
	return hmac.Equal(h.Sum(nil), expected)
}

// ─────────────────────────────────────────────────────────────────────────────
// JWT (HMAC-SHA256, header.payload.signature in base64url)
// ─────────────────────────────────────────────────────────────────────────────

type Claims struct {
	Sub   string
	Roles []string
	Exp   int64
}

type tokenIssuer struct{ secret []byte }

func newTokenIssuer(secret string) *tokenIssuer {
	return &tokenIssuer{secret: []byte(secret)}
}

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func (t *tokenIssuer) Issue(c Claims) string {
	header := b64url([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := b64url([]byte(fmt.Sprintf(`{"sub":%q,"roles":%q,"exp":%d}`,
		c.Sub, strings.Join(c.Roles, ","), c.Exp)))
	msg := header + "." + payload
	h := hmac.New(sha256.New, t.secret)
	h.Write([]byte(msg))
	return msg + "." + b64url(h.Sum(nil))
}

func (t *tokenIssuer) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("malformed token")
	}
	msg := parts[0] + "." + parts[1]
	h := hmac.New(sha256.New, t.secret)
	h.Write([]byte(msg))
	expected := h.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(expected, got) {
		return Claims{}, errors.New("invalid signature")
	}
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	// Parse the minimal JSON we produce
	var sub, roles string
	var exp int64
	fmt.Sscanf(string(payload), `{"sub":%q,"roles":%q,"exp":%d}`, &sub, &roles, &exp)
	if time.Now().Unix() > exp {
		return Claims{}, errors.New("token expired")
	}
	var roleList []string
	if roles != "" {
		roleList = strings.Split(roles, ",")
	}
	return Claims{Sub: sub, Roles: roleList, Exp: exp}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// REFRESH TOKEN STORE (opaque token → session)
// ─────────────────────────────────────────────────────────────────────────────

type refreshSession struct {
	UserID    string
	ExpiresAt time.Time
}

type refreshStore struct {
	mu     sync.Mutex
	tokens map[string]refreshSession
}

func newRefreshStore() *refreshStore {
	return &refreshStore{tokens: map[string]refreshSession{}}
}

func (rs *refreshStore) Issue(userID string, ttl time.Duration) string {
	b := make([]byte, 32)
	rand.Read(b) //nolint:errcheck
	token := base64.RawURLEncoding.EncodeToString(b)
	rs.mu.Lock()
	rs.tokens[token] = refreshSession{UserID: userID, ExpiresAt: time.Now().Add(ttl)}
	rs.mu.Unlock()
	return token
}

func (rs *refreshStore) Consume(token string) (string, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	s, ok := rs.tokens[token]
	if !ok {
		return "", errors.New("refresh token not found")
	}
	delete(rs.tokens, token) // rotate: old token invalid after use
	if time.Now().After(s.ExpiresAt) {
		return "", errors.New("refresh token expired")
	}
	return s.UserID, nil
}

func (rs *refreshStore) Revoke(token string) {
	rs.mu.Lock()
	delete(rs.tokens, token)
	rs.mu.Unlock()
}

// ─────────────────────────────────────────────────────────────────────────────
// TOTP (RFC 6238 — Time-based One-Time Password)
// ─────────────────────────────────────────────────────────────────────────────

type totpService struct{}

func (ts *totpService) GenerateSecret() string {
	b := make([]byte, 20)
	rand.Read(b) //nolint:errcheck
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}

func (ts *totpService) OTPUri(secret, account, issuer string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		issuer, account, secret, issuer)
}

func (ts *totpService) computeCode(secret string, t int64) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, uint64(t))
	h := hmac.New(sha1.New, key)
	h.Write(msg)
	digest := h.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	code := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", code%uint32(math.Pow10(6))), nil
}

func (ts *totpService) Verify(secret, code string) bool {
	t := time.Now().Unix() / 30
	for _, step := range []int64{-1, 0, 1} { // ±1 window for clock drift
		expected, err := ts.computeCode(secret, t+step)
		if err == nil && expected == code {
			return true
		}
	}
	return false
}

func (ts *totpService) CurrentCode(secret string) string {
	code, _ := ts.computeCode(secret, time.Now().Unix()/30)
	return code
}

// ─────────────────────────────────────────────────────────────────────────────
// RBAC
// ─────────────────────────────────────────────────────────────────────────────

type rbacPolicy struct {
	rolePerms map[string][]string
}

func newRBAC() *rbacPolicy {
	return &rbacPolicy{rolePerms: map[string][]string{
		"admin":  {"orders:read", "orders:write", "orders:delete", "users:read", "users:write"},
		"user":   {"orders:read", "orders:write"},
		"viewer": {"orders:read"},
	}}
}

func (r *rbacPolicy) HasPermission(roles []string, perm string) bool {
	for _, role := range roles {
		for _, p := range r.rolePerms[role] {
			if p == perm {
				return true
			}
		}
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// USER STORE
// ─────────────────────────────────────────────────────────────────────────────

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Roles        []string
	TOTPSecret   string
	MFAEnabled   bool
}

type userStore struct {
	mu    sync.RWMutex
	users map[string]*User // email → user
}

func newUserStore() *userStore { return &userStore{users: map[string]*User{}} }

func (s *userStore) Create(email, hash string, roles []string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[email]; exists {
		return nil, errors.New("email already registered")
	}
	u := &User{ID: fmt.Sprintf("usr-%d", len(s.users)+1), Email: email, PasswordHash: hash, Roles: roles}
	s.users[email] = u
	return u, nil
}

func (s *userStore) GetByEmail(email string) (*User, bool) {
	s.mu.RLock()
	u, ok := s.users[email]
	s.mu.RUnlock()
	return u, ok
}

// ─────────────────────────────────────────────────────────────────────────────
// AUTH SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type AuthService struct {
	users    *userStore
	hasher   *passwordHasher
	issuer   *tokenIssuer
	refresh  *refreshStore
	totp     *totpService
	rbac     *rbacPolicy
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewAuthService() *AuthService {
	return &AuthService{
		users:      newUserStore(),
		hasher:     newPasswordHasher(),
		issuer:     newTokenIssuer("super-secret-jwt-key-change-in-prod"),
		refresh:    newRefreshStore(),
		totp:       &totpService{},
		rbac:       newRBAC(),
		accessTTL:  15 * time.Minute,
		refreshTTL: 7 * 24 * time.Hour,
	}
}

func (a *AuthService) Register(email, password string, roles []string) (*User, error) {
	hash := a.hasher.Hash(password)
	return a.users.Create(email, hash, roles)
}

type TokenPair struct{ Access, Refresh string }

func (a *AuthService) Login(email, password string) (TokenPair, error) {
	u, ok := a.users.GetByEmail(email)
	if !ok || !a.hasher.Verify(password, u.PasswordHash) {
		return TokenPair{}, errors.New("invalid credentials")
	}
	return a.issuePair(u), nil
}

func (a *AuthService) issuePair(u *User) TokenPair {
	access := a.issuer.Issue(Claims{
		Sub:   u.ID,
		Roles: u.Roles,
		Exp:   time.Now().Add(a.accessTTL).Unix(),
	})
	refresh := a.refresh.Issue(u.ID, a.refreshTTL)
	return TokenPair{Access: access, Refresh: refresh}
}

func (a *AuthService) Refresh(refreshToken string) (TokenPair, error) {
	userID, err := a.refresh.Consume(refreshToken)
	if err != nil {
		return TokenPair{}, err
	}
	// find user by ID — linear scan for demo
	a.users.mu.RLock()
	var found *User
	for _, u := range a.users.users {
		if u.ID == userID {
			found = u
			break
		}
	}
	a.users.mu.RUnlock()
	if found == nil {
		return TokenPair{}, errors.New("user not found")
	}
	return a.issuePair(found), nil
}

func (a *AuthService) SetupMFA(email string) (secret, uri string, err error) {
	a.users.mu.Lock()
	defer a.users.mu.Unlock()
	u, ok := a.users.users[email]
	if !ok {
		return "", "", errors.New("user not found")
	}
	secret = a.totp.GenerateSecret()
	u.TOTPSecret = secret
	uri = a.totp.OTPUri(secret, email, "GoBook")
	return secret, uri, nil
}

func (a *AuthService) VerifyMFA(email, code string) error {
	u, ok := a.users.GetByEmail(email)
	if !ok {
		return errors.New("user not found")
	}
	if !a.totp.Verify(u.TOTPSecret, code) {
		return errors.New("invalid TOTP code")
	}
	a.users.mu.Lock()
	u.MFAEnabled = true
	a.users.mu.Unlock()
	return nil
}

func (a *AuthService) Authenticate(token string) (Claims, error) {
	return a.issuer.Verify(token)
}

func (a *AuthService) Authorize(claims Claims, permission string) error {
	if !a.rbac.HasPermission(claims.Roles, permission) {
		return fmt.Errorf("forbidden: role %v lacks permission %q", claims.Roles, permission)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Capstone B: Auth Service ===")
	fmt.Println()

	svc := NewAuthService()

	// ── REGISTER ──────────────────────────────────────────────────────────────
	fmt.Println("--- Register users ---")
	admin, _ := svc.Register("admin@example.com", "s3cr3tP@ss!", []string{"admin"})
	user, _ := svc.Register("alice@example.com", "aliceP@ss!", []string{"user"})
	fmt.Printf("  Registered: %s (roles=%v)\n", admin.Email, admin.Roles)
	fmt.Printf("  Registered: %s (roles=%v)\n", user.Email, user.Roles)
	_, err := svc.Register("admin@example.com", "other", []string{"user"})
	fmt.Printf("  Duplicate:  %v\n", err)
	fmt.Println()

	// ── LOGIN ─────────────────────────────────────────────────────────────────
	fmt.Println("--- Login ---")
	pair, err := svc.Login("alice@example.com", "aliceP@ss!")
	fmt.Printf("  Login OK:      access[0:20]=%s...\n", pair.Access[:20])
	_, err = svc.Login("alice@example.com", "wrongpass")
	fmt.Printf("  Wrong password: %v\n", err)
	fmt.Println()

	// ── ACCESS TOKEN VERIFY ───────────────────────────────────────────────────
	fmt.Println("--- Access token verification ---")
	claims, err := svc.Authenticate(pair.Access)
	fmt.Printf("  Valid token:   sub=%s roles=%v\n", claims.Sub, claims.Roles)
	_, err = svc.Authenticate("bad.token.here")
	fmt.Printf("  Bad token:     %v\n", err)
	fmt.Println()

	// ── REFRESH TOKEN ROTATION ────────────────────────────────────────────────
	fmt.Println("--- Refresh token rotation ---")
	newPair, err := svc.Refresh(pair.Refresh)
	fmt.Printf("  Refresh OK:    new access[0:20]=%s...\n", newPair.Access[:20])
	_, err = svc.Refresh(pair.Refresh) // old token now consumed
	fmt.Printf("  Reuse old refresh: %v\n", err)
	fmt.Println()

	// ── RBAC ──────────────────────────────────────────────────────────────────
	fmt.Println("--- RBAC permission checks ---")
	adminPair, _ := svc.Login("admin@example.com", "s3cr3tP@ss!")
	adminClaims, _ := svc.Authenticate(adminPair.Access)
	userClaims, _ := svc.Authenticate(newPair.Access)

	perms := []string{"orders:read", "orders:delete", "users:write"}
	for _, perm := range perms {
		aErr := svc.Authorize(adminClaims, perm)
		uErr := svc.Authorize(userClaims, perm)
		aResult := "ALLOW"
		uResult := "ALLOW"
		if aErr != nil {
			aResult = "DENY"
		}
		if uErr != nil {
			uResult = "DENY"
		}
		fmt.Printf("  %-20s  admin=%-5s  user=%-5s\n", perm, aResult, uResult)
	}
	fmt.Println()

	// ── MFA / TOTP ────────────────────────────────────────────────────────────
	fmt.Println("--- TOTP MFA setup and verify ---")
	secret, uri, _ := svc.SetupMFA("alice@example.com")
	fmt.Printf("  TOTP secret: %s\n", secret)
	fmt.Printf("  OTP URI:     %s\n", uri[:60]+"...")

	totp := &totpService{}
	code := totp.CurrentCode(secret)
	fmt.Printf("  Current code: %s\n", code)

	err = svc.VerifyMFA("alice@example.com", code)
	fmt.Printf("  Verify real code:  %v\n", err)
	err = svc.VerifyMFA("alice@example.com", "000000")
	fmt.Printf("  Verify wrong code: %v\n", err)
}
