# Deep Dive into Go: Building Production-Ready Systems
## Master Table of Contents — From Level 0 to Level 1000

> A self-taught path from "I have never written code" to "I architect production
> Go systems at scale." Every chapter is a runnable folder. Every chapter
> follows the same 23-section structure (see `CHAPTER_TEMPLATE.md`).
> Every code file is heavily commented, opinionated, and production-grade.

---

## How to read this book

There are three legitimate paths through this material:

1. **Linear path (recommended for first-time learners).** Read every chapter in
   order. Run every program. Do every exercise. Build every mini-project. This
   is the path that takes you from zero to senior. Expect 4–6 months of
   evening study, or 6–10 weeks full-time.
2. **Bridge path (for engineers from another stack).** Skim Part I, read the
   "Coming from X" callouts in each chapter (Java/Python/JS/C++/Rust), then go
   deep on Parts III–VI. Expect 4–6 weeks.
3. **Interview-prep path.** Read the *Concept Introduction*, *Common Mistakes*,
   *Senior Engineer Best Practices*, *Interview Questions*, and *Chapter
   Summary* sections of every chapter. Skip exercises. Expect 2–3 weeks.

Every chapter ends with a **Revision Checkpoint** — a short list of self-test
questions. If you cannot answer them without looking, re-read the chapter.

---

## Book structure at a glance

The book is organized into **seven parts**, **thirteen sections**, and roughly
**ninety chapters**. The seven parts are pedagogical groupings that map onto
the thirteen sections from the curriculum spec.

```
PART I    — Foundations                             (Section 1)
PART II   — The Core Language                       (Section 2)
PART III  — Designing Software in Go                (Sections 3, 5)
PART IV   — Concurrency & Systems                   (Sections 4, 9)
PART V    — Building Backends                       (Sections 6, 7, 8)
PART VI   — Production Engineering                  (Sections 10, 11, 12)
PART VII  — Capstone Projects                       (Section 13)
```

---

# PART I — FOUNDATIONS
*From "what is a compiler" to "I just shipped my first Go binary."*

### Chapter 1 — Why Go Exists
The 2007 Google problem. Slow C++ builds, awkward Java, and the scale wall.
Pike, Thompson, Griesemer's design constraints. Why Go is small on purpose.
Where Go does and does not belong in modern stacks.

### Chapter 2 — A Map of the Go Ecosystem
The toolchain (`go build`, `go run`, `go test`, `go vet`, `go mod`, `go work`).
The standard library philosophy. Where third-party libraries live. The release
cadence and backward compatibility promise.

### Chapter 3 — Installing Go and Setting Up Your Environment
Official installer vs. `gvm` vs. distro packages. `GOROOT`, `GOPATH`,
`GOBIN`, `GOMODCACHE` demystified. VS Code + `gopls`. GoLand. Vim/Neovim.
Verifying the install. Hello World, but properly.

### Chapter 4 — The Go Workspace and Project Structure
Modules vs. the legacy GOPATH world. Module path semantics. Multi-module
workspaces with `go.work`. The "standard" Go project layout (and why the
official position is *there is no standard layout*).

### Chapter 5 — How `go run`, `go build`, and `go install` Actually Work
The compile-link-execute pipeline. Build cache. Cross-compilation
(`GOOS`/`GOARCH`). Static linking and why Go binaries are big. CGO and the
moment your binary stops being statically linked.

### Chapter 6 — Coming From Another Language: Mental Model Transfer
Side-by-side: Java → Go, Python → Go, JavaScript → Go, C++ → Go, Rust → Go.
What translates cleanly, what does not, and what habits to unlearn.

### Chapter 7 — Your First Real Program: a CLI Word Counter
End-to-end: read stdin, parse flags, count words/lines/bytes, write to stdout,
exit with the right code. A hundred-line program that touches packages,
imports, errors, slices, strings, and the standard library — your first taste
of "Go feels different."

---

# PART II — THE CORE LANGUAGE
*Every keyword. Every operator. Every type. No hand-waving.*

### Chapter 8 — Variables, Constants, and the Zero Value
Declaration forms (`var`, `:=`, `const`). The zero value contract and why it
removes a class of bugs. `iota` deeply explained.

