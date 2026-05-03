# Chapter 63 Checkpoint — Structured Logging

## Self-assessment questions

1. What is the difference between `log/slog.NewTextHandler` and `NewJSONHandler`? When would you use each?
2. How do you create a child logger that pre-attaches `user_id` and `request_id` to every subsequent log line?
3. Why does `slog.Group` exist? Give a concrete example where it improves readability in JSON output.
4. What are the four methods you must implement to create a custom `slog.Handler`?
5. How do you inject a per-request logger into context, and how does a handler retrieve it?
6. What level should production services log at, and why?
7. How do you silence log output in unit tests?

## Checklist

- [ ] Can use JSON and Text handlers and configure their log level threshold
- [ ] Can attach structured attributes using `slog.String`, `slog.Int`, `slog.Bool`, `slog.Any`
- [ ] Can create child loggers with `With()` and understand that all subsequent calls carry the pre-set attrs
- [ ] Can use `slog.Group()` to produce nested JSON objects
- [ ] Can propagate a logger through context using a typed context key
- [ ] Can implement a custom `slog.Handler` that wraps an inner handler and adds fields
- [ ] Can write HTTP logging middleware that injects a per-request child logger into context
- [ ] Can build a multi-handler that fans log records out to multiple destinations/formats
- [ ] Know how to discard logs in tests

## Answers

1. `TextHandler` emits `key=value` pairs — human-readable but hard to parse programmatically. `JSONHandler` emits one JSON object per line — machine-parseable, ideal for log aggregation systems (Loki, CloudWatch, Datadog). Use `TextHandler` for local dev/CLI tools, `JSONHandler` for deployed services.

2. `child := logger.With(slog.String("user_id", uid), slog.String("request_id", rid))` — every call on `child` will include those two fields.

3. `slog.Group` namespaces related fields under a key, producing `{"request":{"method":"GET","path":"/users"}}` instead of flat `method=GET path=/users`. Useful for separating request, response, and db sub-objects.

4. `Enabled(ctx, level) bool`, `Handle(ctx, Record) error`, `WithAttrs([]Attr) Handler`, `WithGroup(string) Handler`.

5. Store with `ctx = context.WithValue(ctx, keyLogger, logger)` using a typed key. Retrieve with `ctx.Value(keyLogger).(*slog.Logger)`. Use accessor functions (`loggerFromCtx`, `withLogger`) to keep the key private.

6. `LevelInfo`. Debug generates too much volume; info gives operational visibility without cost.

7. `slog.New(slog.NewTextHandler(io.Discard, nil))` — all records are swallowed.
