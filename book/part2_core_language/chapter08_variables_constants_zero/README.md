# Chapter 8 — Variables, Constants, and the Zero Value

> **Reading time:** ~28 minutes (7,000 words). **Code:** 3 runnable
> programs (~310 lines). **Target Go version:** 1.22+.
>
> Three concepts every Go program rests on: how variables come into
> existence, how constants stay fixed at compile time, and what value
> a freshly-declared variable holds before you write to it. The third
> point — the zero value — is one of Go's quiet power moves; it
> removes a class of bugs that other languages call "uninitialized"
> and ship in production.

---

## 1. Concept Introduction

A *variable* in Go is a named storage location of a specific type.
You declare it with `var` (or the short form `:=`) and the compiler
allocates space for it. Until you write to it, it holds the *zero
value* of its type.

A *constant* in Go is a named value fixed at compile time. The
compiler substitutes constants where they're used; they never have
runtime storage. Constants come in two flavors — typed and untyped —
and the untyped form is the surprise that makes Go's type system
feel both strict and ergonomic.

The *zero value* is the default value a variable holds the instant
it's declared without an initializer: `0` for numbers, `""` for
strings, `false` for booleans, `nil` for pointers, slices, maps,
channels, function values, and interface values. Every Go type has
a useful zero value, and idiomatic Go *relies* on this.

> **Working definition:** Go's variable model is "every name has a
> type, every type has a useful zero value, and every constant is
> either typed (locked) or untyped (adaptive)." Internalize the
> three rules and almost every confusing-at-first-glance Go
> declaration suddenly makes sense.

---

## 2. Why This Exists

Most languages let you declare a variable without initializing it:
in C, an `int x;` is whatever was on the stack at that address.
Read it before writing it, and you get *undefined behavior* — your
program might work most of the time and crash mysteriously once a
year. Java's solution was to require initialization for local
variables (compile error) but auto-initialize fields to zeroes (a
runtime guarantee). Go's solution is simpler and more uniform:
*every variable, everywhere, starts at its type's zero value*. No
exceptions. No special cases for fields vs. locals.

Constants exist because you need a way to name values that the
compiler can substitute everywhere they're used, without runtime
storage cost and without the risk of mutation. Go's untyped-constant
mechanism additionally lets the same constant adapt to whatever
numeric type the call site needs, which is a quiet but huge
ergonomic win for code that mixes `int32`, `int64`, `float64`, and
`time.Duration`.

---

## 3. Problem It Solves

1. **Uninitialized-memory bugs.** Zero values mean reading a
   variable before assignment is well-defined; you can never
   dereference garbage.
2. **Boilerplate constructors.** A `var s Server` is already a
   useful, zero-valued `Server`. You don't need a `NewServer()` to
   "initialize" the simple case.
3. **Type-conversion ceremony.** Untyped constants adapt to context.
   `const Pi = 3.14` works in a `float32`, `float64`, or `complex128`
   context without explicit conversion.
4. **Magic numbers.** `iota` gives you typed-enum-like value
   sequences without naming each value or maintaining ordinals.
5. **Configuration-vs-data confusion.** `const MaxRetries = 3`
   communicates "this is a knob, not a runtime value" — and the
   compiler enforces it.
6. **Refactoring safety.** Renaming a variable across the package is
   safe because the compiler refuses anything ambiguous; no shadowing
   silently kicks in.

---

## 4. Historical Context

Go's variable and constant model was designed in 2007–2009 by
Griesemer/Pike/Thompson with explicit reference to:

* **Pascal and Modula** — the "type after the name" declaration
  syntax (`x int` rather than `int x`), and the strict "no implicit
  numeric conversions" rule. Niklaus Wirth's languages taught the
  team that disciplined typing scales.
* **C** — the `var` keyword's absence (Go uses `var` because C's
  declaration grammar is famously confusing — `int *x[5]` versus
  `(int *)x[5]`).
* **JavaScript and Python** — the *short variable declaration* `:=`
  was added to give Go some of the local ergonomics of dynamic
  languages without sacrificing static types.
* **Modula-2** — the untyped-constant mechanism. Go's `const Pi =
  3.14` is closer to Modula-2's compile-time value-with-kind than
  to C's `#define` macro or Java's `static final` field.

