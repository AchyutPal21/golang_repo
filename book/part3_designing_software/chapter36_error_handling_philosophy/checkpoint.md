# Chapter 36 — Revision Checkpoint

## Questions

1. Why does `wrappedErr == ErrNotFound` return false even when the wrapped error contains `ErrNotFound`?
2. What is the difference between `errors.Is` and `errors.As`?
3. State the golden rule of error handling in one sentence and give one example of violating it.
4. When is `panic` appropriate, and when is it not?
5. How does `errors.Join` differ from manually concatenating error strings?

## Answers

1. `==` checks **pointer equality** (for errors created with `errors.New`, the pointer
   is the identity). `fmt.Errorf("context: %w", ErrNotFound)` creates a *new* error
   value that wraps `ErrNotFound` — the outer value has a different pointer, so `==`
   returns false. `errors.Is` walks the chain by calling `Unwrap()` repeatedly until
   it finds a match or reaches the end. That is why `errors.Is` is always correct for
   sentinel comparison and `==` is always wrong for wrapped errors.

2. `errors.Is(err, target)` checks whether **any error in the chain equals `target`**
   (using the sentinel's identity or a custom `Is` method). Use it to test for
   conditions: "is this a not-found error?"
   `errors.As(err, &target)` walks the chain looking for an error whose type can be
   **assigned to `target`**. Use it to extract a typed error value and read its fields:
   "is this a `ValidationError`? If so, give me the field name."

3. **Handle OR propagate — never both.** Example of violation:
   ```go
   if err != nil {
       log.Println("database error:", err) // handles (logs)
       return err                           // also propagates — caller logs again
   }
   ```
   The fix: either log and don't return, or wrap and return without logging.

4. **Appropriate**: programmer errors (unreachable code reached, violated precondition,
   nil slice index that "can never happen"), initialisation failures (`Must(cfg.Load())`
   where the app cannot start), and signal propagation across goroutine stack frames
   in rare internal libraries.
   **Not appropriate**: expected errors — network timeouts, not-found results, invalid
   user input, payment declined. These are normal program states that callers must be
   able to handle without crashing.

5. `errors.Join(errs...)` returns a single error value whose `Unwrap() []error` method
   returns the original slice. This means `errors.Is` and `errors.As` can still
   inspect each individual wrapped error. Manual concatenation (`strings.Join` of
   error messages) produces an opaque string — the individual errors are lost and
   cannot be inspected programmatically. `errors.Join` also returns `nil` when all
   inputs are nil, eliminating the "check if the slice is empty" boilerplate.
