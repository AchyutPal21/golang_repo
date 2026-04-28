# Chapter 21 — Methods: Functions With Receivers

> **Part II · Core Language** | Estimated reading time: 20 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Methods are what turn data into behaviour. Understanding when to use a value receiver vs. a pointer receiver, what method sets are, and how method values/expressions work is essential before tackling interfaces. The method set rules are also the most common source of "does not implement interface" compile errors.

---

## 21.1 — Syntax

```go
func (receiver Type) MethodName(params) ReturnTypes { ... }
```

The receiver can be any named type defined in the same package:

```go
type Duration float64

func (d Duration) String() string { ... }
```

Methods can be defined on structs, named scalars, named slices, named maps — anything except pointer types, interface types, or types from other packages.

---

## 21.2 — Value vs pointer receiver

| Receiver | Can mutate? | Called on | Auto-dereference? |
|---|---|---|---|
| `T` | No (copy) | `T` or `*T` | Yes |
| `*T` | Yes | `*T` or addressable `T` | Yes |

**Use a pointer receiver when**:
- The method modifies the receiver
- The receiver is large (avoid copy cost)
- Consistency: if any method uses `*T`, all should

**Use a value receiver when**:
- The method is a pure read (e.g., `Area()`, `String()`)
- The type is a small immutable value (like `Point`)

---

## 21.3 — Method sets

The **method set** of a type determines which interfaces it satisfies.

- Method set of `T`: all methods with receiver `T`
- Method set of `*T`: all methods with receiver `T` **plus** all with receiver `*T`

This asymmetry explains the most common interface assignment error:

```go
var s ShapeI = Rect{...}  // error if Scale has *Rect receiver
var s ShapeI = &Rect{...} // OK: *Rect's method set includes Scale
```

---

## 21.4 — Nil receiver

A method with a pointer receiver can be called on a nil pointer, as long as the method guards against it:

```go
func (t *Tree) Sum() int {
    if t == nil { return 0 }
    return t.Value + t.Left.Sum() + t.Right.Sum()
}

var root *Tree // nil
root.Sum()     // safe: 0
```

This pattern is used extensively in recursive tree/list data structures.

---

## 21.5 — Method expressions and method values

A **method expression** turns a method into a regular function with the receiver as the first parameter:

```go
areaFn := Circle.Area        // func(Circle) float64
areaFn(Circle{Radius: 5})    // == Circle{Radius: 5}.Area()
```

A **method value** binds a method to a specific receiver:

```go
c := Circle{Radius: 5}
bound := c.Area   // func() float64 — c is captured
bound()           // == c.Area()
```

Method values are useful when passing methods as callbacks or storing them in function variables.

---

## Running the examples

```bash
cd book/part2_core_language/chapter21_methods

go run ./examples/01_method_basics  # value/pointer receivers, nil receiver, method expressions
go run ./examples/02_method_sets    # method sets, interface satisfaction, addressability

go run ./exercises/01_geometry      # Shape interface with Circle, Rectangle, Triangle
```

---

## Exercises

### [exercises.md](exercises.md)

---

## Revision checkpoint

### [checkpoint.md](checkpoint.md)

---

## Key takeaways

1. **Pointer receiver** when mutating; **value receiver** when reading. Be consistent within a type.
2. **`*T`'s method set** includes all of `T`'s methods plus its own pointer-receiver methods.
3. To satisfy an interface with pointer-receiver methods, assign `&value`, not `value`.
4. **Nil pointer receiver** methods are valid and useful for recursive data structures.
5. **Method values** bind a method to a receiver — usable as first-class function values.

---

## Cross-references

- **Chapter 20** — Structs: embedding promotes methods
- **Chapter 22** — Interfaces: method sets determine interface satisfaction
- **Chapter 23** — Embedding and Composition: promoted methods and interface satisfaction
