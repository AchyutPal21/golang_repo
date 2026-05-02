// FILE: book/part3_designing_software/chapter39_encoding/examples/02_custom_encoding/main.go
// CHAPTER: 39 — Encoding
// TOPIC: json.Marshaler / json.Unmarshaler, custom types, encoding/csv,
//        encoding/binary, and base64.
//
// Run (from the chapter folder):
//   go run ./examples/02_custom_encoding

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CUSTOM json.Marshaler / json.Unmarshaler
// ─────────────────────────────────────────────────────────────────────────────

// Money encodes as a string "USD 12.34" rather than separate fields.
type Money struct {
	Cents    int64
	Currency string
}

func (m Money) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf("%s %.2f", m.Currency, float64(m.Cents)/100)
	return json.Marshal(s) // delegate to string marshaling
}

func (m *Money) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	var currency string
	var amount float64
	if _, err := fmt.Sscanf(s, "%s %f", &currency, &amount); err != nil {
		return fmt.Errorf("Money.UnmarshalJSON: invalid format %q", s)
	}
	m.Currency = currency
	m.Cents = int64(amount * 100)
	return nil
}

// Duration wraps time.Duration with a human-readable JSON form ("1h30m").
type Duration struct{ time.Duration }

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("Duration.UnmarshalJSON: %w", err)
	}
	d.Duration = dur
	return nil
}

type Config struct {
	Name    string   `json:"name"`
	Timeout Duration `json:"timeout"`
	Budget  Money    `json:"budget"`
}

func demoCustomJSON() {
	fmt.Println("=== Custom json.Marshaler / Unmarshaler ===")

	cfg := Config{
		Name:    "payment-service",
		Timeout: Duration{30 * time.Second},
		Budget:  Money{Cents: 50000, Currency: "USD"},
	}

	data, _ := json.MarshalIndent(cfg, "  ", "  ")
	fmt.Println("  marshalled:")
	fmt.Println(string(data))

	raw := `{"name":"billing","timeout":"2m30s","budget":"EUR 99.99"}`
	var cfg2 Config
	_ = json.Unmarshal([]byte(raw), &cfg2)
	fmt.Printf("  unmarshalled: name=%s timeout=%s budget=%s %.2f\n",
		cfg2.Name, cfg2.Timeout, cfg2.Budget.Currency, float64(cfg2.Budget.Cents)/100)
}

// ─────────────────────────────────────────────────────────────────────────────
// encoding/csv
// ─────────────────────────────────────────────────────────────────────────────

type Product struct {
	SKU   string
	Name  string
	Price float64
	Stock int
}

func demoCSV() {
	fmt.Println()
	fmt.Println("=== encoding/csv ===")

	products := []Product{
		{"WIDGET-A", "Widget Alpha", 9.99, 100},
		{"GADGET-B", "Gadget Beta", 49.99, 25},
		{"TOOL-C", "Tool Charlie", 19.99, 200},
	}

	// Write CSV.
	var buf strings.Builder
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"sku", "name", "price", "stock"}) // header
	for _, p := range products {
		_ = w.Write([]string{
			p.SKU, p.Name,
			fmt.Sprintf("%.2f", p.Price),
			fmt.Sprintf("%d", p.Stock),
		})
	}
	w.Flush()
	fmt.Println("  CSV output:")
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		fmt.Println("   ", line)
	}

	// Read CSV.
	fmt.Println("  Parsing back:")
	r := csv.NewReader(strings.NewReader(buf.String()))
	records, _ := r.ReadAll()
	for i, rec := range records {
		if i == 0 {
			fmt.Printf("  header: %v\n", rec)
			continue
		}
		fmt.Printf("  row %d: sku=%s name=%s price=%s stock=%s\n",
			i, rec[0], rec[1], rec[2], rec[3])
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// encoding/binary — fixed-size binary encoding
// ─────────────────────────────────────────────────────────────────────────────

type PacketHeader struct {
	Version  uint8
	Type     uint8
	Length   uint16
	Sequence uint32
}

func demoEncoding() {
	fmt.Println()
	fmt.Println("=== encoding/binary ===")

	hdr := PacketHeader{Version: 1, Type: 3, Length: 256, Sequence: 42}

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, hdr); err != nil {
		fmt.Println("  write error:", err)
		return
	}
	fmt.Printf("  encoded: %d bytes  hex: %X\n", buf.Len(), buf.Bytes())

	var hdr2 PacketHeader
	if err := binary.Read(&buf, binary.BigEndian, &hdr2); err != nil {
		fmt.Println("  read error:", err)
		return
	}
	fmt.Printf("  decoded: version=%d type=%d length=%d seq=%d\n",
		hdr2.Version, hdr2.Type, hdr2.Length, hdr2.Sequence)
}

// ─────────────────────────────────────────────────────────────────────────────
// encoding/base64
// ─────────────────────────────────────────────────────────────────────────────

func demoBase64() {
	fmt.Println()
	fmt.Println("=== encoding/base64 ===")

	payload := []byte(`{"user":"alice","role":"admin","exp":1737849600}`)

	// Standard (padded).
	encoded := base64.StdEncoding.EncodeToString(payload)
	fmt.Printf("  standard:   %s\n", encoded)

	decoded, _ := base64.StdEncoding.DecodeString(encoded)
	fmt.Printf("  decoded:    %s\n", decoded)

	// URL-safe (no + / padding) — for JWTs, query params.
	urlEncoded := base64.RawURLEncoding.EncodeToString(payload)
	fmt.Printf("  url-safe:   %s\n", urlEncoded)

	// Basic auth header simulation.
	creds := base64.StdEncoding.EncodeToString([]byte("alice:s3cr3t"))
	fmt.Printf("  basic auth: Authorization: Basic %s\n", creds)
}

func main() {
	demoCustomJSON()
	demoCSV()
	demoEncoding()
	demoBase64()
}