### Chapter 9 — The Type System: Numbers, Strings, Booleans
Sized vs. unsized integers. `int` is platform-dependent — and why that matters.
Floats and IEEE-754 traps. Strings as immutable byte sequences. Runes vs.
bytes vs. characters: the UTF-8 rabbit hole.

### Chapter 10 — Type Conversion, Type Assertion, Type Switch
The three operations beginners conflate. When the compiler converts for you,
when it refuses, and the runtime cost of each form.

### Chapter 11 — Operators, Precedence, and Bitwise Tricks
The full operator table. `&^` (bit clear) — Go's hidden gem. Practical bitwise
patterns: flag sets, alignment, fast modulo for powers of two.

### Chapter 12 — Control Flow: `if`, `for`, `switch`, `goto`
Go has one looping keyword. `switch` is more powerful than C's. `goto` is in
the language and where it is legitimate. Labelled `break`/`continue`.

### Chapter 13 — Functions: First-Class Citizens
Multiple return values. Named returns and naked returns (and why to avoid
them). Variadic functions. Functions as values, parameters, and return types.

### Chapter 14 — Closures and the Capture Model
What a closure captures (variables, not values). The classic loop-variable
bug — and why Go 1.22 fixed it. Closures as state machines.

### Chapter 15 — `defer`, `panic`, `recover`
Defer execution order. The cost of `defer` (and how Go 1.14+ made it nearly
free). When `panic` is correct. The `recover` idiom — and the contract it
must obey.

### Chapter 16 — Pointers and Memory Addressing
Pointers without pointer arithmetic. When to take an address. Pointer
receivers vs. value receivers. The escape analysis preview (full treatment in
Part VI).

### Chapter 17 — Arrays, the Real Underlying Type
Why arrays are second-class compared to slices. Fixed-size buffers, hashing
keys, fast lookup tables. Array equality and copy semantics.

### Chapter 18 — Slices: The Most Important Type in Go
The slice header (`ptr, len, cap`). Growth strategy. The aliasing trap.
`append`, `copy`, `make`. The classic "two slices share backing storage" bug.

### Chapter 19 — Maps: Hash Tables Built In
Go's map internals (buckets, hash seeds, growth). Iteration order is
randomized — by design. Concurrent map access and why it crashes hard.

### Chapter 20 — Structs and Composite Literals
Field tags. Struct embedding (composition, not inheritance). Empty struct
`struct{}` and its uses. Memory layout and field ordering for size.

### Chapter 21 — Methods: Functions With Receivers
Value receivers vs. pointer receivers. The receiver naming convention. Method
sets, addressability, and the rules that catch every newcomer.

### Chapter 22 — Interfaces: Go's Killer Feature
Implicit satisfaction. The empty interface (`any`). Interface internals (itab,
type word, data word). The "accept interfaces, return structs" rule.

### Chapter 23 — Embedding and Composition
Field promotion, method promotion. Diamond conflicts and how Go resolves
them. Why composition replaces inheritance in idiomatic Go.

### Chapter 24 — Generics: Type Parameters and Constraints
The motivation. `comparable`, `~`, type sets. Writing generic functions and
generic types. When generics make code worse, not better.

### Chapter 25 — Reflection: Programming the Type System
`reflect.Type`, `reflect.Value`, `reflect.Kind`. The performance cost.
Building a generic JSON encoder from scratch. When NOT to use reflect.

---

# PART III — DESIGNING SOFTWARE IN GO
*OOP without classes. Architecture without ceremony.*

### Chapter 26 — OOP in Go: A Different Mental Model
Encapsulation via package boundaries. Polymorphism via interfaces.
Composition instead of inheritance. The "no class" liberation.

### Chapter 27 — Interface-Driven Design
Designing from the consumer side. Small interfaces (`io.Reader`, `io.Writer`).
The "one method interface" pattern. Interface segregation in practice.

### Chapter 28 — Dependency Injection the Go Way
Constructor injection. Functional options. Wire-style code generation vs.
manual wiring. Why DI frameworks rarely earn their cost in Go.

### Chapter 29 — SOLID Principles, Translated to Go
Each principle, with idiomatic Go examples and counter-examples. Where
classical SOLID needs adjustment for Go's design.

### Chapter 30 — Clean Architecture and Hexagonal Architecture
Ports and adapters. The dependency rule. Domain/application/infra layers in
a real Go service. When the indirection helps and when it hurts.

