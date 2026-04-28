# Chapter 11 — Operators, Precedence, and Bitwise Tricks

> **Reading time:** ~20 minutes (5,000 words). **Code:** 3 runnable
> programs (~280 lines). **Target Go version:** 1.22+.
>
> Go has fewer operators than most languages and only five
> precedence levels. The full table fits on one screen. The
> interesting parts are bitwise — including `&^` (bit clear), Go's
> hidden gem — and the rules around what can be combined with what.

---

## 1. Concept Introduction

An *operator* combines values to produce a new value. Go's
operators fall into five families:

* **Arithmetic** — `+ - * / % ++ --`.
* **Comparison** — `== != < <= > >=`.
* **Logical** — `&& || !`.
* **Bitwise** — `& | ^ << >> &^`.
* **Address-of / dereference** — `& *` (pointer operations).

Plus assignment compounds (`+=`, `*=`, `<<=`, `&^=`, ...) and the
channel operators `<-` (send/receive). Five precedence levels
total — vastly fewer than C++ or Java.

> **Working definition:** Go's operator surface is small,
> strictly typed (operands must match types), and obeys five
> precedence levels. Memorize the levels once, then use
> parentheses generously when in doubt — `gofmt` won't strip them.

---

## 2. Why This Exists

Operators are unavoidable. The choice is how many of them to
have, what types they accept, and what their precedence is. Go's
choices:

* **Fewer operators.** No `++` for floats. No `?:` ternary. No
  comma operator. Every operator does one thing on one set of
  types.
* **Five precedence levels.** C has 15. Java has roughly 14.
  Five is small enough that you can hold it in your head.
* **Strict typing.** `int + int64` is a compile error. There's
  no implicit promotion ladder to memorize.
* **Bitwise on integers only.** `&`, `|`, `^`, `<<`, `>>`, `&^`
  apply only to integer types. Floats and strings are not
  bit-shifted.

---

## 3. Problem It Solves

1. **Operator-precedence bugs.** A small precedence table makes
   surprises rare.
2. **Implicit-conversion bugs.** Strict typing eliminates the
   "did this promote to float and lose precision?" class.
3. **Bit-flag readability.** `&^` (bit clear) is a one-character
   alternative to `& ~mask` that doesn't even compile in many
   languages.
4. **Increment-as-statement.** `i++` and `i--` are *statements*,
   not expressions. You can't write `j = i++`. Eliminates a
   classic bug class.

---

## 4. Historical Context

Most of Go's operator design is C, minus the parts the team
viewed as bug magnets:

* `++` and `--` exist but are *statements*, not expressions
  (Java's `j = i++` is illegal in Go).
* No `?:` ternary. The team felt it was overused for short
  expressions and obscured intent. Use `if`/`else`.
* No comma operator (`a, b` as expression). The team kept the
  syntax for *multiple assignment* but rejected the comma-
  expression form C uses in `for` loop conditions.
* `<<` and `>>` follow C's semantics (arithmetic shift on
  signed types, logical on unsigned).
* `&^` (bit clear) was inherited from B and Plan 9 C. Most
  modern languages dropped it; Go kept it.

---

## 5. How Industry Uses It

* **Bit flags everywhere.** Permissions, feature flags, network
  protocols. The combination `iota` + `<<` (Chapter 8) plus
  `|` to set, `&^` to clear, `&` to test is universal.
* **Modular arithmetic.** `idx % len(buf)` for ring buffers;
  `idx & (len(buf)-1)` for power-of-two ring buffers (one
  instruction faster).
* **Shift for fast multiply/divide by powers of two.** Used in
  hash table sizing, allocator size classes, GC mark bits.
* **Comparison chains via `&&`/`||`.** Short-circuit semantics
  guarantee left-to-right evaluation; production code relies on
  this.
* **`!=` for change detection.** `if newState != oldState { emit() }`.

---

## 6. Real-World Production Use Cases

**HTTP method matching.**

```go
if r.Method == http.MethodGet || r.Method == http.MethodHead { ... }
```

`||` short-circuits; the second comparison runs only if the
first is false.

**Permission masks.**

```go
perms := PermRead | PermWrite           // set both
perms = perms &^ PermWrite              // clear write
canRead := perms&PermRead == PermRead   // test
```

**Power-of-two ring buffer.**

```go
const N = 1024 // power of 2
var buf [N]event
i := 0
push := func(e event) {
    buf[i&(N-1)] = e // & is a single AND instruction; % is divide
    i++
}
```

**Atomic counter.** `sync/atomic.AddInt64(&count, 1)` uses CAS;
Chapter 46 covers this.

