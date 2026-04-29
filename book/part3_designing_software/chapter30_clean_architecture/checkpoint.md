# Chapter 30 — Revision Checkpoint

## Questions

1. State the dependency rule for Clean Architecture.
2. What is the difference between a primary port and a secondary port?
3. Why must the domain layer have no imports from infrastructure?
4. What is a driving adapter? Give two examples.
5. Where is the only valid place to import concrete infrastructure types?

## Answers

1. Dependencies always point **inward**. Transport depends on application; application
   depends on domain. No inner layer may import an outer layer. The domain has zero
   imports from any other layer of the application. This ensures that business rules
   are independent of frameworks, databases, and delivery mechanisms.

2. A **primary (driving) port** is the interface the application *exposes* to the
   outside world — it describes what the application *can do* (use-case interface).
   A driving adapter (HTTP handler, CLI, cron job) calls into this port.
   A **secondary (driven) port** is an interface the application *requires* from the
   outside world — it describes what the application *needs* (repository, event bus).
   A driven adapter (Postgres, SMTP, Kafka) implements this port. The application
   defines both; infrastructure implements only the driven ports.

3. If the domain imported database or HTTP packages, it would be impossible to test
   domain logic without spinning up those services. It would also create an import
   cycle when infrastructure needs to return domain types. Domain purity means: unit
   tests of business rules compile and run with `go test` and no external services.

4. A **driving adapter** translates an external caller's protocol into use-case calls.
   Examples: (a) an HTTP handler that parses a JSON request body and calls
   `svc.Reserve(sku, qty)`; (b) a CLI adapter that reads command-line flags and
   calls the same method. Both call the *same* application service through the same
   primary port — swapping the HTTP framework does not touch business logic.

5. Concrete infrastructure types (Postgres drivers, SMTP clients, event bus clients)
   are imported only in the **composition root** — `main()` (or the binary's
   top-level initialiser). No service, no use case, no domain entity ever imports
   a concrete infrastructure type. The composition root is the only place where
   all layers meet.
