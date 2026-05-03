# Chapter 62 Exercises — Input Validation

## Exercise 1 — Validated API (`exercises/01_validated_api`)

Build a products CRUD API with comprehensive validation on all inputs.

### Product schema

| Field | Type | Rules |
|---|---|---|
| `name` | string | required, min 2, max 100 |
| `description` | string | optional, max 1000 |
| `price` | float64 | required, must be > 0 |
| `category` | string | required, enum: `electronics`, `clothing`, `books`, `home`, `sports` |
| `stock` | int | required, must be ≥ 0 |
| `sku` | string | required, pattern `[A-Z]{2}-[0-9]{4}` |

### Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/products` | List all products (supports `?category=` filter) |
| `POST` | `/products` | Create product — 201 or 422 with field errors |
| `GET` | `/products/{id}` | Get by ID — 200 or 404 |
| `PUT` | `/products/{id}` | Update (full replacement) — 200 or 422 |
| `DELETE` | `/products/{id}` | Delete — 204 or 404 |

### Query parameter validation

For `GET /products`, validate optional query params:
- `?category=X` — must be one of the valid categories if provided
- `?min_price=X` — must be a valid positive number if provided
- `?max_price=X` — must be a valid positive number and ≥ `min_price` if both provided

### Error response format

```json
{
  "error": "validation failed",
  "fields": [
    {"field": "price", "rule": "positive", "message": "price must be greater than zero"},
    {"field": "sku", "rule": "pattern", "message": "SKU must match [A-Z]{2}-[0-9]{4}"},
    {"field": "category", "rule": "enum", "message": "invalid category"}
  ]
}
```

### Expected behaviour

| Scenario | Status |
|---|---|
| Valid product creation | 201 |
| Missing name | 422 with field error |
| Price ≤ 0 | 422 with field error |
| Invalid SKU format | 422 with field error |
| Invalid category | 422 with field error |
| Stock < 0 | 422 with field error |
| Multiple errors at once | 422 with all fields listed |
| Get non-existent product | 404 |
| Valid update | 200 |
| Invalid query param | 422 with field error |

### Hints

- Compile the SKU regex once at package level: `var reSKU = regexp.MustCompile(`^[A-Z]{2}-[0-9]{4}$`)`
- Write a `validateProduct(p Product) ValidationErrors` function used by both create and update
- Write a `validateListQuery(q url.Values) ValidationErrors` for query param validation
- Use `strconv.ParseFloat` for price query params; treat parse failure as a validation error
