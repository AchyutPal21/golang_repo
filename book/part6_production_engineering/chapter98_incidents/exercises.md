# Chapter 98 Exercises — Incident Management & Debugging

## Exercise 1 (provided): Incident Timeline & Debugging Toolkit

Location: `exercises/01_incident_timeline/main.go`

Combined debugging and incident management toolkit:
- `IncidentTimeline` with severity-tagged events
- `HealthSnapshot` capturing goroutine/memory/GC state
- Incident report generator (markdown format)
- MTTR calculator from timeline events

## Exercise 2 (self-directed): Panic Recovery Middleware

Build an HTTP middleware that recovers from panics:
- Capture panic value and full stack trace (`runtime.Stack`)
- Respond with HTTP 500 and a structured JSON error body
- Log the incident with: request ID, path, method, user-agent, panic value, truncated stack
- Return a unique `incident_id` in the response for client-side correlation
- Never leak the stack trace to the HTTP response

## Exercise 3 (self-directed): Goroutine Leak Detector

Build a goroutine leak detector for tests:
- `LeakDetector` struct with `Before()` and `After(t *testing.T)` methods
- `Before()` records the current goroutine count and known goroutine names
- `After()` waits up to 100ms for goroutines to settle, then compares
- On leak: print which goroutines are new (use `runtime.Stack(buf, true)`)
- Ignore known Go runtime goroutines (signal handler, finalizer, etc.)

## Stretch Goal: Runbook Generator

Build a `Runbook` type that:
- Stores ordered diagnostic steps: `Step{Name, Command, ExpectedOutput string}`
- `Run(alertName string)` — prints the runbook for the given alert
- Supports alert types: `high_error_rate`, `goroutine_leak`, `oom`, `slow_latency`
- Each step includes: what to check, what command to run, what the output means
