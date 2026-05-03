# Chapter 69 Checkpoint — ORM vs Builder vs Raw SQL

## Self-assessment questions

1. What are the three common approaches to database access in Go, and when would you use each?
2. How does a query builder prevent SQL injection while still allowing dynamic queries?
3. What is the Repository pattern, and what problem does it solve?
4. How do you write a dynamic WHERE clause with optional filters safely in Go?
5. What is the main benefit of having an in-memory repository implementation?
6. How does a caching repository work without the service layer knowing about the cache?

## Checklist

- [ ] Can write raw SQL with parameterised queries using `?` placeholders
- [ ] Can build a type-safe query builder with composable `Where/OrderBy/Limit/Offset`
- [ ] Can define a `Repository` interface that abstracts data access
- [ ] Can implement the same interface in both in-memory and SQL forms
- [ ] Can write dynamic SQL with optional filters using the `WHERE 1=1` pattern
- [ ] Can wrap a repository with a transparent caching layer
- [ ] Know the tradeoffs of raw SQL, query builder, and ORM
- [ ] Know when to invalidate cache entries (on write/update/delete)

## Answers

1. Raw SQL for simple, performance-critical, or complex queries; query builder for dynamic filter APIs where conditions vary by request; ORM for rapid CRUD development where productivity matters more than control. Most large applications use a mix: ORM for simple cases, raw SQL for complex queries.

2. A query builder always passes values as separate `args` (parameterised), never interpolating them into the SQL string. Even when the WHERE conditions vary, the values remain placeholders (`?`) — the database driver handles escaping, making injection impossible.

3. The Repository pattern defines a data-access interface in terms of your domain objects. It separates business logic from database concerns. The service code only knows about the interface — it doesn't know if data comes from SQLite, PostgreSQL, or an in-memory map.

4. Use the `WHERE 1=1` base and append `AND column = ?` for each non-nil filter. Collect values in a slice. Example: `if filter.Category != nil { q += " AND category = ?"; args = append(args, *filter.Category) }`.

5. The in-memory implementation enables unit tests for the service layer without a running database. Tests run in microseconds and don't need teardown. It also serves as a simple dev environment where persistence doesn't matter.

6. The caching repository implements the same interface as the underlying repo and is passed to the service in its place. The service calls `repo.GetByID(ctx, id)` — it has no idea whether the result came from cache or the database. This is the Decorator pattern applied to a Go interface.
