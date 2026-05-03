// FILE: book/part5_building_backends/chapter67_transactions_acid/exercises/01_order_saga/main.go
// CHAPTER: 67 — Transactions & ACID
// EXERCISE: Build an order processing saga with atomic multi-table transactions:
//   - Schema: customers, products, orders, order_items, payments
//   - PlaceOrder: deduct inventory, create order + items, record payment — all atomic
//   - CancelOrder: restore inventory, mark order cancelled, refund payment — all atomic
//   - Optimistic locking on inventory (version column)
//   - ReserveInventory: optimistic lock with retry
//
// Run (from the chapter folder):
//   go run ./exercises/01_order_saga

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// ─────────────────────────────────────────────────────────────────────────────
// SCHEMA
// ─────────────────────────────────────────────────────────────────────────────

const schema = `
PRAGMA journal_mode=WAL;

CREATE TABLE IF NOT EXISTS customers (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    name    TEXT    NOT NULL,
    email   TEXT    NOT NULL UNIQUE,
    balance INTEGER NOT NULL DEFAULT 0 CHECK(balance >= 0)
);

CREATE TABLE IF NOT EXISTS products (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    name    TEXT    NOT NULL,
    price   INTEGER NOT NULL,  -- cents
    stock   INTEGER NOT NULL DEFAULT 0 CHECK(stock >= 0),
    version INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS orders (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id INTEGER NOT NULL REFERENCES customers(id),
    status      TEXT    NOT NULL DEFAULT 'pending', -- pending|completed|cancelled
    total_cents INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS order_items (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id   INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id),
    qty        INTEGER NOT NULL,
    unit_price INTEGER NOT NULL  -- snapshot of price at order time
);

CREATE TABLE IF NOT EXISTS payments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id   INTEGER NOT NULL REFERENCES orders(id),
    amount     INTEGER NOT NULL,
    status     TEXT    NOT NULL DEFAULT 'paid', -- paid|refunded
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

// ─────────────────────────────────────────────────────────────────────────────
// TYPES
// ─────────────────────────────────────────────────────────────────────────────

type OrderItem struct {
	ProductID int64
	Qty       int
}

type OrderResult struct {
	OrderID    int64
	Total      int
	PaymentID  int64
	ItemsCount int
}

var (
	errInsufficientStock   = errors.New("insufficient stock")
	errInsufficientBalance = errors.New("insufficient balance")
	errOrderNotFound       = errors.New("order not found")
	errAlreadyCancelled    = errors.New("order already cancelled")
	errOptimisticConflict  = errors.New("optimistic lock conflict")
)

// ─────────────────────────────────────────────────────────────────────────────
// PLACE ORDER — atomic: deduct inventory + create order/items + charge customer
// ─────────────────────────────────────────────────────────────────────────────

func placeOrder(ctx context.Context, db *sql.DB, customerID int64, items []OrderItem) (*OrderResult, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Create the order shell.
	res, err := tx.ExecContext(ctx,
		`INSERT INTO orders (customer_id, status) VALUES (?, 'pending')`, customerID)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	orderID, _ := res.LastInsertId()

	totalCents := 0

	for _, item := range items {
		// Read product with version (optimistic lock pattern via SELECT — no lock in SQLite).
		var price, stock, version int
		err := tx.QueryRowContext(ctx,
			`SELECT price, stock, version FROM products WHERE id = ?`, item.ProductID,
		).Scan(&price, &stock, &version)
		if err != nil {
			return nil, fmt.Errorf("read product %d: %w", item.ProductID, err)
		}
		if stock < item.Qty {
			return nil, fmt.Errorf("%w: product %d has %d, need %d",
				errInsufficientStock, item.ProductID, stock, item.Qty)
		}

		// Deduct stock with version check (optimistic locking).
		upd, err := tx.ExecContext(ctx,
			`UPDATE products SET stock = stock - ?, version = version + 1 WHERE id = ? AND version = ?`,
			item.Qty, item.ProductID, version)
		if err != nil {
			return nil, fmt.Errorf("deduct stock: %w", err)
		}
		if n, _ := upd.RowsAffected(); n == 0 {
			return nil, fmt.Errorf("%w on product %d", errOptimisticConflict, item.ProductID)
		}

		// Insert order item.
		_, err = tx.ExecContext(ctx,
			`INSERT INTO order_items (order_id, product_id, qty, unit_price) VALUES (?, ?, ?, ?)`,
			orderID, item.ProductID, item.Qty, price)
		if err != nil {
			return nil, fmt.Errorf("insert item: %w", err)
		}
		totalCents += price * item.Qty
	}

	// Update order total.
	tx.ExecContext(ctx, `UPDATE orders SET total_cents = ?, status = 'completed' WHERE id = ?`,
		totalCents, orderID)

	// Charge customer.
	upd, err := tx.ExecContext(ctx,
		`UPDATE customers SET balance = balance - ? WHERE id = ? AND balance >= ?`,
		totalCents, customerID, totalCents)
	if err != nil {
		return nil, fmt.Errorf("charge customer: %w", err)
	}
	if n, _ := upd.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("%w: customer %d", errInsufficientBalance, customerID)
	}

	// Record payment.
	payRes, err := tx.ExecContext(ctx,
		`INSERT INTO payments (order_id, amount) VALUES (?, ?)`, orderID, totalCents)
	if err != nil {
		return nil, fmt.Errorf("record payment: %w", err)
	}
	paymentID, _ := payRes.LastInsertId()

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &OrderResult{
		OrderID:    orderID,
		Total:      totalCents,
		PaymentID:  paymentID,
		ItemsCount: len(items),
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// CANCEL ORDER — atomic: restore inventory + cancel order + refund customer
// ─────────────────────────────────────────────────────────────────────────────

func cancelOrder(ctx context.Context, db *sql.DB, orderID int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var customerID, totalCents int64
	var status string
	err = tx.QueryRowContext(ctx,
		`SELECT customer_id, total_cents, status FROM orders WHERE id = ?`, orderID,
	).Scan(&customerID, &totalCents, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return errOrderNotFound
	}
	if err != nil {
		return err
	}
	if status == "cancelled" {
		return errAlreadyCancelled
	}

	// Restore inventory for each item.
	rows, err := tx.QueryContext(ctx,
		`SELECT product_id, qty FROM order_items WHERE order_id = ?`, orderID)
	if err != nil {
		return err
	}
	var restores []struct{ productID, qty int64 }
	for rows.Next() {
		var r struct{ productID, qty int64 }
		rows.Scan(&r.productID, &r.qty)
		restores = append(restores, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, r := range restores {
		tx.ExecContext(ctx,
			`UPDATE products SET stock = stock + ?, version = version + 1 WHERE id = ?`,
			r.qty, r.productID)
	}

	// Mark order cancelled.
	tx.ExecContext(ctx, `UPDATE orders SET status = 'cancelled' WHERE id = ?`, orderID)

	// Refund customer.
	tx.ExecContext(ctx, `UPDATE customers SET balance = balance + ? WHERE id = ?`, totalCents, customerID)

	// Record refund in payments.
	tx.ExecContext(ctx, `INSERT INTO payments (order_id, amount, status) VALUES (?, ?, 'refunded')`,
		orderID, totalCents)

	return tx.Commit()
}

// ─────────────────────────────────────────────────────────────────────────────
// REPORTING QUERIES
// ─────────────────────────────────────────────────────────────────────────────

func printOrderSummary(db *sql.DB, orderID int64) {
	var status string
	var total int
	var created time.Time
	db.QueryRow(`SELECT status, total_cents, created_at FROM orders WHERE id = ?`, orderID).
		Scan(&status, &total, &created)
	fmt.Printf("  order %d: status=%s total=$%.2f\n", orderID, status, float64(total)/100)

	rows, _ := db.Query(`
		SELECT p.name, oi.qty, oi.unit_price
		FROM order_items oi JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id = ?`, orderID)
	defer rows.Close()
	for rows.Next() {
		var name string
		var qty, price int
		rows.Scan(&name, &qty, &price)
		fmt.Printf("    %-20s x%d  @$%.2f  = $%.2f\n", name, qty,
			float64(price)/100, float64(price*qty)/100)
	}
}

func printCustomerBalance(db *sql.DB, customerID int64) {
	var name string
	var balance int
	db.QueryRow(`SELECT name, balance FROM customers WHERE id = ?`, customerID).Scan(&name, &balance)
	fmt.Printf("  customer %s: balance=$%.2f\n", name, float64(balance)/100)
}

func printStock(db *sql.DB) {
	rows, _ := db.Query(`SELECT name, stock FROM products ORDER BY id`)
	defer rows.Close()
	for rows.Next() {
		var name string
		var stock int
		rows.Scan(&name, &stock)
		fmt.Printf("  %-20s stock=%d\n", name, stock)
	}
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

	ctx := context.Background()

	// Seed.
	db.Exec(`INSERT INTO customers (name, email, balance) VALUES ('Alice', 'alice@example.com', 50000)`) // $500
	db.Exec(`INSERT INTO customers (name, email, balance) VALUES ('Bob', 'bob@example.com', 1000)`)      // $10
	db.Exec(`INSERT INTO products (name, price, stock) VALUES ('Keyboard', 12999, 10)`)                  // $129.99
	db.Exec(`INSERT INTO products (name, price, stock) VALUES ('Mouse', 4999, 5)`)                       // $49.99
	db.Exec(`INSERT INTO products (name, price, stock) VALUES ('Monitor', 34999, 2)`)                    // $349.99

	check := func(label string, err error) {
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", label, err)
		} else {
			fmt.Printf("  ✓ %s\n", label)
		}
	}

	fmt.Println("=== Order Saga (Atomic Transactions) ===")

	// ── STOCK BEFORE ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Initial state ---")
	printStock(db)
	printCustomerBalance(db, 1) // alice
	printCustomerBalance(db, 2) // bob

	// ── SUCCESSFUL ORDER ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Alice places order: keyboard + mouse ---")
	result, err := placeOrder(ctx, db, 1, []OrderItem{
		{ProductID: 1, Qty: 1}, // keyboard
		{ProductID: 2, Qty: 2}, // 2x mouse
	})
	check("place order", err)
	if err == nil {
		fmt.Printf("  order.id=%d payment.id=%d total=$%.2f\n",
			result.OrderID, result.PaymentID, float64(result.Total)/100)
		printOrderSummary(db, result.OrderID)
		printCustomerBalance(db, 1)
	}

	// ── STOCK AFTER ORDER ───────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Stock after Alice's order ---")
	printStock(db)

	// ── INSUFFICIENT FUNDS ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Bob tries to order monitor ($349.99, only $10 balance) ---")
	_, err = placeOrder(ctx, db, 2, []OrderItem{{ProductID: 3, Qty: 1}})
	if errors.Is(err, errInsufficientBalance) {
		fmt.Printf("  ✓ rejected: %v\n", err)
	} else {
		check("bob order should fail", err)
	}

	// Stock must be unchanged after rollback.
	fmt.Println("--- Stock unchanged after failed order ---")
	printStock(db)

	// ── INSUFFICIENT STOCK ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Alice tries to order 6 mice (only 3 remaining) ---")
	_, err = placeOrder(ctx, db, 1, []OrderItem{{ProductID: 2, Qty: 6}})
	if errors.Is(err, errInsufficientStock) {
		fmt.Printf("  ✓ rejected: %v\n", err)
	} else {
		check("insufficient stock should fail", err)
	}

	// ── CANCEL ORDER ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Cancel Alice's order ---")
	check("cancel order", cancelOrder(ctx, db, result.OrderID))
	printOrderSummary(db, result.OrderID)
	printCustomerBalance(db, 1)

	fmt.Println()
	fmt.Println("--- Stock restored after cancellation ---")
	printStock(db)

	// ── DOUBLE CANCEL ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Double-cancel should fail ---")
	err = cancelOrder(ctx, db, result.OrderID)
	if errors.Is(err, errAlreadyCancelled) {
		fmt.Printf("  ✓ double cancel rejected: %v\n", err)
	} else {
		check("double cancel", err)
	}
}
