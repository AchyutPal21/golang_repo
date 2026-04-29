# Chapter 26 — OOP in Go

> **Part III · Designing Software** | Estimated reading time: 22 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Most engineers arrive at Go carrying mental models from Java, Python, C++, or C#. Those models include classes, inheritance hierarchies, virtual methods, and abstract base classes. Go has none of those — but it covers the same design goals more cleanly. This chapter maps the concepts you already know onto the Go constructs that replace them, so you stop fighting the language and start using it fluently.

---

## 26.1 — The mental model shift

| OOP concept | Go equivalent |
|---|---|
| Class | Struct + methods |
| Abstract class | Interface |
| Inheritance | Embedding + interface satisfaction |
| Virtual method | Interface method dispatch |
| Constructor | `NewXxx` function |
| Private field | Unexported field (lowercase) |
| `toString()` | `fmt.Stringer` (`String() string`) |
| `equals()` | `==` on comparable types; `reflect.DeepEqual` for deep equality |
| Static method | Package-level function |
| Singleton | Package-level variable + `sync.Once` |

The shift is not about giving up expressiveness — it is about replacing an implicit mechanism (the compiler enforcing an inheritance chain) with an explicit one (a named interface anyone can satisfy).

---

## 26.2 — No classes: structs + methods

```go
type Dog struct {
    Name string
    age  int // unexported — enforced by the compiler, not a convention
}

func (d Dog) Speak() string { return "Woof!" }
func (d *Dog) Birthday()    { d.age++ }
```

This is exactly what a class gives you: encapsulated state + behaviour. The difference: `Dog` does not inherit from anything. Its "interface" is defined separately, and any other type can satisfy it independently.

---

## 26.3 — No abstract classes: interfaces

Java's abstract class serves two purposes: share implementation (via concrete methods) and define a contract (via abstract methods). Go separates these:

- **Contract** → interface
- **Shared implementation** → embedding or package-level functions

```go
// Contract
type Speaker interface { Speak() string }

// Shared implementation (if needed) via embedding
type BaseAnimal struct{ Name string }
func (b BaseAnimal) Describe() string { return b.Name }

type Dog struct {
    BaseAnimal
    Breed string
}
func (d Dog) Speak() string { return "Woof!" }
```

---

## 26.4 — No inheritance: composition

Go's composition model forces you to be explicit about what you're combining. There is no `extends` keyword — instead:

1. **Embed** to reuse implementation
2. **Satisfy interfaces** to achieve polymorphism

```go
// Want logging + retry + multi-channel? Compose wrappers.
sender := &LoggingSender{
    inner: NewMultiSender(
        &EmailSender{...},
        &SMSSender{...},
    ),
}
```

Each wrapper knows only about `Sender`. Adding a new channel (`WebhookSender`) requires zero changes to `LoggingSender` or `MultiSender` — just implement `Send`.

---

## 26.5 — Constructors: NewXxx functions

Go has no `new` keyword for custom types (only the built-in `new(T)` for zero-allocation). The canonical constructor pattern:

```go
func NewConfig(host string, port int) (*Config, error) {
    if host == "" {
        return nil, fmt.Errorf("host is required")
    }
    return &Config{host: host, port: port}, nil
}
```

Rules:
- Return `(*T, error)` when construction can fail — never panic in constructors
- Validate all invariants at construction time
- Return a pointer when the type has methods that mutate state
- Keep the zero value useful where possible — sometimes `NewXxx` is unnecessary

---

## 26.6 — Encapsulation: unexported fields

Go enforces encapsulation at the **package** level, not the type level. Fields and methods starting with a lowercase letter are invisible outside the package:

```go
type BankAccount struct {
    owner   string  // invisible outside package
    balance float64 // invisible outside package
}
```

Unlike Java's `private` (per-instance) or `protected` (per-hierarchy), Go's unexported fields are accessible to all code in the same package. This is intentional: tests, helpers, and sibling types in the same package share full access.

---

## 26.7 — fmt.Stringer: the toString() equivalent

Implement `String() string` to control how your type prints:

```go
func (d Dog) String() string {
    return fmt.Sprintf("Dog(%s)", d.Name)
}
fmt.Println(Dog{Name: "Rex"}) // Dog(Rex)
```

`fmt.Printf`, `fmt.Println`, `log.Println`, and string formatting all check for `fmt.Stringer` automatically. Implementing it is the first method you should add to any new type.

---

## 26.8 — Polymorphism without inheritance

The notification example in example 02 shows the full pattern:

- `Sender` is a one-method interface
- `EmailSender`, `SMSSender`, `SlackSender` are concrete types — they don't know about each other
- `MultiSender`, `LoggingSender`, `RetryingSender` are composable wrappers — they work with *any* `Sender`
- `NotificationService` depends only on `Sender` — it can be tested with a fake

This achieves everything an OOP notification hierarchy achieves, with no coupling between concrete types.

---

## 26.9 — What Go deliberately omits

| Feature | Why Go omits it |
|---|---|
| Method overloading | Ambiguous call sites; variadic and interfaces cover 95% of use cases |
| Operator overloading | Implicit behaviour; makes code harder to read |
| Generics for inheritance | Type parameters replace template hierarchies |
| `protected` visibility | Package-level visibility is sufficient |
| `final` / `sealed` | Embedding with shadowing covers most use cases |
| Covariant return types | Not needed without subtype polymorphism |

Every omission is a deliberate trade-off that keeps the language simple and readable.

---

## Running the examples

```bash
cd book/part3_designing_software/chapter26_oop_in_go

go run ./examples/01_classes_vs_go               # Java concepts → Go idioms
go run ./examples/02_composition_over_inheritance # notification system via composition

go run ./exercises/01_shape_hierarchy             # interface-based shape refactor
```

---

## Exercises

### [exercises.md](exercises.md)

---

## Revision checkpoint

### [checkpoint.md](checkpoint.md)

---

## Key takeaways

1. **Struct + methods** replaces class. **Interface** replaces abstract class.
2. **Embedding** provides shared implementation. **Interface satisfaction** provides polymorphism. They are separate.
3. **`NewXxx` functions** are constructors. Validate invariants; return `(*T, error)` on failure.
4. **Unexported fields** enforce encapsulation at the package level.
5. **Composition of wrappers** replaces inheritance hierarchies — and is more extensible.
6. Go's omissions (overloading, `protected`, `final`) are deliberate; learn to work without them.

---

## Cross-references

- **Chapter 22** — Interfaces: the full interface model
- **Chapter 23** — Embedding: promotion mechanics in depth
- **Chapter 27** — Interface-Driven Design: designing around interfaces from the start
- **Chapter 28** — Dependency Injection: `NewXxx` + interfaces for testable code
