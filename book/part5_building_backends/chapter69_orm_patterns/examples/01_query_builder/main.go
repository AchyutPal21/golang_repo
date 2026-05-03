// FILE: book/part5_building_backends/chapter69_orm_patterns/examples/01_query_builder/main.go
// CHAPTER: 69 — ORM vs Builder vs Raw SQL
// TOPIC: Build a type-safe SQL query builder from scratch:
//        SELECT with WHERE/ORDER/LIMIT/OFFSET, INSERT, UPDATE, DELETE,
//        parameterised placeholders, and compare Raw vs Builder vs ORM tradeoffs.
//
// Run (from the chapter folder):
//   go run ./examples/01_query_builder

package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// ─────────────────────────────────────────────────────────────────────────────
// QUERY BUILDER
// ─────────────────────────────────────────────────────────────────────────────

// QB builds SQL queries with positional placeholders (?).
type QB struct {
	table      string
	cols       []string
	conditions []string
	args       []any
	orderBy    string
	limit      int
	offset     int
	op         string // SELECT|INSERT|UPDATE|DELETE
	setClauses []string
}

func From(table string) *QB {
	return &QB{table: table, op: "SELECT", limit: -1}
}

func (q *QB) Select(cols ...string) *QB {
	q.cols = cols
	return q
}

func (q *QB) Where(cond string, args ...any) *QB {
	q.conditions = append(q.conditions, cond)
	q.args = append(q.args, args...)
	return q
}

func (q *QB) OrderBy(col string) *QB {
	q.orderBy = col
	return q
}

func (q *QB) Limit(n int) *QB {
	q.limit = n
	return q
}

func (q *QB) Offset(n int) *QB {
	q.offset = n
	return q
}

func (q *QB) Build() (string, []any) {
	switch q.op {
	case "SELECT":
		return q.buildSelect()
	case "INSERT":
		return q.buildInsert()
	case "UPDATE":
		return q.buildUpdate()
	case "DELETE":
		return q.buildDelete()
	}
	return "", nil
}

func (q *QB) buildSelect() (string, []any) {
	cols := "*"
	if len(q.cols) > 0 {
		cols = strings.Join(q.cols, ", ")
	}
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "SELECT %s FROM %s", cols, q.table)
	if len(q.conditions) > 0 {
		fmt.Fprintf(sb, " WHERE %s", strings.Join(q.conditions, " AND "))
	}
	if q.orderBy != "" {
		fmt.Fprintf(sb, " ORDER BY %s", q.orderBy)
	}
	if q.limit >= 0 {
		fmt.Fprintf(sb, " LIMIT %d", q.limit)
	}
	if q.offset > 0 {
		fmt.Fprintf(sb, " OFFSET %d", q.offset)
	}
	return sb.String(), q.args
}

// Insert returns a new QB configured for INSERT.
func Insert(table string, colsAndVals ...any) *QB {
	q := &QB{table: table, op: "INSERT"}
	// colsAndVals: "col1", val1, "col2", val2, ...
	for i := 0; i+1 < len(colsAndVals); i += 2 {
		col, _ := colsAndVals[i].(string)
		q.cols = append(q.cols, col)
		q.args = append(q.args, colsAndVals[i+1])
	}
	return q
}

func (q *QB) buildInsert() (string, []any) {
	placeholders := make([]string, len(q.cols))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		q.table,
		strings.Join(q.cols, ", "),
		strings.Join(placeholders, ", "),
	), q.args
}

// Update returns a new QB for UPDATE.
func Update(table string) *QB {
	return &QB{table: table, op: "UPDATE"}
}

func (q *QB) Set(col string, val any) *QB {
	q.setClauses = append(q.setClauses, col+" = ?")
	q.args = append(q.args, val)
	return q
}

func (q *QB) buildUpdate() (string, []any) {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "UPDATE %s SET %s", q.table, strings.Join(q.setClauses, ", "))
	whereArgs := q.args[len(q.setClauses):]
	if len(q.conditions) > 0 {
		fmt.Fprintf(sb, " WHERE %s", strings.Join(q.conditions, " AND "))
	}
	return sb.String(), append(q.args[:len(q.setClauses)], whereArgs...)
}

// Delete returns a QB for DELETE.
func Delete(table string) *QB {
	return &QB{table: table, op: "DELETE"}
}

func (q *QB) buildDelete() (string, []any) {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "DELETE FROM %s", q.table)
	if len(q.conditions) > 0 {
		fmt.Fprintf(sb, " WHERE %s", strings.Join(q.conditions, " AND "))
	}
	return sb.String(), q.args
}

// ─────────────────────────────────────────────────────────────────────────────
// SCHEMA
// ─────────────────────────────────────────────────────────────────────────────

