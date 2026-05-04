# Chapter 89 Checkpoint — Logging Strategy

## Concept checks

1. Why should `DEBUG` logs never be enabled in production by default?

2. A health-check endpoint is called 10 000 times/minute. You want to log
   at most 10 events/minute. Which sampling strategy is best — head-based
   1-in-N or rate-based? What value of N would you use?

3. You apply a `ScrubbingHandler`. An upstream service sends the field
   `"auth_header": "Bearer eyJhbGci..."`. The key is not in `sensitiveKeys`.
   How do you ensure it gets redacted?

4. A developer calls `logger.Error("request failed", "user", user)` where
   `user` is a struct containing an email address. The ScrubbingHandler only
   inspects `slog.Attr` string values. Will the email be scrubbed?

5. Explain why `slog.LevelVar` is preferred over restarting the service to
   change log verbosity in production.

## Code review

```go
func handleLogin(w http.ResponseWriter, r *http.Request) {
    email := r.FormValue("email")
    pass  := r.FormValue("password")
    slog.Info("login attempt", "email", email, "password", pass)
    // ...
}
```

List all problems and propose a corrected version.

## Expected answers

1. DEBUG logs typically include raw payloads, SQL queries, and internal
   state — too expensive in volume and too risky for PII leakage.

2. Rate-based (2 tokens/s → ~120/min if bursts are acceptable; or
   1 token/6s → exactly 10/min). Head-based 1-in-1000 would also work
   (10 000/1000 = 10), but is not adaptive to traffic shape.

3. Add `"auth_header"` to `sensitiveKeys`, or add a regex pattern that
   matches Bearer tokens, or normalise the key before lookup
   (`strings.HasPrefix(key, "auth")`).

4. No — the struct's `LogValue()` method (or default `%v` formatting)
   would need to be called to produce a string. Implement `slog.LogValuer`
   on the user struct to control what gets logged, or convert to a string
   before passing to `slog`.

5. `slog.LevelVar` is an atomic value that any handler can read on each
   log call, so changing it takes effect immediately for all in-flight
   requests without restarting the process.

**Code problems**: (1) password logged in plain text, (2) email logged at
INFO level violates PII policy, (3) no scrubbing handler in the chain.
Fix: use a `ScrubbingHandler`, replace `"password", pass` with nothing (or
`"password_len", len(pass)`), and replace `"email", email` with
`"email_domain", emailDomain(email)`.
