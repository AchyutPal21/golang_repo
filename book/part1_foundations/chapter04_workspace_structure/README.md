# Chapter 4 — The Go Workspace and Project Structure

> **Reading time:** ~30 minutes (7,500 words). **Code:** 3 example
> programs across two modules (~250 lines). **Target Go version:** 1.22+.
>
> This chapter takes you from "I can run a single `.go` file" to "I
> can lay out a real Go project that scales to a hundred files and
> three teams." We'll cover modules, packages, the import graph,
> visibility rules, the `internal/` directory, multi-module workspaces
> (`go work`), and the **non-existence** of an official "standard
> project layout."

---

## 1. Concept Introduction

A Go project is built from three nested concepts:

* **A *module*** — the unit of versioning and distribution. Declared
  by `go.mod`. Has a globally unique import path (e.g.
  `github.com/example/notes`). One module per `go.mod` file.
* **A *package*** — the unit of compilation and naming. Every
  directory under a module is a package. The directory's `.go` files
  share a `package <name>` declaration.
* **A *workspace*** — the unit of *local development*, optional. A
  `go.work` file at a parent directory tells the toolchain "treat
  these local modules as the source of truth, even if `go.mod`
  references say otherwise."

> **Working definition:** a Go project's structure is its module
> boundary plus its tree of packages, with import paths derived
> mechanically from directory layout. Visibility is controlled by
> capitalization (exported vs. unexported) and by the magic
> directory name `internal/`. No frameworks. No XML. No build
> manifests beyond `go.mod`.

---

## 2. Why This Exists

Languages take two stances on project structure:

* **Convention-bound:** Java's package-equals-directory, Python's
  `__init__.py` packages, Rust's `Cargo.toml` + `src/`. The structure
  is enforced by the toolchain.
* **Convention-free:** C, C++. You can put files wherever you like;
  the build system decides what's a unit. The cost is bikeshedding.

Go is convention-bound: the *directory tree* is the *package tree*,
and an *import path* is computed mechanically from the module path
plus the relative directory. There is no "include-path" search like
C, no "classpath" like Java, no `sys.path` manipulation like Python.
This is one of the things that makes a Go codebase legible to anyone
who knows the language: from any import statement, you know exactly
where to find the source.

The trade is that Go is opinionated about *layout* below the package
level. There is exactly one way to declare visibility (capitalize the
first letter of the identifier). There is exactly one way to mark a
package as "internal to this module" (put it under `internal/`). The
rules are small and unambiguous, which is the point.

What Go is *not* opinionated about: how you structure the
*application logic* — `cmd/`, `pkg/`, `internal/`, `service/`,
`domain/`, layered, hexagonal, flat. The Go team explicitly publishes
no "official project layout." This catches every newcomer off guard.

---

## 3. Problem It Solves

Concrete problems Go's structure rules solve:

1. **Import cycle detection.** The compiler refuses cyclic imports.
   Combined with the package-equals-directory rule, this forces
   acyclic dependency graphs.
2. **API surface control.** `internal/` directories cannot be
   imported from outside the module. This gives library authors a
   first-class private namespace.
3. **Visibility.** Capitalization is checked by the compiler: if it
   compiles, you've gotten the public/private split right.
4. **Versioning at the right granularity.** A module is the unit of
   versioning. You don't need to think about which package is on
   which version; they all move together.
5. **Local development of split modules.** `go.work` lets you edit
   two modules simultaneously without committing temporary `replace`
   directives.
6. **Reproducible imports.** An import path is a URL. Anyone who
   reads your code knows exactly where the dependency lives.

---

## 4. Historical Context

The Go layout story has three phases.

**The GOPATH era (2009–2018).** Every Go project lived under a
single shared workspace tree:
`$GOPATH/src/<full-import-path>`. So if you imported
`github.com/foo/bar`, the source had to be at
`~/go/src/github.com/foo/bar`. This worked for Google's monorepo
and was awkward for everyone else. There was no version pinning;
`go get` always pulled the default branch. This drove tools like
`dep` and `glide` to fill the gap.

**The modules transition (2018–2021).** Go 1.11 introduced modules
(`go.mod` + `go.sum`) as an opt-in. Go 1.16 made them the default.
The transition decoupled disk layout from import path: a project
could now live anywhere on disk, and its import path was declared
explicitly.

**The workspaces era (2022–present).** Go 1.18 introduced
`go.work`, the "monorepo of independent modules" file. Go 1.22 added
the `go work use` and `go work edit` subcommands for cleaner
management. Workspaces are now the recommended pattern for
multi-module local development.

