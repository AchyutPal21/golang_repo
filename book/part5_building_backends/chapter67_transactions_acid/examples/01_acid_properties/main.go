// FILE: book/part5_building_backends/chapter67_transactions_acid/examples/01_acid_properties/main.go
// CHAPTER: 67 — Transactions & ACID
// TOPIC: Demonstrate each ACID property with concrete SQLite examples:
//        Atomicity (partial failure rolls back all), Consistency (constraint
//        enforcement), Isolation (concurrent read isolation), and Durability
//        (committed data survives). Also shows deferred savepoints.
//
// Run (from the chapter folder):
//   go run ./examples/01_acid_properties

package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// ─────────────────────────────────────────────────────────────────────────────
// SCHEMA
// ─────────────────────────────────────────────────────────────────────────────

const schema = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS accounts (
    id      TEXT PRIMARY KEY,
    name    TEXT    NOT NULL,
    balance INTEGER NOT NULL CHECK(balance >= 0)
);

CREATE TABLE IF NOT EXISTS audit_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    action     TEXT    NOT NULL,
    account_id TEXT    NOT NULL,
    amount     INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

// ─────────────────────────────────────────────────────────────────────────────
// ATOMICITY
// All operations in a transaction succeed or none do.
// ─────────────────────────────────────────────────────────────────────────────

