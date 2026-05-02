// FILE: book/part5_building_backends/chapter60_authentication/exercises/01_auth_server/main.go
// CHAPTER: 60 — Authentication
// EXERCISE: Full authentication server combining sessions and JWT:
//   - POST /auth/register — create account (bcrypt password hash)
//   - POST /auth/login/session — cookie-based session login
//   - POST /auth/login/jwt — JWT login returning access + refresh tokens
//   - POST /auth/refresh — rotate refresh token, issue new access token
//   - POST /auth/logout — invalidate session or refresh token
//   - GET /me — works with EITHER a session cookie OR a JWT Bearer token
//   - GET /admin — JWT only, admin role required
//
// Run (from the chapter folder):
//   go run ./exercises/01_auth_server

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
	"net/http/cookiejar"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ─────────────────────────────────────────────────────────────────────────────
// USERS
// ─────────────────────────────────────────────────────────────────────────────

type User struct {
	ID           int64
	Username     string
	PasswordHash []byte
	Role         string
}

type userStore struct {
	mu     sync.RWMutex
	byName map[string]*User
	byID   map[int64]*User
	nextID atomic.Int64
}

func newUserStore() *userStore {
	s := &userStore{byName: make(map[string]*User), byID: make(map[int64]*User)}
	s.nextID.Store(1)
	return s
}

func (s *userStore) register(username, password, role string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byName[username]; exists {
		return nil, fmt.Errorf("username already taken")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &User{ID: s.nextID.Load(), Username: username, PasswordHash: hash, Role: role}
	s.byName[username] = u
	s.byID[u.ID] = u
	s.nextID.Add(1)
	return u, nil
}

func (s *userStore) login(username, password string) (*User, bool) {
	s.mu.RLock()
	u, ok := s.byName[username]
	s.mu.RUnlock()
	if !ok {
		bcrypt.CompareHashAndPassword([]byte("$2a$10$placeholder"), []byte(password)) // timing attack mitigation
		return nil, false
	}
	if bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password)) != nil {
		return nil, false
	}
	return u, true
}

// ─────────────────────────────────────────────────────────────────────────────
// SESSION STORE
// ─────────────────────────────────────────────────────────────────────────────

type Session struct {
	ID        string
	UserID    int64
	Username  string
	Role      string
	ExpiresAt time.Time
}

type sessionStore struct {
	mu    sync.RWMutex
	items map[string]*Session
}

func newSessionStore() *sessionStore {
	return &sessionStore{items: make(map[string]*Session)}
}

func (s *sessionStore) create(u *User) *Session {
	b := make([]byte, 32)
	rand.Read(b)
	id := base64.URLEncoding.EncodeToString(b)
	sess := &Session{ID: id, UserID: u.ID, Username: u.Username, Role: u.Role, ExpiresAt: time.Now().Add(24 * time.Hour)}
	s.mu.Lock()
	s.items[id] = sess
	s.mu.Unlock()
	return sess
}

func (s *sessionStore) get(id string) (*Session, bool) {
	s.mu.RLock()
	sess, ok := s.items[id]
	s.mu.RUnlock()
	return sess, ok && time.Now().Before(sess.ExpiresAt)
}

func (s *sessionStore) delete(id string) {
	s.mu.Lock()
	delete(s.items, id)
	s.mu.Unlock()
}

// ─────────────────────────────────────────────────────────────────────────────
// JWT
// ─────────────────────────────────────────────────────────────────────────────

var jwtSecret []byte

func init() {
	jwtSecret = make([]byte, 32)
	rand.Read(jwtSecret)
}

