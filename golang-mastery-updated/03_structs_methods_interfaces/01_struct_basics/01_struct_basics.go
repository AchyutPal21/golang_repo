// 01_struct_basics.go
//
// STRUCTS — Go's primary mechanism for grouping related data.
//
// Go has no classes. Instead, it uses structs (data) + functions/methods (behavior).
// This is a deliberate design decision: keep data and behavior loosely coupled,
// prefer composition over a deep inheritance hierarchy.
//
// A struct is a composite type that groups named fields together.
// Think of it as a blueprint; each variable of that type is an instance.

package main

import (
	"encoding/json"
	"fmt"
)

// ─── 1. Basic Struct Declaration ──────────────────────────────────────────────
//
// Syntax: type <Name> struct { <field> <type> ... }
// Convention: PascalCase for exported (public) types and fields.
// Fields that start with a lowercase letter are unexported (package-private).

type Person struct {
	Name string // exported — visible outside this package
	Age  int    // exported
	email string // unexported — only accessible within this package
}

// ─── 2. Nested Structs ────────────────────────────────────────────────────────
//
// Structs can contain other structs as fields (composition).
// This is the foundation of Go's approach to building complex types.
// There is NO inheritance — only composition.

type Address struct {
	Street  string
	City    string
	Country string
	ZipCode string
}

// Employee has a nested Address struct as a NAMED FIELD.
// To access city: e.Address.City — you must go through the field name.
// Contrast this with embedding (see 02_struct_embedding.go), where the
// field name is the type name and fields are promoted to the outer struct.
type Employee struct {
	Person  Person  // named field — Person is a field OF Employee
	Address Address // named field
	Role    string
	Salary  float64
}

// ─── 3. Struct Tags ───────────────────────────────────────────────────────────
//
// Tags are metadata attached to struct fields using backtick literals.
// They are invisible at runtime unless explicitly read via the reflect package.
// The most common use is encoding/json — the json package reads these tags
// to map between Go field names and JSON keys.
//
// Format: `key:"value,options"` (multiple tags: `json:"..." db:"..."`)
//
// Common json tag options:
//   json:"name"        — rename field in JSON
//   json:"name,omitempty" — omit if zero value
//   json:"-"           — always omit this field from JSON

type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price,omitempty"` // omitted if price == 0.0
	InternalSKU string  `json:"-"`               // never serialized
	Description string  `json:"description,omitempty"`
}

// ─── 4. Anonymous Structs ─────────────────────────────────────────────────────
//
// You can declare a struct type inline without naming it.
// WHY: Useful for one-off data groupings that don't deserve a named type:
//   - test fixtures / table-driven tests
//   - temporary grouping of return values (prefer named types in production)
//   - JSON unmarshal targets for a single use
//
// Anonymous structs are zero-cost abstractions — no runtime overhead.

// ─── 5. Struct Equality ───────────────────────────────────────────────────────
//
// Two struct values are equal (==) if ALL their corresponding fields are equal.
// REQUIREMENT: All fields must be comparable types.
// Slices, maps, and functions are NOT comparable → struct containing them
// cannot be compared with ==. Use reflect.DeepEqual() for those, or
// define a custom Equal method.

type Point struct {
	X, Y int // shorthand: multiple fields of the same type on one line
}

// ─── 6. Structs Are VALUE Types ───────────────────────────────────────────────
//
// Assigning a struct copies ALL its fields. Modifying the copy does NOT
// affect the original. This is safe but can be expensive for large structs.
//
// To share and mutate the original, use a POINTER to the struct (*Person).
// Rule of thumb: if you need to modify the struct OR it's large (>~64 bytes),
// use a pointer. Otherwise, value semantics are fine and often preferred
// because they're easier to reason about (no aliasing surprises).

