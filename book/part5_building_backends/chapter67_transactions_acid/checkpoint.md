# Chapter 67 Checkpoint — Transactions & ACID

## Self-assessment questions

1. Explain Atomicity in your own words. Give a real-world example where it matters.
2. What database mechanism enforces Consistency in SQL databases?
3. What is a "dirty read" and which isolation level prevents it?
4. What is the difference between REPEATABLE READ and SERIALIZABLE?
5. What is optimistic locking, and when should you use it over pessimistic locking?
6. How do you do a partial rollback within a transaction?
7. What causes deadlocks, and what is the standard prevention strategy?

## Checklist

- [ ] Can explain all four ACID properties with examples
- [ ] Know the four isolation levels and which anomalies each prevents
- [ ] Can write a multi-operation transaction with `defer tx.Rollback()` + `tx.Commit()` in Go
- [ ] Can implement optimistic locking with a version column
- [ ] Know when to use optimistic vs pessimistic locking
- [ ] Can use `SAVEPOINT` for partial rollback within a transaction
- [ ] Know what a deadlock is and how to prevent it via consistent lock ordering
- [ ] Know the Postgres error code for deadlock (`40P01`) and serialization failure (`40001`)

## Answers

1. Atomicity means all operations in a transaction commit together, or none do. Example: bank transfer — debiting $100 from Alice and crediting $100 to Bob must happen together. If the credit fails, the debit must be rolled back. Without atomicity, money could disappear.

2. Constraints: CHECK, NOT NULL, UNIQUE, FOREIGN KEY. The database enforces these before allowing a COMMIT. If any constraint would be violated, the statement (or transaction) is rejected.

3. A dirty read is when transaction T2 reads data written by T1 that hasn't been committed yet — if T1 rolls back, T2 read invalid data. READ COMMITTED prevents dirty reads (the default in PostgreSQL and MySQL).

4. REPEATABLE READ guarantees that if you read a row twice in the same transaction, you'll see the same value both times (no non-repeatable reads). SERIALIZABLE additionally prevents phantom reads (new rows appearing in range queries) and provides full serializability — transactions execute as if they ran one at a time.

5. Optimistic locking doesn't hold database locks — it reads a version, does work, then checks the version at write time (`UPDATE ... WHERE version = ?`). If the version changed (0 rows affected), another process modified the row; re-read and retry. Use optimistic when conflicts are rare (reads >> writes). Pessimistic (`SELECT FOR UPDATE`) holds locks for the duration — use when conflicts are frequent or atomicity is critical.

6. Use `SAVEPOINT name` to mark a point, `ROLLBACK TO SAVEPOINT name` to undo everything since then, and `RELEASE SAVEPOINT name` to remove the savepoint. The outer transaction remains open and can still be committed or rolled back.

7. Deadlocks occur when transaction T1 holds lock A and waits for B, while T2 holds B and waits for A. Prevention: always acquire locks in the same order across all transactions. For row locking, sort the IDs and always lock the lower ID first. PostgreSQL detects and aborts one transaction (`40P01`) — retry it.
