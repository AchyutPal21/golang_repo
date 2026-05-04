# Chapter 82 — Testing Fundamentals

Go's `testing` package is minimal by design. Combined with table-driven tests and subtests, it handles virtually every testing need without a separate framework.

## Table-driven tests

The idiomatic Go test pattern: define a slice of cases, iterate, call `t.Run` for each. Each subtest gets its own name and can be run in isolation (`go test -run TestFoo/case_name`).

```go
cases := []struct {
    name  string
    input int
    want  string
}{
    {"fizz", 3, "Fizz"},
    {"buzz", 5, "Buzz"},
    {"fizzbuzz", 15, "FizzBuzz"},
    {"number", 7, "7"},
}
for _, tc := range cases {
    tc := tc
    t.Run(tc.name, func(t *testing.T) {
        t.Parallel()
        if got := FizzBuzz(tc.input); got != tc.want {
            t.Errorf("FizzBuzz(%d) = %q, want %q", tc.input, got, tc.want)
        }
    })
}
```

## Test helpers

Use `t.Helper()` so failures point to the call site, not inside the helper.

```go
func assertEqual[T comparable](t testing.TB, got, want T, msg string) {
    t.Helper()
    if got != want {
        t.Errorf("%s: got %v, want %v", msg, got, want)
    }
}
```

`testing.TB` accepts both `*testing.T` and `*testing.B` — write helpers against the interface.

## Subtests and grouping

Group related cases under a common prefix:

```go
t.Run("Total/simple", ...)
t.Run("Total/discount", ...)
t.Run("Summary/contains_id", ...)
```

Run only a group: `go test -run TestOrder/Total`.  
Run parallel subtests: add `t.Parallel()` inside each subtest.

## Fixtures

Create shared test state in a constructor; reset between tests when the fixture is mutated.

```go
func newFixture(t *testing.T) *Fixture {
    t.Helper()
    // setup ...
    t.Cleanup(func() { /* teardown */ })
    return &Fixture{...}
}
```

## Golden files

Store expected output in `testdata/*.golden`. Pass `-update` flag to regenerate:

```go
var update = flag.Bool("update", false, "update golden files")

got := render(input)
golden := filepath.Join("testdata", t.Name()+".golden")
if *update {
    os.WriteFile(golden, []byte(got), 0644)
}
want, _ := os.ReadFile(golden)
if got != string(want) {
    t.Errorf("output mismatch — run with -update to regenerate")
}
```

## Fuzz testing (Go 1.18+)

```go
func FuzzAdd(f *testing.F) {
    f.Add(1, 2)        // seed corpus
    f.Fuzz(func(t *testing.T, a, b int) {
        result := Add(a, b)
        if result != a+b {
            t.Errorf("Add(%d, %d) = %d", a, b, result)
        }
    })
}
// go test -fuzz=FuzzAdd -fuzztime=30s
```

## Coverage

```bash
go test -coverprofile=cover.out ./...
go tool cover -html=cover.out      # browser view
go tool cover -func=cover.out      # per-function summary
```

Aim for coverage of **behaviour**, not lines. 100% line coverage with no edge-case assertions is worthless.

## Examples in this chapter

| File | Topic |
|------|-------|
| `examples/01_table_driven/main.go` | Table-driven tests, helpers, error cases |
| `examples/02_subtests/main.go` | Subtests, fixtures, golden files, property-based |
| `exercises/01_test_suite/main.go` | Full suite for a URL shortener — all patterns combined |
