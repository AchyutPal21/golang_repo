# Chapter 13 — Functions: First-Class Citizens

> **Part II · Core Language** | Estimated reading time: 25 min | Runnable examples: 3 | Exercises: 1

---

## Why this chapter matters

Functions in Go are more than callable blocks of code. They are values, they carry state in closures, they can be stored in maps and slices, returned from other functions, and composed into pipelines. Understanding the full function model — multiple returns, variadic parameters, `init`, function types — is the prerequisite for every pattern in the chapters that follow: closures (Ch 14), defer/panic/recover (Ch 15), methods (Ch 21), and interfaces (Ch 22).

---

## 13.1 — The basic signature

```go
func name(param1 Type1, param2 Type2) ReturnType { ... }
```

Parameters of the same type can be grouped:

```go
func add(a, b int) int { return a + b }
```

Go has no default arguments, no keyword arguments, and no function overloading. These constraints are intentional: they make call sites unambiguous to read.

---

## 13.2 — Multiple return values

Go functions can return more than one value. This is the idiomatic way to return a result alongside an error — instead of exceptions, Go functions say "here is what I computed, and here is whether it worked".

```go
func divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}
```

The caller unpacks with multi-assignment:

```go
result, err := divide(10, 3)
if err != nil {
    log.Fatal(err)
}
```

### The blank identifier

When a return value is not needed, use `_` to discard it:

```go
_, wordCount, _ := fileStats(text)
```

> **Rule**: never ignore errors by assigning them to `_`. Discard only values you genuinely do not need. See example 01 for a multi-value function where ignoring some values is legitimate.

---

## 13.3 — Named return values

Return values can be given names in the signature:

```go
func minMax(nums []int) (min, max int) {
    ...
    return // naked return
}
```

Named returns serve two purposes:

1. **Documentation** — the names appear in godoc and make the signature self-describing without a comment.
2. **Naked returns** — a bare `return` statement returns the current values of the named variables, useful in short functions to avoid repeating variable names.

> **Style rule**: use naked returns only in short functions (< ~10 lines). In longer functions they harm readability because the reader cannot see what is being returned.

---

## 13.4 — The error idiom

Multiple return values unlock Go's canonical error-handling pattern:

```go
value, err := doSomething()
if err != nil {
    // handle or propagate
    return zeroValue, fmt.Errorf("doSomething: %w", err)
}
// value is safe to use
```

This pattern appears at every layer of Go programs. `%w` wraps the error for `errors.Is` and `errors.As` unwrapping (covered in detail in Chapter 41 — Errors as Values).

---

## 13.5 — Variadic functions

A variadic function accepts zero or more values of a given type as its last parameter, using `...`:

```go
func sum(nums ...int) int { ... }

sum()           // nums = []int{}
sum(1, 2, 3)    // nums = []int{1, 2, 3}
```

Inside the function, the variadic parameter is a slice. You can spread an existing slice into a variadic call:

```go
nums := []int{10, 20, 30}
total := sum(nums...) // spreads the slice
```

### Required plus variadic

A non-variadic first parameter enforces at least one argument:

```go
func max(first int, rest ...int) int { ... }

max(5)          // OK: rest = []int{}
max(3, 1, 4)    // OK: rest = []int{1, 4}
// max()        // compile error
```

> `fmt.Println`, `fmt.Printf`, `append` are all variadic. The `...` operator is not special syntax for those functions — it is a general language feature.

---

## 13.6 — The `init` function

`init` is a special function with no parameters and no return values:

```go
func init() {
    // package-level setup
}
```

Rules:
- Runs automatically before `main`, after all package-level `var` declarations are initialised.
- A package can have multiple `init` functions, in multiple files.
- You cannot call `init` manually — the compiler rejects it.
- `init` runs once per package import, not once per use.

Common uses: registering drivers (`database/sql` drivers use this), validating configuration, seeding random state. The idiomatic preference is to avoid `init` for anything that can fail, because `init` cannot return an error.

---

## 13.7 — Functions as values

In Go, functions are first-class values. A function can be:

- Assigned to a variable
- Passed as an argument
- Returned from another function
- Stored in a slice, map, or struct field

```go
type Transformer func(string) string

var t Transformer = strings.ToUpper
fmt.Println(t("hello")) // "HELLO"
```

Naming a function type with `type` makes signatures readable and allows method definitions on the type.

---

## 13.8 — Higher-order functions

A function that takes or returns another function is called *higher-order*:

```go
func apply(ss []string, t Transformer) []string {
    out := make([]string, len(ss))
    for i, s := range ss {
        out[i] = t(s)
    }
    return out
}
```

This is the building block of functional-style patterns in Go: `filter`, `map`, `reduce`. The standard library uses this extensively (`sort.Slice`, `http.HandleFunc`, `sync/atomic.CompareAndSwapInt64`).

> Go does not have built-in `map`/`filter`/`reduce` functions. Since Go 1.23, `iter` and `slices` packages cover the most common use cases. For prior versions (and for learning), implementing them yourself is instructive.

---

## 13.9 — Function factories

A factory function returns a new function value configured by its arguments:

