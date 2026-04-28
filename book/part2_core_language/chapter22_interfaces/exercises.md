# Chapter 22 — Exercises

## 22.1 — io.Reader/Writer pipeline

Run [`exercises/01_io_pipeline`](exercises/01_io_pipeline/main.go).

`limitedReader` and `rot13Reader` both implement `io.Reader` and can be
chained. The ROT-13 test verifies it is its own inverse.

Try:
- Add a `loggingReader` that prints each call to `Read` with byte count.
- Implement a `multiReader` that reads from a list of `io.Reader`s in sequence
  (like `io.MultiReader`).
- Implement a `teeReader` that reads from r and writes everything read to w
  (like `io.TeeReader`).

## 22.2 ★ — Plugin system

Design a `Plugin` interface with `Name() string` and `Execute(ctx Context) error`.
Build a `Registry` that registers plugins by name and runs them.
Write two plugins: a logging plugin and a metrics plugin.
Demonstrate registering, listing, and running them.

## 22.3 ★ — Sort interface

Implement `sort.Interface` on a custom `ByLength []string` type so that
strings are sorted shortest-first, longest-last, with alphabetical tiebreaking.
Use `sort.Sort`, then verify with `sort.IsSorted`.
Also implement a generic `SortFunc[T]` that wraps `sort.Slice` and accepts
a less function: `SortFunc(items, func(a, b T) bool)`.