### Chapter 31 — Design Patterns I — Creational
Factory, Builder, Singleton, Prototype, Object Pool. Idiomatic Go forms.
Patterns that disappear in Go (and why).

### Chapter 32 — Design Patterns II — Structural
Adapter, Decorator, Facade, Proxy, Composite. `http.Handler` as a built-in
Decorator example. Wrappers everywhere.

### Chapter 33 — Design Patterns III — Behavioral
Strategy, Observer, Command, State, Iterator, Template Method, Visitor.
How channels and closures replace half of these.

### Chapter 34 — The Repository Pattern, Done Right
Domain types vs. persistence types. Mapping between them. Why "use the ORM
type as your domain type" is a decade-1 mistake.

### Chapter 35 — Service Layer and Use Cases
Thin handlers, fat services. Transaction boundaries. Cross-cutting concerns
(logging, metrics, tracing) with middleware/decorators.

### Chapter 36 — Error Handling Philosophy
Errors are values. The `error` interface. Sentinel errors. `errors.Is`,
`errors.As`. Wrapping with `fmt.Errorf("%w", err)`. The "don't just check,
handle" rule.

### Chapter 37 — Custom Error Types and Error Trees
Typed errors with metadata. HTTP-mappable errors. Joining errors with
`errors.Join`. The new `errors` package primitives end-to-end.

### Chapter 38 — Files, Streams, and Buffered I/O
`os.File`, `bufio`, `io.Reader`, `io.Writer`. The composition of streams.
The `io.Pipe` trick. Tee readers and limit readers in production.

### Chapter 39 — Encoding: JSON, XML, YAML, CSV, gob
Struct tags. Custom (Un)Marshalers. Streaming decoders for huge payloads.
Choosing a format for an API or a config file.

### Chapter 40 — Configuration Management
Flags, env vars, files, secrets. The 12-factor view. Hot reload. Layered
config with override precedence. Why configuration is harder than it looks.

---

# PART IV — CONCURRENCY & SYSTEMS
*Goroutines, channels, sync — and the production patterns built on them.*

### Chapter 41 — The Concurrency Mental Model
Concurrency vs. parallelism. The `GOMAXPROCS` story. CSP (Communicating
Sequential Processes) and why Go picked it. Tony Hoare's paper, in plain
English.

### Chapter 42 — Goroutines: Internals
Stacks (1 KB → 1 GB). The scheduler (G/M/P model). Preemption (cooperative
pre-1.14, asynchronous post-1.14). Why a goroutine is not a thread.

### Chapter 43 — Channels: Internals and Patterns
The `hchan` struct. Send/receive semantics. Buffered vs. unbuffered. Closing
channels (and the rules that prevent panics). The "channel direction" type
system trick.

### Chapter 44 — `select`, Timeouts, and Cancellation
Multi-way communication. The `default` branch. Random selection. Building
timeouts, debouncers, and rate-limiters with `select`.

### Chapter 45 — `sync` Primitives
`Mutex`, `RWMutex`, `WaitGroup`, `Once`, `Cond`, `Pool`. When each is the
right tool. The mutex-vs-channel decision tree.

### Chapter 46 — `sync/atomic` and Lock-Free Programming
Atomic loads/stores/CAS. The Go 1.19 typed atomics. When lock-free wins and
when it loses. Memory ordering — the Go memory model in practice.

### Chapter 47 — The `context` Package
Deadlines, cancellation, request-scoped values. `context.Background` vs.
`context.TODO`. The "first parameter, named ctx" convention. Misuse patterns.

### Chapter 48 — Concurrency Patterns I — Worker Pools
Bounded parallelism. Backpressure. Graceful shutdown. The semaphore pattern.
`golang.org/x/sync/errgroup`.

### Chapter 49 — Concurrency Patterns II — Pipelines, Fan-In/Fan-Out
Stage composition. Cancelled-pipeline cleanup. Why explicit cancellation is
not optional.

### Chapter 50 — Concurrency Patterns III — Pub/Sub, Rate Limiting, Throttling
In-process pub/sub. Token bucket. Leaky bucket. `golang.org/x/time/rate`.
Adaptive concurrency.

### Chapter 51 — Race Conditions and the Race Detector
What a data race is. Why some races are "benign" — and why that's a lie.
`go test -race`. Reading race-detector output. The cost in production.

