# Chapter 5 — Revision Checkpoint

## Questions

1. What's the difference between `go run`, `go build`, and `go install`?
2. What does the build cache do, and where does it live?
3. What does `-ldflags='-s -w'` do, and what's the trade-off?
4. How do you cross-compile a Go binary?
5. Why is `CGO_ENABLED=0` important for production server builds?
6. How do you stamp a version string at build time?
7. What is a reproducible build, and how do you produce one in Go?
8. What does `runtime/debug.ReadBuildInfo` return?

## Answers

1. **`go run`** compiles to a temp directory, runs the binary,
   deletes it. **`go build`** compiles and writes the binary to
   the current directory (or `-o` path). **`go install`** writes
   the binary to `$GOBIN` (default `~/go/bin`). All three share
   the same compile-link pipeline.

2. The build cache (`$GOCACHE`, default `~/.cache/go-build` on
   Linux) holds compiled package archives keyed by content hash.
   A second build with the same inputs reuses the cache, making
   it nearly instantaneous. The cache self-trims; `go clean
   -cache` purges it.

3. `-s` strips the symbol table; `-w` strips DWARF debug info.
   Together they shrink the binary by ~30%. The trade-off is that
   stack traces lose function names and you can't attach a
   debugger. Use in production; not in dev.

4. Set `GOOS` (target OS) and `GOARCH` (target CPU) before `go
   build`: `GOOS=linux GOARCH=arm64 go build -o app-arm64 .`.
   Pure-Go binaries cross-compile from any host to any target
   without extra toolchains. cgo cross-compile requires a target-
   specific C compiler.

5. `CGO_ENABLED=0` produces a fully-static binary that runs on
   `scratch` and `distroless/static` images. With `CGO_ENABLED=1`
   (the default), parts of the standard library may dynamically
   link against `libc`, breaking on those minimal images. For
   server builds, `CGO_ENABLED=0` is the right default.

6. Define a `var version = "dev"` (or similar) in `package main`,
   then build with `-ldflags '-X main.version=1.2.3'`. The linker
   substitutes the string. Caveats: must be `var` not `const`;
   must be type `string`. For commit hashes, prefer
   `runtime/debug.ReadBuildInfo` since 1.18.

7. **A reproducible build** produces bit-for-bit identical binaries
   given the same source and toolchain. Recipe: `-trimpath
   -ldflags='-s -w -buildid='`, plus a pinned toolchain
   (`GOTOOLCHAIN=go1.22.5`), plus a pinned dep tree (`go.sum`).
   After this, two builders should produce binaries with matching
   SHA-256 hashes.

8. `runtime/debug.ReadBuildInfo()` returns a `*BuildInfo` struct
   with the Go version, the main module's path and version, the
   transitive dep list, and a settings list (build flags,
   GOOS/GOARCH, VCS info if `-buildvcs=true`). Available since
   Go 1.12; auto-embedded VCS info since 1.18.
