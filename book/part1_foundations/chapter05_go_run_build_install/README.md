# Chapter 5 — How `go run`, `go build`, and `go install` Actually Work

> **Reading time:** ~28 minutes (7,000 words). **Code:** 3 runnable
> programs (~260 lines). **Target Go version:** 1.22+.
>
> You've used `go run` and `go build` already; this chapter shows you
> what's actually happening when you do. Compile pipeline, build cache,
> linker flags, cross-compilation, static vs. dynamic linking, binary
> size, debug info, version stamping. By the end, you'll be able to
> reason precisely about a Go binary you've never seen.

---

## 1. Concept Introduction

`go run`, `go build`, and `go install` are three faces of the same
underlying compile-and-link pipeline. They differ only in *what they
do with the resulting binary*:

* **`go run`** compiles, then runs the binary in a temp directory,
  then deletes it. Iteration mode.
* **`go build`** compiles, writes the binary to the current directory.
  Production-output mode.
* **`go install`** compiles, writes the binary to `$GOBIN`. Tool-
  installation mode.

> **Working definition:** all three commands invoke the same compile-
> link pipeline; what differs is the binary's destination and
> lifecycle. The pipeline is fast because almost everything is cached
> by content hash.

Beyond the surface command, this chapter teaches you the *flags* and
*environment variables* that change what the pipeline produces:
target OS/arch, debug info, version stamping, static vs. dynamic
linking. These are the controls you actually use in production.

---

## 2. Why This Exists

Most languages hide the compile pipeline behind a build system
(Maven, Gradle, CMake, npm-scripts, Cargo). Go puts it in the language
toolchain itself, with a small enough surface that you can hold it in
your head. This chapter is the "look behind the curtain" pass: when
something goes wrong with a build, you should know exactly where the
problem is.

The other reason is operational. Real Go in production crosses
compiles to multiple OSes and architectures, embeds version
information at build time, strips debug info to shrink images, and
chooses between fully-static and dynamic linking. All of that is
controlled by `go build` flags. Knowing them is the difference
between "I made a binary" and "I made a *production* binary."

---

## 3. Problem It Solves

1. **"Where did this binary come from?"** Build-time stamping
   (commit hash, build date, version) answers this. We'll do it
   with `-ldflags`.
2. **"Why is my binary 30 MB?"** Debug info, the runtime, statically-
   linked stdlib. `-ldflags='-s -w'` plus `upx` gets you to ~10 MB.
3. **"Why doesn't my binary run on my prod box?"** Wrong `GOOS` /
   `GOARCH` / glibc version. Solvable with cross-compile + a
   distroless image.
4. **"Why does build #2 take a minute when nothing changed?"**
   Probably your build cache is being invalidated. We'll show how
   to inspect it.
5. **"Can I check the binary works without installing it?"**
   `go run` does this; `go build && ./binary` does this; the
   difference matters for ergonomics.
6. **"Can I prove this binary corresponds to this commit?"**
   `-trimpath` plus `-ldflags='-buildid='` gives bit-for-bit
   reproducibility.

---

## 4. Historical Context

* **Go 1.0 (2012)** introduced the three commands with their current
  semantics. Almost nothing has changed since.
* **Go 1.5 (2015)** removed the C-based bootstrap. Before this, the
  compiler was C; now it's Go all the way down.
* **Go 1.10 (2018)** introduced the persistent build cache. Before
  this, every `go build` recompiled from scratch.
* **Go 1.12 (2019)** added `runtime/debug.ReadBuildInfo`, exposing
  module versions to the running program. Before this, "what version
  am I?" required `-ldflags` magic.
* **Go 1.18 (2022)** extended build info: now `-buildvcs=true`
  (default) embeds Git commit hash and dirty-state into the binary
  automatically.
* **Go 1.20 (2023)** introduced profile-guided optimization (PGO):
  `go build -pgo=auto` reads `default.pgo` and uses it to inform
  inlining/devirtualization. Modest perf win for hot services.
* **Go 1.21 (2023)** introduced toolchain dispatch (`GOTOOLCHAIN`),
  changing how `go build` resolves which Go version to actually use.

---

## 5. How Industry Uses It

The de facto production patterns:

* **Local dev:** `go run ./cmd/<binary>` for tight iteration.
* **Local binary builds:** `go build -o bin/<binary> ./cmd/<binary>`.
* **Tool install:** `go install ./cmd/<binary>` for things you'll
  invoke daily.
* **Release builds:** `goreleaser` orchestrates `go build` for every
  target OS/arch, embeds version info via `-ldflags`, signs binaries
  with `cosign`, builds container images via `ko`, drafts a GitHub
  release.
