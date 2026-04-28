# Chapter 10 — Type Conversion, Type Assertion, Type Switch

> **Reading time:** ~22 minutes (5,500 words). **Code:** 3 runnable
> programs (~250 lines). **Target Go version:** 1.22+.
>
> Three operations beginners conflate. Conversion changes a value's
> type at compile time when types are convertible. Assertion extracts
> a concrete type from an interface at runtime. Type switch
> dispatches on an interface's dynamic type. Different syntaxes,
> different rules, different costs.

---

## 1. Concept Introduction

Every value in Go has a static type known at compile time. Three
distinct mechanisms move values between types:

* **Conversion** — `T(x)` — produces a value of type `T` from `x`,
  if `T` is assignable from `x`'s type. Compile-time check.
  Examples: `int64(myInt32)`, `string(myBytes)`, `MyType(otherType)`.
* **Type assertion** — `x.(T)` — extracts a concrete `T` from an
  interface value `x`. Runtime check; panics or returns `(zero,
  false)` if `x` doesn't actually hold a `T`.
* **Type switch** — `switch v := x.(type) { case T1: ...; case T2: ... }` —
  dispatches on the dynamic type of an interface. Same runtime
  mechanism as assertion, but multi-way.

> **Working definition:** conversion is for *concrete-to-concrete*
> type changes (compile-time). Assertion is for *interface-to-concrete*
> extraction (runtime). Type switch is multi-way assertion. The
> three never overlap; using the wrong one is a compile error.

---

## 2. Why This Exists

A statically-typed language with interfaces needs a way to:

1. Move values between numeric types (conversion).
2. Recover the concrete type from a polymorphic interface value
   (assertion).
3. Branch on what concrete type an interface holds (type switch).

Go could have collapsed these into one cast operator like Java's
`(T)x`, but the team decided the three operations were
semantically distinct enough to deserve distinct syntax. The
benefit: when you read `x.(T)`, you know an interface is being
unwrapped; when you read `T(x)`, you know it's a value-shape
change. No ambiguity.

---

## 3. Problem It Solves

1. **Numeric-type mixing.** `int32 + int64` is a compile error;
   `int64(myInt32) + myInt64` works.
2. **Polymorphism.** Functions can accept `io.Reader`; callers
   pass an `*os.File`, a `*bytes.Buffer`, etc. Inside the
   function, an assertion can recover the concrete type if needed.
3. **Switch-by-type dispatch.** Without type switch, you'd need
   chained `if _, ok := x.(T); ok` blocks. The dedicated form is
   cleaner and faster.
4. **Safe extraction.** The two-result assertion form
   (`v, ok := x.(T)`) lets you check without panicking.

---

## 4. Historical Context

The conversion-vs-assertion split was settled early in Go's
design. The 2009 release had both forms with their current
syntax. The two-result `v, ok := x.(T)` form was added because
the team didn't want assertion to *always* panic — that would be
hostile to safe runtime polymorphism.

The "comma-ok idiom" (`v, ok := ...`) shows up in three places:

* Type assertion: `v, ok := x.(T)`.
* Map lookup: `v, ok := m[k]`.
* Channel receive: `v, ok := <-ch`.

Same shape, three meanings, one mental model: "did the operation
yield something?"

---

## 5. How Industry Uses It

* **Numeric conversions in I/O.** Reading `int32` from a binary
  protocol, converting to `int` for arithmetic.
* **Type assertions on `error`.** `if pe, ok := err.(*MyError); ok { ... }`
  was the pattern pre-`errors.As`; it's still seen in older code.
* **Type switches on interfaces.** `encoding/json` uses one to
  dispatch on `Marshaler`/`json.Number`/built-in kinds.
* **`reflect.ValueOf(x).Kind()` + switch** — an even more dynamic
  form, covered in Chapter 25.
* **`any` (interface{}) parameters.** Functions like
  `fmt.Printf` accept `any`; internally, type switches dispatch.

---

## 6. Real-World Production Use Cases

**Error inspection.** Pre-`errors.As` (Go 1.13), every Go
codebase had:

```go
if syscallErr, ok := err.(*os.SyscallError); ok {
    // unwrap and inspect
}
```

Modern code uses `errors.As`, which is built on top of type
assertion.

**JSON unmarshaling into `any`.** The standard library returns
unknown JSON values as `any`. Inspecting them requires a type
switch:

