# Chapter 89 Exercises — Logging Strategy

## Exercise 1 (provided): Log Pipeline

Location: `exercises/01_log_pipeline/main.go`

Composes three `slog.Handler` layers:
- Async buffered writer (non-blocking, drop on back-pressure)
- 1-in-5 sampler that always passes WARN/ERROR through
- PII scrubber (email / sensitive key redaction)

## Exercise 2 (self-directed): Tail Sampler

Implement a *tail-based* sampler:
- Buffer the last 100 log records in memory per request ID
- If the request completes with status >= 500 or duration > 1s, flush all
  buffered records to the underlying handler
- Otherwise discard them
- This is how Jaeger/Tempo tail sampling works

## Exercise 3 (self-directed): Dynamic Level via HTTP

Build a small HTTP handler (in-process, no real server) that accepts a POST
body `{"level":"debug"}` and calls `levelVar.Set(...)` to change the log
level at runtime. Write a test that verifies DEBUG records appear after the
level change and disappear after setting back to INFO.

## Stretch Goal: Structured Error Fields

Extend the `ScrubbingHandler` to recognise errors that implement
`interface { Unwrap() []error }` (multi-errors) and log each cause as a
separate attribute `"cause_0"`, `"cause_1"`, etc. Also scrub any string
fields within error messages.
