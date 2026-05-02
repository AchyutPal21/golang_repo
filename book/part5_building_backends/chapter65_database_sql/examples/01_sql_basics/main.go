// FILE: book/part5_building_backends/chapter65_database_sql/examples/01_sql_basics/main.go
// CHAPTER: 65 — database/sql
// TOPIC: Go's database/sql package fundamentals —
//        opening a connection pool, CRUD with Query/QueryRow/Exec,
//        prepared statements, parameter placeholders, scanning rows,
//        NULL handling, and transactions.
//
// Uses modernc.org/sqlite (pure-Go, no CGO required) as the database driver.
// The patterns shown here apply identically to any sql.DB driver (postgres, mysql).
//
// Run (from the chapter folder):
//   go run ./examples/01_sql_basics

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite" // register "sqlite" driver
)

// ─────────────────────────────────────────────────────────────────────────────
// SCHEMA
// ─────────────────────────────────────────────────────────────────────────────

const schema = `
CREATE TABLE IF NOT EXISTS authors (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    bio        TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS books (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT NOT NULL,
    author_id  INTEGER NOT NULL REFERENCES authors(id),
    year       INTEGER NOT NULL,
    isbn       TEXT UNIQUE,
    price      REAL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Author struct {
	ID        int64
	Name      string
	Bio       sql.NullString // NULL-able
	CreatedAt time.Time
}

type Book struct {
	ID        int64
	Title     string
	AuthorID  int64
	Year      int
	ISBN      sql.NullString  // UNIQUE, nullable
	Price     sql.NullFloat64 // nullable
	CreatedAt time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// DATABASE HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func mustOpen(driver, dsn string) *sql.DB {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	return db
}

// ─────────────────────────────────────────────────────────────────────────────
// INSERT
// ─────────────────────────────────────────────────────────────────────────────

func insertAuthor(db *sql.DB, name, bio string) (int64, error) {
	// Use Exec for INSERT/UPDATE/DELETE.
	// Always use parameter placeholders ($1 / ? / @p1) — never string-format SQL.
	var bioVal interface{}
	if bio != "" {
		bioVal = bio
	} // else nil → NULL in the DB

	res, err := db.Exec(
		`INSERT INTO authors (name, bio) VALUES (?, ?)`,
		name, bioVal,
	)
	if err != nil {
		return 0, fmt.Errorf("insertAuthor: %w", err)
	}
	return res.LastInsertId()
}

func insertBook(db *sql.DB, title string, authorID int64, year int, isbn string, price float64) (int64, error) {
	var isbnVal, priceVal interface{}
	if isbn != "" {
		isbnVal = isbn
	}
	if price > 0 {
		priceVal = price
	}
	res, err := db.Exec(
		`INSERT INTO books (title, author_id, year, isbn, price) VALUES (?, ?, ?, ?, ?)`,
		title, authorID, year, isbnVal, priceVal,
	)
	if err != nil {
		return 0, fmt.Errorf("insertBook: %w", err)
	}
	return res.LastInsertId()
}

// ─────────────────────────────────────────────────────────────────────────────
// QUERY
// ─────────────────────────────────────────────────────────────────────────────

// getAuthor uses QueryRow — at most one result.
func getAuthor(db *sql.DB, id int64) (*Author, error) {
	var a Author
	var createdAt string
	err := db.QueryRow(
		`SELECT id, name, bio, created_at FROM authors WHERE id = ?`, id,
	).Scan(&a.ID, &a.Name, &a.Bio, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("author %d: not found", id)
		}
		return nil, fmt.Errorf("getAuthor: %w", err)
	}
	a.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &a, nil
}

// listBooks uses Query — multiple results.
func listBooks(db *sql.DB, authorID int64) ([]*Book, error) {
	rows, err := db.Query(
		`SELECT id, title, author_id, year, isbn, price FROM books WHERE author_id = ? ORDER BY year`,
		authorID,
	)
	if err != nil {
		return nil, fmt.Errorf("listBooks: %w", err)
	}
	defer rows.Close() // always close rows to release the connection back to the pool

	var books []*Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.AuthorID, &b.Year, &b.ISBN, &b.Price); err != nil {
			return nil, fmt.Errorf("listBooks scan: %w", err)
		}
		books = append(books, &b)
	}
	// rows.Err() must be checked — Next() may have stopped due to an error.
	return books, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// PREPARED STATEMENT
// ─────────────────────────────────────────────────────────────────────────────

// bulkInsertBooks uses a prepared statement to insert multiple rows efficiently.
// The statement is compiled once; each Exec reuses the compiled plan.
func bulkInsertBooks(db *sql.DB, books []Book) error {
	stmt, err := db.Prepare(`INSERT INTO books (title, author_id, year) VALUES (?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, b := range books {
		if _, err := stmt.Exec(b.Title, b.AuthorID, b.Year); err != nil {
			return fmt.Errorf("bulk insert %q: %w", b.Title, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// TRANSACTION
// ─────────────────────────────────────────────────────────────────────────────

// transferBooks moves all books from one author to another atomically.
func transferBooks(db *sql.DB, fromAuthorID, toAuthorID int64) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	// Defer a rollback — if Commit succeeds, the tx is already committed and
	// the deferred Rollback is a no-op.
	defer tx.Rollback()

	res, err := tx.Exec(
		`UPDATE books SET author_id = ? WHERE author_id = ?`,
		toAuthorID, fromAuthorID,
	)
	if err != nil {
		return 0, fmt.Errorf("update books: %w", err)
	}
	n, _ := res.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return n, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// UPDATE / DELETE
// ─────────────────────────────────────────────────────────────────────────────

func updateAuthorBio(db *sql.DB, id int64, bio string) error {
	res, err := db.Exec(`UPDATE authors SET bio = ? WHERE id = ?`, bio, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("author %d not found", id)
	}
	return nil
}

func deleteBook(db *sql.DB, id int64) (bool, error) {
	res, err := db.Exec(`DELETE FROM books WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// :memory: — in-memory SQLite database, gone when the process exits.
	db := mustOpen("sqlite", ":memory:")
	defer db.Close()

	if _, err := db.Exec(schema); err != nil {
		log.Fatal("schema:", err)
	}

	fmt.Println("=== database/sql Basics ===")

	check := func(label string, err error) {
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", label, err)
		} else {
			fmt.Printf("  ✓ %s\n", label)
		}
	}

	// ── INSERT ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- INSERT ---")
	aliceID, err := insertAuthor(db, "Alice Chen", "Go and distributed systems expert")
	check("insert Alice", err)
	bobID, err := insertAuthor(db, "Bob Torres", "") // empty bio → NULL
	check("insert Bob (NULL bio)", err)
	fmt.Printf("  alice.id=%d  bob.id=%d\n", aliceID, bobID)

	b1ID, err := insertBook(db, "Concurrency in Go", aliceID, 2023, "978-0-01-000001-1", 49.99)
	check("insert book 1", err)
	b2ID, err := insertBook(db, "Distributed Go", aliceID, 2024, "978-0-01-000002-2", 54.99)
	check("insert book 2", err)
	_, _ = b1ID, b2ID

	// ── QUERY ROW ───────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- QueryRow (single result) ---")
	alice, err := getAuthor(db, aliceID)
	check("get Alice", err)
	if alice != nil {
		fmt.Printf("  name=%s  bio.Valid=%v  bio=%q\n", alice.Name, alice.Bio.Valid, alice.Bio.String)
	}
	bob, err := getAuthor(db, bobID)
	check("get Bob", err)
	if bob != nil {
		fmt.Printf("  name=%s  bio.Valid=%v (NULL bio)\n", bob.Name, bob.Bio.Valid)
	}

	// ErrNoRows.
	_, err = getAuthor(db, 9999)
	fmt.Printf("  ✓ getAuthor(9999): %v\n", err)

	// ── QUERY MULTIPLE ROWS ─────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Query (multiple rows) ---")
	books, err := listBooks(db, aliceID)
	check("list Alice's books", err)
	for _, b := range books {
		fmt.Printf("  id=%d title=%q year=%d isbn=%q price=%.2f\n",
			b.ID, b.Title, b.Year, b.ISBN.String, b.Price.Float64)
	}

	// ── PREPARED STATEMENT ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Prepared statement (bulk insert) ---")
	err = bulkInsertBooks(db, []Book{
		{Title: "Go Microservices", AuthorID: bobID, Year: 2022},
		{Title: "Go and gRPC", AuthorID: bobID, Year: 2023},
		{Title: "Go Testing Guide", AuthorID: bobID, Year: 2024},
	})
	check("bulk insert 3 books for Bob", err)
	bobBooks, _ := listBooks(db, bobID)
	fmt.Printf("  Bob now has %d books\n", len(bobBooks))

	// ── UPDATE ───────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- UPDATE ---")
	err = updateAuthorBio(db, bobID, "gRPC and microservices specialist")
	check("update Bob's bio", err)
	bob2, _ := getAuthor(db, bobID)
	fmt.Printf("  Bob bio: %q\n", bob2.Bio.String)

	// ── TRANSACTION ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- TRANSACTION ---")
	n, err := transferBooks(db, bobID, aliceID)
	check("transfer Bob→Alice", err)
	fmt.Printf("  transferred %d books\n", n)
	aliceBooks, _ := listBooks(db, aliceID)
	bobBooksAfter, _ := listBooks(db, bobID)
	fmt.Printf("  Alice now has %d books; Bob has %d\n", len(aliceBooks), len(bobBooksAfter))

	// ── DELETE ───────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- DELETE ---")
	deleted, err := deleteBook(db, b1ID)
	check("delete book 1", err)
	fmt.Printf("  deleted=%v\n", deleted)
	deleted, err = deleteBook(db, 9999)
	check("delete non-existent book (rows=0)", err)
	fmt.Printf("  deleted=%v (no rows affected)\n", deleted)

	// ── NULL HANDLING ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- NULL handling ---")
	fmt.Println("  sql.NullString: {String, Valid}")
	fmt.Println("  Valid=true  → field has a value")
	fmt.Println("  Valid=false → field is NULL in the database")
	fmt.Printf("  alice.Bio: Valid=%v String=%q\n", alice.Bio.Valid, alice.Bio.String)
	fmt.Printf("  bob.Bio:   Valid=%v (originally NULL)\n", bob.Bio.Valid)
}
