# Chapter 65 — database/sql

## What you'll learn

Go's standard `database/sql` package — the universal interface for SQL databases. You'll learn how to open connections, query rows, scan results, handle NULL values, use prepared statements, manage transactions, configure the connection pool, and cancel queries with context.

## Key concepts

| Concept | API |
|---|---|
| Open (lazy) | `sql.Open("sqlite", dsn)` — does not connect yet |
| Ping (connect) | `db.Ping()` / `db.PingContext(ctx)` |
| Single row | `db.QueryRowContext(ctx, sql, args...).Scan(&fields...)` |
| Multiple rows | `db.QueryContext` → `rows.Next()` → `rows.Scan` → `rows.Close()` → `rows.Err()` |
| Execute | `db.ExecContext(ctx, sql, args...)` → `.LastInsertId()` / `.RowsAffected()` |
| Prepared stmt | `db.PrepareContext` → `stmt.ExecContext` → `stmt.Close()` |
| Transaction | `db.BeginTx` → operations → `tx.Commit()` + `defer tx.Rollback()` |
| NULL values | `sql.NullString`, `sql.NullFloat64`, `sql.NullInt64`, `sql.NullTime` |

## Files

| File | Topic |
|---|---|
| `examples/01_sql_basics/main.go` | CRUD, prepared statements, transactions, NULLs |
| `examples/02_connection_pool/main.go` | Pool settings, context cancellation/timeout, concurrent reads, db.Stats() |
| `exercises/01_user_store/main.go` | Users + sessions store; CRUD, sessions, atomic register+session transaction |

## The query lifecycle

```go
// Always use Context variants in production
rows, err := db.QueryContext(ctx, `SELECT id, name FROM products WHERE active = ?`, true)
if err != nil {
    return nil, err
}
defer rows.Close()  // ← REQUIRED: returns connection to pool

var products []Product
for rows.Next() {
    var p Product
    if err := rows.Scan(&p.ID, &p.Name); err != nil {
        return nil, err
    }
    products = append(products, p)
}
return products, rows.Err()  // ← check for iteration errors
```

## Transaction pattern

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil { return err }
defer tx.Rollback()  // no-op if committed; safe guard if panic

// ... operations using tx.ExecContext, tx.QueryRowContext ...

return tx.Commit()
```

## NULL handling

```go
type Product struct {
    ID          int
    Name        string
    Description sql.NullString  // optional
    Price       sql.NullFloat64 // optional
}

// Read
rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price)

// Write
db.ExecContext(ctx, `INSERT INTO products(name, description) VALUES (?, ?)`,
    p.Name,
    sql.NullString{String: p.Description, Valid: p.Description != ""},
)

// Use
if p.Description.Valid {
    fmt.Println(p.Description.String)
}
```

## Connection pool settings (production)

```go
db.SetMaxOpenConns(25)               // max simultaneous connections
db.SetMaxIdleConns(25)               // keep these idle (match MaxOpen)
db.SetConnMaxLifetime(5 * time.Minute)  // rotate old connections
db.SetConnMaxIdleTime(2 * time.Minute)  // close idle connections
```

## SQLite notes

- Use `modernc.org/sqlite` — pure Go, no CGO required.
- For in-memory databases shared across goroutines: `file::memory:?cache=shared` with `SetMaxOpenConns(1)`.
- For file databases: `file:data.db?_foreign_keys=on`.

## Production tips

- Always `defer rows.Close()` immediately after a successful `QueryContext`.
- Always check `rows.Err()` after iterating — it catches mid-iteration errors.
- Always `defer tx.Rollback()` immediately after `BeginTx` — it's a no-op on committed transactions.
- Use `PrepareContext` for queries executed in a loop — avoids re-parsing on every iteration.
- Set `SetConnMaxLifetime` to avoid stale connections when DB is behind a load balancer.
