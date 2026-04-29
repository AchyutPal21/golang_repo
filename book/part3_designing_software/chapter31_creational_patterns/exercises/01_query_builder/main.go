// EXERCISE 31.1 — Build a SQL query builder using the Builder pattern.
//
// QueryBuilder constructs SELECT statements step by step with method chaining.
// Build() validates the result and returns the SQL string.
//
// Run (from the chapter folder):
//   go run ./exercises/01_query_builder

package main

import (
	"fmt"
	"strings"
)

// ─── Query Builder ────────────────────────────────────────────────────────────

type QueryBuilder struct {
	table      string
	columns    []string
	conditions []string
	orderBy    string
	limit      int
	err        error
}

func Select(columns ...string) *QueryBuilder {
	if len(columns) == 0 {
		return &QueryBuilder{err: fmt.Errorf("Select requires at least one column")}
	}
	return &QueryBuilder{columns: columns}
}

func (q *QueryBuilder) From(table string) *QueryBuilder {
	if q.err != nil {
		return q
	}
	if strings.TrimSpace(table) == "" {
		q.err = fmt.Errorf("From: table name cannot be empty")
		return q
	}
	q.table = table
	return q
}

func (q *QueryBuilder) Where(condition string) *QueryBuilder {
	if q.err != nil {
		return q
	}
	q.conditions = append(q.conditions, condition)
	return q
}

func (q *QueryBuilder) OrderBy(column string) *QueryBuilder {
	if q.err != nil {
		return q
	}
	q.orderBy = column
	return q
}

func (q *QueryBuilder) Limit(n int) *QueryBuilder {
	if q.err != nil {
		return q
	}
	if n <= 0 {
		q.err = fmt.Errorf("Limit must be positive, got %d", n)
		return q
	}
	q.limit = n
	return q
}

func (q *QueryBuilder) Build() (string, error) {
	if q.err != nil {
		return "", q.err
	}
	if q.table == "" {
		return "", fmt.Errorf("Build: From() is required")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "SELECT %s FROM %s", strings.Join(q.columns, ", "), q.table)
	if len(q.conditions) > 0 {
		fmt.Fprintf(&sb, " WHERE %s", strings.Join(q.conditions, " AND "))
	}
	if q.orderBy != "" {
		fmt.Fprintf(&sb, " ORDER BY %s", q.orderBy)
	}
	if q.limit > 0 {
		fmt.Fprintf(&sb, " LIMIT %d", q.limit)
	}
	return sb.String(), nil
}

// ─── Product factory: creates a family of pre-configured queries ──────────────

type QueryTemplate struct{ builder func() *QueryBuilder }

func NewQueryTemplate(b func() *QueryBuilder) *QueryTemplate {
	return &QueryTemplate{builder: b}
}

func (t *QueryTemplate) WithCondition(condition string) (string, error) {
	return t.builder().Where(condition).Build()
}

func main() {
	fmt.Println("=== Query Builder ===")

	// Simple query
	sql, err := Select("id", "name", "email").
		From("users").
		Build()
	fmt.Printf("  %s  err=%v\n", sql, err)

	// Full query with conditions, order, limit
	sql, err = Select("id", "title", "published_at").
		From("articles").
		Where("published_at IS NOT NULL").
		Where("author_id = 42").
		OrderBy("published_at DESC").
		Limit(10).
		Build()
	fmt.Printf("  %s  err=%v\n", sql, err)

	// Wildcard
	sql, err = Select("*").
		From("products").
		Where("stock > 0").
		OrderBy("price ASC").
		Build()
	fmt.Printf("  %s  err=%v\n", sql, err)

	fmt.Println()
	fmt.Println("=== Validation errors ===")

	_, err = Select().From("users").Build()
	fmt.Println("  no columns:", err)

	_, err = Select("id").From("").Build()
	fmt.Println("  empty table:", err)

	_, err = Select("id").From("orders").Limit(-5).Build()
	fmt.Println("  bad limit:", err)

	_, err = Select("id").Build()
	fmt.Println("  missing From:", err)

	fmt.Println()
	fmt.Println("=== Query Template (Prototype-style reuse) ===")
	activeUsersTemplate := NewQueryTemplate(func() *QueryBuilder {
		return Select("id", "email", "last_login").
			From("users").
			Where("active = true").
			OrderBy("last_login DESC")
	})

	q1, _ := activeUsersTemplate.WithCondition("plan = 'pro'")
	fmt.Println("  pro users:", q1)

	q2, _ := activeUsersTemplate.WithCondition("country = 'US'")
	fmt.Println("  US users: ", q2)
}