* **Container builds:** multi-stage Dockerfile. First stage uses
  `golang:1.22-alpine` and runs `CGO_ENABLED=0 go build`. Final
  stage is `gcr.io/distroless/static` or `scratch`, copies the
  binary from stage 1, sets `CMD`. Result: ~10 MB images.
* **CI:** `go build ./...` first to fail fast on compile errors,
  then `go test ./...`. The build cache is shared across runs via
  `actions/cache`.
* **Reproducible builds:** `-trimpath -ldflags='-s -w -buildid='`
  plus a pinned Go toolchain via `GOTOOLCHAIN=go1.22.5`.

---

## 6. Real-World Production Use Cases

**The 8 MB container image.** A team building a Kubernetes operator
needs a tiny base image for security and speed. Their Dockerfile:

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags='-s -w -X main.version=$(git rev-parse HEAD)' \
    -o /bin/operator ./cmd/operator

FROM gcr.io/distroless/static:nonroot
COPY --from=build /bin/operator /operator
USER nonroot:nonroot
ENTRYPOINT ["/operator"]
```

Result: ~12 MB image. No shell, no package manager, no glibc.
Smaller attack surface, faster cold-start, faster image pulls.

**The single static binary deploy.** A founder ships a SaaS backend
to a $5 VPS. The deploy script:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
    -trimpath -ldflags='-s -w -X main.version=$(git describe --tags)' \
    -o app ./cmd/server
scp app prod:/usr/local/bin/
ssh prod 'systemctl restart app'
```

That's it. No Docker, no Kubernetes. `app` is one file with no
runtime dependency, copied to one server.

**Cross-platform CLI release.** A team ships a CLI tool to end-users
on macOS (Intel + Apple Silicon), Linux (amd64 + arm64), Windows.
`goreleaser` runs five `go build` commands in parallel from one
Linux CI runner — no SDK juggling, no Mac in the loop. The artifacts
are signed, packaged as `.tar.gz`/`.zip`, and uploaded to a GitHub
release with a single config file.

**Reproducible build for compliance.** A bank needs to prove that
the binary in production matches the source at a given commit. They
build with `-trimpath -ldflags='-s -w -buildid=' GOFLAGS='-mod=vendor'
GOTOOLCHAIN=go1.22.5`. SHA-256 of the binary is recorded in a
sealed log. Auditors can re-run the build and verify the hash.

**Version-stamped binary.** Every service in a platform team's
fleet has a `--version` flag whose output includes Git commit, build
date, Go version, and module versions of key dependencies. Built
via `-ldflags '-X main.version=...'` plus `runtime/debug.ReadBuildInfo`.
On-call paste that output into incident tickets.

---

## 7. Beginner-Friendly Explanation

Three commands, three jobs:

* **`go run main.go`** — "run this once, I don't care about the
  binary." Used during development. Adds a small amount of overhead
  (writing a temp file, launching it) but it's nearly invisible.
* **`go build`** — "make a real binary I can keep." Without flags,
  it writes the binary in your current directory, named after the
  package's last path component (or the directory name).
* **`go install`** — "make a binary AND put it somewhere on my
  `$PATH`." It writes to `$GOBIN`, which defaults to `~/go/bin`,
  which is probably already in your `$PATH`. Use this for tools.

Two flags worth knowing on day one:

* **`-o <path>`** — name the output binary. `go build -o bin/server
  ./cmd/server` gives you `bin/server` regardless of the package's
  name.
* **`-v`** — print each package as it's compiled. Useful when you
  wonder "what's it actually building?"

Two environment variables worth knowing on day one:

* **`GOOS`** — target OS. Set to `linux`, `darwin`, `windows`, etc.
* **`GOARCH`** — target CPU. Set to `amd64`, `arm64`, etc.

Set both at once for cross-compilation: `GOOS=linux GOARCH=arm64
go build -o app-linux-arm64 ./cmd/server`. That's the *whole*
cross-compile story for pure-Go binaries.

> **Coming From Java —** `go build` is `mvn package`, except it's
> 50x faster, has no XML, and the output isn't a JAR — it's a real
> executable. `go install` is roughly `mvn install` + putting the
> shaded jar on `$PATH`.