A few historical points worth knowing:

* **`:=` was nearly cut.** Pike has said publicly that `:=` was
  controversial in early design discussions; some team members felt
  it added a second declaration form unnecessarily. It survived
  because the alternative (always writing `var x =`) felt heavy
  for local code.
* **`iota` is from APL, via Pascal.** The name is borrowed from
  APL's `ι` (iota) function, which generates an integer sequence.
  Pascal had it; Go inherited the name.
* **Zero values are non-negotiable.** Multiple Go FAQ entries push
  back on requests for "uninitialized" or "lazily initialized"
  variables. The team views zero values as a foundation other
  features (composite literals, generic instantiation, GC
  semantics) rest on.

---

## 5. How Industry Uses It

Idiomatic patterns you'll see in every production Go codebase:

* **Zero-value-ready types.** A `bytes.Buffer{}` works without any
  setup. A `sync.Mutex{}` is unlocked. A `sync.WaitGroup{}` has zero
  pending. Library authors design types so this is true.
* **Short declaration in functions, `var` at package level.** Inside
  a function, almost everything is `x := f()`. At package level,
  almost everything is `var x = f()` or `var x T`.
* **Typed constants for exported "enums."** A package exports
  `type Status int` plus `const ( StatusOK Status = iota;
  StatusFailed; ... )`. The type carries meaning across function
  boundaries.
* **`const` for derived limits.** Buffer sizes, retry counts, magic
  numbers all live in `const` blocks at the top of files. Reviewers
  scan the consts first to understand the configuration surface.
* **`_` for dropped values.** `if _, ok := m[k]; ok { ... }`
  is everywhere. The blank identifier is a real part of the
  variable-system surface.
* **Multiple-assignment swap.** `a, b = b, a` is the idiomatic swap;
  no temporary needed. Used heavily in slice manipulation and sort
  routines.

---

## 6. Real-World Production Use Cases

**Zero-value HTTP server.** Standard `net/http`:

```go
var srv http.Server
srv.Addr = ":8080"
srv.Handler = mux
srv.ListenAndServe()
```

The zero `http.Server{}` is perfectly usable; you set only the
fields you care about. This is the design ethos: types should be
useful at zero.

**Typed-status enum.** A payments service:

```go
type PaymentStatus int

const (
    PaymentPending PaymentStatus = iota
    PaymentAuthorized
    PaymentCaptured
    PaymentFailed
    PaymentRefunded
)
```

Five values, ordered, type-distinct. A function that takes
`PaymentStatus` cannot accidentally receive an `OrderStatus`; the
compiler enforces it. The `iota`-driven sequence keeps the values
contiguous, so you can range over them or use them as array indices.

**Untyped-constant ergonomics.** A config struct:

```go
const DefaultTimeout = 30 * time.Second  // untyped int * Duration → Duration
```

`30` is an untyped integer; `time.Second` is a `time.Duration`.
Multiplication yields `time.Duration`. Without untyped constants,
you'd have to write `time.Duration(30) * time.Second`, which is
strictly equivalent but visibly noisier.

**Build-time configuration via const + iota.** A feature-flag
package:

```go
type Feature int

const (
    FeatureNone Feature = 0
    FeatureBasicSearch Feature = 1 << iota
    FeatureAdvancedSearch
    FeatureBetaUI
)

const AllFeatures = FeatureBasicSearch | FeatureAdvancedSearch | FeatureBetaUI
```

Bit-flag enums. `iota` plus the `<<` operator gives you a clean
pattern for "set of independent options as one int."

**Zero-value-safe sync primitives.** A worker:

```go
type Worker struct {
    once sync.Once    // zero value is "not yet done"
    mu   sync.Mutex   // zero value is unlocked
    done chan struct{} // zero value is nil — must initialize
}
```

`sync.Once` and `sync.Mutex` are usable at zero. Channels are not
— a `nil` channel blocks forever on send/receive. The Worker must
initialize `done` in a constructor or `init()` method. This
distinction is one a reviewer will catch.

---

## 7. Beginner-Friendly Explanation

Variables in Go are simple if you remember three things:

