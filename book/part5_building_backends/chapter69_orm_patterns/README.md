# Chapter 69 — ORM vs Builder vs Raw SQL

## What you'll learn

Three approaches to database access in Go — raw SQL strings, a custom query builder, and the repository pattern — with the tradeoffs of each. You'll build a type-safe query builder from scratch, implement the repository interface in both in-memory and SQL forms, and add a transparent caching layer.

## The three approaches

| Approach | Best for | Watch out for |
|---|---|---|
| **Raw SQL** | One-off queries, complex joins, performance-critical paths | String typos, injection risk if args not parameterised |
| **Query Builder** | Dynamic filter APIs, composable conditions | Builder's API limits expressiveness for advanced SQL |
| **ORM** (GORM/Ent) | Rapid CRUD, migrations, relationships | N+1 queries, magic behavior, hidden performance cost |

## Files

| File | Topic |
|---|---|
| `examples/01_query_builder/main.go` | `QB` type — SELECT/INSERT/UPDATE/DELETE with composable WHERE/ORDER/LIMIT |
| `examples/02_repository_pattern/main.go` | `ProductRepository` interface, in-memory and SQL implementations |
| `exercises/01_product_repository/main.go` | Full repository with search, stock management, and caching wrapper |

## Query builder pattern

```go
// Composable, safe — args always parameterised
q, args := From("products").
    Select("id", "name", "price").
    Where("category = ?", category).
    Where("price < ?", maxPrice).
    OrderBy("price ASC").
    Limit(20).
    Offset(page * 20).
    Build()

rows, err := db.QueryContext(ctx, q, args...)
```

## Repository interface

```go
type ProductRepository interface {
    Create(ctx context.Context, p Product) (*Product, error)
    GetByID(ctx context.Context, id int64) (*Product, error)
    Search(ctx context.Context, f SearchFilter) ([]*Product, error)
    Update(ctx context.Context, p Product) error
    Delete(ctx context.Context, id int64) (bool, error)
}
```

## Caching layer pattern

```go
// Wrap any repo with caching — service code doesn't change
type cachingRepo struct {
    inner ProductRepository
    cache map[int64]*Product
}

func (c *cachingRepo) GetByID(ctx context.Context, id int64) (*Product, error) {
    if p, ok := c.cache[id]; ok { return p, nil }  // cache hit
    p, err := c.inner.GetByID(ctx, id)
    if err == nil { c.cache[id] = p }               // populate
    return p, err
}
```

## Dynamic filter SQL pattern

```go
q := "SELECT ... FROM products WHERE 1=1"
var args []any
if f.Category != nil {
    q += " AND category = ?"
    args = append(args, *f.Category)
}
if f.MinPrice != nil {
    q += " AND price_cent >= ?"
    args = append(args, *f.MinPrice)
}
rows, _ := db.QueryContext(ctx, q, args...)
```

The `WHERE 1=1` trick allows unconditional AND clauses — clean and safe.

## Production recommendations

- Use raw SQL for simple, stable queries; reach for a builder when filters are dynamic
- Keep the repository interface narrow — don't leak SQL concepts (table names, column names) through it
- In-memory implementation enables fast unit tests with zero infrastructure
- Caching layer: always invalidate on write (`Update`, `Delete`, `AdjustStock`)
- For large codebases: consider `sqlc` (generates type-safe Go from SQL) as a middle ground
