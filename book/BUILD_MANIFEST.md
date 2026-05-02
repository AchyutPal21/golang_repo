# Build Manifest — How This Book Gets Built

> A working document for the author (you) and any collaborator. Tracks
> *what* needs to be produced, *which existing assets feed into it*, and the
> *state* of every chapter. This file is the single source of truth for
> "what's left." `BOOK.md` is the table of contents; this is the project
> plan.

---

## Build phases

The book is built in seven phases, mirroring the seven parts. Each phase
has a target Go-version baseline and a definition of done.

| Phase | Part | Chapters | Target | DoD |
| --- | --- | --- | --- | --- |
| 1 | Foundations | 1–7 | Go 1.22 | All 7 chapter folders runnable + READMEs complete + 1 reviewer pass |
| 2 | Core Language | 8–25 | Go 1.22 | All 18 chapters; `go test ./...` green; cross-refs verified |
| 3 | Designing Software | 26–40 | Go 1.22 | Includes a worked example service used as a running thread |
| 4 | Concurrency & Systems | 41–56 | Go 1.22 | Each chapter has at least one `-race`-clean concurrent program |
| 5 | Building Backends | 57–81 | Go 1.22 | `docker-compose.yml` per chapter where infra is needed; CI green |
| 6 | Production Engineering | 82–102 | Go 1.22 | Profiling, tracing, deployment chapters demonstrated against the running service |
| 7 | Capstone Projects | A–J | Go 1.22 | 10 standalone projects; each has its own README, deploy script, and "scaling discussion" appendix |

Total: **102 numbered chapters + 10 capstone projects + 7 appendices.**

---

## Existing assets and how they feed in

The repo already contains substantial material under
`golang-mastery-updated/`. Re-use, do not duplicate:

| Existing path | Lines | Feeds into |
| --- | --- | --- |
| `01_fundamentals/01_how_go_runs/` | 171 | Chapter 5 (`go run`/`build`/`install`) |
| `01_fundamentals/02_variables/` | 221 | Chapter 8 (Variables, Constants, Zero Value) |
| `01_fundamentals/03–05_types_*` | 619 | Chapter 9 (Type System) |
| `01_fundamentals/06_constants_iota/` | 272 | Chapter 8 |
| `01_fundamentals/07_operators/` | 235 | Chapter 11 |
| `01_fundamentals/08_control_flow/` | 288 | Chapter 12 |
| `01_fundamentals/09_pointers/` | 250 | Chapter 16 |
| `01_fundamentals/10_defer_panic_recover/` | 257 | Chapter 15 |
| `01_fundamentals/11_fmt_printing/` | 266 | Chapter 7 + Appendix D |
| `02_functions/01–09_*` | 2,723 | Chapters 13–15, 21 |
| `03_structs_methods_interfaces/01–10_*` | 3,061 | Chapters 20–23, 27 |
| `04_error_handling/01–08_*` | 2,590 | Chapters 36–37 |
| `05_collections/01–08_*` | 2,394 | Chapters 17–19 |
| `06_concurrency/01–10_*` | 2,659 | Chapters 41–50 |
| `07_packages_modules/01–07_*` | 2,267 | Chapters 2, 4, 82 |
| `08_standard_library/01–10_*` | 3,141 | Chapters 38–39, 47, Appendix D |
| `09_generics/01–07_*` | 2,254 | Chapter 24 |
| `10_advanced_patterns/01–08_*` | 2,808 | Chapters 25, 31–33, 87–89 |

**Adaptation rule:** existing files are *raw material*, not finished
chapters. Each is rewritten to fit the 23-section structure, has its
heavy-comment density preserved, and is split or merged as needed to map
onto the chapter granularity in `BOOK.md`.

**Preservation rule:** the original folder is kept at its current path until
the corresponding new chapter passes the quality gates in
`CHAPTER_TEMPLATE.md`. Only then is the original retired (moved to
`legacy/` rather than deleted, in case a future revision wants to compare).

---

## Per-chapter status table

