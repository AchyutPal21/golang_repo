// FILE: book/part5_building_backends/chapter64_api_error_handling/exercises/01_error_catalog/main.go
// CHAPTER: 64 — API Error Handling
// EXERCISE: Build a complete error catalog for an Orders API:
//   - All errors are RFC 7807 Problem Details (application/problem+json)
//   - Error catalog: NOT_FOUND, VALIDATION_ERROR, CONFLICT, UNAUTHORIZED,
//     FORBIDDEN, PAYMENT_FAILED, INVENTORY_INSUFFICIENT, RATE_LIMITED, INTERNAL
//   - Handlers return error (using the error-middleware pattern)
//   - Correlation ID is included in every error response
//   - 5xx errors log cause internally but return generic message to client
//   - Test all error types: 400, 401, 403, 404, 409, 422, 429, 500, 503
//
// Run (from the chapter folder):
//   go run ./exercises/01_error_catalog

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// RFC 7807
// ─────────────────────────────────────────────────────────────────────────────

type Problem struct {
	Type          string `json:"type"`
	Title         string `json:"title"`
	Status        int    `json:"status"`
	Detail        string `json:"detail,omitempty"`
	Instance      string `json:"instance,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

const typeBase = "https://orders.example.com/errors/"

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT KEY
// ─────────────────────────────────────────────────────────────────────────────

type ctxKey int

const keyCorrID ctxKey = iota

func corrID(r *http.Request) string {
	v, _ := r.Context().Value(keyCorrID).(string)
	return v
}

func withCorrID(r *http.Request, id string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), keyCorrID, id))
}

// ─────────────────────────────────────────────────────────────────────────────
// TYPED ERRORS
// ─────────────────────────────────────────────────────────────────────────────

type APIError struct {
	Status  int
	ErrType string
	Title   string
	Detail  string
	Cause   error // internal, not exposed
}

func (e *APIError) Error() string { return e.Detail }
func (e *APIError) Unwrap() error { return e.Cause }

func errNotFound(resource, id string) *APIError {
	return &APIError{Status: 404, ErrType: "not-found", Title: "Not Found",
		Detail: fmt.Sprintf("%s '%s' not found", resource, id)}
}

func errValidation(detail string) *APIError {
	return &APIError{Status: 422, ErrType: "validation-error", Title: "Validation Error", Detail: detail}
}

func errConflict(detail string) *APIError {
	return &APIError{Status: 409, ErrType: "conflict", Title: "Conflict", Detail: detail}
}

func errUnauthorized() *APIError {
	return &APIError{Status: 401, ErrType: "unauthorized", Title: "Unauthorized", Detail: "authentication required"}
}

func errForbidden(detail string) *APIError {
	return &APIError{Status: 403, ErrType: "forbidden", Title: "Forbidden", Detail: detail}
}

func errPaymentFailed(detail string) *APIError {
	return &APIError{Status: 402, ErrType: "payment-failed", Title: "Payment Failed", Detail: detail}
}

func errInventory(item string, needed, available int) *APIError {
	return &APIError{Status: 409, ErrType: "inventory-insufficient", Title: "Insufficient Inventory",
		Detail: fmt.Sprintf("item %s: need %d, available %d", item, needed, available)}
}

func errRateLimit(retryAfter int) *APIError {
	return &APIError{Status: 429, ErrType: "rate-limited", Title: "Too Many Requests",
		Detail: fmt.Sprintf("rate limit exceeded, retry after %d seconds", retryAfter)}
}

func errInternal(cause error) *APIError {
	return &APIError{Status: 500, ErrType: "internal-error", Title: "Internal Server Error",
		Detail: "an unexpected error occurred", Cause: cause}
}

func errServiceUnavailable(cause error) *APIError {
	return &APIError{Status: 503, ErrType: "service-unavailable", Title: "Service Unavailable",
		Detail: "downstream service is temporarily unavailable", Cause: cause}
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLER ADAPTER
// ─────────────────────────────────────────────────────────────────────────────

type Handler func(w http.ResponseWriter, r *http.Request) error

func Adapt(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			handleErr(w, r, err)
		}
	}
}

func handleErr(w http.ResponseWriter, r *http.Request, err error) {
	var ae *APIError
	if !errors.As(err, &ae) {
		ae = errInternal(err)
	}
	if ae.Status >= 500 && ae.Cause != nil {
		fmt.Printf("  [ERROR] cid=%s status=%d cause=%v\n", corrID(r), ae.Status, ae.Cause)
	}
	p := Problem{
		Type:          typeBase + ae.ErrType,
		Title:         ae.Title,
		Status:        ae.Status,
		Detail:        ae.Detail,
		Instance:      r.URL.Path,
		CorrelationID: corrID(r),
	}
	w.Header().Set("Content-Type", "application/problem+json")
	if ae.ErrType == "rate-limited" {
		w.Header().Set("Retry-After", "5")
	}
	w.WriteHeader(ae.Status)
	json.NewEncoder(w).Encode(p)
}

// ─────────────────────────────────────────────────────────────────────────────
// CORRELATION ID MIDDLEWARE
// ─────────────────────────────────────────────────────────────────────────────

func corrIDMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Correlation-ID")
		if id == "" {
			id = fmt.Sprintf("cid-%x", rand.Int63())
		}
		w.Header().Set("X-Correlation-ID", id)
		next.ServeHTTP(w, withCorrID(r, id))
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type Order struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Item       string `json:"item"`
	Quantity   int    `json:"quantity"`
	Status     string `json:"status"`
}

var orders = map[string]*Order{
	"ord-1": {ID: "ord-1", CustomerID: "cust-1", Item: "widget", Quantity: 2, Status: "pending"},
	"ord-2": {ID: "ord-2", CustomerID: "cust-2", Item: "gadget", Quantity: 1, Status: "shipped"},
}

var inventory = map[string]int{
	"widget": 5,
	"gadget": 0,
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func getOrder(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	o, ok := orders[id]
	if !ok {
		return errNotFound("order", id)
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(o)
}

func createOrder(w http.ResponseWriter, r *http.Request) error {
	if r.Header.Get("Authorization") == "" {
		return errUnauthorized()
	}
	var o Order
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		return &APIError{Status: 400, ErrType: "bad-request", Title: "Bad Request",
			Detail: "invalid JSON: " + err.Error()}
	}
	if strings.TrimSpace(o.Item) == "" {
		return errValidation("item is required")
	}
	if o.Quantity <= 0 {
		return errValidation("quantity must be positive")
	}
	// Inventory check.
	avail, exists := inventory[o.Item]
	if !exists {
		return errNotFound("item", o.Item)
	}
	if avail < o.Quantity {
		return errInventory(o.Item, o.Quantity, avail)
	}
	// Duplicate order check.
	for _, existing := range orders {
		if existing.CustomerID == o.CustomerID && existing.Item == o.Item && existing.Status == "pending" {
			return errConflict(fmt.Sprintf("pending order for item %s already exists for customer %s", o.Item, o.CustomerID))
		}
	}
	o.ID = fmt.Sprintf("ord-%d", len(orders)+1)
	o.Status = "pending"
	orders[o.ID] = &o
	w.Header().Set("Location", "/orders/"+o.ID)
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(o)
}

func cancelOrder(w http.ResponseWriter, r *http.Request) error {
	if r.Header.Get("Authorization") == "" {
		return errUnauthorized()
	}
	id := r.PathValue("id")
	o, ok := orders[id]
	if !ok {
		return errNotFound("order", id)
	}
	if o.Status == "shipped" {
		return errForbidden("cannot cancel a shipped order")
	}
	o.Status = "cancelled"
	return json.NewEncoder(w).Encode(o)
}

func processPayment(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if _, ok := orders[id]; !ok {
		return errNotFound("order", id)
	}
	// Simulate payment gateway failure.
	if id == "ord-1" {
		return errPaymentFailed("card ending in 4242 was declined")
	}
	return errInternal(fmt.Errorf("payment gateway timeout after 30s"))
}

func callDownstream(w http.ResponseWriter, r *http.Request) error {
	return errServiceUnavailable(fmt.Errorf("inventory service returned 503"))
}

var requestCount int

func rateLimitedEndpoint(w http.ResponseWriter, r *http.Request) error {
	requestCount++
	if requestCount > 3 {
		requestCount = 0
		return errRateLimit(5)
	}
	return json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /orders/{id}", Adapt(getOrder))
	mux.HandleFunc("POST /orders", Adapt(createOrder))
	mux.HandleFunc("DELETE /orders/{id}", Adapt(cancelOrder))
	mux.HandleFunc("POST /orders/{id}/pay", Adapt(processPayment))
	mux.HandleFunc("GET /downstream", Adapt(callDownstream))
	mux.HandleFunc("GET /limited", Adapt(rateLimitedEndpoint))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: corrIDMW(mux)}).Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}

	do := func(method, path, body, auth string) (int, map[string]any, http.Header) {
		var br *strings.Reader
		if body != "" {
			br = strings.NewReader(body)
		} else {
			br = strings.NewReader("")
		}
		req, _ := http.NewRequest(method, base+path, br)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		req.Header.Set("X-Correlation-ID", "test-cid-abc")
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil, nil
		}
		defer resp.Body.Close()
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out, resp.Header
	}

	check := func(label string, code, want int) {
		mark := "✓"
		if code != want {
			mark = "✗"
		}
		fmt.Printf("  %s %-58s %d\n", mark, label, code)
	}

	fmt.Printf("=== Error Catalog — %s ===\n\n", base)

	fmt.Println("--- Full error catalog ---")
	code, body, _ := do("GET", "/orders/ord-99", "", "")
	check("404 Not Found", code, 404)
	fmt.Printf("    type=%v cid=%v\n", body["type"], body["correlation_id"])

	code, body, _ = do("POST", "/orders", `{"item":"","quantity":0}`, "Bearer tok")
	check("422 Validation Error", code, 422)
	fmt.Printf("    detail=%v\n", body["detail"])

	code, body, _ = do("POST", "/orders", `{"customer_id":"cust-1","item":"widget","quantity":2}`, "Bearer tok")
	check("409 Conflict (duplicate order)", code, 409)
	fmt.Printf("    type=%v\n", body["type"])

	code, body, _ = do("POST", "/orders", `{"customer_id":"cust-3","item":"gadget","quantity":5}`, "Bearer tok")
	check("409 Inventory Insufficient", code, 409)
	fmt.Printf("    type=%v\n", body["type"])

	code, body, _ = do("POST", "/orders", `{"customer_id":"cust-1","item":"widget","quantity":1}`, "")
	check("401 Unauthorized", code, 401)

	code, body, _ = do("DELETE", "/orders/ord-2", "", "Bearer tok")
	check("403 Forbidden (shipped order)", code, 403)
	fmt.Printf("    detail=%v\n", body["detail"])

	code, body, _ = do("POST", "/orders/ord-1/pay", "", "")
	check("402 Payment Failed", code, 402)
	fmt.Printf("    type=%v\n    detail=%v\n", body["type"], body["detail"])

	code, body, _ = do("GET", "/downstream", "", "")
	check("503 Service Unavailable", code, 503)

	// Rate limiting — trigger after 3 calls.
	for i := 0; i < 3; i++ {
		do("GET", "/limited", "", "")
	}
	code, body, h := do("GET", "/limited", "", "")
	check("429 Rate Limited", code, 429)
	fmt.Printf("    Retry-After: %s\n", h.Get("Retry-After"))

	fmt.Println()
	fmt.Println("--- Internal error (cause logged, generic message to client) ---")
	// ord-2 is not ord-1 so it falls through to the internal error path.
	code, body, _ = do("POST", "/orders/ord-2/pay", "", "")
	check("500 Internal Error", code, 500)
	fmt.Printf("    client sees: %v\n", body["detail"])

	fmt.Println()
	fmt.Println("--- Correlation ID in all error responses ---")
	code, body, _ = do("GET", "/orders/ord-xyz", "", "")
	fmt.Printf("  GET /orders/ord-xyz → %d  correlation_id=%v\n", code, body["correlation_id"])
}