> **Coming From Python —** There's no `python myscript.py`
> equivalent — the closest is `go run`. There's also no `pip
> install -e .`; `go install` plays that role for command-line
> tools.

> **Coming From C/C++ —** `go build` is `cmake && make`, except
> the dependency graph is implicit (from imports), the cache is
> per-package and content-addressed, and you don't write any build
> scripts. There is no `make install` step in development.

> **Coming From Rust —** `go build` is `cargo build`. `go install`
> is `cargo install`. The biggest difference is speed (Go's
> compiler is faster than rustc) and binary size (Go binaries are
> bigger because the runtime is included; Rust's optimizer
> typically produces smaller binaries).

---

## 8. Deep Technical Explanation

### 8.1. The compile-link pipeline, in detail

When you `go build ./cmd/server`:

1. **Module resolution.** Toolchain finds the enclosing `go.mod`,
   resolves the dep graph using minimum-version selection, fetches
   any missing modules from `$GOPROXY`. Verifies hashes against
   `go.sum`.
2. **Package discovery.** Compiles the import graph reachable from
   `./cmd/server`. Each package is a directory of `.go` files.
3. **Per-package compilation.** For each package:
   * **Parse.** Source → AST.
   * **Type-check.** AST → typed AST. Reports type errors.
   * **SSA build.** Typed AST → static-single-assignment IR.
   * **Optimize.** Inlining, dead-code elimination, escape analysis,
     bounds-check elimination, devirtualization, and (with PGO)
     profile-guided variants.
   * **Code generation.** SSA → architecture-specific assembly →
     machine code. Result: a `.a` archive in the build cache.
4. **Linking.** All package archives + the runtime + any cgo glue
   are linked into one executable. The default linker is
   `cmd/link`, internal-linking. With cgo, an external linker
   (`cc` / `ld`) is invoked.
5. **Output.** Binary written to its destination (cwd for `go
   build`, `$GOBIN` for `go install`, temp dir for `go run`).

The build cache is keyed by content hash of *every input* — source
files, build flags, Go toolchain version, dep graph. A second build
with the same inputs is nearly instantaneous.

### 8.2. The build cache

* **Location:** `$GOCACHE`. Defaults to `~/.cache/go-build` on Linux,
  `~/Library/Caches/go-build` on macOS.
* **Layout:** keyed directories, content-addressed. Don't poke at
  it; the toolchain is the only safe consumer.
* **Inspect:** `go env GOCACHE` shows the path; `du -sh $(go env
  GOCACHE)` shows size.
* **Clear:** `go clean -cache` removes everything. Rarely needed.
* **Trim policy:** the cache self-trims (entries older than a few
  weeks are evicted) to bound disk usage.

The cache is content-addressed, so you can share it across CI runs
safely. GitHub Actions' `actions/setup-go@v5` does this with
`cache: true`.

### 8.3. The most useful flags

Listed roughly in order of how often you'll use them:

* **`-o <path>`** — output binary path. Always set in CI; default
  filename guessing is fragile.
* **`-v`** — verbose. Print each package as it compiles.
* **`-x`** — print every command the toolchain runs. Powerful for
  debugging.
* **`-race`** — enable the race detector. 5–10x slowdown, catches
  data races. Use in test runs and dev, not production.
* **`-trimpath`** — strip absolute paths from the binary (replaces
  `/home/you/proj/file.go` with `proj/file.go`). Required for
  reproducibility.
* **`-ldflags <flags>`** — pass flags to the linker. The most
  important production flag.
* **`-gcflags <flags>`** — pass flags to the compiler. Used for
  inspecting compiler decisions.
* **`-tags <tag>,<tag>`** — build tags, controlling conditional
  compilation. We'll cover them in Chapter 7.
* **`-mod=<mode>`** — module mode: `mod` (default; modify go.sum
  freely), `readonly` (refuse to add to go.sum), `vendor` (use
  vendor/ directory).
* **`-buildvcs=<bool>`** — embed VCS info (Git commit, dirty state).
  Default true since 1.18.
* **`-pgo=<file>`** — profile-guided optimization. Auto-detects
  `default.pgo` since Go 1.21.
* **`-cover`** — instrument for coverage. Mostly used with `go
  test`.

### 8.4. `-ldflags` deep dive

`-ldflags` is the linker flag bag. The flags you'll use most:

* **`-s`** — omit the symbol table. Smaller binary; stack traces
  lose function names. Use in production *if* you accept this.
* **`-w`** — omit DWARF debug info. Smaller binary; you can't
  attach a debugger. Use in production.
* **`-X importpath.name=value`** — set a string variable at link
  time. The variable must be a `var` (not `const`) of type `string`
  in the named package.
* **`-buildid=<id>`** — set the build ID. Empty string makes builds
  reproducible across users.
* **`-extldflags <flags>`** — flags passed to the external linker
  (when using cgo).

The classic version-stamping pattern:

```go
package main

