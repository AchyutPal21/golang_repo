# Chapter 31 — Creational Patterns

> **Part III · Designing Software** | Estimated reading time: 22 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Creational patterns control how objects are constructed. Go does not have class hierarchies or `new` keywords, so the patterns look different from their Java/C++ forms — but the problems they solve are identical: hiding complex construction, enforcing invariants, reusing expensive objects, and creating families of related objects.

---

## 31.1 — Factory Method

A function (or method on a factory type) that returns an interface. The caller receives the interface and never touches the concrete type.

```go
func NewLogger(format string) Logger {
    switch format {
    case "json":  return &jsonLogger{}
    case "noop":  return noopLogger{}
    default:      return &consoleLogger{prefix: "APP"}
    }
}
```

**When to use**: when the concrete type should be an implementation detail; when you want to swap implementations at runtime (test vs production, different providers).

---

## 31.2 — Builder

Constructs a complex object step by step with method chaining. `Build()` validates the accumulated state and returns the product (or an error).

```go
req, err := NewRequest("POST", "https://api.example.com/orders").
    WithBearerToken("tok_abc123").
    WithJSON(`{"sku":"WIDGET","qty":3}`).
    WithTimeout(10 * time.Second).
    Build()
```

Key design: store the first error encountered; skip all subsequent operations; surface it only in `Build()`. This allows fluent chaining without `if err != nil` on every line.

**When to use**: objects with many optional fields, some mutually dependent; when validation must happen over the whole configuration rather than field by field.

---

## 31.3 — Abstract Factory

An interface whose methods are factory methods — creates families of related objects:

```go
type UIFactory interface {
    NewButton(label string) Button
    NewTextInput(placeholder string) TextInput
}
```

`LightUIFactory` and `DarkUIFactory` each produce an entire family of themed widgets. The rendering code depends only on `UIFactory` — swapping the factory swaps the entire theme.

**When to use**: when the system must work with multiple families of related products and must be independent of how those families are created.

---

## 31.4 — Singleton

A package-level variable initialised exactly once with `sync.Once`. The accessor function returns the shared instance:

```go
var once sync.Once
var cfg *AppConfig

func GetConfig() *AppConfig {
    once.Do(func() { cfg = loadConfig() })
    return cfg
}
```

**Caution**: singletons make testing harder and hide dependencies. Prefer constructor injection. Use singletons only for genuinely global, stateless objects (loggers, config, connection pools managed by the pool itself).

---

## 31.5 — Prototype

Creates new objects by cloning an existing one. Implement `Clone()` returning a deep copy:

```go
func (d *DocumentTemplate) Clone() *DocumentTemplate {
    sections := make([]string, len(d.Sections))
    copy(sections, d.Sections)
    // deep copy map...
    return &DocumentTemplate{..., Sections: sections, Metadata: metadata}
}
```

**When to use**: when construction is expensive and many variants start from a configured base; when you need many independent copies with slight variations.

---

## 31.6 — Object Pool

Reuse expensive-to-create objects. `Get()` returns a ready object; `Put()` returns it to the pool:

```go
pool := &ProcessorPool{}
proc := pool.Get()     // allocates only if pool is empty
proc.Process(data)
pool.Put(proc)         // reset and return for reuse
```

The standard library's `sync.Pool` is the canonical Go implementation for short-lived allocations.

---

## Running the examples

```bash
cd book/part3_designing_software/chapter31_creational_patterns

go run ./examples/01_factory_builder      # Factory Method, Builder, Abstract Factory
go run ./examples/02_singleton_prototype  # Singleton, Prototype, Object Pool

go run ./exercises/01_query_builder       # SQL SELECT builder with validation
```

---

## Key takeaways

1. **Factory Method** — return interfaces, hide concrete types. Swap implementations without changing callers.
2. **Builder** — accumulate state, validate in `Build()`. Store the first error; chain without `if err` on every call.
3. **Abstract Factory** — factory interface that produces families of related objects.
4. **Singleton** — `sync.Once` + package-level variable. Prefer DI; use singletons sparingly.
5. **Prototype** — `Clone()` with deep copy. Useful when many variants start from a configured base.
6. **Object Pool** — `Get()`/`Put()` to reuse expensive objects. Use `sync.Pool` for short-lived allocations.

---

## Cross-references

- **Chapter 28** — Functional Options: an alternative to Builder for optional configuration
- **Chapter 14** — Closures: factory functions use closures to capture configuration
- **Chapter 32** — Structural Patterns: Adapter and Decorator build on the interface foundation created here
