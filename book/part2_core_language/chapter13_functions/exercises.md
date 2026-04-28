# Chapter 13 — Exercises

## 13.1 — String pipeline

Run [`exercises/01_pipeline`](exercises/01_pipeline/main.go).

The exercise implements `pipeline(s string, steps ...Transformer) string` and
demonstrates it with a Caesar cipher built from two factories:
`shiftBy(n int) Transformer` and `onlyLetters() Transformer`.

Study the implementation, then try:
- Extend `pipeline` to report which step produced each intermediate value.
- Write a `reverse() Transformer` that reverses a string.
- What happens when you call `shiftBy(0)`? `shiftBy(26)`? `shiftBy(-26)`?

## 13.2 ★ — Memoized Fibonacci

Write a memoized Fibonacci using the `memoize` pattern from example 03.
Note: naïve recursive memoize does not work without a shared cache across
recursive calls — you need to store the function in a variable first and
reference it within the closure.

```go
var fib func(int) int
fib = memoize(func(n int) int {
    if n <= 1 { return n }
    return fib(n-1) + fib(n-2)
})
```

Why does this work? Trace through the closure environment mentally.

## 13.3 ★ — Typed function registry

Build a `Registry[T]` that maps string names to `func() T`.
Implement `Register(name string, factory func() T)` and
`Build(name string) (T, error)`.
Use it to register three different greeting functions and dispatch by name.
