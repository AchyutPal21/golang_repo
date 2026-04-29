# Chapter 31 — Revision Checkpoint

## Questions

1. What does a Factory Method return in Go, and why is that the right type to return?
2. How does a Builder handle validation without forcing callers to check errors on every chained call?
3. What is the difference between a Builder and Functional Options (Chapter 28)?
4. Why is `sync.Once` preferred over `init()` for singletons?
5. What must a `Clone()` method do that a simple struct copy (`t2 := *t1`) does not?

## Answers

1. A Go factory method returns an **interface** — not a concrete pointer. This hides
   the concrete type from the caller, making it possible to swap implementations
   (test fake, production implementation, alternative provider) without changing any
   call site. If the factory returned a concrete `*jsonLogger`, callers would become
   coupled to it and could not be given a `consoleLogger` at runtime.

2. The builder stores the **first error encountered** in a field (`b.err`). Every
   method checks `if b.err != nil { return b }` and skips its work. The error is
   surfaced only when `Build()` is called. This lets callers chain methods fluently
   without `if err != nil` after every step, while still getting a clear error at the
   end if any step failed.

3. **Builder**: mutable struct with chained setters; `Build()` validates and constructs
   the final object. Good for objects with many inter-dependent fields, or when
   construction requires sequential validation across multiple fields.
   **Functional Options** (`func(*T) error`): each option is an independent function;
   `NewServer` applies them in order. Better for optional configuration on an already-
   constructable type, especially in public APIs where backward compatibility matters.
   Both are valid; the choice depends on whether fields are inter-dependent and whether
   the caller needs a separate "build" step.

4. `init()` runs **unconditionally** when the package is imported — even in tests that
   do not need the singleton. `sync.Once` runs **lazily** on the first call to the
   accessor function, and only then. This makes it safe to import the package in tests
   without triggering expensive initialisation (database connections, file reads,
   network calls) that the test does not need.

5. A simple struct copy (`t2 := *t1`) does a **shallow copy**: pointer fields and
   slice/map headers are copied, but both copies point to the same underlying data.
   Modifying a slice element in `t2` also modifies `t1`. `Clone()` must perform a
   **deep copy**: allocate new backing arrays for slices (`make` + `copy`), allocate
   new maps and copy each key-value pair. After a correct `Clone()`, the two objects
   are fully independent — no shared mutable state.
