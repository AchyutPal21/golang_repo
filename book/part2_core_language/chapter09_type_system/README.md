# Chapter 9 — The Type System: Numbers, Strings, Booleans

> **Reading time:** ~24 minutes (6,000 words). **Code:** 3 runnable
> programs (~290 lines). **Target Go version:** 1.22+.
>
> Go's primitive types look familiar — `int`, `float64`, `string`,
> `bool` — until they don't. `int` is platform-dependent. Strings are
> immutable byte sequences, not "characters." Floats have IEEE-754
> traps you must know. UTF-8, runes, and bytes are three things,
> not one. Get this chapter right and you'll never write a string-
> handling bug in Go.

---

## 1. Concept Introduction

A *type* in Go is a compile-time tag that defines (a) what values an
identifier can hold and (b) what operations are valid on it. Go's
type system is strict: you can't add an `int32` to an `int64`
without conversion, can't compare a `string` to a `[]byte` directly,
can't pass a `time.Duration` where an `int64` is expected.

Three families of primitives:

* **Numbers.** Sized integers (`int8`...`int64`), unsigned variants
  (`uint8`...`uint64`), platform-dependent (`int`, `uint`,
  `uintptr`), floats (`float32`, `float64`), complex
  (`complex64`, `complex128`).
* **Booleans.** `true` and `false`. Cannot convert to/from
  integers; no truthy/falsy.
* **Strings.** Immutable sequences of *bytes* (not characters).
  Iteration produces *runes* (Unicode code points). Backed by
  UTF-8 by convention.

> **Working definition:** Go's primitives are deliberately strict —
> every numeric type is distinct, strings are bytes (not chars),
> bools are booleans-and-only-booleans. The strictness eliminates a
> class of bugs that languages with implicit conversion ship.

---

## 2. Why This Exists

Languages that allow implicit conversion (`if (count) { ... }`,
`"5" + 3`, `int + double`) save typing at the cost of bugs. C's
silent integer narrowing has produced thousands of CVEs. JavaScript's
`==` semantics are a punchline. Go's design lesson: every conversion
is explicit, every comparison requires the same type, every numeric
operation requires matching operands. Verbose at the line level;
quietly safer at the program level.

The string design is unusual for a 2009 language: Go strings are
immutable byte sequences, not character arrays. Rob Pike's "Strings,
bytes, runes and characters in Go" essay (2013) is the canonical
treatment; the design assumes UTF-8 throughout, which simplifies
networking and file I/O at the cost of making "what's the 5th
character?" surprisingly subtle.

---

## 3. Problem It Solves

1. **Silent narrowing.** `int32 = int64Val` is a compile error,
   not a wraparound at runtime.
2. **Sign confusion.** `int + uint` is a compile error; you must
   convert one explicitly.
3. **String/byte/char ambiguity.** Three concepts, three types.
   No "is this a 1-byte char or a 4-byte int" question.
4. **Encoding bugs.** Strings being byte-immutable plus the
   UTF-8 convention means most file I/O "just works" without
   encoding choices.
5. **Float/int blending.** `int * float64` doesn't compile. You
   convert.
6. **Boolean misuse.** `if 0` doesn't compile. Either `if x == 0`
   or `if !ok` — no third option.

---

## 4. Historical Context

Go's primitive design follows from C-with-Pascal-discipline:

* **From C:** the named integer sizes (`int8`/16/32/64), the
  unsigned variants, the platform-dependent `int`/`uint`.
* **From Pascal/Modula:** the strict separation of numeric types
  — no implicit promotions, no silent narrowing.
* **From Plan 9 / Unicode:** the UTF-8-as-string convention. Ken
  Thompson and Rob Pike literally co-designed UTF-8 (in 1992, on
  a placemat at a New Jersey diner) and brought it to Go natively.
* **From Java:** the `bool`/`int` separation. Java was the first
  mainstream language to forbid `if (x)` for non-booleans; Go
  inherited it.

