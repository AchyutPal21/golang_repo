# Chapter 12 — Control Flow: `if`, `for`, `switch`, `goto`

> **Reading time:** ~22 minutes (5,500 words). **Code:** 3 runnable
> programs (~270 lines). **Target Go version:** 1.22+.
>
> Go has only one looping keyword (`for`), one conditional
> (`if`), one multi-way branch (`switch`), and a real `goto`.
> The combinations are powerful — `switch` has features C's
> doesn't, `for` covers what `while` and `do/while` cover
> elsewhere, labelled break/continue replace several escape
> patterns. By the end of this chapter you'll have a complete
> mental map of Go control structures.

---

## 1. Concept Introduction

Control flow is *what runs next*. Go's control structures:

* **`if`** — conditional, with an optional initializer clause.
* **`for`** — loops. Counter form, condition-only form,
  infinite form, range form.
* **`switch`** — multi-way branch. Expression switch and type
  switch (Chapter 10).
* **`goto`** — unconditional jump within a function. Real, but
  rarely used.
* **`break`, `continue`, `fallthrough`, `return`** — escape
  statements.
* **Labels** — names that `break`/`continue`/`goto` can target.

> **Working definition:** Go's control flow is deliberately
> minimal. One loop keyword, one conditional, one multi-way
> branch, plus a `goto` for the rare cases. Combined with
> labelled `break`/`continue`, it covers everything other
> languages need three or four keywords for.

---

## 2. Why This Exists

Other languages give you `while`, `do/while`, `for-each`,
`for-in`, `for-of`. Go gives you `for` and tells you to use
clauses. The team's argument: a single form with multiple
clauses is easier to read than four near-identical keywords.

Similarly, Go's `switch` doesn't need explicit `break` because
cases don't fall through by default — the C/Java mistake of
"forgot to break" doesn't exist. When you *want* fallthrough,
the `fallthrough` keyword makes it explicit.

`goto` exists because Niklaus Wirth was right (in *Modula-2*):
sometimes a forward jump out of a deep loop is cleaner than the
alternative. Go just refuses to let you jump *over* a variable
declaration or *into* a block.

---

## 3. Problem It Solves

1. **`while`/`do-while`/`for` keyword sprawl.** One `for`
   covers all.
2. **Switch fall-through bugs.** Go's switch doesn't fall
   through by default; you opt in with `fallthrough`.
3. **Tagless switch as if-else chain.** `switch { case x > 10:
   ...; case x > 5: ... }` is cleaner than `if/else if`.
4. **Multi-level break.** Labels let `break outer` escape from
   nested loops cleanly — no flag variables.
5. **Initializer scope.** `if x := f(); x != nil { ... }` keeps
   the variable scoped tightly.
6. **Range iteration.** `for i, v := range slice` is the only
   form you need for slices, maps, channels, strings, and (since
   1.22) integers.

---

## 4. Historical Context

Go's control-flow design is largely C with three changes:

* **No parentheses around conditions.** `if x > 0 { ... }`,
  not `if (x > 0) { ... }`. Reduces visual noise.
* **Mandatory braces.** `if x > 0 { do() }` — single-line bodies
  must still be braced. Eliminates the `goto fail` class of bugs
  (Apple's 2014 SSL/TLS bug came from a mis-indented unbraced
  `if`).
* **Switch doesn't fall through.** Major change from C/Java.

`for range` over an integer (`for i := range 10`) was added in
Go 1.22. Before that, you'd write `for i := 0; i < 10; i++`.

`for` loop variable scoping was *changed* in Go 1.22. Before
1.22, the variable in `for i := 0; ...; i++` was shared across
iterations — a closure capturing `i` saw the final value, not
the value at iteration time. In 1.22 the variable is per-
iteration. This was a long-discussed change; it broke a lot of
people's mental model and fixed a lot of subtle bugs.

---

## 5. How Industry Uses It

* **`if err != nil { ... }`** — the most common control statement
  in Go. Every line of error-checking is one of these.
* **`for { select { ... } }`** — the standard event-loop shape
  for goroutines.
* **`for _, v := range slice`** — the canonical iteration.
* **`switch` over `errors.Is` checks** — tagless switch as a
  cleaner alternative to chained if-elifs.
* **`break label`** — escape out of nested goroutine selectors.
* **`goto retry`** — occasionally seen in retry loops where
  break/continue would be awkward.