### Chapter 52 — Deadlocks, Livelocks, Goroutine Leaks
The four conditions. Lock ordering. Detection in practice. The `pprof`
goroutine profile. The goroutine leak audit you should run before every
release.

### Chapter 53 — Networking I — TCP and UDP from Scratch
The `net` package. `net.Listen`, `net.Dial`. Reading/writing on a connection.
A toy chat server. A toy DNS resolver.

### Chapter 54 — Networking II — HTTP/1.1 Internals
The HTTP request/response cycle. The `http.Request`/`http.Response` types.
`Transport`, `RoundTripper`, connection pooling. Keep-alive and HTTP/2
upgrades.

### Chapter 55 — Networking III — TLS, HTTP/2, HTTP/3
Setting up TLS with self-signed and ACME. ALPN and protocol negotiation. When
HTTP/2 helps (multiplexing) and hurts (head-of-line). HTTP/3 status today.

### Chapter 56 — Building a Production HTTP Server From Scratch
Graceful shutdown. Server timeouts (each one matters). Connection limits.
The middleware chain. Wiring metrics, tracing, logging. Why you almost never
need a framework.

---

# PART V — BUILDING BACKENDS
*From "I can write a handler" to "I run a service in production."*

### Chapter 57 — Designing REST APIs in Go
Resource modeling. Versioning strategies (URL, header, content). Idempotency.
Pagination. Filtering. Sparse fieldsets. Status codes that mean what they say.

### Chapter 58 — Routing: `net/http` ServeMux, chi, gorilla, gin, echo
The 1.22 `net/http` ServeMux upgrade. When the standard library is enough.
When to reach for a router. A defensible comparison table.

### Chapter 59 — Middleware Architecture
Function composition. Per-route vs. global middleware. Order matters.
Building auth, logging, metrics, tracing, rate limiting middleware.

### Chapter 60 — Authentication: Sessions, Cookies, JWT
Cookie security flags. CSRF protection. JWT — when it's the right call and
when it isn't. Refresh tokens. Logout that actually logs out.

### Chapter 61 — Authorization: RBAC, ABAC, Policy Engines
Roles vs. permissions vs. policies. Casbin, OPA. Where policy lives. The
audit-log requirement most teams ignore.

### Chapter 62 — Validation: Input at the Edge
`go-playground/validator`. Custom validators. Validation that produces
client-friendly errors. Why your domain types should not depend on a
validator library.

### Chapter 63 — Structured Logging with `log/slog`
Why log levels are not enough. Structured logs. Context propagation. Log
sampling. Log injection prevention.

### Chapter 64 — Error Handling at the API Boundary
Domain errors → HTTP responses. The "problem details" RFC 7807. Hiding
internals. Correlation IDs and how to make on-call easier.

### Chapter 65 — Persistence I — `database/sql` Done Properly
The standard library API. Connection pooling tuning. Prepared statements.
`sql.Null*` types. The `Scanner`/`Valuer` interfaces.

### Chapter 66 — Persistence II — PostgreSQL with `pgx`
The native protocol driver. Why `pgx` beats `lib/pq`. COPY for bulk loads.
LISTEN/NOTIFY for cheap pub/sub.

### Chapter 67 — Persistence III — Transactions, Isolation, ACID
The four anomalies. The four isolation levels. `SERIALIZABLE` retry loops.
The "transactions over multiple HTTP calls" anti-pattern.

### Chapter 68 — Persistence IV — Migrations
`golang-migrate`, `goose`, `atlas`. Forward-only vs. reversible. Online
schema change. The pre-commit hook that catches schema drift.

### Chapter 69 — Persistence V — ORM vs. SQL Builders vs. Raw
GORM, Ent, `sqlx`, `sqlc`. The honest tradeoffs. Why `sqlc` is the
mainstream-best-practice in 2026.

### Chapter 70 — Persistence VI — The Repository Pattern, Production Form
Domain repos vs. SQL adapters. Unit-of-work. Read replicas. Sharding hints.
The query that should always have an index.

### Chapter 71 — Caching: Local, Distributed, and the Cache Stampede
`sync.Map`, `groupcache`, Redis. Cache-aside, write-through, write-behind.
TTL strategy. Singleflight. The thundering-herd problem.

