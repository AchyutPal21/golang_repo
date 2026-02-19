package main

// =============================================================================
// MODULE 03: STRUCTS, METHODS & INTERFACES — The heart of Go
// =============================================================================
// Run: go run 03_structs_methods_interfaces/main.go
//
// This module covers:
//   - Structs (definition, embedding, tags, anonymous)
//   - Methods (value receivers, pointer receivers)
//   - Interfaces (duck typing, polymorphism)
//   - Type assertions and type switches
//   - The empty interface (any)
//   - Stringer, error interfaces
// =============================================================================

import (
	"fmt"
	"math"
)

// =============================================================================
// STRUCTS — Custom composite types
// =============================================================================
// A struct is a collection of named fields.
// It's Go's way to group related data — like a class without inheritance.

// Basic struct definition
type Point struct {
	X float64
	Y float64
}

// Multiple fields of same type — shorthand
type Rectangle struct {
	Width, Height float64 // both are float64
}

// Nested struct
type Address struct {
	Street string
	City   string
	State  string
	Zip    string
}

type Person struct {
	Name    string
	Age     int
	Email   string
	Address Address // nested struct — composition
}

// Anonymous struct fields (embedding) — promotes fields and methods
// This is Go's version of inheritance — COMPOSITION over INHERITANCE
type Animal struct {
	Name   string
	Weight float64
}

type Dog struct {
	Animal        // embedded (anonymous field) — Dog "inherits" Animal's fields
	Breed  string
	Trained bool
}

