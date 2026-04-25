# Chapter 7 — Revision Checkpoint

## Questions

1. What goes in `cmd/<name>/main.go` vs. `internal/<name>/`?
2. Why should error messages go to stderr and data to stdout?
3. What exit codes does a UNIX-tool-style CLI use, and what does
   each mean?
4. When should you call `os.Exit` deep in your code? What's the
   alternative?
5. What does `bufio.Scanner` do that you couldn't do with
   `os.ReadFile`?
6. How do you count UTF-8 runes vs bytes in Go?
7. Where would you add a `--version` flag, and what should it
   print?
8. Why is the table-driven test pattern preferred over many
   single-case test functions?

## Answers

1. **`cmd/<name>/main.go`** is the entry point: parse flags,
   wire dependencies, call into the logic, exit with the right
   code. It should be tiny — typically 30–80 lines. **`internal/<name>/`**
   holds the pure logic: the types, the algorithms, the I/O
   abstractions. Tests live there. Keeping logic out of `main`
   makes it testable.

2. So shell pipelines work. `mytool < input | other-tool`
   should not see error messages mixed into the data stream.
   Many UNIX tools rely on this convention; mixing them breaks
   composability.

3. **0** = success. **1** = runtime error (file not found,
   I/O failed). **2** = usage error (bad flags, wrong arg
   count). Some tools use higher numbers for specific failure
   modes; consistency within a tool's docs is what matters.

4. **Almost never.** `os.Exit` skips deferred functions, so
   files don't close, buffers don't flush. **Alternative:**
   wrap your logic in a `run() error` function; have `main`
   call it and `os.Exit(1)` only at the top level if `err !=
   nil`.

5. **Stream the file** — read it in chunks, not all at once.
   `os.ReadFile` reads the whole file into memory; for a 10 GB
   file that's a problem. `bufio.Scanner` reads line by line
   (or by your custom `SplitFunc`), keeping memory bounded.

6. **`len(s)`** for bytes (where `s` is `string` or `[]byte`).
   **`utf8.RuneCount(b)`** for runes in `[]byte`.
   **`utf8.RuneCountInString(s)`** for runes in a `string`. For
   ASCII they're identical; for UTF-8 the rune count is smaller.

7. **In every CLI you ship**, even a tiny one. It should print
   the human-readable version (set via
   `-ldflags '-X main.version=...'`), the Go version
   (`runtime.Version()`), and (since 1.18) the VCS commit and
   dirty-state from `runtime/debug.ReadBuildInfo`.

8. **One test function per behavior, not per case.** Table-
   driven tests scale: adding a case is one struct literal, not
   a new `func Test...`. Subtests via `t.Run(tc.name, ...)`
   give you independent failures (one bad case doesn't mask the
   others). The pattern is so dominant in Go that linters can
   recognize and check it.
