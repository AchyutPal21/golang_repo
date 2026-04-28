# Chapter 21 — Exercises

## 21.1 — Geometry shapes

Run [`exercises/01_geometry`](exercises/01_geometry/main.go).

All three shapes satisfy `Shape` using value receivers, so both `Shape` and
`*Shape` work as interface values.

Try:
- Add a `Scale(factor float64)` method with a pointer receiver to each shape.
  Does the `Shape` interface still work with value receivers? Why not?
- Add a `Largest(shapes []Shape) Shape` function.
- Add a `Sort(shapes []Shape)` that sorts by area ascending using `sort.Slice`.

## 21.2 ★ — Method chaining (builder pattern)

Implement a `QueryBuilder` using method chaining:

```go
query := NewQuery("users").
    Select("id", "name").
    Where("age > ?", 18).
    OrderBy("name").
    Limit(10).
    Build()
```

Each method must return `*QueryBuilder` to enable chaining.
`Build()` returns the final SQL string.

## 21.3 ★ — Functional options with methods

Combine the functional options pattern (Chapter 14) with methods:
define a `Server` type whose constructor accepts `...Option`, and
add `WithLogger`, `WithTimeout`, `WithMaxConns` option functions.
Implement a `String()` method on `Server` that describes the configuration.
