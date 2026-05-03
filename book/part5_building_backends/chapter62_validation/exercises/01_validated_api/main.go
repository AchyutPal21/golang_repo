// FILE: book/part5_building_backends/chapter62_validation/exercises/01_validated_api/main.go
// CHAPTER: 62 — Validation
// EXERCISE: Products API with full validation.
//
// Product rules:
//   Name        : 2-100 chars, required
//   Description : max 1000 chars
//   Price       : > 0 (float64)
//   Category    : one of [electronics, clothing, food, other]
//   Stock       : >= 0 (integer)
//   SKU         : matches [A-Z]{2}-[0-9]{4}
//
// Routes:
//   POST /products          — create, validates all fields
//   PUT  /products/{id}     — partial update, validates only provided fields
//   GET  /products          — list, validates category / min_price / max_price query params
//
// Run (from the chapter folder):
//   go run ./exercises/01_validated_api

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CHAINABLE VALIDATOR
// ─────────────────────────────────────────────────────────────────────────────

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Validator accumulates errors across multiple rule checks.
type Validator struct {
	errs []FieldError
}

func (v *Validator) add(field, msg string) *Validator {
	v.errs = append(v.errs, FieldError{Field: field, Message: msg})
	return v
}

// String rules — all return *Validator for chaining.

func (v *Validator) Required(field, s string) *Validator {
	if strings.TrimSpace(s) == "" {
		return v.add(field, "field is required")
	}
	return v
}

func (v *Validator) MinLen(field, s string, min int) *Validator {
	if len(s) > 0 && len(s) < min {
		return v.add(field, fmt.Sprintf("must be at least %d characters (got %d)", min, len(s)))
	}
	return v
}

func (v *Validator) MaxLen(field, s string, max int) *Validator {
	if len(s) > max {
		return v.add(field, fmt.Sprintf("must be at most %d characters (got %d)", max, len(s)))
	}
	return v
}

func (v *Validator) Matches(field, s string, re *regexp.Regexp, hint string) *Validator {
	if s != "" && !re.MatchString(s) {
		return v.add(field, fmt.Sprintf("must match pattern %s", hint))
	}
	return v
}

func (v *Validator) OneOf(field, s string, allowed ...string) *Validator {
	if s == "" {
		return v
	}
	for _, a := range allowed {
		if s == a {
			return v
		}
	}
	return v.add(field, fmt.Sprintf("must be one of [%s] (got %q)", strings.Join(allowed, ", "), s))
}

// Numeric rules.

func (v *Validator) PositiveFloat(field string, n float64) *Validator {
	if n <= 0 {
		return v.add(field, fmt.Sprintf("must be greater than 0 (got %g)", n))
	}
	return v
}

func (v *Validator) NonNegativeInt(field string, n int) *Validator {
	if n < 0 {
		return v.add(field, fmt.Sprintf("must be >= 0 (got %d)", n))
	}
	return v
}

func (v *Validator) HasErrors() bool    { return len(v.errs) > 0 }
func (v *Validator) Errors() []FieldError { return v.errs }

// ─────────────────────────────────────────────────────────────────────────────
// PATTERNS
// ─────────────────────────────────────────────────────────────────────────────

var reSKU = regexp.MustCompile(`^[A-Z]{2}-[0-9]{4}$`)

var validCategories = []string{"electronics", "clothing", "food", "other"}

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	Stock       int     `json:"stock"`
	SKU         string  `json:"sku"`
}

// ProductInput is used for both create and update (all fields optional for
// partial updates; the zero value of a pointer means "not provided").
type ProductInput struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price"`
	Category    *string  `json:"category"`
	Stock       *int     `json:"stock"`
	SKU         *string  `json:"sku"`
}