A few opinions that became conventions along the way:

* **`cmd/`** for executable subdirectories, popularized by
  Kubernetes and Docker.
* **`internal/`** as a compiler-enforced private directory, added
  in Go 1.4 (2014).
* **`pkg/`** for "the public packages of this module," popularized
  by some tutorials and the `pkg` directory of `go.dev` itself.
  Russ Cox has on multiple occasions said `pkg/` is *not*
  recommended. Many shops use it anyway. Pick a side and stick to
  it; do not mix.

> **Senior Architect Note —** When you read a Go project for the
> first time, look at its top-level directory. Three healthy
> patterns: (a) flat — packages directly at the root, no
> subdirectories like `cmd/`, used by small libraries; (b) `cmd/` +
> `internal/` — common for services and tools; (c) `cmd/` +
> `internal/` + `pkg/` — services that intentionally export
> reusable packages. Anything more elaborate (e.g.
> `application/services/handlers/v2/internal/...`) is usually a
> smell.

---

## 5. How Industry Uses It

The de facto industry layouts in 2026:

**Single-binary tool (CLI, daemon):**
```
example.com/tool/
├── go.mod
├── go.sum
├── main.go              # the entry point
├── tool.go              # core logic, package "tool" at the root
└── internal/
    ├── parse/...
    └── render/...
```
Package at the root is named after the module. The binary is built
with `go build .` from the root.

**Multi-binary repo with shared internals:**
```
example.com/platform/
├── go.mod
├── cmd/
│   ├── api/main.go      # the API server binary
│   ├── worker/main.go   # the worker binary
│   └── migrate/main.go  # the migration tool
└── internal/
    ├── auth/
    ├── store/
    ├── http/
    └── jobs/
```
This is the dominant pattern at scale. `cmd/<name>/main.go`
produces a binary called `<name>`. `internal/` is private to this
module and carries the shared code.

**Library:**
```
github.com/example/redis-client/
├── go.mod
├── client.go             # package "redis"
├── client_test.go
├── pool.go
├── doc.go                # package-level docs
└── internal/
    └── proto/...         # private to this module
```
No `cmd/`, no `pkg/`. The package lives at the module root because
that's the import path users will use.

**Multi-module monorepo (with `go.work`):**
```
example.com/myorg/
├── go.work               # local-dev only, gitignored
├── shared-lib/
│   ├── go.mod
│   └── ...
├── api-service/
│   ├── go.mod
│   └── ...
└── worker-service/
    ├── go.mod
    └── ...
```
Each subdirectory is its own module with its own version. The
`go.work` file at the root coordinates local development;
production builds use the modules' own `go.mod` directives.

A non-exhaustive sample of real-world layouts:

* **Kubernetes:** `cmd/`, `pkg/`, `staging/`, `vendor/`. The
  `staging/` directory is an internal pre-publish area for libraries
  that will become their own modules. Vendored. Hundreds of `cmd/`
  binaries.
* **Docker / Moby:** `cmd/`, `daemon/`, `client/`, `image/`. No
  `pkg/`. Highly idiomatic.
* **Hugo:** `commands/`, `hugolib/`, `parser/`, `tpl/`. Idiosyncratic,
  no `cmd/`. Old-style, predates the convention.
* **Caddy:** `cmd/caddy/`, `caddyhttp/`, `caddyconfig/`,
  `modules/`. Composes via plugin modules.
* **Cobra:** `cobra.go` at the root, no subdirectories. Pure
  library style.
* **`golang/go` itself:** Its own `src/` is the toolchain source;
  packages at the top of `src/` are the standard library.
  Standard-library style is its own thing — don't imitate it for
  application code.

There is no winner. Pick one of the three healthy patterns above
based on your project type and stick to it.

---

## 6. Real-World Production Use Cases

**Multi-binary service repo at a startup.** A 30-person team
building three services (API, worker, scheduler) and a CLI puts
them all in one Git repo, one Go module. Layout: `cmd/{api,worker,
scheduler,cli}/main.go` plus `internal/{auth,store,events,jobs}/`.
The single `go.mod` keeps cross-service refactors trivial; the
`internal/` boundary keeps domain code from leaking out. Compile
times stay reasonable up to ~200 KLOC at this layout.

