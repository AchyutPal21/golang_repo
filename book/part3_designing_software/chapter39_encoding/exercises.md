# Chapter 39 — Exercises

## 39.1 — API client simulation

Run [`exercises/01_api_client`](exercises/01_api_client/main.go).

A simulated order API client with custom encoding types (`Timestamp`, `Cents`), streaming NDJSON decode, CSV export, and base64 auth header generation.

Try:
- Add a `LineItem.Subtotal() Cents` method and include it as `subtotal` in the JSON output using `MarshalJSON` on `LineItem`.
- Add `DisallowUnknownFields` to the streaming decoder and introduce an extra field in one NDJSON line — observe the error and handle it gracefully so the remaining lines still decode.
- Add a `RawURLEncoding` variant of the auth header (suitable for embedding in a URL query parameter) and verify it contains no `+`, `/`, or `=` characters.

## 39.2 ★ — Configuration file reader

Build a `ConfigLoader` that reads JSON or CSV configuration depending on the file extension:
- `.json` → `json.Decoder` into a `map[string]any`
- `.csv` → `csv.Reader` into `[]map[string]string` (first row is headers)

Use `strings.NewReader` for the underlying source. Print all key-value pairs.

## 39.3 ★★ — Binary protocol codec

Define a `Message` struct:

```go
type Message struct {
    Version   uint8
    Type      uint8
    Flags     uint16
    Timestamp uint32 // Unix seconds
    Length    uint32 // body byte count
}
```

Implement:
- `Encode(m Message, body []byte) ([]byte, error)` — write header then body
- `Decode(data []byte) (Message, []byte, error)` — read header, then read `Length` bytes of body

Verify round-trip for a body of 256 bytes. Check that a truncated input returns an appropriate error.