### Chapter 72 — Redis with Go
`go-redis`. Strings, hashes, sorted sets, streams. Lua scripting.
Distributed locks (and why naive ones are wrong). Redlock — the debate.

### Chapter 73 — Message Queues: NATS, RabbitMQ, Kafka
Publishing/consuming. Delivery guarantees (at-most/at-least/exactly-once).
Idempotent consumers. Dead-letter queues. Backpressure.

### Chapter 74 — Kafka with Go
`segmentio/kafka-go` vs. `confluent-kafka-go`. Partitions, consumer groups,
offsets. Exactly-once semantics — the truth. Schema registries.

### Chapter 75 — gRPC: Schema-First Services
Protobuf. `protoc-gen-go-grpc`. Unary, server-streaming, client-streaming,
bidirectional. Interceptors. Error model. Grpc-Gateway for REST.

### Chapter 76 — GraphQL with `gqlgen`
Schema-first vs. code-first. N+1 and dataloaders. Auth in resolvers.
Federation. When GraphQL is the right call (and when REST/gRPC wins).

### Chapter 77 — WebSockets, SSE, and Long-Lived Connections
The two production options. Heartbeats. Reconnection. Scaling sticky sessions.
The hub pattern in Go.

### Chapter 78 — Background Jobs and Schedulers
`asynq`, `river`, `gocraft/work`. Cron jobs. Distributed schedulers.
Idempotency. Retry policy. Job poisoning.

### Chapter 79 — Rate Limiting, Circuit Breakers, Retries
Token bucket revisited at the API edge. Exponential backoff with jitter.
The circuit breaker state machine. `sony/gobreaker`. Hedged requests.

### Chapter 80 — Idempotency at the API Boundary
Idempotency keys. The PUT-vs-POST debate. Storing idempotency results.
Race conditions on first request.

### Chapter 81 — Event-Driven Architecture in Go
Domain events. Outbox pattern. CDC with Debezium. Saga orchestration vs.
choreography. The "eventual consistency is hard" reality check.

---

# PART VI — PRODUCTION ENGINEERING
*Testing, performance, observability, deploys, scaling.*

### Chapter 82 — Testing Fundamentals: `testing` Package
`*testing.T`, `*testing.B`. Subtests. Table-driven tests. `t.Helper()`.
Test parallelism. Golden files.

### Chapter 83 — Mocking and Test Doubles
Hand-rolled fakes. `gomock`, `mockery`. The "don't mock what you don't own"
rule. Interface seams. Fakes vs. mocks vs. stubs vs. spies.

### Chapter 84 — Integration Testing with `testcontainers-go`
Real Postgres in CI. Real Redis. Real Kafka. The "no mocks at the storage
boundary" rule and why it caught real bugs.

### Chapter 85 — End-to-End Testing
Black-box HTTP tests. Test data lifecycle. Flake budget. Snapshot testing.
Contract testing with Pact.

### Chapter 86 — Benchmarking with `go test -bench`
Microbenchmarks. `b.ReportAllocs()`, `b.ResetTimer()`. `benchstat`. The
"benchmark lies" trap. Comparing CPU and allocs across versions.

### Chapter 87 — Profiling with `pprof`
CPU, heap, goroutine, block, mutex profiles. Flame graphs. The four-step
performance investigation. `go tool trace` for the scheduler view.

### Chapter 88 — Performance Tuning Patterns
Allocation reduction. `sync.Pool`. Pre-allocation. String building.
JSON encoding alternatives. The "measure twice, optimize once" discipline.

### Chapter 89 — The Garbage Collector and Escape Analysis
The Go GC: tri-color, concurrent, non-generational. Why it's tuned for
latency, not throughput. Escape analysis with `-gcflags=-m`. Heap vs. stack
in practice.

### Chapter 90 — Memory and CPU Profiling Production Services
Continuous profiling (Pyroscope, Grafana Profiles). Sampling overhead. Safe
profiling endpoints. Reading profiles you didn't generate.

### Chapter 91 — Observability I — Logging Strategy
Structured logs at scale. Sampling. PII redaction. Centralized aggregation
(Loki, Elastic, Datadog).

### Chapter 92 — Observability II — Metrics with Prometheus
The four metric types. Cardinality discipline. Histograms vs. summaries.
Native histograms. RED and USE methods.

