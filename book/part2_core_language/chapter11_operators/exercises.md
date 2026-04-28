# Chapter 11 — Exercises

## 11.1 — Flag-set type

Run [`exercises/01_flag_set`](exercises/01_flag_set/main.go) and
read the implementation. Then add a `Count() int` method that
returns the number of bits set (use `math/bits.OnesCount16`).

## 11.2 — Power-of-2 ceiling

Write `nextPow2(n uint64) uint64` that returns the smallest
power of 2 ≥ n. Use bit operations only (no `math.Pow`).
Hint: `math/bits.Len64(n-1)` gives you the bit position.

## 11.3 ★ — Constant-time equality

Implement `equal(a, b []byte) bool` that runs in constant time
(no early exit). Compare to
`crypto/subtle.ConstantTimeCompare(a, b)` and explain the
difference: when it matters, when it doesn't.
