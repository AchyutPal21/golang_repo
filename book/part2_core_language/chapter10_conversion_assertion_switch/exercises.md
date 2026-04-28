# Chapter 10 — Exercises

## 10.1 — Predict the output

Run all three examples. Predict each line before reading it.

```bash
go run ./examples/01_conversions
go run ./examples/02_type_assertions
go run ./examples/03_type_switch
```

## 10.2 — Safe-assertion refactor

Open [`exercises/01_safe_assert/main.go`](exercises/01_safe_assert/main.go).
The reference solution already shows the comma-ok pattern; deliberately
break it back to panicky form and observe the panic on `nil` or `3.14`.

## 10.3 ★ — Decode `any` into a typed config

Write a function `func DecodeConfig(raw any) (Config, error)` that takes
the result of `json.Unmarshal(_, &any{})` and extracts a `Config{Host
string; Port int; Verbose bool}` if the shape matches; returns a
descriptive error otherwise.

Hint: input will be `map[string]any`; numbers come back as `float64`.
