# Chapter 32 — Structural Patterns

> **Part III · Designing Software** | Estimated reading time: 22 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Structural patterns describe how types and objects are composed. Go's implicit interface satisfaction and first-class interface values make structural patterns concise and idiomatic — no inheritance required. This chapter covers Adapter, Decorator, Proxy, Composite, and Facade.

---

## 32.1 — Adapter

Converts the interface of one type into the interface a consumer expects. Useful when integrating third-party libraries or legacy systems.

```go
type MessageSender interface { Send(to, subject, body string) error }

// SMSAdapter wraps ExternalSMSClient and satisfies MessageSender.
type SMSAdapter struct{ client *ExternalSMSClient }

func (a *SMSAdapter) Send(to, subject, body string) error {
    return a.client.SendSMS(to, subject+": "+body)
}
```

Rule: adapters do translation, not business logic. If you find yourself adding validation or rules in an adapter, move them to a service.

---

## 32.2 — Decorator

Wraps a value that implements interface I and returns a new value that also implements I — with added behaviour before and/or after the delegation:

```go
type loggingMessageSender struct{ inner MessageSender }

func (l *loggingMessageSender) Send(to, subject, body string) error {
    start := time.Now()
    err := l.inner.Send(to, subject, body)
    fmt.Printf("[LOG] to=%s elapsed=%s err=%v\n", to, time.Since(start), err)
    return err
}
```

Decorators compose: `WithLogging(WithRetry(base, 3))` — each layer adds one concern. This is how Go implements cross-cutting concerns without inheritance.

---

## 32.3 — Proxy

Controls access to an object; implements the same interface as the real subject:

- **Caching proxy** — returns cached results; delegates on miss.
- **Access control proxy** — checks permissions before delegating.
- **Lazy proxy** — defers expensive initialisation until first call.

The distinction from Decorator: a Proxy *controls access*; a Decorator *adds behaviour*. In practice, the structural implementation is the same.

---

## 32.4 — Composite

Treats individual objects and groups uniformly through a shared interface:

```go
type FileSystemNode interface {
    Name() string
    Size() int
    Display(indent int)
}
```

`File` is a leaf; `Directory` holds `[]FileSystemNode`. Callers call `Size()` or `Display()` on the root — the tree walks itself. Adding a new node type requires no changes to consumers.

---

## 32.5 — Facade

Provides a simplified interface to a complex subsystem:

```go
func (m *MediaConverter) ConvertToMP4(input, output string) {
    // hides: codec setup, video decode, audio decode, encode, mix
}
```

The facade does not prevent direct access to the subsystem — advanced callers can still use the individual components. The facade just handles the 80% case.

---

## 32.6 — Middleware chains

Middleware is Decorator applied to request handlers. The `Chain` function composes middleware functions right-to-left so the first in the list runs outermost:

```go
handler := Chain(routerHandler,
    WithLogging,
    WithAuth,
    WithRateLimit(4),
)
```

---

## Running the examples

```bash
cd book/part3_designing_software/chapter32_structural_patterns

go run ./examples/01_adapter_decorator      # Adapter (SMS, legacy email), Decorator (log, retry, rate limit)
go run ./examples/02_proxy_composite_facade # Proxy (cache, ACL), Composite (fs tree), Facade (media)

go run ./exercises/01_middleware_chain      # HTTP-style middleware stack with Chain builder
```

---

## Key takeaways

1. **Adapter** — translates interfaces; wraps external/legacy types.
2. **Decorator** — adds behaviour before/after; composes cross-cutting concerns.
3. **Proxy** — controls access; same interface as the real subject.
4. **Composite** — uniform treatment of leaves and composites via shared interface.
5. **Facade** — simplifies a complex subsystem; does not restrict advanced access.
6. **Middleware** — Decorator applied to request handlers; compose with `Chain`.

---

## Cross-references

- **Chapter 27** — Consumer-side interfaces enable implicit adapter satisfaction
- **Chapter 14** — Closures: middleware functions capture their `next` handler
- **Chapter 33** — Behavioral Patterns: Strategy, Observer, Command
