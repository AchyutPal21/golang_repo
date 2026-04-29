# Chapter 29 — Exercises

## 29.1 — Refactor violations

Run [`exercises/01_refactor_violations`](exercises/01_refactor_violations/main.go).

Each section demonstrates one SOLID violation and the corrected design.

Try:
- Add an `XMLExporter` to the OCP section without modifying `exportReport`.
- Add a `RateLimitedNotifier` wrapper that enforces a max-per-minute budget without modifying `AlertService`.
- Write a `retryLogger` that wraps `Logger` and retries failed log writes up to 3 times — verify it satisfies the `Entries()` contract.

## 29.2 ★ — Violation hunt

Pick any module from the `golang-mastery-updated/` source material (or any file in the existing book). Identify at least one SOLID violation per principle and describe the fix in a comment at the top of a new file.

## 29.3 ★★ — Plugin system

Build a mini plugin system that embodies OCP:

```go
type Plugin interface {
    Name() string
    Execute(input string) (string, error)
}

type Pipeline struct{ plugins []Plugin }
```

`Pipeline.Run(input)` threads `input` through each plugin in order.
Add `UppercasePlugin`, `TrimPlugin`, and `PrefixPlugin` without modifying `Pipeline`.
Wire them in `main()` — no framework, no reflection.
