// 02_visibility_exported.go
//
// VISIBILITY AND EXPORTED IDENTIFIERS — Deep Dive
//
// Go uses a single, elegant rule for access control:
//   - Identifier starts with an UPPERCASE letter → EXPORTED (public)
//   - Identifier starts with a lowercase letter  → UNEXPORTED (package-private)
//
// That's the entire visibility system. No keywords (public/private/protected).
// No annotations. Just the case of the first Unicode letter.
//
// This file covers:
//   - Why Go chose case-based visibility
//   - Exported vs unexported: variables, constants, types, functions, methods, fields
//   - Visibility across package boundaries
//   - Struct fields — exported vs unexported
//   - Embedded types and promoted fields
//   - The internal/ package mechanism (directory-level enforcement)
//   - Design principles: what to export and what to hide

package main

import (
	"fmt"
	"math"
)

// ============================================================
// PART 1: WHY CASE-BASED VISIBILITY?
// ============================================================
//
// The Go designers made a deliberate trade-off. Many languages use explicit
// keywords: Java has public/private/protected/package-private, C# has
// public/private/protected/internal, Rust has pub(crate)/pub(super)/pub.
//
// Problems with keyword-based visibility:
//   - Verbose: every declaration needs an access modifier.
//   - Decision fatigue: 3–4 visibility levels to choose from each time.
//   - Easy to forget: Java defaults to package-private, many devs forget.
//   - No visual signal in call sites: "myField" vs "MyField" immediately tells
//     you whether you're crossing a package boundary.
//
// Benefits of case-based visibility:
//   1. ZERO overhead — the compiler reads a single character.
//   2. VISUAL signal at the call site: somepackage.Exported vs internal.
//   3. Fewer decisions: binary choice (export or not).
//   4. Documentation: godoc only documents exported identifiers by default,
//      so uppercase = "this is part of your API".
//   5. Encourages small, focused APIs: if you want something exported,
//      you have to think about naming it with a capital letter.
//
// The trade-off: Go has only two visibility levels (exported, unexported).
// There is no "protected" (visible to subclasses — Go has no inheritance) or
// "friend" access. The internal/ directory partially compensates for the lack
// of a "module-private" level.

// ============================================================
// PART 2: EXPORTED vs UNEXPORTED IDENTIFIERS
// ============================================================

// --- Exported constant (part of the package's public API) ---
// Pi is the mathematical constant π, exported for use by any importer.
const Pi = 3.14159265358979

// --- Unexported constant (implementation detail) ---
// goldenRatio is used internally; importers don't need to know it exists.
const goldenRatio = 1.61803398874989

// --- Exported type ---
// Shape is an exported interface. Any package can implement or use it.
type Shape interface {
	Area() float64
	Perimeter() float64
	// Describe is exported — part of the contract.
	Describe() string
}

// --- Unexported type ---
// registry holds a list of all shapes created in this package.
// External packages cannot reference this type directly.
type registry struct {
	shapes []Shape
}

// --- Unexported package-level variable ---
var defaultRegistry = &registry{}

// --- Exported function that interacts with the unexported registry ---
// Register adds a Shape to the package's default registry.
// Notice: callers use Register(), but never see registry or defaultRegistry.
func Register(s Shape) {
	defaultRegistry.shapes = append(defaultRegistry.shapes, s)
}

// AllShapes returns a copy of all registered shapes.
// Returning a copy (not the slice itself) prevents external code from
// modifying the internal registry — an important encapsulation boundary.
func AllShapes() []Shape {
	result := make([]Shape, len(defaultRegistry.shapes))
	copy(result, defaultRegistry.shapes)
	return result
}

// ============================================================
// PART 3: STRUCT FIELDS — EXPORTED vs UNEXPORTED
// ============================================================
//
// Struct fields follow exactly the same rule as package-level identifiers.
// An exported field is readable and writable by any package.
// An unexported field is accessible only within the defining package.
//
// Design guidance:
//   - Unexported fields enforce invariants: use methods to control access.
//   - Exported fields are part of your API — changing their type is a breaking change.
//   - JSON/XML encoding: only EXPORTED fields are encoded by default.
//     (encoding/json skips unexported fields entirely.)
//   - For simple data-transfer structs (DTOs), exported fields with json tags
//     are idiomatic.
//   - For structs that enforce invariants (e.g., a valid email address),
//     unexported fields + constructor + methods is the right pattern.

// Circle has both exported and unexported fields.
type Circle struct {
	// Exported field — callers can read/write Color freely.
	Color string

	// Exported field with a json struct tag.
	// json:"label" controls how encoding/json serializes this field.
	Label string `json:"label"`

	// Unexported field — radius can only be set through NewCircle or SetRadius.
	// This enforces the invariant: radius must be > 0.
	radius float64
}

