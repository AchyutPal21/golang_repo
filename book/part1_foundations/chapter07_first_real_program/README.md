# Chapter 7 — Your First Real Program: a CLI Word Counter

> **Reading time:** ~32 minutes (8,000 words). **Code:** 3 versions
> of the same program, ~600 lines total. **Target Go version:** 1.22+.
>
> The capstone of Part I. We build the same program — a `wc` clone —
> three times: a 30-line minimal version, a 100-line flag-driven
> version, a production-shaped 250-line version with proper layout,
> tests, and a `--version` flag. By the end, you've touched packages,
> imports, errors, slices, strings, I/O, flags, structured logging,
> and tests — every concept Part I has surfaced — in service of one
> useful program.

---

## 1. Concept Introduction

A *real* Go program is not a snippet. It has:

* A clear command-line surface (flags, args, stdin/stdout, exit codes).
* A clean separation between the entry point (`main`) and the logic
  (testable functions/methods elsewhere).
* Error handling at every step: messages aimed at humans, exit codes
  aimed at scripts.
* Tests for the logic, runnable with `go test`.
* A `--version` (or `-v`) flag that says what's running.
* A README that tells someone unfamiliar how to use it.

This chapter walks you through the construction of one such program
in three increasing-fidelity passes. The pattern carries across
every Go CLI you'll write.

> **Working definition:** a "real" Go program is a single binary
> with a clear UNIX-tool interface, layered so the logic is
> separately testable, the entry point is a thin shell, and
> production concerns (errors, signals, exit codes, version) are
> first-class. Anything less is a snippet.

---

## 2. Why This Exists

Most tutorials end at "hello world" or "fibonacci." That's a poor
preparation for real work. Real Go programs are not 20 lines; they're
a few hundred lines split across a handful of files, with a proper
layout and tests. The shape of even a small CLI tool reveals patterns
you'll use in services: separating I/O from logic, parsing
configuration, structured errors, unit tests against pure functions.

A `wc` clone is the right vehicle because:

1. The spec is short enough to write from memory.
2. It exercises strings, bytes, runes, slices, file I/O, stdin
   handling, exit codes, flags.
3. It has interesting edge cases (UTF-8 vs bytes, very-large files,
   piped input, multiple file args).
4. It's testable: pure logic plus thin I/O layer.
5. Real `wc` is in your `coreutils` package — you can compare.

---

## 3. Problem It Solves

This chapter teaches you to:

1. **Lay out a single-binary CLI tool** in idiomatic Go.
2. **Parse command-line flags** with the standard `flag` package.
3. **Read from stdin or files** using the `io.Reader` abstraction.
4. **Stream large files** without loading them into memory.
5. **Return correct exit codes** for shell pipelines.
6. **Write tests for the pure logic** of the tool.
7. **Stamp a `--version`** flag with `runtime/debug.ReadBuildInfo`.
8. **Ship a useful artifact** — the `wc` clone is genuinely usable.

---

## 4. Historical Context

`wc` (word count) was one of the original UNIX utilities, written
by Ken Thompson and Dennis Ritchie in the early 1970s. The original
C implementation is ~50 lines. Every UNIX-like system since has
shipped a version. The classic flag set: `-l` (lines), `-w`
(words), `-c` (bytes), `-m` (characters/runes).

It's the perfect first-real-program because:

* The spec is *finished* — no decisions to make about features.
* The logic is *small* but not *trivial*.
* The edge cases (UTF-8, large files, piped input) are real
  systems-programming concerns.
* You can `diff` your output against `/usr/bin/wc` to verify.

There's also a meta-historical point: Ken Thompson is one of Go's
designers. Building `wc` in Go is a small homage to the language's
heritage.

---

## 5. How Industry Uses It

CLIs like the one we're building are everywhere in production Go:

* `kubectl`, `gh`, `terraform`, `helm`, `cosign`, `goreleaser` —
  all are single-binary Go CLIs.
* Internal ops tools at every Go shop — health checks, schema
  diff tools, migration runners, log filters.
* "One-shot operators" in Kubernetes — Job containers that run a
  Go binary to do one thing and exit.
* Sidecar agents — small Go binaries deployed next to a main
  service to handle one concern (log shipping, metric scraping).