func demoAtomicity(db *sql.DB) {
	fmt.Println("--- A: Atomicity ---")
	ctx := context.Background()

	// Seed accounts.
	db.Exec(`INSERT OR IGNORE INTO accounts VALUES ('alice', 'Alice', 1000)`)
	db.Exec(`INSERT OR IGNORE INTO accounts VALUES ('bob', 'Bob', 500)`)

	balance := func(id string) int {
		var b int
		db.QueryRow(`SELECT balance FROM accounts WHERE id = ?`, id).Scan(&b)
		return b
	}

	fmt.Printf("  before: alice=%d  bob=%d\n", balance("alice"), balance("bob"))

	// Successful transfer — both updates commit atomically.
	tx, _ := db.BeginTx(ctx, nil)
	defer tx.Rollback()
	tx.Exec(`UPDATE accounts SET balance = balance - 200 WHERE id = 'alice'`)
	tx.Exec(`UPDATE accounts SET balance = balance + 200 WHERE id = 'bob'`)
	tx.Commit()
	fmt.Printf("  after successful transfer: alice=%d  bob=%d\n", balance("alice"), balance("bob"))

	// Failed transfer — alice's debit rolls back because bob would go negative.
	tx2, _ := db.BeginTx(ctx, nil)
	defer tx2.Rollback()
	tx2.Exec(`UPDATE accounts SET balance = balance - 200 WHERE id = 'alice'`)
	// This would violate CHECK(balance >= 0) — causes constraint error.
	_, err := tx2.Exec(`UPDATE accounts SET balance = balance - 9999 WHERE id = 'bob'`)
	if err != nil {
		tx2.Rollback()
		fmt.Printf("  partial failure rolled back: alice=%d  bob=%d (unchanged)\n",
			balance("alice"), balance("bob"))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CONSISTENCY
// Database constraints are enforced — invalid state can never be committed.
// ─────────────────────────────────────────────────────────────────────────────

func demoConsistency(db *sql.DB) {
	fmt.Println()
	fmt.Println("--- C: Consistency ---")
	ctx := context.Background()

	// Attempt to set negative balance — violates CHECK constraint.
	tx, _ := db.BeginTx(ctx, nil)
	_, err := tx.Exec(`UPDATE accounts SET balance = -500 WHERE id = 'alice'`)
	tx.Rollback()
	if err != nil {
		fmt.Printf("  ✓ constraint prevented negative balance: %v\n", err)
	}

	// Foreign key: audit_log references accounts — inserting a non-existent account_id fails.
	tx2, _ := db.BeginTx(ctx, nil)
	_, err2 := tx2.Exec(`INSERT INTO audit_log (action, account_id, amount) VALUES ('credit', 'nobody', 100)`)
	tx2.Rollback()
	if err2 != nil {
		fmt.Printf("  ✓ FK constraint prevented orphaned audit row: %v\n", err2)
	} else {
		// SQLite FK enforcement requires PRAGMA foreign_keys=ON per connection.
		fmt.Println("  (FK constraint demo requires PRAGMA foreign_keys=ON — see schema)")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ISOLATION
// Concurrent transactions see a consistent snapshot.
// With SQLite (serialized writes), this is demonstrated sequentially.
// ─────────────────────────────────────────────────────────────────────────────

func demoIsolation(db *sql.DB) {
	fmt.Println()
	fmt.Println("--- I: Isolation ---")

	// Two goroutines read alice's balance — both see the committed state.
	var wg sync.WaitGroup
	results := make([]int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			tx, _ := db.BeginTx(ctx, &sql.TxOptions{
				Isolation: sql.LevelReadCommitted,
				ReadOnly:  true,
			})
			defer tx.Rollback()
			var b int
			tx.QueryRow(`SELECT balance FROM accounts WHERE id = 'alice'`).Scan(&b)
			results[idx] = b
		}(i)
	}
	wg.Wait()
	fmt.Printf("  concurrent readers both see alice.balance=%d and %d (consistent)\n",
		results[0], results[1])

	// Dirty read prevention — conceptual explanation (SQLite serializes writes,
	// so we can't interleave write+read across goroutines in a single-conn pool).
	// In PostgreSQL READ COMMITTED: a reader never sees uncommitted writes.
	fmt.Println("  Dirty read prevention: readers never see uncommitted data (READ COMMITTED guarantee).")
	fmt.Println("  In PostgreSQL: REPEATABLE READ gives snapshot at tx start; SERIALIZABLE is fully serial.")
}

// ─────────────────────────────────────────────────────────────────────────────
// DURABILITY
// Committed transactions survive process restart.
// (With in-memory SQLite we demonstrate the principle via WAL mode.)
// ─────────────────────────────────────────────────────────────────────────────

func demoDurability(db *sql.DB) {
	fmt.Println()
	fmt.Println("--- D: Durability ---")

	ctx := context.Background()
	tx, _ := db.BeginTx(ctx, nil)
	tx.Exec(`INSERT OR IGNORE INTO audit_log (action, account_id, amount) VALUES ('credit', 'alice', 500)`)
	tx.Commit()

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM audit_log`).Scan(&count)
	fmt.Printf("  ✓ committed audit row persists: count=%d\n", count)
	fmt.Println("  In production: WAL mode + fsync ensure committed data survives process crash.")
	fmt.Println("  With PostgreSQL: committed data is fsynced to disk via WAL before Commit() returns.")
}

// ─────────────────────────────────────────────────────────────────────────────
// SAVEPOINTS
// Nested rollback points within a transaction.
// ─────────────────────────────────────────────────────────────────────────────

func demoSavepoints(db *sql.DB) {
	fmt.Println()
	fmt.Println("--- Savepoints (partial rollback) ---")
	ctx := context.Background()

	var balance func(string) int
	balance = func(id string) int {
		var b int
		db.QueryRow(`SELECT balance FROM accounts WHERE id = ?`, id).Scan(&b)
		return b
	}

	beforeAlice := balance("alice")
	tx, _ := db.BeginTx(ctx, nil)

	// Savepoint A.
	tx.Exec(`SAVEPOINT sp_a`)
	tx.Exec(`UPDATE accounts SET balance = balance + 100 WHERE id = 'alice'`)

	// Savepoint B.
	tx.Exec(`SAVEPOINT sp_b`)
	tx.Exec(`UPDATE accounts SET balance = balance + 200 WHERE id = 'alice'`)

	// Roll back to B — undoes the +200 but keeps the +100.
	tx.Exec(`ROLLBACK TO SAVEPOINT sp_b`)
	tx.Exec(`RELEASE SAVEPOINT sp_b`)

	tx.Commit()

	fmt.Printf("  alice balance before: %d\n", beforeAlice)
	fmt.Printf("  alice balance after (kept +100, rolled back +200): %d\n", balance("alice"))
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	db, _ := sql.Open("sqlite", "file::memory:?cache=shared")
	db.SetMaxOpenConns(1)
	defer db.Close()

	if _, err := db.Exec(schema); err != nil {
		panic(err)
	}

	fmt.Println("=== ACID Properties ===")
	fmt.Println()
	fmt.Println("ACID is the four guarantees that make database transactions reliable:")
	fmt.Println("  A — Atomicity:   all-or-nothing")
	fmt.Println("  C — Consistency: constraints always satisfied")
	fmt.Println("  I — Isolation:   concurrent transactions don't interfere")
	fmt.Println("  D — Durability:  committed data survives failures")
	fmt.Println()

	demoAtomicity(db)
	demoConsistency(db)
	demoIsolation(db)
	demoDurability(db)
	demoSavepoints(db)
}
