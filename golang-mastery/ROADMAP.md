# Go Mastery — Complete Curriculum

> From beginner to senior Go developer. Every module is a runnable, heavily
> commented file. Read it, run it, modify it, break it, fix it.

---

## How to use this curriculum

```bash
cd /home/achyut-pal/Desktop/upskill-go/golang-mastery

# Run any module
go run 01_fundamentals/main.go
go run 02_functions/main.go
# ... etc.

# Run with race detector (important for Module 06)
go run -race 06_concurrency/main.go
```

**Study method for each module:**
1. Read the file top to bottom — understand every comment
2. Run it — see the output
3. Modify a section — predict what changes
4. Break something intentionally — understand the error
5. Add your own experiments at the bottom

---

## Module Map

| Module | File | Topics | Priority |
|--------|------|---------|----------|
| 01 | `01_fundamentals/main.go` | Variables, types, operators, control flow, pointers, fmt | ⭐⭐⭐ MUST |
| 02 | `02_functions/main.go` | Functions, multiple returns, variadic, closures, defer, panic/recover, recursion | ⭐⭐⭐ MUST |
| 03 | `03_structs_methods_interfaces/main.go` | Structs, methods, interfaces, type assertion, type switch, embedding | ⭐⭐⭐ MUST |
| 04 | `04_error_handling/main.go` | error interface, custom errors, wrapping, errors.Is/As, sentinel errors | ⭐⭐⭐ MUST |
| 05 | `05_collections/main.go` | Arrays, slices (internals), maps, sorting, 2D slices | ⭐⭐⭐ MUST |
| 06 | `06_concurrency/main.go` | Goroutines, channels, select, WaitGroup, Mutex, atomic, worker pool, pipeline | ⭐⭐⭐ MUST |
| 07 | `07_packages_modules/main.go` | Package system, visibility, go modules, toolchain commands | ⭐⭐ HIGH |
| 08 | `08_standard_library/main.go` | fmt, strings, strconv, encoding/json, time, os, io, sort, regexp, net/url | ⭐⭐⭐ MUST |
| 09 | `09_generics/main.go` | Type parameters, constraints, generic types (Stack, Queue, Option, Result) | ⭐⭐ HIGH |
| 10 | `10_advanced_patterns/main.go` | Context, design patterns, reflection, circuit breaker, middleware, testing | ⭐⭐ HIGH |

---

## Learning Order (recommended)

### Week 1 — Foundation
- [ ] Module 01: Fundamentals
- [ ] Module 02: Functions
- [ ] Module 05: Collections
- [ ] Module 03: Structs, Methods, Interfaces

### Week 2 — Core Mastery
- [ ] Module 04: Error Handling
- [ ] Module 06: Concurrency
- [ ] Module 07: Packages & Modules

### Week 3 — Production Ready
- [ ] Module 08: Standard Library
- [ ] Module 09: Generics
- [ ] Module 10: Advanced Patterns

---

## Key Concepts to Master (Checklist)

### Fundamentals
- [ ] All 4 ways to declare variables
- [ ] Every built-in type and its zero value
- [ ] Difference between := and var
- [ ] iota and const blocks
- [ ] All loop forms (for, while-style, range, infinite)
- [ ] defer LIFO order and argument evaluation
- [ ] Pointer vs value (when to use each)

### Functions
- [ ] Multiple return values
- [ ] Named return values
- [ ] Variadic functions and spread operator
- [ ] Functions as first-class values
- [ ] Closures and variable capture
- [ ] panic / recover pattern

### Structs & Interfaces
- [ ] Struct embedding (composition over inheritance)
- [ ] Value receiver vs pointer receiver (and when to choose)
- [ ] Interface satisfaction (duck typing)
- [ ] Empty interface (any) and type assertion
- [ ] Type switch
- [ ] fmt.Stringer interface
- [ ] Functional options pattern

### Error Handling
- [ ] error interface
- [ ] errors.New vs fmt.Errorf
- [ ] Custom error types with fields
- [ ] Error wrapping with %w
- [ ] errors.Is (sentinel matching)
- [ ] errors.As (type matching)
- [ ] Multiple error collection

### Collections
- [ ] Slice internals: pointer + len + cap
- [ ] How append grows capacity
- [ ] Slice sharing the underlying array (and gotchas)
- [ ] copy() for independent slices
- [ ] Map zero value behavior (nil vs empty)
- [ ] Map as a set (map[T]struct{})
- [ ] sort.Slice with custom comparator

### Concurrency
- [ ] Goroutines vs OS threads
- [ ] sync.WaitGroup pattern
- [ ] Unbuffered vs buffered channels
- [ ] Channel directions (send-only, receive-only)
- [ ] select statement
- [ ] Done channel cancellation
- [ ] sync.Mutex (when to use vs channels)
- [ ] sync.RWMutex for read-heavy workloads
- [ ] atomic operations
- [ ] Worker pool pattern
- [ ] Pipeline pattern
- [ ] sync.Once for initialization
- [ ] Race detector (-race flag)

### Standard Library
- [ ] fmt: all format verbs
- [ ] strings: Builder, all manipulation functions
- [ ] strconv: Atoi, Itoa, ParseX, FormatX
- [ ] encoding/json: Marshal, Unmarshal, struct tags
- [ ] time: Now, Format (reference time!), Parse, Duration
- [ ] os: ReadFile, WriteFile, Open, Getenv, Args
- [ ] io: Reader, Writer, Copy, ReadAll
- [ ] bufio: Scanner, NewReader
- [ ] regexp: MustCompile, MatchString, FindAllString

### Advanced
- [ ] context.WithTimeout / WithCancel / WithDeadline / WithValue
- [ ] Context propagation through call chain
- [ ] reflect.TypeOf, reflect.ValueOf
- [ ] Generic type parameters and constraints
- [ ] Design patterns: Singleton, Observer, Middleware, Circuit Breaker
- [ ] Table-driven tests

---

## Tools to Install

```bash
# Formatter (essential)
go install golang.org/x/tools/cmd/goimports@latest

# Static analysis (catches bugs fmt misses)
go install honnef.co/go/tools/cmd/staticcheck@latest

# Better linter (covers many linters)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Security scanner
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Mock generator
go install github.com/vektra/mockery/v2@latest

# Enum string generator
go install golang.org/x/tools/cmd/stringer@latest

# LSP (IDE support)
go install golang.org/x/tools/gopls@latest
```

---

## After Completing All Modules

Build these projects to solidify your knowledge:

1. **CLI tool** — file organizer or system monitor (use `os`, `flag`/`cobra`)
2. **REST API** — CRUD server (use `net/http`, `encoding/json`, database)
3. **Concurrent scraper** — goroutines + channels + rate limiting
4. **In-memory cache** — generics + sync.RWMutex + TTL
5. **gRPC service** — protobuf + streaming
6. **Chat server** — WebSockets + goroutines + broadcast

---

## Reference: Go Proverbs (by Rob Pike)

- Don't communicate by sharing memory; share memory by communicating.
- Concurrency is not parallelism.
- Errors are values.
- Don't just check errors, handle them gracefully.
- Don't panic.
- Make the zero value useful.
- The bigger the interface, the weaker the abstraction.
- Accept interfaces, return concrete types.
- A little copying is better than a little dependency.
- Clear is better than clever.
- Documentation is for users.
