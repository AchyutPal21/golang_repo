# Chapter 4 — Exercises

## Exercise 4.1 — Inspect a real module

**Goal.** Internalize the relationship between file path, module
path, package, and import path.

**Task.** Run the layout-anatomy tool on three different files:

```bash
go run ./examples/01_module_anatomy ./examples/01_module_anatomy/main.go
go run ./examples/01_module_anatomy ../chapter01_why_go_exists/examples/01_hello/main.go
go run ./examples/01_module_anatomy ../chapter01_why_go_exists/examples/03_http_server/main.go
```

For each, write down (a) the module root, (b) the package import
path, (c) whether it's `internal/` or public.

**Acceptance.** Your answers match the tool's output and you can
explain *why*.

---

## Exercise 4.2 — Run the workspace demo

**Goal.** See `go.work` actually do its job.

**Task.**

1. `cd examples/02_workspace_demo && go run ./svc`. Observe the
   greeting from `lib`.
2. Edit `lib/lib.go` — change the greeting text.
3. Re-run `go run ./svc`. Confirm the new text appears immediately,
   without any `go get` step.
4. Move `go.work` aside (`mv go.work go.work.off`) and try again.
   Read the error.
5. Restore `go.work`.

**Acceptance.** You can explain why the build broke without
`go.work` and why no `go get` was needed with it.

---

## Exercise 4.3 ★ — Initialize a layout

**Goal.** Practice making a layout decision and committing to it.

**Task.** Decide on a project type for a hypothetical service called
`tasks`:

* HTTP API for creating, listing, completing tasks
* Postgres for storage
* Redis for rate limiting
* A CLI tool for admin operations

Sketch the directory tree on paper. Pick: flat / `cmd/`+`internal/` /
multi-module. Then create the directories with `go mod init` to
match.

**Acceptance.** A working `go build ./...` from the project root,
and a one-paragraph design note defending your layout choice.
