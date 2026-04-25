# Chapter 2 — Exercises

## Exercise 2.1 — Audit your install

**Goal.** Read your Go install end-to-end and explain every value.

**Task.** Run:

```bash
go run ./exercises/01_audit_install
```

The program prints `GOROOT`, `GOPATH`, `GOPROXY`, `GOSUMDB`, the
build cache path and size, the module cache path and size, and the
list of installed tools in `$GOBIN`.

For each value, write a one-line answer to: *what is this, why does
it have the value it does, what would happen if it were unset?*

**Acceptance.** You can answer those three questions for every line
of output without looking it up.

---

## Exercise 2.2 — Set up `golangci-lint` against the book

**Goal.** Stand up the de-facto linter against this book repo.

**Task.**

1. Install: `go install
   github.com/golangci/golangci-lint/cmd/golangci-lint@latest`.
2. Add `~/go/bin` to `$PATH` if you haven't (Common Mistake #1
   from the chapter).
3. From the book root: `golangci-lint run ./...`.
4. Read the output. If there are findings, fix them or open an
   issue.

**Acceptance.** Zero findings, or an issue describing each finding
left open.

**Stretch.** Add a `.golangci.yml` at the book root that enables
`gosec`, `staticcheck`, `revive`, `errcheck`, and `gosimple`. Re-run.

---

## Exercise 2.3 ★ — Run `govulncheck` and read the report

**Goal.** Understand the Go vulnerability scanner, in practice.

**Task.**

1. Install: `go install
   golang.org/x/vuln/cmd/govulncheck@latest`.
2. From the book root: `govulncheck ./...`.
3. If there are findings, look up each one on `vuln.go.dev` and
   write a one-paragraph summary of what the vulnerability is, what
   triggers it, and how the report decided whether your code reaches
   it.

**Acceptance.** A short writeup per finding, or "no findings,
clean."

**Stretch.** Add a GitHub Actions workflow that runs `govulncheck`
weekly and on every push to `main`, opening an issue on findings.

---

## Exercise 2.4 ★ — Read the standard library

**Goal.** Make `go doc` your first instinct.

**Task.** Pick three packages from the inventory printed by Example
3 (`go run ./examples/03_stdlib_inventory`) that you've never used.
For each one, run `go doc <pkg>`, read the package overview, and
write down two functions or types you'd reach for in real code.

**Acceptance.** A short list. The point is the habit, not the list.
