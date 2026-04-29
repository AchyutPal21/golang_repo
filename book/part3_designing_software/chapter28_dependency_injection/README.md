# Chapter 28 — Dependency Injection

> **Part III · Designing Software** | Estimated reading time: 20 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Dependency injection (DI) is the practice of giving a component its collaborators from the outside rather than letting it create them internally. In Go, this is straightforward: pass dependencies as constructor parameters. The result is code that is testable, swappable, and readable without any framework. This chapter shows both the constructor injection pattern and the functional options pattern for more complex configuration needs.

---

## 28.1 — Why DI matters

Without DI:
```go
func NewUserService() *UserService {
    return &UserService{
        db:     connectToPostgres(), // hard-wired production dependency
        mailer: newSMTPMailer(),     // impossible to replace in tests
    }
}
```

With DI:
```go
func NewUserService(store UserStore, mailer Mailer, clock Clock) *UserService {
    return &UserService{store: store, mailer: mailer, clock: clock}
}
```

The caller controls what gets injected. Tests inject fakes. Production injects real implementations.

---

## 28.2 — Constructor injection

The canonical Go DI pattern: every dependency is an explicit constructor parameter.

Rules:
1. **Accept interfaces** — the dependency's type should be an interface, not a concrete type
2. **Make all dependencies explicit** — hidden global state is not DI
3. **Validate at construction** — fail fast, not at first use
4. **Inject the clock** — `time.Now()` is a hidden dependency; inject a `Clock` interface for deterministic tests

---

## 28.3 — The wiring layer

All DI wiring happens in `main` (or a top-level constructor/initialiser):

```go
func main() {
    db := postgres.New(os.Getenv("DATABASE_URL"))
    mailer := smtp.New(os.Getenv("SMTP_HOST"))
    clock := realclock.New()
    svc := user.NewService(db, mailer, clock)
    // ...
}
```

This is the **composition root** — the single place in the program where all concrete types are assembled. Everything else works with interfaces.

---

## 28.4 — Test fakes

With constructor injection, test fakes are trivial:

```go
fakeStore := newFakeUserStore()
fakeMailer := &fakeMailer{}
fixedClock := fixedClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

svc := NewUserService(fakeStore, fakeMailer, fixedClock)
// Test with full control of all external dependencies.
```

No mocking framework needed. No `reflect` magic. Just structs that implement interfaces.

---

## 28.5 — Functional options

When a type has many optional configuration parameters, constructor injection gets unwieldy. The functional options pattern solves this:

```go
type Option func(*Server) error

func WithPort(port int) Option {
    return func(s *Server) error {
        s.port = port
        return nil
    }
}

s, err := NewServer(
    WithHost("api.example.com"),
    WithPort(443),
    WithTLS(),
    WithTimeout(10 * time.Second),
)
```

Benefits:
- **Backward compatible** — adding a new option never breaks existing callers
- **Self-documenting** — option names read like named parameters
- **Validating** — each option returns an error; `NewServer` collects them
- **Composable** — options are values; you can predefine option slices

---

## 28.6 — When to use functional options vs a config struct

| Functional options | Config struct |
|---|---|
| Public API that evolves over time | Internal use where all callers are known |
| Options have validation logic | Simple value assignment |
| Some options are mutually exclusive | All options are independent |
| Options need to run side effects | Pure data |

Both are valid. Functional options are more common in library APIs; config structs are fine for internal code.

---

## 28.7 — DI frameworks in Go

Go has DI frameworks (`google/wire`, `uber-go/fx`, `uber-go/dig`). They are useful when the object graph is large (50+ types). For most applications, manual wiring in `main` is sufficient and much simpler to read and debug.

---

## Running the examples

```bash
cd book/part3_designing_software/chapter28_dependency_injection

go run ./examples/01_manual_di          # constructor injection, fakes, fixed clock
go run ./examples/02_functional_options # functional options with validation + presets

go run ./exercises/01_service_wiring    # OrderService wired from three dependencies
```

---

## Key takeaways

1. **Constructor injection** is Go's DI — pass interfaces as constructor params.
2. **All wiring happens in `main`** (the composition root). Services never create their own dependencies.
3. **Inject the clock** — `time.Now()` is a hidden dependency that breaks deterministic tests.
4. **Test fakes** are plain structs implementing interfaces — no framework needed.
5. **Functional options** handle many optional parameters ergonomically, with validation and backward compatibility.

---

## Cross-references

- **Chapter 27** — Interface-Driven Design: consumer-side interfaces enable DI
- **Chapter 14** — Closures: the closure inside each functional option
- **Chapter 36** — Error Handling Philosophy: wrapping errors in DI constructors