Status legend:
- `▢ planned` — listed in BOOK.md, no work done
- `◐ drafting` — content being written
- `◑ review` — content drafted, needs editorial pass
- `■ done` — passes all 10 quality gates
- `★ canonical` — done *and* serves as a reference exemplar

### Part I — Foundations

| # | Chapter | Status | Source asset(s) | Notes |
| --- | --- | --- | --- | --- |
| 1 | Why Go Exists | ★ canonical | (new) | The reference exemplar. Other chapters must match this depth. |
| 2 | A Map of the Go Ecosystem | ■ done | (new) | Toolchain tour + stdlib inventory; 3 examples; vet/build/run green |
| 3 | Installing Go and Setting Up | ■ done | (new) | OS matrix, install self-check, editor smoke-test |
| 4 | The Go Workspace and Project Structure | ■ done | (new) | go.work, layouts; module anatomy + workspace demo with two nested modules |
| 5 | How `go run`/`build`/`install` Work | ■ done | `01_fundamentals/01_how_go_runs` | Build cache, cross-compile, -ldflags, build inspector |
| 6 | Coming From Another Language | ■ done | (new) | Java/Py/JS/C++/Rust transfer tables; Python→Go and JS→Go side-by-sides |
| 7 | Your First Real Program (CLI) | ■ done | (new) | `wc` clone in 3 versions: minimal → flags → cmd/+internal/ with tests |

**Part I summary:** 7 chapters, 22 markdown files, 28 Go source files, ~50,000 words of prose, ~1,800 lines of runnable Go. All quality gates 1–4 pass (vet, build, run, test). Quality gates 5–10 (review pass, callouts, cross-refs, reading-time, read-out-loud) tracked separately.

**Part II summary:** 18 chapters (Ch 8–25), 63 runnable `main.go` examples, 18 README.md files with full 23-section structure, 18 exercises.md + checkpoint.md files. `go vet ./part2_core_language/...` and `go build ./part2_core_language/...` both clean. Covers: types, operators, control flow, functions, closures, defer/panic/recover, pointers, arrays, slices, maps, structs, methods, interfaces, embedding, generics, reflection.

### Part II — Core Language

| # | Chapter | Status | Source asset(s) | Notes |
| --- | --- | --- | --- | --- |
| 8 | Variables, Constants, Zero Value | ■ done | `01_fundamentals/02,06` | iota, zero values, 3 examples, vet/build/run green |
| 9 | Numbers, Strings, Booleans | ■ done | `01_fundamentals/03,04,05` | UTF-8, IEEE-754 traps, 3 examples |
| 10 | Conversion / Assertion / Switch | ■ done | `03_structs_methods_interfaces/08` | typed-nil gotcha, mini AST evaluator |
| 11 | Operators | ■ done | `01_fundamentals/07` | `&^`, bitwise patterns, FlagSet exercise |
| 12 | Control Flow | ■ done | `01_fundamentals/08` | for, switch, labels/goto; vet clean |
| 13 | Functions | ■ done | `02_functions/01,06,09` | multi-return, variadic, init, HOF, memoize |
| 14 | Closures | ■ done | `02_functions/02` | capture model, loop-var trap, 1.22 fix, patterns |
| 15 | defer/panic/recover | ■ done | `02_functions/03,04` + `01_fundamentals/10` | LIFO, arg eval, tx rollback, cost |
| 16 | Pointers | ■ done | `01_fundamentals/09` | escape analysis, *T optional, alignment |
| 17 | Arrays | ■ done | `05_collections/01` | value semantics, slice backing, matrix exercise |
| 18 | Slices | ■ done | `05_collections/02,03,04` | header internals, aliasing traps, 3-index, generic Stack |
| 19 | Maps | ■ done | `05_collections/05,06` | nil trap, iteration randomisation, Set, sync.Map |
| 20 | Structs | ■ done | `03_structs_methods_interfaces/01,02,07` | tags, JSON, embedding, layout, BankAccount |
| 21 | Methods | ■ done | `03_structs_methods_interfaces/03` | value/ptr receivers, method sets, nil receiver |
| 22 | Interfaces | ■ done | `03_structs_methods_interfaces/04,05,06,09` | itab, typed-nil, io pipeline, testing |
| 23 | Embedding & Composition | ■ done | `03_structs_methods_interfaces/02` | mixin, diamond resolution, middleware |
| 24 | Generics | ■ done | `09_generics/01–07` (full chapter) | Map/Filter/Reduce, ~T, OrderedSet |
| 25 | Reflection | ■ done | `10_advanced_patterns/04` | TypeOf/ValueOf, fillDefaults, DeepEqual, cost |

