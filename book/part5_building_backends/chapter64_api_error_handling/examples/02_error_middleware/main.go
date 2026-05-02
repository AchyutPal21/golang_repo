// FILE: book/part5_building_backends/chapter64_api_error_handling/examples/02_error_middleware/main.go
// CHAPTER: 64 — API Error Handling
// TOPIC: Error handling middleware pattern — handlers return errors instead of
//        writing responses directly; a central error handler converts them to
//        RFC 7807 problem details. Eliminates error-handling boilerplate.
//
// Pattern: define HandlerFunc as func(w, r) error, wrap in an adapter that
// catches the returned error and dispatches to the central error handler.
// This is similar to how frameworks like echo handle errors.
//
// Run (from the chapter folder):
//   go run ./examples/02_error_middleware

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PROBLEM DETAIL (RFC 7807)
// ─────────────────────────────────────────────────────────────────────────────

type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

const problemCT = "application/problem+json"
const typeBase = "https://api.example.com/errors/"

// ─────────────────────────────────────────────────────────────────────────────
// TYPED API ERRORS — carry status code + problem details
// ─────────────────────────────────────────────────────────────────────────────

type APIError struct {
	Problem
	cause error  // underlying error (logged, not exposed to client)
	file  string // source location for debugging
	line  int
}

func (e *APIError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Detail, e.cause)
	}
	return e.Detail
}

func (e *APIError) Unwrap() error { return e.cause }

// newAPIError creates an APIError, capturing source location.
func newAPIError(p Problem, cause error) *APIError {
	_, file, line, _ := runtime.Caller(1)
	// Strip the full path to just filename.
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		file = file[idx+1:]
	}
	return &APIError{Problem: p, cause: cause, file: file, line: line}
}

// ─────────────────────────────────────────────────────────────────────────────
// ERROR CONSTRUCTORS
// ─────────────────────────────────────────────────────────────────────────────

func ErrNotFound(resource, id string) *APIError {
	return newAPIError(Problem{
		Type:   typeBase + "not-found",
		Title:  "Not Found",
		Status: 404,
		Detail: fmt.Sprintf("%s '%s' does not exist", resource, id),
	}, nil)
}

func ErrBadRequest(detail string, cause error) *APIError {
	return newAPIError(Problem{
		Type:   typeBase + "bad-request",
		Title:  "Bad Request",
		Status: 400,
		Detail: detail,
	}, cause)
}

func ErrValidation(detail string) *APIError {
	return newAPIError(Problem{
		Type:   typeBase + "validation-error",
		Title:  "Validation Error",
		Status: 422,
		Detail: detail,
	}, nil)
}

func ErrUnauthorized(detail string) *APIError {
	return newAPIError(Problem{
		Type:   typeBase + "unauthorized",
		Title:  "Unauthorized",
		Status: 401,
		Detail: detail,
	}, nil)
}

func ErrInternal(cause error) *APIError {
	return newAPIError(Problem{
		Type:   typeBase + "internal-error",
		Title:  "Internal Server Error",
		Status: 500,
		Detail: "an unexpected error occurred",
	}, cause)
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLER TYPE — returns error instead of writing response
// ─────────────────────────────────────────────────────────────────────────────

type Handler func(w http.ResponseWriter, r *http.Request) error

// Adapt converts a Handler to http.HandlerFunc, routing errors to handleErr.
func Adapt(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			handleErr(w, r, err)
		}
	}
}

