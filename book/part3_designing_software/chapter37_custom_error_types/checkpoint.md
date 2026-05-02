# Chapter 37 — Revision Checkpoint

## Questions

1. Why must error struct methods use a pointer receiver?
2. What does `Unwrap()` return and what does the standard library do with it?
3. When should you implement a custom `Is()` method instead of relying on pointer identity?
4. What is the typed nil trap and how do you avoid it?
5. What is the advantage of error behaviour interfaces over type-switching on concrete error types?

## Answers

1. `errors.As` looks for the first error in the chain that is **assignable to `*T`**
   (the pointer type). If you define methods on `T` (value receiver), `errors.As`
   with a `**T` target cannot find it. More importantly, error values are almost
   always passed as interfaces — storing a value-type error in an interface creates
   a copy, and the copy's address differs on every call. Using a pointer receiver
   means all copies of the error *point to the same struct*, making pointer-identity
   comparisons consistent.

2. `Unwrap()` returns the **wrapped cause** (`error`). `errors.Is` calls `Unwrap()`
   recursively to walk the chain until it finds a matching error or reaches `nil`.
   `errors.As` does the same but checks assignability. `fmt.Errorf("... %w ...", err)`
   creates a wrapper that also implements `Unwrap()` automatically — no custom type
   needed for simple wrapping.

3. Implement custom `Is()` when two different *instances* of the same error type
   should match each other based on a field value (typically a code or category)
   rather than pointer identity. Example: `APIError{Code: 401, Message: "token expired"}`
   and `APIError{Code: 401, Message: "invalid token"}` are different objects but
   represent the same *kind* of error. Custom `Is()` that compares `Code` makes
   `errors.Is(err, ErrUnauthorised)` work for both without requiring callers to hold
   the exact pointer.

4. The **typed nil trap**: a function with return type `error` can return a
   `*ConcreteError` variable that is `nil`. The interface value has a non-nil type
   (the `*ConcreteError` type) and a nil value — so the interface itself is non-nil,
   and `err == nil` returns false. The fix: return `nil` explicitly (bare untyped nil)
   when there is no error. Never assign to a typed variable and return that variable:
   ```go
   // WRONG:
   var err *MyError
   if ok { err = &MyError{...} }
   return err  // non-nil interface even when err == nil
   
   // CORRECT:
   if ok { return &MyError{...} }
   return nil
   ```

5. Error behaviour interfaces (`Retryable`, `Categorised`) let the *caller* ask
   "can I retry this?" without knowing the concrete error type. New error types
   gain retry support by implementing the interface — no changes to callers.
   Type-switching (`switch e := err.(type) { case *NetworkError: ... }`) couples
   the caller to every concrete type: adding `*TimeoutError` requires editing
   every switch. Behaviour interfaces also compose naturally with `errors.As`,
   which walks the chain, so the behaviour is discoverable even when the error
   is deeply wrapped.
