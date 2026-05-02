// EXERCISE 37.1 — Build a typed HTTP error system.
//
// HTTPError carries a status code and a safe-to-expose message.
// Implement custom Is() (match by status code), Unwrap(), and an HTTPStatus() method.
// A handler uses errors.Is and errors.As to produce correct HTTP responses.
//
// Run (from the chapter folder):
//   go run ./exercises/01_http_errors

package main

import (
	"errors"
	"fmt"
)

// ─── HTTPError ────────────────────────────────────────────────────────────────

type HTTPError struct {
	Status  int
	Message string
	Cause   error
}

func (e *HTTPError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("HTTP %d %s: %v", e.Status, e.Message, e.Cause)
	}
	return fmt.Sprintf("HTTP %d %s", e.Status, e.Message)
}

func (e *HTTPError) Unwrap() error { return e.Cause }

// Is matches by status code — message and cause are ignored.
func (e *HTTPError) Is(target error) bool {
	t, ok := target.(*HTTPError)
	if !ok {
		return false
	}
	return e.Status == t.Status
}

func (e *HTTPError) HTTPStatus() int { return e.Status }

// Sentinel HTTP errors.
var (
	ErrHTTP400 = &HTTPError{Status: 400, Message: "bad request"}
	ErrHTTP401 = &HTTPError{Status: 401, Message: "unauthorised"}
	ErrHTTP403 = &HTTPError{Status: 403, Message: "forbidden"}
	ErrHTTP404 = &HTTPError{Status: 404, Message: "not found"}
	ErrHTTP409 = &HTTPError{Status: 409, Message: "conflict"}
	ErrHTTP422 = &HTTPError{Status: 422, Message: "unprocessable entity"}
	ErrHTTP500 = &HTTPError{Status: 500, Message: "internal server error"}
)

func NewHTTPError(sentinel *HTTPError, detail string, cause error) *HTTPError {
	return &HTTPError{Status: sentinel.Status, Message: detail, Cause: cause}
}

// ─── Simulated service ────────────────────────────────────────────────────────

var users = map[int]string{1: "alice", 2: "bob"}

func getUser(callerID, targetID int) (string, error) {
	if callerID == 0 {
		return "", NewHTTPError(ErrHTTP401, "missing auth token", nil)
	}
	if callerID != targetID && callerID != 999 { // 999 = admin
		return "", NewHTTPError(ErrHTTP403,
			fmt.Sprintf("user %d cannot read user %d", callerID, targetID), nil)
	}
	name, ok := users[targetID]
	if !ok {
		dbErr := fmt.Errorf("SELECT failed: no rows for id=%d", targetID)
		return "", NewHTTPError(ErrHTTP404,
			fmt.Sprintf("user %d not found", targetID), dbErr)
	}
	return name, nil
}

func createUser(callerID int, name string) error {
	if callerID == 0 {
		return NewHTTPError(ErrHTTP401, "missing auth token", nil)
	}
	if name == "" {
		return NewHTTPError(ErrHTTP422, "name is required", nil)
	}
	for _, n := range users {
		if n == name {
			return NewHTTPError(ErrHTTP409,
				fmt.Sprintf("user %q already exists", name), nil)
		}
	}
	newID := len(users) + 1
	users[newID] = name
	return nil
}

// ─── Handler ──────────────────────────────────────────────────────────────────

type Response struct {
	Status int
	Body   string
}

func handleGetUser(callerID, targetID int) Response {
	name, err := getUser(callerID, targetID)
	if err == nil {
		return Response{200, fmt.Sprintf(`{"id":%d,"name":%q}`, targetID, name)}
	}

	var he *HTTPError
	if errors.As(err, &he) {
		return Response{he.HTTPStatus(), fmt.Sprintf(`{"error":%q}`, he.Message)}
	}
	return Response{500, `{"error":"internal server error"}`}
}

func handleCreateUser(callerID int, name string) Response {
	err := createUser(callerID, name)
	if err == nil {
		return Response{201, fmt.Sprintf(`{"name":%q,"status":"created"}`, name)}
	}

	// Use errors.Is for pattern matching.
	switch {
	case errors.Is(err, ErrHTTP401):
		return Response{401, `{"error":"authentication required"}`}
	case errors.Is(err, ErrHTTP409):
		return Response{409, `{"error":"user already exists"}`}
	case errors.Is(err, ErrHTTP422):
		var he *HTTPError
		errors.As(err, &he)
		return Response{422, fmt.Sprintf(`{"error":%q}`, he.Message)}
	default:
		return Response{500, `{"error":"internal server error"}`}
	}
}

func printResp(label string, r Response) {
	fmt.Printf("  %-45s  HTTP %d  %s\n", label, r.Status, r.Body)
}

func main() {
	fmt.Println("=== GET /users/{id} ===")
	printResp("unauthenticated (callerID=0)", handleGetUser(0, 1))
	printResp("alice reads own profile (1→1)", handleGetUser(1, 1))
	printResp("alice reads bob's profile (1→2)", handleGetUser(1, 2))
	printResp("admin reads alice (999→1)", handleGetUser(999, 1))
	printResp("alice reads missing user (1→99)", handleGetUser(1, 99))

	fmt.Println()
	fmt.Println("=== POST /users ===")
	printResp("unauthenticated create", handleCreateUser(0, "carol"))
	printResp("create carol (new user)", handleCreateUser(1, "carol"))
	printResp("create carol again (conflict)", handleCreateUser(1, "carol"))
	printResp("create with empty name", handleCreateUser(1, ""))

	fmt.Println()
	fmt.Println("=== errors.Is chain test ===")
	err := fmt.Errorf("service: %w",
		NewHTTPError(ErrHTTP404, "resource not found", fmt.Errorf("db error")))
	fmt.Println("  Is ErrHTTP404:", errors.Is(err, ErrHTTP404))
	fmt.Println("  Is ErrHTTP403:", errors.Is(err, ErrHTTP403))

	var he *HTTPError
	if errors.As(err, &he) {
		fmt.Printf("  extracted status=%d msg=%q\n", he.Status, he.Message)
	}
}