// handleErr is the central error handler.
func handleErr(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		apiErr.Problem.Instance = r.URL.Path
		// Log internal errors with cause + source location.
		if apiErr.Status >= 500 {
			fmt.Printf("  [ERROR] %s:%d %v\n", apiErr.file, apiErr.line, apiErr.cause)
		}
		w.Header().Set("Content-Type", problemCT)
		w.WriteHeader(apiErr.Status)
		json.NewEncoder(w).Encode(apiErr.Problem)
		return
	}
	// Untyped error — treat as 500.
	fmt.Printf("  [ERROR] untyped: %v\n", err)
	w.Header().Set("Content-Type", problemCT)
	w.WriteHeader(500)
	json.NewEncoder(w).Encode(Problem{
		Type:   typeBase + "internal-error",
		Title:  "Internal Server Error",
		Status: 500,
		Detail: "an unexpected error occurred",
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type Post struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

var posts = map[string]*Post{
	"1": {ID: "1", Title: "First Post", Body: "Hello world"},
	"2": {ID: "2", Title: "Second Post", Body: "More content"},
}

// ─────────────────────────────────────────────────────────────────────────────
// HANDLERS — return error; no boilerplate writeJSON calls
// ─────────────────────────────────────────────────────────────────────────────

func getPost(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	p, ok := posts[id]
	if !ok {
		return ErrNotFound("post", id)
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(p)
}

func createPost(w http.ResponseWriter, r *http.Request) error {
	// Auth check.
	if r.Header.Get("Authorization") == "" {
		return ErrUnauthorized("bearer token required")
	}

	var p Post
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		return ErrBadRequest("cannot decode request body", err)
	}
	if strings.TrimSpace(p.Title) == "" {
		return ErrValidation("title is required")
	}
	if strings.TrimSpace(p.Body) == "" {
		return ErrValidation("body is required")
	}

	posts[p.ID] = &p
	w.Header().Set("Location", "/posts/"+p.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(p)
}

func panicPost(w http.ResponseWriter, r *http.Request) error {
	// Simulate an internal error that is converted to ErrInternal.
	err := callService()
	if err != nil {
		return ErrInternal(err)
	}
	w.Write([]byte(`{"ok":true}`))
	return nil
}

func callService() error {
	return fmt.Errorf("database connection refused: connection to 10.0.0.1:5432 timed out")
}

func unhandledPanic(w http.ResponseWriter, r *http.Request) error {
	return errors.New("raw untyped error — becomes 500")
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /posts/{id}", Adapt(getPost))
	mux.HandleFunc("POST /posts", Adapt(createPost))
	mux.HandleFunc("GET /internal-error", Adapt(panicPost))
	mux.HandleFunc("GET /untyped-error", Adapt(unhandledPanic))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: mux}).Serve(ln)

	client := &http.Client{Timeout: 5 * time.Second}

	do := func(method, path, body, auth string) (int, string, map[string]any) {
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

	fmt.Printf("=== Error Middleware — %s ===\n\n", base)

	fmt.Println("--- Success paths ---")
	code, _, _ := do("GET", "/posts/1", "", "")
	check("GET /posts/1 → 200", code, 200)

	code, _, _ = do("POST", "/posts", `{"id":"5","title":"New","body":"text"}`, "Bearer tok")
	check("POST /posts (auth+valid) → 201", code, 201)

	fmt.Println()
	fmt.Println("--- Error paths (all return application/problem+json) ---")
	code, ct, body := do("GET", "/posts/99", "", "")
	check("GET /posts/99 → 404", code, 404)
	fmt.Printf("    Content-Type: %s\n    type: %v\n    detail: %v\n", ct, body["type"], body["detail"])

	code, ct, body = do("POST", "/posts", `{"id":"6","title":"New","body":"text"}`, "")
	check("POST /posts (no auth) → 401", code, 401)
	fmt.Printf("    type: %v\n", body["type"])

	code, ct, body = do("POST", "/posts", `{"id":"7","title":""}`, "Bearer tok")
	check("POST /posts (no title) → 422", code, 422)
	fmt.Printf("    type: %v\n    detail: %v\n", body["type"], body["detail"])

	code, ct, body = do("POST", "/posts", `bad-json`, "Bearer tok")
	check("POST /posts (bad JSON) → 400", code, 400)
	fmt.Printf("    type: %v\n", body["type"])

	fmt.Println()
	fmt.Println("--- Internal error (cause logged, not exposed) ---")
	code, _, body = do("GET", "/internal-error", "", "")
	check("GET /internal-error → 500", code, 500)
	fmt.Printf("    client sees: %v\n", body["detail"])

	fmt.Println()
	fmt.Println("--- Untyped error → 500 ---")
	code, _, _ = do("GET", "/untyped-error", "", "")
	check("GET /untyped-error → 500", code, 500)
}
