# Chapter 4 — Revision Checkpoint

## Questions

1. What's the difference between a *module* and a *package*?
2. How is a package's import path computed?
3. What does putting a package under `internal/` do?
4. Why is the convention `cmd/<name>/main.go`?
5. When should you use `go.work`? When should you NOT commit it?
6. Why does the module path for a v2+ module include `/v2`?
7. What does the toolchain's import-resolution order look like?
8. Why is `pkg/` controversial?

## Answers

1. **Module:** unit of versioning and distribution; declared by
   `go.mod`; one per Git repo (typically). **Package:** unit of
   compilation and naming; one per directory; declared by
   `package <name>`. A module contains many packages.

2. Mechanically. Module path + relative directory from module
   root. If module is `github.com/me/notes` and the package lives
   at `<module>/internal/store/postgres`, the import path is
   `github.com/me/notes/internal/store/postgres`. There's no
   discovery, no search path, no per-project config that changes
   this.

3. Restricts importability to packages rooted at the directory
   *above* `internal/`. Code in another module cannot import an
   `internal/` package. The toolchain enforces this; it's a
   compiler-level privacy boundary, not a convention.

4. Multiple binaries in one module need disambiguation, and
   `cmd/<name>/main.go` lets `go build ./cmd/<name>` produce a
   binary called `<name>`. By convention, `cmd/<name>/main.go` is
   tiny — it parses flags, sets up the logger, then calls into
   `internal/<name>.Run(ctx)` so the logic stays testable.

5. **Use `go.work`** when you're editing two local modules
   simultaneously and want changes in one to be picked up by the
   other immediately. **Don't commit it** because it contains
   developer-local paths; the only people for whom those paths
   exist are you (and maybe CI, if CI bothered to set them up the
   same way). Commit `go.mod` and `go.sum`; gitignore `go.work*`.

6. Go's module path is the import path is the on-disk identity of
   the package. If a v2 changes the API, importing it under the
   same name as v1 would silently break callers. Encoding the
   major version into the path (`/v2`) makes the upgrade explicit:
   v1 and v2 can coexist in the same dep tree without conflict.

7. (a) Workspace local — if `go.work` is active and the package's
   module is in `use`, use the local copy. (b) Vendor — if
   `vendor/` exists and the build is in vendor mode, look there.
   (c) Module cache — if the resolved version is already cached,
   use it. (d) Proxy — fetch from `$GOPROXY`, verify against
   `go.sum`, place in cache, use it.

8. `pkg/` is a community convention with no official endorsement.
   Some teams use it for "public" packages and `internal/` for
   private; others put everything at the root. Both work. The
   criticism: `pkg/` adds an extra path component to every import,
   and the public/private split is already enforced by `internal/`.
   Pick a side based on your team's convention; defend it.
