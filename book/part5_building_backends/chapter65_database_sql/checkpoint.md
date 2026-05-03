# Chapter 65 Checkpoint — database/sql

## Self-assessment questions

1. What does `sql.Open()` actually do? When does Go actually connect to the database?
2. Why must you always call `defer rows.Close()` after a successful `QueryContext`?
3. What happens if you forget `rows.Err()` after iterating?
4. How do you represent an optional (nullable) string column in a Go struct?
5. Why is `defer tx.Rollback()` safe to call even after a successful `tx.Commit()`?
6. When should you use a prepared statement vs a regular `ExecContext`?
7. What is `SetConnMaxLifetime` for, and why does it matter in cloud deployments?

## Checklist

- [ ] Can open a database and ping it to verify connectivity
- [ ] Can query a single row with `QueryRowContext` and scan all fields
- [ ] Can query multiple rows with `QueryContext`, iterate with `rows.Next()`, and check `rows.Err()`
- [ ] Can execute INSERT/UPDATE/DELETE and read `LastInsertId` / `RowsAffected`
- [ ] Can use `sql.NullString` and other Null types for nullable columns
- [ ] Can create and use a prepared statement for batch operations
- [ ] Can wrap multiple operations in a transaction with proper `defer tx.Rollback()` + `tx.Commit()`
- [ ] Can configure the connection pool for production
- [ ] Can cancel a query using context timeout

## Answers

1. `sql.Open` validates the driver name and DSN format and returns a `*sql.DB` handle — it does NOT open a TCP connection. The first real connection is opened lazily on the first query, or explicitly with `db.Ping()`.

2. `rows.Close()` returns the database connection to the pool. If you don't call it, the connection is held until the `*sql.Rows` is garbage collected, which can exhaust the pool under load. `defer rows.Close()` right after a successful `QueryContext` makes it impossible to forget.

3. You miss errors that occurred during row iteration (e.g. a network failure mid-scan). The error is accumulated in `rows` and only accessible via `rows.Err()`. Forgetting it means you may return an incomplete result slice silently as if the query succeeded.

4. Use `sql.NullString{String: string, Valid: bool}`. `Valid` is `true` when the value is not NULL. Read: check `ns.Valid` before using `ns.String`. Write: `sql.NullString{String: s, Valid: s != ""}`.

5. After `tx.Commit()` returns successfully, `tx.Rollback()` detects that the transaction is already done and returns `sql.ErrTxDone` — it does nothing harmful. The `defer` pattern is safe because `Rollback` is idempotent for committed transactions.

6. Use prepared statements for queries in a loop (e.g. bulk insert 1000 rows). The database parses and plans the query once; subsequent executions reuse the plan. For one-off queries, the overhead of `PrepareContext` is not worth it — use `ExecContext` directly.

7. `SetConnMaxLifetime` closes connections that have been open longer than the specified duration. Cloud databases sit behind load balancers and firewalls that silently kill TCP connections after idle periods. Without this setting, your pool may hold connections that appear open but are actually dead, causing the next query to fail unexpectedly.