---

## 7. Beginner-Friendly Explanation

If you've used C, Java, or Python, you already know most of Go's
operators. The few surprises:

* **`++` and `--` are statements.** `i++` on its own line, not
  inside an expression.
* **No `?:` ternary.** Use `if`/`else` or define a helper.
* **`&^` is "bit clear."** `a &^ b` is the same as `a & (^b)` —
  zero out the bits of `a` that are set in `b`.
* **`^` is XOR for binary** but *also* unary bitwise NOT.
  `^x` flips every bit of `x`. `a ^ b` is XOR.
* **Strict typing.** `int + int32` doesn't compile. Convert one
  side.

Five precedence levels, highest to lowest:

1. `*  /  %  <<  >>  &  &^`
2. `+  -  |  ^`
3. `==  !=  <  <=  >  >=`
4. `&&`
5. `||`

When in doubt, parenthesize. `gofmt` keeps your parentheses.

> **Coming From C/C++ —** No comma operator, no `++`/`--` as
> expression, fewer precedence levels. `&^` is the welcome
> addition.

> **Coming From Python —** No `**` for power; use `math.Pow` or
> bit shifts for powers of two. No `//` for floor division;
> `int / int` already truncates. No `and`/`or`; use `&&`/`||`.

> **Coming From Java —** Same operators, fewer of them. Java's
> `>>>` (unsigned right shift) doesn't exist; use uint type and
> `>>`. No `instanceof`; use type assertion (Chapter 10).

---

## 8. Deep Technical Explanation

### 8.1. Arithmetic operators

`+ - * /` work on numeric types (integers, floats, complex). `%`
works only on integers and produces the remainder (sign matches
the dividend).

```go
-7 % 3   // -1 (sign matches -7, the dividend)
7 % -3   // 1
```

`+` also concatenates strings. `"hello, " + name`. No other
arithmetic operator works on strings.

`/` on integers truncates toward zero:

```go
7 / 2    // 3
-7 / 2   // -3 (not -4)
```

`++` and `--` are postfix-only and are *statements*, not
expressions:

```go
i++                  // OK
j := i++             // compile error
for i := 0; i < n; i++  // OK
```

### 8.2. Comparison operators

`== != < <= > >=` produce `bool`. Both operands must be of the
same type (or one must be an untyped constant compatible with
the other).

* **Integers, floats, strings, booleans, channels, pointers** —
  fully comparable.
* **Structs** — comparable if all fields are comparable. `==` is
  field-by-field.
* **Arrays** — comparable if their element type is.
* **Interfaces** — comparable; equal if both type word and data
  word match.
* **Slices, maps, functions** — *not* comparable except to `nil`.
  Comparison is a compile error.

The not-comparable types are the surprise. To compare two slices
deeply, use `slices.Equal` (Go 1.21+) or write a loop.

### 8.3. Logical operators

`&& || !` operate on `bool`. Short-circuit:

* `a && b` — `b` evaluates only if `a` is `true`.
* `a || b` — `b` evaluates only if `a` is `false`.

This matters when `b` has side effects:

```go
if user != nil && user.IsAdmin() { ... } // safe — IsAdmin only called if user != nil
```

`!` is unary, applies to `bool`.

### 8.4. Bitwise operators

Apply to integer types only.

| Op | Meaning |
| --- | --- |
| `&` | AND |
| `|` | OR |
| `^` | XOR (binary), bitwise NOT (unary) |
| `<<` | left shift |
| `>>` | right shift |
| `&^` | bit clear (AND NOT) |

`&^` is Go's hidden gem:

```go
flags &^= PermWrite  // clear the PermWrite bit, leave others alone
```

Equivalent to `flags = flags & ^PermWrite`. The dedicated
operator is more readable.

Shifts:

* `x << n` — multiply by 2ⁿ (signed and unsigned).
* `x >> n` — for unsigned, divide by 2ⁿ (logical shift, fills
  with 0). For signed, arithmetic shift (fills with sign bit).
* The shift amount must be unsigned or untyped constant.
  `int(x) << uint(y)` is the older idiom; since Go 1.13 you can
  use `int(x) << y` if `y` is non-negative.

### 8.5. Address-of and dereference

`&x` takes the address of `x`, producing a pointer.
`*p` dereferences the pointer `p`. We'll cover these properly in
Chapter 16.

```go
x := 42
p := &x  // p is *int
v := *p  // v is int, copy of x
*p = 7   // mutate x through the pointer
```

### 8.6. Channel operators