var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

func main() {
    fmt.Printf("%s built from %s at %s\n", version, commit, date)
}
```

```bash
go build \
    -ldflags "-s -w \
              -X main.version=$(git describe --tags) \
              -X main.commit=$(git rev-parse HEAD) \
              -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o myapp ./cmd/myapp
```

This is so common that `goreleaser` does it for you with one line
of YAML config.

Since Go 1.18, `runtime/debug.ReadBuildInfo()` exposes Git commit
and dirty-state automatically (with `-buildvcs=true`, the default).
You can read this without `-ldflags`:

```go
import "runtime/debug"

info, _ := debug.ReadBuildInfo()
for _, s := range info.Settings {
    if s.Key == "vcs.revision" {
        fmt.Println("commit:", s.Value)
    }
}
```

This is preferable to `-X` for commit hashes; reserve `-X` for the
human-readable version string.

### 8.5. Cross-compilation

Pure-Go binaries cross-compile from any host to any target with no
toolchain switching. The mechanism: `GOOS` and `GOARCH`.

```bash
# From a Linux laptop, build for macOS Apple Silicon:
GOOS=darwin GOARCH=arm64 go build -o app-darwin-arm64 ./cmd/app

# For Windows amd64:
GOOS=windows GOARCH=amd64 go build -o app.exe ./cmd/app

# For a Raspberry Pi 4:
GOOS=linux GOARCH=arm64 go build -o app-pi ./cmd/app

# For a Raspberry Pi Zero (32-bit ARM):
GOOS=linux GOARCH=arm GOARM=7 go build -o app-pi-zero ./cmd/app
```

Common `GOOS`/`GOARCH` combinations:

| GOOS | GOARCH | Use |
| --- | --- | --- |
| `linux` | `amd64` | most servers |
| `linux` | `arm64` | AWS Graviton, Raspberry Pi 4 |
| `linux` | `arm` | older ARM (specify GOARM=5/6/7) |
| `darwin` | `amd64` | Intel Macs |
| `darwin` | `arm64` | Apple Silicon |
| `windows` | `amd64` | most Windows |
| `windows` | `arm64` | newer Windows on ARM |
| `freebsd` | `amd64` | FreeBSD servers |
| `js` | `wasm` | WebAssembly for browsers |
| `wasip1` | `wasm` | WebAssembly with WASI |

Run `go tool dist list` to see every supported combination
(currently ~50).

**Caveat: cgo doesn't cross-compile easily.** If your binary
imports a C library, you need a target-specific C cross-compiler.
The fix is usually to set `CGO_ENABLED=0`, which produces a fully-
static, no-cgo binary. This is also what most production Go
services do anyway, because it's smaller and easier to deploy.

### 8.6. Static vs dynamic linking

By default, Go produces *almost-static* binaries on Linux:

* Standard-library Go: statically linked. Always.
* Pure-Go third-party packages: statically linked.
* Packages using cgo (e.g. `os/user` on glibc, `net` with the C
  resolver): dynamically linked against `libc`.

Set `CGO_ENABLED=0` to get a fully-static binary. The cost: lose
cgo, lose access to a few stdlib features that fall back to cgo
(the most common: `os/user.Lookup` on glibc, `net.LookupHost` with
the C resolver). Most server applications don't need any of this.

```bash
# Fully static:
CGO_ENABLED=0 go build -o app .

