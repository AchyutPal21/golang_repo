# Example 02 — Workspace Demo

A two-module workspace bound by `go.work`.

## Layout

```
02_workspace_demo/
├── go.work           # binds lib/ and svc/ for local development
├── lib/
│   ├── go.mod        # module example.com/wsdemo/lib
│   └── lib.go        # package lib
└── svc/
    ├── go.mod        # module example.com/wsdemo/svc, requires lib v0.0.0
    └── main.go       # imports example.com/wsdemo/lib
```

## Run it

From this directory (`examples/02_workspace_demo`):

```bash
go run ./svc
```

You should see output from `lib.Greeting`. Now edit `lib/lib.go` to
change the greeting string, save, re-run — the change is picked up
immediately, *without* `go get` or any version bump.

## Try removing the workspace

```bash
mv go.work go.work.disabled
go run ./svc
```

The build fails with something like *"example.com/wsdemo/lib@v0.0.0:
reading example.com/wsdemo/lib/go.mod: no such module"* — because
without `go.work`, the toolchain tries to resolve `lib v0.0.0` from
the public proxy, which has never heard of `example.com/wsdemo/lib`.

Restore the workspace:

```bash
mv go.work.disabled go.work
```

Build works again. This is exactly the use case `go.work` was
designed for: local-only resolution of modules that haven't been
(or shouldn't be) published.

## Why isn't this a `replace` directive?

You *could* add `replace example.com/wsdemo/lib => ../lib` to
`svc/go.mod`. That would also work. But it would be **committed**,
which means every other developer (and CI) would also be redirected
to a local path that doesn't exist on their machine. `go.work` is
gitignored by convention; `replace` in `go.mod` is committed.
Use `go.work` for "my laptop right now"; reserve `replace` for
"this fork should be used by everyone."
