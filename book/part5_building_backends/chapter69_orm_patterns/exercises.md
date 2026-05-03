# Chapter 69 Exercises — ORM vs Builder vs Raw SQL

## Exercise 1 — Product Repository (`exercises/01_product_repository`)

Implement a complete product catalog system using the repository pattern with three implementations.

### Domain

```go
type Product struct {
    ID, Name, Description, Category string
    PriceCent, Stock int
    Active bool
    CreatedAt time.Time
}

type SearchFilter struct {
    Category *string
    MinPrice, MaxPrice *int
    Active *bool
    Query string  // LIKE on name/description
}
```

### Repository interface

```go
type ProductRepository interface {
    Create(ctx, Product) (*Product, error)
    GetByID(ctx, id) (*Product, error)
    Search(ctx, SearchFilter) ([]*Product, error)
    Update(ctx, Product) error
    Delete(ctx, id) (bool, error)
    AdjustStock(ctx, StockAdjustment) (int, error)  // returns new stock
    CountByCategory(ctx) (map[string]int, error)
}
```

### Three implementations

**1. In-memory** (`memRepo`)
- Protected with `sync.RWMutex`
- `Search` applies all filter fields
- `AdjustStock` returns `ErrInsufficientStock` if result < 0

**2. SQL** (`sqlRepo`)
- Dynamic WHERE with `WHERE 1=1` pattern
- `active` column stored as INTEGER (0/1)
- `AdjustStock`: read current, check new >= 0, update in one query

**3. Caching wrapper** (`cachingRepo`)
- Wraps any `ProductRepository`
- Caches by ID — hit on `GetByID` if ID is in cache
- `Create` pre-populates cache
- `Update`, `Delete`, `AdjustStock` invalidate the cached entry
- Expose `CacheStats() (hits, misses int)` for inspection

### Key requirements

- Both in-memory and SQL implementations must produce identical behaviour
- `ErrNotFound` and `ErrInsufficientStock` must be sentinel errors checkable with `errors.Is`
- Caching repo must be transparent — service code never calls cache-specific methods

### Hints

- `WHERE 1=1` lets you unconditionally append `AND ...` for each optional filter
- SQLite stores booleans as integers — scan into `var active int`, convert to `bool`
- Use pointer fields in `SearchFilter` (`*string`, `*int`) so callers can omit filters by passing `nil`
- Use `strings.Contains(strings.ToLower(name), strings.ToLower(query))` for case-insensitive in-memory search
