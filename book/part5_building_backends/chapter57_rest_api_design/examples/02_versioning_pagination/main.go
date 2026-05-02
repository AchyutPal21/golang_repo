// FILE: book/part5_building_backends/chapter57_rest_api_design/examples/02_versioning_pagination/main.go
// CHAPTER: 57 — REST API Design
// TOPIC: API versioning strategies (URL-prefix, Accept-header) and
//        cursor-based + offset-based pagination.
//
// Run (from the chapter folder):
//   go run ./examples/02_versioning_pagination

package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type Product struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price_cents"` // stored in cents
}

// ─────────────────────────────────────────────────────────────────────────────
// SEEDED DATA (100 products)
// ─────────────────────────────────────────────────────────────────────────────

var products []Product

func init() {
	names := []string{
		"Widget", "Gadget", "Doohickey", "Thingamajig", "Whatsit",
		"Gizmo", "Contraption", "Device", "Apparatus", "Mechanism",
	}
	rng := rand.New(rand.NewSource(42))
	for i := 1; i <= 100; i++ {
		products = append(products, Product{
			ID:    i,
			Name:  fmt.Sprintf("%s %d", names[i%len(names)], i),
			Price: rng.Intn(9900) + 100, // 100–9999 cents
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func queryInt(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return def
	}
	return v
}

// ─────────────────────────────────────────────────────────────────────────────
// VERSIONING STRATEGY 1 — URL PREFIX
//
// /v1/products  →  price in cents (original contract)
// /v2/products  →  price as a float dollars string (new contract)
//
// URL versioning is simple to route and cache. The downside is that it puts
// version information in the resource identifier, which purists argue breaks
// REST (the URI should identify the resource, not its representation).
// In practice, URL versioning is the most widely used approach.
// ─────────────────────────────────────────────────────────────────────────────

type ProductV1 struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	PriceCents int    `json:"price_cents"`
}

type ProductV2 struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	PriceDollars string `json:"price_dollars"` // "12.34"
}

func toV2(p Product) ProductV2 {
	dollars := float64(p.Price) / 100.0
	return ProductV2{ID: p.ID, Name: p.Name, PriceDollars: fmt.Sprintf("%.2f", dollars)}
}

// ─────────────────────────────────────────────────────────────────────────────
// VERSIONING STRATEGY 2 — ACCEPT HEADER (content negotiation)
//
// Accept: application/vnd.shop.v1+json  →  cents
// Accept: application/vnd.shop.v2+json  →  dollars
// (default, no Accept header)           →  v1
//
// Header versioning keeps the URI stable but makes routing harder
// (proxies and CDNs cache on URL by default, not headers).
// ─────────────────────────────────────────────────────────────────────────────

func detectVersion(r *http.Request) string {
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "vnd.shop.v2") {
		return "v2"
	}
	return "v1"
}

// ─────────────────────────────────────────────────────────────────────────────
// PAGINATION STRATEGY 1 — OFFSET / LIMIT
//
// GET /v1/products?offset=0&limit=10
//
// Simple to implement and understand, supports random-access jumping.
// Problem: if items are inserted between two pages, you get duplicate
// or skipped rows ("page drift"). Expensive on large tables (DB must
// scan and discard offset rows).
// ─────────────────────────────────────────────────────────────────────────────

type OffsetPage struct {
	Data   []ProductV1 `json:"data"`
	Total  int         `json:"total"`
	Offset int         `json:"offset"`
	Limit  int         `json:"limit"`
	Next   string      `json:"next,omitempty"`
	Prev   string      `json:"prev,omitempty"`
}

