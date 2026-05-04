# Chapter 89 — Logging Strategy

Production logging is not just `fmt.Println`. It requires structure,
sampling, PII protection, and level discipline — all without crushing
your service's throughput.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | Structured logging & sampling | slog, rate/head sampling, dynamic levels |
| 2 | PII scrubbing | Custom slog.Handler, regex redaction, level-aware masking |
| E | Log pipeline | Async + sampling + scrubbing chain |

## Examples

### `examples/01_structured_sampling`

- `log/slog` JSON handler with RFC3339 time normalisation
- `WithLogger` / `WithRequestID` context propagation
- Head-based sampler: 1-in-N per message key
- Token-bucket rate sampler: N events/second per endpoint key
- Dynamic `slog.LevelVar` — change log level without restart
- Log level strategy reference

### `examples/02_pii_levels`

- Email / phone / credit-card regex redaction
- `ScrubbingHandler` wrapping any `slog.Handler`
- Sensitive key list (`password`, `token`, `secret`, …)
- Level-aware masking — show email in DEBUG, domain-only in INFO+
- Log aggregation pipeline patterns (Fluent Bit / Loki / CloudWatch)

### `exercises/01_log_pipeline`

Three-stage composable pipeline:

```
logger.Info/Warn/Error
  → asyncHandler     (256-slot channel, non-blocking)
  → samplingHandler  (1-in-5 for INFO, always pass WARN/ERROR)
  → scrubbingHandler (email/password scrubbed)
  → slog.JSONHandler → stdout
```

Simulates 100 health-checks + 20 login events → verifies sampling and
scrubbing in the output.

## Key Concepts

**Sampling strategies**
- Head-based (1-in-N): deterministic, low overhead, good for high-volume INFO
- Rate-based (N/s per key): adaptive to traffic shape; allow bursts

**PII scrubbing rules**
1. Scrub by key name first (password, token, ssn, …)
2. Then pattern-match values (email, phone, card)
3. At INFO+ level, never log full email — use domain only
4. Apply scrubbing as a handler layer, not at call sites

**Level discipline**
- DEBUG never in production (too verbose, often contains raw payloads)
- INFO sampled for high-volume endpoints (health, metrics)
- WARN always emitted; alert on sustained rate
- ERROR always emitted; page on spike
- Use `slog.LevelVar` for runtime level changes via signal or admin API

## Running

```bash
go run ./part6_production_engineering/chapter89_logging_strategy/examples/01_structured_sampling
go run ./part6_production_engineering/chapter89_logging_strategy/examples/02_pii_levels
go run ./part6_production_engineering/chapter89_logging_strategy/exercises/01_log_pipeline
```
