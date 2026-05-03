# Chapter 62 Checkpoint — Input Validation

## Self-assessment questions

1. What is the difference between HTTP 400 and HTTP 422? When should you use each?
2. Why should validation collect all errors before returning, rather than stopping at the first failure?
3. What is a `ValidationError` type and what fields should it contain?
4. How do you validate query parameters in an HTTP handler (e.g. `?page=abc`)?
5. When should you pre-compile regular expressions for validation?
6. Why return `application/json` error bodies from API endpoints, even for errors?

## Checklist

- [ ] Can define `ValidationError` and `ValidationErrors` with a useful `Error()` method
- [ ] Can write a collect-all validator that accumulates all field errors
- [ ] Can return HTTP 400 for JSON parse failures and 422 for semantic validation failures
- [ ] Can produce a structured `{"error":"validation failed","fields":[...]}` response
- [ ] Can validate query parameters and convert their string values with error handling
- [ ] Can validate path parameters (e.g. checking format of an ID string)
- [ ] Know common validation rules: required, min/max length, pattern, range, enum
- [ ] Know when to use `regexp.MustCompile` at package level vs inside functions

## Answers

1. 400 Bad Request = malformed input that can't be parsed (bad JSON, wrong content-type, binary when text expected). 422 Unprocessable Entity = input is syntactically valid (JSON parses) but semantically invalid (required field missing, value out of range). Clients use this distinction: 400 means "fix your request format", 422 means "fix your data values".

2. The client gets all problems in a single round-trip instead of discovering them one by one. This is a UX consideration: a form that only tells you about the first error forces the user to submit multiple times. Collect-all is especially valuable for registration forms, bulk import APIs, and anywhere the user typed multiple fields.

3. `ValidationError{Field string, Rule string, Message string}`. `Field` identifies what failed (maps to a JSON key). `Rule` identifies the constraint (useful for i18n client-side). `Message` is human-readable. All three together let clients show precise field-level errors.

4. `r.URL.Query().Get("page")` returns a string. Convert with `strconv.Atoi()` or `strconv.ParseFloat()`. If conversion fails, append a `ValidationError{Field:"page", Rule:"invalid_format"}` to the error slice.

5. At package level with `var reSKU = regexp.MustCompile(...)`. Compilation is expensive; calling `regexp.MustCompile` inside a hot path (per request) wastes CPU and allocates. Package-level `var` compiles once at program start.

6. API clients are programs, not people. Plain text errors require the client to parse free-form strings to decide what to do. JSON errors with stable field names (`error`, `fields`, `field`, `rule`) allow clients to programmatically display field-level error messages in forms without any string parsing.