func listOffsetV1(w http.ResponseWriter, r *http.Request) {
	offset := queryInt(r, "offset", 0)
	limit := queryInt(r, "limit", 10)
	if limit > 50 {
		limit = 50
	}
	if offset >= len(products) {
		writeJSON(w, http.StatusOK, OffsetPage{
			Data: []ProductV1{}, Total: len(products), Offset: offset, Limit: limit,
		})
		return
	}
	end := offset + limit
	if end > len(products) {
		end = len(products)
	}
	slice := products[offset:end]
	out := make([]ProductV1, len(slice))
	for i, p := range slice {
		out[i] = ProductV1{ID: p.ID, Name: p.Name, PriceCents: p.Price}
	}
	page := OffsetPage{
		Data: out, Total: len(products), Offset: offset, Limit: limit,
	}
	base := r.URL.Path
	if end < len(products) {
		page.Next = fmt.Sprintf("%s?offset=%d&limit=%d", base, end, limit)
	}
	if offset > 0 {
		prev := offset - limit
		if prev < 0 {
			prev = 0
		}
		page.Prev = fmt.Sprintf("%s?offset=%d&limit=%d", base, prev, limit)
	}
	writeJSON(w, http.StatusOK, page)
}

// ─────────────────────────────────────────────────────────────────────────────
// PAGINATION STRATEGY 2 — CURSOR-BASED
//
// GET /v2/products?after=42&limit=10
//
// The cursor is the last seen ID. The server returns items with ID > cursor.
// No page drift — inserts don't affect what the client has already seen.
// Cannot jump to an arbitrary page. Ideal for feeds and real-time streams.
// ─────────────────────────────────────────────────────────────────────────────

type CursorPage struct {
	Data       []ProductV2 `json:"data"`
	NextCursor int         `json:"next_cursor,omitempty"` // 0 means last page
	HasMore    bool        `json:"has_more"`
}

func listCursorV2(w http.ResponseWriter, r *http.Request) {
	after := queryInt(r, "after", 0)
	limit := queryInt(r, "limit", 10)
	if limit > 50 {
		limit = 50
	}

	var result []ProductV2
	for _, p := range products {
		if p.ID > after {
			result = append(result, toV2(p))
			if len(result) == limit+1 { // fetch one extra to detect hasMore
				break
			}
		}
	}

	hasMore := len(result) > limit
	if hasMore {
		result = result[:limit]
	}

	page := CursorPage{Data: result, HasMore: hasMore}
	if hasMore && len(result) > 0 {
		page.NextCursor = result[len(result)-1].ID
	}
	if page.Data == nil {
		page.Data = []ProductV2{}
	}
	writeJSON(w, http.StatusOK, page)
}

// ─────────────────────────────────────────────────────────────────────────────
// ACCEPT-HEADER VERSIONED ENDPOINT
// ─────────────────────────────────────────────────────────────────────────────

