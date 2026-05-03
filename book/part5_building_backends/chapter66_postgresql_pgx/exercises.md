# Chapter 66 Exercises — PostgreSQL with pgx

## Exercise 1 — Product Store (`exercises/01_product_store`)

Build a product inventory store using pgx v5 with proper NULL handling, batch operations, and audit history.

### Schema

```sql
products (id BIGSERIAL PK, name TEXT, description TEXT?, price NUMERIC(10,2),
          category TEXT, stock INT, created_at TIMESTAMPTZ)

price_history (id BIGSERIAL PK, product_id BIGINT FK→products(id),
               old_price NUMERIC(10,2), new_price NUMERIC(10,2), changed_at TIMESTAMPTZ)
```

### Operations

| Method | Description |
|---|---|
| `CreateProduct(ctx, name, desc, category, price, stock)` | INSERT with RETURNING |
| `GetProduct(ctx, id)` | SELECT by PK; return typed error if not found |
| `ListByCategory(ctx, category)` | SELECT WHERE category = $1 ORDER BY name |
| `SearchProducts(ctx, query)` | ILIKE on name OR description |
| `LowStockReport(ctx, threshold)` | SELECT WHERE stock < $1 ORDER BY stock ASC |
| `UpdatePrice(ctx, productID, newPrice)` | Transaction: read old price, update, insert history |
| `BulkUpdateStock(ctx, updates map[int64]int)` | pgx.Batch for all updates in one round trip |
| `DeleteProduct(ctx, id)` | DELETE; return bool whether row was deleted |
| `PriceHistory(ctx, productID)` | SELECT price_history WHERE product_id = $1 |

### Key requirements

- Use `pgtype.Text` for the nullable `description` column
- `UpdatePrice` must be a transaction: read with `SELECT FOR UPDATE`, update, insert history — all in one atomic operation
- `BulkUpdateStock` must use `pgx.Batch` — not individual `Exec` calls
- Gracefully skip live queries if `DATABASE_URL` is not set or Postgres is unreachable — print a usage hint instead

### Error handling

- Use `errors.As(err, &pgErr)` to check `pgconn.PgError.Code` for `"23505"` (duplicate key)
- Return `fmt.Errorf("product %d: not found", id)` wrapping `pgx.ErrNoRows`

### Running with a real database

```bash
# Start Postgres
docker run -d --name pg -e POSTGRES_PASSWORD=secret -p 5432:5432 postgres:16

# Run the exercise
DATABASE_URL=postgres://postgres:secret@localhost:5432/postgres?sslmode=disable \
  go run ./exercises/01_product_store
```

### Hints

- `INSERT ... RETURNING` fetches the inserted row in the same round trip — no need for a follow-up `SELECT`
- `pgtype.Text{String: desc, Valid: desc != ""}` constructs a nullable string
- `pool.Acquire(ctx)` borrows a `*pgxpool.Conn`; call `.Release()` when done to return it
- Batch results must be read in the same order they were queued