### Part III — Designing Software in Go

| # | Chapter | Status | Source asset(s) | Notes |
| --- | --- | --- | --- | --- |
| 26 | OOP in Go | ■ | (new) | Mental model shift |
| 27 | Interface-Driven Design | ■ | `03_structs_methods_interfaces/09` | Consumer-side |
| 28 | Dependency Injection | ■ | `03_structs_methods_interfaces/07` (functional options) | Wire vs manual |
| 29 | SOLID in Go | ■ | (new) | Each principle, Go-flavored |
| 30 | Clean / Hexagonal | ■ | (new) | Running-thread service starts |
| 31 | Patterns I — Creational | ■ | `10_advanced_patterns/01` | |
| 32 | Patterns II — Structural | ■ | `10_advanced_patterns/02` | |
| 33 | Patterns III — Behavioral | ■ | `10_advanced_patterns/03` | |
| 34 | Repository Pattern | ■ | (new) | Domain ≠ persistence |
| 35 | Service Layer | ■ | (new) | Thin handlers |
| 36 | Error Handling Philosophy | ■ | `04_error_handling/01,03,04,06` | |
| 37 | Custom Error Types | ■ | `04_error_handling/02,05,07,08` | errors.Join |
| 38 | Files / Streams / Buffered I/O | ■ | `08_standard_library/05,06` | io composition |
| 39 | Encoding | ■ | `08_standard_library/04` + (new) | JSON/XML/YAML/CSV |
| 40 | Configuration | ■ | (new) | 12-factor, secrets |

### Part IV — Concurrency & Systems

| # | Chapter | Status | Source asset(s) | Notes |
| --- | --- | --- | --- | --- |
| 41 | Concurrency Mental Model | ■ done | (new) | CSP, G/M/P, actor vs shared-memory, pipeline |
| 42 | Goroutines: Internals | ■ done | `06_concurrency/01` | G/M/P scheduler, stack growth, work stealing, leak detection |
| 43 | Channels: Internals | ■ done | `06_concurrency/02,03` | hchan, done channel, semaphore, ownership transfer, RingBuffer[T] |
| 44 | select / Timeouts / Cancel | ■ done | `06_concurrency/04` | priority select, nil disable, time.NewTimer safe pattern, retry |
| 45 | sync Primitives | ■ done | `06_concurrency/05,07` | Mutex/RWMutex, Cond, Once, Pool, Cache with TTL |
| 46 | sync/atomic | ■ done | `06_concurrency/06` + `08_standard_library/10` | typed API, CAS, atomic.Value hot-reload, metrics registry |
| 47 | context Package | ■ done | `06_concurrency/10` | WithCancel/Timeout/Deadline, propagation, WithValue, pipeline |
| 48 | Worker Pools | ■ done | `06_concurrency/08` | errgroup, scatter-gather, semaphore, RunBatch orchestrator |
| 49 | Pipelines, Fan-In/Out | ■ done | `06_concurrency/09` | stage signature, back-pressure, merge, ordered fan-out, CSV pipeline |
| 50 | Pub/Sub, Rate Limit, Throttle | ■ done | (new) | broker, token bucket, sliding window, throttle, leaky bucket, event bus |
| 51 | Race Detector | ■ done | (new) | -race flag, 5 patterns, fixes: atomic/mutex/channel/Once/capture |
| 52 | Deadlocks, Leaks | ■ done | (new) | lock ordering, self-deadlock, livelock, goroutine leaks, leak finder |
| 53 | Networking I — TCP/UDP | ■ done | (new) | echo server, graceful shutdown, semaphore, deadlines, UDP patterns |
| 54 | Networking II — HTTP/1.1 | ■ done | `08_standard_library/09` | ServeMux, middleware, JSON CRUD API, retry client, REST exercise |
| 55 | Networking III — TLS / H2 / H3 | ■ done | (new) | self-signed cert, HTTPS, mTLS, H2 multiplexing, ALPN |
| 56 | Production HTTP Server | ■ done | (new) | recovery, slog, rate limit, health/ready, graceful shutdown, metrics |