1. **You write the type after the name** — `var x int`, not
   `int x`. (Pascal-style. Don't fight it.)
2. **`:=` declares and assigns in one step**, only inside functions.
   `x := 7` is shorthand for `var x int = 7` (with the type
   inferred from `7`).
3. **A bare `var` declaration assigns the zero value.** `var x int`
   gives you `x = 0`. `var s string` gives you `s = ""`. `var p *int`
   gives you `p = nil`.

Constants are simpler still: `const X = 7` says "X is the value 7
forever; the compiler will substitute it everywhere."

The deep idea: there is no "uninitialized" state in Go. Every
variable is *always* either at its zero value or at a value you
gave it. There's no "undefined" trap to fall into.

> **Coming From Java —** `var x int` is roughly Java's `int x`,
> except you don't get the "definite assignment" compile error —
> you get the zero value. `:=` is roughly `var x = ...` in newer
> Java. Constants are `static final` but typed differently
> (Go's untyped constants have no Java equivalent).

> **Coming From Python —** `:=` is the closest Go gets to Python's
> `x = 7`. Type inference means you usually don't write the type;
> the compiler figures it out. The big difference: types are
> compile-time, not duck-typed at runtime.

> **Coming From JavaScript —** `var` and `let` map to Go's `var`.
> `const` exists in both. The zero-value rule is the surprise —
> reading a Go variable before assigning is well-defined; reading
> a JS variable before assigning is `undefined` (or a TDZ error).

> **Coming From C++ —** Go's variable model is *much* simpler.
> No declaration-vs-definition split, no extern, no `static`
> storage class. Every variable has one declaration in one place,
> initialized or zero, and that's it.

> **Coming From Rust —** No `let mut` vs `let`; all Go variables
> are mutable. No exhaustive initialization required (zero value
> handles it). No type inference subtleties — Go's inference is
> strictly local.

---

## 8. Deep Technical Explanation

### 8.1. Variable declaration forms

Go has *four* ways to introduce a variable. Knowing when each is
idiomatic is the first thing a senior Go engineer drills into
juniors.

```go
// Form 1: var with type and initializer
var x int = 42

// Form 2: var with type, no initializer (zero value)
var y int

// Form 3: var with initializer, type inferred
var z = 42

// Form 4: short declaration (functions only)
w := 42
```

**Form 1** is rarely idiomatic — it states the type and the value,
which is redundant since the value already implies the type. Use it
only when the inference would be wrong (e.g. `var x int64 = 42`
when you specifically need `int64` but the literal is untyped).

**Form 2** is the zero-value declaration. Use when you want to
declare a variable and assign to it later: `var err error` before
a series of operations that may set it.

**Form 3** is rare; just use Form 4 inside a function. Form 3 is
useful at package scope where `:=` is not allowed.

**Form 4** is the workhorse inside functions. Most local variables
look like this.

### 8.2. The grouped `var` declaration

You can declare multiple variables in one block:

```go
var (
    serverAddr = ":8080"
    timeout    = 30 * time.Second
    maxConns   = 100
)
```

This is the idiomatic form at package scope when you have several
related variables. The grouped form is also the only way to
declare *multiple* package-level variables with `var` while
keeping `gofmt`-aligned columns.

### 8.3. Multiple assignment

Go supports multi-value left-hand sides:

```go
a, b := 1, 2
x, y := y, x   // swap, no temp variable

f, err := os.Open(name)   // (T, error) idiom — everywhere

count, _ := f.Read(buf)   // _ discards the second return
```

The `_` *blank identifier* is a real syntactic feature: it's a
named target that always discards. Use it to ignore a return value
the function forced you to acknowledge.

### 8.4. The zero value, in full

Every type in Go has a zero value:

| Type | Zero value |
| --- | --- |
| `bool` | `false` |
| `int`, `int8`...`int64`, `uint*`, `uintptr` | `0` |
| `float32`, `float64` | `0.0` |
| `complex64`, `complex128` | `(0+0i)` |
| `string` | `""` (empty, length 0) |
| `*T` (pointer) | `nil` |
| `[]T` (slice) | `nil` (length 0, capacity 0) |
| `map[K]V` | `nil` (read-only; reads return the value's zero, writes panic) |
| `chan T` | `nil` (blocks forever on send/receive) |
| `interface{}` (`any`) | `nil` (no type, no value) |
| `func(...)...` | `nil` |
| `[N]T` (array) | array with each element zero-valued |
| `struct { ... }` | struct with each field zero-valued |

A few subtleties worth memorizing:

* `nil` slice and empty slice (`[]T{}`) are *different but
  interchangeable for almost all purposes*. `len(nil) == 0`,
  ranging over `nil` works, `append(nil, x)` returns a new slice.
* `nil` map is read-only-friendly, write-hostile. `m["k"] = "v"`
  on a nil map panics.
* `nil` channel blocks both send and receive forever. This is a
  legitimate idiom for `select` cases (you can disable a case by
  setting its channel to `nil`).
* Interfaces have a *two-word internal layout* (type word + data
  word). A `nil` interface has both words zero. An interface that
  holds a typed nil pointer has a non-nil type word and a nil data
  word — and is *not equal to* `nil`. We'll cover this in Chapter
  22; for now, know that "interface == nil" is a more subtle check
  than it looks.

### 8.5. Variable scope

Go scoping is lexical and unsurprising:

* **Package-scope** for `var` outside any function: visible
  throughout the package.
* **File-scope** for `import` declarations.
* **Function-scope** for parameters, named returns, and locals
  declared with `var` or `:=` at the top of `func`.
* **Block-scope** for declarations inside `{}` (including `if`,
  `for`, `switch` initializer clauses).

```go
func f() {
    x := 1
    if y := 2; y > 0 {
        // x and y both visible here
        z := 3
        // z visible here
    }
    // x visible; y and z out of scope
}
```

The `if x := f(); x > 0 { ... }` pattern is idiomatic for
"compute, check, use, discard." It scopes `x` tightly to the `if`
branches.

### 8.6. Variable shadowing

Inside a nested block, `:=` *creates a new variable* of the same
name, shadowing the outer one. This is a frequent source of bugs:

```go
err := f()
if err != nil {
    err := g()  // shadow! changes only the inner err.
    log.Println(err)
}
// outer err is still whatever f() returned; g()'s err was lost.
```

`go vet` catches the most dangerous cases. `golangci-lint` with
`-E shadow` catches more. The rule: inside a block, use `=` (assign)
when you mean to update an outer variable; use `:=` (declare) only
when you mean a new one.

### 8.7. Constants, deeply

A constant is fixed at compile time. The compiler substitutes it
where it's used; there's no runtime storage.

```go
const Pi = 3.14159
const Greeting = "hello"
const MaxConns = 100
```

Constants can be:

* **Untyped** (`const X = 7`) — has a *kind* (integer, floating,
  string, boolean, complex, rune) but no specific type. Adapts to
  context.
* **Typed** (`const X int32 = 7`) — locked to that type. Cannot be
  used in a context that expects `int64` without explicit
  conversion.

You can do constant arithmetic:

```go
const KB = 1024
const MB = KB * 1024  // computed at compile time
```

You *cannot* call non-builtin functions in a constant expression:

```go
const X = math.Sqrt(2)   // error: function call not allowed
const Y = len("hello")   // OK: len is a builtin and "hello" is a const
```

Why constants matter beyond "fixed values": untyped constants are
the mechanism by which `time.Duration(30) * time.Second` looks
like just `30 * time.Second`. The `30` adapts.

### 8.8. The `iota` mechanism

`iota` is a special pre-declared identifier inside `const`
blocks. It starts at 0 and increments by 1 for each `const`
specification (line) in the block.

```go
const (
    A = iota   // 0
    B          // 1 (implicit "= iota")
    C          // 2
    D          // 3
)
```

Once you've declared a const without an explicit `=` clause, it
inherits the previous one's expression. So:

```go
const (
    KB = 1 << (10 * (iota + 1))  // 1 << 10 = 1024
    MB                           // 1 << 20
    GB                           // 1 << 30
    TB                           // 1 << 40
)
```

`iota` resets to 0 in each new `const` block. Inside a single
block, it increments per *line*, not per identifier — so
`A, B = iota, iota` on one line is `A=0, B=0`, then the next line
gets `iota=1`.

The skip-with-`_` trick is common:

```go
const (
    Sunday = iota  // 0
    Monday         // 1
    Tuesday        // 2
    _              // 3 — skipped
    Thursday       // 4
)
```

`iota` plus a typed constant gives you a typed enum:

```go
type Weekday int

const (
    Sunday Weekday = iota
    Monday
    Tuesday
    Wednesday
    Thursday
    Friday
    Saturday
)
```

Now `Weekday` is a distinct type; functions that accept a
`Weekday` cannot accidentally take an `int`.

### 8.9. The `_` blank identifier

`_` is a write-only universal target. Use cases:

* **Discard a return value:** `_, err := f()`.
* **Discard a range key:** `for _, v := range slice { ... }`.
* **Suppress unused-import errors during dev:** `import _ "package"`
  imports for side effects only.
* **Type assertions where you don't need the value:**
  `_, ok := iface.(*Concrete)`.

`_` is not a variable; you can't read from it. `x = _` is a
compile error.

---

## 9. Internal Working (How Go Handles It)

* **Variable storage.** Local variables typically live on the
  goroutine's stack unless escape analysis decides they outlive
  the function (in which case they're heap-allocated). Package-
  level variables live in the data segment (initialized) or BSS
  (zero-valued). The compiler's `-gcflags=-m` reports escape
  decisions.
* **Zero initialization.** The runtime zero-initializes memory
  on allocation. For stack variables, the compiler emits zeroing
  instructions on function entry. For heap allocations, the
  allocator zero-fills before returning the block. There is no
  performance cost beyond the zeroing itself; modern CPUs do this
  with rep-stos or wider vector instructions.
* **Constant resolution.** The compiler evaluates constant
  expressions during type-checking, before code generation. The
  resulting values live in the compiler's constant table; they
  may be inlined into the binary as immediates (e.g. as part of
  an `MOV` instruction's operand) or stored in the rodata section
  if too large.
* **`iota` tracking.** During parsing of a `const` block, the
  parser maintains a per-block `iota` counter. Each `ConstSpec`
  AST node carries its own `iota` value at the time it was parsed.
* **Untyped-constant types.** The compiler tracks an internal
  "kind" (`UntypedInt`, `UntypedFloat`, etc.) and resolves to a
  concrete type at the point of use. `cmd/compile/internal/types2`
  is where this lives if you want to read the source.

---

## 10. Syntax Breakdown

```go
// Single var with type
var x int

// Single var with initializer (type inferred)
var x = 42

// Single var with both
var x int = 42

// Grouped vars
var (
    a int
    b string = "hello"
    c = 3.14
)

// Multiple in one var
var a, b, c = 1, "two", 3.0

// Short declaration (in function only)
x := 42
a, b := 1, 2

// Mixed declare/assign with :=
// (at least one variable on the left must be new)
existing := 1
existing, fresh := 2, 3   // existing assigned, fresh declared

// Constant — always typed at compile time
const Pi = 3.14159
const TypedPi float32 = 3.14
const (
    A = iota  // 0
    B          // 1
    C          // 2
)

// Blank identifier
_, err := f()
```

---

## 11. Multiple Practical Examples

### Example 1 — `examples/01_declarations`

```bash
go run ./examples/01_declarations
```

Demonstrates all four declaration forms side by side, plus the
multiple-assignment idiom, plus shadowing — and the `vet` warning
shadowing produces. Read the comments; run it; then deliberately
introduce a shadowing bug and see it caught.

### Example 2 — `examples/02_zero_values`

```bash
go run ./examples/02_zero_values
```

Prints the zero value of every built-in type, plus a struct, an
array, a slice, a map, a channel, a function, and an interface.
Use it as a lookup when you forget what `var x map[string]int`
gives you.

### Example 3 — `examples/03_iota_patterns`

```bash
go run ./examples/03_iota_patterns
```

Three real-world `iota` patterns: a typed-enum (Weekday), a
bit-flag set (file permissions), and a unit ladder (KB/MB/GB).
Each is a copy-pasteable idiom.

---

## 12. Good vs Bad Examples

**Good — leans on zero values:**

```go
type Server struct {
    Addr     string  // zero "" → use default ":8080"
    Handler  http.Handler  // zero nil → use http.DefaultServeMux
    Timeout  time.Duration // zero 0 → no timeout
}

var s Server
s.ListenAndServe()  // works without any setup
```

**Bad — fights zero values:**

```go
type Server struct {
    Addr     string
    Handler  http.Handler
    Timeout  time.Duration
}

func NewServer() *Server {
    return &Server{
        Addr:    ":8080",
        Handler: http.DefaultServeMux,
        Timeout: 30 * time.Second,
    }
}

s := NewServer()
```

The bad form forces every caller through `NewServer`. If they
forget, they don't get sensible defaults — they crash on a nil
handler. Good Go API design uses the zero value as the default
and lets callers override.

**Good — typed enum with iota:**

```go
type Status int

const (
    StatusUnknown Status = iota
    StatusOK
    StatusFailed
)

func (s Status) String() string {
    return [...]string{"unknown", "ok", "failed"}[s]
}
```

**Bad — string-typed "enum":**

```go
const (
    StatusUnknown = "unknown"
    StatusOK      = "ok"
    StatusFailed  = "failed"
)

func process(status string) { ... }  // typo-prone!
```

The bad form lets callers pass `"okay"` or `"OK"` and the compiler
won't catch it. The good form makes wrong values impossible at
compile time.

**Good — short declaration:**

```go
func handle(r *http.Request) error {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        return fmt.Errorf("read body: %w", err)
    }
    // ...
}
```

**Bad — verbose `var` in function:**

```go
func handle(r *http.Request) error {
    var body []byte
    var err error
    body, err = io.ReadAll(r.Body)
    if err != nil {
        return fmt.Errorf("read body: %w", err)
    }
    // ...
}
```

Same effect, twice the lines.

---

## 13. Common Mistakes

1. **Shadowing in `if`/`for`:** `if err := f(); err != nil { return err }`
   is fine; `if err := f(); err != nil { err := g(); log(err) }`
   silently shadows. `go vet` catches it.
2. **Writing to a `nil` map:** `var m map[string]int; m["k"] = 1`
   panics. Always `m := make(map[string]int)` or
   `m := map[string]int{}`.
3. **Comparing a `nil` interface that holds a typed-nil pointer:**
   surprise — the interface is *not* nil. We cover this in Chapter
   22.
4. **Using a typed constant where untyped would adapt:**
   `const Sec int32 = 1; time.Duration(Sec) * time.Second` is needed;
   `const Sec = 1; Sec * time.Second` works directly.
5. **Forgetting `iota` resets per block:** two separate `const ()`
   blocks each start `iota` at 0.
6. **Treating `iota` as a variable:** it isn't. You can't take its
   address; you can't print it; it only exists inside `const`
   blocks.
7. **Mass-assigning unrelated variables on one line:**
   `a, b, c, d, e := 1, "two", 3.0, []int{4}, &s{}` is technically
   legal, illegible.
8. **Using `var x = nil`:** doesn't compile — the compiler can't
   infer a type from `nil`. Write `var x *T` or `var x []T`.
9. **Using `:=` at package level:** doesn't compile. `:=` only
   works inside functions.
10. **Re-declaring on `:=` when you meant to assign:** if at least
    one variable on the LHS is new, `:=` declares the new ones and
    assigns the old ones. If *all* are existing, `:=` is a compile
    error and you must use `=`.

---

## 14. Debugging Tips

* **`go vet ./...`** catches most shadowing, format-string, and
  unused-variable mistakes. Run it on every save.
* **Add `-shadow` to `golangci-lint`** for stricter shadowing
  detection (off by default; intentional).
* **`go build`** refuses unused locals and unused imports. This is
  not a warning; it's an error. Fix it; don't bypass.
* **`fmt.Printf("%T\n", x)`** prints the dynamic type of `x`.
  Useful when you're not sure if you have an `int` or `int32`.
* **`fmt.Printf("%#v\n", x)`** prints a Go-syntax representation —
  great for inspecting struct values during debugging.
* **`reflect.ValueOf(x).Kind()`** when you need to decide branches
  on the kind of a variable (we'll see in Chapter 25).

---

## 15. Performance Considerations

* **Zero-init has near-zero cost.** Modern CPUs zero memory with
  vector instructions; the cost dominates only for very large
  allocations.
* **Constants compile to immediates.** `const X = 100` plus `n :=
  X * 2` produces the same machine code as `n := 200`. There's no
  runtime constant lookup.
* **Untyped constants don't allocate.** `30 * time.Second` is a
  compile-time computation. The result is a typed constant; no
  allocation, no math at runtime.
* **`iota` is a compile-time mechanism.** The generated code has
  no notion of "iota"; the compiler bakes the resulting values in.
* **Package-level `var` initialization runs once at program start.**
  If the initializer is expensive (e.g. parsing a big embed), you
  pay that cost on cold start. Keep init logic small or move to
  lazy `sync.Once`.

---

## 16. Security Considerations

* **Zero-value sensitive fields are zero, not "uninitialized."**
  A `var apiKey string` is `""`, not random bytes. Better than C,
  but you still need to *populate* it before use.
* **Constants are visible in the binary.** A `const APIKey =
  "..."` is reverse-engineerable from the binary's rodata section.
  Don't bake secrets into source; use env vars or secret managers
  (Chapter 40 covers this).
* **The blank identifier silently discards.** `_, err := f()` then
  forgetting to check `err` is a class of bug. Lints catch it.
  `errcheck` in `golangci-lint` is the relevant linter.
* **`var` package-level mutable state is a footgun.** Many Go
  vulnerabilities involve shared mutable globals. Prefer constants
  or carefully-encapsulated state.

---

## 17. Senior Engineer Best Practices

1. **Use `:=` inside functions; `var` at package level.** Don't
   mix.
2. **Lean on the zero value.** Design types so `var x T` is
   useful.
3. **Use `var` for "I'll assign in a moment"** — `var err error`
   above a series of operations.
4. **Group related package-level vars and consts.** `var ( ... )`
   blocks make the file's configuration surface scannable.
5. **Use typed enums with `iota`** for any "set of values" that
   crosses package boundaries.
6. **Write `String()` methods for typed-enum types.** Otherwise
   `fmt.Println(StatusOK)` prints `1` instead of `ok`.
7. **Avoid one-letter variable names** except for narrow scopes
   (loop indices, receiver names). `i`, `j`, `k`, `n`, `r`, `w`,
   `b`, `s` are fine; `aa`, `xy`, `tmp` are not.
8. **Reach for `const` aggressively.** Magic numbers in production
   code should be named.
9. **Don't shadow.** When `go vet` warns, fix it.
10. **`_` is for things you genuinely don't need.** Don't use it
    to silence `errcheck`; handle the error.

---

## 18. Interview Questions

1. *(junior)* What is the zero value of `string` in Go?
2. *(junior)* What's the difference between `var x int` and
   `x := 0`?
3. *(mid)* Explain the difference between `[]T(nil)` and `[]T{}`.
4. *(mid)* What does `iota` do?
5. *(mid)* What's an "untyped constant"?
6. *(senior)* Why does Go forbid unused local variables?
7. *(senior)* What happens when you write to a nil map? A nil
   slice? A nil channel?
8. *(senior)* Why does Go's variable model use `:=` at all
   instead of just `var`?
9. *(staff)* Explain how variable shadowing can cause silent
   bugs, and how `go vet` catches them.

---

## 19. Interview Answers

1. The empty string `""`.

2. `var x int` declares `x` and assigns the zero value (`0`).
   `x := 0` declares `x` and assigns the literal `0`. They produce
   the same result, but `var x int` is the only form available
   at package scope; `x := 0` only works inside a function.

3. `[]T(nil)` is a nil slice — pointer is nil, length 0, cap 0.
   `[]T{}` is an empty slice — pointer is non-nil, length 0, cap 0.
   For most purposes (range, len, append) they behave identically.
   The difference matters when comparing to `nil`
   (`s == nil` distinguishes them) and when JSON-marshaling
   (a nil slice marshals to `null`; an empty slice marshals to
   `[]`).

4. `iota` is a special pre-declared identifier inside `const`
   blocks. It starts at 0 and increments by 1 per *line*, used to
   generate value sequences without naming each. Resets per
   `const` block. Idiomatic for typed enums.

5. A constant declared without an explicit type
   (`const X = 7`) has a *kind* (integer, floating, string, etc.)
   but no specific type. It adapts to the context where it's
   used: `var x int32 = X` works; `var y float64 = X` works.
   Untyped constants give Go the ergonomics of dynamic typing
   without the runtime cost.

6. Because unused locals are usually a sign of a bug — code that
   was meant to use the value but doesn't, or a leftover from a
   refactor. Forbidding them at compile time is a small ceremony
   that catches a class of dead code. The same rule for unused
   imports keeps imports lean and signals package surface clearly.

7. **Nil map** writes panic. Reads return the value's zero value
   (no panic). **Nil slice** writes also panic (slices are
   read-only-friendly when nil; you need `append` or `make` to
   grow them). **Nil channel** sends and receives block forever
   — a legitimate idiom in `select` cases to disable a branch.

8. `:=` keeps Go's local-variable ergonomics close to a dynamic
   language without sacrificing static typing. Without `:=`,
   every local would need `var x = ...`, which is a meaningful
   amount of extra code in long functions. Pike has said the
   form was almost cut; it stayed because the alternative felt
   too heavy.

9. Inside a nested block, `:=` creates a *new* variable of the
   same name, hiding the outer one. Common mistake:
   `if err := f(); err != nil { err = g() ... }` — wait, that's
   `=` so it's safe. Versus
   `if true { err := g(); ... }` — that's `:=`, which silently
   shadows. The bug: you `log.Println(err)` thinking it's the
   outer `err`, but it's the inner one. `go vet -shadow` catches
   the most common patterns; `golangci-lint` extends the
   detection. Fix: use `=` to update existing variables; use
   `:=` only when you mean a new one.

---

## 20. Hands-On Exercises

**Exercise 8.1 — Zero-value tour.** Open
`examples/02_zero_values/main.go`, run it, and predict each
output before reading the printed value. Mismatches are your
mental-model gaps.

**Exercise 8.2 — Convert a `map[string]int` to use `iota`.** Pick
a string-keyed map you've used in your work for an "enum" — like
`map[string]int{"low": 1, "med": 2, "high": 3}`. Rewrite as a
typed-enum with `iota` and a `String()` method.

**Exercise 8.3 ★ — Find a shadowing bug.** Run
`go run ./exercises/01_safe_defaults`. The program has a
deliberate shadowing bug. Read the comments; identify it; fix it;
verify with `go vet -all`.

---

## 21. Mini Project Tasks

**Task — A `Status` package with iota, String(), and JSON marshaling.**

Create a new package `internal/status` with:

* A `type Status int`.
* An `iota`-driven `const ()` block of statuses.
* A `String() string` method.
* Implementations of `json.Marshaler` and `json.Unmarshaler` so
  the value round-trips as a string in JSON.
* Tests that confirm round-trip encoding.

This is a real production pattern; building it once teaches you
the wiring.

---

## 22. Chapter Summary

* Variables come in four declaration forms; `:=` is the
  function-local workhorse, `var` is everything else.
* Every type has a useful zero value. Idiomatic Go relies on
  this.
* Constants are compile-time fixed; untyped constants adapt to
  context.
* `iota` generates value sequences in `const` blocks; it's the
  Go-idiomatic enum mechanism.
* Multiple assignment, the blank identifier, and grouped `var`/
  `const` blocks are first-class language features, not curiosities.
* Shadowing is real, common, and best caught with `go vet`.

Updated working definition: *Go's variable model is "every name
has a type, every type has a useful zero value, and every constant
is either typed (locked) or untyped (adaptive)." Internalize the
three rules and almost every Go declaration becomes obvious.*

---

## 23. Advanced Follow-up Concepts

* **Russ Cox, "The Go Memory Model"** (2014, updated 2022) — the
  formal contract that says when one goroutine's writes are
  visible to another's reads. Visible-after-zero-init is a
  consequence.
* **The Go Spec, sections "Variables" and "Constants"** — the
  authoritative reference. Short and worth a read.
* **`golang.org/issue/377`** — the proposal thread that led to
  `iota`. Read it for the design rationale.
* **`go.dev/blog/constants`** — Rob Pike on Go's constants. The
  best long-form treatment of untyped constants on the web.
* **`go vet` documentation** — the full list of checks, including
  `-shadow`.
