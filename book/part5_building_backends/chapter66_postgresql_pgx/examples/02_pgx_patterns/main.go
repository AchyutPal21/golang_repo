// FILE: book/part5_building_backends/chapter66_postgresql_pgx/examples/02_pgx_patterns/main.go
// CHAPTER: 66 — PostgreSQL with pgx
// TOPIC: pgx advanced patterns — transactions, pgtype for NULL handling,
//        CollectRows struct scanning, serializable isolation & retry,
//        pgconn error code handling, and advisory locks.
//
// Run (from the chapter folder):
//   go run ./examples/02_pgx_patterns
//
// With a live Postgres:
//   DATABASE_URL=postgres://user:pass@localhost:5432/mydb go run ./examples/02_pgx_patterns

package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─────────────────────────────────────────────────────────────────────────────
// TYPES — pgtype for nullable fields
// ─────────────────────────────────────────────────────────────────────────────

// Account uses pgtype for nullable columns.
// pgtype.Text{Valid: true, String: "..."} — present value
// pgtype.Text{Valid: false}               — NULL
type Account struct {
	ID          int64
	Name        string
	Email       pgtype.Text // nullable
	Balance     int64       // cents
	LockedUntil pgtype.Timestamptz
	CreatedAt   time.Time
}

type TxRecord struct {
	ID        int64
	AccountID int64
	Amount    int64
	Note      string
	CreatedAt time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// SCHEMA
// ─────────────────────────────────────────────────────────────────────────────

const ddl = `
CREATE TABLE IF NOT EXISTS accounts (
    id           BIGSERIAL PRIMARY KEY,
    name         TEXT        NOT NULL,
    email        TEXT,
    balance      BIGINT      NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS tx_records (
    id         BIGSERIAL PRIMARY KEY,
    account_id BIGINT      NOT NULL REFERENCES accounts(id),
    amount     BIGINT      NOT NULL,
    note       TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

// ─────────────────────────────────────────────────────────────────────────────
// TRANSACTION WITH RETRY
// Serializable transactions may fail with code 40001 (serialization failure).
// The correct response is to retry the whole transaction.
// ─────────────────────────────────────────────────────────────────────────────

func withRetry(ctx context.Context, pool *pgxpool.Pool, maxRetries int,
	fn func(tx pgx.Tx) error,
) error {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{
			IsoLevel: pgx.Serializable,
		})
		if err != nil {
			return err
		}

		err = fn(tx)
		if err != nil {
			_ = tx.Rollback(ctx)
			if isSerializationFailure(err) && attempt < maxRetries {
				// Exponential backoff before retry.
				time.Sleep(time.Duration(attempt*attempt) * 10 * time.Millisecond)
				continue
			}
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			if isSerializationFailure(err) && attempt < maxRetries {
				time.Sleep(time.Duration(attempt*attempt) * 10 * time.Millisecond)
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("max retries exceeded")
}

func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001"
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// TRANSFER — atomic debit+credit using serializable transaction
// ─────────────────────────────────────────────────────────────────────────────

func transfer(ctx context.Context, pool *pgxpool.Pool, fromID, toID, amount int64, note string) error {
	return withRetry(ctx, pool, 3, func(tx pgx.Tx) error {
		// Read both balances (SELECT FOR UPDATE to prevent concurrent modifications).
		var fromBalance, toBalance int64
		err := tx.QueryRow(ctx,
			`SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`, fromID,
		).Scan(&fromBalance)
		if err != nil {
			return fmt.Errorf("read from account: %w", err)
		}
		err = tx.QueryRow(ctx,
			`SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`, toID,
		).Scan(&toBalance)
		if err != nil {
			return fmt.Errorf("read to account: %w", err)
		}

		if fromBalance < amount {
			return fmt.Errorf("insufficient balance: have %d, need %d", fromBalance, amount)
		}

		// Debit from.
		_, err = tx.Exec(ctx, `UPDATE accounts SET balance = balance - $1 WHERE id = $2`, amount, fromID)
		if err != nil {
			return fmt.Errorf("debit: %w", err)
		}
		// Credit to.
		_, err = tx.Exec(ctx, `UPDATE accounts SET balance = balance + $1 WHERE id = $2`, amount, toID)
		if err != nil {
			return fmt.Errorf("credit: %w", err)
		}

		// Record both legs.
		_, err = tx.Exec(ctx,
			`INSERT INTO tx_records (account_id, amount, note) VALUES ($1, $2, $3), ($4, $5, $6)`,
			fromID, -amount, note,
			toID, amount, note,
		)
		return err
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// COLLECTROWS — scanning rows into structs by name
// ─────────────────────────────────────────────────────────────────────────────

func listAccounts(ctx context.Context, pool *pgxpool.Pool) ([]*Account, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, name, email, balance, locked_until, created_at FROM accounts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Email, &a.Balance, &a.LockedUntil, &a.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, &a)
	}
	return accounts, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// ADVISORY LOCKS — application-level distributed mutex
// pg_try_advisory_xact_lock acquires for the duration of the transaction.
// ─────────────────────────────────────────────────────────────────────────────

func withAdvisoryLock(ctx context.Context, pool *pgxpool.Pool, lockID int64, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var acquired bool
	if err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock($1)`, lockID).Scan(&acquired); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	if !acquired {
		return fmt.Errorf("could not acquire advisory lock %d", lockID)
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// PGTYPE DEMO
// ─────────────────────────────────────────────────────────────────────────────

func showPgTypeUsage() {
	fmt.Println("--- pgtype NULL handling ---")

	// pgtype.Text — nullable string
	present := pgtype.Text{String: "user@example.com", Valid: true}
	absent := pgtype.Text{Valid: false} // NULL

	if present.Valid {
		fmt.Printf("  email (present): %s\n", present.String)
	}
	if !absent.Valid {
		fmt.Println("  email (absent): NULL")
	}

	// pgtype.Timestamptz — nullable timestamp
	ts := pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true}
	fmt.Printf("  locked_until: %v\n", ts.Time.Format(time.RFC3339))

	// Numeric — arbitrary precision (use for money in prod)
	fmt.Println("  pgtype.Numeric — arbitrary precision decimal for money columns")
	fmt.Println("  pgtype.UUID    — native UUID without string conversion")
	fmt.Println("  pgtype.Array   — PostgreSQL arrays (TEXT[], INT[], etc.)")
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/upskill_go?sslmode=disable"
	}

	ctx := context.Background()
	fmt.Println("=== pgx Patterns — Transactions, pgtype, Retry ===")
	fmt.Println()

	// pgtype demo always runs (no DB needed).
	showPgTypeUsage()
	fmt.Println()

	// Show isolation levels reference.
	fmt.Println("--- PostgreSQL isolation levels ---")
	levels := []struct{ level, anomalies string }{
		{"READ COMMITTED (default)", "may read uncommitted data from concurrent txns in practice — dirty reads prevented, non-repeatable reads possible"},
		{"REPEATABLE READ", "snapshot at transaction start — phantom reads possible"},
		{"SERIALIZABLE", "full serializability — may fail with 40001, must retry"},
	}
	for _, l := range levels {
		fmt.Printf("  %-28s  %s\n", l.level, l.anomalies)
	}
	fmt.Println()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil || pool.Ping(ctx) != nil {
		fmt.Printf("  (skip live queries: no Postgres at %q)\n", dsn)
		fmt.Println()
		demoTransactionPattern()
		return
	}
	defer pool.Close()

	// Create schema.
	if _, err := pool.Exec(ctx, ddl); err != nil {
		fmt.Printf("  ✗ DDL: %v\n", err)
		return
	}
	fmt.Println("--- Schema ready ---")

	// Seed accounts.
	createAccount := func(name, email string, balance int64) (int64, error) {
		var id int64
		err := pool.QueryRow(ctx,
			`INSERT INTO accounts (name, email, balance) VALUES ($1, $2, $3) RETURNING id`,
			name,
			pgtype.Text{String: email, Valid: email != ""},
			balance,
		).Scan(&id)
		return id, err
	}

	aliceID, err := createAccount("Alice", "alice@example.com", 10000)
	if err != nil {
		fmt.Printf("  ✗ create alice: %v\n", err)
		return
	}
	bobID, err := createAccount("Bob", "", 5000) // no email → NULL
	if err != nil {
		fmt.Printf("  ✗ create bob: %v\n", err)
		return
	}
	fmt.Printf("  ✓ alice.id=%d (balance=100.00)  bob.id=%d (balance=50.00)\n", aliceID, bobID)

	// ── TRANSFER TRANSACTION ────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Transfer (serializable transaction with retry) ---")
	err = transfer(ctx, pool, aliceID, bobID, 3000, "payment for services")
	if err != nil {
		fmt.Printf("  ✗ transfer: %v\n", err)
	} else {
		fmt.Println("  ✓ transferred 30.00 alice→bob")
	}

	// Verify balances.
	accounts, err := listAccounts(ctx, pool)
	if err != nil {
		fmt.Printf("  ✗ list: %v\n", err)
	} else {
		for _, a := range accounts {
			email := "NULL"
			if a.Email.Valid {
				email = a.Email.String
			}
			fmt.Printf("  id=%-3d %-8s balance=%6d email=%s\n", a.ID, a.Name, a.Balance, email)
		}
	}

	// Insufficient funds.
	err = transfer(ctx, pool, aliceID, bobID, 999999, "huge transfer")
	if err != nil {
		fmt.Printf("  ✓ insufficient balance error: %v\n", err)
	}

	// ── ADVISORY LOCK ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Advisory lock ---")
	lockID := int64(rand.Int31())
	err = withAdvisoryLock(ctx, pool, lockID, func(tx pgx.Tx) error {
		fmt.Printf("  ✓ acquired advisory lock %d — running exclusive section\n", lockID)
		return nil
	})
	if err != nil {
		fmt.Printf("  ✗ advisory lock: %v\n", err)
	}

	// ── POOL STATS ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Pool stats ---")
	s := pool.Stat()
	fmt.Printf("  total=%d idle=%d acquired=%d max=%d\n",
		s.TotalConns(), s.IdleConns(), s.AcquiredConns(), s.MaxConns())
}

// ─────────────────────────────────────────────────────────────────────────────
// TRANSACTION PATTERN — shown when no Postgres is available
// ─────────────────────────────────────────────────────────────────────────────

func demoTransactionPattern() {
	fmt.Println("--- Transaction pattern (code walkthrough) ---")
	fmt.Println(`
  pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
  defer tx.Rollback(ctx)  // safe no-op after Commit

  tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2", amount, fromID)
  tx.Exec(ctx, "UPDATE accounts SET balance = balance + $1 WHERE id = $2", amount, toID)

  tx.Commit(ctx)

  Retry loop for 40001 (serialization_failure):
    for attempt := 1; attempt <= maxRetries; attempt++ {
        err := fn(tx)
        if isSerializationFailure(err) { continue }
        break
    }`)
	fmt.Println()
	fmt.Println("--- pgx.TxOptions isolation levels ---")
	fmt.Println("  pgx.ReadCommitted   (default)")
	fmt.Println("  pgx.RepeatableRead")
	fmt.Println("  pgx.Serializable    (retry on 40001)")
}