```go
switch v := raw.(type) {
case map[string]any: ...
case []any: ...
case string: ...
case float64: ...   // numbers come back as float64 by default
case bool: ...
case nil: ...
}
```

**Plugin dispatch.** A worker that handles different message
types reads them off a channel and dispatches with a type switch
— no reflection, no string-keyed registry, just compile-time-
typed branches.

**Database driver scan.** `database/sql/driver.Valuer` and
`Scanner` interfaces require type assertions to convert
`driver.Value` into Go types.

---

## 7. Beginner-Friendly Explanation

Three syntaxes; here's the cheat sheet:

| Operation | Syntax | When it fails |
| --- | --- | --- |
| Conversion | `T(x)` | Compile error |
| Assertion (panicky) | `x.(T)` | Runtime panic |
| Assertion (safe) | `v, ok := x.(T)` | `ok = false` |
| Type switch | `switch v := x.(type) { case T: ... }` | Falls through to `default` |

Conversion: `int64(x)` from `int32`. Compile-time. If the types
aren't convertible, the compiler refuses.

Assertion: `r.(*os.File)` to extract a concrete type from an
interface. Runtime check. If the interface doesn't actually hold
that type, you get a panic — *unless* you use the `, ok` form,
which gives you a safe boolean.

Type switch: when you need to handle several possible concrete
types from one interface. It's a chain of assertions in one
syntax.

---

## 8. Deep Technical Explanation

### 8.1. Conversion

`T(x)` produces a value of type `T` from `x`. Rules:

* **Numeric.** Any numeric type converts to any other numeric
  type. May truncate (`int64 → int8`) or change sign (`int → uint`).
  Lossy conversions are allowed; the language doesn't warn.
* **String/bytes/runes.** `string ↔ []byte` and `string ↔ []rune`
  are special-cased conversions that copy data and (for runes)
  encode/decode UTF-8.
* **Pointer.** `*T → unsafe.Pointer → *U` is allowed via
  `unsafe`. Otherwise pointer types don't convert.
* **Named types.** A type defined as `type Celsius float64`
  converts to `float64` and vice versa: `Celsius(myFloat)`,
  `float64(myCelsius)`. They share the underlying type.
* **Slice ↔ array pointer (Go 1.17+).** `(*[N]byte)(slice)` works
  if `len(slice) >= N`.
* **Slice ↔ array (Go 1.20+).** `[N]byte(slice)` works if
  `len(slice) >= N`.

What's NOT conversion:

* `int → string` is *legal* but means "interpret int as Unicode
  code point" — almost never what you want. `vet` warns. Use
  `strconv.Itoa`.
* No implicit conversion. `int32 + int64` is a compile error.

### 8.2. Type assertion

`x.(T)` requires `x` to be of an interface type. It returns a
value of type `T`:

```go
var r io.Reader = os.Stdin
f := r.(*os.File)  // OK if r holds *os.File; PANICS otherwise
```

The panicky form should only be used when you're certain (and
crashing is acceptable if you're wrong). The safer form:

```go
f, ok := r.(*os.File)
if !ok {
    // handle non-file reader
}
```

Two cases worth knowing:

* **Asserting an interface.** `x.(io.Closer)` checks if `x`'s
  dynamic type satisfies `io.Closer`. Common in middleware:
  `if c, ok := body.(io.Closer); ok { c.Close() }`.
* **Asserting `T` from `any`.** `any` (alias for `interface{}`)
  holds any value. `v, ok := x.(int)` extracts an `int` if `x`
  holds one.

### 8.3. Type switch

```go
switch v := x.(type) {
case int:
    // v is int here
case string:
    // v is string here
case nil:
    // x is the nil interface
case io.Reader:
    // v is io.Reader (note: an interface case)
default:
    // v is x's dynamic type, unrecognised
}
```

The pseudo-keyword `(type)` is only valid inside a `switch`
expression. `v` is implicitly retyped per case — a feature you
won't see anywhere else in Go.

You can group types in one case:

```go
switch x.(type) {
case int, int32, int64:
    // v is the original interface here (NOT retyped)
}
```

When multiple types share a case, `v` keeps the interface type
because there's no single concrete type to assign.

### 8.4. The two-word interface and `nil`