The shape of every one of these is the same: thin `main`, logic
in a package, flags, signals, structured errors, exit codes. Once
you've built one, you've built the template for hundreds.

---

## 6. Real-World Production Use Cases

**Migration runner.** A team needs a one-shot binary that runs
database migrations and exits cleanly. They write a 200-line Go
CLI that takes a connection string from `--db` (or `DATABASE_URL`),
applies up-migrations from a directory, and exits 0 on success or
1 on failure. Deployed as a Kubernetes Job. Same shape as `wc`,
different verbs.

**Log filter.** A team's log volume is too high; they pre-filter
logs at the source with a 100-line Go CLI that reads stdin, drops
known-noise lines, and writes filtered output to stdout. Deployed
as a sidecar.

**Schema-diff tool.** A team writes a 300-line CLI that compares
two GraphQL schema files and reports breaking changes. Run in CI
on every PR; fails the build on breaking changes.

**Health probe.** A 50-line CLI that hits an HTTP endpoint and
exits 0 if the response is 2xx, 1 otherwise. Used by Docker
HEALTHCHECK, Kubernetes liveness probes.

**Self-update tool.** A 250-line CLI that downloads a newer
version of itself, verifies the signature with `cosign`, replaces
the binary in place. Used by every Go tool that does
self-upgrade.

All of these have the same shape as `wc`. Once you can build
`wc`, you can build any of them.

---

## 7. Beginner-Friendly Explanation

A CLI program in Go has four moving parts:

1. **`func main`** — the entry point. Should be 5–20 lines:
   parse flags, set up resources, call into the real logic, exit
   with the right code.
2. **A logic function**, defined in another package or another
   file. Takes its inputs explicitly, returns `(result, error)`.
   Easy to test.
3. **Flag parsing.** The standard library's `flag` package gives
   you `-x`/`--x`/`-y value`/`--help` for free.
4. **I/O layer.** Reads from `os.Stdin` or files; writes to
   `os.Stdout` or files. Errors go to `os.Stderr`.

Three rules of thumb:

* **`main` shouldn't have business logic.** It's wiring.
* **Errors print to stderr; data prints to stdout.** This lets
  pipelines work: `mytool < input | wc` should not see error
  messages mixed into the data.
* **Exit non-zero on error.** Otherwise shell scripts can't tell
  success from failure.

If you internalize these three rules, your first hundred Go CLIs
will look almost the same. That's a feature.

---

## 8. Deep Technical Explanation

### 8.1. The minimal version (`v1`)

A 30-line `wc` clone for a single file:

```go
package main

import (
    "bufio"
    "fmt"
    "os"
)

func main() {
    if len(os.Args) != 2 {
        fmt.Fprintln(os.Stderr, "usage: v1 <file>")
        os.Exit(2)
    }
    f, err := os.Open(os.Args[1])
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    defer f.Close()

    var lines, words, bytes int
    sc := bufio.NewScanner(f)
    sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
    for sc.Scan() {
        line := sc.Bytes()
        lines++
        bytes += len(line) + 1 // +1 for the newline `Scanner` strips
        for _, r := range string(line) {
            _ = r // each rune doesn't matter; we count words
        }
        // a word is a maximal run of non-whitespace characters
        inWord := false
        for _, b := range line {
            if b == ' ' || b == '\t' {
                inWord = false
            } else if !inWord {
                inWord = true
                words++
            }
        }
    }
    if err := sc.Err(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    fmt.Printf("%d %d %d %s\n", lines, words, bytes, os.Args[1])
}
```

Notes:

* Streams the file, line by line, with `bufio.Scanner`. Doesn't
  load the whole file into memory.
* `sc.Buffer(..., 1024*1024)` raises the max line size to 1 MB.
  The default is 64 KB and a long line will fail.
* Exit code 2 for usage errors, 1 for runtime errors. This
  matches conventions used by `grep`, `awk`, `wc`.
* Errors go to `os.Stderr`; the count goes to `os.Stdout`.

This works. It's also wrong as a production tool: no flags, no
multi-file support, can't read stdin, can't output specific
counts. We'll fix that in v2 and v3.

### 8.2. Version 2: flags and stdin

