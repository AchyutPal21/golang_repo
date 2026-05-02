# Chapter 39 — Revision Checkpoint

## Questions

1. What is the difference between `json.Marshal` and `json.Encoder.Encode`?
2. When do you need a pointer receiver on `UnmarshalJSON` and why?
3. What does `json:"-"` do and how does it differ from `json:",omitempty"`?
4. What encoding does `base64.RawURLEncoding` omit compared to `base64.StdEncoding`, and why does that matter for JWTs?
5. Why does `encoding/binary` require fixed-size types and what happens if you pass a `string` or `[]byte`?

## Answers

1. `json.Marshal` encodes a value into a `[]byte` in memory — the entire encoded form is held before you can use it. `json.Encoder.Encode` writes directly to an `io.Writer` and appends a newline. For large payloads or NDJSON streams, `Encoder` is preferred because it never builds the full encoded form in memory; each call encodes and writes one value incrementally.

2. `UnmarshalJSON` must have a pointer receiver (`*T`) because it needs to modify the value it is called on. A value receiver gets a copy, so changes would be discarded. Additionally, `json.Unmarshal` looks for the `json.Unmarshaler` interface on `*T`; a value receiver would not satisfy the interface for pointer values and the method would never be called during unmarshalling.

3. `json:"-"` permanently excludes the field — it is never serialised or deserialised regardless of its value. `json:",omitempty"` still participates in both marshalling and unmarshalling, but during marshalling the field is skipped when it holds the zero value (empty string, 0, nil, false, empty slice/map). The two serve different purposes: `-` is for fields that must never leave the process (passwords, internal state); `omitempty` is for optional fields that should be absent from the output when not set.

4. `base64.RawURLEncoding` replaces `+` with `-` and `/` with `_`, and omits the `=` padding characters. `base64.StdEncoding` uses `+`, `/`, and `=`. The URL-safe variant matters for JWTs because the three standard characters (`+`, `/`, `=`) have special meaning in URLs and query parameters and would need percent-encoding. Using `RawURLEncoding` produces a string that can be embedded in a URL or query parameter without any escaping.

5. `encoding/binary` maps each struct field directly to bytes with no length prefix and no type information in the output. Variable-length types like `string` and `[]byte` have no fixed byte size, so `binary.Write` cannot know how many bytes to allocate or how to reconstruct the value on `Read` without additional metadata. Passing them causes a runtime panic or error. For variable-length data, the convention is to include an explicit `Length uint32` field in the header and write/read the body separately after the fixed-size header.