// NewCircle is the constructor (factory function) for Circle.
// Constructors for types with unexported fields are the idiomatic Go pattern.
// The function name convention: New<TypeName> when there's one main constructor.
func NewCircle(color string, label string, radius float64) (*Circle, error) {
	if radius <= 0 {
		// Validation happens here, protecting the invariant.
		return nil, fmt.Errorf("NewCircle: radius must be > 0, got %.2f", radius)
	}
	return &Circle{Color: color, Label: label, radius: radius}, nil
}

// Radius returns the unexported radius. A getter method exposes the value
// without allowing direct mutation.
//
// Go naming convention: getters are named after the field (NOT GetRadius).
// Just "Radius()", not "GetRadius()". This avoids the Java-style getter verbosity.
func (c *Circle) Radius() float64 {
	return c.radius
}

// SetRadius is a setter that validates before mutating.
func (c *Circle) SetRadius(r float64) error {
	if r <= 0 {
		return fmt.Errorf("SetRadius: radius must be > 0, got %.2f", r)
	}
	c.radius = r
	return nil
}

// Area implements the Shape interface (exported method, uppercase).
func (c *Circle) Area() float64 {
	return math.Pi * c.radius * c.radius
}

// Perimeter implements Shape.
func (c *Circle) Perimeter() float64 {
	return 2 * math.Pi * c.radius
}

// Describe implements Shape.
func (c *Circle) Describe() string {
	return fmt.Sprintf("Circle(%s, r=%.2f)", c.Label, c.radius)
}

// ============================================================
// PART 4: EMBEDDED TYPES AND PROMOTED FIELDS
// ============================================================
//
// When you embed a type in a struct, its exported fields and methods are
// "promoted" to the outer struct. Unexported fields are NOT promoted —
// they are accessible only through the embedded field's name.
//
// Promotion means:
//   outer.Field      works (instead of outer.Inner.Field)
//   outer.Method()   works (instead of outer.Inner.Method())
//
// Important: unexported FIELD names of an embedded type are promoted
// syntactically only within the same package. External packages see the
// promoted exported fields/methods but cannot access unexported ones.

// BaseShape provides common metadata that other shapes can embed.
type BaseShape struct {
	// ID is exported — promoted when embedded.
	ID int

	// createdAt is unexported — NOT promoted outside this package.
	createdAt string
}

// stamp sets the unexported field (only callable within this package).
func (b *BaseShape) stamp(ts string) {
	b.createdAt = ts
}

// CreatedAt exposes the unexported createdAt via a method.
func (b *BaseShape) CreatedAt() string {
	return b.createdAt
}

// Rectangle embeds BaseShape, promoting BaseShape.ID and BaseShape.CreatedAt().
type Rectangle struct {
	BaseShape // embedded — no field name, just the type

	// Exported fields
	Width  float64
	Height float64
}

func NewRectangle(id int, w, h float64) *Rectangle {
	r := &Rectangle{
		BaseShape: BaseShape{ID: id},
		Width:     w,
		Height:    h,
	}
	r.stamp("2026-01-01") // calling unexported method — OK, same package
	return r
}

func (r *Rectangle) Area() float64      { return r.Width * r.Height }
func (r *Rectangle) Perimeter() float64 { return 2 * (r.Width + r.Height) }
func (r *Rectangle) Describe() string {
	return fmt.Sprintf("Rectangle(id=%d, %.2fx%.2f)", r.ID, r.Width, r.Height)
}

// ============================================================
// PART 5: THE internal/ DIRECTORY — PACKAGE-TREE VISIBILITY
// ============================================================
//
// Problem: You have a multi-package module and want some packages to be
// "shared internally" but not importable by external modules.
//
// Solution: Place them under a directory named "internal".
//
// Rule (enforced by go compiler):
//   A package at path a/b/internal/c/d can ONLY be imported by code
//   rooted at a/b. Specifically, the importing package's path must have
//   a/b as a prefix.
//
// Examples:
//
//   Module root: github.com/acme/myapp
//
//   github.com/acme/myapp/internal/config
//     → Can be imported by: github.com/acme/myapp/...
//     → CANNOT be imported by: github.com/acme/other or any external module
//
//   github.com/acme/myapp/service/internal/db
//     → Can be imported by: github.com/acme/myapp/service/...
//     → CANNOT be imported by: github.com/acme/myapp/cmd/...
//       (because cmd is not rooted at myapp/service)
//
// This gives you a THIRD level of visibility:
//   - Unexported: visible within one package
//   - internal/:  visible within a subtree of the module
//   - Exported:   visible everywhere
//
// Common uses:
//   - Shared helper types used across packages in your module but not
//     intended as a stable public API.
//   - Database layer (internal/db) that only the service layer should use.
//   - Configuration parsing (internal/config) used by multiple commands.