```go
func adder(delta int) func(int) int {
    return func(n int) int {
        return n + delta
    }
}

add10 := adder(10)
add10(5)  // 15
```

`delta` lives in the returned function's *closure environment* — covered in depth in Chapter 14.

---

## 13.10 — Dispatch tables

Functions stored in maps form dispatch tables — an idiomatic alternative to long switch statements when the set of operations is extensible:

```go
var ops = map[string]func(string) string{
    "upper": strings.ToUpper,
    "lower": strings.ToLower,
}

fn := ops[cmd]  // look up, not switch
fn(input)
```

This pattern is pervasive in Go web handlers, command routers, codec registries, and plugin systems.

---

## 13.11 — Function composition

Two Transformers can be composed into a single one:

```go
func chain(f, g Transformer) Transformer {
    return func(s string) string {
        return g(f(s))
    }
}
```

`chain` is a pure function that returns a new function; it has no side effects and can be called freely. Composition is the basis of middleware stacks, pipeline builders, and decorator patterns throughout the Go ecosystem.

---

## 13.12 — Memoization

Because functions are values, you can wrap them to add caching:

```go
func memoize(f func(int) int) func(int) int {
    cache := map[int]int{}
    return func(n int) int {
        if v, ok := cache[n]; ok {
            return v
        }
        v := f(n)
        cache[n] = v
        return v
    }
}
```

The cache map lives in the closure. This pattern works for pure functions (deterministic, no side effects). See example 03 for a runnable demonstration.

---

## 13.13 — Generic higher-order functions (Go 1.18+)

With generics, `filter` and similar utilities become truly reusable:

```go
type Predicate[T any] func(T) bool

func filter[T any](items []T, pred Predicate[T]) []T {
    var out []T
    for _, v := range items {
        if pred(v) {
            out = append(out, v)
        }
    }
    return out
}
```

The standard library's `slices` and `maps` packages (Go 1.21) build on this foundation.

---

## 13.14 — What you cannot do with functions

Go functions are first-class but not unlimited:

| Feature | Go |
|---|---|
| Overloading | No |
| Default arguments | No |
| Keyword arguments | No |
| Methods on `func` literals | No (use named type) |
| `func` equality comparison | Only to `nil` |

`func` values cannot be compared with `==` (you cannot ask "is f the same function as g?") except to test whether a function variable holds `nil`. This prevents a class of subtle bugs common in languages with reflection-based equality.

---

## 13.15 — Practical patterns summary

| Pattern | Construct |
|---|---|
| Value + error result | `func f() (T, error)` |
| Self-documenting multi-return | Named return values |
| Optional arguments | Variadic `...T` |
| Pass behaviour | `func` type parameter |
| Configurable behaviour | Factory returns `func` |
| Extensible dispatch | `map[string]func(...)` |
| Composition | Function that returns `func` |
| Package setup | `func init()` |

---

## Running the examples

All commands run from this chapter's directory:

```bash
cd book/part2_core_language/chapter13_functions

go run ./examples/01_multi_return   # multiple returns, named returns, naked return
go run ./examples/02_variadic       # variadic params, init, slice spreading
go run ./examples/03_function_types # function types, HOF, factories, memoize, dispatch

go run ./exercises/01_pipeline      # string pipeline + Caesar cipher
```

---

## Examples

### [examples/01_multi_return/main.go](examples/01_multi_return/main.go)

Demonstrates: `divide` (value+error idiom), `minMax` (named returns + naked return), `parseCoord` (early-exit via named return), `fileStats` (three independent return values with `_` discard).

### [examples/02_variadic/main.go](examples/02_variadic/main.go)

Demonstrates: `sum` (basic variadic), `max` (required + variadic), `joinWith`, `appendUnique` (slice spread), `init` function with package-level state.

### [examples/03_function_types/main.go](examples/03_function_types/main.go)

Demonstrates: `Transformer`/`Predicate` named function types, `apply`, generic `filter`, `chain` composition, `repeat` factory, `adder` factory, `memoize` closure, dispatch table.

---

## Exercises

### [exercises.md](exercises.md)

---

## Revision checkpoint

### [checkpoint.md](checkpoint.md)

---

## Key takeaways

1. **Multiple returns** are the idiomatic error-reporting mechanism; they replace exceptions.
2. **Named returns** are documentation and enable naked returns in short functions.
3. **Variadic** parameters are slices inside the function; spread with `...`.
4. **`init`** runs automatically before `main`; cannot be called manually; prefer it only for side-effect-free registration.
5. **Functions are values**: they can be stored, passed, and returned — the basis of composition, factories, middleware, and dispatch.
6. **Naming a function type** (`type Transformer func(string) string`) makes code more readable and enables methods.

---

## Cross-references

- **Chapter 14** — Closures and the Capture Model: the closure environment that function factories rely on
- **Chapter 15** — defer, panic, recover: deferred functions as a special execution form
- **Chapter 21** — Methods: functions attached to types via receivers
- **Chapter 22** — Interfaces: the role of function types in interface satisfaction
- **Chapter 41** — Errors as Values: the full error idiom built on multiple returns