```go
flag.BoolVar(&showLines, "l", false, "show line count")
flag.BoolVar(&showWords, "w", false, "show word count")
flag.BoolVar(&showBytes, "c", false, "show byte count")
flag.BoolVar(&showRunes, "m", false, "show rune count")
flag.Parse()

// If none specified, default to all four (matches GNU wc).
if !(showLines || showWords || showBytes || showRunes) {
    showLines, showWords, showBytes = true, true, true
}

args := flag.Args()
if len(args) == 0 {
    // Read from stdin.
    count(os.Stdin, "")
} else {
    for _, name := range args {
        f, err := os.Open(name)
        ...
    }
}
```

The standard `flag` package handles `-l`, `--l`, `-l=true`,
`--help` for free. It does *not* support `--long-flag` aliasing of
short flags out of the box; if you need both, use `pflag` or
`cobra`. For a `wc` clone, the stdlib is sufficient.

Reading from stdin when no file is given is the *unix-pipe-friendly*
behavior. `cat input | wc -w` should work.

### 8.3. Version 3: production layout

The production version splits the program across files and
introduces the `cmd/` + `internal/` pattern from Chapter 4:

```
chapter07_first_real_program/examples/03_v3_production/
├── cmd/
│   └── wc/
│       ├── main.go          # tiny: parse flags, call into internal/wc
│       └── version.go       # the --version flag implementation
└── internal/
    └── wc/
        ├── wc.go            # the Counter type and Count function
        ├── wc_test.go       # tests for the pure logic
        └── format.go        # output formatting
```

Why this layout:

* `main` is 30 lines: parse, dispatch, exit. No logic.
* The `wc` package is pure: `Count(io.Reader) (Stats, error)`.
  Easy to test, easy to embed elsewhere.
* Output formatting is separate, so the same logic can drive a
  CLI table, JSON, or whatever.
* Tests run on the package: `go test ./internal/wc`.

### 8.4. Streaming vs reading-all

For an arbitrary file, you can't `os.ReadFile(name)` — the file
might be 10 GB. Streaming with `bufio.Scanner` is the right
default.

But scanners read line-by-line, which means you can't easily count
*runes* (you have to scan the file twice unless you count both
together). The right pattern: scan once, accumulate all four
counts inline.

```go
sc := bufio.NewScanner(r)
sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
for sc.Scan() {
    line := sc.Bytes()
    s.Lines++
    s.Bytes += int64(len(line)) + 1
    s.Runes += int64(utf8.RuneCount(line)) + 1
    inWord := false
    for _, b := range line {
        if b == ' ' || b == '\t' {
            inWord = false
        } else if !inWord {
            inWord = true
            s.Words++
        }
    }
}
```

One pass, four counts. The +1s are for the stripped newline.

### 8.5. UTF-8: bytes vs runes

`wc -c` counts bytes; `wc -m` counts characters. In ASCII files
they're identical. In UTF-8 files (almost any text in a real
codebase), `wc -m` is smaller because multi-byte runes count as
one character.

Go's stdlib does the right thing if you reach for the right tool:

* `len(line)` — bytes in a `[]byte`.
* `utf8.RuneCount(line)` — runes in a `[]byte`.
* `len(s)` — bytes in a `string`.
* `utf8.RuneCountInString(s)` — runes in a `string`.

Don't use `for range string(line)` to count runes — it works, but
you allocate a string for the conversion. `utf8.RuneCount(line)` is
the zero-allocation form.

### 8.6. Exit codes and error handling

Convention (matches `grep`, `wc`, `awk`):

* **0** — success.
* **1** — runtime error (file not found, I/O failed, parse
  error).
* **2** — usage error (bad flags, wrong number of args).

The pattern in `main`:

```go
if err := run(); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
}
```

Don't sprinkle `os.Exit` calls deep in the code; centralize the
exit logic in `main`. Calling `os.Exit` skips deferred functions —
file handles don't close, buffers don't flush — so it should only
happen at the top level.

### 8.7. The `--version` flag

Every production CLI should have one. Use
`runtime/debug.ReadBuildInfo` for VCS info; use `-ldflags` for the
human-readable version string.

```go
var version = "dev"

func printVersion() {
    fmt.Printf("wc %s", version)
    if info, ok := debug.ReadBuildInfo(); ok {
        for _, s := range info.Settings {
            if s.Key == "vcs.revision" {
                fmt.Printf(" (%s)", s.Value[:8])
            }
        }
    }
    fmt.Printf(" %s\n", runtime.Version())
}
```

