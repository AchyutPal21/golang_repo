// FILE: book/part1_foundations/chapter04_workspace_structure/examples/03_layout_choices/main.go
// CHAPTER: 04 — The Go Workspace and Project Structure
// TOPIC: A printable summary of the three healthy layout patterns.
//
// Run (from the chapter folder):
//   go run ./examples/03_layout_choices
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   Layout decisions are made under pressure on day one of a project. This
//   prints the three healthy patterns side-by-side as a decision aid: pick
//   one, defend it. The patterns are described in the chapter prose; this
//   is the ASCII version you'd paste into a design doc.
// ─────────────────────────────────────────────────────────────────────────────

package main

import "fmt"

func main() {
	fmt.Print(`
THREE HEALTHY GO PROJECT LAYOUTS
═══════════════════════════════════════════════════════════════════

A. FLAT LIBRARY  —  for: a Go library with one purpose

   github.com/example/redis-client/
   ├── go.mod
   ├── client.go            # package "redis"
   ├── pool.go
   ├── doc.go               # package overview docs
   ├── client_test.go
   └── internal/
       └── proto/...        # implementation detail; private

   When to use: you're publishing a library. The package lives at
   the module root because that's the import path users see.


B. SERVICE  —  for: an executable, possibly multiple binaries

   github.com/example/notes/
   ├── go.mod
   ├── README.md
   ├── cmd/
   │   ├── notesd/main.go        # the daemon
   │   └── notes-cli/main.go     # an admin CLI
   └── internal/
       ├── api/                  # HTTP handlers
       ├── store/                # persistence
       │   └── postgres/
       ├── auth/
       └── jobs/

   When to use: you're building a service or set of services that
   share code. cmd/<name>/main.go is the entry; logic lives under
   internal/, which makes it private to this module and testable
   from any other internal/ package.


C. MULTI-MODULE MONOREPO  —  for: independent versioning

   github.com/example/platform/
   ├── go.work               # gitignored — local dev only
   ├── shared-tracing/       # its own module: example.com/platform/tracing
   │   ├── go.mod
   │   └── tracing.go
   ├── shared-store/         # its own module
   │   ├── go.mod
   │   └── store.go
   ├── api-service/          # its own module
   │   ├── go.mod
   │   ├── main.go
   │   └── internal/...
   └── worker-service/       # its own module
       ├── go.mod
       ├── main.go
       └── internal/...

   When to use: shared libraries need to ship hotfixes independently
   of consuming services, or external consumers want to pin to a
   specific version of a shared library.

═══════════════════════════════════════════════════════════════════
DECISION ALGORITHM

1. Are you publishing a single library? → Pattern A (flat).
2. Are you building one service or a tightly-coupled set of
   services? → Pattern B (cmd/ + internal/). Default for ~80% of
   real projects.
3. Do you need INDEPENDENT VERSIONING for shared libs in this
   repo? → Pattern C (multi-module). Reach for this only when
   the pain is real; one go.mod is simpler.

Avoid: pkg/ unless you're publishing a public SDK inside a service
repo. Avoid: layouts copied from "golang-standards/project-layout"
without considering whether each directory earns its place.

`)
}
