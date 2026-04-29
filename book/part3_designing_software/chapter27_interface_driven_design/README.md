# Chapter 27 — Interface-Driven Design

> **Part III · Designing Software** | Estimated reading time: 20 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Interfaces are Go's primary tool for decoupling. The way you define them determines how testable, extensible, and readable your code is. The single most important rule — **define interfaces where they are used, not where they are implemented** — is not obvious and is routinely violated by engineers coming from Java or C#. This chapter makes that rule concrete and shows why it matters.

---

## 27.1 — Producer-side vs consumer-side interfaces

**Producer-side** (Java/C# style): the library defines a big interface and tells consumers to implement it.

```go
// db package
type DatabaseInterface interface {
    GetUser(id int) (User, error)
    SaveUser(u User) error
    DeleteUser(id int) error
    ListUsers() ([]User, error)
    GetProduct(id int) (Product, error)
    // ... 20 more methods
}
```

Problems:
- Every consumer imports and depends on the full interface
- Adding a method to the interface breaks all implementors
- Test fakes must implement all 20+ methods even if the test needs one

**Consumer-side** (Go idiom): each consumer defines *only* the interface it needs.

```go
// userservice package
type UserReader interface {
    GetUser(id int) (User, error)
}

// reportservice package
type UserLister interface {
    ListUsers() ([]User, error)
}
```

The concrete `realDB` satisfies both without knowing either interface exists. Each consumer is decoupled from every other consumer.

---

## 27.2 — The import graph advantage

Consumer-side interfaces reverse the import direction:

```
Producer-side:  userservice → db (must import db to use DatabaseInterface)
Consumer-side:  db → nothing (db knows nothing about userservice)
                userservice defines UserReader locally
```

This is how the standard library is structured. `io.Reader` is defined in `io` — but any package can accept it without importing `os`, `net`, or `bytes`. The concrete types travel with their packages; the interface stays with the consumer.

---

## 27.3 — Narrow interfaces

The most useful interfaces have one or two methods. From the standard library:

| Interface | Methods | Satisfied by |
|---|---|---|
| `io.Reader` | 1 | 50+ types |
| `io.Writer` | 1 | 30+ types |
| `fmt.Stringer` | 1 | Any type with `String()` |
| `error` | 1 | Any type with `Error()` |
| `sort.Interface` | 3 | Any sortable collection |

A narrower interface is easier to satisfy, easier to fake in tests, and more likely to be useful across unrelated types.

---

## 27.4 — Interface composition

Build larger interfaces by embedding smaller ones:

```go
type ReadCache interface { Getter }
type WriteCache interface { Setter }
type Cache interface { Getter; Setter; Deleter }
```

Consumers that only read receive `ReadCache`. Cache warmers receive `WriteCache`. The cache manager receives `Cache`. The same `memCache` satisfies all three.

---

## 27.5 — The role model

Every interface you design should answer: *what role does the caller need the value to play?*

- A handler needs something to respond to requests → `http.Handler`
- A logger needs something to write lines → `io.Writer` (often)
- A service needs something to persist data → a narrow storage interface

If you can name the role in one word (`Reader`, `Sender`, `Validator`, `Renderer`), the interface is probably well-sized.

---

## 27.6 — When to define an interface

Define an interface when:
1. You have two or more implementations (real + fake for testing)
2. The implementation will change (swap DB, swap email provider)
3. You want to accept values from unrelated packages

Do **not** define an interface when:
- There is only one implementation and no test fake needed
- You own both sides — just use the concrete type
- You are wrapping a third-party type you don't control (use thin wrapper functions instead)

---

## Running the examples

```bash
cd book/part3_designing_software/chapter27_interface_driven_design

go run ./examples/01_consumer_defined  # UserService, ReportService, AuditLog on same realDB
go run ./examples/02_narrow_interfaces # Cache split into Getter/Setter/Deleter + io.Writer pipeline

go run ./exercises/01_storage_abstraction # TodoService with MemoryStore via consumer interfaces
```

---

## Key takeaways

1. **Define interfaces where they are used** — in the consumer package, not the producer.
2. **Narrow interfaces** (1-2 methods) are more reusable and easier to satisfy.
3. **Compose interfaces** from smaller ones rather than building monolithic contracts.
4. Consumer-side interfaces **reverse the import graph** — the concrete type has no dependency on its consumers.
5. Only define an interface when you have (or expect) **multiple implementations**.

---

## Cross-references

- **Chapter 22** — Interfaces: mechanics and the typed-nil trap
- **Chapter 26** — OOP in Go: abstract class → interface mental model
- **Chapter 28** — Dependency Injection: wiring consumer-defined interfaces at startup