func listHeaderVersioned(w http.ResponseWriter, r *http.Request) {
	ver := detectVersion(r)
	limit := queryInt(r, "limit", 5)
	if limit > 10 {
		limit = 10
	}
	slice := products[:limit]
	if ver == "v2" {
		out := make([]ProductV2, len(slice))
		for i, p := range slice {
			out[i] = toV2(p)
		}
		w.Header().Set("Content-Type", "application/vnd.shop.v2+json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(out)
		return
	}
	out := make([]ProductV1, len(slice))
	for i, p := range slice {
		out[i] = ProductV1{ID: p.ID, Name: p.Name, PriceCents: p.Price}
	}
	w.Header().Set("Content-Type", "application/vnd.shop.v1+json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(out)
}

// ─────────────────────────────────────────────────────────────────────────────
// TEST HARNESS
// ─────────────────────────────────────────────────────────────────────────────

func run(client *http.Client, method, url, accept string) (int, string, string) {
	req, _ := http.NewRequest(method, url, nil)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err.Error()
	}
	defer resp.Body.Close()
	ct := resp.Header.Get("Content-Type")
	buf := make([]byte, 8192)
	n, _ := resp.Body.Read(buf)
	return resp.StatusCode, ct, strings.TrimSpace(string(buf[:n]))
}

func main() {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/products", listOffsetV1)
	mux.HandleFunc("/v2/products", listCursorV2)
	mux.HandleFunc("/products", listHeaderVersioned)

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}

	fmt.Printf("=== Versioning & Pagination — %s ===\n\n", base)

	check := func(label string, code, want int) {
		mark := "✓"
		if code != want {
			mark = "✗"
		}
		fmt.Printf("  %s %-50s %d\n", mark, label, code)
	}

	// ── URL versioning ────────────────────────────────────────────────────────
	fmt.Println("--- URL-prefix versioning ---")

	code, _, body := run(client, "GET", base+"/v1/products?offset=0&limit=3", "")
	check("GET /v1/products (offset=0, limit=3)", code, 200)
	var op OffsetPage
	json.Unmarshal([]byte(body), &op)
	fmt.Printf("    total=%d offset=%d returned=%d next=%s\n",
		op.Total, op.Offset, len(op.Data), op.Next)
	if len(op.Data) > 0 {
		fmt.Printf("    first item: id=%d price_cents=%d\n", op.Data[0].ID, op.Data[0].PriceCents)
	}

	code, _, body = run(client, "GET", base+"/v1/products?offset=95&limit=10", "")
	check("GET /v1/products (offset=95, last page)", code, 200)
	json.Unmarshal([]byte(body), &op)
	fmt.Printf("    returned=%d next=%q\n", len(op.Data), op.Next)

	fmt.Println()
	fmt.Println("--- Cursor-based pagination (v2) ---")

	code, _, body = run(client, "GET", base+"/v2/products?after=0&limit=5", "")
	check("GET /v2/products (after=0, limit=5)", code, 200)
	var cp CursorPage
	json.Unmarshal([]byte(body), &cp)
	fmt.Printf("    has_more=%v next_cursor=%d returned=%d\n",
		cp.HasMore, cp.NextCursor, len(cp.Data))
	if len(cp.Data) > 0 {
		fmt.Printf("    first item: id=%d price_dollars=%s\n", cp.Data[0].ID, cp.Data[0].PriceDollars)
	}

	// Follow the cursor to page 2.
	url2 := fmt.Sprintf("%s/v2/products?after=%d&limit=5", base, cp.NextCursor)
	code, _, body = run(client, "GET", url2, "")
	check("GET /v2/products (page 2 via cursor)", code, 200)
	json.Unmarshal([]byte(body), &cp)
	fmt.Printf("    has_more=%v next_cursor=%d returned=%d\n",
		cp.HasMore, cp.NextCursor, len(cp.Data))

	// Walk to the last page.
	cursor := cp.NextCursor
	pages := 2
	for cp.HasMore {
		u := fmt.Sprintf("%s/v2/products?after=%d&limit=10", base, cursor)
		_, _, b := run(client, "GET", u, "")
		json.Unmarshal([]byte(b), &cp)
		cursor = cp.NextCursor
		pages++
	}
	fmt.Printf("    walked all pages: %d total pages to exhaust 100 items\n", pages)

	// ── Accept-header versioning ──────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Accept-header versioning ---")

	code, ct, body := run(client, "GET", base+"/products?limit=2", "application/vnd.shop.v1+json")
	check("GET /products (Accept: v1)", code, 200)
	fmt.Printf("    Content-Type: %s\n", ct)
	var v1items []ProductV1
	json.Unmarshal([]byte(body), &v1items)
	if len(v1items) > 0 {
		fmt.Printf("    price_cents=%d\n", v1items[0].PriceCents)
	}

	code, ct, body = run(client, "GET", base+"/products?limit=2", "application/vnd.shop.v2+json")
	check("GET /products (Accept: v2)", code, 200)
	fmt.Printf("    Content-Type: %s\n", ct)
	var v2items []ProductV2
	json.Unmarshal([]byte(body), &v2items)
	if len(v2items) > 0 {
		fmt.Printf("    price_dollars=%s\n", v2items[0].PriceDollars)
	}

	code, ct, _ = run(client, "GET", base+"/products?limit=2", "")
	check("GET /products (no Accept → v1 default)", code, 200)
	fmt.Printf("    Content-Type: %s\n", ct)
}
