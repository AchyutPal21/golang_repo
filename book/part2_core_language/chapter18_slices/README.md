# Chapter 18 — Slices: The Most Important Type

> **Part II · Core Language** | Estimated reading time: 28 min | Runnable examples: 3 | Exercises: 1

---

## Why this chapter matters

If you understand one Go type deeply, make it slices. They are the central data structure of almost every Go program: HTTP request bodies, database rows, JSON arrays, file lines, worker queues — all slices. Slice bugs (aliasing, append-past-cap overwrites, nil vs empty confusion) are among the most common sources of subtle production defects. Understanding the three-word header is the key to all of it.

---

## 18.1 — The slice header

A slice is not the data itself — it is a three-word descriptor:

```
┌──────────┬───────┬──────────┐
│  pointer │  len  │   cap    │
└──────────┴───────┴──────────┘
```

- **pointer**: address of the first element in the underlying array
- **len**: number of elements visible through this slice
- **cap**: number of elements from the pointer to the end of the underlying array

Assignment and function calls copy this header. Two headers can point to the same backing array — this is aliasing.

---

## 18.2 — nil vs empty slice

```go
var ns []int      // nil slice:   ptr=nil, len=0, cap=0
es := []int{}     // empty slice: ptr=<non-nil>, len=0, cap=0

ns == nil // true
es == nil // false
len(ns) == len(es) // true (both 0)
```

**nil slice** is the zero value — safe to range over and append to, but not identical to an empty slice. This matters when marshalling to JSON (`null` vs `[]`) and when comparing with `reflect.DeepEqual`.

**Rule**: return `nil` (not `[]T{}`) from functions that produce "no results" unless your callers specifically need to distinguish empty from absent.

---

## 18.3 — make

```go
make([]T, n)     // len=n, cap=n — elements zero-initialised
make([]T, n, m)  // len=n, cap=m (m >= n)
```

Pre-allocating with `make([]T, 0, n)` avoids reallocation when you know the final size:

```go
result := make([]int, 0, len(input))
for _, v := range input {
    if pred(v) {
        result = append(result, v)
    }
}
```

---

## 18.4 — append and growth

`append` returns a new slice header. You **must** use the return value:

```go
s = append(s, v) // correct
append(s, v)     // wrong: return value discarded
```

When `len == cap`, `append` allocates a new, larger backing array and copies. The growth factor is approximately 2× for small slices, tapering toward 1.25× for large ones (Go 1.18+). After growth, the old and new slices no longer share memory.

Watch the doubling in the growth tracker output:
```
len=1  cap=1
len=2  cap=2
len=3  cap=4
len=5  cap=8
len=9  cap=16
...
```

---

## 18.5 — Aliasing: the #1 slice bug

Sub-slicing does not copy data:

```go
original := []int{1, 2, 3, 4, 5}
alias := original[1:4] // shares backing array

alias[0] = 200
// original is now [1 200 3 4 5]
```

This is intentional and efficient — but dangerous when unexpected.

---

## 18.6 — The append-within-cap trap

When `append` does not need to grow the backing array, it writes into the existing array. This can corrupt a slice the caller still holds:

```go
base := make([]int, 3, 6) // [1 2 3] with spare capacity
sub := base[:2]            // [1 2], cap=6

sub = append(sub, 99)      // fits within cap — writes to base[2]
// base is now [1 2 99] — corrupted
```

**Fix**: use the 3-index slice to limit capacity:

```go
sub := base[:2:2] // cap=2; append forces new allocation
```

---

## 18.7 — The 3-index slice

`s[low:high:max]` creates a slice with:
- `len = high - low`
- `cap = max - low`

The maximum subscript position prevents reads or appends from going further into the backing array than intended. Use it whenever you hand a sub-slice to external code that might append.

---

## 18.8 — copy

`copy(dst, src)` copies `min(len(dst), len(src))` elements and returns the count. It handles overlapping source and destination correctly.

```go
dst := make([]int, 3)
n := copy(dst, src) // n == min(len(dst), len(src))
```

To clone a slice:

```go
clone := make([]int, len(s))
copy(clone, s)
// or with append: clone := append([]int(nil), s...)
```

---

## 18.9 — Common slice idioms

### Delete (unordered, O(1))

```go
s[i] = s[len(s)-1]
s = s[:len(s)-1]
```

### Delete (ordered, O(n))

```go
s = append(s[:i], s[i+1:]...)
```

### Insert at index i

```go
s = append(s, 0)
copy(s[i+1:], s[i:])
s[i] = v
```

### Filter in-place (reuses backing array)

```go
out := s[:0]
for _, v := range s {
    if keep(v) {
        out = append(out, v)
    }
}
```

### Deduplicate (sorted input)

```go
out := s[:1]
for _, v := range s[1:] {
    if v != out[len(out)-1] {
        out = append(out, v)
    }
}
```

---

## 18.10 — Function mutation gotcha

A slice header is a value. When you pass a slice to a function and the function appends beyond capacity, the function gets a new backing array — the caller's slice is unchanged:

```go
func bad(s []int) {
    s = append(s, 99) // new header — caller doesn't see it
}

func good(s []int) []int {
    return append(s, 99) // caller must reassign: s = good(s)
}
```

**Rule**: any function that needs to grow a slice must return the new slice, or accept `*[]T`.

---

## 18.11 — Defensive copy

When your function stores a slice provided by the caller, copy it to avoid aliasing:

```go
func (q *Queue) Enqueue(items []int) {
    cp := make([]int, len(items))
    copy(cp, items)
    q.data = append(q.data, cp...)
}
```

Similarly, when returning a sub-slice from an internal buffer, return a copy unless you explicitly document the aliasing contract.

---

## 18.12 — Slices of slices

A `[][]T` is a slice of independent slice headers. Each inner slice has its own backing array:

```go
grid := make([][]int, rows)
for i := range grid {
    grid[i] = make([]int, cols)
}
```

This is not the same as a flat `[rows*cols]T` array laid out contiguously in memory. For cache-efficient 2D data, use a flat slice and compute indices manually: `data[row*cols + col]`.

---

## Running the examples

```bash
cd book/part2_core_language/chapter18_slices

go run ./examples/01_slice_header    # header internals, nil vs empty, make, copy, 3-index
go run ./examples/02_append_growth   # growth tracking, pre-alloc, delete, insert, filter
go run ./examples/03_aliasing_traps  # all three aliasing traps + fixes

go run ./exercises/01_stack          # generic stack + balanced parentheses checker
```

---

## Exercises

### [exercises.md](exercises.md)

---

## Revision checkpoint

### [checkpoint.md](checkpoint.md)

---

## Key takeaways

1. A slice is a **three-word header** (pointer, len, cap) — not the data itself.
2. **nil slice ≠ empty slice** — both have len=0 but different pointer values and JSON representations.
3. **append must be assigned back** — `s = append(s, v)`.
4. **Sub-slicing aliases** the backing array — writes through one slice affect the other.
5. **append within cap** overwrites the original backing array — use `s[lo:hi:hi]` to prevent it.
6. To **clone** a slice: `append([]T(nil), s...)` or `copy(dst, s)`.
7. Pre-allocate with `make([]T, 0, n)` when the final size is known.

---

## Cross-references

- **Chapter 17** — Arrays: the backing store slices point into
- **Chapter 19** — Maps: similar gotchas around nil vs empty, reference semantics
- **Chapter 24** — Generics: `slices` package (Go 1.21) for sort, search, clone
