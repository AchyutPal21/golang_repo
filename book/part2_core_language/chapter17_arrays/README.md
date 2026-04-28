# Chapter 17 — Arrays: The Real Underlying Type

> **Part II · Core Language** | Estimated reading time: 16 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Arrays are the foundation slices are built on. They are rarely used directly, but understanding their value semantics — especially the contrast with slices — prevents a large class of bugs. Fixed-size arrays also have legitimate production uses: cryptographic digests, IP addresses, look-up tables, and any case where the size is an invariant you want the compiler to enforce.

---

## 17.1 — Declaration

```go
var a [5]int          // zero-initialised: [0 0 0 0 0]
b := [5]int{1,2,3,4,5}
c := [...]string{"x","y","z"} // compiler counts: len == 3
d := [10]int{0:1, 5:5, 9:9}  // sparse: unspecified indices are zero
```

The length is **part of the type**: `[3]int` and `[4]int` are distinct, incompatible types. You cannot pass a `[3]int` where a `[4]int` is expected.

---

## 17.2 — Value semantics

Assignment and function calls **copy** the entire array:

```go
a := [3]int{1, 2, 3}
b := a       // full copy — b is independent of a
b[0] = 999
fmt.Println(a) // [1 2 3] — unchanged
```

This is the key distinction from slices. Passing a large array to a function is expensive — the entire array is copied. For large arrays, pass a pointer:

```go
func process(arr *[1024]float64) { ... }
```

---

## 17.3 — Comparable

Arrays are comparable with `==` if their element type is comparable:

```go
[3]int{1,2,3} == [3]int{1,2,3} // true
[3]int{1,2,3} == [3]int{1,2,4} // false
```

This makes arrays usable as map keys — useful for fixed-size identifiers.

---

## 17.4 — Fixed-size type uses

When size is a semantic invariant, use an array type:

```go
type SHA256Digest [32]byte    // always exactly 32 bytes
type IPv4Address  [4]byte     // comparable — usable as map key
type RGBPixel     [3]uint8
```

These types carry their size in the type system. A function that accepts `SHA256Digest` cannot accidentally receive a shorter or longer hash.

---

## 17.5 — Array as slice backing store

Every slice is a triple: `(pointer, length, capacity)`. The pointer points into an underlying array. You can slice an array directly:

```go
arr := [5]int{1, 2, 3, 4, 5}
s := arr[1:4]  // slice backed by arr; s shares memory
s[0] = 200     // modifies arr[1]
```

`arr[:]` gives a slice covering the whole array. This is how `make([]T, n)` works internally — the runtime allocates an array and returns a slice header pointing to it.

---

## 17.6 — Multi-dimensional arrays

```go
var grid [3][3]int
grid[1][2] = 9
```

A `[m][n]T` is an array of `m` elements, each of type `[n]T`. Iteration with `range` works naturally:

```go
for i, row := range grid {
    for j, v := range row {
        _ = v
    }
}
```

---

## Running the examples

```bash
cd book/part2_core_language/chapter17_arrays

go run ./examples/01_array_basics    # declaration, init, comparison
go run ./examples/02_array_as_value  # value semantics, pointer, slice backing

go run ./exercises/01_matrix         # matrix transpose, trace, multiply
```

---

## Exercises

### [exercises.md](exercises.md)

---

## Revision checkpoint

### [checkpoint.md](checkpoint.md)

---

## Key takeaways

1. Array length is **part of the type** — `[3]int ≠ [4]int`.
2. Arrays have **value semantics** — assignment and function calls copy the entire array.
3. Arrays are **comparable** — usable as map keys.
4. Use arrays for **fixed-size types** where size is a semantic invariant (digests, addresses).
5. Arrays are the **backing store** for slices — every slice points into one.

---

## Cross-references

- **Chapter 18** — Slices: built on top of arrays; the type you actually use
- **Chapter 19** — Maps: arrays as map keys via comparability
- **Chapter 20** — Structs: field alignment applies to struct fields, not array elements
