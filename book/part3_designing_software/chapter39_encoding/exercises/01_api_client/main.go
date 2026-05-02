// FILE: book/part3_designing_software/chapter39_encoding/exercises/01_api_client/main.go
// CHAPTER: 39 — Encoding
// EXERCISE: Simulate an API client that marshals requests, unmarshals responses,
//           exports a report as CSV, and encodes auth credentials in base64.
//
// Run (from the chapter folder):
//   go run ./exercises/01_api_client

package main

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusPaid      OrderStatus = "paid"
	StatusShipped   OrderStatus = "shipped"
	StatusDelivered OrderStatus = "delivered"
)

// Timestamp wraps time.Time with a compact date-only JSON form ("2006-01-02").
type Timestamp struct{ time.Time }

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time.Format("2006-01-02"))
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("Timestamp.UnmarshalJSON: %w", err)
	}
	t.Time = parsed
	return nil
}

// Cents encodes as a decimal string "12.34" rather than an integer.
type Cents int64

func (c Cents) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%.2f", float64(c)/100))
}

func (c *Cents) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	var v float64
	if _, err := fmt.Sscanf(s, "%f", &v); err != nil {
		return fmt.Errorf("Cents.UnmarshalJSON: invalid %q", s)
	}
	*c = Cents(v * 100)
	return nil
}

type LineItem struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Quantity int    `json:"qty"`
	Price    Cents  `json:"price"`
}

type Order struct {
	ID         string      `json:"id"`
	CustomerID string      `json:"customer_id"`
	Status     OrderStatus `json:"status"`
	PlacedAt   Timestamp   `json:"placed_at"`
	Items      []LineItem  `json:"items"`
	Total      Cents       `json:"total"`
}

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED API CLIENT
// ─────────────────────────────────────────────────────────────────────────────

// simulateRequest encodes req to JSON (as if POSTing), "receives" a response,
// and decodes it into resp.
func simulateRequest(req, resp any) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	// In real code this would be an http.Post. Here we echo the request back
	// as the response (server echoes it unchanged).
	return json.Unmarshal(data, resp)
}

// ─────────────────────────────────────────────────────────────────────────────
// CSV EXPORT
// ─────────────────────────────────────────────────────────────────────────────

func exportOrdersCSV(orders []Order) (string, error) {
	var sb strings.Builder
	w := csv.NewWriter(&sb)

	_ = w.Write([]string{"order_id", "customer_id", "status", "placed_at", "total_usd"})
	for _, o := range orders {
		_ = w.Write([]string{
			o.ID,
			o.CustomerID,
			string(o.Status),
			o.PlacedAt.Format("2006-01-02"),
			fmt.Sprintf("%.2f", float64(o.Total)/100),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// AUTH HEADER
// ─────────────────────────────────────────────────────────────────────────────

func basicAuthHeader(user, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// 1. Build an order and round-trip it through JSON.
	fmt.Println("=== JSON round-trip ===")

	order := Order{
		ID:         "ORD-001",
		CustomerID: "CUST-42",
		Status:     StatusPaid,
		PlacedAt:   Timestamp{time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
		Items: []LineItem{
			{SKU: "WIDGET-A", Name: "Widget Alpha", Quantity: 2, Price: 999},
			{SKU: "GADGET-B", Name: "Gadget Beta", Quantity: 1, Price: 4999},
		},
		Total: 6997,
	}

	pretty, _ := json.MarshalIndent(order, "", "  ")
	fmt.Println("marshalled:")
	fmt.Println(string(pretty))

	var echo Order
	if err := simulateRequest(order, &echo); err != nil {
		fmt.Println("round-trip error:", err)
		return
	}
	fmt.Printf("round-trip: id=%s status=%s placed=%s total=%s\n\n",
		echo.ID, echo.Status, echo.PlacedAt.Format("2006-01-02"),
		fmt.Sprintf("%.2f", float64(echo.Total)/100))

	// 2. Stream-decode a batch of orders with json.Decoder.
	fmt.Println("=== Streaming decoder ===")

	ndjson := `{"id":"ORD-002","customer_id":"CUST-7","status":"shipped","placed_at":"2026-04-20","items":[],"total":"129.99"}
{"id":"ORD-003","customer_id":"CUST-99","status":"delivered","placed_at":"2026-04-25","items":[],"total":"49.50"}
{"id":"ORD-004","customer_id":"CUST-7","status":"pending","placed_at":"2026-05-01","items":[],"total":"19.99"}`

	var orders []Order
	dec := json.NewDecoder(strings.NewReader(ndjson))
	for dec.More() {
		var o Order
		if err := dec.Decode(&o); err != nil {
			fmt.Println("decode error:", err)
			break
		}
		orders = append(orders, o)
		fmt.Printf("  decoded: id=%s status=%-9s placed=%s total=%s\n",
			o.ID, o.Status, o.PlacedAt.Format("2006-01-02"),
			fmt.Sprintf("%.2f", float64(o.Total)/100))
	}

	// 3. Export to CSV.
	fmt.Println()
	fmt.Println("=== CSV export ===")
	csvOut, err := exportOrdersCSV(append([]Order{echo}, orders...))
	if err != nil {
		fmt.Println("csv error:", err)
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(csvOut), "\n") {
		fmt.Println(" ", line)
	}

	// 4. Auth header.
	fmt.Println()
	fmt.Println("=== Base64 auth ===")
	fmt.Printf("  Authorization: %s\n", basicAuthHeader("service-account", "tok3n"))

	decoded, _ := base64.StdEncoding.DecodeString(
		strings.TrimPrefix(basicAuthHeader("service-account", "tok3n"), "Basic "),
	)
	fmt.Printf("  decoded creds: %s\n", decoded)
}