// Struct tags — metadata used by encoding packages (JSON, XML, DB)
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`            // - means SKIP this field in JSON
	Email    string `json:"email,omitempty"` // omitempty: skip if zero value
	IsAdmin  bool   `json:"is_admin"`
}

// =============================================================================
// METHODS — Functions attached to a type
// =============================================================================
// Syntax: func (receiver TypeName) MethodName(params) returnType
// The receiver can be:
//   - Value receiver:   (r Rectangle) — works on a COPY of the struct
//   - Pointer receiver: (r *Rectangle) — works on the ORIGINAL struct

// Method with VALUE receiver — gets a copy, cannot modify original
func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

func (r Rectangle) Perimeter() float64 {
	return 2 * (r.Width + r.Height)
}

// Stringer interface — defines how a type prints itself
// If you implement String() string, fmt.Println will use it automatically
func (r Rectangle) String() string {
	return fmt.Sprintf("Rectangle(%.1f x %.1f)", r.Width, r.Height)
}

// Method with POINTER receiver — modifies the original
func (r *Rectangle) Scale(factor float64) {
	r.Width *= factor  // modifies the actual Rectangle
	r.Height *= factor // modifies the actual Rectangle
}

// Methods on Point
func (p Point) Distance(other Point) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func (p *Point) Translate(dx, dy float64) {
	p.X += dx
	p.Y += dy
}

func (p Point) String() string {
	return fmt.Sprintf("Point(%.1f, %.1f)", p.X, p.Y)
}

// Methods on non-struct types — you can attach methods to ANY type in the package
type Celsius float64
type Fahrenheit float64

func (c Celsius) ToFahrenheit() Fahrenheit {
	return Fahrenheit(c*9/5 + 32)
}

func (f Fahrenheit) ToCelsius() Celsius {
	return Celsius((f - 32) * 5 / 9)
}

func (c Celsius) String() string {
	return fmt.Sprintf("%.1f°C", float64(c))
}

// Methods on Animal
func (a Animal) Describe() string {
	return fmt.Sprintf("%s (%.1f kg)", a.Name, a.Weight)
}

func (a *Animal) Feed(amount float64) {
	a.Weight += amount
}

// Methods on Dog — it can call Animal's methods too
func (d Dog) String() string {
	return fmt.Sprintf("Dog{%s, breed=%s, trained=%v}", d.Animal.Describe(), d.Breed, d.Trained)
}

// =============================================================================
// INTERFACES — Go's most powerful feature
// =============================================================================
// An interface defines a SET OF METHOD SIGNATURES.
// Any type that implements ALL methods of an interface automatically
// satisfies the interface — NO explicit declaration needed.
// This is called DUCK TYPING: "if it quacks like a duck, it's a duck."

// Define an interface
type Shape interface {
	Area() float64
	Perimeter() float64
}

// Another interface
type Stringer interface {
	String() string
}

// Interface with a single method (common in Go — small interfaces are powerful)
type Writer interface {
	Write(data string) (int, error)
}

type Reader interface {
	Read() string
}

// Composing interfaces — embed smaller interfaces into larger ones
type ReadWriter interface {
	Reader
	Writer
}

// --- Types that implement Shape ---

// Circle — implicitly implements Shape because it has Area() and Perimeter()
type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

func (c Circle) Perimeter() float64 {
	return 2 * math.Pi * c.Radius
}

func (c Circle) String() string {
	return fmt.Sprintf("Circle(r=%.1f)", c.Radius)
}

// Triangle — also implements Shape
type Triangle struct {
	A, B, C float64 // side lengths
}

func (t Triangle) Area() float64 {
	s := (t.A + t.B + t.C) / 2 // semi-perimeter
	return math.Sqrt(s * (s - t.A) * (s - t.B) * (s - t.C))
}

func (t Triangle) Perimeter() float64 {
	return t.A + t.B + t.C
}

// --- Using the interface ---

// This function works with ANY Shape — polymorphism!
func printShapeInfo(s Shape) {
	fmt.Printf("Area: %.2f, Perimeter: %.2f\n", s.Area(), s.Perimeter())
}

// Total area of any collection of shapes
func totalArea(shapes []Shape) float64 {
	total := 0.0
	for _, s := range shapes {
		total += s.Area()
	}
	return total
}

// =============================================================================
// EMPTY INTERFACE — any
// =============================================================================
// interface{} (or 'any' in Go 1.18+) is satisfied by EVERY type.
// Used when you don't know the type at compile time.
// Similar to Object in Java — but you must assert the type to use the value.

func describe(i interface{}) {
	fmt.Printf("value: %v, type: %T\n", i, i)
}

// =============================================================================
// TYPE ASSERTION — extracting concrete type from interface
// =============================================================================
// syntax: value, ok := interfaceVar.(ConcreteType)
// If ok is false, the assertion failed — value is the zero type.
// If you use value := interfaceVar.(ConcreteType) without ok and it fails → PANIC.

// =============================================================================
// TYPE SWITCH — check multiple types cleanly
// =============================================================================
func typeSwitch(i interface{}) string {
	switch v := i.(type) {
	case int:
		return fmt.Sprintf("int: %d", v)
	case float64:
		return fmt.Sprintf("float64: %f", v)
	case string:
		return fmt.Sprintf("string: %q", v)
	case bool:
		return fmt.Sprintf("bool: %v", v)
	case []int:
		return fmt.Sprintf("[]int with %d elements", len(v))
	case nil:
		return "nil"
	default:
		return fmt.Sprintf("unknown type: %T", v)
	}
}

// =============================================================================
// INTERFACE VALUES — internal representation
// =============================================================================
// An interface value is a pair: (type, value)
// nil interface: both type and value are nil
// non-nil interface with nil value: type is set, value is nil — subtle!

// =============================================================================
// PRACTICAL PATTERNS
// =============================================================================

// Pattern 1: Accept interface, return concrete type
// This makes your functions flexible (polymorphic input)

// Pattern 2: Builder pattern using methods
type QueryBuilder struct {
	table      string
	conditions []string
	limit      int
}

func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{table: table, limit: -1}
}

func (q *QueryBuilder) Where(condition string) *QueryBuilder {
	q.conditions = append(q.conditions, condition)
	return q // return self for chaining
}

func (q *QueryBuilder) Limit(n int) *QueryBuilder {
	q.limit = n
	return q
}

func (q *QueryBuilder) Build() string {
	query := "SELECT * FROM " + q.table
	for i, cond := range q.conditions {
		if i == 0 {
			query += " WHERE " + cond
		} else {
			query += " AND " + cond
		}
	}
	if q.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", q.limit)
	}
	return query
}

// Pattern 3: Option pattern using functional options
type Server struct {
	host    string
	port    int
	timeout int
}

type ServerOption func(*Server)

func WithHost(host string) ServerOption {
	return func(s *Server) { s.host = host }
}

func WithPort(port int) ServerOption {
	return func(s *Server) { s.port = port }
}

func WithTimeout(t int) ServerOption {
	return func(s *Server) { s.timeout = t }
}

func NewServer(opts ...ServerOption) *Server {
	s := &Server{host: "localhost", port: 8080, timeout: 30} // defaults
	for _, opt := range opts {
		opt(s) // apply each option
	}
	return s
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("=== MODULE 03: STRUCTS, METHODS & INTERFACES ===")

	// -------------------------------------------------------------------------
	// SECTION 1: Struct basics
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Structs ---")

	// Positional initialization (order matters)
	p1 := Point{1.0, 2.0}
	fmt.Println("p1:", p1)

	// Named field initialization (order doesn't matter, safer)
	p2 := Point{X: 5.0, Y: 3.0}
	fmt.Println("p2:", p2)

	// Partial initialization — unset fields get zero values
	p3 := Point{X: 10} // Y defaults to 0.0
	fmt.Println("p3:", p3)

	// Zero value struct — all fields zero
	var p4 Point
	fmt.Println("p4 (zero):", p4)

	// Pointer to struct — using &
	pp := &Point{3.0, 4.0}
	fmt.Println("pp:", pp)
	fmt.Println("*pp:", *pp)

	// new() — creates zero-value struct and returns pointer
	pp2 := new(Point)
	pp2.X = 1 // Go auto-dereferences: pp2.X is same as (*pp2).X
	pp2.Y = 2
	fmt.Println("pp2:", *pp2)

	// Anonymous struct — one-off struct without type name
	config := struct {
		Host string
		Port int
	}{
		Host: "localhost",
		Port: 8080,
	}
	fmt.Println("config:", config)

	// Struct comparison — structs are comparable if all fields are comparable
	a := Point{1, 2}
	b := Point{1, 2}
	c := Point{1, 3}
	fmt.Println("a == b:", a == b) // true
	fmt.Println("a == c:", a == c) // false

	// Nested struct
	employee := Person{
		Name:  "Achyut",
		Age:   25,
		Email: "achyut@example.com",
		Address: Address{
			Street: "123 Main St",
			City:   "Kolkata",
			State:  "WB",
			Zip:    "700001",
		},
	}
	fmt.Printf("Employee: %+v\n", employee)
	fmt.Println("City:", employee.Address.City) // access nested fields

	// -------------------------------------------------------------------------
	// SECTION 2: Methods
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Methods ---")

	rect := Rectangle{Width: 5, Height: 3}
	fmt.Println("rect:", rect)        // uses String() method
	fmt.Println("Area:", rect.Area()) // 15
	fmt.Println("Perimeter:", rect.Perimeter()) // 16

	// Pointer receiver method — modifies the original
	rect.Scale(2) // Go automatically takes address: (&rect).Scale(2)
	fmt.Println("After Scale(2):", rect)
	fmt.Println("New Area:", rect.Area()) // 60

	// Value vs pointer receiver — IMPORTANT distinction:
	// Use pointer receiver when:
	//   1. Method needs to MODIFY the receiver
	//   2. Struct is large (avoid copy overhead)
	// Use value receiver when:
	//   1. Method only READS the receiver
	//   2. Receiver is a small struct or basic type

	// Methods on custom types
	temp := Celsius(100)
	fmt.Println("Celsius:", temp)
	fmt.Printf("In Fahrenheit: %.1f°F\n", float64(temp.ToFahrenheit()))

	// Struct embedding — promoted methods
	fmt.Println("\n--- Embedding ---")

	dog := Dog{
		Animal:  Animal{Name: "Rex", Weight: 25.0},
		Breed:   "German Shepherd",
		Trained: true,
	}

	// Can access Animal fields directly through Dog (promoted)
	fmt.Println("Name:", dog.Name)   // promoted from Animal
	fmt.Println("Weight:", dog.Weight) // promoted from Animal
	fmt.Println("Breed:", dog.Breed)

	// Can call Animal methods through Dog
	fmt.Println("Describe:", dog.Animal.Describe()) // explicit
	fmt.Println("Describe:", dog.Describe())        // promoted — same thing

	dog.Feed(2.5) // pointer receiver on Animal, called through Dog
	fmt.Println("Weight after feed:", dog.Weight)

	fmt.Println("Dog:", dog)

	// -------------------------------------------------------------------------
	// SECTION 3: Interfaces
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Interfaces ---")

	// Concrete types
	rect2 := Rectangle{Width: 4, Height: 6}
	circle := Circle{Radius: 5}
	triangle := Triangle{A: 3, B: 4, C: 5}

	// All three satisfy the Shape interface — polymorphism!
	shapes := []Shape{rect2, circle, triangle}

	for _, s := range shapes {
		fmt.Printf("%T → ", s)
		printShapeInfo(s)
	}

	fmt.Printf("Total area: %.2f\n", totalArea(shapes))

	// Interface variable — holds (type, value) pair
	var s Shape = circle
	fmt.Printf("s holds: type=%T, value=%v\n", s, s)

	s = rect2 // same variable, now holds Rectangle
	fmt.Printf("s now holds: type=%T, value=%v\n", s, s)

	// -------------------------------------------------------------------------
	// SECTION 4: Type assertion
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Type Assertion ---")

	var shape Shape = Circle{Radius: 3}

	// Safe assertion — check ok
	if c, ok := shape.(Circle); ok {
		fmt.Println("It's a Circle! Radius:", c.Radius)
	}

	// Try wrong type
	if _, ok := shape.(Rectangle); !ok {
		fmt.Println("It's NOT a Rectangle")
	}

	// Unsafe assertion — panics if wrong type
	// c := shape.(Circle) // will panic if shape isn't a Circle

	// -------------------------------------------------------------------------
	// SECTION 5: Type switch
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Type Switch ---")

	values := []interface{}{42, 3.14, "hello", true, []int{1, 2, 3}, nil}
	for _, v := range values {
		fmt.Println(typeSwitch(v))
	}

	// -------------------------------------------------------------------------
	// SECTION 6: Empty interface (any)
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Empty Interface (any) ---")

	describe(42)
	describe("hello")
	describe(true)
	describe(Point{1, 2})
	describe(nil)

	// Slice of any — can hold mixed types
	mixed := []any{1, "two", 3.0, true, Point{4, 5}}
	for _, v := range mixed {
		fmt.Printf("%T: %v\n", v, v)
	}

	// -------------------------------------------------------------------------
	// SECTION 7: Nil interface subtlety
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Nil Interface Subtlety ---")

	var s1 Shape             // nil interface: type=nil, value=nil
	fmt.Println("nil interface:", s1 == nil) // true

	// This is a subtle trap!
	var c2 *Circle            // nil pointer of type *Circle
	// Don't assign nil pointer to interface — interface won't be nil!
	// var s2 Shape = c2     // s2 has type=*Circle, value=nil → s2 != nil
	_ = c2

	// -------------------------------------------------------------------------
	// SECTION 8: Practical patterns
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Builder Pattern ---")

	query := NewQueryBuilder("users").
		Where("age > 18").
		Where("active = true").
		Limit(10).
		Build()
	fmt.Println("Query:", query)

	fmt.Println("\n--- Functional Options Pattern ---")

	s3 := NewServer(
		WithHost("0.0.0.0"),
		WithPort(9090),
		WithTimeout(60),
	)
	fmt.Printf("Server: %+v\n", *s3)

	defaultServer := NewServer() // uses all defaults
	fmt.Printf("Default Server: %+v\n", *defaultServer)

	// -------------------------------------------------------------------------
	// SECTION 9: Struct tags (JSON encoding)
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Struct Tags (JSON) ---")

	u := User{
		ID:       1,
		Username: "achyut",
		Password: "secret123",       // will be omitted in JSON
		IsAdmin:  true,
	}
	// Email is empty → omitempty will skip it in JSON output
	fmt.Printf("User struct: %+v\n", u)
	// For actual JSON encoding, use encoding/json package (Module 08)

	// -------------------------------------------------------------------------
	// SECTION 10: Interface composition example
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Interface Composition ---")

	// Implementing the Stringer interface (from fmt package)
	// Any type with String() string implements fmt.Stringer
	// fmt.Println calls String() automatically
	fmt.Println(Point{3, 4})       // calls Point.String()
	fmt.Println(Circle{5})         // calls Circle.String()
	fmt.Println(rect)              // calls Rectangle.String()

	fmt.Println("\n=== MODULE 03 COMPLETE ===")
}
