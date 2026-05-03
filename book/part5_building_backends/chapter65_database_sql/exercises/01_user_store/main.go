// FILE: book/part5_building_backends/chapter65_database_sql/exercises/01_user_store/main.go
// CHAPTER: 65 — database/sql
// EXERCISE: Build a User store backed by SQLite using database/sql:
//   - Schema: users (id, username, email, password_hash, role, created_at)
//   - Schema: sessions (id, user_id, token, created_at, expires_at)
//   - CRUD operations for users with proper NULL handling
//   - Session management: create, get by token, delete expired
//   - Transaction: register user + create initial session atomically
//   - Prepared statements for batch operations
//   - Context timeout on all queries
//
// Run (from the chapter folder):
//   go run ./exercises/01_user_store

package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// ─────────────────────────────────────────────────────────────────────────────
// SCHEMA
// ─────────────────────────────────────────────────────────────────────────────

const schema = `
CREATE TABLE IF NOT EXISTS users (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    username     TEXT NOT NULL UNIQUE,
    email        TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role         TEXT NOT NULL DEFAULT 'user',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
`

// ─────────────────────────────────────────────────────────────────────────────
// TYPES
// ─────────────────────────────────────────────────────────────────────────────

type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

type Session struct {
	ID        int64
	UserID    int64
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// USER STORE
// ─────────────────────────────────────────────────────────────────────────────

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func randToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ─────────────────────────────────────────────────────────────────────────────
// USER CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (s *Store) CreateUser(ctx context.Context, username, email, passwordHash, role string) (*User, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)`,
		username, email, passwordHash, role,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetUserByID(ctx, id)
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	var u User
	var createdAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash, role, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("user %d: not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	var createdAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash, role, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("user with email %q: not found", email)
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &u, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, email, role, created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		var u User
		var createdAt string
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &createdAt); err != nil {
			return nil, err
		}
		u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (s *Store) UpdateRole(ctx context.Context, userID int64, role string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET role = ? WHERE id = ?`, role, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user %d: not found", userID)
	}
	return nil
}

func (s *Store) DeleteUser(ctx context.Context, userID int64) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SESSION MANAGEMENT
// ─────────────────────────────────────────────────────────────────────────────

const timeFmt = "2006-01-02T15:04:05Z"

func (s *Store) CreateSession(ctx context.Context, userID int64, ttl time.Duration) (*Session, error) {
	token := randToken()
	exp := time.Now().UTC().Add(ttl)
	expiresAt := exp.Format(timeFmt)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)`,
		userID, token, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	id, _ := res.LastInsertId()
	return &Session{ID: id, UserID: userID, Token: token, ExpiresAt: exp}, nil
}

func (s *Store) GetSessionByToken(ctx context.Context, token string) (*Session, error) {
	var sess Session
	var createdAt, expiresAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, token, created_at, expires_at FROM sessions WHERE token = ?`, token,
	).Scan(&sess.ID, &sess.UserID, &sess.Token, &createdAt, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, err
	}
	sess.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	sess.ExpiresAt, _ = time.Parse(timeFmt, expiresAt)
	if sess.ExpiresAt.IsZero() {
		sess.ExpiresAt, _ = time.Parse("2006-01-02 15:04:05", expiresAt)
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}
	return &sess, nil
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at < datetime('now')`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ─────────────────────────────────────────────────────────────────────────────
// TRANSACTION: register + create session atomically
// ─────────────────────────────────────────────────────────────────────────────

type RegisterResult struct {
	User    *User
	Session *Session
}

func (s *Store) RegisterWithSession(ctx context.Context, username, email, passwordHash string) (*RegisterResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Create user.
	res, err := tx.ExecContext(ctx,
		`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`,
		username, email, passwordHash,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	userID, _ := res.LastInsertId()

	// Create session.
	token := randToken()
	expiresAt := time.Now().UTC().Add(24 * time.Hour).Format(timeFmt)
	sessRes, err := tx.ExecContext(ctx,
		`INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)`,
		userID, token, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}
	sessID, _ := sessRes.LastInsertId()

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &RegisterResult{
		User:    &User{ID: userID, Username: username, Email: email, Role: "user"},
		Session: &Session{ID: sessID, UserID: userID, Token: token},
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// PREPARED STATEMENT: bulk insert users
// ─────────────────────────────────────────────────────────────────────────────

func (s *Store) BulkCreateUsers(ctx context.Context, users []struct{ username, email, hash string }) (int, error) {
	stmt, err := s.db.PrepareContext(ctx,
		`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	count := 0
	for _, u := range users {
		if _, err := stmt.ExecContext(ctx, u.username, u.email, u.hash); err != nil {
			return count, fmt.Errorf("insert %s: %w", u.username, err)
		}
		count++
	}
	return count, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	if _, err := db.Exec(schema); err != nil {
		panic(err)
	}

	store := NewStore(db)
	ctx := context.Background()

	check := func(label string, err error) bool {
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", label, err)
			return false
		}
		fmt.Printf("  ✓ %s\n", label)
		return true
	}

	fmt.Println("=== User Store (database/sql + SQLite) ===")

	// ── CREATE ───────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Create users ---")
	alice, err := store.CreateUser(ctx, "alice", "alice@example.com", "hash_alice", "admin")
	check("create alice", err)
	bob, err := store.CreateUser(ctx, "bob", "bob@example.com", "hash_bob", "user")
	check("create bob", err)
	fmt.Printf("  alice.id=%d  bob.id=%d\n", alice.ID, bob.ID)

	// ── READ ────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Read users ---")
	u, err := store.GetUserByID(ctx, alice.ID)
	check("get alice by id", err)
	fmt.Printf("  username=%s  role=%s\n", u.Username, u.Role)

	u2, err := store.GetUserByEmail(ctx, "bob@example.com")
	check("get bob by email", err)
	fmt.Printf("  username=%s  email=%s\n", u2.Username, u2.Email)

	_, err = store.GetUserByID(ctx, 9999)
	fmt.Printf("  ✓ GetUserByID(9999): %v\n", err)

	// ── LIST ─────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Bulk create + list ---")
	n, err := store.BulkCreateUsers(ctx, []struct{ username, email, hash string }{
		{"carol", "carol@example.com", "hash_carol"},
		{"dave", "dave@example.com", "hash_dave"},
		{"eve", "eve@example.com", "hash_eve"},
	})
	check(fmt.Sprintf("bulk create %d users", n), err)
	users, err := store.ListUsers(ctx)
	check("list users", err)
	fmt.Printf("  total users: %d\n", len(users))
	for _, uu := range users {
		fmt.Printf("  id=%d username=%-8s role=%s\n", uu.ID, uu.Username, uu.Role)
	}

	// ── UPDATE ────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Update role ---")
	check("promote carol to editor", store.UpdateRole(ctx, 3, "editor"))
	carol, _ := store.GetUserByID(ctx, 3)
	fmt.Printf("  carol.role = %s\n", carol.Role)

	// ── SESSION ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Sessions ---")
	sess, err := store.CreateSession(ctx, alice.ID, 24*time.Hour)
	check("create session for alice", err)
	fmt.Printf("  token: %s...\n", sess.Token[:12])

	sess2, err := store.GetSessionByToken(ctx, sess.Token)
	if check("get session by token", err) {
		fmt.Printf("  session user_id=%d expires_in=%s\n", sess2.UserID, time.Until(sess2.ExpiresAt).Round(time.Minute))
	}

	_, err = store.GetSessionByToken(ctx, "invalid-token")
	fmt.Printf("  ✓ GetSessionByToken(invalid): %v\n", err)

	// ── TRANSACTION ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- RegisterWithSession (atomic transaction) ---")
	result, err := store.RegisterWithSession(ctx, "frank", "frank@example.com", "hash_frank")
	check("register frank + create session", err)
	fmt.Printf("  user.id=%d  session.id=%d\n", result.User.ID, result.Session.ID)

	// ── DELETE + CLEANUP ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Delete + cleanup ---")
	deleted, err := store.DeleteUser(ctx, bob.ID)
	check("delete bob", err)
	fmt.Printf("  deleted=%v\n", deleted)
	deleted, _ = store.DeleteUser(ctx, 9999)
	fmt.Printf("  ✓ delete non-existent: deleted=%v\n", deleted)

	// Cleanup expired sessions (none expired yet).
	n64, err := store.DeleteExpiredSessions(ctx)
	check("delete expired sessions", err)
	fmt.Printf("  expired sessions cleaned: %d\n", n64)

	// ── CONTEXT TIMEOUT ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Context timeout ---")
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()
	_, err = store.ListUsers(timeoutCtx)
	fmt.Printf("  ✓ ListUsers with expired context: %v\n", err)
}