const schema = `
CREATE TABLE IF NOT EXISTS products (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    name     TEXT    NOT NULL,
    category TEXT    NOT NULL,
    price    INTEGER NOT NULL,  -- cents
    stock    INTEGER NOT NULL DEFAULT 0
);
`

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	db, _ := sql.Open("sqlite", "file::memory:?cache=shared")
	db.SetMaxOpenConns(1)
	defer db.Close()
	db.Exec(schema)

	ctx := context.Background()

	// ── INSERT ────────────────────────────────────────────────────────────────
	fmt.Println("=== Query Builder Demo ===")
	fmt.Println()
	fmt.Println("--- INSERT ---")

	products := []struct{ name, cat string; price, stock int }{
		{"Keyboard", "electronics", 12999, 50},
		{"Mouse", "electronics", 4999, 120},
		{"Monitor", "electronics", 34999, 15},
		{"Notebook", "office", 299, 500},
		{"Pen Pack", "office", 199, 1000},
		{"Desk Chair", "furniture", 29900, 10},
	}
	for _, p := range products {
		q, args := Insert("products", "name", p.name, "category", p.cat, "price", p.price, "stock", p.stock).Build()
		db.ExecContext(ctx, q, args...)
	}
	fmt.Printf("  inserted %d products\n", len(products))

	// ── SELECT ALL ────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- SELECT all ---")
	q, args := From("products").Build()
	fmt.Printf("  SQL: %s\n", q)

	rows, _ := db.QueryContext(ctx, q, args...)
	defer rows.Close()
	for rows.Next() {
		var id, price, stock int
		var name, cat string
		rows.Scan(&id, &name, &cat, &price, &stock)
		fmt.Printf("  id=%-2d %-12s %-11s $%6.2f stock=%d\n", id, name, cat, float64(price)/100, stock)
	}

	// ── SELECT with WHERE + ORDER + LIMIT ─────────────────────────────────────
	fmt.Println()
	fmt.Println("--- SELECT electronics, ordered by price ASC, limit 2 ---")
	q, args = From("products").
		Select("id", "name", "price").
		Where("category = ?", "electronics").
		OrderBy("price ASC").
		Limit(2).
		Build()
	fmt.Printf("  SQL: %s  args=%v\n", q, args)
	rows2, _ := db.QueryContext(ctx, q, args...)
	defer rows2.Close()
	for rows2.Next() {
		var id, price int
		var name string
		rows2.Scan(&id, &name, &price)
		fmt.Printf("  id=%d name=%-10s $%.2f\n", id, name, float64(price)/100)
	}

	// ── SELECT with multiple WHERE ─────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- SELECT WHERE category=electronics AND price < 10000 ---")
	q, args = From("products").
		Where("category = ?", "electronics").
		Where("price < ?", 10000).
		OrderBy("name ASC").
		Build()
	fmt.Printf("  SQL: %s\n", q)
	rows3, _ := db.QueryContext(ctx, q, args...)
	defer rows3.Close()
	for rows3.Next() {
		var id, price, stock int
		var name, cat string
		rows3.Scan(&id, &name, &cat, &price, &stock)
		fmt.Printf("  %-10s $%.2f\n", name, float64(price)/100)
	}

	// ── UPDATE ────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- UPDATE price for keyboards ---")
	q, args = Update("products").
		Set("price", 11999).
		Set("stock", 45).
		Where("name = ?", "Keyboard").
		Build()
	fmt.Printf("  SQL: %s  args=%v\n", q, args)
	res, _ := db.ExecContext(ctx, q, args...)
	n, _ := res.RowsAffected()
	fmt.Printf("  rows affected: %d\n", n)

	// ── DELETE ────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- DELETE office products with stock > 900 ---")
	q, args = Delete("products").
		Where("category = ?", "office").
		Where("stock > ?", 900).
		Build()
	fmt.Printf("  SQL: %s  args=%v\n", q, args)
	res2, _ := db.ExecContext(ctx, q, args...)
	n2, _ := res2.RowsAffected()
	fmt.Printf("  rows deleted: %d\n", n2)

	// ── PAGINATION ────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Pagination (page 1, size 2) ---")
	pageSize, page := 2, 0
	q, args = From("products").
		OrderBy("id ASC").
		Limit(pageSize).
		Offset(page * pageSize).
		Build()
	fmt.Printf("  SQL: %s\n", q)
	rows4, _ := db.QueryContext(ctx, q, args...)
	defer rows4.Close()
	for rows4.Next() {
		var id, price, stock int
		var name, cat string
		rows4.Scan(&id, &name, &cat, &price, &stock)
		fmt.Printf("  id=%d %-12s $%.2f\n", id, name, float64(price)/100)
	}

	// ── TRADEOFFS ────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("=== Raw SQL vs Builder vs ORM ===")
	tradeoffs := []struct{ approach, pros, cons string }{
		{"Raw SQL",
			"Full control, most performant, readable",
			"Tedious for dynamic queries, typos in strings"},
		{"Query Builder",
			"Type-safe params, composable conditions, no SQL injection risk",
			"Overhead vs raw, limited to builder's API"},
		{"ORM (GORM/Ent)",
			"Rapid development, migrations, relationships, hooks",
			"Magic queries, N+1 risk, performance surprises, large dep"},
	}
	for _, t := range tradeoffs {
		fmt.Printf("\n  %-16s\n    PROs: %s\n    CONs: %s\n", t.approach, t.pros, t.cons)
	}
	fmt.Println()
	fmt.Println("  Recommendation:")
	fmt.Println("    - Small teams / CRUD-heavy: ORM for speed, add raw for complex queries")
	fmt.Println("    - Performance-critical paths: raw SQL or query builder")
	fmt.Println("    - Dynamic filter APIs: query builder prevents injection while staying flexible")
}