// validateCreate validates all fields for a new product.
func validateCreate(p ProductInput) []FieldError {
	var v Validator

	// Name: required, 2-100 chars
	name := ""
	if p.Name != nil {
		name = *p.Name
	}
	v.Required("name", name)
	v.MinLen("name", name, 2)
	v.MaxLen("name", name, 100)

	// Description: max 1000 chars (optional)
	if p.Description != nil {
		v.MaxLen("description", *p.Description, 1000)
	}

	// Price: required, > 0
	if p.Price == nil {
		v.add("price", "field is required")
	} else {
		v.PositiveFloat("price", *p.Price)
	}

	// Category: required, enum
	cat := ""
	if p.Category != nil {
		cat = *p.Category
	}
	v.Required("category", cat)
	v.OneOf("category", cat, validCategories...)

	// Stock: required, >= 0
	if p.Stock == nil {
		v.add("stock", "field is required")
	} else {
		v.NonNegativeInt("stock", *p.Stock)
	}

	// SKU: required, pattern
	sku := ""
	if p.SKU != nil {
		sku = *p.SKU
	}
	v.Required("sku", sku)
	v.Matches("sku", sku, reSKU, "[A-Z]{2}-[0-9]{4}")

	return v.Errors()
}

// validateUpdate validates only the fields that are provided (partial update).
func validateUpdate(p ProductInput) []FieldError {
	var v Validator

	if p.Name != nil {
		v.MinLen("name", *p.Name, 2)
		v.MaxLen("name", *p.Name, 100)
		if strings.TrimSpace(*p.Name) == "" {
			v.add("name", "name cannot be blank")
		}
	}

	if p.Description != nil {
		v.MaxLen("description", *p.Description, 1000)
	}

	if p.Price != nil {
		v.PositiveFloat("price", *p.Price)
	}

	if p.Category != nil {
		v.OneOf("category", *p.Category, validCategories...)
	}

	if p.Stock != nil {
		v.NonNegativeInt("stock", *p.Stock)
	}

	if p.SKU != nil {
		v.Matches("sku", *p.SKU, reSKU, "[A-Z]{2}-[0-9]{4}")
	}

	return v.Errors()
}

// ─────────────────────────────────────────────────────────────────────────────
// IN-MEMORY STORE
// ─────────────────────────────────────────────────────────────────────────────

type Store struct {
	mu   sync.RWMutex
	data map[int]Product
	next int
}

func newStore() *Store {
	s := &Store{data: make(map[int]Product), next: 1}
	return s
}

func (s *Store) Create(p Product) Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.next
	s.next++
	s.data[p.ID] = p
	return p
}

func (s *Store) Get(id int) (Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data[id]
	return p, ok
}

func (s *Store) Update(id int, input ProductInput) (Product, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.data[id]
	if !ok {
		return Product{}, false
	}
	if input.Name != nil {
		p.Name = *input.Name
	}
	if input.Description != nil {
		p.Description = *input.Description
	}
	if input.Price != nil {
		p.Price = *input.Price
	}
	if input.Category != nil {
		p.Category = *input.Category
	}
	if input.Stock != nil {
		p.Stock = *input.Stock
	}
	if input.SKU != nil {
		p.SKU = *input.SKU
	}
	s.data[id] = p
	return p, true
}

func (s *Store) List(category string, minPrice, maxPrice float64) []Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Product
	for _, p := range s.data {
		if category != "" && p.Category != category {
			continue
		}
		if minPrice > 0 && p.Price < minPrice {
			continue
		}
		if maxPrice > 0 && p.Price > maxPrice {
			continue
		}
		out = append(out, p)
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeValidationErrors(w http.ResponseWriter, errs []FieldError) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"error":  "validation failed",
		"fields": errs,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func handleCreateProduct(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input ProductInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if errs := validateCreate(input); len(errs) > 0 {
			writeValidationErrors(w, errs)
			return
		}
		p := Product{
			Name:        *input.Name,
			Description: strOrEmpty(input.Description),
			Price:       *input.Price,
			Category:    *input.Category,
			Stock:       *input.Stock,
			SKU:         *input.SKU,
		}
		created := store.Create(p)
		writeJSON(w, http.StatusCreated, created)
	}
}

func handleUpdateProduct(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil || id <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}
		var input ProductInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if errs := validateUpdate(input); len(errs) > 0 {
			writeValidationErrors(w, errs)
			return
		}
		updated, found := store.Update(id, input)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