### 8.8. Tests for pure logic

The whole point of the `cmd/` + `internal/` split is testability.
The `Count(io.Reader)` function is easy to test:

```go
func TestCount_basic(t *testing.T) {
    r := strings.NewReader("hello world\nfoo bar\n")
    s, err := Count(r)
    if err != nil {
        t.Fatalf("Count: %v", err)
    }
    if s.Lines != 2 || s.Words != 4 {
        t.Errorf("got %v, want lines=2 words=4", s)
    }
}
```

Run with `go test ./internal/wc`. We'll go deep on testing in
Chapter 82; for now, table-driven tests are the idiom:

```go
func TestCount_table(t *testing.T) {
    cases := []struct {
        name  string
        input string
        want  Stats
    }{
        {"empty", "", Stats{}},
        {"one line", "hello\n", Stats{Lines: 1, Words: 1, Bytes: 6, Runes: 6}},
        {"utf-8", "héllo\n", Stats{Lines: 1, Words: 1, Bytes: 7, Runes: 6}},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got, err := Count(strings.NewReader(tc.input))
            if err != nil {
                t.Fatalf("Count: %v", err)
            }
            if got != tc.want {
                t.Errorf("got %v want %v", got, tc.want)
            }
        })
    }
}
```

---

## 9. Internal Working (How Go Handles It)

* **`os.Stdin`, `os.Stdout`, `os.Stderr`** are package-level
  `*os.File` values, opened by the runtime on program start.
  Mutating them (e.g. for tests) is fine but uncommon; pass the
  reader/writer as a parameter instead.
* **`bufio.Scanner`** wraps an `io.Reader` and buffers reads.
  The buffer grows on demand up to a configurable max. The
  default `SplitFunc` is `ScanLines`, which strips the newline.
* **`flag.Parse()`** populates the `flag.CommandLine` global
  with parsed values. Subsequent `flag.Args()` returns the
  positional args left over.
* **`os.Exit(code)`** calls the runtime's exit syscall. It does
  *not* run deferred functions. Don't use it deep in the code.
* **`utf8.RuneCount`** walks the input byte-by-byte, decoding
  each rune. Linear time, zero allocations.

---

## 10. Syntax Breakdown

We've already seen the key idioms; here they are in one block as
reference:

```go
// Flag declaration:
var verbose bool
flag.BoolVar(&verbose, "v", false, "verbose output")

// Stream a file:
f, err := os.Open(name)
if err != nil { ... }
defer f.Close()

// Stream stdin or a file with one type:
var r io.Reader = os.Stdin
if name != "" { r, _ = os.Open(name) }

// Read line-by-line:
sc := bufio.NewScanner(r)
sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
for sc.Scan() {
    line := sc.Bytes()
    // ...
}
if err := sc.Err(); err != nil { ... }

// Print to stderr / stdout:
fmt.Fprintln(os.Stderr, "error:", err)
fmt.Fprintf(os.Stdout, "%d %d %d %s\n", lines, words, bytes, name)

// Exit cleanly:
os.Exit(1)
```

---

## 11. Multiple Practical Examples

Three versions of the same program, in increasing fidelity.

### Example 1 — `examples/01_v1_minimal`

```bash
go run ./examples/01_v1_minimal /etc/passwd
```

Single file, no flags, no stdin. ~30 lines. The "smallest useful
`wc`" baseline.

### Example 2 — `examples/02_v2_flags`

```bash
go run ./examples/02_v2_flags -l -w README.md
echo "hello world" | go run ./examples/02_v2_flags -w
```

Flags, multi-file, stdin support. ~110 lines.

### Example 3 — `examples/03_v3_production`

The production-shaped version. Layout:

```
03_v3_production/
├── cmd/wc/main.go       # entry point
├── cmd/wc/version.go    # --version flag
└── internal/wc/
    ├── wc.go            # Counter + Count function
    ├── wc_test.go       # table-driven tests
    └── format.go        # output formatting
```

Build and run:

```bash
go run ./examples/03_v3_production/cmd/wc -l -w -m README.md
go test ./examples/03_v3_production/internal/wc
go run ./examples/03_v3_production/cmd/wc --version
```

