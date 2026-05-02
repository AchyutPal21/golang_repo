// FILE: book/part5_building_backends/chapter62_validation/examples/02_form_api_validation/main.go
// CHAPTER: 62 — Validation
// TOPIC: Validation in HTTP context —
//        JSON body decoding + struct validation → 422 with per-field errors,
//        query parameter validation (pagination), path parameter validation.
//
// Error response format:
//   {"error":"validation failed","fields":[{"field":"email","message":"..."}]}
//
// Run (from the chapter folder):
//   go run ./examples/02_form_api_validation

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// VALIDATION PRIMITIVES  (same as example 01, inlined for self-containment)
// ─────────────────────────────────────────────────────────────────────────────

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Validator struct {
	errs []FieldError
}

func (v *Validator) add(field, msg string) {
	v.errs = append(v.errs, FieldError{Field: field, Message: msg})
}

func (v *Validator) Required(field, s string) {
	if strings.TrimSpace(s) == "" {
		v.add(field, "field is required")
	}
}

func (v *Validator) MinLen(field, s string, min int) {
	if len(s) < min {
		v.add(field, fmt.Sprintf("must be at least %d characters (got %d)", min, len(s)))
	}
}

func (v *Validator) MaxLen(field, s string, max int) {
	if len(s) > max {
		v.add(field, fmt.Sprintf("must be at most %d characters (got %d)", max, len(s)))
	}
}

func (v *Validator) Matches(field, s string, re *regexp.Regexp, hint string) {
	if s != "" && !re.MatchString(s) {
		v.add(field, fmt.Sprintf("must be a valid %s", hint))
	}
}

func (v *Validator) InRange(field string, n, min, max int) {
	if n < min || n > max {
		v.add(field, fmt.Sprintf("must be between %d and %d (got %d)", min, max, n))
	}
}

func (v *Validator) OneOf(field, s string, allowed ...string) {
	for _, a := range allowed {
		if s == a {
			return
		}
	}
	v.add(field, fmt.Sprintf("must be one of [%s]", strings.Join(allowed, ", ")))
}

func (v *Validator) HasErrors() bool { return len(v.errs) > 0 }
func (v *Validator) Errors() []FieldError { return v.errs }

var (
	reEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ─────────────────────────────────────────────────────────────────────────────
// HTTP HELPERS
// ─────────────────────────────────────────────────────────────────────────────

// writeValidationError writes a 422 response with per-field error details.
func writeValidationError(w http.ResponseWriter, errs []FieldError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(map[string]any{
		"error":  "validation failed",
		"fields": errs,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// ─────────────────────────────────────────────────────────────────────────────
// REQUEST TYPES
// ─────────────────────────────────────────────────────────────────────────────

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
	Role  string `json:"role"`
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

// POST /users — decode JSON body, validate, return 201 or 422.
func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}

	var v Validator
	v.Required("name", req.Name)
	v.MinLen("name", req.Name, 2)
	v.MaxLen("name", req.Name, 50)
	v.Required("email", req.Email)
	v.Matches("email", req.Email, reEmail, "email address")
	v.InRange("age", req.Age, 13, 120)
	v.Required("role", req.Role)
	v.OneOf("role", req.Role, "admin", "editor", "viewer")

	if v.HasErrors() {
		writeValidationError(w, v.Errors())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":    42,
		"name":  req.Name,
		"email": req.Email,
		"role":  req.Role,
	})
}

// GET /users — validate query params (page, per_page).
func handleListUsers(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		pageStr = "1"
	}
	perPageStr := r.URL.Query().Get("per_page")
	if perPageStr == "" {
		perPageStr = "20"
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "page must be an integer"})
		return
	}
	perPage, err := strconv.Atoi(perPageStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "per_page must be an integer"})
		return
	}

	var v Validator
	v.InRange("page", page, 1, 10000)
	v.InRange("per_page", perPage, 1, 100)

	if v.HasErrors() {
		// Query param errors → 400 Bad Request (client sent bad query string)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error":  "invalid query parameters",
			"fields": v.Errors(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"page":     page,
		"per_page": perPage,
		"users":    []string{"alice", "bob"},
	})
}