func handleListProducts(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		category := q.Get("category")
		minPriceStr := q.Get("min_price")
		maxPriceStr := q.Get("max_price")

		var v Validator
		if category != "" {
			v.OneOf("category", category, validCategories...)
		}

		var minPrice, maxPrice float64
		if minPriceStr != "" {
			p, err := strconv.ParseFloat(minPriceStr, 64)
			if err != nil || p < 0 {
				v.add("min_price", fmt.Sprintf("must be a non-negative number (got %q)", minPriceStr))
			} else {
				minPrice = p
			}
		}
		if maxPriceStr != "" {
			p, err := strconv.ParseFloat(maxPriceStr, 64)
			if err != nil || p < 0 {
				v.add("max_price", fmt.Sprintf("must be a non-negative number (got %q)", maxPriceStr))
			} else {
				maxPrice = p
			}
		}

		if v.HasErrors() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": "invalid query parameters", "fields": v.Errors()})
			return
		}

		products := store.List(category, minPrice, maxPrice)
		if products == nil {
			products = []Product{}
		}
		writeJSON(w, http.StatusOK, products)
	}
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	store := newStore()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /products", handleCreateProduct(store))
	mux.HandleFunc("PUT /products/{id}", handleUpdateProduct(store))
	mux.HandleFunc("GET /products", handleListProducts(store))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: mux}).Serve(ln) //nolint:errcheck

	client := &http.Client{Timeout: 3 * time.Second}

	post := func(path, body string) (int, map[string]any) {
		req, _ := http.NewRequest("POST", base+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil
		}
		defer resp.Body.Close()
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}

	put := func(path, body string) (int, map[string]any) {
		req, _ := http.NewRequest("PUT", base+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil
		}
		defer resp.Body.Close()
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}

	get := func(path string) (int, any) {
		req, _ := http.NewRequest("GET", base+path, nil)
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil
		}
		defer resp.Body.Close()
		var out any
		json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}

	show := func(label string, code int, body any) {
		b, _ := json.MarshalIndent(body, "    ", "  ")
		fmt.Printf("  %-55s [%d]\n    %s\n\n", label, code, b)
	}

	fmt.Printf("=== Products API Validation — %s ===\n\n", base)

	fmt.Println("--- POST /products ---")
	fmt.Println()

	code, body := post("/products", `{
		"name":"Laptop Pro",
		"description":"A fine laptop",
		"price":999.99,
		"category":"electronics",
		"stock":10,
		"sku":"LP-1234"
	}`)
	show("Valid product → 201", code, body)

	code, body = post("/products", `{}`)
	show("Empty body (all required missing) → 422", code, body)

	code, body = post("/products", `{
		"name":"X",
		"price":-5,
		"category":"gadgets",
		"stock":-1,
		"sku":"bad-sku"
	}`)
	show("Multiple violations → 422", code, body)

	fmt.Println("--- PUT /products/{id} (partial update) ---")
	fmt.Println()

	// Create a product to update.
	code, created := post("/products", `{"name":"T-Shirt","price":19.99,"category":"clothing","stock":100,"sku":"TS-0001"}`)
	id := int(created["id"].(float64))
	fmt.Printf("  Created product id=%d [%d]\n\n", id, code)

	code, body = put(fmt.Sprintf("/products/%d", id), `{"price":24.99}`)
	show("Partial update (price only) → 200", code, body)

	code, body = put(fmt.Sprintf("/products/%d", id), `{"price":-10}`)
	show("Partial update (invalid price) → 422", code, body)

	code, body = put(fmt.Sprintf("/products/%d", id), `{"category":"magic"}`)
	show("Partial update (invalid category) → 422", code, body)

	fmt.Println("--- GET /products (query filters) ---")
	fmt.Println()

	// Add another product in food category.
	post("/products", `{"name":"Apple","price":0.99,"category":"food","stock":500,"sku":"AP-0001"}`) //nolint:errcheck

	var gbody any
	var gcode int
	gcode, gbody = get("/products")
	show("List all → 200", gcode, gbody)

	gcode, gbody = get("/products?category=clothing&min_price=5&max_price=100")
	show("Filter clothing 5-100 → 200", gcode, gbody)

	gcode, gbody = get("/products?category=weapons")
	show("Invalid category → 400", gcode, gbody)

	gcode, gbody = get("/products?min_price=abc")
	show("Invalid min_price → 400", gcode, gbody)
}
