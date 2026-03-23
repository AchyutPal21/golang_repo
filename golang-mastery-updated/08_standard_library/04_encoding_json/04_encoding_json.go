// 04_encoding_json.go
//
// encoding/json: Go's built-in JSON codec.
//
// WHY THE JSON PACKAGE MATTERS:
// JSON is the lingua franca of web APIs. The encoding/json package uses
// REFLECTION to inspect types at runtime, which makes it flexible but
// also means it has a performance cost. For performance-critical paths,
// libraries like encoding/json/v2, json-iterator, or easyjson exist, but
// the standard library package is correct and sufficient for most use cases.
//
// TWO MODES:
//   Marshal/Unmarshal  — in-memory: []byte in, struct out (or vice versa)
//   Encoder/Decoder    — streaming: io.Writer/io.Reader (for large data or HTTP)
//
// CRITICAL RULES:
//   1. Only EXPORTED (capital) fields are marshaled/unmarshaled.
//   2. Use struct tags to control field names in JSON.
//   3. ALWAYS close response bodies before unmarshaling (for HTTP).
//   4. For streaming, prefer Decoder/Encoder over Marshal/Unmarshal.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: Basic Marshal / Unmarshal
// ─────────────────────────────────────────────────────────────────────────────

type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
	Zip    string `json:"zip,omitempty"` // omitted if empty string
}

type Person struct {
	// json:"name"     → field is named "name" in JSON
	// json:",omitempty" → omit if zero value (empty string, 0, nil, false)
	// json:"-"        → NEVER include in JSON (good for passwords, tokens)
	// json:"-,"       → field named "-" in JSON (the comma makes it literal)
	Name    string  `json:"name"`
	Age     int     `json:"age"`
	Email   string  `json:"email,omitempty"`
	Address Address `json:"address"`
	Score   float64 `json:"score,omitempty"`
	secret  string  // unexported — NEVER marshaled (invisible to reflection)
	Token   string  `json:"-"` // exported but tagged to exclude
}

