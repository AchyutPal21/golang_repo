# Chapter 76 Exercises — gRPC

## Exercise 1 — Product Service (`exercises/01_product_service`)

Build a complete product service with CRUD operations, streaming search, real-time stock watch, and a tracing interceptor chain.

### Domain types

```go
type Category string   // "books", "electronics", "clothing"

type Product struct {
    ID         string
    Name       string
    Category   Category
    Price      int   // cents
    StockCount int
    Tags       []string
    CreatedAt  time.Time
}
```

### Service interface

```go
type ProductService interface {
    CreateProduct(ctx, *Product) (*Product, error)
    GetProduct(ctx, id string) (*Product, error)
    UpdateProduct(ctx, *Product) (*Product, error)
    DeleteProduct(ctx, id string) error
    SearchProducts(ctx, *SearchRequest, send func(*Product) error) error
    WatchStock(ctx, productID string, send func(int) error) error
}
```

### SearchRequest

```go
type SearchRequest struct {
    Query    string    // substring match on Name
    Category Category  // empty = all categories
    MinPrice int       // 0 = no lower bound
    MaxPrice int       // 0 = no upper bound
}
```

### Validation rules

- `CreateProduct`: name required; price >= 0
- `GetProduct` / `DeleteProduct`: return `CodeNotFound` if ID missing
- `UpdateProduct`: return `CodeNotFound` if ID missing

### WatchStock

`WatchStock` blocks until ctx is done, sending stock count to `send` whenever `UpdateProduct` is called for the watched product. Use a per-product channel registered in the server's watcher map; deregister on return.

### Interceptors

Implement two interceptors:

**`TracingInterceptor`**: assigns an auto-incrementing trace ID, stores it in context, logs method name and duration after each call.

**`AuthInterceptor`**: reads a token from context; returns `CodePermDenied` if missing.

Chain them: tracing wraps auth wraps the handler.

### Client wrapper

Implement a `Client` that:
- Holds the base context with an auth token
- Wraps each call through the interceptor chain
- Exposes typed call methods (or a generic `call` helper)

### Demonstration

1. **Seed** 4 products across 3 categories
2. **Create** a new product; verify assigned ID
3. **Get** `p-1`; verify name and price
4. **Update** `p-1` stock; verify new value
5. **Delete** `p-3`; verify subsequent Get returns `CodeNotFound`
6. **SearchProducts**: stream all books; verify count
7. **SearchProducts**: price range 3000–6000 cents; verify results
8. **WatchStock**: start watching `p-1` in a goroutine; call `UpdateProduct` twice; verify at least 1 event received
9. **Error cases**: empty name → `CodeInvalidArg`; missing ID → `CodeNotFound`

### Hints

- The `Chain` function should wrap interceptors right-to-left so the first interceptor is outermost
- `WatchStock` must deregister the channel on return to avoid goroutine leaks
- Use `sync.RWMutex` for the products map; watcher registration needs the write lock
- The watcher channel should be buffered (8) to avoid blocking `UpdateProduct`
- Use `atomic.Int64` for the trace counter — interceptors may be called concurrently