**Part IV summary:** 16 chapters (Ch 41–56), 48 runnable `main.go` examples (all `go vet` + `go run` clean), 16 README.md + exercises.md + checkpoint.md files. Covers: concurrency mental model, goroutine/channel internals, select patterns, sync primitives, atomic, context, worker pools, pipelines, pub/sub, race detector, deadlocks/leaks, TCP/UDP/HTTP/TLS/H2, production server. All `-race` clean examples in Ch41–52.

### Part V — Building Backends

| # | Chapter | Status | Source asset(s) | Notes |
| --- | --- | --- | --- | --- |
| 57 | REST API Design | ▢ | (new) | Versioning, idempotency |
| 58 | Routing options | ▢ | (new) | net/http 1.22, chi, gin, echo |
| 59 | Middleware | ▢ | (new) | Composition |
| 60 | Authentication | ▢ | (new) | Sessions/JWT |
| 61 | Authorization | ▢ | (new) | RBAC/ABAC/policy |
| 62 | Validation | ▢ | (new) | go-playground/validator |
| 63 | Structured Logging | ▢ | (new) | log/slog |
| 64 | API Error Handling | ▢ | (new) | RFC 7807 |
| 65 | database/sql | ▢ | (new) | Pool tuning |
| 66 | PostgreSQL with pgx | ▢ | (new) | LISTEN/NOTIFY |
| 67 | Transactions / ACID | ▢ | (new) | Retry loops |
| 68 | Migrations | ▢ | (new) | atlas/goose |
| 69 | ORM vs Builder vs Raw | ▢ | (new) | sqlc bias |
| 70 | Repository in Production | ▢ | (new) | Read replicas |
| 71 | Caching | ▢ | (new) | Stampede, singleflight |
| 72 | Redis | ▢ | (new) | go-redis, Redlock |
| 73 | Message Queues | ▢ | (new) | NATS/RabbitMQ/Kafka tour |
| 74 | Kafka | ▢ | (new) | Exactly-once truth |
| 75 | gRPC | ▢ | (new) | Protobuf, interceptors |
| 76 | GraphQL with gqlgen | ▢ | (new) | Dataloaders |
| 77 | WebSockets / SSE | ▢ | (new) | Hub pattern |
| 78 | Background Jobs | ▢ | (new) | asynq, river |
| 79 | Rate / Breaker / Retry | ▢ | (new) | gobreaker |
| 80 | Idempotency | ▢ | (new) | API edge |
| 81 | Event-Driven Architecture | ▢ | (new) | Outbox, saga |

### Part VI — Production Engineering

| # | Chapter | Status | Source asset(s) | Notes |
| --- | --- | --- | --- | --- |
| 82 | Testing Fundamentals | ▢ | `07_packages_modules/07` + `10_advanced_patterns/07` | Subtests, table-driven |
| 83 | Mocking | ▢ | (new) | gomock/mockery |
| 84 | testcontainers-go | ▢ | (new) | Real Postgres in CI |
| 85 | E2E Testing | ▢ | (new) | Snapshot, contract |
| 86 | Benchmarking | ▢ | `10_advanced_patterns/08` | benchstat |
| 87 | pprof | ▢ | (new) | Flame graphs |
| 88 | Performance Patterns | ▢ | `10_advanced_patterns/08` | sync.Pool |
| 89 | GC and Escape | ▢ | (new) | -gcflags=-m |
| 90 | Production Profiling | ▢ | (new) | Continuous profiling |
| 91 | Logging Strategy | ▢ | (new) | Sampling, PII |
| 92 | Prometheus | ▢ | (new) | Histograms, RED/USE |
| 93 | OpenTelemetry | ▢ | (new) | Spans, baggage |
| 94 | Dockerizing | ▢ | (new) | Multi-stage, distroless |
| 95 | Kubernetes | ▢ | (new) | Probes, HPA |
| 96 | CI/CD | ▢ | (new) | GitHub Actions, goreleaser |
| 97 | Deploying | ▢ | (new) | Blue/green, canary |
| 98 | Reliability | ▢ | (new) | SLO/SLI |
| 99 | Distributed Building Blocks | ▢ | (new) | Raft, leader election |
| 100 | Microservices vs Monolith | ▢ | (new) | Honest take |
| 101 | Security | ▢ | (new) | govulncheck, OWASP |
| 102 | Incidents / Post-Mortems | ▢ | (new) | Goroutine dumps |

