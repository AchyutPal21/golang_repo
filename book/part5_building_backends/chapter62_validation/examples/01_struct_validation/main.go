// FILE: book/part5_building_backends/chapter62_validation/examples/01_struct_validation/main.go
// CHAPTER: 62 — Validation
// TOPIC: Manual struct validation without external libraries —
//        ValidationError, multi-error accumulation, required/length/regex/
//        range/enum rules, clean formatted output.
//
// Run (from the chapter folder):
//   go run ./examples/01_struct_validation

package main

import (
	"fmt"
	"regexp"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// VALIDATION TYPES
// ─────────────────────────────────────────────────────────────────────────────

// ValidationError describes a single validation failure.
type ValidationError struct {
	Field   string
	Rule    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s (%s): %s", e.Field, e.Rule, e.Message)
}

// ValidationErrors is a slice of ValidationError that satisfies the error
// interface by joining all messages.
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	msgs := make([]string, len(ve))
	for i, e := range ve {
		msgs[i] = e.Error()
	}
	return "validation failed:\n  " + strings.Join(msgs, "\n  ")
}

// ─────────────────────────────────────────────────────────────────────────────
// VALIDATOR
// ─────────────────────────────────────────────────────────────────────────────

// Validator accumulates errors and applies rules to a named field.
// Errors are always collected (not fail-fast), so a single call to Validate
// returns all errors at once.
type Validator struct {
	errs ValidationErrors
}

func (v *Validator) addError(field, rule, message string) {
	v.errs = append(v.errs, ValidationError{Field: field, Rule: rule, Message: message})
}

// Required fails if s is empty after trimming whitespace.
func (v *Validator) Required(field, s string) {
	if strings.TrimSpace(s) == "" {
		v.addError(field, "required", "field is required")
	}
}

// MinLen fails if len(s) < min.
func (v *Validator) MinLen(field, s string, min int) {
	if len(s) < min {
		v.addError(field, "min_length", fmt.Sprintf("must be at least %d characters (got %d)", min, len(s)))
	}
}

// MaxLen fails if len(s) > max.
func (v *Validator) MaxLen(field, s string, max int) {
	if len(s) > max {
		v.addError(field, "max_length", fmt.Sprintf("must be at most %d characters (got %d)", max, len(s)))
	}
}

// Matches fails if s does not match the provided compiled regexp.
func (v *Validator) Matches(field, s string, re *regexp.Regexp, description string) {
	if !re.MatchString(s) {
		v.addError(field, "pattern", fmt.Sprintf("must match %s", description))
	}
}

// InRange fails if n < min or n > max.
func (v *Validator) InRange(field string, n, min, max int) {
	if n < min || n > max {
		v.addError(field, "range", fmt.Sprintf("must be between %d and %d (got %d)", min, max, n))
	}
}

// OneOf fails if s is not in the allowed set.
func (v *Validator) OneOf(field, s string, allowed ...string) {
	for _, a := range allowed {
		if s == a {
			return
		}
	}
	v.addError(field, "enum", fmt.Sprintf("must be one of [%s] (got %q)", strings.Join(allowed, ", "), s))
}

// Errors returns nil if there are no validation errors, otherwise the
// accumulated ValidationErrors slice (as an error).
func (v *Validator) Errors() error {
	if len(v.errs) == 0 {
		return nil
	}
	return v.errs
}

// ─────────────────────────────────────────────────────────────────────────────
// COMMON REGEX PATTERNS
// ─────────────────────────────────────────────────────────────────────────────

var (
	reEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	rePhone = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN STRUCT & VALIDATION
// ─────────────────────────────────────────────────────────────────────────────

type CreateUserRequest struct {
	Name  string
	Email string
	Phone string
	Age   int
	Role  string
}

// Validate validates a CreateUserRequest and returns all errors at once.
func Validate(req CreateUserRequest) error {
	var v Validator

	v.Required("name", req.Name)
	v.MinLen("name", req.Name, 2)
	v.MaxLen("name", req.Name, 50)

	v.Required("email", req.Email)
	if req.Email != "" {
		v.Matches("email", req.Email, reEmail, "a valid email address")
	}

	if req.Phone != "" {
		v.Matches("phone", req.Phone, rePhone, "a valid phone number")
	}

	v.InRange("age", req.Age, 13, 120)

	v.Required("role", req.Role)
	v.OneOf("role", req.Role, "admin", "editor", "viewer", "guest")

	return v.Errors()
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	cases := []struct {
		label string
		req   CreateUserRequest
	}{
		{
			label: "valid request",
			req: CreateUserRequest{
				Name:  "Alice",
				Email: "alice@example.com",
				Phone: "+14155552671",
				Age:   30,
				Role:  "admin",
			},
		},
		{
			label: "missing required fields",
			req: CreateUserRequest{
				Name:  "",
				Email: "",
				Age:   30,
				Role:  "admin",
			},
		},
		{
			label: "invalid email + invalid role",
			req: CreateUserRequest{
				Name:  "Bob",
				Email: "not-an-email",
				Age:   25,
				Role:  "superuser",
			},
		},
		{
			label: "name too short + age out of range",
			req: CreateUserRequest{
				Name:  "X",
				Email: "x@example.com",
				Age:   5,
				Role:  "viewer",
			},
		},
		{
			label: "name too long + invalid phone",
			req: CreateUserRequest{
				Name:  strings.Repeat("A", 55),
				Email: "valid@example.com",
				Phone: "abc",
				Age:   40,
				Role:  "guest",
			},
		},
		{
			label: "multiple errors at once (all rules violated)",
			req: CreateUserRequest{
				Name:  "",
				Email: "bad-email",
				Phone: "!!",
				Age:   200,
				Role:  "root",
			},
		},
	}

	fmt.Println("=== Manual Struct Validation ===")
	fmt.Println()

	for _, c := range cases {
		fmt.Printf("─── %s ───\n", c.label)
		err := Validate(c.req)
		if err == nil {
			fmt.Println("  VALID ✓")
		} else {
			fmt.Printf("  INVALID:\n")
			for _, ve := range err.(ValidationErrors) {
				fmt.Printf("    • [%-12s] %-10s : %s\n", ve.Field, ve.Rule, ve.Message)
			}
		}
		fmt.Println()
	}

	fmt.Println("=== Error interface demo ===")
	req := CreateUserRequest{Name: "", Email: "bad", Age: 5, Role: "wizard"}
	if err := Validate(req); err != nil {
		fmt.Printf("err.Error():\n%s\n", err)
	}
}