An interface is two words internally: a type word and a data
word. The interface is `nil` *only* when *both* are zero.

```go
var p *MyError = nil
var err error = p   // err is NOT nil! Type word holds *MyError.
fmt.Println(err == nil) // false
```

This is one of Go's most-asked interview questions. The fix:
return `nil` directly, not a typed nil pointer wrapped in an
interface.

```go
func f() error {
    var p *MyError
    if condition {
        p = &MyError{...}
    }
    if p == nil {
        return nil // explicitly nil interface
    }
    return p
}
```

We'll go deeper in Chapter 22.

---

## 9. Internal Working (How Go Handles It)

* **Conversion** is mostly compile-time. `int64(int32)` becomes a
  sign-extension instruction at the machine-code level.
* **Type assertion** consults the interface's *itab* (the
  compiler-generated method-set descriptor) at runtime. The
  comparison is one pointer compare against the target type's
  itab pointer.
* **Type switch** generates an itab compare per case in
  declaration order. With many cases, the compiler may use a
  lookup table. The cost per case is constant.
* **Comma-ok forms** compile to the same itab compare followed by
  a conditional move; no extra cost beyond the boolean.

Read `runtime/iface.go` for the canonical implementation.

---

## 10. Syntax Breakdown

```go
// Conversion (concrete → concrete)
n := int64(myInt32)
s := string(myBytes)
f := float64(myInt)
c := MyType(otherType) // works if same underlying type

// Assertion (interface → concrete) — panicky
f := r.(*os.File)

// Assertion — safe (comma-ok)
f, ok := r.(*os.File)
if !ok { ... }

// Assertion to an interface (also OK)
c, ok := x.(io.Closer)

// Type switch
switch v := x.(type) {
case int: fmt.Println("int:", v)
case string: fmt.Println("string:", v)
case nil: fmt.Println("nil")
default: fmt.Printf("other (%T): %v\n", v, v)
}
```

---

## 11. Multiple Practical Examples

### `examples/01_conversions`

Numeric, string/byte, and named-type conversions side by side.
Shows what the compiler accepts and what produces panics or
wrong values.

```bash
go run ./examples/01_conversions
```

### `examples/02_type_assertions`

Two-result assertions in error inspection and `io.Reader`
narrowing. Demonstrates the typed-nil-interface gotcha.

```bash
go run ./examples/02_type_assertions
```

### `examples/03_type_switch`

A small expression evaluator: walks an `any`-typed AST node
tree, dispatches with a type switch.

```bash
go run ./examples/03_type_switch
```

---

## 12. Good vs Bad Examples

**Good:**

```go
v, ok := x.(*MyType)
if !ok { return errBadInput }
// use v
```

**Bad:**

```go
v := x.(*MyType) // panics if x is something else
```

Use the panicky form *only* when you've already type-switched
and the case is unreachable, or when a panic is the right
response.

**Good — type switch when many cases:**

```go
switch v := x.(type) {
case *Cat: v.Meow()
case *Dog: v.Bark()
case *Bird: v.Sing()
default: log.Println("unknown animal")
}
```

**Bad — chained assertions:**

```go
if c, ok := x.(*Cat); ok { c.Meow(); return }
if d, ok := x.(*Dog); ok { d.Bark(); return }
if b, ok := x.(*Bird); ok { b.Sing(); return }
log.Println("unknown animal")
```

The type switch generates better code (single dispatch) and is
easier to read.

---

## 13. Common Mistakes

1. **Conflating conversion and assertion.** `int(myInterface)`
   is a compile error; you want `myInterface.(int)`.
2. **Panicky assertion in untrusted contexts.** Always use the
   comma-ok form when the type isn't guaranteed.
3. **Assuming `string(intVal)` formats a number.** It interprets
   as a rune. Use `strconv.Itoa`.
4. **Forgetting that JSON numbers come back as `float64`.** A
   type switch on `any` for a JSON number must match `float64`,
   not `int`.
5. **Typed-nil-interface confusion.** Returning a typed `*T`
   that happens to be nil through an `error` interface yields
   a non-nil interface.
6. **Type switching on a concrete type.** Type switch only
   works on interface values.
7. **Putting multiple types in one case and trying to use the
   typed `v`.** Inside a multi-type case, `v` keeps the
   interface type.