# Verify (Linux):
file app
# → ELF 64-bit LSB executable, x86-64, version 1 (SYSV),
#   statically linked, no section header
```

For production server builds, `CGO_ENABLED=0` is almost always the
right default. It enables the smallest container images
(`scratch`, `distroless/static`).

### 8.7. Binary size

A "hello, world" Go binary is ~2 MB. A medium service with imports
is 15–30 MB. Why so big?

* The Go runtime is ~600 KB.
* The standard library packages you use are statically linked.
* DWARF debug info is ~10–30% of the binary.
* The symbol table is another ~5%.

Size optimizations, in order of impact:

1. **`-ldflags='-s -w'`** — strip symbols and DWARF. ~30% smaller.
2. **`-trimpath`** — small saving, also helps reproducibility.
3. **`upx --best <binary>`** — compresses the binary with self-
   extraction. Often 50%+ smaller. Adds startup latency (decompress
   on launch); not for latency-sensitive services.
4. **Reduce imports.** The `crypto/*` and `net/*` trees pull in a
   lot of code. If you don't need TLS, don't import `crypto/tls`.

For *most* production services, `-ldflags='-s -w'` is enough. UPX
is reserved for tools that must fit in tight CI artifacts.

### 8.8. Reproducible builds

A *reproducible build* produces bit-for-bit identical binaries
given the same source. Required for some compliance regimes; useful
for everyone else.

Recipe:

```bash
go build \
    -trimpath \
    -ldflags='-s -w -buildid=' \
    -o app ./cmd/app
```

* `-trimpath` removes user-specific paths.
* `-ldflags='-buildid='` removes the random build ID.
* Pin the toolchain via `GOTOOLCHAIN=go1.22.5` so two builders use
  the same compiler.
* Build with `-mod=vendor` against a vendored tree, or pin
  `$GOPROXY` to a known-deterministic mirror.

After this, two builders running the same commit should produce
binaries with the same SHA-256.

### 8.9. PGO (profile-guided optimization)

Since Go 1.21, you can hand the compiler a profile of your workload
and it'll specialize for it. Modest perf wins (5–15% on CPU-bound
hot paths).

Workflow:

1. Capture a CPU profile in production:
   `curl http://prod:6060/debug/pprof/profile?seconds=30 > default.pgo`.
2. Place `default.pgo` next to the `main` package.
3. `go build -pgo=auto ./cmd/server` — the toolchain picks it up.

The profile guides inlining and devirtualization decisions. For
services with stable hot paths (most), PGO is free perf. For
services with shifting hot paths, the gain may not justify the
profile-management overhead.

---

## 9. Internal Working (How Go Handles It)

* The `go` driver lives in `cmd/go/`. It's the user-facing program;
  it `exec`s the actual tools.
* The compiler is `cmd/compile/`. Pipeline: parser →
  `cmd/compile/internal/syntax` → AST →
  `cmd/compile/internal/types2` (type checker) →
  `cmd/compile/internal/ir` (IR) →
  `cmd/compile/internal/ssa` (SSA build + optimize) → arch-specific
  backend.
* The linker is `cmd/link/`. Reads package archives, resolves
  symbols, applies relocations, writes the executable in ELF/Mach-O/
  PE format.
* The runtime, statically linked into every binary, is `runtime/`.
  Includes the scheduler, GC, allocator, channel/select machinery,
  reflection metadata reader.
* The build cache is implemented in `cmd/go/internal/cache`. Each
  cache entry is a content-addressed file plus a metadata header.

Read these for entertainment value some weekend: `cmd/go/internal/work/exec.go`
shows you the action graph the toolchain builds before kicking off
work. It's beautifully clear.

---

## 10. Syntax Breakdown

The "syntax" here is the command line. Three canonical forms you'll
see:

```bash
# Development iteration:
go run ./cmd/server

# Production build with version info:
go build \
    -trimpath \
    -ldflags="-s -w \
              -X main.version=$(git describe --tags) \
              -X main.commit=$(git rev-parse HEAD)" \
    -o bin/server \
    ./cmd/server

# Tool installation:
go install github.com/some/tool/cmd/sometool@latest
```

Subcommand-specific flags come *before* the package path; flags after
are passed *to the program* (in `go run`'s case).

---

## 11. Multiple Practical Examples

### Example 1 — `examples/01_build_inspector`

```bash
go run ./examples/01_build_inspector
go run ./examples/01_build_inspector $(which go)
```

A program that uses `runtime/debug.ReadBuildInfo` plus
`debug/buildinfo` to inspect itself or any other Go binary on disk.
Reports: Go version, module path, dep tree, VCS info, build flags.
This is exactly how a `--version` subcommand should work in
production.

### Example 2 — `examples/02_cross_compile_demo`

```bash
go run ./examples/02_cross_compile_demo
```

Uses `runtime` to print the *currently running* OS and architecture,
then loops through a few `GOOS`/`GOARCH` combinations and prints the
shell command you'd use to cross-compile this very file for each.

### Example 3 — `examples/03_ldflags_version`

```bash
# Default — version says "dev":
go run ./examples/03_ldflags_version

# Custom — version stamped at build time:
go build -ldflags "-X main.version=1.2.3 -X main.commit=abc1234" \
    -o /tmp/app ./examples/03_ldflags_version
/tmp/app
```

A minimal version-stamping demo. The pattern is identical at scale;
this just removes everything else.

---

## 12. Good vs Bad Examples

**Good production build command:**

```bash
CGO_ENABLED=0 \
GOOS=linux \
GOARCH=amd64 \
go build \
    -trimpath \
    -ldflags='-s -w -X main.version='"$VERSION" \
    -o bin/server \
    ./cmd/server
```

**Bad:**

```bash
go build .
```

Why bad in production:

1. No `-o` — the binary's name is unpredictable.
2. No `-trimpath` — local paths leak into the binary.
3. No `-ldflags='-s -w'` — debug info inflates the binary by 30%.
4. No version stamping — the binary can't say what it is.
5. No `CGO_ENABLED=0` — depending on imports, may produce a
   dynamically-linked binary that breaks on `scratch` images.

The bad form is fine for `go run` style iteration. It is not fine
for shipping.

**Good Dockerfile fragment:**

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags='-s -w' \
    -o /app ./cmd/server
```

**Bad:**

```dockerfile
RUN go build -o /app ./cmd/server
```

The bad form may pull in cgo bindings (depending on imports), which
breaks on `scratch` and `distroless/static`. The good form
forecloses that.

---

## 13. Common Mistakes

1. **Using `go run` in production.** `go run` recompiles every
   invocation. Build the binary once, run it many times.
2. **Forgetting `-o`.** `go build ./cmd/foo/bar/baz` writes to
   `./baz`, which may surprise CI scripts. Always set `-o`.
3. **Mixing `CGO_ENABLED=1` with `scratch` images.** Symptom:
   binary won't run in the container; "no such file or directory"
   for the binary itself. Fix: set `CGO_ENABLED=0`.
4. **Stripping symbols in dev.** With `-ldflags='-s -w'` you lose
   stack traces. Don't strip in development; strip only in release
   builds.
5. **Using `-X` to set non-string variables.** `-X` only works on
   `var` of type `string`. It silently does nothing for `const`,
   ints, or other types.
6. **Not pinning Go version in CI.** Symptom: builds drift between
   local and CI as Go releases ship. Fix: `go-version-file: go.mod`.
7. **Cross-compiling with cgo and being surprised it fails.**
   cgo cross-compile requires a target-specific C compiler. Set
   `CGO_ENABLED=0` or invest in a cross toolchain.
8. **Building for the wrong architecture on Apple Silicon.** You
   want `darwin/arm64`, not `darwin/amd64`, on M-series Macs unless
   you specifically need to test Rosetta. Check `go env GOARCH`.
9. **Treating `go install pkg@latest` as reproducible.** It isn't
   — `latest` resolves at install time. Pin the version:
   `go install pkg@v1.2.3`.
10. **Caching `~/.cache/go-build` *across users* or *across CI
    images* with mismatched Go versions.** The cache keys include
    the toolchain version, so this *should* be safe, but mixing
    architectures or OSes in a shared cache is a sharp edge.
    Cache per (Go version, OS, arch).

---

## 14. Debugging Tips

* **`go build -x ./cmd/foo`** prints every command the toolchain
  runs. Read it when builds feel mysterious.
* **`go build -work ./cmd/foo`** keeps the temporary build directory
  so you can inspect intermediates. Combined with `-x`, this is
  the nuclear option.
* **`go env GOCACHE`** + `du -sh` to see how big the build cache
  has grown.
* **`go clean -cache`** when you suspect cache corruption (rare,
  but real).
* **`go list -m -json all | jq`** to read the resolved dep graph.
* **`go version -m <binary>`** to introspect a built binary's
  module versions.
* **`go tool nm <binary>`** to list symbols (when not stripped).
* **`go tool objdump -s <symbol> <binary>`** to disassemble.
* **`file <binary>`** (on Linux/macOS) to confirm static vs
  dynamic linking.

---

## 15. Performance Considerations

* **Cold builds dominate CI.** The single biggest CI win is
  caching the module cache (`~/go/pkg/mod`) and the build cache
  (`~/.cache/go-build`) keyed by `go.sum`.
* **Build parallelism.** `go build` parallelizes per-package
  compilation up to `GOMAXPROCS`. Use a CI runner with multiple
  cores for large projects.
* **Avoid build-tagging hell.** Each combination of build tags is
  a separate cache key. Five build tags = potentially 32 cache
  entries.
* **`-trimpath`** has near-zero build-time cost; ship it always.
* **PGO** adds ~10% build time for ~5–15% runtime perf. Worth it
  for hot services; overkill for low-volume tools.

---

## 16. Security Considerations

* **`-trimpath`** removes user paths from the binary. Without it,
  the binary leaks information about your build machine.
* **`-buildvcs=true`** (default) embeds Git commit and dirty-state
  in the binary. Useful for tracing incidents back to source. Some
  teams set `-buildvcs=false` for proprietary closed-source
  releases.
* **Reproducible builds** let you (and others) verify that a
  binary in production really does correspond to a public source
  commit. This is the basis for SLSA and most supply-chain security
  frameworks.
* **`go install pkg@latest`** runs arbitrary code from arbitrary
  modules. Each `go install` line in a CI script is a supply-chain
  decision. Pin versions.
* **Stripping symbols (`-s -w`)** doesn't prevent reverse
  engineering, but it slows it down. Don't rely on it for IP
  protection; rely on it to shrink production artifacts.

---

## 17. Senior Engineer Best Practices

1. **`go build` for binaries; `go run` for iteration; `go install`
   for tools.** Don't conflate.
2. **Always set `-o`** in CI and scripts.
3. **`-trimpath` everywhere.** It's the cheapest reproducibility
   win available.
4. **`-ldflags='-s -w'` in production builds.** Smaller, faster
   to ship.
5. **`CGO_ENABLED=0`** in production server builds unless you
   *know* you need cgo. This unlocks `scratch`/`distroless` images
   and avoids glibc compatibility traps.
6. **Embed version info via `-X main.version`**, plus rely on
   `runtime/debug.ReadBuildInfo` for VCS info.
7. **Pin Go version in CI** via `go-version-file: go.mod`.
8. **Build *and* test with `-race`** in at least one CI lane.
9. **Adopt PGO** for hot services with stable workloads.
10. **Write a `--version` flag for every binary.** It pays off the
    first time you have to triage a production incident.

---

## 18. Interview Questions

1. *(junior)* What's the difference between `go run` and `go build`?
2. *(junior)* How do you cross-compile a Go binary for Linux from
   a macOS laptop?
3. *(mid)* What does `-ldflags='-s -w'` do?
4. *(mid)* Why is a Go "hello world" binary ~2 MB?
5. *(senior)* How do you embed a Git commit hash in a Go binary?
6. *(senior)* What's the difference between `CGO_ENABLED=1` and
   `CGO_ENABLED=0`, and why does it matter for container deploys?
7. *(senior)* Walk me through what happens when I `go build
   ./cmd/server` on a fresh machine.
8. *(senior)* What is a reproducible build and how do you produce
   one in Go?
9. *(staff)* What is profile-guided optimization, and when would
   you adopt it?

---

## 19. Interview Answers

1. **`go run`** compiles to a temp directory, runs the binary,
   deletes it. **`go build`** compiles and writes the binary to
   the current directory (or wherever `-o` says). `go run` is for
   iteration; `go build` is for shipping.

2. `GOOS=linux GOARCH=amd64 go build -o app ./cmd/server` (or
   `GOARCH=arm64` for Graviton). Pure-Go cross-compilation needs
   no extra toolchain. If your code uses cgo, set
   `CGO_ENABLED=0` or install a target-specific C cross-compiler.

3. `-s` strips the symbol table; `-w` strips DWARF debug info.
   Together they shrink the binary by ~30%. The cost is that
   stack traces lose function names and you can't attach a
   debugger. Use in production; don't use in dev.

4. The Go runtime is statically linked into every binary. That
   includes the scheduler, garbage collector, memory allocator,
   channel/select machinery, and reflection metadata. ~600 KB of
   runtime + ~1.4 MB of standard library code (formatting,
   reflection, etc.) = ~2 MB. The trade is that you ship one file
   with no runtime dependency on the host.

5. Two ways. Modern: rely on `-buildvcs=true` (the default since
   1.18) and read it via `runtime/debug.ReadBuildInfo`'s
   `Settings` (`vcs.revision`). Classic: set a `var commit
   string` in `package main` and stamp it with
   `-ldflags '-X main.commit=$(git rev-parse HEAD)'`. The
   modern approach is preferable for the commit; reserve `-X` for
   the human-readable version string.

6. `CGO_ENABLED=1` (default on most platforms) allows the binary
   to link against C code. This means parts of the standard
   library (`os/user`, `net` with the C resolver) link against
   `libc` dynamically. The binary won't run on `scratch` or
   `distroless/static` because there's no `libc` there.
   `CGO_ENABLED=0` produces a fully-static, libc-free binary that
   runs anywhere. For server work, `CGO_ENABLED=0` is the right
   default.

7. Walk through it: toolchain finds `go.mod`, resolves the dep
   graph (MVS), fetches missing modules from `$GOPROXY`, verifies
   `go.sum`. For each package in the import graph: parse, type-
   check, build SSA, optimize, generate machine code, write to
   build cache. Linker reads package archives, links with the
   runtime, writes the executable. Build cache makes subsequent
   builds nearly instantaneous.

8. **A reproducible build** produces a bit-for-bit identical
   binary given the same source and toolchain. In Go, the recipe
   is `-trimpath -ldflags='-s -w -buildid='`, plus a pinned
   toolchain (`GOTOOLCHAIN=go1.22.5`), plus a pinned dep tree
   (`go.sum` committed and verified). After this, two builders
   produce identical SHA-256 hashes. This is the basis for
   supply-chain security frameworks like SLSA.

9. **PGO** (profile-guided optimization) hands the compiler a
   profile of your workload (a CPU profile from production) and
   the compiler uses it to make better inlining and
   devirtualization decisions. Available since Go 1.21 with
   `-pgo=auto` (reads `default.pgo`). Adoption: capture a
   30-second production CPU profile, commit it as `default.pgo`,
   rebuild. Expect 5–15% perf win on CPU-bound hot paths. Adopt
   for hot services with stable workloads; skip for low-volume
   tools or services with shifting hot paths.

---

## 20. Hands-On Exercises

**Exercise 5.1 — Build inspector.** Run
`go run ./examples/01_build_inspector $(which gopls)`. Read the
output. Identify (a) the Go version `gopls` was built with, (b)
its module path, (c) a few of its dependencies. Now run it on a
binary you built yourself.

**Exercise 5.2 — Binary diet.** Write a shell script that builds
the same program three ways:

1. `go build -o app .`
2. `go build -ldflags='-s -w' -o app-stripped .`
3. `go build -ldflags='-s -w' -trimpath -o app-stripped-trimmed .`

Print the size of each. Quantify the savings.

**Exercise 5.3 ★ — Stamp a version.** Modify
`examples/03_ldflags_version` to additionally print the Git
commit short-hash and the build date. Use `runtime/debug.ReadBuildInfo`
for the commit (since 1.18 it's automatic) and `-ldflags` for the
date. Build, run, confirm both fields are set correctly.

---

## 21. Mini Project Tasks

**Task — A `release.sh` script.** Write a shell script
`release.sh` that produces release artifacts for your platform's
Go service, for three OS/arch combinations (linux/amd64,
linux/arm64, darwin/arm64). For each:

* Build with `-trimpath -ldflags='-s -w -X main.version=...'`.
* Compress with `gzip` or `xz`.
* Compute SHA-256 and append to a `SHA256SUMS` file.

This is the essence of `goreleaser`; doing it once by hand
teaches you what it's automating.

---

## 22. Chapter Summary

* `go run`, `go build`, and `go install` share one compile-link
  pipeline; they differ only in the binary's destination.
* The build cache makes second builds nearly free. Cache it in CI.
* `-o` names the output; `-trimpath` strips local paths;
  `-ldflags='-s -w'` shrinks the binary by ~30%.
* `-ldflags '-X main.var=value'` stamps string variables at link
  time. Use it for the version string; rely on `runtime/debug.ReadBuildInfo`
  for VCS info.
* Cross-compile with `GOOS` + `GOARCH`. Pure-Go binaries cross to
  any target without extra toolchains.
* `CGO_ENABLED=0` produces fully-static binaries that run on
  `scratch`/`distroless/static`. Default it for production server
  builds.
* Reproducible builds: `-trimpath -ldflags='-s -w -buildid='`
  plus a pinned toolchain.
* PGO is free perf for hot services with stable workloads.

Updated working definition: *the three commands wrap one fast,
content-addressed compile-link pipeline; the flags
(`-o`, `-trimpath`, `-ldflags`, `-tags`) are the production
controls. Knowing them is the difference between making a binary
and shipping one.*

---

## 23. Advanced Follow-up Concepts

* **`cmd/go`'s documentation** — `go help build`, `go help run`,
  `go help install`, `go help build` are all worth reading.
* **`go.dev/doc/go1.21#pgo`** — the PGO release notes.
* **Russ Cox, "Reproducing Bug fixes for security flaws since 1.16"** —
  context on the build determinism work.
* **Filippo Valsorda, "Toolchain dispatch"** (2023) — how `GOTOOLCHAIN`
  resolves which Go to use.
* **`goreleaser` documentation** at `goreleaser.com` — the de
  facto release tool's config language is itself an education in
  what a "release pipeline" needs.
* **SLSA documentation** at `slsa.dev` — the supply-chain security
  framework that Go's reproducible-build features support.

> **End of Chapter 5.** Move on to [Chapter 6 — Coming From Another
> Language](../chapter06_coming_from_another_language/README.md).
