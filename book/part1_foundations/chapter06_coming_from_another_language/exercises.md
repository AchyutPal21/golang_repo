# Chapter 6 — Exercises

## Exercise 6.1 — Translate one of your own programs

**Goal.** Force the mental shift by doing the work, on something
you actually know.

**Task.** Pick a 50–200 line program you've written in your prior
language. Translate it to Go *without keeping the structure of the
original.* Resist the urge to translate line-by-line; rewrite as
you'd write it natively in Go.

Things to actively notice:

1. Where did you reach for a class hierarchy? Replace with
   composition.
2. Where did you reach for try/except? Replace with `(value, error)`.
3. Where did you reach for a decorator/middleware? Replace with
   function wrapping.
4. Where did you reach for async/await? Replace with goroutines +
   channels (or a synchronous call wrapped in `go func()`).
5. Where did you reach for a third-party library? Check
   `pkg.go.dev/std` first.

**Acceptance.** The Go version compiles, passes equivalent tests,
and is *not* obviously a translation. Self-review: does it feel
idiomatic, or like X-with-Go-syntax?

---

## Exercise 6.2 — Read the translation table

**Task.** Run:

```bash
go run ./examples/01_translation_table <your-language>
```

For each row, ask yourself "which of my reflexes does this kill?"
Pick the three rows that surprised you most and write a one-
paragraph note for each on what it forces you to give up.

---

## Exercise 6.3 — Read both side-by-sides

**Task.** Read
[`examples/02_python_to_go/main.go`](examples/02_python_to_go/main.go)
and
[`examples/03_javascript_to_go/main.go`](examples/03_javascript_to_go/main.go)
end to end, comments and all. Even if neither is your background
language, they show two different *shapes* of translation: a
straightforward data-shape translation (Python) and a deeper
concurrency-model translation (JavaScript).

**Acceptance.** You can articulate, in one sentence each, what the
mental shift was for each.

---

## Exercise 6.4 ★ — Implement a Promise-like Future

**Goal.** Internalize the goroutine + channel model by building
the JS abstraction yourself.

**Task.** In a new file under `exercises/` (call it `02_future/main.go`),
implement:

```go
type Future[T any] struct { /* … */ }

func StartFuture[T any](fn func() (T, error)) *Future[T]
func (f *Future[T]) Get(ctx context.Context) (T, error)
```

Calling `StartFuture(fn)` should spawn a goroutine that runs `fn`
and stashes its result. `Get` blocks until the result is available
or the context is cancelled.

Wire up a small main that starts three Futures with different
sleeps, then awaits all three with `Get` and prints them in the
order they completed.

**Acceptance.** A working generic Future, ~50–100 lines, no
external deps.

(You'll know everything you need by Chapter 47. This is a landmark
exercise to revisit.)
