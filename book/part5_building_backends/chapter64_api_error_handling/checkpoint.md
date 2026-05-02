# Chapter 64 — API Error Handling

## Questions

1. What is RFC 7807 and what problem does it solve compared to ad-hoc `{"error": "..."}` responses?
2. What is the purpose of the `type` field in an RFC 7807 Problem Detail, and why should it be a URI?
3. Why is the `detail` field for 5xx errors typically a generic message while 4xx errors contain specific information?
4. Explain the handler-returns-error pattern. What advantage does it have over writing error responses inside each handler?
5. What is the difference between `title` and `detail` in a Problem Detail, and which field should client code inspect to distinguish error types?

## Answers

1. **RFC 7807** defines a standard JSON response body structure for HTTP API errors with a registered media type (`application/problem+json`). Without it, every API invents its own error format: `{"error": "..."}`, `{"message": "...", "code": 404}`, `{"errors": [...]}` — clients must read documentation for each API to understand the structure. RFC 7807 solves this by providing a common schema with stable field names (`type`, `title`, `status`, `detail`, `instance`) that generic HTTP client libraries and API gateways can understand. The `type` URI is machine-readable so clients can write conditional logic against it without hardcoding HTTP status codes.

2. The `type` field is a URI (not necessarily dereferenceable, but stable) that uniquely identifies the problem class. It serves as the **stable identifier** that clients program against: `if problem.Type == "https://api.example.com/errors/not-found"`. This is more reliable than checking `status` (multiple 4xx errors can mean different things) or `title` (which might be translated or slightly changed). The URI convention also allows documentation: if the URI is actually resolvable (e.g., an HTML page at that URL), it provides human-readable documentation about the error type, what causes it, and how to fix it.

3. **5xx errors are the server's fault.** The detail of a 5xx error typically includes internal information (database connection strings, stack traces, service names, hostnames) that would help an attacker exploit the system. Sending `"an unexpected error occurred"` prevents information disclosure. The actual cause is logged internally with full detail so engineers can diagnose it. **4xx errors are the client's fault.** The client needs to understand exactly what they did wrong so they can fix their request: `"field 'email' must be a valid email address"` is actionable. Generic 4xx messages like `"bad request"` force clients to guess what to fix.

4. The **handler-returns-error pattern** defines handlers as `func(w, r) error`. A wrapper adapts them to `http.HandlerFunc` by calling the handler and routing any returned error to a central `handleErr` function. Advantages: **(1)** Eliminates repetitive error-writing boilerplate — each handler calls `return ErrNotFound(...)` instead of 5+ lines of JSON encoding. **(2)** Guarantees consistent error format — all errors go through the same conversion logic. **(3)** Centralizes cross-cutting concerns (logging, correlation IDs) in one place. **(4)** Handlers are easier to test because they return errors rather than writing to a `ResponseWriter` — you can inspect the error without parsing HTTP responses.

5. `title` is a **stable, human-readable summary** of the error type — it does not change between requests for the same `type` URI (e.g., always `"Not Found"` for `not-found`). `detail` is **specific to the occurrence** — it changes per request (e.g., `"article '42' does not exist"` vs `"article '99' does not exist"`). Client **code** should inspect `type` (the URI) to branch logic, not `title` (which might be translated) or `detail` (which might be reformatted). `title` is for humans reading an error response; `type` is for machines processing it.