**Library with a strong public API.** A team open-sourcing a
distributed-tracing library publishes
`github.com/example/tracing` with the public API at the root
(`tracer.go`, `span.go`, `propagation.go`) and implementation
details under `internal/encoding/` and `internal/sampling/`. Users
import `github.com/example/tracing`; the `internal/` packages are
*physically unreachable* from user code. This lets the library
authors refactor internals without breaking users.

**Monorepo with independent versioning.** A platform team manages
five services and three shared libraries in one Git repo, but
wants each component to have its own version (so a bug fix in
shared-lib can ship without redeploying every service). Each lives
in its own subdirectory with its own `go.mod`. A root `go.work`
file (gitignored) lets engineers edit shared-lib alongside a
service and see changes immediately. CI builds each module
independently.

**Plugin-style architecture.** A platform exposes plugin interfaces
in `pkg/plugins/` and consumes them through `internal/registry/`.
External users of the platform implement the plugin interfaces in
their *own* modules, then `go install` them. This is how Caddy and
Hugo extension systems work.

**`vendor/`-committed regulated build.** A bank with strict
supply-chain controls runs `go mod vendor` and commits the
`vendor/` directory. Builds use `-mod=vendor` so they never reach
the network. Security review can audit every dep by reading
`vendor/`. The build is deterministic without any external service.

---

## 7. Beginner-Friendly Explanation

If you're new to all of this, the rules are short:

1. **A module** is your project. Its `go.mod` file lives at the
   project root. It has a name (its import path).
2. **A package** is a folder of `.go` files with the same `package
   <name>` line at the top. The folder's name is *usually* the
   package name, but doesn't have to be.
3. **The import path** of a package is `<module path>/<directory
   path>`. So if your module is `github.com/me/notes` and you have
   a folder `internal/store/`, that package's import path is
   `github.com/me/notes/internal/store`.
4. **Capital letters mean public.** A function called `Save` is
   accessible from other packages; a function called `save` is
   not. Same for types, constants, variables, struct fields.
5. **Anything inside an `internal/` directory** is private to the
   module. Code outside the module can't import it. It's a hard
   compiler rule.
