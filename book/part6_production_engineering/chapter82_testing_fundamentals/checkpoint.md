# Chapter 82 Checkpoint — Testing Fundamentals

## Concepts to know

- [ ] What makes a test "table-driven"? What are the benefits over individual test functions?
- [ ] Why do you write `tc := tc` inside a range loop before `t.Run`? (Which Go version removed the need?)
- [ ] What does `t.Parallel()` do? When should you call it, and when should you avoid it?
- [ ] What is `t.Helper()` for? What happens if you omit it from a test helper?
- [ ] What is a golden file test? When is it useful vs. a plain assertion?
- [ ] What is a fuzz test? What input does the fuzzer generate beyond the seed corpus?
- [ ] What is `testing.TB`? Why write helpers against it instead of `*testing.T`?
- [ ] What does `t.Cleanup` do? How does it compare to `defer` inside a test?
- [ ] How do you run only one subtest from the command line?
- [ ] What does `go test -coverprofile` produce and how do you view it?

## Code exercises

### 1. Table-driven palindrome

Write table-driven tests for `IsPalindrome`. Include: empty string, single char, classic phrases with spaces/punctuation, non-palindromes.

### 2. Error case table

Write a table-driven test for a `ParseAge(s string) (int, error)` function. Cases should cover: valid integers, negative numbers, zero, non-numeric strings, and empty input.

### 3. Fixture with cleanup

Write a `newTempDB(t *testing.T) *DB` helper that:
- Creates an in-memory DB
- Registers `t.Cleanup` to reset it
- Calls `t.Helper()` so test failure lines point to the caller

### 4. Golden file

Write a `Render(order *Order) string` function and a golden file test for it. Show both the normal path and the `-update` path in comments.

## Quick reference

```bash
# Run all tests
go test ./...

# Run specific test
go test -run TestFizzBuzz ./...

# Run specific subtest
go test -run "TestFizzBuzz/fizz" ./...

# Run with race detector
go test -race ./...

# Coverage
go test -coverprofile=c.out ./...
go tool cover -html=c.out

# Fuzz
go test -fuzz=FuzzFoo -fuzztime=30s

# Verbose output
go test -v ./...

# Update golden files (custom flag)
go test -run TestGolden -update ./...
```

```go
// Table test skeleton
func TestFoo(t *testing.T) {
    cases := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid", "hello", "HELLO", false},
        {"empty", "", "", false},
    }
    for _, tc := range cases {
        tc := tc
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            got, err := Foo(tc.input)
            if (err != nil) != tc.wantErr {
                t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
            }
            if got != tc.want {
                t.Errorf("got %q, want %q", got, tc.want)
            }
        })
    }
}
```

## What to remember

- Name subtests as `Group/Case` — enables targeted `go test -run` filtering.
- Always call `t.Helper()` in assertion helpers — failure lines point to the caller.
- `t.Fatalf` stops the current subtest; `t.Errorf` continues (use Fatal when continuing makes no sense).
- Golden files are ideal when output is multi-line prose or JSON that's hard to write as a string literal.
- Fuzz tests find edge cases you didn't think of — run them in CI with a time budget (`-fuzztime=60s`).
- Coverage measures line execution, not correctness — complement with assertion-rich tests.
