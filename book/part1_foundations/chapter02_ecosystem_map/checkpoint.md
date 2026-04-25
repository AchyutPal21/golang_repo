# Chapter 2 — Revision Checkpoint

Answer in your own words; verify by re-reading the chapter.

---

## Questions

1. Name the five core subcommands of `go` you'd use in a typical day.
2. What does `go.mod` contain? What does `go.sum` contain?
3. What is the *module proxy*, and what guarantees does it provide?
4. What is *minimum-version selection* and how does it differ from
   "latest compatible" resolution?
5. When would you set `GOPRIVATE`?
6. What is the difference between `go install` and `go build`?
7. What is `go work` for, and when would you *not* use it?
8. What does the Go 1 compatibility promise actually promise?
9. Where would you find documentation for a third-party Go module?
10. Why is `golangci-lint` considered the de facto linter despite not
    being part of the standard distribution?

---

## Answers

1. `go run`, `go build`, `go test`, `go vet`, `go mod tidy`. (You'd
   also reach for `go fmt` and `go doc` regularly.)

2. **`go.mod`** declares the module path, the minimum Go version, and
   `require`/`replace`/`exclude`/`retract` directives for
   dependencies. **`go.sum`** lists cryptographic hashes (SHA-256,
   base64) of every module version resolved transitively, including
   their `go.mod` files. Both are committed.

3. The module proxy (`proxy.golang.org` by default) is a caching CDN
   for every public Go module version. It guarantees **availability**
   (deleted upstreams still resolve), **integrity** (responses match
   the public checksum log at `sum.golang.org`), and **speed** (CDN
   edge serves bytes faster than direct Git clones).

4. **MVS** picks, for each module in the dependency graph, the
   *minimum version that satisfies every requirement*. So if A
   requires X v1.2 and B requires X v1.5, the build uses v1.5. **It
   is not** "the latest compatible," which is what npm and pip do.
   The difference: MVS produces stable builds over time. `go build`
   today and tomorrow give the same result without a lockfile,
   because nothing in the algorithm reaches for "latest."

5. When the module path you're depending on (or developing) should
   not be fetched through the public proxy or verified against the
   public checksum DB. Typical case: private internal modules at
   `github.com/yourorg/*`. Without `GOPRIVATE`, the toolchain leaks
   the module path to the public proxy in resolution requests.

6. **`go build`** writes the binary to the *current directory* under
   the package's name. **`go install`** writes it to `$GOBIN`
   (defaulting to `$GOPATH/bin`). `go install` is the idiomatic way
   to install command-line tools so they end up on your `$PATH`;
   `go build` is for producing a deliverable in a build pipeline.

7. **`go work`** is for local development across multiple modules:
   you're editing both a service and a library it depends on, and
   want changes in the library to be picked up immediately by the
   service without committing a `replace` directive. **Don't use it**
   in CI or in committed code; the `go.work` file is developer-local
   and should be in `.gitignore`. Don't use it as an indefinite
   substitute for tagging real versions of shared libs.

8. The promise: no backward-incompatible changes to the language or
   standard library within Go 1.x. Code that builds on Go 1.0 still
   builds on Go 1.22, with rare and well-documented exceptions.
   What it *doesn't* promise: bug compatibility (real bugs may be
   fixed), runtime-internal stability (the GC may get faster,
   programs may get faster or use less memory), or compatibility for
   `unsafe`, `runtime`, or low-level packages.

9. `pkg.go.dev` (web), or `go doc <module>` (terminal). Both read
   doc comments straight from source, so the docs you see are always
   in sync with the released code.

10. The standard distribution ships `go vet`, which catches a narrow
    set of bugs. `golangci-lint` aggregates ~40 community linters
    behind one config and one binary, so a team gets `errcheck`,
    `staticcheck`, `gosec`, `revive`, `gosimple`, `unused`, etc.
    without per-linter setup. Every Go shop runs it; the Go team
    has not absorbed it because they prefer the standard distribution
    to remain minimal.
