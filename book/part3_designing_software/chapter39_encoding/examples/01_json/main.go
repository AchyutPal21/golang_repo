// FILE: book/part3_designing_software/chapter39_encoding/examples/01_json/main.go
// CHAPTER: 39 — Encoding
// TOPIC: encoding/json — Marshal, Unmarshal, Encoder, Decoder, struct tags,
//        omitempty, custom field names, unknown fields, and streaming JSON.
//
// Run (from the chapter folder):
//   go run ./examples/01_json

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// STRUCT TAGS
// ─────────────────────────────────────────────────────────────────────────────

// User demonstrates the most common struct tag patterns.
type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Password  string    `json:"-"`              // never serialise
	Phone     string    `json:"phone,omitempty"` // omit if empty
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"` // nil → omitted
}

// Address uses snake_case in JSON.
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
	Zip     string `json:"zip,omitempty"`
}

type UserWithAddress struct {
	User
	Address Address `json:"address"`
}

// ─────────────────────────────────────────────────────────────────────────────
// MARSHAL / UNMARSHAL
// ─────────────────────────────────────────────────────────────────────────────

func demoMarshal() {
	fmt.Println("=== Marshal ===")
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	u := User{
		ID:        1,
		Email:     "alice@example.com",
		Name:      "Alice",
		Password:  "secret123", // should not appear in output
		CreatedAt: now,
	}

	// Compact.
	compact, _ := json.Marshal(u)
	fmt.Println("  compact:", string(compact))

	// Indented.
	indented, _ := json.MarshalIndent(u, "  ", "  ")
	fmt.Println("  indented:")
	fmt.Println(string(indented))
}

func demoUnmarshal() {
	fmt.Println()
	fmt.Println("=== Unmarshal ===")

	raw := `{"id":2,"email":"bob@example.com","name":"Bob","phone":"555-1234","created_at":"2026-02-01T09:00:00Z"}`

	var u User
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		fmt.Println("  error:", err)
		return
	}
	fmt.Printf("  id=%d email=%s name=%s phone=%s\n", u.ID, u.Email, u.Name, u.Phone)
	fmt.Printf("  created_at=%s\n", u.CreatedAt.Format(time.RFC3339))
	fmt.Printf("  password=%q (empty — was not in JSON)\n", u.Password)
}

// ─────────────────────────────────────────────────────────────────────────────
// STREAMING ENCODER / DECODER
//
// For large payloads or NDJSON (newline-delimited JSON), use Encoder/Decoder
// instead of Marshal/Unmarshal — avoids loading the entire payload into memory.
// ─────────────────────────────────────────────────────────────────────────────

func demoStreaming() {
	fmt.Println()
	fmt.Println("=== Streaming encoder (NDJSON) ===")

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // don't escape < > &

	events := []map[string]any{
		{"type": "login", "user": "alice", "ip": "10.0.0.1"},
		{"type": "view", "user": "alice", "path": "/dashboard"},
		{"type": "logout", "user": "alice"},
	}
	for _, e := range events {
		_ = enc.Encode(e) // Encode appends a newline
	}

	fmt.Print("  encoded NDJSON:\n")
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		fmt.Println("   ", line)
	}

	fmt.Println()
	fmt.Println("=== Streaming decoder (NDJSON) ===")
	dec := json.NewDecoder(&buf)
	for dec.More() {
		var event map[string]any
		if err := dec.Decode(&event); err != nil {
			fmt.Println("  decode error:", err)
			break
		}
		fmt.Printf("  event: type=%s user=%s\n", event["type"], event["user"])
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// UNKNOWN / EXTRA FIELDS
// ─────────────────────────────────────────────────────────────────────────────

type Strict struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func demoUnknownFields() {
	fmt.Println()
	fmt.Println("=== Unknown fields ===")

	raw := `{"name":"Carol","email":"carol@example.com","extra_field":"ignored","another":42}`

	// Default: unknown fields are silently ignored.
	var s Strict
	_ = json.Unmarshal([]byte(raw), &s)
	fmt.Printf("  default (ignore unknown): name=%s email=%s\n", s.Name, s.Email)

	// Strict: DisallowUnknownFields returns an error on unknown fields.
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	var s2 Strict
	err := dec.Decode(&s2)
	fmt.Printf("  strict (DisallowUnknownFields): err=%v\n", err)
}

// ─────────────────────────────────────────────────────────────────────────────
// GENERIC JSON — map[string]any and []any
// ─────────────────────────────────────────────────────────────────────────────

func demoGeneric() {
	fmt.Println()
	fmt.Println("=== Generic JSON (map[string]any) ===")

	raw := `{"status":"ok","data":{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}],"total":2}}`

	var result map[string]any
	_ = json.Unmarshal([]byte(raw), &result)

	status := result["status"].(string)
	data := result["data"].(map[string]any)
	users := data["users"].([]any)
	total := data["total"].(float64) // JSON numbers decode to float64

	fmt.Printf("  status=%s  total=%.0f\n", status, total)
	for _, u := range users {
		um := u.(map[string]any)
		fmt.Printf("  user id=%.0f name=%s\n", um["id"].(float64), um["name"].(string))
	}
}

func main() {
	demoMarshal()
	demoUnmarshal()
	demoStreaming()
	demoUnknownFields()
	demoGeneric()
}
