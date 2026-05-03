# Chapter 62 — Input Validation

## What you'll learn

How to validate incoming data before it reaches your business logic — collecting all errors at once (not fail-fast), writing reusable validation rules, returning machine-readable error payloads, and validating query parameters and path parameters in addition to request bodies.

## Key concepts

| Concept | Description |
|---|---|
| `ValidationError` | Single field failure: field name, rule violated, human message |
| `ValidationErrors` | Slice of errors; `Error()` joins all messages; returned from validators |
| Collect-all approach | Run every rule before returning — client gets all problems in one round trip |
| HTTP 422 Unprocessable Entity | Semantically invalid input (parsed OK, business rules violated) |
| HTTP 400 Bad Request | Malformed input (JSON parse error, wrong content-type) |

## Files

| File | Topic |
|---|---|
| `examples/01_struct_validation/main.go` | Rule functions, `ValidationErrors`, collect-all pattern |
| `examples/02_form_api_validation/main.go` | Validation in HTTP handlers; 422 with structured error body; query params |
| `exercises/01_validated_api/main.go` | Products CRUD API with comprehensive validation rules |

## Validation error types

```go
type ValidationError struct {
    Field   string `json:"field"`
    Rule    string `json:"rule"`
    Message string `json:"message"`
}

type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
    msgs := make([]string, len(ve))
    for i, e := range ve { msgs[i] = e.Field + ": " + e.Message }
    return strings.Join(msgs, "; ")
}
```

## Collect-all validator pattern

```go
func validateProduct(p Product) ValidationErrors {
    var errs ValidationErrors

    if strings.TrimSpace(p.Name) == "" {
        errs = append(errs, ValidationError{Field: "name", Rule: "required", Message: "name is required"})
    } else if len(p.Name) < 2 {
        errs = append(errs, ValidationError{Field: "name", Rule: "min_length", Message: "name must be at least 2 characters"})
    }

    if p.Price <= 0 {
        errs = append(errs, ValidationError{Field: "price", Rule: "positive", Message: "price must be greater than zero"})
    }

    validCategories := map[string]bool{"electronics": true, "clothing": true, "books": true}
    if p.Category != "" && !validCategories[p.Category] {
        errs = append(errs, ValidationError{Field: "category", Rule: "enum", Message: "invalid category"})
    }

    return errs
}
```

## 422 response pattern

```go
func handleCreate(w http.ResponseWriter, r *http.Request) {
    var input Product
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        // 400 — malformed JSON
        http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
        return
    }

    if errs := validateProduct(input); len(errs) > 0 {
        // 422 — parsed but invalid
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnprocessableEntity)
        json.NewEncoder(w).Encode(map[string]any{
            "error":  "validation failed",
            "fields": errs,
        })
        return
    }
    // ... proceed
}
```

## Common rules

| Rule | Check |
|---|---|
| `required` | `strings.TrimSpace(s) != ""` |
| `min_length` | `len(s) >= n` |
| `max_length` | `len(s) <= n` |
| `pattern` | `regexp.MustCompile(pattern).MatchString(s)` |
| `range` | `v >= min && v <= max` |
| `enum` | `validValues[v]` |
| `positive` | `v > 0` |
| `non_negative` | `v >= 0` |

## Production tips

- Validate at the boundary — as close to the HTTP handler as possible, before domain logic.
- Use regex patterns compiled once at package level (`var reEmail = regexp.MustCompile(...)`).
- Return `application/json` error bodies — never plain text for API endpoints.
- Distinguish 400 (parse failure) from 422 (semantic failure) — clients handle them differently.
- For large APIs, consider a declarative validation library like `go-playground/validator`.
