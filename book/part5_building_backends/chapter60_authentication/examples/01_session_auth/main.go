// FILE: book/part5_building_backends/chapter60_authentication/examples/01_session_auth/main.go
// CHAPTER: 60 — Authentication
// TOPIC: Cookie-based session authentication —
//        bcrypt password hashing, session ID generation with crypto/rand,
//        in-memory session store, Set-Cookie / HttpOnly / SameSite / Secure flags,
//        and the login/logout/protected-page flow.
//
// Run (from the chapter folder):
//   go run ./examples/01_session_auth

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ─────────────────────────────────────────────────────────────────────────────
// USER STORE — bcrypt hashed passwords
// ─────────────────────────────────────────────────────────────────────────────

type User struct {
	ID           int
	Username     string
	PasswordHash []byte
}

var userDB map[string]*User // username → User

func init() {
	userDB = make(map[string]*User)
	for i, u := range []struct{ name, pass string }{
		{"alice", "hunter2"},
		{"bob", "password123"},
	} {
		hash, _ := bcrypt.GenerateFromPassword([]byte(u.pass), bcrypt.DefaultCost)
		userDB[u.name] = &User{ID: i + 1, Username: u.name, PasswordHash: hash}
	}
}

func checkPassword(u *User, password string) bool {
	return bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password)) == nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SESSION STORE
// ─────────────────────────────────────────────────────────────────────────────

type Session struct {
	ID        string
	UserID    int
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]*Session)}
}

func (s *SessionStore) Create(userID int, username string, ttl time.Duration) *Session {
	b := make([]byte, 32)
	rand.Read(b)
	id := base64.URLEncoding.EncodeToString(b)

	sess := &Session{
		ID:        id,
		UserID:    userID,
		Username:  username,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return sess
}

func (s *SessionStore) Get(id string) (*Session, bool) {
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil, false
	}
	return sess, true
}

func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// ─────────────────────────────────────────────────────────────────────────────
// COOKIE HELPERS
// ─────────────────────────────────────────────────────────────────────────────

const sessionCookieName = "session_id"

func setSessionCookie(w http.ResponseWriter, sessionID string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,           // not accessible from JavaScript
		SameSite: http.SameSiteLaxMode, // CSRF mitigation
		// Secure: true,           // enable in production (requires HTTPS)
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // instruct browser to delete immediately
		HttpOnly: true,
	})
}

func sessionFromRequest(r *http.Request) string {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func handleLogin(sessions *SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		u, ok := userDB[strings.ToLower(creds.Username)]
		if !ok || !checkPassword(u, creds.Password) {
			// Always return the same message — do not reveal whether user exists.
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		sess := sessions.Create(u.ID, u.Username, 24*time.Hour)
		setSessionCookie(w, sess.ID, sess.ExpiresAt)
		writeJSON(w, http.StatusOK, map[string]any{
			"msg":      "logged in",
			"username": u.Username,
		})
	}
}

func handleLogout(sessions *SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessID := sessionFromRequest(r)
		if sessID != "" {
			sessions.Delete(sessID)
		}
		clearSessionCookie(w)
		writeJSON(w, http.StatusOK, map[string]string{"msg": "logged out"})
	}
}

func handleProfile(sessions *SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessID := sessionFromRequest(r)
		sess, ok := sessions.Get(sessID)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":    sess.UserID,
			"username":   sess.Username,
			"session_id": sess.ID[:8] + "...", // never expose full session ID in API responses
			"expires_at": sess.ExpiresAt,
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	sessions := NewSessionStore()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", handleLogin(sessions))
	mux.HandleFunc("POST /logout", handleLogout(sessions))
	mux.HandleFunc("GET /profile", handleProfile(sessions))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: mux}).Serve(ln)

	// Use cookiejar so the client automatically sends cookies on subsequent requests.
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Timeout: 5 * time.Second, Jar: jar}

	do := func(method, path, body string) (int, map[string]any) {
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

	fmt.Printf("=== Session Authentication — %s ===\n\n", base)

	fmt.Println("--- Password hashing ---")
	hash, _ := bcrypt.GenerateFromPassword([]byte("hunter2"), bcrypt.DefaultCost)
	fmt.Printf("  bcrypt hash of 'hunter2': %s\n", string(hash)[:29]+"...")
	fmt.Printf("  cost=%d — bcrypt is intentionally slow to resist brute force\n", bcrypt.DefaultCost)

	fmt.Println()
	fmt.Println("--- Login / session creation ---")
	code, body := do("GET", "/profile", "")
	check("GET /profile (unauthenticated) → 401", code, 401)

	code, body = do("POST", "/login", `{"username":"alice","password":"wrong"}`)
	check("POST /login (wrong password) → 401", code, 401)
	fmt.Printf("    error: %v\n", body["error"])

	code, body = do("POST", "/login", `{"username":"alice","password":"hunter2"}`)
	check("POST /login (correct) → 200", code, 200)
	fmt.Printf("    msg: %v  username: %v\n", body["msg"], body["username"])

	fmt.Println()
	fmt.Println("--- Session cookie propagation ---")
	// Cookie jar sends the session cookie automatically.
	code, body = do("GET", "/profile", "")
	check("GET /profile (with session cookie) → 200", code, 200)
	fmt.Printf("    user_id: %v  username: %v\n", body["user_id"], body["username"])
	fmt.Printf("    session_id (truncated): %v\n", body["session_id"])

	fmt.Println()
	fmt.Println("--- Logout and session invalidation ---")
	code, _ = do("POST", "/logout", "")
	check("POST /logout → 200", code, 200)

	// Session is deleted server-side; cookie is cleared client-side.
	code, _ = do("GET", "/profile", "")
	check("GET /profile (after logout) → 401", code, 401)

	fmt.Println()
	fmt.Println("--- Cookie flags ---")
	// Log in again to show the Set-Cookie header.
	req, _ := http.NewRequest("POST", base+"/login", strings.NewReader(`{"username":"bob","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(req)
	resp.Body.Close()
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookieName {
			fmt.Printf("  Set-Cookie: %s=[...] HttpOnly=%v SameSite=%v\n",
				c.Name, c.HttpOnly, c.SameSite)
		}
	}
}