You'll come back to this layout for every CLI you write.

---

## 12. Good vs Bad Examples

**Good `main`:**

```go
func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    flag.Parse()
    // ... real logic, returning error ...
    return nil
}
```

**Bad:**

```go
func main() {
    flag.Parse()
    f, err := os.Open(...)
    if err != nil {
        fmt.Println(err) // wrong: stdout, not stderr
        os.Exit(1)
    }
    // ... 100 lines of logic inline in main ...
}
```

Why bad: `main` is doing the work; the logic is untestable; errors
print to stdout, polluting pipelines.

**Good error handling:**

```go
f, err := os.Open(name)
if err != nil {
    return fmt.Errorf("open %s: %w", name, err)
}
defer f.Close()
```

The `%w` wraps the error so callers can `errors.Is`/`errors.As`
the underlying type (we'll cover this in Chapter 36).

**Bad:**

```go
f, _ := os.Open(name)        // ignored error
defer f.Close()              // panics on nil if open failed
// ... use f ...
```

Ignoring errors is a class of bug `golangci-lint` will flag with
`errcheck`. Don't.

---

## 13. Common Mistakes

1. **Reading the whole file into memory.** `os.ReadFile` is fine
   for small files; for arbitrary files, stream.
2. **Forgetting to close files.** `defer f.Close()` *immediately*
   after a successful `os.Open`. Don't defer in a loop without
   thinking — closures capture by reference.
3. **Using `fmt.Println` for errors.** Errors go to `os.Stderr`
   via `fmt.Fprintln(os.Stderr, ...)`. Otherwise pipes break.
4. **Calling `os.Exit` deep in the code.** Skips deferred
   cleanup. Centralize in `main`.
5. **Not setting `bufio.Scanner` buffer size.** Default 64 KB
   max; long lines fail with `bufio.Scanner: token too long`.
6. **Counting bytes by `utf8.RuneCountInString`.** Wrong tool;
   that counts runes. Use `len()` for bytes.
7. **Mixing the count functions and the I/O.** Hard to test.
   Keep `Count(io.Reader)` pure; do I/O in `main`.
8. **Using `flag.PrintDefaults()` with a custom `Usage`.**
   Either set `flag.Usage = ...` and call it yourself, or rely
   on the default. Don't half-customize.
9. **Forgetting `--version`.** The first time you have to triage
   a production incident, you'll wish for it.
10. **Defaulting flags to "all true."** Match the GNU `wc`
    convention: if the user specifies no flags, show line/word/
    byte. If they specify any, show only those.

---

## 14. Debugging Tips

* **Run with `-race`** if you add goroutines: `go run -race
  ./examples/03_v3_production/cmd/wc ...`.
* **Compare to `/usr/bin/wc`.** Run both on the same file and
  diff the output. The byte count should match exactly; word
  counts should match for ASCII; rune counts may differ
  depending on `wc -m` semantics.
* **`bufio.Scanner` errors are silent** unless you check `sc.Err()`
  after the loop. Always check.
* **`go test -v ./...`** in the chapter folder runs all tests
  with verbose output.
* **`go test -run TestCount_basic ./internal/wc`** runs one test
  by name. Useful for tight iteration.

---

## 15. Performance Considerations

For a `wc` clone, performance is real:

* **Streaming with `bufio.Scanner`** keeps memory bounded.
* **`utf8.RuneCount`** is O(n) but zero-allocation.
* **Large files**: the bottleneck is disk read. The Go program
  is typically within 2x of `/usr/bin/wc` on large files.
* **Many small files**: `os.Open` overhead dominates. If you
  cared, you'd open in parallel with goroutines, but that's
  rarely worth it for a CLI.

For benchmarking, write a `Benchmark` function:

```go
func BenchmarkCount(b *testing.B) {
    data := bytes.Repeat([]byte("hello world\n"), 100000)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = Count(bytes.NewReader(data))
    }
}
```

`go test -bench=. ./internal/wc`. We'll cover benchmarking
properly in Chapter 86.

---

## 16. Security Considerations

Even a CLI tool has security concerns:

* **Untrusted file paths.** If your tool accepts paths from a
  config file or env var, beware of `../../etc/passwd`-style
  inputs. Use `filepath.Clean` and reject paths outside the
  expected root.
* **Symlinks.** `os.Open` follows symlinks. If your tool walks a
  directory tree, decide explicitly whether to follow.
* **Binary data.** A "word counter" run on `/dev/urandom` should
  not crash; it should report a count and exit. Test for it.
* **Resource limits.** Reading from stdin without a size limit
  is fine for `wc`; for tools that buffer, set explicit size
  caps.
* **Signal handling.** A long-running CLI should exit cleanly on
  Ctrl-C. We covered this pattern in Chapter 1's HTTP server;
  for a one-shot CLI it usually doesn't matter.

---

## 17. Senior Engineer Best Practices

1. **Pure logic in a separate package**, called from `main`.
2. **`main` returns nothing**; wrap real logic in `run() error`
   and call `os.Exit` only at the top.
3. **Errors to stderr; data to stdout.** Always.
4. **Exit codes follow convention** (0/1/2 for success/runtime/
   usage).
5. **Stream, don't slurp.** `os.ReadFile` is for small known-
   size files; otherwise `io.Reader`.
6. **Unbuffered output is fine** for short CLIs; for high-volume
   output, wrap stdout in a `bufio.Writer` and `Flush` before exit.
7. **`--version` as a flag** with build info.
8. **Tests live next to logic** as `_test.go` files; the
   `cmd/<name>/main.go` files don't need tests.
9. **Document `--help`** with `flag.Usage` if the default isn't
   enough.
10. **Match the conventions of similar tools.** A `wc` clone
    should accept the same flags as `wc`; don't reinvent for the
    sake of it.

---

## 18. Interview Questions

1. *(junior)* In a Go CLI, where do error messages go: stdout
   or stderr? Why?
2. *(junior)* What does `defer f.Close()` do?
3. *(mid)* What's the difference between `bufio.Scanner`'s
   `Bytes()` and `Text()`?
4. *(mid)* What's the difference between `len(s)` and
   `utf8.RuneCountInString(s)`?
5. *(senior)* Why is calling `os.Exit` deep in the code an
   anti-pattern?
6. *(senior)* You're writing a CLI that processes a file. The
   file might be 10 GB. How do you read it?
7. *(senior)* How do you implement `--version` in a Go CLI?
8. *(staff)* Describe the layout for a production-grade Go CLI
   with multiple subcommands, tests, and configuration. Defend
   each top-level directory.

---

## 19. Interview Answers

1. **Stderr.** That way the data can flow through pipelines
   (`tool | other-tool`) without being polluted by error text.
   Many UNIX tools rely on this; mixing the two breaks
   composability.

2. `defer f.Close()` schedules `f.Close()` to run when the
   surrounding function returns, regardless of how it returns
   (normal return or panic). Used to guarantee resource cleanup
   without `try/finally`.

3. **`Bytes()`** returns `[]byte` pointing into the scanner's
   internal buffer — *valid only until the next `Scan` call*,
   no allocation. **`Text()`** returns a fresh `string` —
   allocates, but is safe to keep. Use `Bytes` for hot loops;
   `Text` for keeping the value across iterations.

4. **`len(s)`** is the byte length of `s`. **`utf8.RuneCountInString(s)`**
   is the rune count, which is smaller for multi-byte UTF-8.
   For ASCII they're equal. The right tool depends on whether
   you're measuring storage (bytes) or "user-visible characters"
   (runes — though even runes ≠ grapheme clusters for some
   scripts).

5. Because `os.Exit` doesn't run deferred functions. Files don't
   close, buffers don't flush, locks don't release. Centralize
   exit in `main`: have all your logic return `error`, let
   `main` handle the exit code in one place.

6. Stream it. Open with `os.Open`, wrap in `bufio.NewScanner` for
   line-oriented work or `bufio.NewReader` for byte-oriented
   work. Set the scanner's buffer size if lines might be
   long. Process one line/chunk at a time. Memory stays bounded.

7. Best: define a `--version` flag, and on activation print
   `runtime.Version()`, the binary's version string (set via
   `-ldflags '-X main.version=...'`), and (since 1.18) the VCS
   commit from `runtime/debug.ReadBuildInfo`.