8. **Slicing into an array conversion that's too short.** Since
   1.20, `[N]byte(slice)` panics if `len(slice) < N`. Check
   length first.

---

## 14. Debugging Tips

* `fmt.Printf("%T\n", x)` to see the dynamic type of an
  interface.
* `errors.As(err, &target)` for typed-error extraction; it's
  the modern alternative to manual assertion chains.
* `go vet` catches `string(int)` mistakes.
* `reflect.TypeOf(x).String()` if you need the type name as a
  runtime value.

---

## 15. Performance Considerations

* **Conversion is free or near-free.** A sign-extension
  instruction at most. No allocation.
* **Assertion is one pointer compare.** O(1), no allocation.
* **Type switch with N cases is O(N)** in the worst case
  (linear scan of the cases). The compiler may optimize for
  large switches.
* **Avoid `interface{}` when you can.** Generic functions
  (Chapter 24) often eliminate the need for runtime dispatch.

---

## 16. Security Considerations

* **Untrusted JSON.** A type switch on `any` from
  `json.Unmarshal` must handle every possible JSON shape —
  including unexpected nesting. Use `json.RawMessage` to defer
  parsing.
* **Panicky assertion** in a server handler crashes the
  process. Always use comma-ok at trust boundaries.

---

## 17. Senior Engineer Best Practices

1. **Use `errors.As` over manual error-type assertion.**
2. **Comma-ok at trust boundaries; panicky only in deep
   internal code where the contract is established.**
3. **Type switch over chained ifs** when you have ≥3 cases.
4. **Avoid `any` parameters.** Generics (Chapter 24) are
   usually a better answer.
5. **`go vet` on every commit** — catches `string(int)`.
6. **Document interface assertions in comments.** "We assert
   `*os.File` here because…"

---

## 18. Interview Questions

1. *(junior)* What's the difference between `T(x)` and `x.(T)`?
2. *(mid)* Why is `var err error = (*MyErr)(nil); err == nil`
   `false`?
3. *(senior)* When does the panicky form `x.(T)` make sense?
4. *(senior)* Compare type switch to chained `if x, ok :=
   y.(T); ok` — pros, cons, performance.

## 19. Interview Answers

1. **`T(x)`** is a *conversion* — compile-time, between
   concrete types. **`x.(T)`** is a *type assertion* — runtime,
   to extract a concrete type from an interface.

2. The interface has two words: type and data. Even though the
   data word is nil, the type word is `*MyErr` (non-nil), so
   the *interface* is non-nil. Fix: return `nil` directly, not
   a typed nil pointer.

3. When you've already established the type (e.g. inside a
   type switch case), or when the contract guarantees it and
   you want a panic if the contract is violated. In production
   handlers, prefer comma-ok.

4. Type switch is one expression compiled to a single dispatch;
   chained ifs are N comparisons. Type switch is also more
   readable for ≥3 cases. Performance is similar for small N;
   the compiler can optimize the switch for large N. Use type
   switch when there are multiple branches; use comma-ok for a
   single check.

---

## 20. Hands-On Exercises

**10.1** — Run all three examples; for each, predict the output
before reading it.

**10.2** — Open `exercises/01_safe_assert/main.go`. Replace
every panicky assertion with the comma-ok form; handle the
non-match case gracefully.

**10.3 ★** — Build a `Decoder` that takes `any` (e.g. from
`json.Unmarshal`) and returns a typed config struct, returning
an error if the shape doesn't match.

---

## 21. Mini Project Tasks

**Mini-AST evaluator.** Extend `examples/03_type_switch` with:
unary minus, parentheses, variable references via a
`map[string]float64` environment. The dispatch stays a type
switch.

---

## 22. Chapter Summary

* Conversion (`T(x)`) is compile-time, concrete→concrete.
* Type assertion (`x.(T)`) is runtime, interface→concrete.
* Type switch is multi-way assertion.
* Comma-ok forms make all runtime checks safe.
* Typed-nil-interface is a real gotcha; understand it.
* `go vet` catches `string(int)` mistakes.

---

## 23. Advanced Follow-up Concepts

* `errors.As`, `errors.Is` (Chapter 36).
* `reflect.Kind` and `reflect.Value.Convert` (Chapter 25).
* Generics: when type assertion can be replaced (Chapter 24).
* `runtime/iface.go` — itab implementation in the runtime.