// GET /users/{id} — validate path parameter (must be a positive integer).
func handleGetUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("id must be a positive integer (got %q)", idStr),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "name": "Example User"})
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", handleCreateUser)
	mux.HandleFunc("GET /users", handleListUsers)
	mux.HandleFunc("GET /users/{id}", handleGetUser)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: mux}).Serve(ln) //nolint:errcheck

	client := &http.Client{Timeout: 3 * time.Second}

	doPost := func(path, body string) (int, string) {
		req, _ := http.NewRequest("POST", base+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return 0, ""
		}
		defer resp.Body.Close()
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		b, _ := json.MarshalIndent(out, "    ", "  ")
		return resp.StatusCode, string(b)
	}

	doGet := func(path string) (int, string) {
		req, _ := http.NewRequest("GET", base+path, nil)
		resp, err := client.Do(req)
		if err != nil {
			return 0, ""
		}
		defer resp.Body.Close()
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		b, _ := json.MarshalIndent(out, "    ", "  ")
		return resp.StatusCode, string(b)
	}

	fmt.Printf("=== HTTP Validation Tests — %s ===\n\n", base)

	// ── Body validation ───────────────────────────────────────────────────────
	fmt.Println("--- POST /users ---")

	fmt.Println()
	fmt.Println("[1] Valid payload → 201 Created")
	code, body := doPost("/users", `{"name":"Alice","email":"alice@example.com","age":30,"role":"admin"}`)
	fmt.Printf("    Status: %d\n    Body: %s\n", code, body)

	fmt.Println()
	fmt.Println("[2] Missing name and email → 422 Unprocessable Entity")
	code, body = doPost("/users", `{"age":30,"role":"admin"}`)
	fmt.Printf("    Status: %d\n    Body: %s\n", code, body)

	fmt.Println()
	fmt.Println("[3] Invalid email + invalid role → 422")
	code, body = doPost("/users", `{"name":"Bob","email":"not-email","age":25,"role":"superuser"}`)
	fmt.Printf("    Status: %d\n    Body: %s\n", code, body)

	fmt.Println()
	fmt.Println("[4] Age out of range + name too short → 422")
	code, body = doPost("/users", `{"name":"X","email":"x@e.com","age":5,"role":"viewer"}`)
	fmt.Printf("    Status: %d\n    Body: %s\n", code, body)

	// ── Query param validation ────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- GET /users (pagination) ---")

	fmt.Println()
	fmt.Println("[5] Valid pagination → 200")
	code, body = doGet("/users?page=2&per_page=50")
	fmt.Printf("    Status: %d\n    Body: %s\n", code, body)

	fmt.Println()
	fmt.Println("[6] page=0 (invalid) → 400")
	code, body = doGet("/users?page=0&per_page=50")
	fmt.Printf("    Status: %d\n    Body: %s\n", code, body)

	fmt.Println()
	fmt.Println("[7] per_page=200 (too large) → 400")
	code, body = doGet("/users?per_page=200")
	fmt.Printf("    Status: %d\n    Body: %s\n", code, body)

	// ── Path param validation ─────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- GET /users/{id} ---")

	fmt.Println()
	fmt.Println("[8] Valid id → 200")
	code, body = doGet("/users/42")
	fmt.Printf("    Status: %d\n    Body: %s\n", code, body)

	fmt.Println()
	fmt.Println("[9] Non-integer id → 400")
	code, body = doGet("/users/abc")
	fmt.Printf("    Status: %d\n    Body: %s\n", code, body)

	fmt.Println()
	fmt.Println("[10] Negative id → 400")
	code, body = doGet("/users/-5")
	fmt.Printf("    Status: %d\n    Body: %s\n", code, body)
}
