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
| 26 | OOP in Go | ■ done | chapter26_oop_in_go | Mental model shift from class-based OOP; struct embedding vs inheritance |
| 27 | Interface-Driven Design | ■ done | chapter27_interface_driven_design | Consumer-side interfaces, implicit satisfaction, io.Reader composition |
| 28 | Dependency Injection | ■ done | chapter28_dependency_injection | Functional options, Wire vs manual DI, constructor injection |
| 29 | SOLID in Go | ■ done | chapter29_solid_in_go | All 5 principles Go-flavored; interface segregation, Liskov, DIP |
| 30 | Clean / Hexagonal | ■ done | chapter30_clean_architecture | Ports & adapters, running-thread service starts here |
| 31 | Patterns I — Creational | ■ done | chapter31_creational_patterns | Factory, Builder, Singleton, Option; from `10_advanced_patterns/01` |
| 32 | Patterns II — Structural | ■ done | chapter32_structural_patterns | Adapter, Decorator, Proxy, Facade; from `10_advanced_patterns/02` |
| 33 | Patterns III — Behavioral | ■ done | chapter33_behavioral_patterns | Strategy, Observer, Command, Iterator; from `10_advanced_patterns/03` |
| 34 | Repository Pattern | ■ done | chapter34_repository_pattern | Domain ≠ persistence, in-memory + SQL repos, interface contract |
| 35 | Service Layer | ■ done | chapter35_service_layer | Thin handlers, orchestration layer, transaction boundary |
| 36 | Error Handling Philosophy | ■ done | chapter36_error_handling_philosophy | Sentinel, opaque, structured errors; from `04_error_handling/01,03,04,06` |
| 37 | Custom Error Types | ■ done | chapter37_custom_error_types | errors.Join, %w wrapping, As/Is, from `04_error_handling/02,05,07,08` |
| 38 | Files / Streams / Buffered I/O | ■ done | chapter38_files_streams_io | io.Reader/Writer composition, bufio, from `08_standard_library/05,06` |
| 39 | Encoding | ■ done | chapter39_encoding | JSON/XML/CSV/gob, streaming decoder, from `08_standard_library/04` |
| 40 | Configuration | ■ done | chapter40_configuration | 12-factor, env/file/flag layering, secrets, viper pattern |

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
| 57 | REST API Design | ■ done | chapter57_rest_api_design | Versioning, idempotency keys, API evolution |
| 58 | Routing options | ■ done | chapter58_routing_options | net/http 1.22 patterns, chi, gin, echo comparison |
| 59 | Middleware | ■ done | chapter59_middleware | Composition chain, logging, auth, recovery |
| 60 | Authentication | ■ done | chapter60_authentication | Sessions, JWT, refresh tokens |
| 61 | Authorization | ■ done | chapter61_authorization | RBAC, ABAC, policy engine |
| 62 | Validation | ■ done | chapter62_validation | go-playground/validator, custom rules, error messages |
| 63 | Structured Logging | ■ done | chapter63_structured_logging | log/slog, sampling, PII redaction |
| 64 | API Error Handling | ■ done | chapter64_api_error_handling | RFC 7807 problem details, error taxonomy |
| 65 | database/sql | ■ done | chapter65_database_sql | Pool tuning, prepared statements, scanning |
| 66 | PostgreSQL with pgx | ■ done | chapter66_postgresql_pgx | LISTEN/NOTIFY, COPY, pgx v5 |
| 67 | Transactions / ACID | ■ done | chapter67_transactions_acid | Retry loops, savepoints, isolation levels |
| 68 | Migrations | ■ done | chapter68_migrations | atlas/goose, up/down, CI integration |
| 69 | ORM vs Builder vs Raw | ■ done | chapter69_orm_patterns | sqlc, squirrel, GORM trade-offs |
| 70 | Caching | ■ done | chapter70_caching | Stampede, singleflight, TTL, layered cache |
| 71 | Redis | ■ done | chapter71_redis | go-redis, Redlock, pub/sub, sorted sets |
| 72 | Message Queues | ■ done | chapter72_message_queues | Priority queue, pub/sub, middleware, event sourcing |
| 73 | Kafka | ■ done | chapter73_kafka | Topics, partitions, consumer groups, compacted topics |
| 74 | GraphQL | ■ done | chapter74_graphql | N+1, dataloaders, pagination, subscriptions |
| 75 | WebSockets / SSE | ■ done | chapter75_websockets | Hub pattern, rooms, reconnection, Redis fan-out |
| 76 | gRPC | ■ done | chapter76_grpc | Status codes, interceptors, streaming, hedged requests |
| 77 | Background Jobs | ■ done | chapter77_background_jobs | Priority queue, scheduler, distributed lock, DLQ |
| 78 | Rate / Breaker / Retry | ■ done | chapter78_rate_limiting | Token bucket, circuit breaker, resilience policy |
| 79 | Idempotency | ■ done | chapter79_idempotency | IdempStore, inbox, outbox, saga, delivery semantics |
| 80 | Event-Driven Architecture | ■ done | chapter80_event_driven | Outbox relay, event sourcing, CQRS, choreography saga |
| 81 | (see Ch80) | — | — | Merged into chapter80_event_driven |

