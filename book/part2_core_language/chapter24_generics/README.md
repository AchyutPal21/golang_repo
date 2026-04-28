# Chapter 24 — Generics: Type Parameters and Constraints

> **Part II · Core Language** | Estimated reading time: 22 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Generics (Go 1.18) eliminated the need for `interface{}` / `any`-based containers that lose type safety, and for code generation to produce type-specific versions of the same algorithm. Understanding type parameters, constraints, and the `~` operator gives you a powerful tool — and understanding when NOT to use them keeps your code readable.

---

## 24.1 — Syntax

```go
func Map[T, U any](s []T, f func(T) U) []U { ... }

type Stack[T any] struct { items []T }
```

Type parameters are listed in `[...]` before the parameter list. The constraint after each parameter (like `any` or `comparable`) limits what types can be substituted.

---

## 24.2 — Built-in constraints

| Constraint | Meaning |
|---|---|
| `any` | Any type (alias for `interface{}`) |
| `comparable` | Types that support `==` and `!=` |

---

## 24.3 — Custom constraints

Constraints are interface types. The `~T` operator means "any type whose underlying type is T":

```go
type Ordered interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 |
    ~uint | ~float32 | ~float64 | ~string
}

func Min[T Ordered](a, b T) T { if a < b { return a }; return b }
```

`~int` includes `int` itself and any named type like `type MyInt int`.

---

## 24.4 — Type inference

The compiler infers type parameters from the arguments in most cases:

```go
Min(3, 5)           // T inferred as int
Min("a", "b")       // T inferred as string
Map(nums, strconv.Itoa) // T inferred as int, U as string
```

Explicit type parameters are only needed when inference is impossible.

---

## 24.5 — When NOT to use generics

| Avoid generics when | Use instead |
|---|---|
| Single concrete type | Just use that type |
| Behaviour differs per type | Interface + type switch |
| Runtime type inspection | reflect |
| The generic is less readable | Three simple copies |

The canonical wrong use: `func PrintAny[T any](v T) { fmt.Println(v) }` — just use `fmt.Println`.

A useful heuristic: if you can't explain the type parameter in one sentence, the function probably doesn't need generics.

---

## 24.6 — slices and maps packages (Go 1.21)

`golang.org/x/exp/slices` (now `slices` in stdlib since Go 1.21) provides:
- `slices.Sort[S ~[]E, E cmp.Ordered](x S)`
- `slices.Contains[S ~[]E, E comparable](s S, v E) bool`
- `slices.Map`, `slices.Filter` via iterator patterns

---

## Running the examples

```bash
cd book/part2_core_language/chapter24_generics

go run ./examples/01_type_params    # Map, Filter, Reduce, Contains, Pair, Optional
go run ./examples/02_constraints    # Ordered, Number, ~, Clamp, Sum, MyInt

go run ./exercises/01_collections   # generic OrderedSet + Union
```

---

## Key takeaways

1. **Type parameters** enable one implementation for many types, with compile-time type safety.
2. **`any`** means no constraint; **`comparable`** enables `==`.
3. **`~T`** includes the named type and all types with `T` as underlying type.
4. **Type inference** eliminates most explicit type parameter annotations.
5. **Don't overuse generics** — interfaces + type switches often read better for behaviour dispatch.

---

## Cross-references

- **Chapter 22** — Interfaces: constraints are interface types
- **Chapter 18** — Slices: `slices` package uses generics
- **Chapter 19** — Maps: `maps` package uses generics
