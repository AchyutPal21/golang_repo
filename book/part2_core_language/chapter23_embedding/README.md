# Chapter 23 — Embedding and Composition

> **Part II · Core Language** | Estimated reading time: 16 min | Runnable examples: 1 | Exercises: 1

---

## Why this chapter matters

Embedding is Go's answer to inheritance. Unlike class hierarchies, embedding is additive and explicit — you compose behaviour rather than specialising a base class. Understanding how promotion, shadowing, and interface satisfaction work through embedding is essential for designing clean Go packages.

---

## 23.1 — Field and method promotion

When a struct embeds a type `T`, all exported fields and methods of `T` are *promoted* to the outer struct:

```go
type Animal struct{ Name string }
func (a Animal) Describe() string { return a.Name }

type Dog struct {
    Animal
    Breed string
}

d := Dog{Animal: Animal{Name: "Rex"}}
d.Name     // promoted field
d.Describe() // promoted method
```

The promoted identifiers are accessible directly or via the embedded field name (`d.Animal.Name`).

---

## 23.2 — Mixin pattern

Embedding enables mixins — reusable behaviours added to any struct that needs them:

```go
type Timestamps struct {
    CreatedAt time.Time
    UpdatedAt time.Time
}
func (t *Timestamps) Touch() { ... }

type User struct {
    Timestamps
    ID   int
    Name string
}
type Post struct {
    Timestamps
    ID    int
    Title string
}
```

Both `User` and `Post` gain `Touch()` and timestamp fields without code duplication.

---

## 23.3 — Diamond problem

When two embedded types both promote the same method name, the outer struct must explicitly define the method to resolve the ambiguity:

```go
type D struct{ B; C } // both promote Hello() from embedded A

func (d D) Hello() string {
    return d.B.Hello() + " " + d.C.Hello()
}
```

Ambiguous promoted fields/methods that are not overridden cause a compile error when accessed.

---

## 23.4 — Embedding satisfies interfaces

If `T` satisfies interface `I`, then any struct that embeds `T` (or `*T`) also satisfies `I` through promotion:

```go
type Buffer struct{ data string }
func (b *Buffer) Read() string   { return b.data }
func (b *Buffer) Write(s string) { b.data += s }

type Service struct{ *Buffer; name string }
// Service satisfies ReadWriter through promoted *Buffer methods
```

---

## 23.5 — Embedding vs inheritance: key differences

| Feature | Go embedding | OOP inheritance |
|---|---|---|
| Subtype relationship | No — Dog is not an Animal | Yes |
| Interface satisfaction | Through promoted methods | Through class hierarchy |
| Method shadowing | Explicit, compile-time | Virtual dispatch, runtime |
| Multiple "parents" | Yes — embed many types | Depends on language |
| Diamond resolution | Required, explicit | Language-specific |

---

## Running the examples

```bash
cd book/part2_core_language/chapter23_embedding

go run ./examples/01_composition    # mixin, diamond, interface promotion
go run ./exercises/01_middleware_stack # middleware chain via embedding
```

---

## Key takeaways

1. Embedding **promotes** fields and methods — they appear on the outer type directly.
2. Embedding is **composition, not inheritance** — no subtype relationship.
3. If two embedded types promote the same name, you must **shadow it** or the access is ambiguous.
4. Embedding `T` where `T` satisfies interface `I` makes the outer type also satisfy `I`.
5. The **mixin pattern** (embed a struct for shared behaviour) is idiomatic and widely used.

---

## Cross-references

- **Chapter 20** — Structs: basic embedding mechanics
- **Chapter 21** — Methods: pointer receivers and promotion
- **Chapter 22** — Interfaces: embedding to satisfy interfaces
