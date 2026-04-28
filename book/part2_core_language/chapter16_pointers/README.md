# Chapter 16 — Pointers and Memory Addressing

> **Part II · Core Language** | Estimated reading time: 22 min | Runnable examples: 3 | Exercises: 1

---

## Why this chapter matters

Go is not C — you rarely think about pointers explicitly. But they are always present: every method with a pointer receiver, every slice header, every interface value, every channel uses them. Understanding when Go copies a value and when it shares one is the difference between code that works and code that silently mutates (or fails to mutate) data. This chapter covers the mechanics, the escape-analysis picture, and the handful of real-world pointer patterns you will use every day.

---

## 16.1 — The two pointer operators

`&` takes the address of a variable; `*` dereferences a pointer:

```go
x := 42
p := &x    // p is *int; p holds the address of x
*p = 100   // dereference: sets x to 100
fmt.Println(x) // 100
```

Go auto-dereferences pointer-to-struct field access:

```go
pt := &Point{X: 3, Y: 4}
pt.X = 9   // equivalent to (*pt).X = 9
```

---

## 16.2 — Why pointers exist

Go is a value-semantics language by default: assignment copies. Pointers let you opt in to reference semantics:

| Goal | Mechanism |
|---|---|
| Mutate the caller's variable | Pass a pointer |
| Share a large struct without copying | Pass a pointer |
| Express "no value" (optional) | Use `*T` (nil = absent) |
| Implement a method that mutates the receiver | Pointer receiver |

---

## 16.3 — nil pointer

A pointer's zero value is `nil`. Dereferencing a nil pointer panics at runtime:

```go
var p *int
*p = 1 // panic: nil pointer dereference
```

Always guard before dereferencing a pointer that could be nil:

```go
func zero(p *int) {
    if p == nil { return }
    *p = 0
}
```

---

## 16.4 — new()

`new(T)` allocates a zero-valued `T` on the heap, returns `*T`:

```go
p := new(int)   // *int, pointing to 0
*p = 42
```

`new(T)` is equivalent to `&T{}` for structs and to `var x T; &x` for scalars. In practice, `new` is rarely used; composite literals with `&` are more common for structs.

---

## 16.5 — Pointer comparison

Two pointer values are equal if and only if they point to the same variable:

```go
a, b := 1, 1
&a == &a  // true  — same variable
&a == &b  // false — different variables, even though *&a == *&b
```

This matters: you cannot use `==` to compare the pointed-to values; you must dereference first.

---

## 16.6 — Stack vs heap: escape analysis

The Go compiler performs *escape analysis* to decide whether a variable lives on the goroutine's stack or on the heap.

**Stack**: fast allocation, automatic reclamation when the function returns, no GC pressure.

**Heap**: survives the function's return, managed by the garbage collector.

A variable *escapes* to the heap when:
- Its address is returned from the function
- It is captured by a closure that outlives the function
- It is assigned to an interface
- It is too large for the stack

```go
func stackAlloc() int {
    x := 42       // stays on stack
    return *(&x)
}

func heapAlloc() *int {
    x := 42       // escapes: address is returned
    return &x
}
```

You never manage this manually — the compiler decides. To see the decisions:

```bash
go build -gcflags="-m" ./mypackage
```

Output lines like `x escapes to heap` or `moved to heap: x` tell you what escaped and why.

> Heap allocations add GC pressure. In hot paths, prefer value semantics and passing by value over pointers to structs. Profile first; don't optimise blindly.

---

## 16.7 — Pointer receivers

Methods that modify the receiver must use a pointer receiver:

```go
func (p *Point) Scale(factor float64) {
    p.X *= factor
    p.Y *= factor
}
```

Consistency rule: if any method needs a pointer receiver, all methods on that type should use pointer receivers. Mixing causes subtle bugs with interface satisfaction.

---

## 16.8 — Optional values with *T

The zero value of `*T` is `nil`, which lets you represent "not set" distinctly from the zero value of `T`:

```go
type Config struct {
    Port  *int  // nil = use default; 0 = bind to random port
    Debug *bool // nil = inherit from env
}
```

Helper functions make this ergonomic:

```go
func intPtr(n int) *int   { return &n }
func boolPtr(b bool) *bool { return &b }

cfg := Config{Port: intPtr(9090)}
```

This pattern is common in API clients, JSON unmarshalling, and configuration structs where you need to distinguish "caller did not supply a value" from "caller supplied zero".

---

## 16.9 — Pointer to interface: an antipattern

You almost never need `*SomeInterface`. Interfaces are already reference types internally (they hold a pointer to the concrete value). Taking the address of an interface just adds a layer of indirection with no benefit:

```go
// Wrong: *Stringer is almost never what you want
func print(s *Stringer) { (*s).String() }

// Correct: pass the interface value directly
func print(s Stringer) { s.String() }
```

If you find yourself writing `*io.Reader` or `*error`, stop and reconsider.

---

## 16.10 — Struct field layout and alignment

The CPU can only read aligned values efficiently. The compiler pads structs to satisfy alignment requirements:

```go
type Bad struct {
    A bool    // 1 byte + 7 bytes padding
    B int64   // 8 bytes
    C uint8   // 1 byte + 3 bytes padding
    D float32 // 4 bytes
} // total: 24 bytes

type Good struct {
    B int64   // 8 bytes (largest first)
    D float32 // 4 bytes
    A bool    // 1 byte
    C uint8   // 1 byte + 2 bytes padding
} // total: 16 bytes
```

For hot structs (cached, allocated in large numbers), ordering fields largest-first saves memory and improves cache performance. Use `unsafe.Sizeof` and `-gcflags="-m"` to inspect.

---

## 16.11 — No pointer arithmetic

Go has no `++p` or `p + n` for pointers. This eliminates a large class of memory-safety bugs. The `unsafe` package provides `unsafe.Add` for the rare cases (FFI, memory-mapped I/O) where arithmetic is genuinely needed.

---

## Running the examples

```bash
cd book/part2_core_language/chapter16_pointers

go run ./examples/01_pointer_basics    # &, *, nil, new, comparison
go run ./examples/02_escape_analysis   # stack vs heap, closure escape
go run ./examples/03_pointer_patterns  # optional *T, alignment, antipatterns

go run ./exercises/01_linked_list      # pointer-based linked list

# Inspect escape analysis decisions:
go build -gcflags="-m" ./examples/02_escape_analysis
```

---

## Exercises

### [exercises.md](exercises.md)

---

## Revision checkpoint

### [checkpoint.md](checkpoint.md)

---

## Key takeaways

1. `&` takes an address; `*` dereferences. Go auto-dereferences struct field access through a pointer.
2. Dereferencing `nil` panics — always guard.
3. The compiler's escape analysis decides stack vs heap; you don't.
4. Pointer receivers are required for methods that mutate state; be consistent within a type.
5. `*T` is the idiomatic "optional value" in Go when you need to distinguish absent from zero.
6. Never take the address of an interface value — interfaces are already reference types.
7. Order struct fields largest-to-smallest to minimise padding.

---

## Cross-references

- **Chapter 20** — Structs: embedding, field tags, layout
- **Chapter 21** — Methods: pointer vs value receivers, method sets
- **Chapter 22** — Interfaces: why `*SomeInterface` is wrong
- **Chapter 36** — sync: `sync.Mutex` and why pointer receivers matter for synchronisation
