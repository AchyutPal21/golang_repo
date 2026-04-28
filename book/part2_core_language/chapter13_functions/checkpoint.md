# Chapter 13 — Revision Checkpoint

## Questions

1. What is the canonical Go pattern for returning an error from a function?
2. When should you use named return values?
3. What type does a variadic parameter have inside the function body?
4. How do you spread a slice into a variadic call?
5. When does `init` run? Can you call it manually?
6. Can two `func` values be compared with `==`?
7. What is a function factory? Give an example.

## Answers

1. Return two values: the result and an `error`. The caller checks `if err != nil`.
   This is the `(T, error)` idiom — Go's alternative to exceptions.

2. When the names document the return values (appear in godoc and signatures),
   or in short functions where naked `return` reduces repetition. Avoid naked
   returns in functions longer than ~10 lines.

3. A slice of the element type. `func sum(nums ...int)` — `nums` is `[]int`.

4. Use the `...` spread operator: `sum(mySlice...)`.

5. `init` runs before `main`, after all package-level variable initialisations.
   The runtime calls it automatically; the compiler rejects manual calls to `init()`.

6. No — function values can only be compared to `nil`. This is intentional;
   comparing arbitrary function values for equality is not meaningful in Go.

7. A function that returns a configured function value:
   ```go
   func adder(delta int) func(int) int {
       return func(n int) int { return n + delta }
   }
   add10 := adder(10)
   ```
   `delta` is captured in the returned function's closure.