`rune` as a name (instead of `char` or `code_point`) is Go's
contribution. It avoids the confusion of "char = 1 byte" baked
into C.

---

## 5. How Industry Uses It

* **APIs declare specific sizes.** `int64` for IDs (across the
  fleet, where 4 billion isn't enough), `int32` for CPU-friendly
  hot loops, `int` for slice indices and counts.
* **Strings are the lingua franca.** HTTP headers, JSON keys,
  log messages, filenames. The UTF-8 assumption is universal.
* **`[]byte` for I/O.** When you read from a file or socket, you
  get `[]byte`; convert to `string` only when you need string
  semantics.
* **`time.Duration` is `int64`.** Underneath. The typed alias
  makes `5 * time.Second` work without surprise.
* **`uintptr` for unsafe-package work only.** Almost no
  application code touches it.

---

## 6. Real-World Production Use Cases

**Database IDs:** every modern Go service uses `int64` for primary
keys. PostgreSQL `bigint`, sequence-generated, fits exactly.

**HTTP request bodies:** read once into `[]byte` with `io.ReadAll`,
then either parse as JSON (no string conversion needed; `json.Unmarshal`
takes `[]byte`) or convert to string for templating.

**Money:** *never* `float64`. Either `int64` cents or a typed
decimal package (`shopspring/decimal`). IEEE-754 errors compound.

**Time calculations:** always `time.Duration`, never `int` of
seconds. The `time` package's typed arithmetic prevents off-by-1000
unit-mismatch bugs.

**Counter/gauge fields:** `uint64` with `sync/atomic`. Negative
counts make no sense and the `uint` constraint communicates it.

---

## 7. Beginner-Friendly Explanation

Three rules that cover 90% of cases:

1. **Use `int` unless you have a reason.** `int` is whatever the
   platform handles efficiently (64-bit on modern hardware). For
   a slice index, a count, a loop variable — just `int`.
2. **Use `string` for text, `[]byte` for I/O.** Convert between
   them when you must; both conversions copy.
3. **Use `float64` for math; use `int64` cents for money.**
   Floats are fine for physics, statistics, scientific work.
   They are catastrophically wrong for currency.

> **Coming From Java —** `int` is platform-dependent in Go (Java's
> always 32-bit). Strings are immutable in both. Go has no
> `Boolean` boxed type — just `bool`. Go has no `BigInteger` in
> the language; use `math/big` if you need it.

> **Coming From Python —** Go integers don't auto-promote to
> bignums. `int * 2` overflows on `MaxInt + 1` rather than
> growing. Python's `str` is closest to Go's `string`, but Go's
> indexing returns *bytes*, not characters.

> **Coming From JavaScript —** Number is `float64` in JS. Go's
> integer types are real, integer-shaped, with deterministic
> overflow.

> **Coming From C/C++ —** No `char` type per se; `byte` is an
> alias for `uint8`, `rune` is an alias for `int32`. No
> `unsigned long long` etc. — just the named sizes.

> **Coming From Rust —** Same story but stricter. Go's `int` ≠
> Rust's `i32`; Go has implicit untyped constant adaptation that
> Rust doesn't.

---

## 8. Deep Technical Explanation

### 8.1. Integers in full

| Type | Bits | Range |
| --- | --- | --- |
| `int8` | 8 | −128 … 127 |
| `int16` | 16 | −32,768 … 32,767 |
| `int32` | 32 | ±2.1 × 10⁹ |
| `int64` | 64 | ±9.2 × 10¹⁸ |
| `uint8`/`byte` | 8 | 0 … 255 |
| `uint16` | 16 | 0 … 65,535 |
| `uint32` | 32 | 0 … 4.3 × 10⁹ |
| `uint64` | 64 | 0 … 1.8 × 10¹⁹ |
| `int`/`uint` | 32 or 64 (platform) | matches `intptr_t` |
| `uintptr` | platform | wide enough for any pointer |

`int` is 32 bits on 32-bit platforms, 64 on 64-bit. In 2026 you
will almost always be on 64-bit. **Never assume `int` is 64
bits** when storing data on disk or wire — use `int64` or `int32`
explicitly.

`byte` is an alias for `uint8`. `rune` is an alias for `int32`.
The aliases improve readability: `[]byte` for raw bytes, `[]rune`
for Unicode code points, `[]uint8` only when you mean "a byte
buffer that's also numeric." Use the alias that conveys intent.

### 8.2. Integer overflow

Go's integers wrap on overflow — *silently*, no exception, no
panic:

```go
var x int32 = math.MaxInt32
x++ // x is now -2147483648 (wrap)
```

For unsigned types:

```go
var y uint8 = 0
y-- // y is now 255 (wrap)
```

This is C-like behavior; the alternative (panic on overflow)
would cost performance on every arithmetic operation. The
discipline: validate inputs at boundaries, use `math/big` for
arbitrary-precision math, use checked arithmetic helpers
(`math/bits`) for explicit overflow detection.

### 8.3. Floating-point

`float64` is IEEE-754 double-precision; `float32` is
single-precision. The traps are universal across languages:

```go
0.1 + 0.2 == 0.3  // false! (0.30000000000000004)
```

Two cardinal rules:

1. **Never compare floats with `==`.** Use a tolerance:
   `math.Abs(a-b) < epsilon`.
2. **Never represent money as `float64`.** Use `int64` cents (or
   `decimal.Decimal` from `shopspring/decimal`).

`math.NaN()`, `math.Inf(+1)`, `math.Inf(-1)` are real values:

* `NaN != NaN` (always — that's the IEEE rule).
* `1.0 / 0.0` is `+Inf`, not a panic. `0.0 / 0.0` is `NaN`.
* Use `math.IsNaN(x)`, `math.IsInf(x, +1)` to test.

### 8.4. Booleans

`bool` has two values: `true`, `false`. No conversions to/from
integers. No truthy/falsy.

* `if x` — only compiles if `x` is `bool`.
* `if x == 0` — for integers.
* `if x != ""` — for strings.
* `if x != nil` — for pointers/slices/maps/channels/interfaces.

The strict typing is intentional: it prevents bugs like JS's
`if ("0")` being truthy.

### 8.5. Strings, deeply

A `string` is an *immutable sequence of bytes*. Internally:

```
type stringHeader struct {
    Data unsafe.Pointer
    Len  int
}
```

Two-word value: pointer to bytes + length. Strings are passed by
value cheaply (16 bytes on 64-bit platforms regardless of
content), and the underlying bytes are *not* copied. Different
strings can share a backing array (and do, for substrings of
literals).

Operations:

* **`len(s)`** — returns byte count.
* **`s[i]`** — returns the i-th *byte* as `byte` (uint8).
* **`s[i:j]`** — returns a substring (no copy of bytes).
* **`s + t`** — concatenation; allocates new backing storage.
* **`for i, r := range s`** — iterates over runes; `r` is `rune`,
  `i` is the byte index of the start of the rune.

Strings *cannot* be mutated:

```go
s := "hello"
s[0] = 'H' // compile error
```

To mutate, convert to `[]byte`, mutate, convert back:

```go
b := []byte(s)
b[0] = 'H'
s = string(b)
```

Both conversions copy.

### 8.6. UTF-8, runes, and the difference

UTF-8 encodes Unicode code points (runes) in 1–4 bytes. ASCII
characters are 1 byte. Most Latin-script extras (é, ñ) are 2
bytes. CJK characters are 3 bytes. Emoji and rare scripts are 4
bytes. Some "characters" you see (a flag, a family emoji) are
*multiple* code points joined.

Implications:

* `len(s)` is *byte* count, not character count.
* `s[0]` is the first *byte*, which may be the start of a
  multi-byte rune.
* `for i, r := range s` gives you (byte index, rune) — the byte
  index jumps by 1, 2, 3, or 4 per iteration depending on rune
  width.

To count "characters":

* Bytes: `len(s)`.
* Runes (code points): `utf8.RuneCountInString(s)` or
  `len([]rune(s))`. The former is faster (no allocation).
* Grapheme clusters (visible "characters"): use
  `golang.org/x/text/unicode/norm` or `rivo/uniseg`. Out of
  scope for this chapter.

### 8.7. Conversions

Implicit numeric conversions don't exist in Go. Every conversion
is explicit:

```go
var i32 int32 = 5
var i64 int64 = int64(i32)   // explicit, required

var f float64 = float64(i32) // explicit, required

s := "5"
n, err := strconv.Atoi(s)    // string→int via strconv
```

The exception: untyped constants adapt to context (Chapter 8).
That's a constant-system feature, not an implicit conversion.

`string ↔ []byte`:

```go
b := []byte("hello") // copy
s := string(b)       // copy
```

`string ↔ []rune`:

```go
r := []rune("héllo") // copy, also UTF-8-decodes
s := string(r)       // copy, encodes to UTF-8
```

All four allocate. For hot paths, prefer working with `[]byte`
throughout to avoid the conversions.

`int → string`: the surprise:

```go
s := string(65) // "A", not "65"!
```

`string(int)` interprets the int as a rune. This is so
error-prone that `go vet` warns. Use `strconv.Itoa(65)` to get
"65".

---

## 9. Internal Working (How Go Handles It)

* **Strings:** two-word headers (pointer + length). Passed by
  value cheaply. Backing bytes live in the rodata section for
  string literals; on the heap for runtime-built strings.
* **Slices of strings/bytes share backing storage** for
  `s[i:j]` substring operations. Useful for parsers — slicing
  doesn't allocate.
* **`int` is `int64` on 64-bit Linux/macOS/Windows.** The Go
  spec says it's at least 32 bits and matches `intptr_t`.
* **`for range string`** uses `runtime.decoderune` to advance
  through UTF-8 byte by byte, returning runes.
* **Float arithmetic** compiles to native FPU/SSE instructions.
  The compiler does not auto-promote `float32` to `float64` in
  expressions; you must convert.

---

## 10. Syntax Breakdown

```go
// Integer literal forms
n := 42         // decimal
n := 0o755      // octal (Go 1.13+)
n := 0x1A       // hex
n := 0b1010     // binary (Go 1.13+)
n := 1_000_000  // digit separators (Go 1.13+)

// Float literal forms
f := 3.14
f := 1e10
f := 1.5e-3
f := 0x1.fp+1   // hex float

// String literal forms
s := "hello\n"      // interpreted: \n is a newline
s := `hello\n`      // raw: backslash-n is two characters

// Rune literal forms
r := 'A'           // rune literal (an int32)
r := 'é'      // 'é' via unicode escape
r := '\xFF'        // 255 via hex byte

// Conversions
n := int32(myInt)
s := string(myBytes)
f := float64(myInt)
```

---

## 11. Multiple Practical Examples

### 1. `examples/01_int_sizes`

Prints the size of every integer type, the platform's `int`
size, and demonstrates overflow/wraparound behavior.

```bash
go run ./examples/01_int_sizes
```

### 2. `examples/02_strings_bytes_runes`

The canonical UTF-8 demonstration: take a multi-byte string,
show `len`, byte indexing, rune iteration, conversions.

```bash
go run ./examples/02_strings_bytes_runes
```

### 3. `examples/03_floats_traps`

The four IEEE-754 traps everyone hits: 0.1+0.2≠0.3, NaN≠NaN,
divide by zero, money-as-float.

```bash
go run ./examples/03_floats_traps
```

---

## 12. Good vs Bad Examples

**Good:** explicit conversions.

```go
n := int64(myInt32) * 1024
```

**Bad:** trying to mix.

```go
n := myInt32 * 1024  // 1024 is untyped; this is int32, may overflow
```

**Good:** byte slice for I/O, string for keys.

```go
data, _ := io.ReadAll(r.Body)
hash := sha256.Sum256(data) // takes []byte
```

**Bad:** unnecessary conversions.

```go
data, _ := io.ReadAll(r.Body)
s := string(data)        // copy
hash := sha256.Sum256([]byte(s)) // copy
```

**Good:** `strconv.Itoa`, not `string(int)`.

```go
s := strconv.Itoa(42) // "42"
```

**Bad:**

```go
s := string(42) // "*", not "42" — int converted to rune
```

---

## 13. Common Mistakes

1. **`string(intVal)` to format a number.** Use `strconv.Itoa` or
   `fmt.Sprint`.
2. **Comparing floats with `==`.** Use a tolerance.
3. **Indexing a string by character.** `s[i]` gives bytes.
   Iterate with `for range` for runes.
4. **Storing money as `float64`.** Use `int64` cents.
5. **Assuming `int` is 64-bit on disk.** It's platform-dependent;
   serialize as `int64`.
6. **Mixing `int32` and `int`.** Compile error; convert one
   explicitly.
7. **`len(s)` to count characters.** That's bytes; use
   `utf8.RuneCountInString` for runes.
8. **`s[0] = 'X'`.** Strings are immutable; convert to `[]byte`
   first.
9. **Treating `byte` and `rune` as interchangeable.** Different
   sizes (`uint8` vs `int32`); different uses.
10. **Forgetting that `string` is byte-immutable** — `[]byte(s)`
    is a copy, not a view.

---

## 14. Debugging Tips

* **`fmt.Printf("%T %v\n", x, x)`** — prints type + value.
* **`fmt.Printf("%q\n", s)`** — prints a string with escapes
  visible.
* **`fmt.Printf("%x\n", b)`** — prints a byte slice as hex.
* **`go vet`** catches `string(int)` mistakes.
* **`utf8.ValidString(s)`** checks if a string is valid UTF-8.

---

## 15. Performance Considerations

* **String concatenation** with `+` allocates each step. For
  building strings in a loop, use `strings.Builder` (Chapter 38)
  or `bytes.Buffer`.
* **`string ↔ []byte` conversion** is O(n); always copies.
  Avoid in hot loops.
* **`utf8.RuneCount(b)`** is faster than `len([]rune(string(b)))`
  because it doesn't allocate a slice.
* **Float arithmetic** is one CPU cycle per op on modern hardware.
  Don't avoid it unless profiling tells you to.
* **Untyped constants compile to immediates.** No runtime cost.

---

## 16. Security Considerations

* **Untrusted strings can be invalid UTF-8.** `utf8.ValidString`
  before processing if it matters.
* **Float comparisons in security checks** are a bug source.
  Use integer comparisons.
* **Integer overflow** is silent. Validate input ranges before
  arithmetic with attacker-controlled values.
* **Timing-safe string comparison** uses `crypto/subtle.ConstantTimeCompare`,
  not `==`. (`==` is fast but variable-time.)

---

## 17. Senior Engineer Best Practices

1. **Default to `int`** for indices, counts, loop vars.
2. **Use specific sizes** (`int32`, `int64`) for wire/disk
   formats.
3. **Use `byte` for byte buffers**, `rune` for code points,
   `int8`/`uint8` only for numeric byte values.
4. **Never `float64` for money.**
5. **Convert at boundaries**, not in the middle of logic.
6. **Use `strconv` for number ↔ string**, not `string(int)`.
7. **Iterate strings with `range`** when you need runes.
8. **`utf8.ValidString` untrusted input** before processing.
9. **`strings.Builder` in loops**, not `+=`.
10. **Don't mix string and `[]byte`** unnecessarily; pick one
    per pipeline.

---

## 18. Interview Questions

1. *(junior)* What's the difference between `byte` and `rune`?
2. *(junior)* What does `len(s)` return for a string?
3. *(mid)* Why is `string(65)` `"A"` and not `"65"`?
4. *(mid)* How do you count characters in a UTF-8 string?
5. *(senior)* Why never use `float64` for money?
6. *(senior)* What's the internal layout of a Go string?
7. *(senior)* Walk through what `for i, r := range s` does
   internally.

## 19. Interview Answers

1. `byte` is an alias for `uint8` (1 byte). `rune` is an alias
   for `int32` (a Unicode code point, 1–4 bytes when UTF-8
   encoded). Use `byte` for byte buffers, `rune` for code points.

2. The number of bytes, not characters. For ASCII they're equal;
   for UTF-8 with multi-byte runes the byte count is larger.

3. `string(intVal)` interprets the int as a Unicode code point.
   65 is U+0041, which is "A". To format a number as its string
   representation, use `strconv.Itoa(65)` → `"65"`. `go vet`
   flags the common mistake.

4. `utf8.RuneCountInString(s)` for runes (Unicode code points).
   Note: a "user-visible character" can be multiple runes (an
   emoji with skin tone is 2 runes). For grapheme clusters use
   `rivo/uniseg`.

5. IEEE-754 floats are binary, not decimal. `0.1` cannot be
   represented exactly; arithmetic compounds the error. Storing
   $1.10 + $2.20 might give $3.3000000000000003. Use `int64`
   cents (or `shopspring/decimal`) for currency.

6. Two words: pointer to bytes + length. The bytes are immutable
   and may be shared (literals live in rodata; substrings share
   backing storage with their source). Passing a string is two
   word-copies; always cheap.

7. The runtime walks the byte-encoded UTF-8 with
   `runtime.decoderune`, which decodes one rune at a time. `i`
   is the byte index of the start of the current rune; `r` is
   the rune value. Each iteration advances `i` by 1–4 bytes
   depending on the rune's UTF-8 width.

---

## 20. Hands-On Exercises

**9.1 — Integer-size tour.** Run
[`examples/01_int_sizes`](examples/01_int_sizes/main.go) and
predict the wraparound output before reading it.

**9.2 — UTF-8 iteration.** Open
[`exercises/01_utf8_iterate/main.go`](exercises/01_utf8_iterate/main.go).
Add a function that takes a string and returns the rune at
position `n` (counting runes, not bytes).

**9.3 ★ — Money type.** Define `type Cents int64` with `Add`,
`Sub`, `Format` methods. Test against `0.1 + 0.2 = 0.3`-style
cases that would fail with `float64`.

---

## 21. Mini Project Tasks

**Task — A `String()` audit.** Walk a Go file (your project, or
the book repo). For every `string(x)` conversion where `x` is an
integer type, decide: was the author trying to format a number?
If yes, change to `strconv.Itoa`. This is the kind of cleanup
new Go shops do once.

---

## 22. Chapter Summary

* Numeric types are distinct; conversions are explicit.
* `int` is platform-dependent. Use `int64`/`int32` for
  serialized data.
* Booleans are booleans; no truthy/falsy.
* Strings are immutable byte sequences with UTF-8 conventions.
* `byte` = `uint8`; `rune` = `int32`. Use the alias that
  conveys intent.
* `len(s)` is bytes; `utf8.RuneCountInString` is runes.
* Never `string(int)` to format a number; use `strconv.Itoa`.
* Never `float64` for money.

---

## 23. Advanced Follow-up Concepts

* Rob Pike, "Strings, bytes, runes and characters in Go" (2013).
* IEEE-754 — the float standard. Read the "subnormals" section.
* `shopspring/decimal` — production decimal library.
* `golang.org/x/text/unicode/norm` — Unicode normalization.
* `crypto/subtle` — constant-time string ops.
