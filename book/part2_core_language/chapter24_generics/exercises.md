# Chapter 24 — Exercises

## 24.1 — Generic OrderedSet

Run [`exercises/01_collections`](exercises/01_collections/main.go).

`OrderedSet` maintains insertion order while preventing duplicates, using
a slice for order and a map for O(1) membership testing.

Try:
- Implement `Intersection[T comparable](a, b *OrderedSet[T]) *OrderedSet[T]`.
- Implement `Difference[T comparable](a, b *OrderedSet[T]) *OrderedSet[T]` (elements in a but not b).
- Why does `Remove` swap with the last element rather than shifting? What does this break?

## 24.2 ★ — Generic result type

Implement `Result[T any]` with `Ok(v T) Result[T]` and `Err(e error) Result[T]`.
Add `Map[T, U any](r Result[T], f func(T) U) Result[U]` and
`FlatMap[T, U any](r Result[T], f func(T) Result[U]) Result[U]`.
Build a pipeline: parse string → validate range → format output, using `FlatMap`.

## 24.3 ★ — Constraint design

Design a `Number` constraint that includes all integer and float types.
Implement `Statistics[T Number](data []T) (mean, variance float64)`.
Why must the return type be `float64` even when `T` is `int`?
