// FILE: book/part5_building_backends/chapter64_api_error_handling/examples/01_rfc7807/main.go
// CHAPTER: 64 — API Error Handling
// TOPIC: RFC 7807 "Problem Details for HTTP APIs" —
//        standard error envelope, typed errors, error catalog,
//        and error wrapping with %w.
//
// RFC 7807 defines a standard JSON body for API errors:
//
//   Content-Type: application/problem+json
//   {
//     "type":     "https://api.example.com/errors/not-found",
//     "title":    "Resource Not Found",
//     "status":   404,
//     "detail":   "Article with id 42 does not exist",
//     "instance": "/articles/42"
//   }
//
// Run (from the chapter folder):
//   go run ./examples/01_rfc7807

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// RFC 7807 PROBLEM DETAIL
// ─────────────────────────────────────────────────────────────────────────────

// Problem is an RFC 7807-compliant error representation.
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	// Extension fields are embedded via any struct.
}

// ProblemWithExtensions allows adding domain-specific fields.
type ValidationProblem struct {
	Problem
	Errors []FieldError `json:"errors"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

const problemContentType = "application/problem+json"

func writeProblem(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(p.Status)
	json.NewEncoder(w).Encode(p)
}

func writeValidationProblem(w http.ResponseWriter, p ValidationProblem) {
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(p.Status)
	json.NewEncoder(w).Encode(p)
}

// ─────────────────────────────────────────────────────────────────────────────
// ERROR CATALOG
// Well-known problem types are defined as constants.
// The "type" URI is a stable identifier — clients can program against it.
// ─────────────────────────────────────────────────────────────────────────────

const typeBase = "https://api.example.com/errors/"

var (
	ProblemNotFound = func(resource, id, instance string) Problem {
		return Problem{
			Type:     typeBase + "not-found",
			Title:    "Resource Not Found",
			Status:   http.StatusNotFound,
			Detail:   fmt.Sprintf("%s with id %s does not exist", resource, id),
			Instance: instance,
		}
	}

	ProblemBadRequest = func(detail, instance string) Problem {
		return Problem{
			Type:     typeBase + "bad-request",
			Title:    "Bad Request",
			Status:   http.StatusBadRequest,
			Detail:   detail,
			Instance: instance,
		}
	}

	ProblemValidation = func(instance string, errs []FieldError) ValidationProblem {
		return ValidationProblem{
			Problem: Problem{
				Type:     typeBase + "validation-error",
				Title:    "Validation Error",
				Status:   http.StatusUnprocessableEntity,
				Detail:   "One or more fields failed validation",
				Instance: instance,
			},
			Errors: errs,
		}
	}

	ProblemConflict = func(detail, instance string) Problem {
		return Problem{
			Type:     typeBase + "conflict",
			Title:    "Conflict",
			Status:   http.StatusConflict,
			Detail:   detail,
			Instance: instance,
		}
	}

	ProblemUnauthorized = Problem{
		Type:   typeBase + "unauthorized",
		Title:  "Unauthorized",
		Status: http.StatusUnauthorized,
		Detail: "Authentication is required",
	}

	ProblemForbidden = func(detail string) Problem {
		return Problem{
			Type:   typeBase + "forbidden",
			Title:  "Forbidden",
			Status: http.StatusForbidden,
			Detail: detail,
		}
	}

	ProblemInternal = Problem{
		Type:   typeBase + "internal-error",
		Title:  "Internal Server Error",
		Status: http.StatusInternalServerError,
		Detail: "An unexpected error occurred",
	}

	ProblemRateLimit = func(retryAfter int) Problem {
		return Problem{
			Type:   typeBase + "rate-limit-exceeded",
			Title:  "Too Many Requests",
			Status: http.StatusTooManyRequests,
			Detail: fmt.Sprintf("Rate limit exceeded. Retry after %d seconds", retryAfter),
		}
	}
)

// ─────────────────────────────────────────────────────────────────────────────
// SENTINEL ERRORS — domain errors that map to HTTP problems
// ─────────────────────────────────────────────────────────────────────────────

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// DomainError wraps a sentinel with extra context.
type DomainError struct {
	Code    string
	Message string
	Err     error // wrapped sentinel
}

func (e *DomainError) Error() string { return e.Message }
func (e *DomainError) Unwrap() error { return e.Err }

func notFoundError(resource, id string) error {
	return &DomainError{Code: "NOT_FOUND", Message: fmt.Sprintf("%s %s not found", resource, id), Err: ErrNotFound}
}

func conflictError(msg string) error {
	return &DomainError{Code: "CONFLICT", Message: msg, Err: ErrConflict}
}

// toProblem converts a domain error to an RFC 7807 Problem.
func toProblem(err error, instance string) Problem {
	var de *DomainError
	if errors.As(err, &de) {
		switch {
		case errors.Is(de, ErrNotFound):
			return Problem{Type: typeBase + "not-found", Title: "Not Found", Status: 404, Detail: de.Message, Instance: instance}
		case errors.Is(de, ErrConflict):
			return Problem{Type: typeBase + "conflict", Title: "Conflict", Status: 409, Detail: de.Message, Instance: instance}
		case errors.Is(de, ErrForbidden):
			return Problem{Type: typeBase + "forbidden", Title: "Forbidden", Status: 403, Detail: de.Message, Instance: instance}
		}
	}
	return ProblemInternal
}

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN + HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

type Article struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

var db = map[string]*Article{
	"1": {ID: "1", Title: "Go Concurrency", Body: "..."},
	"2": {ID: "2", Title: "REST in Go", Body: "..."},
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, ok := db[id]
	if !ok {
		writeProblem(w, ProblemNotFound("article", id, r.URL.Path))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	var a Article
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeProblem(w, ProblemBadRequest("invalid JSON: "+err.Error(), r.URL.Path))
		return
	}

	// Validation.
	var ferrs []FieldError
	if strings.TrimSpace(a.Title) == "" {
		ferrs = append(ferrs, FieldError{Field: "title", Message: "required"})
	} else if len(a.Title) > 200 {
		ferrs = append(ferrs, FieldError{Field: "title", Message: "must be ≤ 200 characters"})
	}
	if strings.TrimSpace(a.Body) == "" {
		ferrs = append(ferrs, FieldError{Field: "body", Message: "required"})
	}
	if len(ferrs) > 0 {
		writeValidationProblem(w, ProblemValidation(r.URL.Path, ferrs))
		return
	}

	// Conflict check.
	if _, exists := db[a.ID]; exists {
		writeProblem(w, ProblemConflict(fmt.Sprintf("article with id %s already exists", a.ID), r.URL.Path))
		return
	}

	db[a.ID] = &a
	w.Header().Set("Location", fmt.Sprintf("/articles/%s", a.ID))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

// handleService demonstrates converting domain errors → problems.
func handleService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := serviceGet(id)
	if err != nil {
		writeProblem(w, toProblem(err, r.URL.Path))
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func serviceGet(id string) error {
	if _, ok := db[id]; !ok {
		return notFoundError("article", id)
	}
	if id == "2" {
		return conflictError("article 2 is currently locked for editing")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /articles/{id}", handleGet)
	mux.HandleFunc("POST /articles", handleCreate)
	mux.HandleFunc("GET /service/{id}", handleService)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: mux}).Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}

	do := func(method, path, body string) (int, string, map[string]any) {
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
		resp, err := client.Do(req)
		if err != nil {
			return 0, "", nil
		}
		defer resp.Body.Close()
		ct := resp.Header.Get("Content-Type")
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, ct, out
	}

	check := func(label string, code, want int) {
		mark := "✓"
		if code != want {
			mark = "✗"
		}
		fmt.Printf("  %s %-52s %d\n", mark, label, code)
	}

	checkCT := func(label, ct string) {
		mark := "✓"
		if !strings.Contains(ct, "problem+json") {
			mark = "✗"
		}
		fmt.Printf("  %s %-52s %s\n", mark, label, ct)
	}

	fmt.Printf("=== RFC 7807 Error Handling — %s ===\n\n", base)

	fmt.Println("--- Not Found (404) ---")
	code, ct, body := do("GET", "/articles/99", "")
	check("GET /articles/99 → 404", code, 404)
	checkCT("Content-Type", ct)
	fmt.Printf("    type:   %v\n    title:  %v\n    detail: %v\n", body["type"], body["title"], body["detail"])

	fmt.Println()
	fmt.Println("--- Validation Error (422) ---")
	code, ct, body = do("POST", "/articles", `{"title":"","body":""}`)
	check("POST /articles (empty fields) → 422", code, 422)
	checkCT("Content-Type", ct)
	fmt.Printf("    type:   %v\n    title:  %v\n", body["type"], body["title"])
	if errs, ok := body["errors"].([]any); ok {
		for _, e := range errs {
			if m, ok := e.(map[string]any); ok {
				fmt.Printf("    field=%-10s message=%v\n", m["field"], m["message"])
			}
		}
	}

	fmt.Println()
	fmt.Println("--- Bad Request (400) ---")
	code, ct, body = do("POST", "/articles", `not-json`)
	check("POST /articles (bad JSON) → 400", code, 400)
	checkCT("Content-Type", ct)
	fmt.Printf("    type: %v\n", body["type"])

	fmt.Println()
	fmt.Println("--- Conflict (409) ---")
	db["3"] = &Article{ID: "3", Title: "Existing", Body: "..."}
	code, ct, body = do("POST", "/articles", `{"id":"3","title":"Dup","body":"body"}`)
	check("POST /articles (duplicate id) → 409", code, 409)
	checkCT("Content-Type", ct)
	fmt.Printf("    type: %v\n    detail: %v\n", body["type"], body["detail"])

	fmt.Println()
	fmt.Println("--- Domain error → Problem conversion ---")
	code, ct, body = do("GET", "/service/99", "")
	check("GET /service/99 (not found) → 404", code, 404)
	fmt.Printf("    type: %v\n", body["type"])

	code, ct, body = do("GET", "/service/2", "")
	check("GET /service/2 (conflict) → 409", code, 409)
	fmt.Printf("    type: %v\n    detail: %v\n", body["type"], body["detail"])

	fmt.Println()
	fmt.Println("--- Standard problem types in catalog ---")
	for _, name := range []string{"not-found", "validation-error", "conflict", "unauthorized", "forbidden", "bad-request", "internal-error", "rate-limit-exceeded"} {
		fmt.Printf("  %s%s\n", typeBase, name)
	}
}