`ch <- v` sends. `<-ch` receives. `<-` is also part of the
channel type to indicate direction (`<-chan T` for receive-only,
`chan<- T` for send-only). Covered in Chapter 43.

### 8.7. Compound assignment

Almost every binary operator has a compound form:

```go
x += 1
x *= 2
x &^= mask
x <<= 3
```

Pure syntactic sugar. `x += y` compiles to the same code as
`x = x + y`.

### 8.8. The full precedence table

```
Precedence 5 (highest):  *  /  %  <<  >>  &  &^
Precedence 4:            +  -  |  ^
Precedence 3:            ==  !=  <  <=  >  >=
Precedence 2:            &&
Precedence 1 (lowest):   ||
```

Unary operators (`! - + ^ * &`) bind tighter than any binary
operator. The five binary levels are all that matter.

Left-associative for all binary operators. So `a - b - c` is
`(a - b) - c`.

---

## 9. Internal Working (How Go Handles It)

* The compiler emits a single machine instruction for most
  operators. `&`, `|`, `^`, `<<`, `>>` are CPU primitives.
* `&^` compiles to AND-NOT, two instructions on most ISAs (one
  instruction on ARM with the BIC encoding).
* `&&` and `||` are *not* lazily evaluated by emitting `if`
  branches; the compiler uses conditional jumps to skip the RHS
  when the LHS short-circuits.
* `++` and `--` compile to INC/DEC plus a memory store; no
  expression result, hence statement-only.
* String `+` allocates new backing storage; in a loop, use
  `strings.Builder`.

---

## 10. Syntax Breakdown

```go
// Arithmetic
a + b    a - b    a * b    a / b    a % b
+a       -a       // unary
i++      i--

// Comparison
a == b   a != b   a < b   a <= b   a > b   a >= b

// Logical
a && b   a || b   !a

// Bitwise
a & b    a | b    a ^ b    a &^ b   ^a
a << n   a >> n

// Assignment
a = b
a += b   a -= b   a *= b   a /= b   a %= b
a &= b   a |= b   a ^= b   a &^= b
a <<= n  a >>= n

// Address & dereference
p := &x
v := *p

// Channels (Chapter 43)
ch <- v
v := <-ch
v, ok := <-ch
```

---

## 11. Multiple Practical Examples

### `examples/01_operator_table`

Runs every operator on representative values; prints inputs,
operator, and result. Great for "is this what I think it is?"

```bash
go run ./examples/01_operator_table
```

### `examples/02_bitwise_patterns`

Set/clear/test/toggle bits in a permission mask; power-of-two
modulo trick; XOR swap.

```bash
go run ./examples/02_bitwise_patterns
```

### `examples/03_precedence_traps`

Three or four expressions where precedence is non-obvious.
Predict, then run.

```bash
go run ./examples/03_precedence_traps
```

---

## 12. Good vs Bad Examples

**Good — short-circuit safety:**

```go
if user != nil && user.IsAdmin() { ... }
```

**Bad — separate statements that don't short-circuit:**

```go
ok1 := user != nil
ok2 := user.IsAdmin() // panics if user is nil
if ok1 && ok2 { ... }
```

**Good — `&^` for clearing:**

```go
flags &^= PermWrite
```

**Bad — verbose:**

```go
flags = flags & ^PermWrite
```

(Both compile to the same code, but the dedicated operator
communicates intent.)

**Good — parens when in doubt:**

```go
if (a&mask) == expected { ... }
```

**Bad — relying on memorized precedence:**

```go
if a&mask == expected { ... }  // works, but reader has to think
```

The precedence makes the bad form correct, but parentheses are
free and aid readers.

---

## 13. Common Mistakes

1. **Trying `j = i++`.** Doesn't compile.
2. **Comparing slices with `==`.** Doesn't compile (except to
   `nil`). Use `slices.Equal`.
3. **Comparing maps with `==`.** Doesn't compile. Use a loop or
   `maps.Equal`.
4. **Comparing functions with `==`.** Doesn't compile (except to
   `nil`).
5. **Mixing `int` and `int32` in a bitwise expression.** Compile
   error.
6. **Shifting by a negative amount.** Runtime panic.
7. **Confusing `^` (XOR) with `**` (power).** Go has no power
   operator; use `math.Pow`.
8. **Forgetting that `%` follows the dividend's sign.** `-7 % 3`
   is `-1`, not `2`.
9. **Floating-point modulo.** `%` on floats is a compile error;
   use `math.Mod`.
10. **String `+` in a tight loop.** Quadratic. Use
    `strings.Builder`.

---

## 14. Debugging Tips

