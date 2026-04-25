# Chapter 3 — Revision Checkpoint

## Questions

1. What four things on disk constitute a "working Go install"?
2. Why should you avoid distro Go packages?
3. Where does `go install` write its output by default? Why does
   that path need to be on `$PATH`?
4. What does `gopls` do, and how does your editor talk to it?
5. What's the difference between `GOTOOLCHAIN=auto` and
   `GOTOOLCHAIN=local`?
6. Name three tools every Go engineer should `go install` on day
   one.
7. In CI, what's the right way to pin the Go version?
8. Why is setting `GOROOT` manually almost always wrong?

## Answers

1. The toolchain (`/usr/local/go` or equivalent), the workspace
   (`$GOPATH`, default `~/go`), shell `PATH` entries that include
   the toolchain and `~/go/bin`, and an editor talking to `gopls`.

2. They lag the upstream release. Ubuntu LTS 22.04 ships Go 1.18;
   Debian stable ships 1.19. Both are unsupported. Modern features
   (generics, structured logging, `GOTOOLCHAIN`) are missing.

3. **`$GOBIN`** if set, otherwise **`$GOPATH/bin`** (default
   `~/go/bin`). It must be on `$PATH` so the shell can find tools
   you've installed (`gopls`, `golangci-lint`, etc.).

4. `gopls` is the official Go language server. It runs as a
   long-lived process; your editor sends LSP messages over
   stdin/stdout to request completions, hovers, diagnostics, and
   refactors. Every modern editor has an LSP client; the Go
   extension wires `gopls` up automatically.

5. **`GOTOOLCHAIN=auto`** (the default since 1.21): if a project's
   `go.mod` requires a newer Go than the one installed, the
   toolchain transparently downloads and runs that newer version.
   **`GOTOOLCHAIN=local`**: never download; always use the system
   toolchain, and refuse the build if it's too old. Use `local` in
   air-gapped or strict-CI environments.

6. `gopls`, `golangci-lint`, `govulncheck`. Optional but useful:
   `dlv` (debugger), `goreleaser`, `benchstat`, `staticcheck`.

7. `actions/setup-go@v5` with `go-version-file: go.mod`. That
   pins CI to whatever `go.mod` declares, so CI/local versions
   never drift.

8. The toolchain locates itself via the `go` binary's path; setting
   `GOROOT` manually only matters in unusual install layouts. If
   you find yourself needing it, something else is wrong (e.g. two
   Go installs on `$PATH`).