6. **Each directory has exactly one package.** Two `.go` files in
   the same folder must share a `package` line. (Test files can
   use `<name>_test`, but that's a different topic.)

That's it. Three nouns (module, package, workspace), one
visibility rule (capitalization), one privacy rule (`internal/`).

> **Coming From Java —** A *module* is roughly a Maven `pom.xml`
> project. A *package* is roughly a Java package. The visibility
> rule is the surprise: there's no `public`/`private` keyword;
> capitalization does the same job.

> **Coming From Python —** A *module* is the project. A *package*
> is closer to a Python module than a Python package. There is no
> `__init__.py`; the directory itself is the package.

> **Coming From JavaScript —** A *module* is roughly an `npm`
> package. A *package* is roughly a directory of files that re-
> export a coherent API. There's no separate "barrel file"; every
> exported identifier in any file in the directory is part of the
> package's surface.

> **Coming From Rust —** A *module* is roughly a `crate`. A *Go
> package* is between a Rust module and a crate. Rust's `pub`
> keyword is replaced by capitalization; Rust's `pub(crate)` is
> replaced by `internal/`.

---

## 8. Deep Technical Explanation

Now for the rules in their full precision.

### 8.1. The module (`go.mod`)

A module is a directory tree rooted at a directory containing
`go.mod`. The `module` directive in that file sets the *module path*
— a string that is also the import prefix for every package in the
module.

```go
module github.com/example/notes

go 1.22.0

require (
    github.com/jackc/pgx/v5 v5.5.0
)
```

Rules:

* The module path is *not* the directory name on disk. It's
  declared. Convention: it matches the Git URL where the module is
  hosted, so anyone can `go get` it.
* The module path *is* the import prefix. If you have a package at
  `<module>/store/postgres`, its import path is
  `github.com/example/notes/store/postgres`.
* Every Go file under the module belongs to *some* package; the
  package's import path is computed mechanically.
* You can nest modules — i.e. a subdirectory can have its own
  `go.mod`. This *removes* that subtree from the parent module.
  The parent treats the nested module as a separate dependency.
* A module path with a major version v2 or higher must include the
  major version: `github.com/foo/bar/v2`. This forces breaking-
  change visibility into the import statement.

### 8.2. The package (`package <name>`)

A package is a directory of `.go` files. Rules:

* All non-test files in a directory must share the same `package
  <name>` declaration. Mixing is a compile error.
* The package name does **not** have to match the directory name,
  but it usually does. Common exceptions: `main` (for executables),
  `<dir>_test` (for external test packages).
* The package name appears unprefixed in import-using code:
  `import "github.com/example/notes/store/postgres"` lets you call
  `postgres.Open(...)`.
* Package names should be short, lowercase, no underscores or
  mixedCaps. `httputil`, not `httpUtil`. `userprefs`, not
  `userPrefs`.
* If a directory has a `<name>_test.go` file, that file's package
  may be `<name>_test` instead of `<name>`. This creates an
  *external* test package that can only access the exported
  surface of `<name>`. Use this when you want to test from a
  consumer's perspective.

### 8.3. The import graph and the cycle rule

Imports form a directed graph. Go enforces it must be acyclic:
package A cannot import B if B (transitively) imports A. The
compiler reports a clear error if you try.

This rule has consequences that catch newcomers:

* If you find yourself "needing" a cyclic import, the design is
  wrong. Extract the shared code into a third package both can
  depend on.
* You cannot have "two halves" of a domain model in two packages
  that reference each other. Either keep them in one package or
  decouple them with an interface.

### 8.4. Visibility: capitalization

Go's visibility rule is famously simple:

* An identifier whose first letter is **uppercase** is *exported*
  (visible to other packages).
* An identifier whose first letter is **lowercase** is *unexported*
  (visible only within its package).

This applies to every name: types, functions, constants, variables,
struct fields, methods. You cannot annotate a name with
`public`/`private` keywords; the case does it.

```go
package store

type User struct {       // exported
    ID    int64          // exported field
    name  string         // unexported field — invisible outside
}

func NewUser(id int64) *User { return &User{ID: id} }  // exported
func (u *User) save() {}                               // unexported
```

A common stumbling block: returning a struct with unexported fields
from an exported function is *fine* — the caller has the value, just
can't read or write the unexported fields directly. This is the
right way to enforce invariants.

### 8.5. The `internal/` rule

A package whose import path contains `/internal/<anything>` may
**only** be imported by packages rooted at the directory **above**
the `internal/`. This is enforced by the toolchain.

```
example.com/platform/
├── api/
│   └── handler.go              // can import "example.com/platform/internal/auth"
├── internal/
│   └── auth/auth.go            // package auth
└── ...
```

If `someone-else.com/their/code` tried to `import
"example.com/platform/internal/auth"`, the build fails with:

```
use of internal package not allowed
```

`internal/` can be nested. `pkg/internal/foo/internal/bar` is
private to packages at and below `pkg/internal/foo`. This is how
library authors expose a *limited* internal API to a subset of the
codebase without exposing it module-wide.

### 8.6. The workspace (`go.work`)

A workspace is a directory containing a `go.work` file:

```text
go 1.22.0

use (
    ./shared-lib
    ./api-service
    ./worker-service
)
```

Within a workspace, the `go` toolchain prefers local source over
proxy-fetched modules. A `replace` in any of the modules' `go.mod`
files is overridden by the workspace's `use` directive.

When to use:

* Editing two modules simultaneously locally.
* Monorepos where the modules are intentionally separate but
  developed together.
* Trying out a fork of a third-party module against your service.

When not to use:

* In CI. CI should build each module independently from its
  committed `go.mod`.
* In committed repo state. **Don't commit `go.work`.** It's
  developer-local. Add it to `.gitignore`. (`go.work.sum` is also
  developer-local; same rule.)

### 8.7. Vendoring

`go mod vendor` writes the full transitive dep tree into a
`vendor/` directory at the module root. Then `go build -mod=vendor`
uses that directory exclusively, never touching the module cache or
the network.

Why vendor:

* Air-gapped builds.
* Compliance audits that need every byte of the dep tree visible
  in `git log`.
* Defense against upstream deletion (the module proxy already
  protects against this for public modules; vendoring is belt-and-
  suspenders).

Why not:

* Repo bloat. Vendoring is typically tens of MB.
* Review noise. Every dep update is hundreds of file changes.
* The proxy already provides reproducibility for public modules.

In 2026, vendoring is common in regulated industries and rare
elsewhere. Kubernetes vendors. Most others don't.

### 8.8. The "standard layout" non-debate

There is no official Go standard project layout. The Go team
deliberately publishes none. The community-maintained
`golang-standards/project-layout` repo is *not* official, and the
Go team has, on multiple occasions, said it's misleading because it
implies an authority it doesn't have.

What is *real* convention:

* `cmd/<name>/main.go` for binaries when there are multiple.
* `internal/` for module-private packages.
* `go.mod` at the module root.
* Tests in `_test.go` files alongside the code they test.

Beyond that, layout is a project decision. Different is fine; the
language doesn't care.

> **Architecture Review —** A team that adopts the unofficial
> "standard layout" with `pkg/`, `internal/`, `api/`, `web/`,
> `configs/`, `init/`, `scripts/`, `build/`, `deployments/`, `test/`,
> `docs/`, `tools/`, `examples/`, `third_party/`, `githooks/`,
> `assets/`, `website/` directories at the root of every service is
> usually carrying ceremony from a different stack. Three top-
> level dirs (`cmd/`, `internal/`, optionally `pkg/`) is enough for
> 95% of real services.

---

## 9. Internal Working (How Go Handles It)

* **Module resolution.** When you run `go build`, the toolchain
  walks up from the current directory looking for a `go.mod` — that
  becomes the module root. If you're in a `go.work` workspace, the
  workspace root is found similarly. The module path from `go.mod`
  becomes the import-path prefix.
* **Package compilation.** Each directory that holds Go files is
  one package; the toolchain compiles them as a unit, producing one
  package object archive. Cross-package boundaries are resolved
  through the import graph.
* **Import resolution order.** When a Go file says `import
  "example.com/foo/bar"`, the toolchain checks: (1) is there a
  workspace-local module at this path? (2) is there a vendored copy?
  (3) is there a cached copy in `$GOMODCACHE`? (4) fetch from the
  proxy. This order is documented in `go help importpath`.
* **The `internal/` enforcement** is implemented in `cmd/go/internal/
  modload/import.go`. The check happens during import resolution,
  not at compile time, so an attempt to import an `internal/`
  package from outside the allowed scope fails before the compiler
  ever sees the code.
* **Workspace overrides.** With `go.work`, the toolchain marks any
  module listed under `use` as "available locally"; `replace`
  directives are computed implicitly.

If you ever wonder where Go thinks a package is coming from, run
`go list -m -json <pkg>`. The `Path`, `Dir`, `Version`, `GoMod`
fields reveal the full resolution.

---

## 10. Syntax Breakdown

The "syntax" of a Go project is mostly file paths. A canonical
example:

```
github.com/example/notes/
├── go.mod                              # module github.com/example/notes
├── go.sum
├── README.md
├── cmd/
│   └── notesd/
│       └── main.go                     # package main, the binary entry
├── internal/
│   ├── note/
│   │   ├── note.go                     # package note (domain types)
│   │   ├── note_test.go                # internal tests
│   │   └── note_storage.go
│   └── store/
│       └── postgres/
│           ├── postgres.go             # package postgres
│           └── postgres_test.go
└── pkg/
    └── notesclient/
        └── client.go                   # package notesclient (public API)
```

Computed import paths:

* `github.com/example/notes/cmd/notesd` (executable, `package main`)
* `github.com/example/notes/internal/note` (private)
* `github.com/example/notes/internal/store/postgres` (private)
* `github.com/example/notes/pkg/notesclient` (public client SDK)

---

## 11. Multiple Practical Examples

### Example 1 — `examples/01_module_anatomy`

```bash
go run ./examples/01_module_anatomy
```

A program that, given a Go file path on the command line, prints
that file's resolved module path, package import path, and exported-
identifier count. Use it to confirm your mental model of the rules.

### Example 2 — `examples/02_workspace_demo`

A two-module workspace with a `go.work` file. A library module
under `lib/` exports a function; a service module under `svc/`
imports it.

```bash
cd ./examples/02_workspace_demo
go run ./svc
```

The `go.work` file in this directory binds the two modules
together. Edit `lib/lib.go`, save, re-run `go run ./svc` — the
change is picked up immediately, with no `go get` step in between.

### Example 3 — `examples/03_layout_choices`

Not a runnable program but a documented set of layout templates,
in three subfolders showing the three healthy patterns: a flat
library, a `cmd/` + `internal/` service, and a multi-module
monorepo. Read each `LAYOUT.md` for the rationale.

---

## 12. Good vs Bad Examples

**Good `cmd/` layout:**

```
cmd/
├── api/main.go        # only main, no logic; calls internal/api.Run()
├── worker/main.go     # only main, no logic; calls internal/worker.Run()
└── migrate/main.go    # only main; calls internal/store.Migrate()
```

**Bad:**

```
cmd/
└── server/
    ├── main.go         # 500-line main with HTTP handlers inline
    ├── auth.go         # accidentally also package main
    └── db.go           # accidentally also package main
```

Why bad: putting business logic in `package main` makes it
untestable from any other package (you can't import `package main`).
The fix is to put the logic under `internal/` and have `main.go`
just call `internal/<name>.Run(ctx)`.

**Good package naming:**

```go
package httputil   // no underscores, no mixedCaps
package store
package note
```

**Bad:**

```go
package http_util  // underscores
package storeMain  // mixedCaps
package util       // too vague
```

`util` is a smell because it has no domain meaning; it accumulates
unrelated helpers and becomes a dependency hub for the whole
codebase. Pick a focused name.

**Good visibility:**

```go
type Config struct {
    Port    int      // exported field
    Timeout time.Duration
    secret  string   // unexported, set by the constructor
}

func NewConfig(secret string) *Config { ... }
```

**Bad:**

```go
type Config struct {
    Port    int
    Timeout time.Duration
    Secret  string  // exported — anyone can read or overwrite
}
```

The bad form leaks secrets to the package's full surface and lets
callers bypass any constructor logic.

---

## 13. Common Mistakes

1. **Module path != Git URL.** Setting the module path to `notes`
   instead of `github.com/example/notes` makes the module
   non-importable from anywhere except itself. Always use the full
   URL form.
2. **Two packages in one directory.** Putting `package foo` and
   `package bar` files in the same folder is a compile error. One
   directory = one package.
3. **`internal/` placed wrong.** A common mistake is to place
   `internal/foo` at the *repo* root above `go.mod`. The `internal/`
   rule is module-relative; it must live within a module.
4. **Trying to `import "C"` without CGO.** `package main` works fine
   without CGO, but if you `import "C"`, you've opted in. You'll lose
   the static-binary story; act deliberately.
5. **Cyclic imports caused by "model" packages.** A `model/`
   package that references `repository/`, which references
   `model/`. The fix: keep models pure data with no behavior, or
   put the repository in the same package as the model.
6. **Committing `go.work`.** Symptom: teammates' builds break
   because the workspace's local paths don't exist on their
   machines. Fix: `.gitignore` it.
7. **Using `pkg/` for everything.** `pkg/` is for *intentionally
   exported* packages. `pkg/internal/...` is contradictory. If
   everything goes in `pkg/`, it's just a synonym for "the rest
   of the code"; flatten it.
8. **Treating `cmd/` as logic.** `cmd/<name>/main.go` should be
   tiny — flag parsing, logger setup, then a call into `internal/`.
9. **Ignoring the `internal/` mechanism.** You don't need a
   convention like "don't import this from outside" enforced by
   review when the toolchain can enforce it for free. Use
   `internal/`.
10. **Mixing single-module and multi-module styles.** Either it's
    one module or many. Don't sprinkle `go.mod` into random
    subdirectories of a single-module project.

---

## 14. Debugging Tips

* **`go list -m`** prints the module path of the current directory's
  module.
* **`go list -m all`** prints every module in the dep graph.
* **`go list <pkg>`** prints the import path of the named package.
* **`go list -deps -test ./...`** lists every package transitively
  reachable, including test deps.
* **`go env GOMOD`** prints the `go.mod` the toolchain is using for
  the current directory.
* **`go env GOWORK`** prints the active `go.work` file (or "off" if
  none).
* **`go mod why <pkg>`** explains why a transitive dep is in the
  graph.
* **`go vet ./...`** catches a lot of structural mistakes (e.g.
  unused imports, shadowed variables) that other tools miss.

---

## 15. Performance Considerations

* **Compile times scale with package count and import-graph
  density.** Splitting one large package into ten small ones rarely
  helps and sometimes hurts (each new edge re-pulls compiled
  dependencies).
* **Avoid one-file-one-package patterns.** A package with thirty
  files is fine. A directory tree with thirty packages of one file
  each is slow to build.
* **`internal/` doesn't affect build speed**, only API surface.
* **`vendor/`** can speed up CI cold-start times because it
  eliminates module fetches, at the cost of repo size.
* **The build cache is per-package.** Changing one file in a
  package re-compiles only that package and its dependents, not the
  whole tree. This is why "split into packages" is rarely a build-
  speed optimization.

---

## 16. Security Considerations

* **`internal/` is a security boundary at the API level**, not at
  the runtime level. Code under `internal/` is still in the same
  process as the rest of the module; it doesn't get sandboxed.
  Don't confuse compile-time visibility with runtime isolation.
* **Module path squatting.** A project named `github.com/yourorg/
  shiny-tool` shouldn't be importable as `shinytool` or
  `shiny_tool`. Choose your module path carefully; renaming is
  painful.
* **`go.work` leaks paths.** A committed `go.work` file with
  developer-local paths can leak directory names from a
  contributor's laptop. Always `.gitignore`.
* **Vendored deps need scanning too.** `govulncheck` works on
  vendored trees. Run it.
* **`unsafe` imports are visible at the package level.** A project
  using `unsafe` should declare it loudly. `go list -deps -f
  '{{.ImportPath}}' ./... | xargs -I{} go list -f '{{if .Imports
  | hasUnsafe}}…' {}` is a useful audit.

---

## 17. Senior Engineer Best Practices

1. **Module path = canonical Git URL.** Always.
2. **Pin `go` version in `go.mod`** to a precise version
   (`go 1.22.0`).
3. **Use `internal/` aggressively.** If a package isn't an
   intentional public API, it goes in `internal/`.
4. **Keep `cmd/<name>/main.go` short.** No logic; only wiring.
   Logic goes in `internal/<name>/Run(ctx)`.
5. **Don't use `pkg/`** unless you're publishing a public SDK
   inside a service repo. Flat is fine.
6. **One package per directory; one purpose per package.** Don't
   make a `util` package; pick a focused name.
7. **Doc comment every exported identifier**, even if it feels
   redundant. It's the contract.
8. **Add a `doc.go` file** to packages that need a paragraph of
   overview docs. It's a convention.
9. **`.gitignore` `go.work*`.** Commit `go.mod` and `go.sum`
   only.
10. **Don't pre-emptively split into modules.** One module is
    simpler. Split when you have a real reason: independent
    versioning, separate deploy lifecycles, or genuine third-party
    consumers.

---

## 18. Interview Questions

1. *(junior)* What is the difference between a *module* and a
   *package* in Go?
2. *(junior)* How do I make a function public (visible to other
   packages)?
3. *(mid)* What does the `internal/` directory do?
4. *(mid)* Why is `pkg/` controversial?
5. *(mid)* When should you use `go.work`?
6. *(senior)* Walk me through the import-resolution order for a
   package: workspace, vendor, module cache, proxy. When does each
   apply?
7. *(senior)* You have two packages that need to reference each
   other. Go won't allow it. What are your options?
8. *(senior)* When would you split a single-module repo into
   multiple modules?
9. *(staff)* Design the layout for a 50-engineer monorepo
   containing 8 services and 4 shared libraries that need
   independent versioning. Defend your choices.

---

## 19. Interview Answers

1. **Module:** unit of versioning and distribution; declared by
   `go.mod`; one per Git repo (typically). **Package:** unit of
   compilation and naming; one per directory; declared by
   `package <name>`. A module contains many packages.

2. Capitalize the first letter of its name. `Save` is exported,
   `save` is not. The compiler enforces it; if it builds, you've
   gotten the public/private split right.

3. **`internal/`** is a directory name with special meaning to the
   toolchain. A package under `<module>/.../internal/<x>` may only
   be imported by packages rooted at the directory *above* the
   `internal/`. This gives library authors a compiler-enforced
   private namespace, useful for hiding implementation details
   from external consumers.

4. **`pkg/`** is a community convention with no official endorsement.
   The Go team has said it's misleading. Some projects use it for
   "public" packages and `internal/` for private; others put
   everything at the root. Both work. The criticism: it duplicates
   what `internal/` already enforces and adds an extra path
   component to every import. It's neither wrong nor right, but
   it's a deliberate choice you should be able to defend.

5. **`go.work`** is for local-development across multiple modules:
   a monorepo where you're editing both a service and a library it
   depends on, and want changes in the library picked up immediately
   without committing a `replace`. Don't commit `go.work` — it's
   developer-local. CI uses each module's own `go.mod`.

6. The toolchain's resolution order (when `go build` needs a
   package): (a) **Workspace local** — if `go.work` is active and
   the package's module is in `use`, use the local copy. (b)
   **Vendor** — if `vendor/` exists and `-mod=vendor` (or
   GOFLAGS) is set, look there. (c) **Module cache** — if the
   resolved version is already in `$GOMODCACHE`, use it. (d)
   **Proxy** — fetch from `$GOPROXY` (typically
   `proxy.golang.org`), verify against `go.sum`, place in cache,
   use it.

7. Three options. (a) Combine them — if they're really tied, they
   should be one package. (b) Extract the shared code into a
   third package both can depend on. (c) Decouple via interface —
   define an interface in package A; let A's code accept that
   interface; have B implement it. The interface lives in A, the
   concrete type in B, but A doesn't import B. The choice depends
   on which design intent matches the domain.

8. **Reasons to split into multiple modules:** (i) different
   release cadences — a shared lib needs to ship hotfixes
   independently of the consuming services; (ii) external
   consumers — the library has users outside this repo who want
   to pin to a specific version; (iii) compile-time isolation —
   the modules' deps are large and you want CI to skip rebuilding
   one when only the other changed. **Reasons not to split:**
   coordination overhead, version-skew bugs, and the fact that one
   module is simpler.

9. Top-level: each service in its own directory with its own
   `go.mod`; each shared lib in its own directory with its own
   `go.mod`. A root `go.work` (gitignored) for local dev. CI
   pipelines per module. Versioning: services as `v0.x.y`
   internal-only; shared libs as `v1.x.y` with semver discipline.
   Defend: independent versioning is the killer requirement; one
   `go.mod` would force services to upgrade in lockstep.
   Counter-argument I'd weigh: a single `go.mod` is simpler and
   modern Go tooling makes per-package CI possible without
   per-module `go.mod`. I'd start single-module if shared libs
   don't yet have external consumers, then split when an actual
   pain point emerges.

---

## 20. Hands-On Exercises

**Exercise 4.1 — Initialize a layout.** Run
`go run ./exercises/01_init_layout` from this chapter folder.
The program prompts for a project type (library / single-binary
service / multi-binary service / monorepo) and a module path,
then generates the appropriate directory tree with stubs.

**Exercise 4.2 — Refactor an existing package.** Pick any
directory in your work codebase that has grown into a "util"-style
package. Split it into focused, named packages. Note the import-
graph edges you create; check for cycles.

**Exercise 4.3 ★ — Multi-module monorepo with `go.work`.**
Create a directory tree with two modules (a `lib` and a `svc`).
Wire them up with `go.work`. Make a change in `lib` and observe
that `svc` picks it up without `go get`. Then remove `go.work` and
observe what breaks.

---

## 21. Mini Project Tasks

**Task — Layout linter.** Write a tool that walks a Go module and
reports layout findings:

* Packages named `util`, `helpers`, `common`, or `misc`.
* Packages with cyclic imports (the toolchain catches these too,
  but a friendly preview is useful).
* `cmd/<name>/main.go` files longer than 100 lines.
* `internal/` directories at the wrong level.

You'll have most of the skills by Chapter 18 (slices) and Chapter
38 (file I/O); revisit then.

---

## 22. Chapter Summary

* A Go *module* is a versioning unit, declared by `go.mod`. A *package*
  is a directory of Go files. A *workspace* (`go.work`) is a
  developer-local view of multiple modules.
* The module path = the import-path prefix, by mechanical
  derivation from directory layout.
* Visibility is controlled by capitalization. There is no
  `public`/`private` keyword.
* `internal/` is a compiler-enforced privacy boundary. Use it.
* The Go team publishes no "official" project layout. Three
  healthy patterns dominate: flat library, `cmd/`+`internal/`
  service, and multi-module monorepo with `go.work`.
* `cmd/<name>/main.go` should be small. Logic belongs under
  `internal/`.
* Commit `go.mod` and `go.sum`. Don't commit `go.work`.
* Don't pre-emptively split into multiple modules. One module is
  simpler; split when independent versioning becomes a real need.

Updated working definition: *a Go project's structure is its
module boundary plus its tree of packages, with import paths
derived mechanically from directory layout. Visibility is
controlled by capitalization and by `internal/`. Layout above the
package level is project-specific, and three healthy patterns
cover most cases.*

---

## 23. Advanced Follow-up Concepts

* **`golang.org/ref/mod`** — the canonical reference for module
  semantics.
* **`golang.org/cmd/go/#hdr-Module_layouts`** — the toolchain's
  documentation on layouts.
* **Russ Cox, "Go modules: v2 and beyond"** (2019) — why major
  versions appear in import paths.
* **Russ Cox, "Why Build Systems Are Hard"** — context for why Go
  refuses configurable build paths.
* **`golang-standards/project-layout`** — *not official*, but
  worth reading once to see the maximalist take so you can choose
  your own.
* **Brian Ketelsen, "Go best practices, six years in"** (2016) —
  still mostly valid; layout patterns from a senior practitioner.

> **End of Chapter 4.** Move on to [Chapter 5 — How `go run`,
> `go build`, and `go install` Actually Work
> ](../chapter05_go_run_build_install/README.md).