* `go vet` flags surprising operator usage (e.g. self-comparison
  via `&&`/`||`).
* Parenthesize then re-read; if the meaning changes, the
  parentheses were necessary.
* `go tool compile -S file.go` prints the assembly; useful when
  you wonder "did the compiler emit a single instruction here?"

---

## 15. Performance Considerations

* Most operators are one CPU instruction.
* `%` is divide; for power-of-two divisors, `&` is faster.
* `*` and `/` on 64-bit integers may be slower than `<<` and
  `>>` for power-of-two multipliers.
* String `+` allocates; `strings.Builder` reuses a buffer.
* Short-circuit avoids redundant work; structure conditions
  with the cheapest test first.

---

## 16. Security Considerations

* **Timing attacks via short-circuit.** Comparison-on-secret
  must be constant-time. Use `crypto/subtle.ConstantTimeCompare`
  for password/token equality.
* **Integer overflow** in shift expressions can produce
  unexpected values when an attacker controls the shift amount;
  validate.

---

## 17. Senior Engineer Best Practices

1. **Parenthesize around `&`, `|`, `^`, `<<`, `>>`** when mixed
   with other ops; the precedence is unintuitive even for
   experts.
2. **`&^` for clearing bits** is more readable than `& ^mask`.
3. **`x & (N-1)` for modulo by power of 2** in hot loops.
4. **Use `strings.Builder` for string concatenation in loops.**
5. **Use `slices.Equal` / `maps.Equal`** for deep equality.
6. **Use `crypto/subtle.ConstantTimeCompare`** for secrets.
7. **Don't fight the type system.** `int + int32` won't compile;
   convert.

---

## 18. Interview Questions

1. *(junior)* What does `&^` do?
2. *(junior)* Can you write `j = i++` in Go?
3. *(mid)* Why doesn't `[]int == []int` compile?
4. *(senior)* What's the difference between `>>` on a signed
   vs unsigned integer?
5. *(senior)* Walk me through the precedence levels.

## 19. Interview Answers

1. Bit clear (AND NOT). `a &^ b` zeroes the bits of `a` that
   are set in `b`. Equivalent to `a & ^b`. Useful for clearing
   bits in a flag mask.

2. No. `++` and `--` are statements in Go, not expressions.
   You write `i++` on its own line; you can't use the result.

3. Slices are not comparable except to `nil`. The reason: slice
   equality is potentially O(n), and Go wants `==` to be
   constant-time semantically. Use `slices.Equal` (Go 1.21+)
   for deep comparison.

4. **Unsigned `>>`** is a logical shift — fills with zeros from
   the left. **Signed `>>`** is arithmetic — fills with the
   sign bit, preserving the sign of negative numbers. Java
   exposes both as `>>>` (logical) and `>>` (arithmetic); Go
   picks based on the operand type.

5. Five levels: (5, highest) `*  /  %  <<  >>  &  &^`; (4)
   `+  -  |  ^`; (3) `==  !=  <  <=  >  >=`; (2) `&&`; (1,
   lowest) `||`. Unary operators bind tighter than any binary.

---

## 20. Hands-On Exercises

**11.1** — Open
[`exercises/01_flag_set/main.go`](exercises/01_flag_set/main.go)
and complete the `Flags` type with `Set`, `Clear`, `Has`,
`Toggle` methods.

**11.2** — Compute the lowest power of 2 ≥ N. Implement two
ways: with `<<` and with `math.Pow`. Compare performance via
`go test -bench`.

**11.3 ★** — Implement constant-time string equality without
using `crypto/subtle`. Then compare to `subtle.ConstantTimeCompare`
and explain the difference.

---

## 21. Mini Project Tasks

**Task — "What ops compile?" lint.** Write a tool that scans a
`.go` file and reports any `==` or `!=` between slice/map/func
types (which won't compile, but the tool can warn about
near-misses like `slices.Equal` candidates). Use `go/parser`
and `go/types` (out of scope for now; revisit after Chapter 25).

---

## 22. Chapter Summary

* Five precedence levels; unary tightest.
* Strict typing — operands must match.
* Bitwise operators on integers only; `&^` is Go's hidden gem.
* `++` and `--` are statements.
* No ternary, no comma operator, no power.
* Slices/maps/funcs not comparable except to `nil`.

---

## 23. Advanced Follow-up Concepts

* The Go Spec, "Operators."
* `crypto/subtle` for constant-time ops.
* `math/bits` for portable integer manipulation (count trailing
  zeros, leading zeros, popcount).
* `slices.Equal` and `maps.Equal` for deep equality.
