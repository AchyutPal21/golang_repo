# Chapter 14 — Revision Checkpoint

## Questions

1. Does a closure capture a variable by value or by reference?
2. What is the loop-variable capture trap? When was it fixed at the language level?
3. You launch 5 goroutines in a loop in Go 1.20, each printing `n`. What do they print?
4. What does `sync.Once` guarantee?
5. Name two production patterns that are fundamentally built on closures.
6. When should you use a struct with methods instead of a closure?

## Answers

1. By reference. The closure holds a pointer to the captured variable, not a
   copy of its value at creation time.

2. Pre-Go-1.22: all iterations of a `for` loop shared the same loop variable.
   Closures or goroutines created inside the loop all captured the same variable
   and would see the final value when called after the loop. Go 1.22 creates a
   new variable per iteration.

3. All five goroutines may print `5` (the value of `n` after the loop). The loop
   finishes before the goroutines run, and they all captured the same `n`.
   Fix: pass `n` as a goroutine argument: `go func(n int) { ... }(n)`.

4. The function passed to `Do` runs exactly once, even when called concurrently
   from multiple goroutines. Subsequent calls block until the first is done and
   then return without executing the function again.

5. Any two of: functional options, middleware/handler wrapping, lazy evaluation,
   memoization, closure-based iterators, stateful generators.

6. When the state has multiple interrelated fields, the type needs to implement
   an interface, the type is part of a public API, or you need to write targeted
   tests for the individual methods.