8. Top-level: `cmd/<binary>/main.go` for each binary;
   `internal/<domain>/` for shared logic; tests next to logic
   files; `go.mod`/`go.sum` at the root; optional `examples/`,
   `docs/`, `Makefile`. Defend: `cmd/` makes multi-binary
   builds explicit; `internal/` enforces module-private API;
   no `pkg/` because public packages live at the root if they
   exist at all. The deliberate emptiness of "no framework, no
   DI container, no service registry" is itself a defense — Go's
   simplicity scales for a CLI.

---

## 20. Hands-On Exercises

**Exercise 7.1 — Run all three versions.** Run each of the three
example versions on the same file. Confirm they produce the same
counts (modulo the flag-controlled output of v2/v3).

**Exercise 7.2 — Write a test.** Add one test case to
`examples/03_v3_production/internal/wc/wc_test.go`:

* Input: a 4-line file containing one tab and one Unicode
  character (e.g. "héllo\tworld\n").
* Verify the byte/rune/word/line counts.

Run with `go test -v ./examples/03_v3_production/internal/wc`.

**Exercise 7.3 ★ — Add a `--chars` flag.** Match GNU `wc -m`:
add a `-m` flag that prints the rune count. The plumbing is
already there in v3 (`Stats.Runes` is computed); you only need
the flag and the formatter update.