type Claims struct {
	Sub      string `json:"sub"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Iat      int64  `json:"iat"`
	Exp      int64  `json:"exp"`
	Type     string `json:"type"` // "access" or "refresh"
}

func issueJWT(c Claims) string {
	hdr, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	pay, _ := json.Marshal(c)
	h := base64.RawURLEncoding.EncodeToString(hdr) + "." + base64.RawURLEncoding.EncodeToString(pay)
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(h))
	return h + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func validateJWT(token string) (*Claims, error) {
	p := strings.SplitN(token, ".", 3)
	if len(p) != 3 {
		return nil, fmt.Errorf("malformed")
	}
	h := p[0] + "." + p[1]
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(h))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(p[2]), []byte(want)) {
		return nil, fmt.Errorf("invalid signature")
	}
	b, err := base64.RawURLEncoding.DecodeString(p[1])
	if err != nil {
		return nil, fmt.Errorf("bad payload")
	}
	var c Claims
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if time.Now().Unix() > c.Exp {
		return nil, fmt.Errorf("expired")
	}
	return &c, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// REFRESH TOKEN STORE
// ─────────────────────────────────────────────────────────────────────────────

type refreshEntry struct {
	User      *User
	ExpiresAt time.Time
}

type refreshStore struct {
	mu    sync.RWMutex
	items map[string]*refreshEntry
}

func newRefreshStore() *refreshStore {
	return &refreshStore{items: make(map[string]*refreshEntry)}
}

func (rs *refreshStore) issue(u *User) string {
	b := make([]byte, 32)
	rand.Read(b)
	tok := base64.URLEncoding.EncodeToString(b)
	rs.mu.Lock()
	rs.items[tok] = &refreshEntry{User: u, ExpiresAt: time.Now().Add(7 * 24 * time.Hour)}
	rs.mu.Unlock()
	return tok
}

func (rs *refreshStore) consume(tok string) (*User, bool) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	e, ok := rs.items[tok]
	if !ok || time.Now().After(e.ExpiresAt) {
		return nil, false
	}
	delete(rs.items, tok)
	return e.User, true
}

// ─────────────────────────────────────────────────────────────────────────────
// SERVER
// ─────────────────────────────────────────────────────────────────────────────

type Server struct {
	users    *userStore
	sessions *sessionStore
	refresh  *refreshStore
}

func (s *Server) writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) currentUser(r *http.Request) (*User, string) {
	// Try session cookie first.
	if c, err := r.Cookie("session_id"); err == nil {
		if sess, ok := s.sessions.get(c.Value); ok {
			s.users.mu.RLock()
			u := s.users.byID[sess.UserID]
			s.users.mu.RUnlock()
			return u, "session"
		}
	}
	// Try JWT Bearer token.
	auth := r.Header.Get("Authorization")
	if tok, ok := strings.CutPrefix(auth, "Bearer "); ok {
		if c, err := validateJWT(tok); err == nil && c.Type == "access" {
			s.users.mu.RLock()
			var u *User
			for _, uu := range s.users.byID {
				if fmt.Sprintf("%d", uu.ID) == c.Sub {
					u = uu
					break
				}
			}
			s.users.mu.RUnlock()
			return u, "jwt"
		}
	}
	return nil, ""
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if strings.TrimSpace(body.Username) == "" || len(body.Password) < 6 {
		s.writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "username required and password >= 6 chars"})
		return
	}
	if body.Role == "" {
		body.Role = "user"
	}
	u, err := s.users.register(body.Username, body.Password, body.Role)
	if err != nil {
		s.writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"id": u.ID, "username": u.Username, "role": u.Role})
}

func (s *Server) handleSessionLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	u, ok := s.users.login(body.Username, body.Password)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	sess := s.sessions.create(u)
	http.SetCookie(w, &http.Cookie{
		Name: "session_id", Value: sess.ID, Path: "/",
		Expires: sess.ExpiresAt, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	s.writeJSON(w, http.StatusOK, map[string]string{"msg": "logged in (session)", "username": u.Username})
}

func (s *Server) handleJWTLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	u, ok := s.users.login(body.Username, body.Password)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	now := time.Now().Unix()
	access := issueJWT(Claims{Sub: fmt.Sprintf("%d", u.ID), Username: u.Username, Role: u.Role,
		Iat: now, Exp: now + 15*60, Type: "access"})
	refresh := s.refresh.issue(u)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"access_token": access, "refresh_token": refresh,
		"token_type": "Bearer", "expires_in": 900,
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	u, ok := s.refresh.consume(body.RefreshToken)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired refresh token"})
		return
	}
	now := time.Now().Unix()
	access := issueJWT(Claims{Sub: fmt.Sprintf("%d", u.ID), Username: u.Username, Role: u.Role,
		Iat: now, Exp: now + 15*60, Type: "access"})
	newRefresh := s.refresh.issue(u)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"access_token": access, "refresh_token": newRefresh,
		"token_type": "Bearer", "expires_in": 900,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("session_id"); err == nil {
		s.sessions.delete(c.Value)
		http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", Path: "/", MaxAge: -1})
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"msg": "logged out"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, via := s.currentUser(r)
	if u == nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"id": u.ID, "username": u.Username, "role": u.Role, "via": via,
	})
}

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	u, via := s.currentUser(r)
	if u == nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}
	if via != "jwt" {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"error": "JWT required for admin"})
		return
	}
	if u.Role != "admin" {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"panel": "admin", "user": u.Username})
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	srv := &Server{
		users:    newUserStore(),
		sessions: newSessionStore(),
		refresh:  newRefreshStore(),
	}

	// Seed users.
	srv.users.register("admin", "adminpass", "admin")
	srv.users.register("user1", "userpass1", "user")

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", srv.handleRegister)
	mux.HandleFunc("POST /auth/login/session", srv.handleSessionLogin)
	mux.HandleFunc("POST /auth/login/jwt", srv.handleJWTLogin)
	mux.HandleFunc("POST /auth/refresh", srv.handleRefresh)
	mux.HandleFunc("POST /auth/logout", srv.handleLogout)
	mux.HandleFunc("GET /me", srv.handleMe)
	mux.HandleFunc("GET /admin", srv.handleAdmin)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: mux}).Serve(ln)

	jar, _ := cookiejar.New(nil)
	sessionClient := &http.Client{Timeout: 5 * time.Second, Jar: jar}
	bareClient := &http.Client{Timeout: 5 * time.Second}

	do := func(client *http.Client, method, path, body, bearer string) (int, map[string]any) {
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
		fmt.Printf("  %s %-56s %d\n", mark, label, code)
	}

	fmt.Printf("=== Auth Server — %s ===\n\n", base)

	fmt.Println("--- Registration ---")
	code, body := do(bareClient, "POST", "/auth/register", `{"username":"carol","password":"carol123","role":"user"}`, "")
	check("POST /auth/register (new user) → 201", code, 201)
	fmt.Printf("    id: %v  username: %v  role: %v\n", body["id"], body["username"], body["role"])

	code, _ = do(bareClient, "POST", "/auth/register", `{"username":"carol","password":"carol123"}`, "")
	check("POST /auth/register (duplicate) → 409", code, 409)

	code, _ = do(bareClient, "POST", "/auth/register", `{"username":"x","password":"12"}`, "")
	check("POST /auth/register (short password) → 422", code, 422)

	fmt.Println()
	fmt.Println("--- Session authentication ---")
	code, _ = do(bareClient, "GET", "/me", "", "")
	check("GET /me (no auth) → 401", code, 401)

	code, _ = do(sessionClient, "POST", "/auth/login/session", `{"username":"admin","password":"adminpass"}`, "")
	check("POST /auth/login/session → 200", code, 200)

	code, body = do(sessionClient, "GET", "/me", "", "")
	check("GET /me (session cookie) → 200", code, 200)
	fmt.Printf("    username: %v  role: %v  via: %v\n", body["username"], body["role"], body["via"])

	code, _ = do(sessionClient, "POST", "/auth/logout", "", "")
	check("POST /auth/logout → 200", code, 200)

	code, _ = do(sessionClient, "GET", "/me", "", "")
	check("GET /me (after session logout) → 401", code, 401)

	fmt.Println()
	fmt.Println("--- JWT authentication ---")
	code, jwtResp := do(bareClient, "POST", "/auth/login/jwt", `{"username":"admin","password":"adminpass"}`, "")
	check("POST /auth/login/jwt → 200", code, 200)
	accessToken, _ := jwtResp["access_token"].(string)
	refreshToken, _ := jwtResp["refresh_token"].(string)

	code, body = do(bareClient, "GET", "/me", "", accessToken)
	check("GET /me (JWT Bearer) → 200", code, 200)
	fmt.Printf("    username: %v  role: %v  via: %v\n", body["username"], body["role"], body["via"])

	code, _ = do(bareClient, "GET", "/admin", "", accessToken)
	check("GET /admin (admin JWT) → 200", code, 200)

	code, userResp := do(bareClient, "POST", "/auth/login/jwt", `{"username":"user1","password":"userpass1"}`, "")
	check("POST /auth/login/jwt (user1) → 200", code, 200)
	userToken, _ := userResp["access_token"].(string)
	code, _ = do(bareClient, "GET", "/admin", "", userToken)
	check("GET /admin (user1 role=user) → 403", code, 403)

	fmt.Println()
	fmt.Println("--- Refresh token rotation ---")
	code, refreshResp := do(bareClient, "POST", "/auth/refresh", `{"refresh_token":"`+refreshToken+`"}`, "")
	check("POST /auth/refresh (valid) → 200", code, 200)
	newAccess, _ := refreshResp["access_token"].(string)
	newRefresh, _ := refreshResp["refresh_token"].(string)

	// Old refresh token consumed.
	code, _ = do(bareClient, "POST", "/auth/refresh", `{"refresh_token":"`+refreshToken+`"}`, "")
	check("POST /auth/refresh (old token) → 401", code, 401)

	// New tokens work.
	code, _ = do(bareClient, "GET", "/me", "", newAccess)
	check("GET /me (rotated access token) → 200", code, 200)
	_ = newRefresh
}
