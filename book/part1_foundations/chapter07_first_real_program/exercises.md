# Chapter 7 — Exercises

## Exercise 7.1 — Run all three versions

**Goal.** Internalize the progression from snippet to production
program.

**Task.** From the chapter folder:

```bash
go run ./examples/01_v1_minimal README.md
go run ./examples/02_v2_flags -l -w README.md
go run ./examples/03_v3_production/cmd/wc -l -w README.md
go run ./examples/03_v3_production/cmd/wc --version
go test -v ./examples/03_v3_production/internal/wc
```

Compare the output of v1 and v2/v3 on the same file. Confirm
they agree on counts (modulo flag-controlled output).

**Acceptance.** All three produce the same counts; tests pass.

---

## Exercise 7.2 — Add a test case

**Goal.** Touch the test surface; the rest of the book uses
testing extensively.

**Task.** Add one case to
[`examples/03_v3_production/internal/wc/wc_test.go`](examples/03_v3_production/internal/wc/wc_test.go):

```
{
    name: "your-name",
    input: "héllo\tworld\n",
    want: Stats{Lines: ?, Words: ?, Bytes: ?, Runes: ?},
}
```

Compute the expected values yourself (don't peek at the
implementation). Run `go test`. If it fails, decide who's wrong:
your math or the code.

**Acceptance.** A new green test case in the table.

---

## Exercise 7.3 ★ — Add a `--chars` (`-m`) flag

**Goal.** A small feature delta exercising the flag → display →
formatter pipeline.

**Task.** v3 already computes `Stats.Runes` and the formatter
already supports it (`Display.Runes`). Verify the wiring works:

```bash
go run ./examples/03_v3_production/cmd/wc -m README.md
go run ./examples/03_v3_production/cmd/wc -l -w -c -m README.md
```

If something doesn't render, add the wiring. Add a test that
runs with `-m` and confirms the rune count is what you expect.

---

## Exercise 7.4 ★★ — Add a `--json` flag

**Goal.** Touch `encoding/json`, which we'll go deep on in
Chapter 39.

**Task.** Add a `--json` flag to
[`examples/03_v3_production/cmd/wc/main.go`](examples/03_v3_production/cmd/wc/main.go).
When set, instead of the human-readable line, emit one JSON
object per file:

```json
{"file":"README.md","lines":120,"words":850,"bytes":5400,"runes":5400}
```

Keep the `internal/wc` package pure; do the JSON encoding in the
`cmd` layer or in `internal/wc/format.go`.

**Acceptance.** `--json` emits valid JSON (`jq .` accepts it);
the default human format still works.

---

## Exercise 7.5 ★★★ — Build a `head` clone

**Goal.** Apply the v3 layout pattern to a different `coreutils`
tool.

**Task.** Write a `head` clone with the same layout:

```
head/
├── cmd/head/main.go
├── cmd/head/version.go
└── internal/head/
    ├── head.go         # ReadFirstN(r io.Reader, n int) ([]string, error)
    └── head_test.go
```

Spec: print the first N (default 10) lines of each named file, or
of stdin if no files given. Support `-n N` to set the count.

**Acceptance.** Behaves like `head -n N` on common inputs;
includes tests; `go test ./...` clean.