**Exercise 7.4 ★★ — Add structured output.** Add a `--json` flag
that emits the stats as a JSON object instead of a table. Use
`encoding/json` (we'll go deep on it in Chapter 39, but the
basics are short).

---

## 21. Mini Project Tasks

**Task — Rewrite a real coreutils tool.** Pick one of `head`,
`tail`, `cat`, `tr`, or `cut` (all 50–500 lines of C in
coreutils). Build a Go CLI that matches its behavior on the
common cases. Constraints:

* Stdlib only.
* `cmd/<name>/main.go` plus `internal/<name>/<name>.go` layout.
* Tests for the pure logic.
* `--version` flag.
* `go test -race ./...` clean.

**Acceptance.** A working tool you'd actually use, plus the
conviction that you can build any CLI you need.

---

## 22. Chapter Summary

* A "real" Go CLI is layered: a thin `main`, pure logic in
  `internal/<name>`, flags via the `flag` package, tests for
  the logic.
* Stream files; don't slurp. Use `bufio.Scanner` for line-
  oriented, `bufio.Reader` for byte-oriented work.
* Errors to stderr; data to stdout. Exit codes follow UNIX
  conventions (0/1/2).
* `--version` should always be present, populated from
  `-ldflags` plus `runtime/debug.ReadBuildInfo`.
* Tests live in `_test.go` files next to the logic. Table-
  driven tests are the idiom.
* The `cmd/<name>/main.go` + `internal/<name>/...` layout is the
  template you'll reuse for every CLI you write.

Updated working definition: *a "real" Go program is a single
binary with a clear UNIX-tool interface, layered so the logic is
separately testable, the entry point is a thin shell, and
production concerns (errors, signals, exit codes, version) are
first-class. Anything less is a snippet.*

This closes Part I. You've now seen *what* Go is, *why* it
exists, *how* its toolchain and ecosystem fit together, *where*
projects live on disk, and *what* a real Go program looks like.
Part II walks every keyword and primitive in the language at the
same depth.

---

## 23. Advanced Follow-up Concepts

* **`spf13/cobra`** — the de facto subcommand framework for
  larger CLIs (used by `kubectl`, `gh`, `helm`). Worth using once
  you outgrow `flag`.
* **`spf13/pflag`** — POSIX-style long flags (`--verbose`).
  Drop-in replacement for `flag` in many projects.
* **`charmbracelet/bubbletea`** — TUI framework for interactive
  CLIs (think `htop`-style interfaces). Pure Go.
* **`urfave/cli`** — alternative to Cobra; less feature-rich,
  smaller surface.
* **`goreleaser`** documentation — the canonical tool for
  shipping Go CLIs to Homebrew, GitHub releases, scoop, apt.
* **The `coreutils` source** at `gnu.org/software/coreutils` —
  read `wc.c`, `cat.c`, `head.c` for ideas of "what the spec
  actually says."

> **End of Chapter 7.** Move on to [Chapter 8 — Variables,
> Constants, and the Zero Value
> ](../../part2_core_language/chapter08_variables_constants_zero/README.md).
