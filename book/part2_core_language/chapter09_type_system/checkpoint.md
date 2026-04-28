# Chapter 9 — Revision Checkpoint

## Questions

1. What's the difference between `byte` and `rune`?
2. How big is `int` on a typical 2026 server?
3. What does `len(s)` return for a UTF-8 string?
4. Why is `string(65)` `"A"` instead of `"65"`?
5. Why never use `float64` for money?
6. How do you compare two floats for "equality"?
7. Why does `s[0] = 'X'` not compile?

## Answers

1. `byte` is an alias for `uint8` (1 byte). `rune` is an alias
   for `int32` (a Unicode code point, 1–4 bytes when UTF-8
   encoded). Use the alias that conveys intent.

2. 64 bits, on every platform you'll likely use. `int` is at
   least 32 bits per the spec, but on every 64-bit OS it's 64.
   Don't rely on this when serializing to disk/wire; use `int64`
   explicitly for those.

3. The number of *bytes*, not runes (or "characters"). For ASCII
   they're the same; for UTF-8 with multi-byte runes, the byte
   count is larger. Use `utf8.RuneCountInString` for runes.

4. `string(intVal)` interprets the int as a Unicode code point;
   65 is U+0041, which is "A". To format a number as its decimal
   string representation, use `strconv.Itoa(65)` → `"65"`.
   `go vet` catches the common mistake.

5. IEEE-754 floats are binary. `0.1` is not exactly representable;
   arithmetic compounds the error. `$0.10 + $0.20 ≠ $0.30`. Use
   `int64` cents (or a decimal package) for currency.

6. Use a tolerance: `math.Abs(a-b) < epsilon`, where epsilon is
   chosen for the problem. Equality with `==` is almost never
   correct for floats.

7. Strings are immutable byte sequences. To mutate, convert to
   `[]byte`, mutate, convert back to `string`. Both conversions
   copy the data.
