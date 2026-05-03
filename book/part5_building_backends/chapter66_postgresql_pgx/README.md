# Chapter 66 — PostgreSQL with pgx

## What you'll learn

How to use `jackc/pgx/v5` — Go's most capable PostgreSQL driver — for production-grade database access: connection pools, the pgx type system for nullable fields, batch queries, bulk COPY, serializable transactions with retry, pgconn error codes, and advisory locks.

## pgx vs database/sql

| Feature | pgx | database/sql |
|---|---|---|
| Placeholders | `$1`, `$2` (always) | `?` (driver-specific) |
| Struct scanning | `pgx.RowToAddrOfStructByName[T]` | Manual `Scan()` calls |
| Batch queries | `pgx.Batch{}` — one round trip | N separate queries |
| Bulk insert | `CopyFrom` (Postgres COPY protocol) | No equivalent |
| Named arguments | `@name` syntax | Driver-dependent |
| Nullable fields | `*T` or `pgtype.Text{Valid, String}` | `sql.NullString` |
| Error codes | `pgconn.PgError.Code` | Driver-specific |
| Pool | `pgxpool.Pool` | `database/sql.DB` |

## Files

| File | Topic |
|---|---|
| `examples/01_pgx_basics/main.go` | Direct connect vs pool, CRUD with RETURNING, named args, batch, CopyFrom |
| `examples/02_pgx_patterns/main.go` | Transactions, serializable retry, pgtype NULLs, advisory locks, pool stats |
| `exercises/01_product_store/main.go` | Full product inventory store with price history and bulk operations |

## Connection string formats

```
postgres://user:pass@host:5432/dbname?sslmode=disable
host=localhost port=5432 user=app password=secret dbname=mydb sslmode=require
```

## Pool setup

```go
cfg, _ := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
cfg.MaxConns = 25
cfg.MinConns = 2
cfg.MaxConnLifetime = 5 * time.Minute
cfg.MaxConnIdleTime = 2 * time.Minute
cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
    // Register custom types or set session params here
    return nil
}
pool, err := pgxpool.NewWithConfig(ctx, cfg)
```

## Common patterns

### INSERT with RETURNING

```go
var p Product
err := pool.QueryRow(ctx, `
    INSERT INTO products (name, price) VALUES ($1, $2)
    RETURNING id, name, price, created_at`,
    name, price,
).Scan(&p.ID, &p.Name, &p.Price, &p.CreatedAt)
```

### Batch queries (N updates in 1 round trip)

```go
batch := &pgx.Batch{}
for id, stock := range updates {
    batch.Queue(`UPDATE products SET stock = $1 WHERE id = $2`, stock, id)
}
br := pool.SendBatch(ctx, batch)
defer br.Close()
for range updates {
    br.Exec()
}
```

### Serializable transaction with retry

```go
for attempt := 1; attempt <= 3; attempt++ {
    tx, _ := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
    err := doWork(tx)
    if err != nil {
        tx.Rollback(ctx)
        if isSerializationFailure(err) { continue }
        return err
    }
    if err := tx.Commit(ctx); err != nil {
        if isSerializationFailure(err) { continue }
        return err
    }
    return nil
}
```

### Error code check

```go
func isDuplicateKey(err error) bool {
    var pg *pgconn.PgError
    return errors.As(err, &pg) && pg.Code == "23505"
}
```

## Important Postgres error codes

| Code | Name | When |
|---|---|---|
| `23505` | `unique_violation` | Duplicate key |
| `23503` | `foreign_key_violation` | FK constraint |
| `23502` | `not_null_violation` | NOT NULL column |
| `40001` | `serialization_failure` | Retry serializable tx |
| `42P01` | `undefined_table` | Schema not found |
| `08006` | `connection_failure` | Network error |

## Running the examples

No Postgres running? All examples gracefully fall back and print the API shapes.

With a real database:
```bash
DATABASE_URL=postgres://postgres:secret@localhost:5432/upskill_go?sslmode=disable \
  go run ./examples/01_pgx_basics
```
