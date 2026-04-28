# Chapter 20 — Structs and Composite Literals

> **Part II · Core Language** | Estimated reading time: 20 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Structs are Go's only mechanism for grouping related data. They underlie everything: HTTP requests, database rows, config objects, domain entities. Knowing how to lay them out efficiently, tag them for marshalling, embed them for composition, and understand their value vs pointer semantics is fundamental to idiomatic Go.

---

## 20.1 — Declaration and composite literals

```go
type Point struct {
    X, Y float64
}

p := Point{1.0, 2.0}         // positional — avoid for > 2 fields
p := Point{X: 1.0, Y: 2.0}  // keyed — preferred
var p Point                   // zero value: {0.0 0.0}
```

Always use keyed literals in production code. Positional literals break silently when fields are added or reordered.

---

## 20.2 — Structs are comparable

Structs are comparable with `==` if all their fields are comparable. This makes them usable as map keys.

---

## 20.3 — Field tags

Field tags are string metadata attached to fields, read at runtime via reflection:

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email,omitempty"`
    Pass  string `json:"-"`          // always omit
    Age   int    `json:"age" db:"age_years"`
}
```

Tags are used by: `encoding/json`, `encoding/xml`, `database/sql` drivers, `gopkg.in/validator.v2`, and many others. The tag format is `key:"value"` pairs separated by spaces.

---

## 20.4 — Anonymous structs

Anonymous structs are useful for one-off data shapes — JSON test fixtures, temporary groupings, table-driven test cases:

```go
cases := []struct {
    input    string
    expected int
}{
    {"hello", 5},
    {"hi", 2},
}
```

---

## 20.5 — Embedding

Embedding a type `T` inside a struct promotes `T`'s fields and methods to the outer type:

```go
type Animal struct{ Name string }
func (a Animal) Describe() string { return a.Name }

type Dog struct {
    Animal          // embedded
    Breed  string
}

d := Dog{Animal: Animal{Name: "Rex"}, Breed: "Lab"}
d.Name     // promoted from Animal
d.Describe() // promoted from Animal
```

Embedding is not inheritance. The outer type does not *become* the inner type; it *has* the inner type and exposes its interface. The distinction matters: `Dog` is not assignable to `Animal`, but it satisfies interfaces that `Animal` satisfies.

---

## 20.6 — Method shadowing

If the outer type defines a method with the same name as an embedded type's method, the outer method shadows the inner one:

```go
func (d Dog) Describe() string { return "Dog: " + d.Animal.Describe() }
```

The inner method is still accessible via the explicit field name: `d.Animal.Describe()`.

---

## 20.7 — Pointer embedding

Embedding `*T` instead of `T` allows the outer struct to share or swap the inner value, and avoids copying large inner structs:

```go
type Server struct {
    *Logger
    addr string
}
```

The `Server` zero value has a nil `*Logger` — you must initialise it before calling any promoted Logger methods.

---

## 20.8 — Memory layout and padding

The compiler inserts padding between fields to satisfy alignment requirements. Field order matters for memory efficiency:

```go
// 24 bytes (7 bytes wasted padding)
type Bad struct { A bool; B int64; C bool; D int32 }

// 16 bytes (2 bytes wasted padding)
type Good struct { B int64; D int32; A bool; C bool }
```

Rule of thumb: order fields largest-to-smallest by size. Use `unsafe.Sizeof` to verify, and `go vet -fieldalignment` (or `fieldalignment` from `golang.org/x/tools`) to catch inefficient layouts automatically.

---

## Running the examples

```bash
cd book/part2_core_language/chapter20_structs

go run ./examples/01_struct_basics  # literals, tags, JSON, comparison, layout
go run ./examples/02_embedding      # promotion, shadowing, pointer embed

go run ./exercises/01_bank_account  # BankAccount with embedded AuditLog
```

---

## Exercises

### [exercises.md](exercises.md)

---

## Revision checkpoint

### [checkpoint.md](checkpoint.md)

---

## Key takeaways

1. Use **keyed composite literals** — positional literals break on field reorder.
2. **Field tags** are metadata for marshalling, validation, ORM, and more.
3. **Embedding** promotes fields and methods; it is composition, not inheritance.
4. Method **shadowing** works top-down; access inner methods via the explicit field name.
5. **Pointer embedding** shares the inner value and avoids copying.
6. Order struct fields **largest-to-smallest** to minimise padding.

---

## Cross-references

- **Chapter 16** — Pointers: pointer-to-struct, auto-dereference
- **Chapter 21** — Methods: pointer vs value receivers on structs
- **Chapter 22** — Interfaces: embedding satisfies interfaces through promoted methods
- **Chapter 23** — Embedding and Composition: full deep-dive
