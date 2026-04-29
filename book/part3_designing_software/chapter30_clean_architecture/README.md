# Chapter 30 — Clean / Hexagonal Architecture

> **Part III · Designing Software** | Estimated reading time: 25 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Clean Architecture and Hexagonal Architecture (Ports and Adapters) are two related ideas with the same core rule: **business logic must not depend on infrastructure details**. HTTP, databases, queues, and file systems are all interchangeable adapters — they plug in and out without touching the domain. This chapter shows how to apply these ideas in Go without any framework.

---

## 30.1 — The four layers

```
┌─────────────────────────────┐
│   Transport / Entry Points  │  HTTP handlers, CLI, gRPC, cron jobs
├─────────────────────────────┤
│   Application / Use Cases   │  Orchestrates domain objects; owns ports
├─────────────────────────────┤
│   Infrastructure / Adapters │  DB, SMTP, event buses, external APIs
├─────────────────────────────┤
│   Domain                    │  Entities, value objects, domain errors
└─────────────────────────────┘
```

**The dependency rule**: arrows point inward. Domain has zero imports. Application imports domain only. Infrastructure implements interfaces defined in application. Transport calls application.

---

## 30.2 — Domain layer

Pure business entities and rules. No `import` of any framework, database, or HTTP package.

```go
type Article struct {
    ID          string
    Title       string
    PublishedAt *time.Time
}

func (a *Article) Publish(now time.Time) error {
    if !a.IsDraft() { return ErrAlreadyPublished }
    a.PublishedAt = &now
    return nil
}
```

Domain errors are also declared here — they are business concepts, not infrastructure codes.

---

## 30.3 — Application layer (use cases)

Orchestrates domain objects. Defines **ports** — interfaces it needs from the outside world:

```go
// Driven (secondary) ports — infrastructure must implement these.
type ArticleRepository interface {
    Save(a Article) error
    FindByID(id string) (Article, error)
}

type PublishNotifier interface {
    NotifyPublished(a Article) error
}
```

The application service receives ports via constructor injection (Chapter 28):

```go
func NewArticleService(repo ArticleRepository, notifier PublishNotifier, ...) *ArticleService
```

---

## 30.4 — Infrastructure layer (adapters)

Concrete implementations of the ports. Each adapter is isolated behind an interface:

- `memArticleRepo` implements `ArticleRepository`
- `postgresArticleRepo` could replace it with zero changes to the application
- `stdoutNotifier`, `smtpNotifier`, `slackNotifier` all implement `PublishNotifier`

---

## 30.5 — Ports and Adapters (Hexagonal Architecture)

Hexagonal Architecture uses the language of **ports** (interfaces) and **adapters** (implementations):

| Term | Meaning in Go |
|---|---|
| Primary port | Interface the application exposes (use-case interface) |
| Driving adapter | HTTP handler, CLI, batch job — calls the primary port |
| Secondary port | Interface the application requires (repository, event bus) |
| Driven adapter | DB, SMTP, queue — implements the secondary port |

The application is the hexagon. Adapters plug into the sides. Swapping an adapter does not affect the hexagon.

---

## 30.6 — The composition root

All concrete types are assembled in `main()`. No layer creates its own dependencies:

```go
func main() {
    store    := postgres.NewStore(os.Getenv("DATABASE_URL"))
    notifier := smtp.NewNotifier(os.Getenv("SMTP_HOST"))
    svc      := app.NewArticleService(store, notifier, ...)
    handler  := http.NewHandler(svc)
    http.ListenAndServe(":8080", handler)
}
```

---

## Running the examples

```bash
cd book/part3_designing_software/chapter30_clean_architecture

go run ./examples/01_layers         # four-layer article publishing service
go run ./examples/02_ports_adapters # inventory service with multiple adapters

go run ./exercises/01_add_adapter   # JSONAdapter primary adapter wired to same service
```

---

## Key takeaways

1. **Dependency rule**: domain ← application ← infrastructure ← transport. Never reversed.
2. **Ports** are interfaces defined by the application layer — not by infrastructure.
3. **Adapters** implement ports. Swapping adapters requires no changes to business logic.
4. **The composition root** (`main`) is the only place where all layers are assembled.
5. **Domain purity**: domain entities must compile and be tested with no framework imports.

---

## Cross-references

- **Chapter 27** — Consumer-side interfaces are how ports work in Go
- **Chapter 28** — Constructor injection is how adapters are delivered to the application
- **Chapter 29** — DIP is the SOLID principle that underlies the dependency rule
- **Chapter 34** — Repository Pattern: deep dive into the persistence port
