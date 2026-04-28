# Chapter 17 — Exercises

## 17.1 — Matrix operations

Run [`exercises/01_matrix`](exercises/01_matrix/main.go).

Implements `transpose`, `trace`, and `multiply` on `[3][3]int`.
All functions receive and return copies — no mutation of the caller's array.

Try:
- Add `add(a, b Mat3) Mat3` for element-wise addition.
- Verify that `multiply(m, identity) == m` for any m using `==`.
- Implement `det(m Mat3) int` (3×3 determinant via cofactor expansion).

## 17.2 ★ — Ring buffer

Implement a fixed-capacity ring buffer using `[8]int`:

```go
type Ring struct {
    buf  [8]int
    head int
    tail int
    len  int
}
```

Implement `Push(v int) bool` (false if full) and `Pop() (int, bool)`.
Why is a power-of-two capacity convenient?

## 17.3 — IPv4 routing table

Using `IPv4Address` as a map key (from example 02), build a simple routing
table that maps subnets to next-hop addresses. Look up a few IPs and print
their next hops. What would you need to implement CIDR prefix matching?
