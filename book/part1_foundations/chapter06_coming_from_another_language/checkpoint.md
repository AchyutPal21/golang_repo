# Chapter 6 — Revision Checkpoint

## Questions

1. Name the four biggest mental shifts when transferring to Go.
2. How does Go express "a class with state and behavior"?
3. How does Go handle errors, given that there are no exceptions?
4. What is Go's equivalent of `async/await`?
5. What is Go's equivalent of inheritance?
6. Why doesn't Go have decorators?
7. What's the typical advice for the first 90 days transferring
   to Go?
8. When does it make sense to keep a hot path in C++/Rust and
   write only the surrounding code in Go?

## Answers

1. (a) **Errors as values** — explicit `(value, error)` returns
   replace exceptions. (b) **Goroutines + channels** — concurrency
   is a language primitive, not a library. (c) **Composition over
   inheritance** — structs + interfaces + embedding replace class
   hierarchies. (d) **Stdlib over frameworks** — most of what
   you'd reach for a framework for is built into the standard
   library.

2. A `struct` for state, plus methods on that struct for behavior.
   Use embedding to compose; use interfaces to express
   polymorphism.

3. Functions that can fail return `(T, error)`. The caller checks
   `err != nil` immediately. Errors propagate by being returned
   (not unwound). `errors.Is`/`errors.As` give typed matching;
   `fmt.Errorf("...: %w", err)` wraps for context. `panic` exists
   for genuinely-unrecoverable errors and is rare in normal code.

4. Goroutines + channels. `go func() { ... }()` spawns concurrent
   work; channels coordinate. The model is preemptive (not an
   event loop), and goroutines run on multiple OS threads truly
   in parallel.

5. Composition via *struct embedding* plus *interfaces*. Embedding
   promotes the methods of an embedded type (looks like
   inheritance, but isn't — there's no virtual dispatch on
   embedded methods). Polymorphism comes from interfaces, which
   are satisfied implicitly by any type with the matching methods.

6. Decorators are ergonomic syntax for "wrap this function." Go
   has the same effect via plain function wrapping (`logged :=
   logging(handler)`) but no syntax sugar. The trade: less
   concise, more legible — you can't accidentally hide a side
   effect behind an annotation.

7. Spend the first month writing pure Go, not your-old-language-
   in-Go. Reject "but in X we did it like…" reflexively for 30
   days. Read 1,000 lines of standard-library source before
   reaching for third-party libraries. Pair-review for the first
   three PRs with someone fluent in Go.

8. When the hot path is genuinely CPU-bound and the perf
   difference matters at scale (e.g. data plane in a high-
   throughput service). The control plane, ops tooling, and
   surrounding code go in Go for productivity. Common patterns:
   C++ data plane + Go control plane; Rust core + Go ops
   tooling. The decision rule: "is developer velocity worth more
   than perf here?" Usually yes everywhere except the inner loop
   of a small number of CPU-bound services.
