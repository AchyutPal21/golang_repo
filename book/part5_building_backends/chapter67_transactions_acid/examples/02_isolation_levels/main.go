// FILE: book/part5_building_backends/chapter67_transactions_acid/examples/02_isolation_levels/main.go
// CHAPTER: 67 — Transactions & ACID
// TOPIC: SQL isolation levels — what anomalies each prevents, how to choose,
//        optimistic locking with version columns, and deadlock detection.
//
// Run (from the chapter folder):
//   go run ./examples/02_isolation_levels

package main

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

// ─────────────────────────────────────────────────────────────────────────────
// ISOLATION LEVEL REFERENCE
// ─────────────────────────────────────────────────────────────────────────────

func showIsolationLevels() {
	fmt.Println("--- Isolation levels and the anomalies they prevent ---")
	fmt.Println()

	type level struct {
		name        string
		dirtyRead   string
		nonRepeat   string
		phantom     string
		serial      string
		useCase     string
	}
	levels := []level{
		{"READ UNCOMMITTED", "possible", "possible", "possible", "no", "Analytics reads — dirty data acceptable"},
		{"READ COMMITTED", "prevented", "possible", "possible", "no", "Default in PostgreSQL/MySQL — most OLTP"},
		{"REPEATABLE READ", "prevented", "prevented", "possible*", "no", "Long reports, avoid re-reading changed rows"},
		{"SERIALIZABLE", "prevented", "prevented", "prevented", "yes", "Financial, inventory — correctness critical"},
	}
	fmt.Printf("  %-20s  %-10s  %-12s  %-10s  %-8s  %s\n",
		"Level", "DirtyRead", "NonRepeat", "Phantom", "Serial", "Use case")
	fmt.Println("  " + fmt.Sprintf("%s", repeatStr("-", 100)))
	for _, l := range levels {
		fmt.Printf("  %-20s  %-10s  %-12s  %-10s  %-8s  %s\n",
			l.name, l.dirtyRead, l.nonRepeat, l.phantom, l.serial, l.useCase)
	}
	fmt.Println()
	fmt.Println("  * MySQL REPEATABLE READ prevents phantoms via gap locks; PostgreSQL does not use gap locks.")
	fmt.Println("  Go sql.LevelSerializable / pgx.Serializable — most strict, may need retry on 40001.")
}

func repeatStr(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// OPTIMISTIC LOCKING — version column, no locks held
// ─────────────────────────────────────────────────────────────────────────────

const schemaOpt = `
CREATE TABLE IF NOT EXISTS inventory (
    id      TEXT    PRIMARY KEY,
    product TEXT    NOT NULL,
    qty     INTEGER NOT NULL CHECK(qty >= 0),
    version INTEGER NOT NULL DEFAULT 1
);
`

type InventoryRow struct {
	ID      string
	Product string
	Qty     int
	Version int
}

func getInventory(db *sql.DB, id string) (*InventoryRow, error) {
	var row InventoryRow
	err := db.QueryRow(
		`SELECT id, product, qty, version FROM inventory WHERE id = ?`, id,
	).Scan(&row.ID, &row.Product, &row.Qty, &row.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("inventory %q: not found", id)
	}
	return &row, err
}

// updateQtyOptimistic uses a version column to detect concurrent modification.
// Returns errConflict if another process modified the row since we read it.
var errConflict = errors.New("optimistic lock conflict: row was modified concurrently")

func updateQtyOptimistic(db *sql.DB, id string, newQty, expectedVersion int) error {
	res, err := db.Exec(
		`UPDATE inventory SET qty = ?, version = version + 1 WHERE id = ? AND version = ?`,
		newQty, id, expectedVersion,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errConflict
	}
	return nil
}

func demoOptimisticLocking(db *sql.DB) {
	fmt.Println("--- Optimistic locking (version column) ---")
	db.Exec(`INSERT OR IGNORE INTO inventory VALUES ('inv-1', 'Widget', 100, 1)`)

	row, _ := getInventory(db, "inv-1")
	fmt.Printf("  read: qty=%d version=%d\n", row.Qty, row.Version)

	// Simulate another process modifying the row between our read and write.
	db.Exec(`UPDATE inventory SET qty = 90, version = version + 1 WHERE id = 'inv-1'`)

	// Our update uses the stale version — should fail.
	err := updateQtyOptimistic(db, "inv-1", 95, row.Version)
	if errors.Is(err, errConflict) {
		fmt.Printf("  ✓ conflict detected: %v\n", err)
		// Re-read and retry.
		row2, _ := getInventory(db, "inv-1")
		err = updateQtyOptimistic(db, "inv-1", 95, row2.Version)
		if err == nil {
			fmt.Printf("  ✓ retry succeeded: qty=95 version=%d\n", row2.Version+1)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DEADLOCK DETECTION (conceptual — demonstrated via error simulation)
// ─────────────────────────────────────────────────────────────────────────────

func demoDeadlockConcept() {
	fmt.Println()
	fmt.Println("--- Deadlock (conceptual) ---")
	fmt.Println(`
  A deadlock occurs when two transactions each hold a lock the other needs:

  T1: LOCK accounts WHERE id='alice'   (T1 holds alice)
  T2: LOCK accounts WHERE id='bob'     (T2 holds bob)
  T1: LOCK accounts WHERE id='bob'     (T1 waits — T2 holds bob)
  T2: LOCK accounts WHERE id='alice'   (T2 waits — T1 holds alice) ← deadlock!

  PostgreSQL detects deadlocks automatically and aborts one transaction with:
    SQLSTATE 40P01 (deadlock_detected)

  Go handling:
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) && pgErr.Code == "40P01" {
        // retry the transaction
    }

  Prevention: always lock rows in the same order (e.g. sort IDs before locking).
  In transfer: always debit/credit lower ID first.`)
}

// ─────────────────────────────────────────────────────────────────────────────
// TRANSACTION PATTERNS SUMMARY
// ─────────────────────────────────────────────────────────────────────────────

func showPatterns() {
	fmt.Println()
	fmt.Println("--- Transaction patterns ---")
	fmt.Println(`
  1. Simple transaction (database/sql):
     tx, _ := db.BeginTx(ctx, nil)
     defer tx.Rollback()           // safe no-op after Commit
     // ... operations ...
     return tx.Commit()

  2. Serializable with retry (pgx):
     for attempt := 1; attempt <= 3; attempt++ {
         tx, _ := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
         err := fn(tx)
         if isSerializationFailure(err) { tx.Rollback(ctx); continue }
         if err == nil { err = tx.Commit(ctx) }
         if isSerializationFailure(err) { tx.Rollback(ctx); continue }
         return err
     }

  3. Optimistic locking (no DB-level locks):
     row := readRow()
     compute new value
     UPDATE ... WHERE version = row.Version  → check RowsAffected
     if 0 rows updated → conflict, re-read and retry

  4. Savepoints (partial rollback):
     SAVEPOINT sp_a
     // risky operation A
     ROLLBACK TO SAVEPOINT sp_a   // undoes A but keeps outer tx
     RELEASE SAVEPOINT sp_a

  5. Advisory locks (application-level mutex):
     SELECT pg_try_advisory_xact_lock($1)  // holds for transaction duration`)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	db, _ := sql.Open("sqlite", "file::memory:?cache=shared")
	db.SetMaxOpenConns(1)
	defer db.Close()

	db.Exec(schemaOpt)

	fmt.Println("=== Isolation Levels & Transaction Patterns ===")
	fmt.Println()

	showIsolationLevels()
	demoOptimisticLocking(db)
	demoDeadlockConcept()
	showPatterns()
}