---

## 6. Real-World Production Use Cases

**Retry-with-backoff:**

```go
for attempt := 0; attempt < maxRetries; attempt++ {
    err := do()
    if err == nil {
        return nil
    }
    if !isRetryable(err) {
        return err
    }
    time.Sleep(backoff(attempt))
}
return errExceededRetries
```

**Event loop (channel select):**

```go
for {
    select {
    case msg := <-incoming:
        process(msg)
    case <-ticker.C:
        flush()
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

**HTTP method dispatch (tagless switch):**

```go
switch r.Method {
case http.MethodGet:
    handleGet(w, r)
case http.MethodPost, http.MethodPut:
    handleUpsert(w, r)
case http.MethodDelete:
    handleDelete(w, r)
default:
    http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
```

**Range over channel for shutdown:**

```go
for msg := range ch {  // exits when ch is closed
    process(msg)
}
```

---

## 7. Beginner-Friendly Explanation

**`if`:**
```go
if x > 0 {
    // do thing
} else if x < 0 {
    // other thing
} else {
    // catch-all
}
```

Initializer clause:
```go
if v, err := f(); err != nil {
    return err
} else {
    // v in scope here
}
```

**`for` — three forms plus range:**
```go
for i := 0; i < 10; i++ { ... }   // C-style
for x < 100 { ... }               // condition-only (replaces `while`)
for { ... }                       // infinite
for i, v := range slice { ... }   // range over slice/array/string/map/chan
for i := range 10 { ... }         // range over int (Go 1.22+)
```

**`switch`:**
```go
switch x {
case 1: ...
case 2, 3: ...    // multi-value case
default: ...
}

switch {           // tagless — like a chain of if-elifs
case x > 100: ...
case x > 10:  ...
default:      ...
}
```

**`break`, `continue`, `return`** end loops/functions.
**`fallthrough`** explicitly continues to the next switch case.

---

## 8. Deep Technical Explanation

### 8.1. `if` with initializer

```go
if x, err := f(); err != nil {
    handle(err)
} else if x > 100 {
    big()
}
// x and err are out of scope here
```

The initializer clause introduces variables scoped to the
`if`/`else if`/`else` block tree. Idiomatic for "compute, check,
use, discard."

### 8.2. The four `for` forms

**1. Counter.** `for init; cond; post { body }`. Init and post
are statements; cond is a bool expression.

**2. Condition-only.** `for cond { body }`. Replaces `while`.

**3. Infinite.** `for { body }`. Standard event-loop shape.

**4. Range.** `for k, v := range x { body }`. Works on:

* **Arrays/slices** — `k` is index (int), `v` is element.
* **Strings** — `k` is byte index (int), `v` is rune.
* **Maps** — `k` is key, `v` is value. Iteration order is
  *randomized* — explicitly, by design.
* **Channels** — only one variable; receives until close.
* **Integers (1.22+)** — `for i := range 10` iterates 0..9.

You can omit either variable. `for range slice` runs N times
without binding anything.

### 8.3. The Go 1.22 loop-variable change

Before 1.22:

```go
for i := 0; i < 3; i++ {
    go func() { fmt.Println(i) }()
}
// Output: 3 3 3 (all goroutines see the final i)
```

After 1.22:

```go
for i := 0; i < 3; i++ {
    go func() { fmt.Println(i) }()
}
// Output: 0 1 2 (in some order; each goroutine sees its own i)
```

The change: each iteration now creates a fresh `i`. The fix
predates 1.22 and was to `i := i` shadow inside the loop body or
pass `i` as a function argument. Both still work; in 1.22+ they
just aren't necessary.

This change is *only* for `for` loops, not for `if`/`switch`.

### 8.4. `switch` features

Multi-value case:
```go
switch ch {
case 'a', 'e', 'i', 'o', 'u': vowel()
}
```

Tagless switch (chain of conditions):
```go
switch {
case x > 100: big()
case x > 10:  med()
default:      small()
}
```

Initializer:
```go
switch x := compute(); {
case x > 100: big()
}
```

Explicit fallthrough:
```go
switch x {
case 1:
    fmt.Println("one")
    fallthrough
case 2:
    fmt.Println("also runs for 1 and 2")
}
```

`fallthrough` must be the last statement in the case.

### 8.5. `break`, `continue`, and labels

`break` exits the innermost `for`/`switch`/`select`. `continue`
jumps to the next iteration of the innermost `for`.

Labelled forms target an outer construct:

```go
outer:
for _, row := range matrix {
    for _, v := range row {
        if v < 0 {
            break outer  // exit BOTH loops
        }
    }
}
```

A label is a Go-level construct; the compiler enforces that the
label refers to a real enclosing statement.

### 8.6. `goto`

```go
retry:
    err := tryOnce()
    if isRetryable(err) {
        time.Sleep(backoff)
        goto retry
    }
```

Restrictions: `goto` cannot jump *over* a variable declaration
or *into* a block. The compiler enforces both.

`goto` is rare; `for`/`break`/`continue` plus a small wrapping
function usually achieves the same shape with clearer intent.

### 8.7. `return`

`return` exits the enclosing function. With named returns,
`return` alone returns the current values of the named return
variables — the "naked return" idiom. We'll cover named returns
in Chapter 13.

---

## 9. Internal Working (How Go Handles It)

* **`if`** compiles to a conditional jump.
* **`for`** compiles to a label, a conditional jump at the top,
  the body, and an unconditional jump back to the label.
* **`switch`** compiles to either a chain of compares (small N)
  or a jump table (large N, dense values). The compiler decides.
* **Range over a slice** is sugar for a counter `for` plus
  indexing. Range over a map uses `runtime.mapiterinit` /
  `mapiternext`. Range over a channel uses `<-ch` until
  `chanrecv` reports the channel is closed.
* **The Go 1.22 loop variable** is implemented by allocating a
  fresh slot per iteration, with closures capturing the new slot.

---

## 10. Syntax Breakdown

```go
if cond { ... } else if cond { ... } else { ... }
if init; cond { ... }

for init; cond; post { ... }
for cond { ... }
for { ... }
for k, v := range coll { ... }

switch v {
case a, b: ...
default: ...
}
switch { case cond: ... }
switch init; v { ... }

break          break label
continue       continue label
fallthrough
return v...

goto label
label:
```

---

## 11. Multiple Practical Examples

### `examples/01_for_forms`

All four `for` forms plus `range` over slice/string/map/channel/
integer.

### `examples/02_switch_powers`

Tagless switch, multi-value case, `fallthrough`, switch-with-
initializer.

### `examples/03_labels_goto`

Labelled break/continue, plus a legitimate `goto retry` case.

---

## 12. Good vs Bad Examples

**Good — initializer in `if`:**

```go
if v, err := f(); err != nil {
    return err
} else {
    use(v)
}
```

**Bad — wider scope than necessary:**

```go
v, err := f()
if err != nil {
    return err
}
use(v)
```

(Both are valid; the first scopes `v` and `err` tightly. Pick
based on whether `v` is used after the `if`.)

**Good — tagless switch:**

```go
switch {
case n < 10:  small()
case n < 100: med()
default:      big()
}
```

**Bad — chained if/else:**

```go
if n < 10 {
    small()
} else if n < 100 {
    med()
} else {
    big()
}
```

(Both compile to the same code; the switch is more readable.)

**Good — labelled break:**

```go
outer:
for _, row := range matrix {
    for _, v := range row {
        if isMatch(v) {
            break outer
        }
    }
}
```

**Bad — flag variable:**

```go
found := false
for _, row := range matrix {
    for _, v := range row {
        if isMatch(v) {
            found = true
            break
        }
    }
    if found {
        break
    }
}
```

---

## 13. Common Mistakes

1. **Pre-1.22 loop-variable capture.** A goroutine inside `for i
   := range slice { go f(i) }` saw the final `i` on every
   invocation. Fixed in 1.22.
2. **Forgetting `break` in C-style switch.** Not a Go problem
   — Go doesn't fall through by default.
3. **Using `fallthrough` to mean "or."** Don't. Multi-value
   case `case a, b:` is what you want.
4. **`continue` inside `select`.** `continue` targets `for`, not
   `select`. Inside `for { select { ... } }`, `continue` works as
   expected.
5. **`break` inside `select` outside a `for`.** Exits the
   `select` only — not the goroutine. Wrap in a `for` with a
   label.
6. **Mutating a map during `range`.** Adds may or may not be
   seen by the iteration; deletes of unvisited keys are safe.
   The spec is intentionally vague.
7. **Iterating a `nil` map** is fine — zero iterations. No
   panic.
8. **Iterating a `nil` slice** is fine — zero iterations.
9. **Forgetting `range` randomizes map order.** It does, on
   purpose. If you need ordered iteration, sort the keys.
10. **Heavy work in the `for` post statement.** It runs every
    iteration; pull invariant code out.

---

## 14. Debugging Tips

* `go vet` catches some unreachable code after `return` or
  `panic`.
* `gopls` warns on shadowed variables in initializer clauses.
* Add `-vet=all` to CI to catch unused break labels.
* `go test -race` catches the pre-1.22 loop-variable issue if
  you've hit it.

---

## 15. Performance Considerations

* `for range` on slices is the same speed as `for i := 0; i < len(s); i++`
  (the compiler unifies them).
* `for range` on maps is unavoidably slower per-element than on
  slices because of the hash-table walk.
* Tagless switch with N cases is O(N) compares for sparse
  values, O(1) jump table for dense.
* `switch x := compute(); ...` only computes `x` once.

---

## 16. Security Considerations

* **Map iteration randomization** prevents a class of
  hash-flooding DoS attacks; don't rely on iteration order.
* **Bounded retry loops.** Always cap the iteration count
  (`for attempt := 0; attempt < max; attempt++`); never
  `for { ... continue }` based on attacker-controlled input.

---

## 17. Senior Engineer Best Practices

1. **Use `if init; cond` to scope the variable tightly.**
2. **Prefer tagless switch over if-else chains.**
3. **Use range over index where possible.**
4. **Don't use `goto` for control flow** that `for`/`break`
   handles.
5. **Use labelled break for nested-loop escape.**
6. **`for range int` (1.22+) over `for i := 0; i < N; i++`.**
7. **Sort keys before iterating a map** when order matters.
8. **In a `for { select { ... } }`, name the loop with a label**
   if you need to break from inside a case.

---

## 18. Interview Questions

1. *(junior)* How do you write a `while` loop in Go?
2. *(junior)* What does `for range 5` do (Go 1.22+)?
3. *(mid)* Why doesn't Go's `switch` need `break`?
4. *(senior)* Explain the Go 1.22 loop-variable change and
   what it fixed.
5. *(senior)* When would you use `goto`?

## 19. Interview Answers

1. `for cond { ... }`. Go has one loop keyword.

2. Iterates `i` from 0 to 4 (inclusive lower, exclusive upper).
   `for range 5 { ... }` runs the body 5 times without binding
   `i`.

3. By design — cases don't fall through. Use `fallthrough`
   explicitly when you want C-like behavior. The change
   eliminates the "forgot to break" bug class.

4. Pre-1.22, the variable in `for i := 0; ...; i++` was shared
   across iterations. Closures captured the same `i`, so
   goroutines spawned in the loop all saw the final value. In
   1.22, each iteration gets a fresh `i`, matching most
   developers' mental model. The fix shipped after years of
   debate; it broke a small amount of code that depended on
   the shared semantics.

5. Rarely. The legitimate cases are deeply-nested retry loops
   or state-machine-like control flow where break/continue
   would require flag variables or extra wrapping functions.
   Most Go code never uses it.

---

## 20. Hands-On Exercises

**12.1** — Implement FizzBuzz with a single `switch`.
**12.2** — Find the first negative number in a 2D matrix using
labelled break.
**12.3 ★** — Write a retry loop that uses `goto retry` and the
equivalent that uses `for` plus `continue`. Compare.

---

## 21. Mini Project Tasks

**Mini event loop.** Build a goroutine that reads from a
channel, periodically flushes, and exits on context cancel —
all in `for { select { ... } }`. Bonus: handle a "drain on
shutdown" case where you process pending messages before exiting.

---

## 22. Chapter Summary

* One loop keyword (`for`), four forms.
* `if`/`switch`/`for` accept initializer clauses.
* `switch` doesn't fall through by default.
* Tagless switch replaces if-else chains.
* `break label` and `continue label` for nested-loop control.
* Go 1.22 changed loop-variable scoping: per-iteration now.
* `goto` exists but is rarely the right answer.

---

## 23. Advanced Follow-up Concepts

* The Go 1.22 release notes — read the loop-var section.
* `golang.org/issue/60078` — the loop-var proposal.
* `select` statement — covered in Chapter 44.
* `fallthrough` performance vs separate cases.
