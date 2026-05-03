# Chapter 65 Exercises — database/sql

## Exercise 1 — User Store (`exercises/01_user_store`)

Build a user authentication store backed by SQLite using only `database/sql`.

### Schema

```sql
CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'user',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);
```

### Operations to implement

**User CRUD**
- `CreateUser(ctx, username, email, passwordHash, role)` → `*User`
- `GetUserByID(ctx, id)` → `*User` or error wrapping `sql.ErrNoRows`
- `GetUserByEmail(ctx, email)` → `*User` or error
- `ListUsers(ctx)` → `[]*User` ordered by ID
- `UpdateRole(ctx, userID, role)` → error (fail if user not found)
- `DeleteUser(ctx, userID)` → `(bool, error)` — bool true if row was deleted

**Session management**
- `CreateSession(ctx, userID, ttl)` → `*Session` with random 32-byte hex token
- `GetSessionByToken(ctx, token)` → `*Session` or error if not found or expired
- `DeleteExpiredSessions(ctx)` → `(int64, error)` rows cleaned

**Atomic registration**
- `RegisterWithSession(ctx, username, email, passwordHash)` → `*RegisterResult`
  - Creates user and session in a single transaction
  - Rolls back both if either fails

**Prepared statement bulk insert**
- `BulkCreateUsers(ctx, users)` → `(int, error)`
  - Uses `PrepareContext` once, then `stmt.ExecContext` in a loop

### Key requirements

- All methods must accept `context.Context` and use `*Context` query variants
- Token must be generated with `crypto/rand` (not `math/rand`)
- Sessions must store `expires_at` in UTC ISO 8601 format and check expiry on retrieval
- `RegisterWithSession` must use `defer tx.Rollback()` + `tx.Commit()` pattern

### Context timeout test

Demonstrate that a query respects context cancellation:

```go
ctx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
defer cancel()
_, err = store.ListUsers(ctx) // should return context deadline exceeded
```

### Hints

- SQLite in-memory with multiple connections: use DSN `file::memory:?cache=shared` and `SetMaxOpenConns(1)`
- SQLite datetime comparison: use `datetime('now')` in the WHERE clause for `DeleteExpiredSessions`
- Time format for SQLite: store as `"2006-01-02T15:04:05Z"` — ISO 8601 with Z suffix; parse with the same layout
- When scanning `created_at` from SQLite, it returns a string, not `time.Time` — scan into a `string` and parse with `time.Parse`
