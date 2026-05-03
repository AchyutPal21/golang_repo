# Chapter 66 Checkpoint — PostgreSQL with pgx

## Self-assessment questions

1. What is the difference between `pgx.Connect` and `pgxpool.New`? When should you use each?
2. pgx uses `$1`, `$2` placeholders. What placeholder does `database/sql` typically use for SQLite/MySQL?
3. How do you represent a nullable `TEXT` column in a pgx struct?
4. What is `pgx.Batch` for, and what problem does it solve?
5. When should you use `CopyFrom` instead of a batch of INSERTs?
6. What Postgres error code signals a serialization failure, and what should you do when you receive it?
7. What does `SELECT ... FOR UPDATE` do in a transaction?

## Checklist

- [ ] Can open a `pgxpool.Pool` with tuned `MaxConns`, `MinConns`, `MaxConnLifetime`, `MaxConnIdleTime`
- [ ] Can insert a row using `INSERT ... RETURNING` and scan it back in one round trip
- [ ] Can use `pgtype.Text` (and similar) for nullable columns
- [ ] Can use named arguments (`@name`) for complex queries
- [ ] Can send a batch of queries in a single round trip with `pgx.Batch`
- [ ] Can bulk-insert rows using `CopyFrom`
- [ ] Can write a serializable transaction with automatic retry on `40001`
- [ ] Can inspect `pgconn.PgError.Code` to detect specific constraint violations
- [ ] Know when to use advisory locks

## Answers

1. `pgx.Connect` opens a single connection — good for scripts, tests, or CLI tools. `pgxpool.New` manages a connection pool — required for HTTP servers where multiple goroutines serve requests concurrently. The pool acquires and returns connections automatically.

2. `database/sql` with SQLite and MySQL uses `?`. With PostgreSQL's standard `lib/pq` it's also `$1`. pgx always uses `$1`/`$2` (PostgreSQL native).

3. Use `pgtype.Text{String: "value", Valid: true}` for a present value and `pgtype.Text{Valid: false}` for NULL. Alternatively, use a pointer `*string` — `nil` maps to NULL.

4. `pgx.Batch` collects multiple SQL statements and sends them to Postgres in a single round trip, then reads all results back together. This eliminates N-1 extra round trips when updating N rows — critical for latency when N is large.

5. `CopyFrom` uses PostgreSQL's binary COPY protocol, which is orders of magnitude faster than batched INSERTs for thousands of rows. Use it for bulk imports, data migrations, or seeding large datasets.

6. Error code `40001` (`serialization_failure`) means the serializable transaction detected a conflict and was aborted to maintain serializability. The correct response is to retry the entire transaction from scratch (re-read, re-write). Never partially retry — start over.

7. `SELECT ... FOR UPDATE` acquires a row-level exclusive lock. Other transactions trying to update or lock the same rows will block until your transaction commits or rolls back. Used in transfer-style operations to prevent two concurrent transactions from reading the same "old" balance and both proceeding.
