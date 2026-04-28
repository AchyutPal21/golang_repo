# Chapter 9 — Exercises

## 9.1 — Integer overflow drill

Predict each line's output before running:

```bash
go run ./examples/01_int_sizes
```

For each wraparound, write down (a) what you expected, (b) what
actually printed, (c) why.

## 9.2 — UTF-8 iteration

Open [`exercises/01_utf8_iterate/main.go`](exercises/01_utf8_iterate/main.go).
The `runeAt` function is implemented; run it and confirm the
output. Then add a `runesInRange(s, from, to)` function that
returns the substring containing runes from index `from` to
`to-1` (Python-slice semantics). Test on multi-byte cases.

## 9.3 ★ — A `Cents` money type

Define `type Cents int64` with methods `Add`, `Sub`, `Mul`,
`Format`. Test the canonical "0.1 + 0.2" failure case:

```go
ten := Cents(10)
twenty := Cents(20)
got := ten.Add(twenty)
if got != Cents(30) { t.Fatal("oh no") }
```

Compare with the `float64` equivalent. Document how much error
accumulates over 1 million additions of 0.01.
