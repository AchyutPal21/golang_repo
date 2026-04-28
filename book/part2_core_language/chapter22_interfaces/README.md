# Chapter 22 — Interfaces: Go's Killer Feature

> **Part II · Core Language** | Estimated reading time: 28 min | Runnable examples: 3 | Exercises: 1

---

## Why this chapter matters

Interfaces are the mechanism that makes Go programs composable, testable, and extensible without inheritance hierarchies. They are why you can write an `io.Copy` that works on files, network connections, in-memory buffers, and gzip streams — all without those types knowing about each other. Understanding the two-word interface value, the typed-nil trap, and the "accept interfaces, return structs" principle is essential for any non-trivial Go program.

---

## 22.1 — Interface declaration

```go
type Shape interface {
    Area() float64
    Perimeter() float64
}
```

An interface defines a **method set** — the set of methods a type must implement to satisfy the interface. There is no `implements` keyword. Satisfaction is checked by the compiler wherever an interface is used.

---

## 22.2 — Implicit satisfaction

Any type with the required methods satisfies the interface automatically:

```go
type Circle struct{ Radius float64 }
func (c Circle) Area() float64      { ... }
func (c Circle) Perimeter() float64 { ... }

var s Shape = Circle{Radius: 5} // OK: Circle has both methods
```

This is Go's duck typing at compile time. The separation means a type in package A can satisfy an interface defined in package B without A ever importing B — the key to decoupling.

---

## 22.3 — Interface internals: the two-word header

An interface value is a pair:
```
┌────────────┬────────────┐
│  *itab     │   data     │
└────────────┴────────────┘
```

- **`*itab`**: pointer to a structure containing the concrete type and a table of method pointers
- **data**: pointer to (or, for small types, copy of) the concrete value

A **nil interface** has both words nil. It compares equal to `nil`.

A **typed nil** has a non-nil `*itab` (the type is known) and a nil data pointer (the value is nil). It does **not** compare equal to `nil`. This is the most common interface bug in Go.

---

## 22.4 — The typed-nil trap

```go
func findUser(fail bool) error {
    var err *myError // nil pointer, type *myError
    if fail {
        err = &myError{"not found"}
    }
    return err // WRONG: wraps nil *myError in error interface
}

err := findUser(false)
err == nil // FALSE — itab is set, data is nil
```

**Fix**: return the interface type directly, never through an intermediate typed variable:

```go
func findUser(fail bool) error {
    if fail {
        return &myError{"not found"}
    }
    return nil // true nil interface
}
```

**Rule**: never assign to a concrete error variable and then return it through an interface.

---

## 22.5 — Interface composition

Interfaces can embed other interfaces:

```go
type Reader interface { Read(p []byte) (n int, err error) }
type Writer interface { Write(p []byte) (n int, err error) }

type ReadWriter interface {
    Reader
    Writer
}
```

`io.ReadWriter`, `io.ReadWriteCloser`, `io.ReadSeeker` — all are built this way. This is the standard library's approach: define small interfaces, compose larger ones.

---

## 22.6 — Small interfaces: the key design principle

The most useful interfaces in Go have one or two methods:

| Interface | Method(s) |
|---|---|
| `io.Reader` | `Read` |
| `io.Writer` | `Write` |
| `io.Closer` | `Close` |
| `fmt.Stringer` | `String() string` |
| `error` | `Error() string` |
| `sort.Interface` | `Len`, `Less`, `Swap` |

Small interfaces are easy to satisfy, easy to fake in tests, and compose well.

---

## 22.7 — Accept interfaces, return structs

The idiomatic Go function signature:

```go
// Accept the narrowest interface that works — maximum flexibility for callers.
func processReader(r io.Reader) error { ... }

// Return a concrete type — callers get the full API, no unwrapping needed.
func NewFile(path string) (*File, error) { ... }
```

Accepting `io.Reader` instead of `*os.File` means your function works with files, network connections, in-memory buffers, compressed streams, and test fakes — without any changes.

Returning a concrete type instead of an interface means callers can access all methods without type assertions.

---

## 22.8 — Interfaces for testing

Dependency injection via interfaces is the idiomatic way to make code testable:

```go
type EmailSender interface {
    Send(to, subject, body string) error
}

type UserService struct {
    email EmailSender
}

// In tests: inject fakeEmailSender
// In production: inject smtpEmailSender
```

No reflection, no mocking framework — just pass a different implementation. This is why interfaces are "Go's killer feature."

---

## 22.9 — The io.Reader pipeline pattern

Wrapping `io.Reader` to transform data on the fly is idiomatic and powerful:

```go
type countingReader struct { r io.Reader; count int64 }
func (c *countingReader) Read(p []byte) (n int, err error) {
    n, err = c.r.Read(p)
    c.count += int64(n)
    return
}
```

Compose wrappers to build pipelines:

```go
src := os.Open("file.gz")
cr := &countingReader{r: src}
gr, _ := gzip.NewReader(cr)
io.Copy(dst, gr)
fmt.Println("compressed bytes read:", cr.count)
```

The entire standard library is built on this pattern: `gzip`, `bufio`, `crypto/cipher`, `net/http`.

---

## 22.10 — The error interface

```go
type error interface {
    Error() string
}
```

The simplest interface in the standard library. Any type with `Error() string` is an error. Use `errors.Is` for sentinel comparison and `errors.As` for type extraction:

```go
if errors.Is(err, ErrNotFound) { ... }

var ve *ValidationError
if errors.As(err, &ve) { /* access ve.Field */ }
```

Implement `Unwrap() error` to support error wrapping chains.

---

## 22.11 — Interface comparability

Two interface values are equal if they have the same dynamic type and equal dynamic value. Comparing interface values holding non-comparable types (slices, maps) panics at runtime.

---

## Running the examples

```bash
cd book/part2_core_language/chapter22_interfaces

go run ./examples/01_interface_basics    # declaration, satisfaction, composition, type switch
go run ./examples/02_interface_internals # itab, typed-nil trap, comparison, overhead
go run ./examples/03_interface_patterns  # accept interfaces, testing, io pipeline, errors

go run ./exercises/01_io_pipeline        # limitedReader + rot13Reader pipeline
```

---

## Exercises

### [exercises.md](exercises.md)

---

## Revision checkpoint

### [checkpoint.md](checkpoint.md)

---

## Key takeaways

1. Interface satisfaction is **implicit** — no `implements` keyword.
2. An interface value is **two words**: `(type, value)`. A nil interface has both nil.
3. **Typed nil** is not a nil interface — it has a type but a nil value. The most common Go gotcha.
4. **Accept interfaces, return structs** — maximum flexibility for callers.
5. **Small interfaces** (1-2 methods) are the most useful and composable.
6. **Interfaces enable testing** via dependency injection — inject fake implementations.
7. The **io.Reader pipeline** pattern is the foundation of streaming I/O in Go.

---

## Cross-references

- **Chapter 10** — Type assertions and type switches (syntactic foundation)
- **Chapter 21** — Methods: method sets determine interface satisfaction
- **Chapter 23** — Embedding and Composition: promoted methods in interfaces
- **Chapter 41** — Errors as Values: full error interface and wrapping pattern
