# Chapter 8 — Exercises

## 8.1 — Zero-value tour

Run [`examples/02_zero_values`](examples/02_zero_values/main.go).
Before reading each printed value, predict it from the declaration.
Mismatches reveal mental-model gaps.

**Acceptance.** You can recall the zero value for every built-in
type without running the program.

## 8.2 — String-keyed map → typed enum

Find a map you've used as an "enum" — e.g.
`map[string]int{"low":1,"med":2,"high":3}` — and rewrite it as a
typed enum:

```go
type Severity int

const (
    SeverityLow Severity = iota
    SeverityMedium
    SeverityHigh
)

func (s Severity) String() string { ... }
```

Add a `String()` method. Confirm `fmt.Println(SeverityHigh)`
prints `"high"`, not `2`.

## 8.3 ★ — Find the shadowing bug

Run:

```bash
go run ./exercises/01_safe_defaults
```

Read the code. Identify the shadowing problem. Fix it; rerun;
verify with:

```bash
go vet ./exercises/01_safe_defaults
```

After the fix, `vet` should be silent.

## 8.4 ★★ — JSON-marshalable typed enum

Build a `Status` package with:

* `type Status int` plus `iota` constants.
* `String() string` method.
* `MarshalJSON` / `UnmarshalJSON` methods so the value
  round-trips as a string in JSON.
* Tests confirming round-trip.

You'll need `encoding/json` and `errors`. The pattern is real
production code; senior reviewers expect you to know it.