func demonstrateValueSemantics() {
	fmt.Println("\n── Value Semantics ──────────────────────────────────")

	original := Person{Name: "Alice", Age: 30}
	copy := original // full copy of all fields

	copy.Name = "Bob" // modifies the copy only

	fmt.Printf("original: %+v\n", original) // Name still "Alice"
	fmt.Printf("copy:     %+v\n", copy)     // Name is "Bob"

	// Pointer semantics: both variables point to the SAME underlying struct
	ptr := &original
	ptr.Name = "Carol" // Go auto-dereferences: (*ptr).Name = "Carol"

	fmt.Printf("original after ptr mutation: %+v\n", original) // Name is "Carol"
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("========================================")
	fmt.Println("  Go Structs — Basics")
	fmt.Println("========================================")

	// ── Struct Literals ─────────────────────────────────────────────────────
	// There are two forms of struct literals:
	//
	// 1. Named fields (PREFERRED): order doesn't matter, zero value for omitted fields
	// 2. Positional (AVOID in production): must supply all fields in declaration order
	//    Brittle — adding a field to the struct breaks all positional literals.

	fmt.Println("\n── Struct Literals ──────────────────────────────────")

	// Named field form — preferred
	p1 := Person{
		Name:  "Alice",
		Age:   30,
		email: "alice@example.com", // unexported, but accessible within same package
	}

	// Positional form — avoid, shown here for completeness
	// This breaks if you add, remove, or reorder fields in Person.
	p2 := Person{"Bob", 25, "bob@example.com"} // risky!

	fmt.Printf("p1 (named):      %+v\n", p1)   // %+v prints field names
	fmt.Printf("p2 (positional): %+v\n", p2)
	fmt.Printf("p1.email (unexported, same pkg): %s\n", p1.email)

	// ── Zero Value Struct ────────────────────────────────────────────────────
	// Every type in Go has a zero value. For structs, it's a struct where
	// every field is set to its zero value (0, "", false, nil, etc.).
	// Zero values mean you never have uninitialized garbage — a Go safety guarantee.

	fmt.Println("\n── Zero Value Struct ────────────────────────────────")

	var zp Person // zero value — no need for New() or constructors
	fmt.Printf("zero Person: %+v\n", zp)
	// Output: zero Person: {Name: Age:0 email:}

	// A struct is "ready to use" at its zero value.
	// This is why types like sync.Mutex, bytes.Buffer have no constructors:
	//   var mu sync.Mutex  // ready to use!
	//   var buf bytes.Buffer  // ready to use!

	// ── Nested Structs ───────────────────────────────────────────────────────
	fmt.Println("\n── Nested Structs ───────────────────────────────────")

	emp := Employee{
		Person: Person{
			Name:  "Charlie",
			Age:   35,
			email: "charlie@corp.com",
		},
		Address: Address{
			Street:  "123 Main St",
			City:    "San Francisco",
			Country: "USA",
			ZipCode: "94105",
		},
		Role:   "Software Engineer",
		Salary: 150000,
	}

	// To access nested fields, chain the field names
	fmt.Printf("Employee name: %s\n", emp.Person.Name)
	fmt.Printf("Employee city: %s\n", emp.Address.City)
	fmt.Printf("Role: %s, Salary: $%.0f\n", emp.Role, emp.Salary)

	// ── Struct Equality ──────────────────────────────────────────────────────
	fmt.Println("\n── Struct Equality ──────────────────────────────────")

	pt1 := Point{X: 1, Y: 2}
	pt2 := Point{X: 1, Y: 2}
	pt3 := Point{X: 3, Y: 4}

	fmt.Printf("pt1 == pt2: %v (same field values)\n", pt1 == pt2) // true
	fmt.Printf("pt1 == pt3: %v (different field values)\n", pt1 == pt3) // false

	// Structs can be used as map keys if all fields are comparable
	visited := map[Point]bool{
		{0, 0}: true,
		{1, 2}: true,
	}
	fmt.Printf("Point{1,2} visited: %v\n", visited[Point{1, 2}]) // true

	// ── Anonymous Structs ────────────────────────────────────────────────────
	fmt.Println("\n── Anonymous Structs ────────────────────────────────")

	// Inline struct type — useful for one-off groupings
	config := struct {
		Host    string
		Port    int
		Debug   bool
	}{
		Host:  "localhost",
		Port:  8080,
		Debug: true,
	}
	fmt.Printf("config: %+v\n", config)

	// Common use in table-driven tests:
	tests := []struct {
		input    string
		expected int
	}{
		{"hello", 5},
		{"world!", 6},
		{"", 0},
	}
	for _, tt := range tests {
		got := len(tt.input)
		status := "PASS"
		if got != tt.expected {
			status = "FAIL"
		}
		fmt.Printf("[%s] len(%q) = %d (expected %d)\n", status, tt.input, got, tt.expected)
	}

	// ── Value Semantics Demo ─────────────────────────────────────────────────
	demonstrateValueSemantics()

	// ── Struct Tags & JSON ───────────────────────────────────────────────────
	fmt.Println("\n── Struct Tags & JSON ───────────────────────────────")

	products := []Product{
		{ID: 1, Name: "Widget", Price: 9.99, InternalSKU: "WDG-001", Description: "A small widget"},
		{ID: 2, Name: "Gadget", Price: 0, InternalSKU: "GDG-002"}, // Price=0, omitempty
	}

	for _, prod := range products {
		jsonBytes, err := json.MarshalIndent(prod, "", "  ")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		fmt.Printf("Product %d JSON:\n%s\n\n", prod.ID, string(jsonBytes))
		// Notice:
		// - "id", "name" are lowercase (from json tag)
		// - "price" is omitted for product 2 (omitempty + zero value)
		// - InternalSKU is NEVER in the JSON (json:"-")
		// - "description" is omitted for product 2 (omitempty + empty string)
	}

	// ── Struct as Function Argument ──────────────────────────────────────────
	fmt.Println("\n── Passing Structs: Value vs Pointer ────────────────")

	alice := Person{Name: "Alice", Age: 30}

	// Passing by value: function gets a COPY
	// Cannot mutate the original
	printPersonByValue(alice)

	// Passing by pointer: function gets a reference to the original
	// CAN mutate the original, no copy overhead for large structs
	birthday(&alice)
	fmt.Printf("After birthday(&alice): age = %d\n", alice.Age) // 31

	// ── Constructor Functions ────────────────────────────────────────────────
	// Go doesn't have constructors, but the convention is a NewXxx function.
	// Use these when:
	//   - you need to validate inputs
	//   - you need to set unexported fields
	//   - the zero value isn't usable directly

	fmt.Println("\n── Constructor Function Pattern ─────────────────────")
	emp2, err := NewEmployee("Diana", 28, "diana@corp.com", "Backend Engineer", 130000)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Created employee: %s, Role: %s\n", emp2.Person.Name, emp2.Role)
	}

	// Attempt to create invalid employee
	_, err = NewEmployee("", 28, "x@y.com", "Engineer", 100000)
	if err != nil {
		fmt.Printf("Validation error: %v\n", err)
	}
}

// printPersonByValue receives a copy of Person.
// Any modifications here are invisible to the caller.
func printPersonByValue(p Person) {
	fmt.Printf("  printPersonByValue: %s, age %d\n", p.Name, p.Age)
}

// birthday takes a POINTER — it can modify the original.
// Convention: pointer receiver / pointer arg when you need mutation or large struct.
func birthday(p *Person) {
	p.Age++ // Go auto-dereferences: (*p).Age++
}

// NewEmployee is a constructor function (the Go pattern for constructors).
// Returns (*Employee, error) — errors from validation, resource allocation, etc.
// WHY return a pointer: the Employee struct is large; callers usually want
// to share and mutate a single instance rather than copy it around.
func NewEmployee(name string, age int, email, role string, salary float64) (*Employee, error) {
	if name == "" {
		return nil, fmt.Errorf("NewEmployee: name cannot be empty")
	}
	if salary < 0 {
		return nil, fmt.Errorf("NewEmployee: salary cannot be negative")
	}
	return &Employee{
		Person:  Person{Name: name, Age: age, email: email},
		Role:    role,
		Salary:  salary,
	}, nil
}
