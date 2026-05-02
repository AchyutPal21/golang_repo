# Chapter 40 — Revision Checkpoint

## Questions

1. Why should secrets never come from a config file checked into version control, and how do you enforce this in Go?
2. What is the purpose of pointer fields in a "partial" JSON config struct?
3. What is the precedence order for the four config layers, and why are CLI flags the highest priority?
4. When should you choose functional options over a plain config struct parameter?
5. What invariant does a validated `Config` struct provide, and why does that matter for the rest of the application?

## Answers

1. Config files are frequently committed to version control, shared with teams, and included in build artefacts. A secret in a file risks leaking through git history, CI logs, container images, and code reviews — surfaces that are hard to audit and even harder to revoke. In Go you enforce this by tagging secret fields with `json:"-"` (preventing JSON deserialisation) and reading them only from environment variables. Wrapping secrets in a `Secret` type whose `String()` returns `"<redacted>"` adds a second layer: even if a `Secret` value is accidentally passed to a logger, the raw string never appears.

2. When you unmarshal JSON into a struct with non-pointer fields, every field is set — fields absent from the JSON are silently set to their Go zero values (0, "", false), which overwrites the defaults you set earlier. With pointer fields, a field absent from the JSON remains `nil`, which you can distinguish from "was explicitly set to zero". After unmarshalling, you only copy non-nil values over the defaults, so the JSON file only needs to contain the fields it wants to override.

3. Precedence (lowest to highest): code defaults → config file → environment variables → CLI flags. CLI flags are highest because they represent an explicit, operator-supplied override for a specific invocation — they should always win. Environment variables sit above file config because they encode deployment context (staging vs production) that the file, which might be shared across environments, should not override. Defaults are lowest because they exist only to ensure the program starts safely with no external configuration.

4. Use functional options when the type is consumed by callers who should not need to know its internal fields, when there are many optional parameters with defaults and the callers will supply only a few, or when the library is public and you want to add new options in future without breaking existing call sites. A plain config struct parameter is simpler and perfectly fine for internal types or when there are only one or two required fields. The key signal for functional options is "callers name only the fields they care about."

5. A validated `Config` provides the invariant that every field is within its accepted range and all required values are present — the program will not encounter "port 0", "invalid log level", or an empty JWT secret at runtime. Because `Load` validates before returning, and the rest of the application receives the config only after `Load` succeeds, defensive checks in service code are unnecessary and can be omitted. This concentrates all validation in one place and allows the rest of the codebase to trust the config unconditionally.
