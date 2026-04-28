# Chapter 15 — Revision Checkpoint

## Questions

1. In what order do deferred calls execute?
2. When are the arguments to a deferred call evaluated?
3. Does `defer` run if the function panics?
4. Under what conditions does `recover()` return a non-nil value?
5. What happens if you call `recover()` outside a deferred function?
6. Name two production uses of `defer`.
7. When should you use `panic` vs. returning an `error`?

## Answers

1. LIFO — last registered, first to run.

2. At the `defer` statement, not when the deferred call executes. Use a closure
   (`defer func() { ... }()`) if you need the deferred call to read the value
   at return time.

3. Yes. Deferred calls run during stack unwinding, before the program crashes.
   This is what makes defer useful for cleanup even in the face of panics.

4. `recover()` returns a non-nil value only when called inside a deferred
   function that is executing because a panic is in progress.

5. It returns `nil` immediately and has no effect. Calling `recover()` from a
   non-deferred context does not intercept panics.

6. Any two of: `defer mu.Unlock()` after acquiring a mutex; `defer f.Close()`
   after opening a file; `defer tx.Rollback()` after beginning a transaction;
   `defer elapsed("fn")()` for timing; annotating errors via named returns.

7. Return an `error` when the caller can reasonably handle the condition
   (missing file, invalid input, network failure). Use `panic` when the program
   is in an unrecoverable state: nil dereference, broken invariant, violated
   contract in library code. The rule of thumb: panics cross package boundaries
   only in extraordinary circumstances; prefer error returns in public APIs.