### Part VII — Capstone Projects

| Project | Status | Notes |
| --- | --- | --- |
| A — URL Shortener | ▢ | Postgres + Redis, Docker, deploy |
| B — Auth Service | ▢ | OAuth, MFA, refresh tokens |
| C — E-commerce Backend | ▢ | Saga, outbox |
| D — Real-Time Chat | ▢ | WebSockets, Redis pub/sub |
| E — Notification Service | ▢ | Provider failover |
| F — Job Queue | ▢ | SQS-clone |
| G — File Upload | ▢ | Multipart, resumable, S3-compat |
| H — API Gateway | ▢ | Reverse proxy + edge features |
| I — Distributed Scheduler | ▢ | Leader election, no-miss cron |
| J — Microservices Platform | ▢ | 5 services + platform layer |

---

## Sequencing rules

1. **Part I before Part II.** Some Part I chapters reference Part II
   concepts; that's fine, but a reader who only completes Part I should
   already have a working mental model.
2. **Concurrency before Backends.** Part V leans heavily on Part IV.
3. **Capstones last.** Each capstone is a flag that says "the reader has
   completed everything before."
4. **Running-thread service.** Starting in Chapter 30 (Clean Architecture),
   the book maintains *one* example service — a notes/snippets backend —
   that grows chapter by chapter through Part V. By Chapter 81 it is a
   complete event-driven microservice. This is separate from the Capstones,
   which are standalone.

---

## Per-chapter deliverables checklist

For each chapter the deliverables are:

- [ ] `README.md` with all 23 sections, no placeholders.
- [ ] At least one runnable `.go` file. Most chapters have 2–6.
- [ ] `exercises.md` + `exercises/` with starter files.
- [ ] `solutions/` with reference solutions (gitignored from print).
- [ ] `checkpoint.md` — 5–10 self-test questions with answers.
- [ ] Cross-references to prior chapters where relevant.
- [ ] `CHANGELOG.md` entry.
- [ ] (Parts VI+) test files, `go test ./...` passes.
- [ ] (Parts V+) `docker-compose.yml` if infra needed.

---

## How to add a chapter

1. Create the folder: `book/partN_<name>/chapterNN_<topic>/`.
2. Copy `CHAPTER_TEMPLATE.md` to that folder's `README.md` and rewrite.
3. Write the runnable `.go` files. They drive the prose, not the other way
   around.
4. Write exercises and a mini-project (if applicable).
5. Run all 10 quality gates.
6. Update this manifest's status table.
7. Commit. Push. PR if collaborating.

---

## Risks and known unknowns

* **Go version drift.** Major Go releases (1.23, 1.24) may change material.
  Each chapter declares its target version; revisits go in `CHANGELOG.md`.
* **Third-party churn.** Chapters relying on `go-redis`, `gqlgen`, etc.
  must pin versions and re-test on dependency upgrades.
* **Cloud provider drift.** Capstones use generic VPS deploys + a separate
  K8s appendix to avoid coupling to one cloud's UI.
* **Scale of work.** Estimated 300,000–500,000 words + 50,000+ lines of Go.
  Realistic timeline, working evenings: 6–12 months. The phased plan above
  is the only way it converges.
