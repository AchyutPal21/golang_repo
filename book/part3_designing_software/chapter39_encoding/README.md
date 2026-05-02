# Chapter 39 — Encoding

> **Part III · Designing Software** | Estimated reading time: 20 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Every Go program that communicates with the outside world needs to encode and decode data. Go's `encoding` sub-packages all share the same philosophy: work with `io.Reader`/`io.Writer` so they compose naturally with the I/O patterns from Chapter 38.

---

## 39.1 — encoding/json basics

### Struct tags

```go
type User struct {
    ID       int        `json:"id"`
    Password string     `json:"-"`              // never serialised
    Phone    string     `json:"phone,omitempty"` // omit when zero
    Updated  *time.Time `json:"updated_at,omitempty"` // nil → omitted
}
```

| Tag value | Effect |
|---|---|
| `json:"name"` | use `name` as the JSON key |
| `json:"-"` | exclude field from JSON entirely |
| `json:",omitempty"` | omit field when it is the zero value |
| `json:"name,omitempty"` | both rename and omit |

### Marshal / Unmarshal

```go
data, err := json.Marshal(v)            // compact
data, err := json.MarshalIndent(v, "", "  ") // pretty
err = json.Unmarshal(data, &v)
```

### Streaming Encoder / Decoder (NDJSON)

For large payloads or newline-delimited JSON, use `Encoder`/`Decoder` to avoid loading everything into memory:

```go
enc := json.NewEncoder(w)
enc.Encode(record) // appends newline

dec := json.NewDecoder(r)
for dec.More() {
    var record T
    dec.Decode(&record)
}
```

### Unknown fields

By default unknown fields are silently ignored. To treat them as errors:

```go
dec := json.NewDecoder(r)
dec.DisallowUnknownFields()
```

### Generic JSON

When the schema is unknown at compile time, decode into `map[string]any`. JSON numbers always decode to `float64`.

---

## 39.2 — Custom json.Marshaler / json.Unmarshaler

Implement the interfaces to control exactly how a type serialises:

```go
func (m Money) MarshalJSON() ([]byte, error) {
    return json.Marshal(fmt.Sprintf("%s %.2f", m.Currency, float64(m.Cents)/100))
}

func (m *Money) UnmarshalJSON(data []byte) error {
    var s string
    json.Unmarshal(data, &s)
    fmt.Sscanf(s, "%s %f", &m.Currency, &amount)
    m.Cents = int64(amount * 100)
    return nil
}
```

Rules:
- `MarshalJSON` on value receiver — called for both `T` and `*T`.
- `UnmarshalJSON` on pointer receiver — required so the method can modify the value.
- Delegate to `json.Marshal`/`json.Unmarshal` for primitives to get correct quoting and escaping.

---

## 39.3 — encoding/csv

```go
w := csv.NewWriter(dst)
w.Write([]string{"col1", "col2"}) // header
w.Write([]string{"val1", "val2"})
w.Flush()

r := csv.NewReader(src)
records, err := r.ReadAll() // [][]string
```

All values are strings — format numbers with `fmt.Sprintf` before writing and parse with `strconv` after reading.

---

## 39.4 — encoding/binary

For fixed-size wire protocols:

```go
type PacketHeader struct {
    Version  uint8
    Type     uint8
    Length   uint16
    Sequence uint32
}

binary.Write(w, binary.BigEndian, hdr)   // 8 bytes
binary.Read(r, binary.BigEndian, &hdr2)
```

Only fixed-size types work (no slices, strings, or interfaces). Use `binary.BigEndian` for network byte order, `binary.LittleEndian` for native formats.

---

## 39.5 — encoding/base64

| Encoding | Use case |
|---|---|
| `base64.StdEncoding` | email, general data |
| `base64.URLEncoding` | URLs, filenames |
| `base64.RawURLEncoding` | JWTs, query params (no padding) |

```go
encoded := base64.StdEncoding.EncodeToString(data)
decoded, err := base64.StdEncoding.DecodeString(encoded)

// Basic auth header
creds := base64.StdEncoding.EncodeToString([]byte("user:pass"))
header := "Authorization: Basic " + creds
```

---

## Running the examples

```bash
cd book/part3_designing_software/chapter39_encoding

go run ./examples/01_json           # struct tags, marshal, streaming, unknown fields, generic
go run ./examples/02_custom_encoding # custom Marshaler, csv, binary, base64

go run ./exercises/01_api_client    # full client: JSON round-trip, NDJSON decode, CSV export, auth
```

---

## Key takeaways

1. **Struct tags** control JSON key names, exclusion (`-`), and zero-value omission (`omitempty`).
2. **Encoder/Decoder** stream data without loading it all into memory — prefer for NDJSON and large payloads.
3. **Custom MarshalJSON/UnmarshalJSON** let you encode any type as any JSON shape — always delegate primitives back to `json.Marshal`/`json.Unmarshal`.
4. **encoding/csv** treats everything as strings; convert explicitly.
5. **encoding/binary** encodes fixed-size structs directly — fast, compact, no schema, no versioning.
6. **encoding/base64** `RawURLEncoding` is the right choice for JWTs and query parameters (no `+`, `/`, or `=`).

---

## Cross-references

- **Chapter 38** — Files, Streams, I/O: `json.Encoder`/`Decoder` accept `io.Writer`/`io.Reader`
- **Chapter 40** — Configuration: `encoding/json` and `encoding/csv` are common config-file formats
