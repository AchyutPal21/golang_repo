# Chapter 34 — Repository Pattern

> **Part III · Designing Software** | Estimated reading time: 22 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

The Repository pattern is the persistence port from Chapter 30, but examined in depth. It decouples business logic from storage mechanics — the service layer never knows whether data lives in Postgres, Redis, a file, or an in-memory map. This chapter covers the interface contract, multiple implementations, the Specification pattern for composable queries, and pagination.

---

## 34.1 — The repository interface

A repository interface lives in the domain or application layer. Its methods:

- Accept and return **domain types** (not `*sql.Rows`, not JSON bytes)
- Use **domain sentinel errors** (`ErrUserNotFound`, not driver errors)
- Hide all storage details behind clean method names

```go
type UserRepository interface {
    Save(u User) (User, error)      // insert (ID==0) or update
    FindByID(id UserID) (User, error)
    FindByEmail(email string) (User, error)
    FindAll() ([]User, error)
    FindActive() ([]User, error)
    Delete(id UserID) error
    Count() (int, error)
}
```

`Save` uses upsert semantics: zero ID = insert (assigns a new ID), non-zero ID = update.

---

## 34.2 — Multiple implementations

The same interface can have radically different backends:

| Implementation | When to use |
|---|---|
| `memUserRepo` | Unit tests; development; caching layer |
| `postgresUserRepo` | Production with relational data |
| `auditUserRepo` | Append-only audit log; keeps all versions |
| `readThroughCache` | Wraps another repo; caches reads |

Swapping implementations requires zero changes to the service layer.

---

## 34.3 — Specification pattern

Instead of adding `FindActiveInCategory(cat string)` for every query combination, define composable predicates:

```go
type Spec interface {
    IsSatisfiedBy(p Product) bool
}

spec := And(And(Active(), InCategory("widgets")), InStock())
results, _ := repo.Query(spec, SortByPrice, Page{Number: 1, Size: 10})
```

`And`, `Or`, `Not` compose leaf specs into arbitrary boolean expressions. The repository evaluates the spec against each record — in SQL this becomes a `WHERE` clause builder.

---

## 34.4 — Pagination

Repositories return `PagedResult` rather than unbounded slices:

```go
type Page struct{ Number, Size int }

type PagedResult struct {
    Items      []Product
    TotalCount int
    Page       Page
}
```

`TotalPages()` is computed from `TotalCount / Page.Size`. The service layer passes `Page{Number: 1, Size: 20}` and iterates until `Page.Number >= TotalPages()`.

---

## 34.5 — Repository rules

1. **No business logic** — a repository stores and retrieves; it does not validate domain rules.
2. **Domain errors only** — wrap driver errors into domain sentinels before returning.
3. **No leaking** — never return `*sql.Tx`, `*mongo.Cursor`, or similar from a repository method.
4. **Narrow interfaces** — if a consumer only reads, give it a `Finder` interface, not the full `Repository`.

---

## Running the examples

```bash
cd book/part3_designing_software/chapter34_repository_pattern

go run ./examples/01_repository_basics  # UserRepository: in-memory + audit implementations
go run ./examples/02_query_spec         # Specification pattern + pagination on ProductRepository

go run ./exercises/01_multi_repo        # OrderService wired with CustomerRepo + OrderRepo
```

---

## Key takeaways

1. **Repository interface** in domain/application layer — no storage details.
2. **Upsert semantics** — `Save` handles insert (zero ID) and update (non-zero ID).
3. **Domain sentinel errors** — wrap driver errors; use `errors.Is` in service code.
4. **Specification pattern** — composable `And`/`Or`/`Not` predicates replace N query methods.
5. **Pagination** — `PagedResult` with `TotalCount` prevents unbounded memory use.

---

## Cross-references

- **Chapter 30** — Clean Architecture: repository is the persistence secondary port
- **Chapter 28** — Dependency Injection: repository injected into service via constructor
- **Chapter 35** — Service Layer: uses repositories as its primary data access mechanism
