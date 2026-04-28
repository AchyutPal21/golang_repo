# Chapter 8 — Revision Checkpoint

## Questions

1. What's the zero value of `*int`, `[]string`, `map[K]V`, and
   `chan int`?
2. When does `m["key"] = "val"` panic?
3. What's the difference between an *untyped* constant and a
   *typed* constant?
4. What does `iota` reset on?
5. When can you use `:=` and when must you use `var`?
6. What does `_, err := f()` do?
7. Why does `go build` refuse unused local variables?
8. Why is "the zero value should be useful" a Go design rule?

## Answers

1. **`*int`** → `nil`. **`[]string`** → `nil` (length 0,
   capacity 0). **`map[K]V`** → `nil` (read-only-friendly,
   write-hostile). **`chan int`** → `nil` (sends/receives block
   forever).

2. When the map is `nil` and you write to it. A `var m
   map[string]int` is nil; `m["k"] = 1` panics. Initialize first
   with `m := make(map[string]int)` or `m := map[string]int{}`.

3. **Untyped** constants (`const Pi = 3.14`) have a *kind*
   (integer, floating, string, etc.) but no specific type. They
   adapt to context — same `Pi` works in `float32`, `float64`,
   `complex128` contexts. **Typed** constants (`const Pi
   float32 = 3.14`) lock to a specific type and need explicit
   conversion to use elsewhere. Default to untyped unless you
   need the typing.

4. The start of a `const ()` block. Two separate `const ()`
   blocks each begin `iota` at 0. Inside one block, `iota`
   increments per *line* (not per identifier).

5. **`:=`** only inside functions, and only when at least one
   variable on the LHS is new. **`var`** is the only form
   available at package scope; it also works inside functions
   when you want to declare a variable for assignment later
   (`var err error`).

6. Calls `f()`, discards the first return via the blank
   identifier `_`, and assigns the second return to `err`. `_`
   is a write-only universal target — useful when a function
   forces you to acknowledge a value you don't need.

7. Because they're almost always a sign of a bug — leftover
   from a refactor, or code meant to use the value but doesn't.
   Forbidding them at compile time catches the class
   mechanically. Same logic for unused imports.

8. So that `var x T` is immediately useful. A `bytes.Buffer{}`
   works; a `sync.Mutex{}` is unlocked; an `http.Server{}` can
   `ListenAndServe`. Library authors design types so this is
   true. The downstream effect: less constructor ceremony, less
   "did I forget to call `init()`?" anxiety.
