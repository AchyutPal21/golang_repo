# Chapter 29 — SOLID in Go

> **Part III · Designing Software** | Estimated reading time: 25 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

SOLID is a set of five design principles that make object-oriented code more maintainable and extensible. Go is not object-oriented in the class-hierarchy sense, but every SOLID principle applies — and Go's type system enforces several of them naturally. This chapter maps each principle to idiomatic Go idioms, shows what a violation looks like, and demonstrates the fix.

---

## 29.1 — S: Single Responsibility Principle

**A type should have one reason to change.**

In Go: keep types small and focused. If a struct handles business logic, persistence, *and* email, it has three reasons to change. Split it into three types, each owned by one concern. A thin coordinator wires them together.

The bad pattern: one `OrderProcessor` that does pricing, tax, emailing, and database writes.
The fix: `PriceCalculator`, `TaxPolicy`, `ConfirmationMailer`, `OrderRepository`, and a `Coordinator` that composes them.

---

## 29.2 — O: Open/Closed Principle

**Open for extension, closed for modification.**

In Go: express extension points as interfaces. New behaviour arrives as new types that satisfy the interface — not as edits to existing `switch` blocks.

The bad pattern: `applyDiscount(type, price, value)` with a `switch` — every new discount type requires editing the function.
The fix: `type Discount interface { Apply(float64) float64 }`. Adding `BuyNGetMDiscount` requires zero edits to `Checkout`.

---

## 29.3 — L: Liskov Substitution Principle

**Subtypes must be substitutable for their base types without breaking correctness.**

In Go: any type satisfying an interface must honour the full contract implied by that interface — not just the method signatures. A type that panics on a method call violates LSP even if it compiles.

The bad pattern: `ReadOnlyBuffer` claims to implement `ReadWriter` but panics on `Write`.
The fix: split `ReadWriter` into `Reader` and `Writer`. `ReadOnlyBuffer` satisfies `Reader` only — which is all it can honestly promise.

---

## 29.4 — I: Interface Segregation Principle

**Clients should not be forced to depend on methods they do not use.**

In Go: keep interfaces narrow. Consumer-side interface definition (Chapter 27) naturally enforces ISP — each consumer defines only the methods it calls.

The bad pattern: a fat `Storage` interface with `Get`, `Set`, `Delete`, `Flush`, `Stats`, `Backup` — any read-only consumer must stub five methods.
The fix: `Getter`, `Setter`, `Deleter` — composed only when a consumer genuinely needs both.

---

## 29.5 — D: Dependency Inversion Principle

**High-level modules should not depend on low-level modules. Both should depend on abstractions.**

In Go: high-level packages define the interfaces they need; low-level packages provide implementations that satisfy those interfaces. This is just constructor injection (Chapter 28) viewed through the SOLID lens.

The bad pattern: `AlertService` directly constructs `emailSender` — tied to one delivery mechanism, impossible to test.
The fix: `AlertService` receives a `Notifier` interface. `emailNotifier`, `smsNotifier`, `slackNotifier`, and `fanOutNotifier` all satisfy it.

---

## 29.6 — SOLID and Go's type system

| Principle | Go mechanism that enforces it naturally |
|---|---|
| SRP | Small focused structs; package-level separation |
| OCP | Implicit interface satisfaction; new types extend without modifying |
| LSP | Interface contracts; narrow interfaces prevent impossible promises |
| ISP | Consumer-side interface definition (Chapter 27) |
| DIP | Constructor injection (Chapter 28); interfaces defined in consumer package |

---

## Running the examples

```bash
cd book/part3_designing_software/chapter29_solid_in_go

go run ./examples/01_srp_ocp            # SRP and OCP with concrete before/after
go run ./examples/02_lsp_isp_dip        # LSP, ISP, and DIP with concrete before/after

go run ./exercises/01_refactor_violations  # five violations, five fixes, all running
```

---

## Key takeaways

1. **SRP** — one type, one reason to change. Split concerns; wire with a coordinator.
2. **OCP** — extend via new types satisfying an interface, not via editing existing `switch` blocks.
3. **LSP** — narrow interfaces let types promise only what they can deliver.
4. **ISP** — consumer-side interfaces are naturally narrow; compose only when needed.
5. **DIP** — high-level types depend on their own interfaces, injected by the composition root.

---

## Cross-references

- **Chapter 27** — Interface-Driven Design: consumer-side interfaces underpin ISP and DIP
- **Chapter 28** — Dependency Injection: constructor injection is DIP in practice
- **Chapter 26** — OOP in Go: composition over inheritance supports SRP