### Chapter 93 — Observability III — Tracing with OpenTelemetry
Spans, traces, baggage. Context propagation across HTTP/gRPC/Kafka.
Sampling decisions. Cost vs. signal.

### Chapter 94 — Containerizing Go: Dockerfiles That Don't Suck
Multi-stage builds. Distroless and `scratch`. The 8 MB image. Caching layers.
Reproducible builds. Security scanning.

### Chapter 95 — Kubernetes for Go Services
Deployment, Service, Ingress. Probes (liveness, readiness, startup). HPA.
ConfigMap vs. Secret. The "12-factor service in K8s" checklist.

### Chapter 96 — CI/CD with GitHub Actions
Build matrix. Caching `go mod` and the build cache. `golangci-lint` in CI.
Release automation with `goreleaser`. Signing with `cosign`.

### Chapter 97 — Deploying Production Services
Blue/green, canary, rolling. Database migration coordination. Feature flags.
Rollback strategy. The pre-deploy checklist.

### Chapter 98 — Reliability Engineering
SLOs, SLIs, error budgets. Graceful degradation. Bulkheads. Health endpoints.
Chaos engineering, in moderation.

### Chapter 99 — Distributed Systems Building Blocks
Consensus (Raft via `hashicorp/raft`). Leader election. Distributed locks.
Consistent hashing. Vector clocks — when you actually need them.

### Chapter 100 — Microservices vs. Monoliths: An Honest Take
The "modular monolith" middle ground. When to split. The cost of network
boundaries. The data-ownership rule.

### Chapter 101 — Security Engineering for Go Services
The OWASP top 10, Go-flavored. Dependency vulnerability scanning (`govulncheck`).
Secrets management. Secure defaults. The audit-log non-negotiable.

### Chapter 102 — Production Incidents and Post-Mortems
Reading goroutine dumps. The "service is slow" investigation playbook. The
five-whys. Writing a blameless post-mortem that actually changes the system.

---

# PART VII — CAPSTONE PROJECTS
*Build real systems end-to-end. Each project is a multi-chapter walkthrough.*

### Project A — URL Shortener
Single binary. Postgres. Redis cache. Rate limiting. Metrics. Docker.
Deployed to a VPS with one command.

### Project B — Authentication Service
Email/password, OAuth, MFA, refresh tokens, password reset, session
revocation. Audit log. Rate limited. Production-grade JWT lifecycle.

### Project C — E-commerce Backend
Product catalog, cart, checkout, payment integration (mock), inventory,
orders. Event-driven. Saga for order fulfillment. Outbox pattern.

### Project D — Real-Time Chat System
WebSockets. Redis pub/sub. Room sharding. Presence. Read receipts. Message
history with cursor pagination.

### Project E — Notification Service
Email, SMS, push. Templates. Provider failover. Retries with backoff.
Idempotency. Rate limits per user.

### Project F — Job Queue System
Distributed worker pool. Cron-style scheduling. Priority queues. Visibility
timeouts. Dead-letter handling. A miniature SQS clone.

### Project G — File Upload Service
Multipart. Resumable uploads. S3-compatible storage. Virus scanning hooks.
Pre-signed URLs. Per-tenant quotas.

### Project H — API Gateway
Reverse proxy. Auth, rate limiting, request transformation, response caching,
circuit breaking. A miniature Kong/Tyk.

### Project I — Distributed Task Scheduler
Leader election. Distributed cron with no missed/duplicate fires. Fan-out
to workers. Web UI for status.

### Project J — Production Microservices Platform
Five services with a shared platform layer (auth, observability, deploy).
gRPC between services, REST at the edge. The capstone of capstones.

---

## Appendices

* **A.** The Go Specification, summarized in 30 pages.
* **B.** The Go Memory Model — annotated.
* **C.** Go release-by-release: every version's notable changes (1.0 → 1.24).
* **D.** Standard library tour — every package, one paragraph each.
* **E.** Recommended further reading: papers, talks, books.
* **F.** Interview question bank — 300 questions with answers, by chapter.
* **G.** "What I'd build next" — open-ended project ideas to extend the book.

---

## Versioning and corrections

This book is built against Go 1.22+ and aims to remain forward-compatible with
each Go release. Each chapter declares the Go version it was written against
in its README header. Corrections and updates are tracked in `CHANGELOG.md`.