### Part VI — Production Engineering

| # | Chapter | Status | Source asset(s) | Notes |
| --- | --- | --- | --- | --- |
| 82 | Testing Fundamentals | ■ done | chapter82_testing_fundamentals | Subtests, table-driven tests, parallel, golden files |
| 83 | Mocking | ■ done | chapter83_mocking | Interface mocks, gomock patterns, stub vs fake |
| 84 | Integration Testing | ■ done | chapter84_integration_testing | testcontainers-go, real Postgres in CI |
| 85 | Benchmarking | ■ done | chapter85_benchmarking | benchstat, pprof-guided optimisation, allocation benchmarks |
| 86 | pprof | ■ done | chapter86_pprof | Flame graphs, CPU/heap/goroutine profiles, net/http/pprof |
| 87 | Performance Patterns | ■ done | chapter87_performance_patterns | sync.Pool, string builder, zero-alloc patterns |
| 88 | GC and Escape Analysis | ■ done | chapter88_gc_escape | -gcflags=-m, escape heuristics, GC tuning |
| 89 | Logging Strategy | ■ done | chapter89_logging_strategy | log/slog structured logging, sampling, PII redaction |
| 90 | Prometheus | ■ done | chapter90_prometheus | Histograms, counters, RED/USE method, alerting rules |
| 91 | OpenTelemetry | ■ done | chapter91_opentelemetry | Traces, spans, baggage, W3C traceparent, sampling |
| 92 | Dockerizing | ■ done | chapter92_dockerizing | Multi-stage builds, distroless, health checks, layer cache |
| 93 | Kubernetes | ■ done | chapter93_kubernetes | Deployments, probes, HPA, PDB, rolling update |
| 94 | CI/CD | ■ done | chapter94_cicd | GitHub Actions workflows, goreleaser, build matrix |
| 95 | Reliability | ■ done | chapter95_reliability | SLO/SLI, error budget, burn rate alerts, circuit breaker |
| 96 | Distributed Building Blocks | ■ done | chapter96_distributed_building_blocks | Leader election, distributed lock, fencing tokens, Raft basics |
| 97 | Security | ■ done | chapter97_security | OWASP top 10 patterns, JWT HMAC, rate limiting, secret scanning |
| 98 | Incidents / Post-Mortems | ■ done | chapter98_incidents | Goroutine dumps, panic recovery, postmortem structure |
| 99 | Production Profiling | ■ done | chapter99_production_profiling | Continuous profiling, heap growth analysis, profile diff/regression |
| 100 | Deploying | ■ done | chapter100_deploying | Blue/green, canary with metric gates, strategy decision engine |
| 101 | Microservices vs Monolith | ■ done | chapter101_microservices_vs_monolith | Strangler fig, seam detection, data ownership, migration planner |
| 102 | (see Ch98–101) | — | — | Covered across incidents, profiling, deploying, microservices chapters |

### Part VII — Capstone Projects

| Project | Status | Source asset(s) | Notes |
| --- | --- | --- | --- |
| A — URL Shortener | ■ done | capstone_a_url_shortener | Base62, Redis cache, rate limiter, click tracker, graceful shutdown |
| B — Auth Service | ■ done | capstone_b_auth_service | bcrypt, JWT, refresh token rotation, TOTP MFA, RBAC |
| C — E-commerce Backend | ■ done | capstone_c_ecommerce_backend | Saga pattern, inventory reservation, payment compensation |
| D — Real-Time Chat | ■ done | capstone_d_realtime_chat | Hub/room pattern, presence tracker, message history, reconnect backoff |
| E — Notification Service | ■ done | capstone_e_notification_service | Multi-channel, provider failover, retry+backoff, dead-letter queue |
| F — Job Queue | ■ done | capstone_f_job_queue | Visibility timeout, priority lanes, DLQ, worker pool |
| G — File Upload | ■ done | capstone_g_file_upload | Multipart assembly, resumable upload, virus scan hook, S3-compat interface |
| H — API Gateway | ■ done | capstone_h_api_gateway | Route table, round-robin LB, circuit breaker, rate limiter, metrics |
| I — Distributed Scheduler | ■ done | capstone_i_distributed_scheduler | Cron parser, leader election, no-miss catch-up, distributed lock |
| J — Microservices Platform | ■ done | capstone_j_microservices_platform | 5 services, message bus, service registry, distributed trace context |

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