// ============================================================
// PART 6: DESIGN PRINCIPLES — WHAT TO EXPORT
// ============================================================
//
// Principle 1 — Start unexported, export only when needed.
//   You can always EXPORT something later (backward compatible change).
//   You can NEVER unexport something without breaking callers.
//   → Be conservative. Export deliberately, not by default.
//
// Principle 2 — Export types, not implementation details.
//   Export the interface, hide the concrete struct when practical.
//   This lets you swap implementations without breaking callers.
//
// Principle 3 — Avoid exported "options structs" for simple functions.
//   For complex configuration, use the functional options pattern or
//   a Config struct. Don't export every tuning knob.
//
// Principle 4 — Exported field = stable contract.
//   If you add a field to an exported struct, callers using struct literals
//   with positional fields will BREAK if they don't update their code.
//   Use named fields in struct literals: Rect{Width: 3, Height: 4}
//   (not Rect{3, 4}) to future-proof your code.
//
// Principle 5 — The "accept interfaces, return structs" guideline.
//   Functions that accept an interface are more flexible (testable).
//   Functions that return a concrete struct are easier to extend
//   (you can add fields/methods without breaking the interface contract).
//   Note: this is a guideline, not an absolute rule.

// drawShape accepts any Shape (interface) — flexible, testable.
func drawShape(s Shape) {
	fmt.Printf("  Drawing: %-35s  area=%.4f  perim=%.4f\n",
		s.Describe(), s.Area(), s.Perimeter())
}

// ============================================================
// MAIN
// ============================================================

func main() {
	fmt.Println("=== 02: Visibility and Exported Identifiers ===")
	fmt.Println()

	// --- Exported vs unexported constants ---
	fmt.Println("--- Constants ---")
	fmt.Printf("  Pi (exported)         = %.10f\n", Pi)
	// goldenRatio is unexported but accessible within this package:
	fmt.Printf("  goldenRatio (unexported, same pkg) = %.10f\n", goldenRatio)
	fmt.Println()

	// --- Struct with unexported field ---
	fmt.Println("--- Circle: unexported radius, exported Color/Label ---")
	c, err := NewCircle("red", "my-circle", 5.0)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("  Color=%q  Label=%q  Radius()=%.2f\n", c.Color, c.Label, c.Radius())

	// c.radius = 3 // ← would compile fine here (same package)
	//              //   but would FAIL in an external package

	err = c.SetRadius(-1)
	fmt.Println("  SetRadius(-1) error:", err)
	fmt.Println()

	// --- Embedded struct and promoted fields ---
	fmt.Println("--- Rectangle: embedded BaseShape, promoted ID and CreatedAt() ---")
	rect := NewRectangle(42, 8.0, 3.0)
	fmt.Printf("  rect.ID (promoted exported field) = %d\n", rect.ID)
	fmt.Printf("  rect.CreatedAt() (promoted method) = %q\n", rect.CreatedAt())
	// rect.createdAt  ← unexported, not accessible in external package
	//                    (accessible here because we're in the same package)
	fmt.Printf("  rect.BaseShape.createdAt (direct access, same pkg) = %q\n", rect.BaseShape.createdAt)
	fmt.Println()

	// --- Registry: unexported registry, exported Register/AllShapes ---
	fmt.Println("--- Package-level registry (unexported type, exported API) ---")
	Register(c)
	Register(rect)
	shapes := AllShapes()
	fmt.Printf("  Registered %d shapes:\n", len(shapes))
	for _, s := range shapes {
		drawShape(s)
	}
	fmt.Println()

	// --- Visibility summary table ---
	fmt.Println("--- Visibility rules summary ---")
	rules := []struct {
		identifier string
		visibility string
		rule       string
	}{
		{"Pi", "Exported", "Uppercase first letter"},
		{"goldenRatio", "Unexported", "Lowercase first letter"},
		{"Shape", "Exported", "Interface, uppercase"},
		{"registry", "Unexported", "Struct, lowercase"},
		{"Circle.Color", "Exported field", "Uppercase field name"},
		{"Circle.radius", "Unexported field", "Lowercase field name"},
		{"Circle.Area()", "Exported method", "Uppercase method name"},
		{"internal/pkg", "Subtree-private", "internal/ directory rule"},
	}
	fmt.Printf("  %-20s  %-18s  %s\n", "Identifier", "Visibility", "Reason")
	fmt.Printf("  %-20s  %-18s  %s\n", "----------", "----------", "------")
	for _, r := range rules {
		fmt.Printf("  %-20s  %-18s  %s\n", r.identifier, r.visibility, r.rule)
	}
	fmt.Println()

	// --- internal/ directory illustration ---
	fmt.Println("--- internal/ directory: three visibility levels ---")
	levels := []string{
		"Unexported (lowercase)  → visible only within the PACKAGE",
		"internal/ directory     → visible within the MODULE SUBTREE",
		"Exported (uppercase)    → visible to ANY importer",
	}
	for i, l := range levels {
		fmt.Printf("  Level %d: %s\n", i+1, l)
	}
	fmt.Println()

	fmt.Println("=== End of 02_visibility_exported.go ===")
}