func basicMarshalUnmarshal() {
	fmt.Println("═══ SECTION 1: Basic Marshal / Unmarshal ═══")

	// Marshal — Go struct → JSON bytes
	p := Person{
		Name:  "Alice",
		Age:   30,
		Email: "alice@example.com",
		Address: Address{
			Street: "123 Main St",
			City:   "Springfield",
			// Zip omitted → will not appear in JSON because of omitempty
		},
		Score:  98.5,
		secret: "hidden",
		Token:  "bearer-xyz", // tagged with json:"-" → excluded
	}

	data, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Marshaled: %s\n\n", data)
	// Note: secret and Token are missing; Zip is missing (omitempty)

	// MarshalIndent — pretty-printed JSON
	pretty, _ := json.MarshalIndent(p, "", "  ") // prefix="", indent="  "
	fmt.Printf("Pretty:\n%s\n\n", pretty)

	// Unmarshal — JSON bytes → Go struct
	jsonData := []byte(`{
		"name": "Bob",
		"age": 25,
		"address": {"street": "456 Oak Ave", "city": "Shelbyville", "zip": "62701"},
		"unknown_field": "ignored by default"
	}`)

	var bob Person
	if err := json.Unmarshal(jsonData, &bob); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Unmarshaled: %+v\n", bob)
	fmt.Printf("Address: %+v\n", bob.Address)
	// unknown_field is silently ignored — this is default behavior
	// (use json.Decoder.DisallowUnknownFields() to make it an error)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: Marshaling primitive types and collections
// ─────────────────────────────────────────────────────────────────────────────

func primitiveAndCollections() {
	fmt.Println("═══ SECTION 2: Primitives & Collections ═══")

	// Maps — keys must be strings (or implement encoding.TextMarshaler)
	m := map[string]interface{}{
		"bool":   true,
		"int":    42,
		"float":  3.14,
		"string": "hello",
		"null":   nil,
		"array":  []int{1, 2, 3},
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	fmt.Printf("Map to JSON:\n%s\n\n", data)

	// Slices — nil slice marshals to "null"; empty slice to "[]"
	var nilSlice []int
	emptySlice := []int{}
	filled := []int{1, 2, 3}

	nilData, _ := json.Marshal(nilSlice)
	emptyData, _ := json.Marshal(emptySlice)
	filledData, _ := json.Marshal(filled)

	fmt.Printf("nil slice:   %s\n", nilData)   // null
	fmt.Printf("empty slice: %s\n", emptyData) // []
	fmt.Printf("filled:      %s\n", filledData) // [1,2,3]

	// COMMON MISTAKE: nil map marshals to null, nil ptr marshals to null
	var nilMap map[string]int
	nilMapData, _ := json.Marshal(nilMap)
	fmt.Printf("nil map:     %s\n\n", nilMapData) // null

	// Pointer fields — nil pointer marshals to "null", non-nil to the value
	type Config struct {
		Timeout  *int    `json:"timeout,omitempty"` // nil = omit; non-nil = value
		Debug    bool    `json:"debug"`
	}
	t := 30
	cfg := Config{Timeout: &t, Debug: true}
	cfgData, _ := json.Marshal(cfg)
	fmt.Printf("Config: %s\n", cfgData)

	cfgNil := Config{Debug: false}
	cfgNilData, _ := json.Marshal(cfgNil)
	fmt.Printf("Config (nil timeout, omitempty): %s\n\n", cfgNilData)
	// Timeout is omitted because it's nil + omitempty

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: Streaming Encoder / Decoder
// ─────────────────────────────────────────────────────────────────────────────

func streamingDemo() {
	fmt.Println("═══ SECTION 3: Encoder / Decoder (Streaming) ═══")

	// WHY streaming matters:
	// json.Marshal reads ALL data into memory as []byte, then you write it.
	// json.Encoder writes DIRECTLY to an io.Writer — no intermediate allocation.
	// For HTTP responses, use Encoder; for HTTP request bodies, use Decoder.

	// json.Encoder — writing JSON to io.Writer
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ") // optional: pretty print
	enc.SetEscapeHTML(false) // don't escape < > & (useful for non-HTML output)

	people := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}
	for _, p := range people {
		if err := enc.Encode(p); err != nil { // Encode adds a trailing newline
			log.Fatal(err)
		}
	}
	fmt.Printf("Encoded stream:\n%s\n", buf.String())

	// json.Decoder — reading JSON from io.Reader
	// Decoder is essential for:
	// 1. HTTP request/response bodies (io.Reader)
	// 2. NDJSON (newline-delimited JSON) — multiple JSON objects, one per line
	// 3. Large JSON files where you don't want to load all into memory

	jsonStream := strings.NewReader(`
		{"name":"Alice","age":30,"address":{"street":"","city":""}}
		{"name":"Bob","age":25,"address":{"street":"","city":""}}
	`)
	dec := json.NewDecoder(jsonStream)

	for {
		var p Person
		err := dec.Decode(&p)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Decoded: %s, age %d\n", p.Name, p.Age)
	}

	// DisallowUnknownFields — fail if JSON has fields not in the struct
	strictJSON := strings.NewReader(`{"name":"Carol","hacker_field":"oops","age":0,"address":{"street":"","city":""}}`)
	strictDec := json.NewDecoder(strictJSON)
	strictDec.DisallowUnknownFields()
	var carol Person
	err := strictDec.Decode(&carol)
	fmt.Printf("\nDisallowUnknownFields error: %v\n", err)

	// UseNumber — store numbers as json.Number instead of float64
	// WHY: By default, Unmarshal into interface{} converts all JSON numbers to
	// float64. This loses precision for large integers (int64 > 2^53).
	// UseNumber keeps them as the string "json.Number" for precise conversion.
	numJSON := strings.NewReader(`{"id": 9007199254740993}`) // > 2^53
	numDec := json.NewDecoder(numJSON)
	numDec.UseNumber()
	var m map[string]interface{}
	numDec.Decode(&m)
	id := m["id"].(json.Number)
	fmt.Printf("\nLarge int as json.Number: %s\n", id)
	i64, _ := id.Int64()
	fmt.Printf("As int64: %d\n", i64) // precise!

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: Custom MarshalJSON / UnmarshalJSON
// ─────────────────────────────────────────────────────────────────────────────

// Duration wraps time.Duration to marshal as human-readable string ("5s", "2m")
// instead of nanoseconds (the default int64 representation).
type Duration struct {
	d int64 // nanoseconds, unexported
}

func NewDuration(ns int64) Duration { return Duration{d: ns} }

// MarshalJSON implements json.Marshaler interface.
// Called automatically by json.Marshal when encoding a Duration value.
func (d Duration) MarshalJSON() ([]byte, error) {
	// Return a JSON string like "5000000000ns" or "5s"
	seconds := d.d / 1_000_000_000
	return json.Marshal(fmt.Sprintf("%ds", seconds))
}

// UnmarshalJSON implements json.Unmarshaler interface.
// Called automatically by json.Unmarshal when decoding into Duration.
func (d *Duration) UnmarshalJSON(data []byte) error {
	// data is the raw JSON bytes including quotes: `"5s"`
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	var seconds int64
	_, err := fmt.Sscanf(s, "%ds", &seconds)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.d = seconds * 1_000_000_000
	return nil
}

// Job uses Duration with custom JSON serialization
type Job struct {
	Name    string   `json:"name"`
	Timeout Duration `json:"timeout"`
}

func customMarshalingDemo() {
	fmt.Println("═══ SECTION 4: Custom MarshalJSON / UnmarshalJSON ═══")

	job := Job{
		Name:    "backup",
		Timeout: NewDuration(300_000_000_000), // 300 seconds
	}

	data, _ := json.MarshalIndent(job, "", "  ")
	fmt.Printf("Custom marshal:\n%s\n\n", data)
	// "timeout" is "300s" instead of 300000000000

	// Round-trip: unmarshal back
	var job2 Job
	json.Unmarshal(data, &job2)
	fmt.Printf("Round-tripped: name=%s timeout=%dns\n", job2.Name, job2.Timeout.d)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: json.RawMessage — deferred decoding
// ─────────────────────────────────────────────────────────────────────────────

func rawMessageDemo() {
	fmt.Println("═══ SECTION 5: json.RawMessage ═══")

	// json.RawMessage is just []byte — but it implements json.Marshaler and
	// json.Unmarshaler, so it captures raw JSON without decoding.
	//
	// WHY:
	// 1. You don't know the type of a field at compile time (polymorphism).
	// 2. You want to decode a large JSON but only parse specific fields now.
	// 3. You want to pass-through JSON without re-marshaling (proxy patterns).

	type Event struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"` // decoded later based on Type
	}

	events := []string{
		`{"type":"click","payload":{"x":100,"y":200}}`,
		`{"type":"keypress","payload":{"key":"Enter","code":13}}`,
	}

	type ClickPayload struct {
		X, Y int
	}
	type KeypressPayload struct {
		Key  string
		Code int
	}

	for _, raw := range events {
		var e Event
		json.Unmarshal([]byte(raw), &e)

		fmt.Printf("Event type: %s\n", e.Type)
		fmt.Printf("Raw payload: %s\n", e.Payload)

		switch e.Type {
		case "click":
			var cp ClickPayload
			json.Unmarshal(e.Payload, &cp)
			fmt.Printf("  Click at (%d, %d)\n", cp.X, cp.Y)
		case "keypress":
			var kp KeypressPayload
			json.Unmarshal(e.Payload, &kp)
			fmt.Printf("  Key: %s (code %d)\n", kp.Key, kp.Code)
		}
	}

	// RawMessage as passthrough — re-marshal without decoding
	original := []byte(`{"complex":{"nested":[1,2,3]},"other":true}`)
	var m map[string]json.RawMessage
	json.Unmarshal(original, &m)
	m["added"] = json.RawMessage(`"new_value"`) // inject a field
	result, _ := json.Marshal(m)
	fmt.Printf("\nPassthrough with injection: %s\n", result)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 6: Common mistakes
// ─────────────────────────────────────────────────────────────────────────────

func commonMistakes() {
	fmt.Println("═══ SECTION 6: Common Mistakes ═══")

	// MISTAKE 1: Unexported fields are silently ignored
	type hidden struct {
		Public  string // exported — marshaled
		private string // unexported — SILENTLY ignored, no error!
	}
	h := hidden{Public: "visible", private: "invisible"}
	data, _ := json.Marshal(h)
	fmt.Printf("Mistake 1 - unexported: %s\n", data) // {"Public":"visible"}

	// MISTAKE 2: interface{} round-trip loses type info
	// Numbers become float64, not int. Nested objects become map[string]interface{}.
	jsonData := []byte(`{"count": 42, "nested": {"key": "val"}}`)
	var result interface{}
	json.Unmarshal(jsonData, &result)
	m := result.(map[string]interface{})
	count := m["count"] // This is float64, not int!
	fmt.Printf("\nMistake 2 - interface{}: count type is %T, value %v\n", count, count)
	// Cannot do count + 1 without type assertion: int(count.(float64)) + 1

	// MISTAKE 3: Forgetting to pass a pointer to Unmarshal
	var p Person
	// json.Unmarshal(data, p)   // WRONG: p is copied, changes are lost
	json.Unmarshal([]byte(`{"name":"test","age":0,"address":{"street":"","city":""}}`), &p) // CORRECT: &p
	fmt.Printf("\nMistake 3 - must pass pointer: %s\n", p.Name)

	// MISTAKE 4: Checking error from Marshal when using concrete types
	// Marshal only fails with circular references or channels/funcs.
	// With normal structs, the error is almost always nil. But always check anyway.
	_, err := json.Marshal(make(chan int)) // channels can't be marshaled
	fmt.Printf("\nMistake 4 - channel marshal error: %v\n", err)

	// MISTAKE 5: Reusing a Decoder on an HTTP response body after error
	// Always check the status code BEFORE trying to decode.
	// Decoding an error response into your success struct gives garbage data.
	fmt.Println("\nMistake 5: always check status code before json.Decode(resp.Body)")

	// MISTAKE 6: Not closing response body
	fmt.Println("Mistake 6: defer resp.Body.Close() immediately after checking err from http.Get")

	// MISTAKE 7: JSON number precision for large int64
	// Default: large ints round-trip through float64 and lose precision
	type WithID struct {
		ID int64 `json:"id"`
	}
	huge := WithID{ID: 9007199254740993} // 2^53 + 1
	d, _ := json.Marshal(huge)
	var back WithID
	json.Unmarshal(d, &back)
	fmt.Printf("\nMistake 7 - large int64 is safe with typed struct: %d == %d: %v\n",
		huge.ID, back.ID, huge.ID == back.ID)
	// With typed struct (int64 field), this is fine.
	// With interface{} it would become float64 and lose the last bit.

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 7: Error handling
// ─────────────────────────────────────────────────────────────────────────────

func errorHandlingDemo() {
	fmt.Println("═══ SECTION 7: Error Handling ═══")

	// *json.SyntaxError — malformed JSON
	_, err := json.Unmarshal([]byte(`{broken`), &struct{}{})
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		fmt.Printf("SyntaxError at offset %d: %v\n", syntaxErr.Offset, syntaxErr)
	}

	// *json.UnmarshalTypeError — wrong type in JSON
	type S struct{ N int }
	var s S
	err2 := json.Unmarshal([]byte(`{"N": "not-a-number"}`), &s)
	var typeErr *json.UnmarshalTypeError
	if errors.As(err2, &typeErr) {
		fmt.Printf("TypeError: field=%s expected=%s got=%s\n",
			typeErr.Field, typeErr.Type, typeErr.Value)
	}

	// *json.InvalidUnmarshalError — forgot to pass pointer
	err3 := json.Unmarshal([]byte(`{}`), struct{}{}) // not a pointer!
	fmt.Printf("InvalidUnmarshal: %v\n", err3)

	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 8: Performance tips
// ─────────────────────────────────────────────────────────────────────────────

func performanceTips() {
	fmt.Println("═══ SECTION 8: Performance Tips ═══")

	fmt.Println(`
PERFORMANCE TIPS FOR encoding/json:

1. REUSE ENCODER:
   Create json.NewEncoder once, call Encode() many times.
   Each call to json.Marshal allocates a new []byte.
   Encoder reuse avoids repeated encoder struct allocations.

2. AVOID interface{}:
   Marshaling/unmarshaling concrete types is faster than interface{}.
   Concrete types: compiler knows structure, type assertions not needed.
   interface{}: reflection is slower, type assertions needed after decode.

3. OMITEMPTY ON ZERO FIELDS:
   Reduces JSON payload size. Less to write, less to read, less memory.
   Add to fields that are commonly zero: pointers, slices, empty strings.

4. STRUCT TAGS COMPILATION:
   Tags are parsed at runtime via reflection. For very hot paths,
   consider codegen tools (easyjson, ffjson) that generate marshal code.

5. json.RawMessage FOR PASSTHROUGH:
   If you receive JSON and need to forward it, store as RawMessage.
   Avoids a decode-then-re-encode round trip.

6. BUFFER REUSE WITH ENCODER:
   var buf bytes.Buffer
   enc := json.NewEncoder(&buf)
   // In a loop:
   buf.Reset()
   enc.Encode(value)
   // buf.Bytes() has your JSON without re-creating the encoder.
`)
	fmt.Println()
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║      Go Standard Library: encoding/json Package       ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	basicMarshalUnmarshal()
	primitiveAndCollections()
	streamingDemo()
	customMarshalingDemo()
	rawMessageDemo()
	commonMistakes()
	errorHandlingDemo()
	performanceTips()

	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("KEY TAKEAWAYS:")
	fmt.Println("  1. Only exported fields are marshaled — silent otherwise")
	fmt.Println("  2. Use Encoder/Decoder for streaming (HTTP bodies, large files)")
	fmt.Println("  3. json.RawMessage for polymorphic payloads or passthrough")
	fmt.Println("  4. json.Number (UseNumber) for large integers via interface{}")
	fmt.Println("  5. Implement MarshalJSON/UnmarshalJSON for custom types")
	fmt.Println("  6. Always pass a pointer to Unmarshal")
	fmt.Println("  7. Check status code before decoding HTTP response body")
}
