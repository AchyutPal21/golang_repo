// FILE: 04_error_handling/08_real_world_errors.go
// TOPIC: Real-World Error Handling — layered errors, HTTP patterns, pipelines
//
// Run: go run 04_error_handling/08_real_world_errors.go

package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ── Domain errors ────────────────────────────────────────────────────────────

var ErrNotFound = errors.New("not found")
var ErrInvalid = errors.New("invalid input")

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on %q: %s", e.Field, e.Message)
}

// ── Simulated data store ──────────────────────────────────────────────────────

var db = map[int]string{
	1: "alice:admin",
	2: "bob:user",
}

func dbGetUser(id int) (string, error) {
	v, ok := db[id]
	if !ok {
		return "", fmt.Errorf("dbGetUser id=%d: %w", id, ErrNotFound)
	}
	return v, nil
}

// ── Service layer — adds business context to errors ───────────────────────────

type User struct {
	ID   int
	Name string
	Role string
}

func parseUser(raw string) (User, error) {
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return User{}, fmt.Errorf("parseUser: %w: expected name:role, got %q",
			ErrInvalid, raw)
	}
	return User{Name: parts[0], Role: parts[1]}, nil
}

func serviceGetUser(id int) (User, error) {
	raw, err := dbGetUser(id)
	if err != nil {
		// Wrap with service context — caller sees the full chain
		return User{}, fmt.Errorf("serviceGetUser: %w", err)
	}
	user, err := parseUser(raw)
	if err != nil {
		return User{}, fmt.Errorf("serviceGetUser id=%d: %w", id, err)
	}
	user.ID = id
	return user, nil
}

// ── HTTP handler layer — maps domain errors to HTTP status codes ───────────────

type HTTPResponse struct {
	Status int
	Body   string
}

func handleGetUser(idStr string) HTTPResponse {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return HTTPResponse{400, fmt.Sprintf("bad request: %v", err)}
	}

	user, err := serviceGetUser(id)
	if err != nil {
		// Map domain errors to HTTP codes using errors.Is:
		if errors.Is(err, ErrNotFound) {
			return HTTPResponse{404, fmt.Sprintf("user %d not found", id)}
		}
		var ve *ValidationError
		if errors.As(err, &ve) {
			return HTTPResponse{422, fmt.Sprintf("unprocessable: %v", ve)}
		}
		return HTTPResponse{500, fmt.Sprintf("internal error: %v", err)}
	}
	return HTTPResponse{200, fmt.Sprintf("user: %+v", user)}
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Real-World Error Handling")
	fmt.Println("════════════════════════════════════════")

	fmt.Println("\n── Layered error wrapping ──")
	testCases := []string{"1", "2", "99", "abc"}
	for _, id := range testCases {
		resp := handleGetUser(id)
		fmt.Printf("  GET /user/%s → %d: %s\n", id, resp.Status, resp.Body)
	}

	fmt.Println("\n── Inspecting full error chain ──")
	_, err := serviceGetUser(99)
	fmt.Printf("  Full error:         %v\n", err)
	fmt.Printf("  errors.Is(NotFound): %v\n", errors.Is(err, ErrNotFound))

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  Each layer wraps with context: fmt.Errorf(\"layer: %w\", err)")
	fmt.Println("  errors.Is() traverses the whole chain")
	fmt.Println("  errors.As() extracts typed errors from chain")
	fmt.Println("  HTTP handler maps domain errors → HTTP status codes")
	fmt.Println("  Error messages: 'action: reason' format (e.g. 'dbGetUser id=99: not found')")
}
